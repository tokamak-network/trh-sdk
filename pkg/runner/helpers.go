package runner

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeTempManifest writes manifest bytes to a temp file and returns its path.
func writeTempManifest(manifest []byte) (string, error) {
	f, err := os.CreateTemp("", "trh-manifest-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := f.Write(manifest); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}
	return filepath.Clean(f.Name()), nil
}

// removeTempFile silently removes a temp file created by writeTempManifest.
func removeTempFile(path string) {
	_ = os.Remove(path)
}
