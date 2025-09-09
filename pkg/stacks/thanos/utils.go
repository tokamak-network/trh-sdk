package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

var estimatedDeployContracts = new(big.Int).SetInt64(80_000_000)
var zeroBalance = new(big.Int).SetInt64(0)

var mapAccountIndexes = map[int]string{
	0: "Admin",
	1: "Sequencer",
	2: "Batcher",
	3: "Proposer",
	4: "Challenger",
}

func displayAccounts(accounts map[int]types.Account) {
	sortedAccounts := make([]types.Account, len(accounts))
	for i, account := range accounts {
		sortedAccounts[i] = account
	}

	for i, account := range sortedAccounts {
		balance, _ := new(big.Int).SetString(account.Balance, 10)
		fmt.Printf("\t%d. %s(%.4f ETH)\n", i, account.Address, utils.WeiToEther(balance))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (t *ThanosStack) getContractAddressFromOutput(_ context.Context, repositoryName, deployFile string, chainID uint64) (map[string]string, error) {
	// Construct the file path
	filePath := fmt.Sprintf("%s/%s/broadcast/%s/%d/run-latest.json", t.deploymentPath, repositoryName, deployFile, chainID)

	// Open and read the file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to open deployment file %s: %w", filePath, err)
	}
	defer file.Close()

	// Parse the JSON structure
	var deploymentData map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&deploymentData); err != nil {
		return nil, fmt.Errorf("failed to decode deployment JSON: %w", err)
	}

	// Extract the transactions array
	transactions, ok := deploymentData["transactions"].([]any)
	if !ok {
		return nil, fmt.Errorf("transactions field not found or not an array in deployment file")
	}

	// Collect all contract addresses from CREATE transactions
	contractAddresses := make(map[string]string)

	// Loop through transactions to find CREATE type
	for _, tx := range transactions {
		txMap, ok := tx.(map[string]any)
		if !ok {
			continue
		}

		// Check if transaction type is CREATE
		txType, ok := txMap["transactionType"].(string)
		if !ok || txType != "CREATE" {
			continue
		}

		// Extract contract address
		contractAddress, ok := txMap["contractAddress"].(string)
		if !ok || contractAddress == "" {
			continue
		}

		contractAddresses[txMap["contractName"].(string)] = contractAddress
	}

	if len(contractAddresses) == 0 {
		return nil, fmt.Errorf("no CREATE transaction found in deployment file")
	}

	return contractAddresses, nil
}
