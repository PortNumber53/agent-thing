package tools

import (
	"agent-thing/internal/docker"
	"encoding/json"
	"fmt"
	"strings"
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

// Execute reads the content of the file.
func (fr *FileReadTool) Execute(args ...string) (string, error) {
	if len(args) == 0 {
		return "Error: file_read requires a file path.", nil
	}
	filePath := args[0]
	fullCommand := strings.Join(args, " ")

	// Use the docker exec to cat the file
	command := fmt.Sprintf("cat %s", filePath)
	output, err := docker.Exec(command)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	var action string
	if strings.Contains(fullCommand, "for download") {
		action = "download"
	} else if strings.Contains(fullCommand, "for copy") {
		action = "copy"
	}

	if action != "" {
		fileName := filePath[strings.LastIndex(filePath, "/")+1:]
		fileInfo := map[string]string{
			"name":    fileName,
			"content": output,
			"action":  action,
		}
		jsonInfo, _ := json.Marshal(fileInfo)
		return fmt.Sprintf("--- FILE_CONTENT ---%s", string(jsonInfo)), nil
	}

	return output, nil
}
