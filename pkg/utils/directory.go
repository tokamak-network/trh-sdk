package utils

import "os"

func CheckExistingSourceCode(folderName string) (bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, err
	}

	// Construct full path
	fullPath := currentDir + "/" + folderName

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
