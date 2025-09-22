package tools

import "fmt"

// FileListTool is a tool for listing files in a directory.
type FileListTool struct{}

// Name returns the name of the tool.
func (fl *FileListTool) Name() string {
	return "file_list"
}

// Description returns a description of the tool.
func (fl *FileListTool) Description() string {
	return "Lists all files and directories in a specified path, including hidden ones. Use '.' for the current directory."
}

// Execute lists the files in the specified directory.
func (fl *FileListTool) Execute(args ...string) (string, error) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	command := fmt.Sprintf("ls -la %s", path)
	// Delegate to the ShellTool to ensure it runs in the persistent shell
	shellTool := &ShellTool{}
	return shellTool.Execute(command)
}
