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

	return ExecuteCommandStream("git", "clone", url, clonePath)
}
