package main

import (
	"agent-thing/internal/config"
	"agent-thing/internal/docker"
	"agent-thing/internal/llm"
	"agent-thing/internal/tools"
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

// AIResponse defines the structure for the AI's JSON response.
type AIResponse struct {
	Tool string   `json:"tool"`
	Args []string `json:"args"`
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("/home/grimlock/.config/agent-thing/config.ini")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the toolset
	toolSet := tools.NewToolSet()
	toolSet.Add(&tools.ShellTool{})

	// Start the Docker container
	if err := docker.StartContainer(cfg.ChrootDir); err != nil {
		log.Fatalf("Failed to start Docker container: %v", err)
	}

	// Ensure the container is stopped on exit
	defer func() {
		if err := docker.StopContainer(); err != nil {
			log.Printf("Failed to stop Docker container: %v", err)
		}
	}()

	// Set up graceful shutdown
	done := make(chan bool, 1)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		fmt.Println("\nCtrl+C detected. Shutting down...")
		done <- true
	}()

	// Initialize the LLM client
	llmClient, err := llm.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	// Goroutine to read user input
	inputChan := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				close(inputChan)
				return
			}
			inputChan <- strings.TrimSpace(input)
		}
	}()

	fmt.Println("\nEnter a task for the agent (or type '/tools', 'exit', or 'quit').")

REPL:
	for {
		fmt.Print("> ")
		select {
		case <-done:
			break REPL

		case userInput, ok := <-inputChan:
			if !ok {
				done <- true
				break REPL
			}

			switch userInput {
			case "exit", "quit":
				fmt.Println("Exiting agent...")
				done <- true
				break REPL
			case "/tools":
				fmt.Println("\nAvailable Tools:")
				fmt.Println(toolSet.GetToolsDescription())
				continue
			case "":
				continue
			}

			// If we get here, it's a task for the AI
			systemPrompt := fmt.Sprintf("You are a helpful AI assistant. Your goal is to accomplish the user's task by selecting the appropriate tool and arguments. "+
				"You have access to the following tools:\n%s\n"+
				"Based on the user's request, decide which tool to use. "+
				"If the user's request is a greeting or a conversational question that does not require a tool, respond with {\\\"tool\\\": \\\"conversation\\\", \\\"args\\\": [\\\"Your conversational response here\\\"]}. "+
				"Otherwise, respond with ONLY a JSON object in the format: {\\\"tool\\\": \\\"tool_name\\\", \\\"args\\\": [\\\"arg1\\\", \\\"arg2\\\"]}", toolSet.GetToolsDescription())

			prompt := fmt.Sprintf("%s\n\nUser Task: %s", systemPrompt, userInput)

			color.Yellow("Agent is thinking...")

			jsonResponse, err := llmClient.GenerateContent(prompt)
			if err != nil {
				color.Red("Failed to get response from LLM: %v", err)
				continue
			}

			cleanedJSON := strings.Trim(strings.TrimSpace(jsonResponse), "`\njson")
			var aiResp AIResponse
			if err := json.Unmarshal([]byte(cleanedJSON), &aiResp); err != nil {
				color.Red("Failed to parse AI response: %v\nRaw response: %s", err, cleanedJSON)
				continue
			}

			if aiResp.Tool == "conversation" {
				color.Green(strings.Join(aiResp.Args, " "))
				continue
			}

			tool, ok := toolSet.Get(aiResp.Tool)
			if !ok {
				color.Red("Error: AI requested an unknown tool: '%s'", aiResp.Tool)
				continue
			}

			color.Cyan("Executing tool '%s' with args: %v", aiResp.Tool, aiResp.Args)
			output, err := tool.Execute(aiResp.Args...)
			if err != nil {
				color.Red("Failed to execute tool '%s': %v", aiResp.Tool, err)
				continue
			}

			color.Green("\n--- Output ---")
			fmt.Println(output)
			color.Green("--------------")
		}
	}

	<-done
	fmt.Println("Shutdown signal received. Container will be stopped.")
}
