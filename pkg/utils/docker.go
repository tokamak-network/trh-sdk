package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type DockerContainer struct {
	Command      string `json:"Command"`
	CreatedAt    string `json:"CreatedAt"`
	ID           string `json:"ID"`
	Image        string `json:"Image"`
	Labels       string `json:"Labels"`
	LocalVolumes string `json:"LocalVolumes"`
	Mounts       string `json:"Mounts"`
	Names        string `json:"Names"`
	Networks     string `json:"Networks"`
	Ports        string `json:"Ports"`
	RunningFor   string `json:"RunningFor"`
	Size         string `json:"Size"`
	State        string `json:"State"`
	Status       string `json:"Status"`
}

func GetDockerContainers(ctx context.Context) ([]string, error) {
	containersOutput, err := ExecuteCommand("docker", "ps", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to get docker containers: %w", err)
	}
	runningContainers := make([]string, 0)

	containersStr := strings.Split(strings.TrimSpace(containersOutput), "\n")
	for _, str := range containersStr {
		var container DockerContainer
		if strings.TrimSpace(str) == "" {
			continue
		}

		if err := json.Unmarshal([]byte(str), &container); err != nil {
			return nil, fmt.Errorf("failed to unmarshal docker container: %w", err)
		}

		if container.State == "running" {
			runningContainers = append(runningContainers, container.Names)
		}
	}

	return runningContainers, nil
}

// Check if Docker is running and start it if necessary
func EnsureDockerReady() error {
	// Check OS
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil
	}

	// Check if Docker daemon is running
	if _, err := ExecuteCommand("docker", "info"); err == nil {
		return nil // already running
	}

	// Start Docker
	if err := startDocker(); err != nil {
		return err
	}

	// Wait until Docker is ready
	maxRetries := 30
	if runtime.GOOS == "darwin" {
		maxRetries = 60 // macOS Docker Desktop
	}

	for i := 0; i < maxRetries; i++ {
		if _, err := ExecuteCommand("docker", "info"); err == nil {
			// Docker is running, fix socket permissions if needed
			fixDockerSocketPermissions()
			return nil // success
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("docker failed to start within timeout")
}

func startDocker() error {
	switch runtime.GOOS {
	case "darwin":
		// Check Docker Desktop paths
		dockerPaths := []string{"/Applications/Docker.app", "/System/Applications/Docker.app"}
		for _, path := range dockerPaths {
			if _, err := os.Stat(path); err == nil {
				// Check if already running
				if _, err := ExecuteCommand("pgrep", "-f", "Docker Desktop"); err == nil {
					return nil
				}
				_, err := ExecuteCommand("open", "-a", path)
				return err
			}
		}

		// Check colima
		if _, err := ExecuteCommand("which", "colima"); err == nil {
			_, err := ExecuteCommand("colima", "start")
			return err
		}

		return fmt.Errorf("docker Desktop not found. Please install from https://docker.com/products/docker-desktop")

	case "linux":
		// Check systemd
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			_, err := ExecuteCommand("sudo", "systemctl", "start", "docker")
			return err
		}

		// Fallback for non-systemd systems
		cmd := exec.Command("sudo", "dockerd")
		return cmd.Start()

	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// fixDockerSocketPermissions fixes Docker socket permission issues
func fixDockerSocketPermissions() {
	switch runtime.GOOS {
	case "linux":
		fixLinuxDockerPermissions()
	}
	// darwin: Docker Desktop in macOS is automatically managing docker socket
}

// fixLinuxDockerPermissions fixes Docker socket permissions on Linux
func fixLinuxDockerPermissions() {
	dockerSock := "/var/run/docker.sock"

	// Check if socket exists
	if _, err := os.Stat(dockerSock); os.IsNotExist(err) {
		return
	}

	// Fix socket permissions (ignore errors - best effort)
	ExecuteCommand("sudo", "chmod", "666", dockerSock)
	ExecuteCommand("sudo", "chgrp", "docker", dockerSock)
}
