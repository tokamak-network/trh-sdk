package types

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteToJSONFile_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	deploymentPath := filepath.Join(tmpDir, "deployments", "Thanos", "Testnet", "some-stack-id")

	cfg := &Config{}
	if err := cfg.WriteToJSONFile(deploymentPath); err != nil {
		t.Fatalf("WriteToJSONFile failed: %v", err)
	}

	settingsPath := filepath.Join(deploymentPath, ConfigFileName)
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatalf("settings.json not created at %s", settingsPath)
	}
}
