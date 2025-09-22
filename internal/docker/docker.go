package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const ContainerName = "dev-environment"

var cli *client.Client
var ctx context.Context

// Init initializes the Docker client.
func Init() error {
	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	ctx = context.Background()
	return nil
}

// StartContainer ensures the development Docker container is running.
func StartContainer(chrootDir string) error {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		if cont.Names[0] == "/"+ContainerName {
			if cont.State != "running" {
				if err := cli.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
					return fmt.Errorf("failed to start container: %w", err)
				}
				fmt.Println("Development container started.")
			}
			return nil
		}
	}

	// If we get here, the container doesn't exist. We need to build the image, then create and start the container.
	fmt.Println("Building development environment image...")
	cmd := exec.Command("docker", "build", "-t", "dev-env-img", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build docker image: %w\n%s", err, string(output))
	}

	fmt.Println("Creating and starting development container...")
	_, err = cli.ContainerCreate(ctx,
		&container.Config{
			Image:        "dev-env-img",
			Cmd:          []string{"tail", "-f", "/dev/null"},
			WorkingDir:   "/app",
			User:         "root",
			Tty:          false,
		},
		&container.HostConfig{
			Binds: []string{fmt.Sprintf("%s:/app", chrootDir)},
		},
		nil, nil, ContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, ContainerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Println("Development container started.")
	return nil
}

// Exec runs a command inside the Docker container.
func Exec(command string) (string, error) {
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/bash", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		User:         "root",
	}

	execID, err := cli.ContainerExecCreate(ctx, ContainerName, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec config: %w", err)
	}

	resp, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to copy output from exec: %w", err)
	}

	// Check the exit code
	inspect, err := cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspect.ExitCode != 0 {
		return "", fmt.Errorf("command exited with non-zero status %d: %s", inspect.ExitCode, errBuf.String())
	}

	return outBuf.String(), nil
}

// StopContainer stops the development container.
func StopContainer() error {
	fmt.Println("Stopping development container...")
	if err := cli.ContainerStop(ctx, ContainerName, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	fmt.Println("Development container stopped.")
	return nil
}

// RebuildContainer stops, removes, and rebuilds the development container.
func RebuildContainer(chrootDir string) error {
	if err := StopContainer(); err != nil {
		return err
	}
	if err := cli.ContainerRemove(ctx, ContainerName, container.RemoveOptions{}); err != nil {
		// Ignore 'no such container' errors
		if !client.IsErrNotFound(err) {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}
	return StartContainer(chrootDir)
}

// ContainerStatus gets the status of the development container.
func ContainerStatus() (string, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		if cont.Names[0] == "/"+ContainerName {
			return fmt.Sprintf("Container '%s' is %s", ContainerName, cont.State), nil
		}
	}

	return fmt.Sprintf("Container '%s' not found.", ContainerName), nil
}
