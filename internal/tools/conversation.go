package tools

import "strings"

// ConversationTool is used to send a message back to the user.
type ConversationTool struct{}

// Name returns the name of the tool.
func (t *ConversationTool) Name() string {
	return "conversation"
}

// Description returns a description of the tool.
func (t *ConversationTool) Description() string {
	return "Use this tool to send a message to the user, ask for clarification, or signal that the task is complete. The arguments will be sent as the message. Usage: conversation \"<message_to_user>\""
}

// Execute simply returns the message to be sent to the user.
func (t *ConversationTool) Execute(args ...string) (string, error) {
	if len(args) == 0 {
		return "Error: missing message to send.", nil
	}
	return strings.Join(args, " "), nil
}
