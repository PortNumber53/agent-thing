package tools

import "fmt"

// Tool defines the interface for an agent tool.
type Tool interface {
	Name() string
	Description() string
	Execute(args ...string) (string, error)
}

// ToolSet holds a collection of tools available to the agent.
type ToolSet struct {
	tools map[string]Tool
}

// NewToolSet creates a new toolset.
func NewToolSet() *ToolSet {
	return &ToolSet{
		tools: make(map[string]Tool),
	}
}

// Add adds a tool to the toolset.
func (ts *ToolSet) Add(tool Tool) {
	ts.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (ts *ToolSet) Get(name string) (Tool, bool) {
	tool, ok := ts.tools[name]
	return tool, ok
}

// GetTools returns all tools in the set.
func (ts *ToolSet) GetTools() []Tool {
	var tools []Tool
	for _, tool := range ts.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolsDescription generates a string describing all available tools.
func (ts *ToolSet) GetToolsDescription() string {
	var desc string
	for _, tool := range ts.tools {
		desc += fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description())
	}
	return desc
}
