package utils

import (
	"os"
	"path/filepath"
)

func CheckExistingSourceCode(deploymentPath string, folderName string) (bool, error) {
	var fullPath string
	if filepath.IsAbs(deploymentPath) {
		fullPath = filepath.Join(deploymentPath, folderName)
	} else {
		fullPath = filepath.Join(".", deploymentPath, folderName)
	}

	// Check if the folder exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check if it's a directory
	return info.IsDir(), nil
}
