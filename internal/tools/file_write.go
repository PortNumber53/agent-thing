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
	writeCommand := fmt.Sprintf("tee %s <<'EOF'\n%s\nEOF", filePath, content)

	// Execute the write command first.
	_, err := docker.Exec(writeCommand)
	if err != nil {
		return "", err
	}

	// Immediately change the ownership of the new file.
	chownCommand := fmt.Sprintf("chown developer:developer %s", filePath)
	_, err = docker.Exec(chownCommand)
	if err != nil {
		// If chown fails, log it but don't fail the whole operation, as the file was still written.
		return fmt.Sprintf("Successfully wrote to %s, but failed to change ownership.", filePath), nil
	}

	return fmt.Sprintf("Successfully wrote to %s", filePath), nil
}
