package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type DeployDRBInput struct {
	RPC        string `json:"rpc"`
	ChainID    uint64 `json:"chain_id"`
	PrivateKey string `json:"private_key"`
}

type DeployDRBContractsOutput struct {
	ContractAddress string `json:"contract_address"`
	ContractName    string `json:"contract_name"`
	ChainID         uint64 `json:"chain_id"`
}

type DeployDRBApplicationOutput struct {
	URL string `json:"url"`
}

type DeployDRBOutput struct {
	DeployDRBContractsOutput   *DeployDRBContractsOutput   `json:"deploy_drb_contracts_output"`
	DeployDRBApplicationOutput *DeployDRBApplicationOutput `json:"deploy_drb_application_output"`
}

func (t *ThanosStack) GetDRBInput(ctx context.Context) (*DeployDRBInput, error) {
	// Ask user to enter the private key
	fmt.Print("Please enter your private key: ")
	privateKey, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to scan private key: %s", err)
	}

	if privateKey == "" {
		return nil, fmt.Errorf("private key cannot be empty")
	}

	privateKey = strings.TrimPrefix(privateKey, "0x")

	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	l2RpcClient, err := ethclient.DialContext(ctx, t.deployConfig.L2RpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}

	deployerAddress := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

	// Get balance of this private key
	balance, err := l2RpcClient.BalanceAt(ctx, deployerAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("balance is 0, please enter a valid private key")
	}

	return &DeployDRBInput{
		RPC:        t.deployConfig.L2RpcUrl,
		ChainID:    t.deployConfig.L2ChainID,
		PrivateKey: t.deployConfig.AdminPrivateKey,
	}, nil
}

func (t *ThanosStack) DeployDRB(ctx context.Context, inputs *DeployDRBInput) (*DeployDRBOutput, error) {
	deployDRBContractsOutput, err := t.deployDRBContracts(ctx, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy drb contracts: %s", err)
	}

	deployDRBApplicationOutput, err := t.deployDRBApplication(ctx, inputs, deployDRBContractsOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy drb application: %s", err)
	}

	return &DeployDRBOutput{
		DeployDRBContractsOutput:   deployDRBContractsOutput,
		DeployDRBApplicationOutput: deployDRBApplicationOutput,
	}, nil
}

func (t *ThanosStack) UninstallDRB(ctx context.Context) error {
	return nil
}

func (t *ThanosStack) deployDRBContracts(ctx context.Context, inputs *DeployDRBInput) (*DeployDRBContractsOutput, error) {
	// Clone the drb repository
	err := t.cloneSourcecode(ctx, "Commit-Reveal2", "https://github.com/tokamak-network/Commit-Reveal2.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone drb repository: %s", err)
	}

	// Checkout to `service`
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd Commit-Reveal2 && git checkout service")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout service: %s", err)
	}

	t.logger.Info("Start to build drb contracts")

	// Build the contracts
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd Commit-Reveal2 && make install && make build")
	if err != nil {
		return nil, fmt.Errorf("failed to build the contracts: %s", err)
	}

	// Validate private key and get deployer address
	privateKey := strings.TrimPrefix(inputs.PrivateKey, "0x")
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	deployerAddress := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

	// Create .env file
	envContent := fmt.Sprintf(`
	# Deployer Configuration
	PRIVATE_KEY=%s
	DEPLOYER=%s
	RPC_URL=%s`, privateKey, deployerAddress.Hex(), inputs.RPC)

	envFilePath := filepath.Join(t.deploymentPath, "Commit-Reveal2", ".env")
	err = os.WriteFile(envFilePath, []byte(envContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create .env file: %w", err)
	}

	t.logger.Info("Deploying DRB contracts")

	// Run forge script to deploy the contracts
	script := fmt.Sprintf(
		"cd Commit-Reveal2 && forge script script/DeployCommitReveal2.s.sol:DeployCommitReveal2 --rpc-url %s --private-key %s --broadcast -vv",
		inputs.RPC,
		privateKey,
	)
	t.logger.Infof("Deploying DRB contracts with script: %s", script)
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy the contracts: %s", err)
	}

	// Get the contract address from the output
	// Commit-Reveal2/broadcast/DeployCommitReveal2.s.sol/{chain_id}/run-latest.json
	contractAddresses, err := t.getDRBContractAddressFromOutput(ctx, "DeployCommitReveal2.s.sol", inputs.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the contract address: %s", err)
	}

	// Determine contract name based on chain ID
	// If chain_id = Sepolia or mainnet, the contract name is CommitReveal2
	// Otherwise, the contract name is CommitReveal2L2
	contractName := "CommitReveal2L2"
	if inputs.ChainID == constants.EthereumMainnetChainID || inputs.ChainID == constants.EthereumSepoliaChainID {
		contractName = "CommitReveal2"
	}

	contractAddress, ok := contractAddresses[contractName]
	if !ok {
		return nil, fmt.Errorf("contract address not found in deployment output")
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("contract address not found in deployment output")
	}

	t.logger.Infof("DRB contract deployed: %s at address %s", contractName, contractAddress)

	return &DeployDRBContractsOutput{
		ContractAddress: contractAddress,
		ContractName:    contractName,
		ChainID:         inputs.ChainID,
	}, nil
}

func (t *ThanosStack) deployDRBApplication(ctx context.Context, inputs *DeployDRBInput, contracts *DeployDRBContractsOutput) (*DeployDRBApplicationOutput, error) {
	// For now, return a placeholder output
	// This can be extended later to deploy a DRB application similar to cross-trade

	// Clone the drb-node repository
	err := t.cloneSourcecode(ctx, "drb-node", "https://github.com/tokamak-network/DRB-node.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone drb-node repository: %s", err)
	}

	// Checkout to `service`
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd DRB-node && git checkout dispute-mechanism")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout dispute-mechanism: %s", err)
	}

	return &DeployDRBApplicationOutput{
		URL: "",
	}, nil
}

func (t *ThanosStack) getDRBContractAddressFromOutput(_ context.Context, deployFile string, chainID uint64) (map[string]string, error) {
	// Construct the file path
	filePath := fmt.Sprintf("%s/Commit-Reveal2/broadcast/%s/%d/run-latest.json", t.deploymentPath, deployFile, chainID)

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

		contractName, ok := txMap["contractName"].(string)
		if ok && contractName != "" {
			contractAddresses[contractName] = contractAddress
		}
	}

	if len(contractAddresses) == 0 {
		return nil, fmt.Errorf("no CREATE transaction found in deployment file")
	}

	return contractAddresses, nil
}
