package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"agent-thing/internal/config"
	"agent-thing/internal/docker"
	"agent-thing/internal/llm"
	"agent-thing/internal/server"
	"agent-thing/internal/tools"

	"github.com/gorilla/websocket"
)

// AIResponse defines the structure for the AI's JSON response.
type AIResponse struct {
	Tool string   `json:"tool"`
	Args []string `json:"args"`
}

var (
	llmClient *llm.Client
	toolSet   *tools.ToolSet
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("/home/grimlock/.config/agent-thing/config.ini")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Docker client
	if err := docker.Init(); err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}

	// Initialize the LLM client
	llmClient, err = llm.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiRPM)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	// Initialize the toolset
	toolSet = tools.NewToolSet()
	toolSet.Add(&tools.ConversationTool{})
	toolSet.Add(&tools.ShellTool{})
	toolSet.Add(&tools.FileReadTool{})
	toolSet.Add(&tools.FileWriteTool{})
	toolSet.Add(&tools.FileListTool{})
	toolSet.Add(&tools.SSHKeyGenTool{})
	toolSet.Add(&tools.DockerStartTool{})
	toolSet.Add(&tools.DockerStopTool{})
	toolSet.Add(&tools.DockerRebuildTool{})
	toolSet.Add(&tools.DockerStatusTool{})

	// Create, configure, and add the autonomous execution tool
	autoTool := &tools.AutonomousExecutionTool{}
	autoTool.LLMClient = llmClient
	autoTool.ToolSet = toolSet
	toolSet.Add(autoTool)

	// Start the Docker container
	if err := docker.StartContainer(cfg.ChrootDir); err != nil {
		log.Fatalf("Failed to start Docker container: %v", err)
	}

	// Start the web server
	srv := server.NewServer(runAgentLogic)
	srv.Start(":8080")
}

// handleConnection is the core logic for a single agent-user session over WebSocket.
type incomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type toolExecPayload struct {
	Tool string   `json:"tool"`
	Args []string `json:"args"`
}

func constructPrompt(userInput, conversationSummary string, ts *tools.ToolSet) string {
	return fmt.Sprintf(`You are a helpful AI assistant. This is the conversation so far: %s

User's request: %s

Available tools:
%s

Decide which tool to use to best respond to the user's request. For complex, multi-step tasks, use the 'autonomous_execution' tool. Respond ONLY with a single, valid JSON object in the following format: {"tool": "<tool_name>", "args": ["<arg1>", "<arg2>", ...]}.`,
		conversationSummary, userInput, ts.GetToolsDescription())
}

func runAgentLogic(conn *websocket.Conn) {
	defer conn.Close()
	var conversationSummary string

	for {
		// Read message from browser
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket closed")
			return
		}

		var req incomingMessage
		if err := json.Unmarshal(msg, &req); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		switch req.Type {
		case "conversation":
			handleConversation(conn, req.Payload, &conversationSummary)
		case "tool_exec":
			handleToolExec(conn, req.Payload, &conversationSummary)
		default:
			log.Printf("Unknown message type: %s", req.Type)
		}
	}
}

func handleConversation(conn *websocket.Conn, payload json.RawMessage, conversationSummary *string) {
	var userInput string
	json.Unmarshal(payload, &userInput)

	if err := conn.WriteMessage(websocket.TextMessage, []byte("Agent is thinking...")); err != nil {
		log.Printf("Write error: %v", err)
	}

	prompt := constructPrompt(userInput, *conversationSummary, toolSet)
	log.Printf("--- Sending Prompt to LLM ---\n%s\n-----------------------------", prompt)

	rawResp, err := llmClient.GenerateContent(prompt)
	if err != nil {
		log.Printf("Error generating content from LLM: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error from LLM: %v", err)))
		return
	}
	log.Printf("--- Received Raw Response from LLM ---\n%s\n------------------------------------", rawResp)

	cleanedResp := cleanLLMResponse(rawResp)
	log.Printf("--- Cleaned LLM Response ---\n%s\n------------------------------", cleanedResp)

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(cleanedResp), &aiResp); err != nil {
		log.Printf("Error unmarshalling LLM response: %v", err)
		log.Printf("LLM raw response that failed to unmarshal: %s", rawResp)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error parsing LLM response: %v", err)))
		return
	}

	if aiResp.Tool == "conversation" {
		response := strings.Join(aiResp.Args, " ")
		conn.WriteMessage(websocket.TextMessage, []byte(response))
		newSummary, _ := updateSummary(*conversationSummary, userInput, response)
		*conversationSummary = newSummary
		return
	}

	executeTool(conn, aiResp.Tool, aiResp.Args, userInput, conversationSummary)
}

func handleToolExec(conn *websocket.Conn, payload json.RawMessage, conversationSummary *string) {
	var toolReq toolExecPayload
	json.Unmarshal(payload, &toolReq)

	userInput := fmt.Sprintf("User directly executed tool '%s'", toolReq.Tool)
	executeTool(conn, toolReq.Tool, toolReq.Args, userInput, conversationSummary)
}

func executeTool(conn *websocket.Conn, toolName string, args []string, userInput string, conversationSummary *string) {
	log.Printf("Attempting to execute tool: '%s' with args: %v", toolName, args)
	tool, ok := toolSet.Get(toolName)
	if !ok {
		errMsg := fmt.Sprintf("Error: Tool '%s' not found.", toolName)
		log.Println(errMsg)
		conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
		return
	}

	thinkingMsg := fmt.Sprintf("Executing tool '%s' with args: %v", tool.Name(), args)
	conn.WriteMessage(websocket.TextMessage, []byte(thinkingMsg))

	output, err := tool.Execute(args...)
	if err != nil {
		errMsg := fmt.Sprintf("Error executing tool '%s': %v", tool.Name(), err)
		log.Println(errMsg)
		conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
		return
	}
	log.Printf("--- Tool '%s' Output ---\n%s\n---------------------------", tool.Name(), output)

	conn.WriteMessage(websocket.TextMessage, []byte(output))

	newSummary, _ := updateSummary(*conversationSummary, userInput, output)
	*conversationSummary = newSummary
}

// updateSummary asks the LLM to summarize the last interaction and add it to the conversation summary.
// cleanLLMResponse removes markdown code fences from a string.
func cleanLLMResponse(response string) string {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	response = strings.TrimSuffix(response, "```")

	return strings.TrimSpace(response)
}

// updateSummary asks the LLM to summarize the last interaction and add it to the conversation summary.
func updateSummary(currentSummary, userInput, agentOutput string) (string, error) {
	summarizationPrompt := fmt.Sprintf("Current Summary:\n%s\n\nLast Interaction:\nUser: %s\nAgent: %s\n\nInstructions:\nUpdate the summary with the key information from the last interaction. Keep it concise. If the last interaction was a simple greeting, you can keep the summary empty.", currentSummary, userInput, agentOutput)

	newSummary, err := llmClient.GenerateContent(summarizationPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to get new summary from LLM: %w", err)
	}

	return strings.TrimSpace(newSummary), nil
}
