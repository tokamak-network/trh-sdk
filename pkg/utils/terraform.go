package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/types"
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

func ReadThanosStackTerraformState(filePath string) (types.ThanosStackTerraformState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return types.ThanosStackTerraformState{}, fmt.Errorf("failed to read file: %w", err)
	}

	var state types.ThanosStackTerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return types.ThanosStackTerraformState{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return state, nil
}
