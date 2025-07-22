package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func CloneRepo(ctx context.Context, l *zap.SugaredLogger, deploymentPath string, url string, folderName string, branch string) error {
	var clonePath string
	if filepath.IsAbs(deploymentPath) {
		clonePath = filepath.Join(deploymentPath, folderName)
	} else {
		clonePath = filepath.Join(".", deploymentPath, folderName)
	}

	// Check if the target directory already exists
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		return fmt.Errorf("destination path '%s' already exists", clonePath)
	}

	return ExecuteCommandStream(ctx, l, "git", "clone", "--branch", branch, url, clonePath)
}

func PullLatestCode(ctx context.Context, l *zap.SugaredLogger, deploymentPath string, folderName string) error {
	var clonePath string
	if filepath.IsAbs(deploymentPath) {
		clonePath = filepath.Join(deploymentPath, folderName)
	} else {
		clonePath = filepath.Join(".", deploymentPath, folderName)
	}

	// Check if the target directory exists
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		return fmt.Errorf("path '%s' does not exist", clonePath)
	}

	// Save the current working directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to the target directory
	if err := os.Chdir(clonePath); err != nil {
		return fmt.Errorf("failed to change directory to '%s': %v", clonePath, err)
	}

	// Execute the git pull command
	return ExecuteCommandStream(ctx, l, "git", "pull")
}
