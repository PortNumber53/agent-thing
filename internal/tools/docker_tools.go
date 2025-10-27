package tools

import (
	"fmt"

	"agent-thing/internal/docker"
)

// DockerStartTool starts the Docker container.
//
// It needs to know which host directory should be mounted into the container
// (the same value configured as CHROOT_DIR in config.ini and passed to
// docker.StartContainer at process startup).
type DockerStartTool struct {
	ChrootDir string
}

func (t *DockerStartTool) Name() string { return "docker_start" }

func (t *DockerStartTool) Description() string {
	return "Starts the Docker container. Usage: docker_start"
}

func (t *DockerStartTool) Execute(args ...string) (string, error) {
	if err := docker.StartContainer(t.ChrootDir); err != nil {
		return "", err
	}
	return "Development container started.", nil
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
//
// Similar to DockerStartTool, this tool needs the configured chrootDir so it
// can remount the correct host directory after rebuilding.
type DockerRebuildTool struct {
	ChrootDir string
}

func (t *DockerRebuildTool) Name() string { return "docker_rebuild" }

func (t *DockerRebuildTool) Description() string {
	return "Rebuilds the Docker container. Usage: docker_rebuild"
}

func (t *DockerRebuildTool) Execute(args ...string) (string, error) {
	if err := docker.RebuildContainer(t.ChrootDir); err != nil {
		return "", err
	}
	return "Development container rebuilt.", nil
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
