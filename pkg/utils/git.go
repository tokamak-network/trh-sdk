package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func CloneRepo(url string, folderName string) error {
	// Get the full path where the repo will be cloned
	clonePath := filepath.Join(".", folderName)

	// Check if the target directory already exists
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' already exists", clonePath)
	}

	// Run the git clone command
	cmd := exec.Command("git", "clone", url, clonePath)

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}
