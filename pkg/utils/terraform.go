package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

type LockInfo struct {
	ID        string `json:"ID"`
	Operation string `json:"Operation"`
	Info      string `json:"Info"`
	Who       string `json:"Who"`
	Version   string `json:"Version"`
	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
	Path      string `json:"Path"`
}

type DynamoDBItem struct {
	LockID struct {
		S string `json:"S"`
	} `json:"LockID"`
	Info struct {
		S string `json:"S"`
	}
}

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

func GetTerraformLockID(ctx context.Context, table string, bucketName string) (string, error) {
	data, err := ExecuteCommand(ctx, "bash", []string{
		"-c",
		fmt.Sprintf(`aws dynamodb scan --table-name %s --output json`, table),
	}...)
	if err != nil {
		return "", err
	}

	var result struct {
		Items []DynamoDBItem `json:"Items"`
	}

	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	for _, item := range result.Items {
		if strings.Contains(item.LockID.S, bucketName) {
			var lockInfo LockInfo
			if err := json.Unmarshal([]byte(item.Info.S), &lockInfo); err != nil {
				return "", fmt.Errorf("failed to parse JSON: %w", err)
			}
			return lockInfo.ID, nil
		}
	}

	return "", nil
}
