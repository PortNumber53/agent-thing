package tools

import (
	"agent-thing/internal/docker"
	"strings"
)

// ShellTool is a tool for executing shell commands in the container.
type ShellTool struct{}

// Name returns the name of the tool.
func (s *ShellTool) Name() string {
	return "shell_exec"
}

// Description returns a description of the tool.
func (s *ShellTool) Description() string {
	return "Executes a shell command inside the development container."
}

// Execute runs the shell command.
func (s *ShellTool) Execute(args ...string) (string, error) {
	if len(args) == 0 {
		return "Error: missing command to execute.", nil
	}
	command := strings.Join(args, " ")
	return docker.Exec(command)
}
