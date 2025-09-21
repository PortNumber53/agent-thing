package docker

import (
	"bytes"
	"fmt"
	"os/exec"
)

const (
	containerName = "dev-environment"
	imageName     = "agent-dev-env"
)

// runCommand is a helper to execute shell commands and return their output.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running command '%s %v': %w: %s", name, args, err, stderr.String())
	}
	return out.String(), nil
}

// StartContainer builds the Docker image and starts the development container.
func StartContainer(chrootDir string) error {
	fmt.Println("Building development environment image...")
	_, err := runCommand("docker", "build", "-t", imageName, ".")
	if err != nil {
		return fmt.Errorf("failed to build docker image: %w", err)
	}

	fmt.Println("Starting development container...")
	volumeMount := fmt.Sprintf("%s:/app", chrootDir)
	_, err = runCommand("docker", "run", "-d", "--name", containerName, "-v", volumeMount, "--workdir", "/app", imageName, "tail", "-f", "/dev/null")
	if err != nil {
		// If the container is already running, we can ignore the error.
		// A better approach would be to check if it exists first.
		fmt.Println("Container may already be running. Continuing...")
	}

	fmt.Println("Development container started.")
	return nil
}

// StopContainer stops and removes the development container.
func StopContainer() error {
	fmt.Println("Stopping development container...")
	_, _ = runCommand("docker", "stop", containerName)

	fmt.Println("Removing development container...")
	_, err := runCommand("docker", "rm", containerName)
	if err != nil {
		// Ignore errors if the container is already removed.
		return nil
	}

	fmt.Println("Development container stopped and removed.")
	return nil
}

// Exec executes a command inside the running development container.
func Exec(command string) (string, error) {
	return runCommand("docker", "exec", "--user", "root", containerName, "/bin/bash", "-c", command)
}
