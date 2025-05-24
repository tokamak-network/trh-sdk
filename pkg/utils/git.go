package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func CloneRepo(l *zap.SugaredLogger, deploymentPath string, url string, folderName string) error {
	// Get the full path where the repo will be cloned
	clonePath := filepath.Join(".", deploymentPath, folderName)

	// Check if the target directory already exists
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' already exists", clonePath)
	}

	return ExecuteCommandStream(l, "git", "clone", url, clonePath)
}

func PullLatestCode(l *zap.SugaredLogger, deploymentPath string, folderName string) error {
	// Get the full path of the target directory
	clonePath := filepath.Join(".", deploymentPath, folderName)

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
	return ExecuteCommandStream(l, "git", "pull")
}
