package tools

import (
	"agent-thing/internal/docker"
	"fmt"
	"strings"
)

// FileWriteTool is a tool for writing content to a file.
type FileWriteTool struct{}

// Name returns the name of the tool.
func (fw *FileWriteTool) Name() string {
	return "file_write"
}

// Description returns a description of the tool.
func (fw *FileWriteTool) Description() string {
	return "Writes content to a specified file in the container. Overwrites the file if it exists."
}

// Execute writes the content to the file.
func (fw *FileWriteTool) Execute(args ...string) (string, error) {
	if len(args) < 2 {
		return "Error: file_write requires at least two arguments: the file path and the content.", nil
	}
	filePath := args[0]
	content := strings.Join(args[1:], " ")

	// To handle multi-line content and special characters, it's safer to use a here-document with `tee`.
	// This prevents issues with shell interpretation of the content string.
	command := fmt.Sprintf("tee %s <<'EOF'\n%s\nEOF", filePath, content)

	_, err := docker.Exec(command)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote to %s", filePath), nil
}
