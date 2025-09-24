package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to determine user home directory: %v", err)
	}
	configPath := filepath.Join(homeDir, ".config", "agent-thing", "config.ini")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check for the 'migrate' subcommand
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		handleMigration(cfg)
		return
	}

	// --- Normal Agent Startup ---

	// Initialize Docker client and start container
	if err := docker.Init(); err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	if err := docker.StartContainer(cfg.ChrootDir); err != nil {
		log.Fatalf("Failed to start Docker container: %v", err)
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

	// Start the web server
	srv := server.NewServer(runAgentLogic)
	srv.Start(":8080")
}

func handleMigration(cfg *config.Config) {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: go run ./agent.go migrate <command> [arguments]")
	}

	command := os.Args[2]

	if command == "create" {
		if len(os.Args) < 4 {
			log.Fatalf("Usage: go run ./agent.go migrate create <name>")
		}
		name := os.Args[3]
		timestamp := time.Now().Format("20060102150405")
		upFile := fmt.Sprintf("db/migrations/%s_%s.up.sql", timestamp, name)
		downFile := fmt.Sprintf("db/migrations/%s_%s.down.sql", timestamp, name)

		if err := os.WriteFile(upFile, []byte("-- up migration here"), 0o644); err != nil {
			log.Fatalf("Failed to create up migration file: %v", err)
		}
		if err := os.WriteFile(downFile, []byte("-- down migration here"), 0o644); err != nil {
			log.Fatalf("Failed to create down migration file: %v", err)
		}

		fmt.Printf("Created migration files:\n%s\n%s\n", upFile, downFile)
		return
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSslMode,
	)

	m, err := migrate.New(
		"file://db/migrations",
		dsn,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while migrating up: %v", err)
		}
		fmt.Println("Migrations applied successfully.")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while migrating down: %v", err)
		}
		fmt.Println("Migrations rolled back successfully.")
	case "status":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
		fmt.Printf("Version: %d, Dirty: %v\n", version, dirty)
	default:
		log.Fatalf("Unknown command: %s", command)
	}
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
