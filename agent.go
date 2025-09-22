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

	// Initialize the toolset
	toolSet = tools.NewToolSet()
	toolSet.Add(&tools.ShellTool{})
	toolSet.Add(&tools.FileReadTool{})
	toolSet.Add(&tools.FileWriteTool{})
	toolSet.Add(&tools.FileListTool{})
	toolSet.Add(&tools.SSHKeyGenTool{})

	// Initialize the LLM client
	llmClient, err = llm.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

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
	return fmt.Sprintf("You are a helpful AI assistant. This is the conversation so far: %s\n\nUser's request: %s\n\nAvailable tools:\n%s\n\nDecide which tool to use to best respond to the user's request. Respond in JSON format.", conversationSummary, userInput, ts.GetToolsDescription())
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
	rawResp, err := llmClient.GenerateContent(prompt)
	if err != nil {
		// handle error
		return
	}

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(rawResp), &aiResp); err != nil {
		// handle error
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
	tool, ok := toolSet.Get(toolName)
	if !ok {
		// handle error
		return
	}

	thinkingMsg := fmt.Sprintf("Executing tool '%s' with args: %v", tool.Name(), args)
	conn.WriteMessage(websocket.TextMessage, []byte(thinkingMsg))

	output, err := tool.Execute(args...)
	if err != nil {
		// handle error
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte(output))

	newSummary, _ := updateSummary(*conversationSummary, userInput, output)
	*conversationSummary = newSummary
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
