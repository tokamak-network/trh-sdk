package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

func ConvertChainNameToNamespace(chainName string) string {
	processed := strings.ToLower(chainName)
	processed = strings.ReplaceAll(processed, " ", "-")
	processed = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(processed, "")
	processed = strings.Trim(processed, "-")
	if len(processed) > 20 {
		processed = processed[:20]
	}
	return fmt.Sprintf("%s-%d", processed, time.Now().Unix())
}
