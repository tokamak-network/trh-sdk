package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		if err := json.Unmarshal([]byte(str), &container); err != nil {
			return nil, fmt.Errorf("failed to unmarshal docker container: %w", err)
		}

		if container.State == "running" {
			runningContainers = append(runningContainers, container.Names)
		}
	}

	return runningContainers, nil
}
