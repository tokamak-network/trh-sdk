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
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type LeaderNodeInput struct {
	PrivateKey string `json:"private_key"`
}

type RegularNodeInput struct {
	PrivateKey string `json:"private_key"`
}

type DeployDRBInput struct {
	RPC              string              `json:"rpc"`
	ChainID          uint64              `json:"chain_id"`
	PrivateKey       string              `json:"private_key"`
	LeaderNodeInput  *LeaderNodeInput    `json:"leader_node_input"`
	RegularNodeInput []*RegularNodeInput `json:"regular_node_input"`
}

type DeployDRBContractsOutput struct {
	ContractAddress string `json:"contract_address"`
	ContractName    string `json:"contract_name"`
	ChainID         uint64 `json:"chain_id"`
}

type DeployDRBApplicationOutput struct {
	LeaderNodeURL   string   `json:"leader_node_url"`
	RegularNodeURLs []string `json:"regular_node_urls"`
}

type DeployDRBOutput struct {
	DeployDRBContractsOutput   *DeployDRBContractsOutput   `json:"deploy_drb_contracts_output"`
	DeployDRBApplicationOutput *DeployDRBApplicationOutput `json:"deploy_drb_application_output"`
}

type DRBDatabaseConfig struct {
	Type          string `json:"type"` // "rds" or "local"
	ConnectionURL string `json:"connection_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	DatabaseName  string `json:"database_name"`
}

func (t *ThanosStack) GetDRBInput(ctx context.Context) (*DeployDRBInput, error) {
	var rpcUrl string
	var chainID uint64

	// Check if current chain config exists
	hasCurrentChain := t.deployConfig != nil &&
		t.deployConfig.L2RpcUrl != "" &&
		t.deployConfig.L2ChainID != 0

	fmt.Println("\n--------------------------------")
	fmt.Println("Network Selection for DRB Deployment")
	fmt.Println("--------------------------------")

	if hasCurrentChain {
		fmt.Printf("Current deployed chain: %s (Chain ID: %d)\n", t.deployConfig.L2RpcUrl, t.deployConfig.L2ChainID)
		fmt.Println("[1] Use current deployed chain")
		fmt.Println("[2] Specify custom network")
		fmt.Print("Please select an option (1-2): ")

		option, err := scanner.ScanInt()
		if err != nil {
			return nil, fmt.Errorf("failed to scan option: %w", err)
		}

		switch option {
		case 1:
			// Use current deployed chain
			rpcUrl = t.deployConfig.L2RpcUrl
			chainID = t.deployConfig.L2ChainID
			fmt.Printf("✅ Using current deployed chain: %s (Chain ID: %d)\n", rpcUrl, chainID)
		case 2:
			// Get custom network configuration
			var err error
			rpcUrl, chainID, err = t.getCustomNetworkInput(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get custom network input: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid option: %d. Please select 1 or 2", option)
		}
	} else {
		// No current chain config, force custom network input
		fmt.Println("No deployed chain configuration found.")
		fmt.Println("Please specify a custom network:")
		var err error
		rpcUrl, chainID, err = t.getCustomNetworkInput(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get custom network input: %w", err)
		}
	}

	// Ask user to enter the private key
	fmt.Print("\nPlease enter your private key: ")
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

	l2RpcClient, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}
	defer l2RpcClient.Close()

	deployerAddress := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

	// Get balance of this private key on the selected network
	balance, err := l2RpcClient.BalanceAt(ctx, deployerAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("balance is 0 on chain %d, please enter a valid private key with balance", chainID)
	}

	fmt.Printf("✅ Private key validated. Address: %s, Balance: %.4f ETH\n", deployerAddress.Hex(), utils.WeiToEther(balance))

	return &DeployDRBInput{
		RPC:        rpcUrl,
		ChainID:    chainID,
		PrivateKey: "0x" + privateKey,
	}, nil
}

// getCustomNetworkInput prompts user for custom network RPC URL and Chain ID
func (t *ThanosStack) getCustomNetworkInput(ctx context.Context) (string, uint64, error) {
	var rpcUrl string
	var chainID uint64
	var err error

	// Get RPC URL with validation
	for {
		fmt.Print("Please enter the RPC URL: ")
		rpcUrl, err = scanner.ScanString()
		if err != nil {
			return "", 0, fmt.Errorf("failed to scan RPC URL: %w", err)
		}

		if rpcUrl == "" {
			fmt.Println("RPC URL cannot be empty. Please try again.")
			continue
		}

		// Validate RPC URL
		if !utils.IsValidL1RPC(rpcUrl) {
			fmt.Println("Invalid RPC URL. Please try again.")
			continue
		}

		break
	}

	// Auto-detect Chain ID from RPC
	// GetChainIDFromL1RPC works with any RPC URL (not just L1), so we use it directly
	fmt.Println("Detecting Chain ID from RPC...")
	detectedChainID, err := utils.GetChainIDFromL1RPC(rpcUrl)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get chain ID from RPC: %w", err)
	}

	fmt.Printf("Detected Chain ID: %d\n", detectedChainID)
	fmt.Print("Do you want to use the detected Chain ID? (Y/n): ")

	useDetected, err := scanner.ScanBool(true)
	if err != nil {
		return "", 0, fmt.Errorf("failed to scan confirmation: %w", err)
	}

	if useDetected {
		chainID = detectedChainID
	} else {
		// Allow manual Chain ID entry
		for {
			fmt.Print("Please enter the Chain ID: ")
			chainIDInput, err := scanner.ScanInt()
			if err != nil {
				fmt.Printf("Invalid Chain ID: %s. Please try again.\n", err)
				continue
			}

			if chainIDInput <= 0 {
				fmt.Println("Chain ID must be greater than 0. Please try again.")
				continue
			}

			// Verify the manual Chain ID matches the RPC
			if uint64(chainIDInput) != detectedChainID {
				fmt.Printf("Warning: Entered Chain ID (%d) does not match detected Chain ID (%d)\n", chainIDInput, detectedChainID)
				fmt.Print("Do you want to continue with the entered Chain ID? (y/N): ")
				confirm, err := scanner.ScanBool(false)
				if err != nil {
					return "", 0, fmt.Errorf("failed to scan confirmation: %w", err)
				}
				if !confirm {
					continue
				}
			}

			chainID = uint64(chainIDInput)
			break
		}
	}

	fmt.Printf("✅ Network configured: %s (Chain ID: %d)\n", rpcUrl, chainID)
	return rpcUrl, chainID, nil
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

// GetDRBDatabaseInput prompts user to select database type and collect configuration
func (t *ThanosStack) GetDRBDatabaseInput(ctx context.Context) (*DRBDatabaseConfig, error) {
	fmt.Println("\n--------------------------------")
	fmt.Println("Database Selection for DRB")
	fmt.Println("--------------------------------")
	fmt.Println("[1] AWS RDS PostgreSQL")
	fmt.Println("[2] Local PostgreSQL (Helm Chart)")
	fmt.Print("Please select database type (1-2): ")

	option, err := scanner.ScanInt()
	if err != nil {
		return nil, fmt.Errorf("failed to scan option: %w", err)
	}

	switch option {
	case 1:
		return t.getRDSDatabaseInput(ctx)
	case 2:
		return t.getDRBLocalDatabaseInput(ctx)
	default:
		return nil, fmt.Errorf("invalid option: %d. Please select 1 or 2", option)
	}
}

// getRDSDatabaseInput collects RDS PostgreSQL configuration
func (t *ThanosStack) getRDSDatabaseInput(ctx context.Context) (*DRBDatabaseConfig, error) {
	var databaseUsername, databasePassword string
	var err error

	// Get database username
	for {
		fmt.Print("Please enter database username: ")
		databaseUsername, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to scan database username: %w", err)
		}
		databaseUsername = strings.ToLower(databaseUsername)

		if databaseUsername == "" {
			fmt.Println("Database username cannot be empty")
			continue
		}

		if err := utils.ValidatePostgresUsername(databaseUsername); err != nil {
			fmt.Printf("Database username is invalid: %s\n", err.Error())
			continue
		}

		if !utils.IsValidRDSUsername(databaseUsername) {
			fmt.Println("Database username is invalid, please try again")
			continue
		}
		break
	}

	// Get database password
	for {
		fmt.Print("Please enter database password: ")
		databasePassword, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to scan database password: %w", err)
		}

		if databasePassword == "" {
			fmt.Println("Database password cannot be empty")
			continue
		}

		if !utils.IsValidRDSPassword(databasePassword) {
			fmt.Println("Database password is invalid, please try again")
			continue
		}
		break
	}

	return &DRBDatabaseConfig{
		Type:         "rds",
		Username:     databaseUsername,
		Password:     databasePassword,
		DatabaseName: "drb",
	}, nil
}

// getDRBLocalDatabaseInput collects local PostgreSQL configuration
func (t *ThanosStack) getDRBLocalDatabaseInput(ctx context.Context) (*DRBDatabaseConfig, error) {
	var databasePassword string
	var err error

	// Get database password
	for {
		fmt.Print("Please enter database password: ")
		databasePassword, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to scan database password: %w", err)
		}

		if databasePassword == "" {
			fmt.Println("Database password cannot be empty")
			continue
		}
		break
	}

	return &DRBDatabaseConfig{
		Type:         "local",
		Username:     "postgres",
		Password:     databasePassword,
		DatabaseName: "drb",
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
	if t.deployConfig.K8s == nil {
		return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	// Get database configuration
	dbConfig, err := t.GetDRBDatabaseInput(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database configuration: %w", err)
	}

	// Clone the tokamak-thanos-stack repository
	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone tokamak-thanos-stack repository: %s", err)
	}

	// Checkout to `feat/add-drb-node`
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd tokamak-thanos-stack && git checkout feat/add-drb-node")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout feat/add-drb-node: %s", err)
	}

	// Deploy database based on type
	var connectionURL string
	switch dbConfig.Type {
	case "rds":
		connectionURL, err = t.deployDRBDatabaseRDS(ctx, dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy RDS database: %w", err)
		}
	case "local":
		connectionURL, err = t.deployDRBDatabaseLocal(ctx, dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy local database: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}

	dbConfig.ConnectionURL = connectionURL
	t.logger.Infof("✅ Database deployed successfully. Connection URL: %s", connectionURL)

	// TODO: Deploy DRB leader and regular nodes using the database configuration

	return &DeployDRBApplicationOutput{
		LeaderNodeURL:   "",
		RegularNodeURLs: []string{},
	}, nil
}

// deployDRBDatabaseRDS deploys AWS RDS PostgreSQL for DRB
func (t *ThanosStack) deployDRBDatabaseRDS(ctx context.Context, dbConfig *DRBDatabaseConfig) (string, error) {
	vpcId := t.deployConfig.AWS.VpcID

	if vpcId == "" {
		return "", fmt.Errorf("VPC ID is not set in deploy config")
	}

	t.logger.Info("Deploying AWS RDS PostgreSQL for DRB...")

	// Create .envrc file for Terraform
	err := t.makeDRBEnvs(
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath),
		".envrc",
		types.AwsDatabaseEnvs{
			DatabaseUserName: dbConfig.Username,
			DatabasePassword: dbConfig.Password,
			DatabaseName:     dbConfig.DatabaseName,
			VpcId:            vpcId,
			AwsRegion:        t.deployConfig.AWS.Region,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create .envrc file: %w", err)
	}

	// Deploy RDS using Terraform
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd drb &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve
		`, t.deploymentPath),
	}...)
	if err != nil {
		return "", fmt.Errorf("failed to deploy RDS: %w", err)
	}

	// Get RDS connection URL from Terraform output
	rdsConnectionUrl, err := utils.ExecuteCommand(ctx, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd drb &&	
		terraform output -json rds_connection_url`, t.deploymentPath),
	}...)
	if err != nil {
		return "", fmt.Errorf("failed to get RDS connection URL: %w", err)
	}

	rdsConnectionUrl = strings.Trim(rdsConnectionUrl, `"`)
	t.logger.Info("✅ RDS PostgreSQL deployed successfully")
	return rdsConnectionUrl, nil
}

// deployDRBDatabaseLocal deploys local PostgreSQL using Helm chart (TODO: Implement)
func (t *ThanosStack) deployDRBDatabaseLocal(ctx context.Context, dbConfig *DRBDatabaseConfig) (string, error) {
	return "", nil
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
