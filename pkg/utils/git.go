package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func CloneRepo(url string, folderName string) error {
	// Get the full path where the repo will be cloned
	clonePath := filepath.Join(".", folderName)

	// Check if the target directory already exists
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' already exists", clonePath)
	}

	return ExecuteCommandStream("git", "clone", "-b", "test/register-candidate-verify", url, clonePath)
}

func PullLatestCode(folderName string) error {
	// Get the full path of the target directory
	clonePath := filepath.Join(".", folderName)

	// Check if the target directory exists
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		return fmt.Errorf("path '%s' does not exist", clonePath)
	}

	// Change the working directory to the target directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(clonePath); err != nil {
		return fmt.Errorf("failed to change directory to '%s': %v", clonePath, err)
	}

	// Execute the git pull command
	return ExecuteCommandStream("git", "pull")
}
