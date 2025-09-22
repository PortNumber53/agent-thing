package tools

import (
	"agent-thing/internal/llm"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// AIResponse defines the structure for the AI's JSON response.
type AIResponse struct {
	Tool string   `json:"tool"`
	Args []string `json:"args"`
}

// AutonomousExecutionTool enables the agent to perform multi-step tasks.
type AutonomousExecutionTool struct {
	ToolSet   *ToolSet
	LLMClient *llm.Client
}

func (t *AutonomousExecutionTool) Name() string { return "autonomous_execution" }

func (t *AutonomousExecutionTool) Description() string {
	return `Executes a sequence of tool calls to achieve a complex goal. Use this for multi-step tasks. The first argument must be the user's original request. Usage: autonomous_execution "<user_request>"`
}

func (t *AutonomousExecutionTool) Execute(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("autonomous_execution requires the user's original request as the first argument")
	}

	originalRequest := args[0]
	var fullOutput string
	var turnHistory string // Running summary of the session
	maxTurns := 15 // Increased max turns for more complex tasks

	for i := 0; i < maxTurns; i++ {
		prompt := t.constructAutonomyPrompt(originalRequest, turnHistory)
		log.Printf("--- Autonomous Turn %d: Sending Prompt to LLM ---\n%s\n----------------------------------------------------", i+1, prompt)

		rawResp, err := t.LLMClient.GenerateContent(prompt)
		if err != nil {
			return "", fmt.Errorf("LLM call failed during autonomous execution: %w", err)
		}

		cleanedResp := t.cleanLLMResponse(rawResp)
		log.Printf("--- Autonomous Turn %d: Cleaned LLM Response ---\n%s\n---------------------------------------------------", i+1, cleanedResp)

		var aiResp AIResponse
		if err := json.Unmarshal([]byte(cleanedResp), &aiResp); err != nil {
			// If the model fails to return valid JSON, we'll treat it as a conversation turn.
			// This helps with robustness when the model wants to just talk.
			aiResp = AIResponse{Tool: "conversation", Args: []string{rawResp}}
		}

		if aiResp.Tool == "conversation" {
			finalMessage := strings.Join(aiResp.Args, " ")
			fullOutput += "\nAgent has completed the task: " + finalMessage
			return fullOutput, nil
		}

		tool, ok := t.ToolSet.Get(aiResp.Tool)
		if !ok {
			// If the tool is not found, inform the agent.
			summary := fmt.Sprintf("Error: Tool '%s' not found.", aiResp.Tool)
			turnHistory += fmt.Sprintf("Step %d: %s\n", i+1, summary)
			fullOutput += fmt.Sprintf("Step %d: %s\n\n", i+1, summary)
			continue
		}

		commandDescription := fmt.Sprintf("%s with args: %v", aiResp.Tool, aiResp.Args)

		log.Printf("--- Autonomous Turn %d: Executing %s ---", i+1, commandDescription)
		toolOutput, err := tool.Execute(aiResp.Args...)

		// The Exec function in docker.go now returns a specific error format we can parse.
		if err != nil {
			if strings.Contains(err.Error(), "command exited with non-zero status") {
				// Extract the actual error message from the shell.
				toolOutput = err.Error()
			}
		}

		summary := summarize(commandDescription, toolOutput, err)
		turnHistory += fmt.Sprintf("Step %d: %s\n", i+1, summary)
		fullOutput += fmt.Sprintf("Step %d: Executed %s\nOutput: %s\n\n", i+1, commandDescription, toolOutput)
	}

	return fullOutput + "\nAgent reached maximum turns. Task may be incomplete.", nil
}

func (t *AutonomousExecutionTool) constructAutonomyPrompt(originalRequest, turnHistory string) string {
	// Manually build the tool description to exclude dangerous tools from the agent's view.
	var toolDescriptions strings.Builder
	for _, tool := range t.ToolSet.GetTools() {
		// Exclude self (recursion) and dangerous tools (docker management, key generation).
		if tool.Name() != t.Name() && !strings.HasPrefix(tool.Name(), "docker_") && tool.Name() != "ssh_key_gen" {
			toolDescriptions.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
		}
	}

	return fmt.Sprintf(`You are an autonomous AI agent. Your goal is to complete the user's request by executing a series of commands. Think step-by-step and use the available tools to solve the problem. If you encounter an error, analyze it and try to recover.

Original user request: "%s"

Here is the history of the actions you have taken so far:
%s

Available tools:
%s

**Your Task:**
1.  **Analyze**: Based on the history of commands and their output, determine the single next best action to get closer to completing the original request.
2.  **Act**: Choose one tool to execute. Use 'shell_exec' for all terminal commands, including 'cd', 'ls', 'git', etc. The shell is stateful; 'cd' will change the directory for all subsequent commands.
3.  **Recover**: If a command fails, analyze the error message in the history and decide on a recovery step.
4.  **Converse**: If you are stuck, need more information, or have completed the task, use the 'conversation' tool to communicate with the user.

Respond ONLY with a single, valid JSON object in the format: {"tool": "<tool_name>", "args": ["<arg1>"]}.`, originalRequest, turnHistory, toolDescriptions.String())
}

// summarize creates a concise summary of a tool's execution for the agent's history.
func summarize(command, output string, err error) string {
	if err != nil {
		return fmt.Sprintf("Action '%s' failed with error: %s", command, err.Error())
	}
	if output != "" {
		return fmt.Sprintf("Action '%s' succeeded with output: %s", command, output)
	}
	return fmt.Sprintf("Action '%s' succeeded.", command)
}

func (t *AutonomousExecutionTool) cleanLLMResponse(response string) string {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}
