package thanos

import (
	"encoding/json"
	"fmt"
	"os"
)

// forgeArtifact represents the JSON structure of a forge build artifact.
type forgeArtifact struct {
	DeployedBytecode struct {
		Object string `json:"object"`
	} `json:"deployedBytecode"`
}

// loadForgeArtifactBytecode reads a forge build artifact JSON and returns the deployedBytecode.
func loadForgeArtifactBytecode(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read artifact %s: %w", path, err)
	}

	var artifact forgeArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return "", fmt.Errorf("failed to parse artifact %s: %w", path, err)
	}

	if artifact.DeployedBytecode.Object == "" {
		return "", fmt.Errorf("empty deployedBytecode in %s", path)
	}

	return artifact.DeployedBytecode.Object, nil
}
