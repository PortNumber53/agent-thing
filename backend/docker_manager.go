package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultContainerName = "dev-environment"
	defaultImageName     = "agent-thing-dev"
	dockerCommandTimeout = 2 * time.Minute
)

type DockerManager struct {
	containerName string
	imageName     string
}

type dockerStatusResponse struct {
	Status      string `json:"status"`
	ContainerId string `json:"containerId,omitempty"`
	Details     string `json:"details,omitempty"`
	Message     string `json:"message,omitempty"`
}

type dockerActionResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Status  string `json:"status,omitempty"`
}

func NewDockerManager() *DockerManager {
	return &DockerManager{
		containerName: defaultContainerName,
		imageName:     defaultImageName,
	}
}

func (m *DockerManager) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJson(w, http.StatusMethodNotAllowed, dockerActionResponse{
			Ok:      false,
			Message: "method not allowed",
		})
		return
	}

	status, err := m.getStatus(r.Context())
	if err != nil {
		writeJson(w, http.StatusInternalServerError, dockerStatusResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, status)
}

func (m *DockerManager) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, dockerActionResponse{Ok: false, Message: "method not allowed"})
		return
	}

	if err := m.startContainer(r.Context()); err != nil {
		writeJson(w, http.StatusInternalServerError, dockerActionResponse{Ok: false, Message: err.Error()})
		return
	}

	writeJson(w, http.StatusOK, dockerActionResponse{Ok: true, Message: "container started"})
}

func (m *DockerManager) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, dockerActionResponse{Ok: false, Message: "method not allowed"})
		return
	}

	if err := m.stopContainer(r.Context()); err != nil {
		writeJson(w, http.StatusInternalServerError, dockerActionResponse{Ok: false, Message: err.Error()})
		return
	}

	writeJson(w, http.StatusOK, dockerActionResponse{Ok: true, Message: "container stopped"})
}

func (m *DockerManager) handleRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, dockerActionResponse{Ok: false, Message: "method not allowed"})
		return
	}

	if err := m.rebuildContainer(r.Context()); err != nil {
		writeJson(w, http.StatusInternalServerError, dockerActionResponse{Ok: false, Message: err.Error()})
		return
	}

	writeJson(w, http.StatusOK, dockerActionResponse{Ok: true, Message: "container rebuilt"})
}

func (m *DockerManager) getStatus(ctx context.Context) (dockerStatusResponse, error) {
	output, err := m.runDocker(ctx, "ps", "-a", "--filter", fmt.Sprintf("name=^/%s$", m.containerName), "--format", "{{.ID}}|{{.Status}}")
	if err != nil {
		return dockerStatusResponse{}, err
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return dockerStatusResponse{Status: "not_found"}, nil
	}

	parts := strings.SplitN(trimmed, "|", 2)
	containerId := strings.TrimSpace(parts[0])
	details := ""
	if len(parts) > 1 {
		details = strings.TrimSpace(parts[1])
	}

	status := "stopped"
	if strings.HasPrefix(strings.ToLower(details), "up") {
		status = "running"
	}

	return dockerStatusResponse{
		Status:      status,
		ContainerId: containerId,
		Details:     details,
	}, nil
}

func (m *DockerManager) startContainer(ctx context.Context) error {
	status, err := m.getStatus(ctx)
	if err != nil {
		return err
	}

	switch status.Status {
	case "running":
		return nil
	case "stopped":
		_, err := m.runDocker(ctx, "start", m.containerName)
		return err
	case "not_found":
		if err := m.buildImage(ctx); err != nil {
			return err
		}
		_, err := m.runDocker(ctx, "run", "-d", "--name", m.containerName, m.imageName, "tail", "-f", "/dev/null")
		return err
	default:
		return fmt.Errorf("unexpected status: %s", status.Status)
	}
}

func (m *DockerManager) stopContainer(ctx context.Context) error {
	status, err := m.getStatus(ctx)
	if err != nil {
		return err
	}
	if status.Status == "running" {
		_, err := m.runDocker(ctx, "stop", m.containerName)
		return err
	}
	return nil
}

func (m *DockerManager) rebuildContainer(ctx context.Context) error {
	_, _ = m.runDocker(ctx, "rm", "-f", m.containerName)

	if err := m.buildImage(ctx); err != nil {
		return err
	}

	_, err := m.runDocker(ctx, "run", "-d", "--name", m.containerName, m.imageName, "tail", "-f", "/dev/null")
	return err
}

func (m *DockerManager) buildImage(ctx context.Context) error {
	projectRootDir, err := findProjectRootDir()
	if err != nil {
		return err
	}

	dockerfilePath := filepath.Join(projectRootDir, "Dockerfile")
	if _, statErr := os.Stat(dockerfilePath); statErr != nil {
		return fmt.Errorf("Dockerfile not found at %s; cannot rebuild image", dockerfilePath)
	}

	_, err = m.runDockerWithDir(ctx, projectRootDir, "build", "-t", m.imageName, "-f", dockerfilePath, projectRootDir)
	return err
}

func findProjectRootDir() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to determine working dir: %w", err)
	}
	if filepath.Base(workingDir) == "backend" {
		return filepath.Dir(workingDir), nil
	}
	return workingDir, nil
}

func (m *DockerManager) runDocker(ctx context.Context, args ...string) (string, error) {
	projectRootDir, _ := findProjectRootDir()
	return m.runDockerWithDir(ctx, projectRootDir, args...)
}

func (m *DockerManager) runDockerWithDir(ctx context.Context, dir string, args ...string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, dockerCommandTimeout)
	defer cancel()

	command := exec.CommandContext(timeoutCtx, "docker", args...)
	if dir != "" {
		command.Dir = dir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		errorMessage := strings.TrimSpace(stderr.String())
		if errorMessage == "" {
			errorMessage = err.Error()
		}
		return "", fmt.Errorf("docker %s failed: %s", strings.Join(args, " "), errorMessage)
	}

	return stdout.String(), nil
}
