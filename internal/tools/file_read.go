package tools

import (
	"agent-thing/internal/docker"
	"fmt"
)

// FileReadTool is a tool for reading the contents of a file.
type FileReadTool struct{}

// Name returns the name of the tool.
func (fr *FileReadTool) Name() string {
	return "file_read"
}

// Description returns a description of the tool.
func (fr *FileReadTool) Description() string {
	return "Reads the entire content of a specified file from the container."
}

// Execute reads the file content.
func (fr *FileReadTool) Execute(args ...string) (string, error) {
	if len(args) != 1 {
		return "Error: file_read requires exactly one argument: the file path.", nil
	}
	filePath := args[0]
	// Use `cat` to read the file content. This is a simple and effective way.
	command := fmt.Sprintf("cat %s", filePath)
	return docker.Exec(command)
}
