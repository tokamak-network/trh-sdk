package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ShutdownState represents the state of shutdown operations
type ShutdownState struct {
	ChainID         uint64 `json:"chainId"`
	L2ChainID       uint64 `json:"l2ChainId"`
	ThanosRoot      string `json:"thanosRoot"`
	DeploymentsPath string `json:"deploymentsPath"`
	DataDir         string `json:"dataDir"`
	LastGenAt       string `json:"lastGenAt,omitempty"`
	LastDryRunAt    string `json:"lastDryRunAt,omitempty"`
	LastSendAt      string `json:"lastSendAt,omitempty"`
	LastSnapshotPath string `json:"lastSnapshotPath,omitempty"`
	LastCommand     string `json:"lastCommand,omitempty"`
}

// GetShutdownStateFilePath returns the path to the shutdown state file
func GetShutdownStateFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	trhDir := filepath.Join(homeDir, ".trh")
	if err := os.MkdirAll(trhDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .trh directory: %w", err)
	}

	return filepath.Join(trhDir, "thanos_shutdown_state.json"), nil
}

// LoadShutdownState loads the shutdown state from file
func LoadShutdownState() (*ShutdownState, error) {
	filePath, err := GetShutdownStateFilePath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty state
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &ShutdownState{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ShutdownState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// SaveShutdownState saves the shutdown state to file
func (s *ShutdownState) Save() error {
	filePath, err := GetShutdownStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write atomically by writing to temp file first
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}

	return nil
}

// UpdateAfterGen updates state after successful gen command
func (s *ShutdownState) UpdateAfterGen(snapshotPath string) {
	s.LastGenAt = time.Now().UTC().Format(time.RFC3339)
	s.LastSnapshotPath = snapshotPath
	s.LastCommand = "gen"
}

// UpdateAfterDryRun updates state after successful dry-run command
func (s *ShutdownState) UpdateAfterDryRun() {
	s.LastDryRunAt = time.Now().UTC().Format(time.RFC3339)
	s.LastCommand = "dry-run"
}

// UpdateAfterSend updates state after successful send command
func (s *ShutdownState) UpdateAfterSend() {
	s.LastSendAt = time.Now().UTC().Format(time.RFC3339)
	s.LastCommand = "send"
}
