package main

import (
	"agent-thing/internal/config"
	"agent-thing/internal/docker"
	"agent-thing/internal/llm"
	"agent-thing/internal/server"
	"agent-thing/internal/tools"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

	// Initialize the toolset
	toolSet = tools.NewToolSet()
	toolSet.Add(&tools.ShellTool{})
	toolSet.Add(&tools.FileReadTool{})
	toolSet.Add(&tools.FileWriteTool{})
	toolSet.Add(&tools.FileListTool{})

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
	srv := server.NewServer(handleConnection)
	srv.Start(":8080")
}

// handleConnection is the core logic for a single agent-user session over WebSocket.
func handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	for {
		// Read a message from the WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		userInput := string(msg)

		if userInput == "/tools" {
			description := fmt.Sprintf("Available Tools:\n%s", toolSet.GetToolsDescription())
			if err := conn.WriteMessage(websocket.TextMessage, []byte(description)); err != nil {
				log.Printf("Write error: %v", err)
			}
			continue
		}

		// Construct the system prompt
		systemPrompt := fmt.Sprintf("You are a helpful AI assistant. Your goal is to accomplish the user's task by thinking step-by-step and using the available tools. "+
			"You can chain commands together to solve complex problems. "+
			"1. **Think**: Analyze the user's request and create a plan. "+
			"2. **Act**: Choose the best tool for the current step in your plan. "+
			"You have access to the following tools:\n%s\n"+
			"If the user's request is a greeting or a conversational question that does not require a tool, respond with {\\\"tool\\\": \\\"conversation\\\", \\\"args\\\": [\\\"Your conversational response here\\\"]}. "+
			"Otherwise, respond with ONLY a JSON object in the format: {\\\"tool\\\": \\\"tool_name\\\", \\\"args\\\": [\\\"arg1\\\", \\\"arg2\\\"]}. "+
			"Example of a multi-step task: User asks to 'rename the file 'old.txt' to 'new.txt'. Your thought process might be: "+
			"1. First, I need to see what files are in the current directory. I will use 'file_list' with '.'. "+
			"2. If I see 'old.txt', I will use 'shell_exec' with 'mv old.txt new.txt'.", toolSet.GetToolsDescription())

		prompt := fmt.Sprintf("%s\n\nUser Task: %s", systemPrompt, userInput)

		if err := conn.WriteMessage(websocket.TextMessage, []byte("Agent is thinking...")); err != nil {
			log.Printf("Write error: %v", err)
		}

		// Get the JSON response from the LLM
		jsonResponse, err := llmClient.GenerateContent(prompt)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get response from LLM: %v", err)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg)); err != nil {
				log.Printf("Write error: %v", err)
			}
			continue
		}

		// Parse and execute
		cleanedJSON := strings.Trim(strings.TrimSpace(jsonResponse), "`\njson")
		var aiResp AIResponse
		if err := json.Unmarshal([]byte(cleanedJSON), &aiResp); err != nil {
			errMsg := fmt.Sprintf("Failed to parse AI response: %v\nRaw response: %s", err, cleanedJSON)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg)); err != nil {
				log.Printf("Write error: %v", err)
			}
			continue
		}

		if aiResp.Tool == "conversation" {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(strings.Join(aiResp.Args, " "))); err != nil {
				log.Printf("Write error: %v", err)
			}
			continue
		}

		tool, ok := toolSet.Get(aiResp.Tool)
		if !ok {
			errMsg := fmt.Sprintf("Error: AI requested an unknown tool: '%s'", aiResp.Tool)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg)); err != nil {
				log.Printf("Write error: %v", err)
			}
			continue
		}

		msg = []byte(fmt.Sprintf("Executing tool '%s' with args: %v", aiResp.Tool, aiResp.Args))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Write error: %v", err)
		}

		output, err := tool.Execute(aiResp.Args...)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to execute tool '%s': %v", aiResp.Tool, err)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(errMsg)); err != nil {
				log.Printf("Write error: %v", err)
			}
			continue
		}

		finalOutput := fmt.Sprintf("--- Output ---\n%s", output)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(finalOutput)); err != nil {
			log.Printf("Write error: %v", err)
		}
	}
}
