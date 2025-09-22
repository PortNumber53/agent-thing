package tools

import (
	"agent-thing/internal/docker"
	"fmt"
)

// SSHKeyGenTool defines a tool for generating SSH keys.
type SSHKeyGenTool struct{}

// Name returns the name of the tool.
func (t *SSHKeyGenTool) Name() string {
	return "ssh_key_gen"
}

// Description returns a description of the tool.
func (t *SSHKeyGenTool) Description() string {
	return "Generates a new ed25519 SSH key pair in the /app/ directory. The key will be named id_ed25519 and will have no passphrase."
}

// Execute generates the SSH key.
func (t *SSHKeyGenTool) Execute(args ...string) (string, error) {
	command := "ssh-keygen -t ed25519 -f /app/id_ed25519 -N ''"
	output, err := docker.Exec(command)
	if err != nil {
		return "", fmt.Errorf("failed to generate ssh key: %w\n%s", err, output)
	}
	return "Successfully generated new ed25519 key pair in /app/id_ed25519.", nil
}
