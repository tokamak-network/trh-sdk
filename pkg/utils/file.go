package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

func CopyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destinationFile.Close()

	// Copy content
	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy content: %v", err)
	}

	// Preserve file permissions
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %v", err)
	}
	if err := os.Chmod(dst, info.Mode()); err != nil {
		return fmt.Errorf("failed to set destination file permissions: %v", err)
	}

	return nil
}

func CheckFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	// If the error is nil, the file exists
	if err == nil {
		return true
	}
	// If the error is not nil, check if it's a "file not found" error
	if os.IsNotExist(err) {
		return false
	}
	// Return false in case of other errors
	return false
}

func CheckDirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println("Error checking directory:", err)
		return false
	}
	return info.IsDir()
}

func ReadConfigFromJSONFile(deploymentPath string) (*types.Config, error) {

	filePath := fmt.Sprintf("%s/%s", deploymentPath, types.ConfigFileName)

	fmt.Println("Reading config from:", filePath)

	fileExist := CheckFileExists(filePath)
	if !fileExist {
		return nil, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// If L2ChainId doesn't exist, fetch it from the L2 RPC
	if config.L2ChainID == 0 {
		l2Provider, err := ethclient.Dial(config.L2RpcUrl)
		if err != nil {
			fmt.Println("Error connecting to L2 blockchain:", err)
			return nil, err
		}

		chainId, err := l2Provider.ChainID(context.Background())
		if err != nil || chainId == nil {
			fmt.Println("Error getting L2 chain id:", err)
			return nil, err
		}
		config.L2ChainID = chainId.Uint64()
	}

	return &config, nil
}

func ReadDeployementConfigFromJSONFile(deploymentPath string, chainId uint64) (*types.Contracts, error) {
	filePath := fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%s", deploymentPath, fmt.Sprintf("%d-deploy.json", chainId))

	fileExist := CheckFileExists(filePath)
	if !fileExist {
		return nil, fmt.Errorf("deployment file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening deployment file:", err)
		return nil, err
	}
	defer file.Close()

	var contracts types.Contracts
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&contracts); err != nil {
		fmt.Println("Error decoding deployment JSON file:", err)
		return nil, err
	}
	return &contracts, nil
}

func ReadMetadataInfoFromJSONFile(deploymentPath string, chainId uint64) (*types.MetadataInfo, error) {
	filePath := fmt.Sprintf("%s/%s", deploymentPath, types.MetadataInfoFileName)

	fileExist := CheckFileExists(filePath)
	if !fileExist {
		return &types.MetadataGenericInfo, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening metadata info file:", err)
		return nil, err
	}
	defer file.Close()

	var metadataInfo types.MetadataInfo
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metadataInfo); err != nil {
		fmt.Println("Error decoding metadata info JSON file:", err)
		return nil, err
	}
	return &metadataInfo, nil
}
