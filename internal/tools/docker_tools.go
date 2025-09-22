package tools

import (
	"agent-thing/internal/docker"
	"fmt"
)

// DockerStartTool starts the Docker container.

type DockerStartTool struct{}

func (t *DockerStartTool) Name() string { return "docker_start" }

func (t *DockerStartTool) Description() string {
	return "Starts the Docker container. Usage: docker_start"
}

func (t *DockerStartTool) Execute(args ...string) (string, error) {
	// The chrootDir will need to be retrieved from config or passed in.
	// For now, we'll assume it's available.
	// This is a simplification and might need to be refactored.
	return "", docker.StartContainer("/app")
}

// DockerStopTool stops the Docker container.

type DockerStopTool struct{}

func (t *DockerStopTool) Name() string { return "docker_stop" }

func (t *DockerStopTool) Description() string {
	return "Stops the Docker container. Usage: docker_stop"
}

func (t *DockerStopTool) Execute(args ...string) (string, error) {
	return "Container stopped.", docker.StopContainer()
}

// DockerRebuildTool rebuilds the Docker container.

type DockerRebuildTool struct{}

func (t *DockerRebuildTool) Name() string { return "docker_rebuild" }

func (t *DockerRebuildTool) Description() string {
	return "Rebuilds the Docker container. Usage: docker_rebuild"
}

func (t *DockerRebuildTool) Execute(args ...string) (string, error) {
	return "Container rebuilt.", docker.RebuildContainer("/app")
}

// DockerStatusTool gets the status of the Docker container.

type DockerStatusTool struct{}

func (t *DockerStatusTool) Name() string { return "docker_status" }

func (t *DockerStatusTool) Description() string {
	return "Gets the status of the Docker container. Usage: docker_status"
}

func (t *DockerStatusTool) Execute(args ...string) (string, error) {
	status, err := docker.ContainerStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get container status: %w", err)
	}
	return status, nil
}
