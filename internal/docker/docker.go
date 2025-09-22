package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const ContainerName = "dev-environment"

var (
	cli       *client.Client
	ctx       context.Context
	shellConn *types.HijackedResponse
)

const EndOfCommandMarker = "END_OF_COMMAND_MARKER_e5d5a7b8-b2e0-4c0f-83b3-2f1b6d7a3b7d"


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
	// Ensure the host directory exists and has the correct permissions for the container's user (UID 1000)
	fmt.Println("Checking host directory permissions...")
	if err := os.MkdirAll(chrootDir, 0755); err != nil {
		return fmt.Errorf("failed to create chroot directory on host: %w", err)
	}
	// This requires the user running the agent to have sudo privileges.
	cmd := exec.Command("sudo", "chown", "-Rv", "1000:1000", chrootDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set ownership of chroot directory: %w\n%s", err, string(output))
	}

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

			// Always start a new persistent shell for this agent instance.
			fmt.Println("Starting persistent shell in container...")
			if err := startPersistentShell(); err != nil {
				return fmt.Errorf("failed to start persistent shell: %w", err)
			}
			return nil
		}
	}

	// If we get here, the container doesn't exist. We need to build the image, then create and start the container.
	fmt.Println("Building development environment image...")
	cmd = exec.Command("docker", "build", "-t", "dev-env-img", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build docker image: %w\n%s", err, string(output))
	}

	fmt.Println("Creating and starting development container...")
	_, err = cli.ContainerCreate(ctx,
		&container.Config{
			Image:      "dev-env-img",
			Cmd:        []string{"tail", "-f", "/dev/null"},
			WorkingDir: "/home/developer",
			User:       "developer",
			Tty:        false,
		},
		&container.HostConfig{
			Binds: []string{fmt.Sprintf("%s:/home/developer", chrootDir)},
		},
		nil, nil, ContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, ContainerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Println("Starting persistent shell in container...")
	if err := startPersistentShell(); err != nil {
		return fmt.Errorf("failed to start persistent shell: %w", err)
	}

	// Automatically configure git
	fmt.Println("Configuring git in development container...")
	cmdGit := "mkdir -p ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts"
	if _, err := Exec(cmdGit); err != nil {
		// Log the error but don't fail the start-up, as it might not be critical
		fmt.Printf("Warning: Failed to automatically configure git: %v\n", err)
	}

	fmt.Println("Development container and persistent shell started.")
	return nil
}

// startPersistentShell starts a long-running interactive shell inside the container.
func startPersistentShell() error {
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/bash"},
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		User:         "developer",
	}

	execID, err := cli.ContainerExecCreate(ctx, ContainerName, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create persistent shell exec: %w", err)
	}

	// Use a pointer for shellConn
	conn, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach to persistent shell: %w", err)
	}
	shellConn = &conn

	return nil
}

// Exec runs a command in the persistent shell.
func Exec(command string) (string, error) {
	if shellConn == nil || shellConn.Conn == nil {
		return "", fmt.Errorf("no persistent shell available")
	}

	// We send the command to stderr so it doesn't pollute stdout. The marker is also sent to stderr.
	fullCommand := fmt.Sprintf("%s; echo -n '%s' $? >&2\n", command, EndOfCommandMarker)
	_, err := shellConn.Conn.Write([]byte(fullCommand))
	if err != nil {
		return "", fmt.Errorf("failed to write to shell stdin: %w", err)
	}

	var stdout, stderr bytes.Buffer
	header := make([]byte, 8)
	for {
		// Read the 8-byte header
		_, err := io.ReadFull(shellConn.Reader, header)
		if err != nil {
			return "", fmt.Errorf("failed to read stream header: %w", err)
		}

		// Get the payload size
		size := binary.BigEndian.Uint32(header[4:])

		// Read the payload
		payload := make([]byte, size)
		_, err = io.ReadFull(shellConn.Reader, payload)
		if err != nil {
			return "", fmt.Errorf("failed to read stream payload: %w", err)
		}

		// Direct the payload to the correct buffer
		if header[0] == 1 { // stdout
			stdout.Write(payload)
		} else { // stderr
			stderr.Write(payload)
		}

		// Check for our end marker in the stderr stream
		if strings.Contains(stderr.String(), EndOfCommandMarker) {
			break
		}
	}

	stderrStr := stderr.String()
	markerIndex := strings.Index(stderrStr, EndOfCommandMarker)
	if markerIndex == -1 {
		return stdout.String(), fmt.Errorf("could not find end of command marker in stderr")
	}

	markerLine := stderrStr[markerIndex:]
	parts := strings.Split(strings.TrimSpace(markerLine), " ")
	if len(parts) < 2 {
		return stdout.String(), fmt.Errorf("could not parse exit code from shell output")
	}
	exitCodeStr := parts[1]

	if exitCodeStr != "0" {
		errorOutput := strings.TrimSpace(stderrStr[:markerIndex])
		return stdout.String(), fmt.Errorf("command exited with non-zero status %s: %s", exitCodeStr, errorOutput)
	}

	return stdout.String(), nil
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
