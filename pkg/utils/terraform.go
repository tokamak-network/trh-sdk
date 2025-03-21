package utils

import (
	"fmt"
	"os"
)

func CheckTerraformState(dir string) (bool, error) {
	stateFilePath := fmt.Sprintf("%s/.terraform/terraform.tfstate", dir)
	if _, err := os.Stat(stateFilePath); os.IsNotExist(err) {
		return false, nil
	}

	_, err := os.ReadFile(stateFilePath)
	if err != nil {
		return false, err
	}

	return true, nil
}
