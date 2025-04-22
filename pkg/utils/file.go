package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

func CopyFile(src, dst string, logFileName string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		LogToFile(logFileName, fmt.Sprintf("failed to open source file: %v", err), true)
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		LogToFile(logFileName, fmt.Sprintf("failed to create destination file: %v", err), true)
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destinationFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		LogToFile(logFileName, fmt.Sprintf("failed to copy content: %v", err), true)
		return fmt.Errorf("failed to copy content: %v", err)
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

func ReadConfigFromJSONFile() (*types.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	fileExist := CheckFileExists(fmt.Sprintf("%s/%s", cwd, types.ConfigFileName))
	if !fileExist {
		return nil, nil
	}

	data, err := os.ReadFile(types.ConfigFileName)
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

func ConvertChainNameToNamespace(chainName string) string {
	processed := strings.ToLower(chainName)
	processed = strings.ReplaceAll(processed, " ", "-")
	processed = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(processed, "")
	processed = strings.Trim(processed, "-")
	if len(processed) > 63 {
		processed = processed[:63]
	}
	return processed
}
