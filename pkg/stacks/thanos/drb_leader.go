package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) GetDRBInput(ctx context.Context) (*types.DeployDRBInput, error) {
	fmt.Println("\n--------------------------------")
	fmt.Println("Network Selection for DRB Deployment")
	fmt.Println("--------------------------------")
	fmt.Println("Please specify the network for DRB deployment:")

	// Always get custom network input (DRB is independent of L2 chain)
	rpcUrl, chainID, err := t.getCustomNetworkInput(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom network input: %w", err)
	}

	// Collect node inputs
	fmt.Println("\n--------------------------------")
	fmt.Println("DRB Node Configuration")
	fmt.Println("--------------------------------")

	// Get leader node private key (used for contract deployment as leader should be contract owner)
	fmt.Print("Please enter the leader node private key: ")
	leaderPrivateKey, err := scanner.ScanPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to scan leader node private key: %w", err)
	}
	if leaderPrivateKey == "" {
		return nil, fmt.Errorf("leader node private key cannot be empty")
	}
	leaderPrivateKey = strings.TrimPrefix(leaderPrivateKey, "0x")

	// Validate leader private key format
	leaderPrivateKeyECDSA, err := crypto.HexToECDSA(leaderPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid leader node private key: %w", err)
	}

	// Validate leader key has balance (needed for contract deployment)
	// Use first RPC URL for balance checking
	firstRpcUrl := strings.Split(rpcUrl, ",")[0]
	firstRpcUrl = strings.TrimSpace(firstRpcUrl)
	l2RpcClient, err := ethclient.DialContext(ctx, firstRpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}
	defer l2RpcClient.Close()

	leaderAddress := crypto.PubkeyToAddress(leaderPrivateKeyECDSA.PublicKey)

	// Get balance of leader private key on the selected network
	balance, err := l2RpcClient.BalanceAt(ctx, leaderAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("leader node balance is 0 on chain %d, please enter a valid private key with balance", chainID)
	}

	fmt.Printf("✅ Leader node private key validated. Address: %s, Balance: %.4f ETH\n", leaderAddress.Hex(), utils.WeiToEther(balance))

	// Format leader private key with 0x prefix for consistency
	if !strings.HasPrefix(leaderPrivateKey, "0x") {
		leaderPrivateKey = "0x" + leaderPrivateKey
	}

	// Collect database configuration
	fmt.Println("\n--------------------------------")
	fmt.Println("Database Selection for DRB")
	fmt.Println("--------------------------------")
	fmt.Println("[1] AWS RDS PostgreSQL")
	fmt.Println("[2] Local PostgreSQL (Helm Chart)")
	fmt.Print("Please select database type (1-2): ")

	dbOption, err := scanner.ScanInt()
	if err != nil {
		return nil, fmt.Errorf("failed to scan database option: %w", err)
	}

	// Get database password with confirmation
	fmt.Println("\n--------------------------------")
	fmt.Println("Database Password")
	fmt.Println("--------------------------------")
	dbPassword, err := scanner.ScanPasswordWithConfirmation()
	if err != nil {
		return nil, fmt.Errorf("failed to scan database password: %w", err)
	}
	if dbPassword == "" {
		return nil, fmt.Errorf("database password cannot be empty")
	}

	var dbConfig *types.DRBDatabaseConfig
	switch dbOption {
	case 1:
		// Validate RDS password format
		if !utils.IsValidRDSPassword(dbPassword) {
			return nil, fmt.Errorf("database password is invalid. RDS password must be 8-128 characters and cannot contain /, ', \", @, or spaces")
		}
		dbConfig = &types.DRBDatabaseConfig{
			Type:         "rds",
			Username:     "postgres",
			Password:     dbPassword,
			DatabaseName: "drb",
		}
	case 2:
		dbConfig = &types.DRBDatabaseConfig{
			Type:         "local",
			Username:     "postgres",
			Password:     dbPassword,
			DatabaseName: "drb",
		}
	default:
		return nil, fmt.Errorf("invalid database option: %d. Please select 1 or 2", dbOption)
	}

	return &types.DeployDRBInput{
		RPC:             rpcUrl,
		ChainID:         chainID,
		PrivateKey:      leaderPrivateKey, // Use leader key for contract deployment
		LeaderNodeInput: &types.LeaderNodeInput{PrivateKey: leaderPrivateKey},
		DatabaseConfig:  dbConfig,
	}, nil
}

// getCustomNetworkInput prompts user for custom network RPC URL(s) and Chain ID
func (t *ThanosStack) getCustomNetworkInput(ctx context.Context) (string, uint64, error) {
	var rpcUrlsInput string
	var chainID uint64
	var err error

	// Get RPC URL(s) with validation
	for {
		fmt.Print("Please enter the RPC URL(s) (comma-separated for multiple URLs): ")
		rpcUrlsInput, err = scanner.ScanString()
		if err != nil {
			return "", 0, fmt.Errorf("failed to scan RPC URL: %w", err)
		}

		if rpcUrlsInput == "" {
			fmt.Println("RPC URL cannot be empty. Please try again.")
			continue
		}

		// Split by comma and validate each URL
		rpcUrls := strings.Split(rpcUrlsInput, ",")
		validUrls := []string{}
		for i, url := range rpcUrls {
			url = strings.TrimSpace(url)
			if url == "" {
				fmt.Printf("Empty URL found at position %d. Please try again.\n", i+1)
				continue
			}

			// Validate each RPC URL
			if !utils.IsValidL1RPC(url) {
				fmt.Printf("Invalid RPC URL at position %d: %s. Please try again.\n", i+1, url)
				continue
			}
			validUrls = append(validUrls, url)
		}

		if len(validUrls) == 0 {
			fmt.Println("No valid RPC URLs found. Please try again.")
			continue
		}

		// Reconstruct the comma-separated string with validated URLs
		rpcUrlsInput = strings.Join(validUrls, ",")
		break
	}

	// Auto-detect Chain ID from first RPC URL
	firstRpcUrl := strings.Split(rpcUrlsInput, ",")[0]
	firstRpcUrl = strings.TrimSpace(firstRpcUrl)

	fmt.Println("Detecting Chain ID from RPC...")
	detectedChainID, err := utils.GetChainIDFromL1RPC(firstRpcUrl)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get chain ID from RPC: %w", err)
	}

	fmt.Printf("Detected Chain ID: %d\n", detectedChainID)
	chainID = detectedChainID

	if strings.Contains(rpcUrlsInput, ",") {
		fmt.Printf("✅ Network configured with %d RPC URLs (Chain ID: %d)\n", len(strings.Split(rpcUrlsInput, ",")), chainID)
	} else {
		fmt.Printf("✅ Network configured: %s (Chain ID: %d)\n", rpcUrlsInput, chainID)
	}
	return rpcUrlsInput, chainID, nil
}

func (t *ThanosStack) DeployDRB(ctx context.Context, inputs *types.DeployDRBInput) (*types.DeployDRBOutput, error) {
	deployDRBContractsOutput, err := t.deployDRBContracts(ctx, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy drb contracts: %s", err)
	}

	deployDRBApplicationOutput, err := t.deployDRBApplication(ctx, inputs, deployDRBContractsOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy drb application: %s", err)
	}

	return &types.DeployDRBOutput{
		DeployDRBContractsOutput:   deployDRBContractsOutput,
		DeployDRBApplicationOutput: deployDRBApplicationOutput,
	}, nil
}

func (t *ThanosStack) UninstallDRB(ctx context.Context) error {
	namespace := constants.DRBNamespace
	t.logger.Info("Starting DRB uninstallation...")

	// Check if DRB namespace exists
	namespaceExists, err := utils.CheckNamespaceExists(ctx, namespace)
	if err != nil {
		t.logger.Warnw("Failed to check DRB namespace existence, will still attempt cleanup", "err", err)
		namespaceExists = false
	}

	// only clean up k8s resources if namespace exists
	if namespaceExists {
		t.logger.Info("DRB namespace exists, cleaning up Kubernetes resources...")

		releases, err := utils.FilterHelmReleases(ctx, namespace, "drb-node")
		if err != nil {
			t.logger.Warnw("Error filtering helm releases, continuing with cleanup", "err", err)
		} else {
			for _, release := range releases {
				t.logger.Infow("Uninstalling Helm release", "release", release, "namespace", namespace)
				_, err = utils.ExecuteCommand(ctx, "helm", []string{
					"uninstall",
					release,
					"--namespace",
					namespace,
				}...)
				if err != nil {
					t.logger.Warnw("Error uninstalling DRB helm chart, continuing", "err", err)
				}
			}
		}

		// Delete Kubernetes Secret
		secretName := "drb-leader-static-key"
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "delete", "secret", secretName, "-n", namespace, "--ignore-not-found=true")

		t.logger.Info(fmt.Sprintf("Deleting DRB namespace: %s", namespace))
		err = t.tryToDeleteK8sNamespace(ctx, namespace)
		if err != nil {
			t.logger.Warnw("Failed to delete DRB namespace, continuing with terraform cleanup", "err", err, "namespace", namespace)
		}

		// Clean up storage that might be left behind
		if err := t.cleanupExistingDRBStorage(ctx, namespace); err != nil {
			t.logger.Warnw("Failed to cleanup DRB storage", "err", err)
		}
	} else {
		t.logger.Info("DRB namespace does not exist, skipping Kubernetes cleanup")
	}

	t.logger.Info("Destroying DRB RDS terraform resources (if any)...")
	err = t.destroyTerraform(ctx, fmt.Sprintf("%s/tokamak-thanos-stack/terraform/drb", t.deploymentPath))
	if err != nil {
		t.logger.Warnf("Failed to destroy DRB RDS terraform resources: %v. Continuing with infrastructure cleanup.", err)
	}

	// Destroy DRB infrastructure (EKS cluster and VPC)
	// This destroys the thanos-stack terraform resources for drb-ecosystem cluster
	t.logger.Info("Destroying DRB infrastructure (EKS cluster and VPC)...")
	thanosStackPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)

	// Check if this is DRB infrastructure by verifying the namespace in .envrc
	envrcPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/.envrc", t.deploymentPath)
	if utils.CheckFileExists(envrcPath) {
		// Read .envrc to check if namespace is drb-ecosystem
		envrcContent, err := os.ReadFile(envrcPath)
		if err == nil {
			envrcStr := string(envrcContent)
			// Only destroy if this is DRB infrastructure (namespace contains "drb-ecosystem")
			if strings.Contains(envrcStr, "drb-ecosystem") {
				// Destroy with -lock=false to bypass state locks
				err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
					"-c",
					fmt.Sprintf(`cd %s && source ../.envrc && terraform destroy -auto-approve -parallelism=1 -lock=false`, thanosStackPath),
				}...)
				if err != nil {
					t.logger.Warnf("Failed to destroy DRB infrastructure (EKS/VPC): %v. You may need to destroy manually.", err)
				} else {
					t.logger.Info("✅ DRB infrastructure (EKS cluster and VPC) destroyed successfully")
				}
			} else {
				t.logger.Info("Skipping infrastructure destroy - this appears to be main chain infrastructure, not DRB")
			}
		}
	}

	t.logger.Info("✅ Uninstall of DRB successfully!")
	return nil
}

func (t *ThanosStack) deployDRBContracts(ctx context.Context, inputs *types.DeployDRBInput) (*types.DeployDRBContractsOutput, error) {
	// Clone the drb repository
	err := t.cloneSourcecode(ctx, "Commit-Reveal2", "https://github.com/tokamak-network/Commit-Reveal2.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone drb repository: %s", err)
	}

	// use full path to avoid issues with working directory
	commitReveal2Path := filepath.Join(t.deploymentPath, "Commit-Reveal2")

	// Checkout to `service` branch
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && git checkout service", commitReveal2Path))
	if err != nil {
		return nil, fmt.Errorf("failed to checkout service: %s", err)
	}

	t.logger.Info("Clearing Forge cache to ensure fresh build...")
	// Clear Forge cache to ensure fresh build
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && forge clean", commitReveal2Path))
	if err != nil {
		t.logger.Warnf("Failed to clear Forge cache (non-critical): %v. Continuing with build.", err)
		// Continue even if cache clear fails - not critical
	}

	t.logger.Info("Start to build drb contracts")

	// Build the contracts
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && make install && make build", commitReveal2Path))
	if err != nil {
		return nil, fmt.Errorf("failed to build the contracts: %s", err)
	}

	// Use leader node private key for contract deployment (leader should be contract owner)
	privateKey := strings.TrimPrefix(inputs.PrivateKey, "0x")
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid leader private key: %w", err)
	}

	leaderAddress := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

	// Get first RPC URL for contract deployment (forge scripts need single URL)
	firstRpcUrl := strings.Split(inputs.RPC, ",")[0]
	firstRpcUrl = strings.TrimSpace(firstRpcUrl)

	// Create .env file
	envContent := fmt.Sprintf(`
	# Leader Node Configuration (Contract Owner)
	PRIVATE_KEY=%s
	DEPLOYER=%s
	RPC_URL=%s`, privateKey, leaderAddress.Hex(), firstRpcUrl)

	envFilePath := filepath.Join(t.deploymentPath, "Commit-Reveal2", ".env")
	err = os.WriteFile(envFilePath, []byte(envContent), 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create .env file: %w", err)
	}

	t.logger.Info("Deploying DRB contracts")

	// Run forge script with direct exec (no shell) to avoid injection from user-controlled RPC URL/private key
	err = utils.ExecuteCommandStreamInDir(ctx, t.logger, commitReveal2Path, "forge",
		"script", "script/DeployCommitReveal2.s.sol:DeployCommitReveal2",
		"--rpc-url", firstRpcUrl, "--private-key", privateKey, "--broadcast", "-vv")
	if err != nil {
		return nil, fmt.Errorf("failed to deploy the contracts. Please check: 1) Leader account has sufficient balance, 2) Gas price is reasonable (use 'cast gas-price --rpc-url %s'), 3) RPC endpoint is accessible, 4) Chain ID (%d) matches the network. Error: %w", firstRpcUrl, inputs.ChainID, err)
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

	contractAddress, exists := contractAddresses[contractName]
	if !exists || contractAddress == "" {
		return nil, fmt.Errorf("contract %s not found in deployment output", contractName)
	}

	t.logger.Infof("✅ DRB contract deployed: %s at address %s", contractName, contractAddress)

	// Deploy ConsumerExampleV2 after CommitReveal2L2 is deployed
	t.logger.Info("Deploying ConsumerExampleV2 contract")
	consumerAddress, err := t.deployConsumerExampleV2(ctx, inputs)
	if err != nil {
		t.logger.Warnf("Failed to deploy ConsumerExampleV2: %v. Continuing without consumer contract.", err)
	} else {
		t.logger.Infof("ConsumerExampleV2 deployed at address: %s", consumerAddress)
	}

	return &types.DeployDRBContractsOutput{
		ContractAddress:          contractAddress,
		ContractName:             contractName,
		ChainID:                  inputs.ChainID,
		ConsumerExampleV2Address: consumerAddress,
	}, nil
}

// deployConsumerExampleV2 deploys the ConsumerExampleV2 contract
func (t *ThanosStack) deployConsumerExampleV2(ctx context.Context, inputs *types.DeployDRBInput) (string, error) {
	// Get private key from inputs (already validated in GetDRBInput)
	privateKey := strings.TrimPrefix(inputs.PrivateKey, "0x")

	t.logger.Info("Deploying ConsumerExampleV2 contract...")

	// Get first RPC URL for contract deployment
	firstRpcUrl := strings.Split(inputs.RPC, ",")[0]
	firstRpcUrl = strings.TrimSpace(firstRpcUrl)

	// use absolute path to avoid issues with working directory
	commitReveal2Path := filepath.Join(t.deploymentPath, "Commit-Reveal2")

	// Run forge script with direct exec (no shell) to avoid injection from user-controlled RPC URL/private key
	err := utils.ExecuteCommandStreamInDir(ctx, t.logger, commitReveal2Path, "forge",
		"script", "script/DeployConsumerExampleV2.s.sol:DeployConsumerExampleV2",
		"--sig", "run()", "--rpc-url", firstRpcUrl, "--private-key", privateKey, "--broadcast", "-vv")
	if err != nil {
		return "", fmt.Errorf("failed to deploy ConsumerExampleV2: %w", err)
	}

	// Extract ConsumerExampleV2 address from the deployment output
	consumerAddresses, err := t.getDRBContractAddressFromOutput(ctx, "DeployConsumerExampleV2.s.sol", inputs.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get ConsumerExampleV2 address from deployment output: %w", err)
	}

	consumerAddress, ok := consumerAddresses["ConsumerExampleV2"]
	if !ok || consumerAddress == "" {
		return "", fmt.Errorf("ConsumerExampleV2 address not found in deployment output")
	}

	t.logger.Infof("✅ ConsumerExampleV2 deployed at address: %s", consumerAddress)
	return consumerAddress, nil
}

// deployDRBInfrastructure deploys a minimal EKS cluster for DRB with separate VPC
func (t *ThanosStack) deployDRBInfrastructure(ctx context.Context) (*types.DRBInfrastructureConfig, error) {
	var err error
	shellConfigFile := utils.GetShellConfigDefault()

	// Check dependencies
	if !dependencies.CheckTerraformInstallation(ctx) {
		t.logger.Warn("Try running `source %s` to set up your environment", shellConfigFile)
		return nil, fmt.Errorf("terraform is not installed")
	}

	if !dependencies.CheckAwsCLIInstallation(ctx) {
		t.logger.Warn("Try running `source %s` to set up your environment", shellConfigFile)
		return nil, fmt.Errorf("aws cli is not installed")
	}

	if t.awsProfile == nil {
		return nil, fmt.Errorf("AWS configuration is not set")
	}

	awsLoginInputs := t.awsProfile.AwsConfig
	awsAccountProfile := t.awsProfile.AccountProfile

	// DRB cluster configuration - use timestamp to make each deployment unique
	timestamp := time.Now().Unix()
	clusterName := fmt.Sprintf("drb-ecosystem-%d", timestamp)
	namespace := constants.DRBNamespace

	// Clean up any existing DRB resources before deployment
	t.logger.Info("⚡️ Cleaning up existing DRB resources...")
	if err = t.cleanupExistingDRBResources(ctx, namespace, clusterName, awsLoginInputs.Region); err != nil {
		t.logger.Warnf("Failed to cleanup existing DRB resources: %v. Continuing with deployment.", err)
	}
	t.logger.Info("✅ Cleaned up existing DRB resources")

	// Clone the charts repository if not already cloned
	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone tokamak-thanos-stack repository: %w", err)
	}

	// use absolute path for tokamak-thanos-stack
	thanosStackPath := filepath.Join(t.deploymentPath, "tokamak-thanos-stack")

	// Checkout to `feat/add-drb-node` branch for alpha release and pull latest changes
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && git checkout feat/add-drb-node && git pull origin feat/add-drb-node", thanosStackPath))
	if err != nil {
		return nil, fmt.Errorf("failed to checkout and pull feat/add-drb-node: %w", err)
	}

	t.logger.Info("Deploying DRB infrastructure (EKS cluster with separate VPC)...")

	// Create dummy deployment file with all required contract fields for Terraform script
	// The generate-thanos-stack-values.sh script reads multiple contract addresses from this file
	dummyDeploymentPath := fmt.Sprintf("%s/drb-dummy-deployment.json", t.deploymentPath)
	dummyDeploymentContent := []byte(`{
		"AddressManager": "0x0000000000000000000000000000000000000000",
		"AnchorStateRegistry": "0x0000000000000000000000000000000000000000",
		"AnchorStateRegistryProxy": "0x0000000000000000000000000000000000000000",
		"DelayedWETH": "0x0000000000000000000000000000000000000000",
		"DelayedWETHProxy": "0x0000000000000000000000000000000000000000",
		"DisputeGameFactory": "0x0000000000000000000000000000000000000000",
		"DisputeGameFactoryProxy": "0x0000000000000000000000000000000000000000",
		"L1CrossDomainMessenger": "0x0000000000000000000000000000000000000000",
		"L1CrossDomainMessengerProxy": "0x0000000000000000000000000000000000000000",
		"L1ERC721Bridge": "0x0000000000000000000000000000000000000000",
		"L1ERC721BridgeProxy": "0x0000000000000000000000000000000000000000",
		"L1StandardBridge": "0x0000000000000000000000000000000000000000",
		"L1StandardBridgeProxy": "0x0000000000000000000000000000000000000000",
		"L1UsdcBridge": "0x0000000000000000000000000000000000000000",
		"L1UsdcBridgeProxy": "0x0000000000000000000000000000000000000000",
		"L2OutputOracle": "0x0000000000000000000000000000000000000000",
		"L2OutputOracleProxy": "0x0000000000000000000000000000000000000000",
		"Mips": "0x0000000000000000000000000000000000000000",
		"OptimismMintableERC20Factory": "0x0000000000000000000000000000000000000000",
		"OptimismMintableERC20FactoryProxy": "0x0000000000000000000000000000000000000000",
		"OptimismPortal": "0x0000000000000000000000000000000000000000",
		"OptimismPortal2": "0x0000000000000000000000000000000000000000",
		"OptimismPortalProxy": "0x0000000000000000000000000000000000000000",
		"PermissionedDelayedWETHProxy": "0x0000000000000000000000000000000000000000",
		"PreimageOracle": "0x0000000000000000000000000000000000000000",
		"ProtocolVersions": "0x0000000000000000000000000000000000000000",
		"ProtocolVersionsProxy": "0x0000000000000000000000000000000000000000",
		"ProxyAdmin": "0x0000000000000000000000000000000000000000",
		"SafeProxyFactory": "0x0000000000000000000000000000000000000000",
		"SafeSingleton": "0x0000000000000000000000000000000000000000",
		"SuperchainConfig": "0x0000000000000000000000000000000000000000",
		"SuperchainConfigProxy": "0x0000000000000000000000000000000000000000",
		"SystemConfig": "0x0000000000000000000000000000000000000000",
		"SystemConfigProxy": "0x0000000000000000000000000000000000000000",
		"SystemOwnerSafe": "0x0000000000000000000000000000000000000000"
	}`)
	if err := os.WriteFile(dummyDeploymentPath, dummyDeploymentContent, 0644); err != nil {
		t.logger.Warnf("Failed to create dummy deployment file: %v", err)
	}

	err = makeTerraformEnvFile(fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath), types.TerraformEnvConfig{
		Namespace:           clusterName,
		AwsRegion:           awsLoginInputs.Region,
		SequencerKey:        "",
		BatcherKey:          "",
		ProposerKey:         "",
		ChallengerKey:       "",
		EksClusterAdmins:    awsAccountProfile.Arn,
		Azs:                 awsAccountProfile.AvailabilityZones,
		DeploymentFilePath:  dummyDeploymentPath,
		L1RpcUrl:            "dummy.com",
		L1RpcProvider:       "dummy.com",
		L1BeaconUrl:         "dummy.com",
		OpGethImageTag:      "dummy",
		ThanosStackImageTag: "dummy",
		MaxChannelDuration:  0,
		TxmgrCellProofTime:  0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform environment file: %w", err)
	}

	// Terraform expects these files even though DRB doesn't use them
	configFilesDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files", t.deploymentPath)
	if err = os.MkdirAll(configFilesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config-files directory: %w", err)
	}

	// Create minimal dummy genesis.json (empty object - DRB doesn't need chain config)
	dummyGenesis := []byte(`{}`)
	if err = os.WriteFile(fmt.Sprintf("%s/genesis.json", configFilesDir), dummyGenesis, 0644); err != nil {
		return nil, fmt.Errorf("failed to create dummy genesis.json: %w", err)
	}

	// Create minimal dummy rollup.json (empty object - DRB doesn't need chain config)
	dummyRollup := []byte(`{}`)
	if err = os.WriteFile(fmt.Sprintf("%s/rollup.json", configFilesDir), dummyRollup, 0644); err != nil {
		return nil, fmt.Errorf("failed to create dummy rollup.json: %w", err)
	}

	// Create minimal dummy prestate.json (empty object - DRB doesn't need chain config)
	dummyPrestate := []byte(`{}`)
	if err = os.WriteFile(fmt.Sprintf("%s/prestate.json", configFilesDir), dummyPrestate, 0644); err != nil {
		return nil, fmt.Errorf("failed to create dummy prestate.json: %w", err)
	}

	// Initialize Terraform backend
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd backend &&
		terraform init -reconfigure &&
		terraform plan &&
		terraform apply -auto-approve
		`, t.deploymentPath),
	}...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terraform backend: %w", err)
	}

	// Get backend bucket name from terraform output and update .envrc before initializing thanos-stack
	backendBucketOutput, err := utils.ExecuteCommand(ctx, "bash", "-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform/backend && source ../.envrc && terraform output -raw backend_bucket_name 2>/dev/null`, t.deploymentPath))
	if err != nil {
		return nil, fmt.Errorf("failed to get backend bucket name: %w", err)
	}
	backendBucketName := strings.TrimSpace(backendBucketOutput)
	if backendBucketName == "" {
		return nil, fmt.Errorf("backend bucket name is empty")
	}

	// Update .envrc with the actual backend bucket name
	envrcPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/.envrc", t.deploymentPath)
	envrcContent, err := os.ReadFile(envrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .envrc file: %w", err)
	}

	// Replace empty backend bucket name with actual value
	envrcStr := string(envrcContent)
	envrcStr = strings.ReplaceAll(envrcStr, `export TF_VAR_backend_bucket_name=""`, fmt.Sprintf(`export TF_VAR_backend_bucket_name="%s"`, backendBucketName))

	// Update TF_CLI_ARGS_init to use the actual bucket name directly
	envrcStr = strings.ReplaceAll(envrcStr,
		`-backend-config='bucket=$TF_VAR_backend_bucket_name'`,
		fmt.Sprintf(`-backend-config='bucket=%s'`, backendBucketName))

	if err = os.WriteFile(envrcPath, []byte(envrcStr), 0644); err != nil {
		return nil, fmt.Errorf("failed to update .envrc file: %w", err)
	}

	t.logger.Infof("Updated .envrc with backend bucket name: %s", backendBucketName)

	// Deploy infrastructure (EKS + VPC)
	t.logger.Info("Deploying EKS infrastructure for DRB...")
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform init -reconfigure &&
		terraform plan &&
		terraform apply -auto-approve`, t.deploymentPath),
	}...)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy DRB infrastructure: %w", err)
	}

	// Get VPC ID from Terraform output
	vpcIdOutput, err := utils.ExecuteCommand(ctx, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform output -json vpc_id`, t.deploymentPath),
	}...)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPC ID from terraform output: %w", err)
	}

	vpcID := strings.Trim(vpcIdOutput, `"`)

	// Configure EKS access
	err = utils.SwitchKubernetesContext(ctx, clusterName, awsLoginInputs.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to configure EKS access: %w", err)
	}

	return &types.DRBInfrastructureConfig{
		ClusterName: clusterName,
		Namespace:   namespace,
		VpcID:       vpcID,
		Region:      awsLoginInputs.Region,
	}, nil
}

func (t *ThanosStack) deployDRBApplication(ctx context.Context, inputs *types.DeployDRBInput, contracts *types.DeployDRBContractsOutput) (*types.DeployDRBApplicationOutput, error) {
	t.logger.Info("Checking DRB infrastructure...")
	drbInfra, err := t.deployDRBInfrastructure(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy DRB infrastructure: %w", err)
	}

	// Store VPC ID for database deployment
	if t.deployConfig == nil {
		t.deployConfig = &types.Config{}
	}
	if t.deployConfig.AWS == nil {
		t.deployConfig.AWS = &types.AWSConfig{}
	}
	t.deployConfig.AWS.VpcID = drbInfra.VpcID
	if t.deployConfig.AWS.Region == "" {
		t.deployConfig.AWS.Region = drbInfra.Region
	}
	if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		t.logger.Warnf("Failed to save VPC ID to config: %v", err)
	}

	// Use database configuration from inputs (collected in GetDRBInput)
	if inputs.DatabaseConfig == nil {
		return nil, fmt.Errorf("database configuration is not set in inputs")
	}

	// Clone the tokamak-thanos-stack repository
	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone tokamak-thanos-stack repository: %s", err)
	}

	// use absolute path for tokamak-thanos-stack
	thanosStackPath := filepath.Join(t.deploymentPath, "tokamak-thanos-stack")

	// Checkout to `feat/add-drb-node` and pull latest changes
	// err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd tokamak-thanos-stack && git checkout feat/add-drb-node && git pull origin feat/add-drb-node")
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && git checkout feat/add-drb-node && git pull origin feat/add-drb-node", thanosStackPath))
	if err != nil {
		return nil, fmt.Errorf("failed to checkout and pull feat/add-drb-node: %s", err)
	}

	// Generate leader node ID
	t.logger.Info("Generating leader node ID...")
	leaderPeerID, err := t.generateLeaderNodeID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate leader node ID: %w", err)
	}
	t.logger.Info("✅ Leader node peer ID generated successfully")

	// Deploy DRB leader node
	appOutput, err := t.deployDRBNodes(ctx, inputs, contracts, inputs.DatabaseConfig, leaderPeerID)
	if err != nil {
		return nil, err
	}

	// Save leader info to JSON file
	if err := t.saveDRBLeaderInfo(ctx, inputs, contracts, appOutput, leaderPeerID, drbInfra); err != nil {
		t.logger.Warnf("Failed to save leader info: %v", err)
		// Don't fail deployment if saving info fails
	}

	return appOutput, nil
}

// saveDRBLeaderInfo saves leader connection information to drb-leader-info.json
func (t *ThanosStack) saveDRBLeaderInfo(ctx context.Context, inputs *types.DeployDRBInput, contracts *types.DeployDRBContractsOutput, appOutput *types.DeployDRBApplicationOutput, leaderPeerID string, infra *types.DRBInfrastructureConfig) error {
	// Extract leader EOA from private key
	leaderPrivateKey := strings.TrimPrefix(inputs.LeaderNodeInput.PrivateKey, "0x")
	leaderPrivateKeyECDSA, err := crypto.HexToECDSA(leaderPrivateKey)
	if err != nil {
		return fmt.Errorf("invalid leader private key: %w", err)
	}
	leaderEOA := crypto.PubkeyToAddress(leaderPrivateKeyECDSA.PublicKey).Hex()

	// Extract IP and port from leader URL
	leaderURL := appOutput.LeaderNodeURL
	leaderIP := ""
	leaderPort := 61280 // Default port
	if strings.HasPrefix(leaderURL, "http://") {
		urlWithoutScheme := strings.TrimPrefix(leaderURL, "http://")
		parts := strings.Split(urlWithoutScheme, ":")
		if len(parts) >= 1 {
			leaderIP = parts[0]
		}
		if len(parts) >= 2 {
			if port, err := strconv.Atoi(parts[1]); err == nil {
				leaderPort = port
			}
		}
	} else {
		leaderIP = leaderURL
	}

	leaderInfo := &types.DRBLeaderInfo{
		LeaderURL:                leaderURL,
		LeaderIP:                 leaderIP,
		LeaderPort:               leaderPort,
		LeaderPeerID:             leaderPeerID,
		LeaderEOA:                leaderEOA,
		CommitReveal2L2Address:   contracts.ContractAddress,
		ConsumerExampleV2Address: contracts.ConsumerExampleV2Address,
		ChainID:                  contracts.ChainID,
		RPCURL:                   inputs.RPC,
		DeploymentTimestamp:      time.Now().UTC().Format(time.RFC3339),
		ClusterName:              infra.ClusterName,
		Namespace:                infra.Namespace,
	}

	// Save to JSON file
	infoFilePath := fmt.Sprintf("%s/drb-leader-info.json", t.deploymentPath)
	infoJSON, err := json.MarshalIndent(leaderInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal leader info: %w", err)
	}

	if err := os.WriteFile(infoFilePath, infoJSON, 0644); err != nil {
		return fmt.Errorf("failed to write leader info file: %w", err)
	}

	t.logger.Infof("✅ Leader information saved to: %s", infoFilePath)
	fmt.Printf("\n✅ Leader information saved to: %s\n", infoFilePath)
	fmt.Println("\n--------------------------------")
	fmt.Println("DRB Leader Node Information")
	fmt.Println("--------------------------------")
	fmt.Printf("Leader URL:              %s\n", leaderInfo.LeaderURL)
	fmt.Printf("Leader IP:               %s\n", leaderInfo.LeaderIP)
	fmt.Printf("Leader Port:             %d\n", leaderInfo.LeaderPort)
	fmt.Printf("Leader Peer ID:          %s\n", leaderInfo.LeaderPeerID)
	fmt.Printf("Leader EOA:              %s\n", leaderInfo.LeaderEOA)
	fmt.Printf("CommitReveal2L2 Address: %s\n", leaderInfo.CommitReveal2L2Address)
	if leaderInfo.ConsumerExampleV2Address != "" {
		fmt.Printf("ConsumerExampleV2 Address: %s\n", leaderInfo.ConsumerExampleV2Address)
	}
	fmt.Printf("Chain ID:                %d\n", leaderInfo.ChainID)
	fmt.Printf("RPC URL:                 %s\n", leaderInfo.RPCURL)
	fmt.Println("--------------------------------")
	fmt.Println("Use 'trh-sdk drb leader-info' to view this information later")
	fmt.Println("--------------------------------")

	return nil
}

// deployDRBDatabaseRDS deploys AWS RDS PostgreSQL for DRB node
func (t *ThanosStack) deployDRBDatabaseRDS(ctx context.Context, dbConfig *types.DRBDatabaseConfig) (string, error) {
	if t.deployConfig == nil || t.deployConfig.AWS == nil {
		return "", fmt.Errorf("AWS configuration is not set in deploy config")
	}

	vpcId := t.deployConfig.AWS.VpcID
	awsRegion := t.deployConfig.AWS.Region

	if vpcId == "" {
		return "", fmt.Errorf("VPC ID is not set in deploy config")
	}
	if awsRegion == "" {
		return "", fmt.Errorf("AWS region is not set in deploy config")
	}

	t.logger.Infof("Deploying AWS RDS PostgreSQL for DRB node...")

	envrcPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath)

	// use timestamp for unique rds naming (avoids conflicts when deploying multiple regular nodes)
	stackName := fmt.Sprintf("drb-%d", time.Now().Unix())

	err := t.makeDRBEnvs(
		envrcPath,
		".envrc",
		types.AwsDatabaseEnvs{
			DatabaseUserName: dbConfig.Username,
			DatabasePassword: dbConfig.Password,
			DatabaseName:     dbConfig.DatabaseName,
			VpcId:            vpcId,
			AwsRegion:        awsRegion,
			StackName:        stackName,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create .envrc file: %w", err)
	}

	// Deploy RDS
	t.logger.Info("Deploying AWS RDS PostgreSQL for DRB node...")
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd drb &&
		terraform init -reconfigure &&
		terraform plan &&
		terraform apply -auto-approve
		`, t.deploymentPath),
	}...)
	if err != nil {
		return "", fmt.Errorf("failed to deploy RDS: %w", err)
	}

	// Get connection URL after deployment
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
	t.logger.Infof("✅ RDS PostgreSQL deployed successfully")
	return rdsConnectionUrl, nil
}

// deployDRBDatabaseLocal deploys local PostgreSQL using Helm chart ( coming soon ...)
func (t *ThanosStack) deployDRBDatabaseLocal(ctx context.Context, dbConfig *types.DRBDatabaseConfig) (string, error) {
	// namespace := t.deployConfig.K8s.Namespace

	// t.logger.Infof("Deploying local PostgreSQL for DRB %s node using Helm chart...", nodeType)

	// // Ensure namespace exists
	// if err := t.ensureNamespaceExists(ctx, namespace); err != nil {
	// 	return "", fmt.Errorf("failed to ensure namespace exists: %w", err)
	// }

	// // Check if PostgreSQL Helm chart path exists
	// chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/drb-postgres", t.deploymentPath)
	// valuesPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/drb-postgres/values.yaml", t.deploymentPath)

	// // Check if chart exists
	// if _, err := os.Stat(chartPath); err != nil {
	// 	return "", fmt.Errorf("PostgreSQL Helm chart not found at %s: %w", chartPath, err)
	// }

	// // Update values.yaml with database configuration
	// if err := utils.UpdateYAMLField(valuesPath, "auth.postgresPassword", dbConfig.Password); err != nil {
	// 	t.logger.Warn("Failed to update postgresPassword in values.yaml, continuing...", "err", err)
	// }

	// if err := utils.UpdateYAMLField(valuesPath, "auth.database", dbConfig.DatabaseName); err != nil {
	// 	t.logger.Warn("Failed to update database name in values.yaml, continuing...", "err", err)
	// }

	// // Install PostgreSQL using Helm with unique release name per node
	// helmReleaseName := fmt.Sprintf("drb-postgres-%s", nodeType)
	// args := []string{
	// 	"install",
	// 	helmReleaseName,
	// 	chartPath,
	// 	"--values", valuesPath,
	// 	"--namespace", namespace,
	// 	"--create-namespace",
	// }

	// Will implement later after alpha release

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

// generateLeaderNodeID clones DRB-node repository, runs the generator script, and extracts the leader peer ID
func (t *ThanosStack) generateLeaderNodeID(ctx context.Context) (string, error) {
	drbNodeRepoURL := "https://github.com/tokamak-network/DRB-node.git"
	branch := "dispute-mechanism"

	// Clone or update DRB-node repository
	t.logger.Info("Cloning DRB-node repository...")
	err := t.cloneSourcecode(ctx, "DRB-node", drbNodeRepoURL)
	if err != nil {
		return "", fmt.Errorf("failed to clone DRB-node repository: %w", err)
	}

	// use absolute path to avoid issues with working directory
	drbNodePath := filepath.Join(t.deploymentPath, "DRB-node")

	// Checkout to dispute-mechanism branch
	t.logger.Infof("Checking out branch: %s", branch)
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd %s && git checkout %s", drbNodePath, branch))
	if err != nil {
		return "", fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}

	// New path after DRB-node refactor: deployment/leader/generate-peer-id.sh, output at deployment/leader/static-key/leadernode.bin
	leaderNodeBinPath := fmt.Sprintf("%s/DRB-node/deployment/leader/static-key/leadernode.bin", t.deploymentPath)

	// Always delete existing file to ensure we get peer ID in output
	if _, err := os.Stat(leaderNodeBinPath); err == nil {
		t.logger.Info("Deleting existing leadernode.bin to generate new peer ID...")
		if err := os.Remove(leaderNodeBinPath); err != nil {
			return "", fmt.Errorf("failed to delete existing leadernode.bin: %w", err)
		}
	}

	// Run the generator script
	t.logger.Info("Running leader node generator script...")
	generatorScriptPath := fmt.Sprintf("%s/DRB-node/deployment/leader/generate-peer-id.sh", t.deploymentPath)

	// Make script executable
	_, err = utils.ExecuteCommand(ctx, "chmod", "+x", generatorScriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to make generator script executable: %w", err)
	}

	// Execute the generator script from deployment/leader
	leaderDeployPath := filepath.Join(drbNodePath, "deployment", "leader")
	output, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("cd %s && ./generate-peer-id.sh", leaderDeployPath))
	if err != nil {
		return "", fmt.Errorf("failed to run generator script: %w", err)
	}

	// Verify that leadernode.bin exists before creating secret
	if _, err := os.Stat(leaderNodeBinPath); os.IsNotExist(err) {
		return "", fmt.Errorf("leader node key file was not found at %s", leaderNodeBinPath)
	}

	// Create Kubernetes Secret from the file
	secretName := "drb-leader-static-key"
	namespace := constants.DRBNamespace
	t.logger.Info("Creating Kubernetes Secret from leadernode.bin...")

	// Ensure namespace exists before creating secret
	if err := t.ensureNamespaceExists(ctx, namespace); err != nil {
		return "", fmt.Errorf("failed to ensure namespace %s exists: %w", namespace, err)
	}

	// Delete existing secret if it exists
	utils.ExecuteCommand(ctx, "kubectl", "delete", "secret", secretName, "-n", namespace, "--ignore-not-found=true")

	// Small delay to ensure delete completes
	time.Sleep(500 * time.Millisecond)

	// Create secret using kubectl create secret generic --from-file
	_, err = utils.ExecuteCommand(ctx, "kubectl", "create", "secret", "generic", secretName,
		"--from-file=leadernode.bin="+leaderNodeBinPath, "-n", namespace)
	if err != nil {
		return "", fmt.Errorf("failed to create Kubernetes Secret: %w", err)
	}
	t.logger.Info("✅ Created Kubernetes Secret for leadernode.bin")

	// Extract peer ID from script output
	var peerID string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LEADER_PEER_ID=") {
			peerID = strings.TrimSpace(strings.TrimPrefix(line, "LEADER_PEER_ID="))
			if peerID != "" {
				return peerID, nil
			}
		}
	}
	if peerID == "" {
		return "", fmt.Errorf("could not extract peer ID from generator script output")
	}

	return peerID, nil
}

// deployDRBInfrastructureForNode creates PV and PVC for a specific DRB node
func (t *ThanosStack) deployDRBInfrastructureForNode(ctx context.Context, namespace string, helmReleaseName string, nodeType string, pvcName string) error {
	if err := t.ensureNamespaceExists(ctx, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	// Get chain name for EFS filesystem ID
	// For DRB, always use "drb-ecosystem" since there's no chain name
	chainName := "drb-ecosystem"

	// Get AWS region
	if t.deployConfig == nil || t.deployConfig.AWS == nil || t.deployConfig.AWS.Region == "" {
		return fmt.Errorf("AWS region is not set in deploy config")
	}
	awsRegion := t.deployConfig.AWS.Region

	// Get EFS filesystem ID
	efsFileSystemId, err := utils.GetEFSFileSystemId(ctx, chainName, awsRegion)
	if err != nil {
		return fmt.Errorf("error getting EFS filesystem ID: %w", err)
	}

	// Create component name for leader node
	componentName := fmt.Sprintf("drb-node-%s", nodeType)

	config := &types.DRBConfig{
		Namespace:           namespace,
		IsPersistenceEnable: true,
		EFSFileSystemId:     efsFileSystemId,
		ChainName:           chainName,
		HelmReleaseName:     strings.TrimSuffix(pvcName, "-pvc"),
	}

	// Get timestamp from existing PV
	timestamp, err := utils.GetTimestampFromExistingPV(ctx, chainName)
	if err != nil {
		// This is expected for first-time DRB deployment
		t.logger.Infof("No existing PV found for %s, generating new timestamp", chainName)
		timestamp = fmt.Sprintf("%d", time.Now().Unix())
	}

	// Create PV and PVC
	drbPV := utils.GenerateStaticPVManifest(componentName, config, "4Gi", timestamp)
	if err := utils.ApplyPVManifest(ctx, t.deploymentPath, componentName, drbPV, "DRBNode"); err != nil {
		return fmt.Errorf("failed to create DRB PV for %s: %w", componentName, err)
	}

	drbPVC := utils.GenerateStaticPVCManifest(componentName, config, "4Gi", timestamp)
	if err := utils.ApplyPVCManifest(ctx, t.deploymentPath, componentName, drbPVC, "DRBNode"); err != nil {
		return fmt.Errorf("failed to create DRB PVC for %s: %w", componentName, err)
	}

	// Add Helm labels and annotations so Helm can manage the PVC when existingClaim is set
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "label", "pvc", pvcName, "-n", namespace, "app.kubernetes.io/managed-by=Helm", "--overwrite"); err != nil {
		return fmt.Errorf("failed to add Helm label to PVC %s: %w", pvcName, err)
	}
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "annotate", "pvc", pvcName, "-n", namespace, fmt.Sprintf("meta.helm.sh/release-name=%s", helmReleaseName), "--overwrite"); err != nil {
		return fmt.Errorf("failed to add Helm release-name annotation to PVC %s: %w", pvcName, err)
	}
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "annotate", "pvc", pvcName, "-n", namespace, fmt.Sprintf("meta.helm.sh/release-namespace=%s", namespace), "--overwrite"); err != nil {
		return fmt.Errorf("failed to add Helm release-namespace annotation to PVC %s: %w", pvcName, err)
	}

	t.logger.Infof("✅ Created DRB PV and PVC for %s", componentName)

	return nil
}

// cleanupExistingDRBResources cleans up all existing DRB resources before deployment
func (t *ThanosStack) cleanupExistingDRBResources(ctx context.Context, namespace, clusterName, region string) error {
	// 1. Uninstall Helm releases in DRB namespace
	t.logger.Info("Uninstalling existing Helm releases...")
	releases, err := utils.FilterHelmReleases(ctx, namespace, "drb-node")
	if err != nil {
		t.logger.Warnf("Failed to get Helm releases: %v", err)
	} else {
		for _, release := range releases {
			t.logger.Infof("Uninstalling Helm release: %s", release)
			_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace, "--ignore-not-found=true")
			if err != nil {
				t.logger.Warnf("Failed to uninstall Helm release %s: %v", release, err)
			}
		}
	}

	// 2. Delete Kubernetes secrets
	t.logger.Info("Deleting Kubernetes secrets...")
	_, _ = utils.ExecuteCommand(ctx, "kubectl", "delete", "secret", "-n", namespace, "--all", "--ignore-not-found=true")

	// 3. Clean up storage (PVs/PVCs)
	if err := t.cleanupExistingDRBStorage(ctx, namespace); err != nil {
		t.logger.Warnf("Failed to cleanup DRB storage: %v", err)
	}

	// 4. Delete namespace
	t.logger.Info("Deleting DRB namespace...")
	exists, err := utils.CheckNamespaceExists(ctx, namespace)
	if err != nil {
		t.logger.Warnf("Failed to check namespace existence: %v", err)
	} else if exists {
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", namespace, "--ignore-not-found=true", "--timeout=60s")
		// Wait a bit for namespace deletion
		time.Sleep(5 * time.Second)
	}

	// 5. Destroy Terraform resources (drb module)
	t.logger.Info("Destroying DRB Terraform resources...")
	drbTerraformPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/drb", t.deploymentPath)
	if err := t.destroyTerraform(ctx, drbTerraformPath); err != nil {
		t.logger.Warnf("Failed to destroy DRB Terraform resources: %v", err)
	}

	// 6. Destroy Terraform resources (thanos-stack for DRB cluster)
	t.logger.Info("Destroying DRB infrastructure Terraform resources...")
	thanosStackPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	envrcPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/.envrc", t.deploymentPath)
	if utils.CheckFileExists(envrcPath) {
		envrcContent, err := os.ReadFile(envrcPath)
		if err == nil {
			envrcStr := string(envrcContent)
			// Only destroy if this is DRB infrastructure (namespace contains "drb-ecosystem")
			if strings.Contains(envrcStr, "drb-ecosystem") {
				if err := t.destroyTerraform(ctx, thanosStackPath); err != nil {
					t.logger.Warnf("Failed to destroy DRB infrastructure Terraform resources: %v", err)
				}
			}
		}
	}

	return nil
}

// cleanupExistingDRBStorage removes existing DRB PVs and PVCs
func (t *ThanosStack) cleanupExistingDRBStorage(ctx context.Context, namespace string) error {
	// Get existing DRB PVCs
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		return fmt.Errorf("failed to get existing PVCs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		pvcName := strings.TrimSpace(line)
		if pvcName == "" || !strings.Contains(pvcName, "drb-node") {
			continue
		}

		// Check if PVC is bound to a pod
		boundOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", pvcName, "-n", namespace, "-o", "jsonpath={.status.phase}")
		if err == nil && strings.TrimSpace(boundOutput) == "Bound" {
			// Check if any pod is using this PVC
			podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", namespace, "-o", "jsonpath={.items[*].spec.volumes[*].persistentVolumeClaim.claimName}")
			if err == nil && strings.Contains(podOutput, pvcName) {
				continue
			}
		}

		// Delete PVC
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "delete", "pvc", pvcName, "-n", namespace, "--ignore-not-found=true")
	}

	// Get existing DRB PVs
	output, err = utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
	if err != nil {
		return fmt.Errorf("failed to get existing PVs: %w", err)
	}

	lines = strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pvName := parts[0]
		status := parts[1]

		// Only delete Released PVs (not Bound or Available)
		if status == "Released" && strings.Contains(pvName, "drb-node") {
			// Remove claimRef to allow reuse
			_, _ = utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", pvName, "-p", `{"spec":{"claimRef":null}}`, "--type=merge")
		}
	}

	return nil
}

// deployDRBNodes deploys DRB leader node only
func (t *ThanosStack) deployDRBNodes(ctx context.Context, inputs *types.DeployDRBInput, contracts *types.DeployDRBContractsOutput, dbConfig *types.DRBDatabaseConfig, leaderPeerID string) (*types.DeployDRBApplicationOutput, error) {
	// Deploy leader database only
	t.logger.Info("Deploying leader database...")
	leaderDBURL, err := t.deploySingleDatabase(ctx, dbConfig.Type, dbConfig.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy leader database: %w", err)
	}
	t.logger.Info("✅ Leader database deployed successfully")

	namespace := constants.DRBNamespace
	helmReleaseName := fmt.Sprintf("drb-node-%d", time.Now().Unix())

	t.logger.Info("Deploying DRB infrastructure (PV and PVC) for leader node...")

	// Create PVC for leader node
	leaderPVCName := fmt.Sprintf("%s-leader-pvc", helmReleaseName)
	if err := t.deployDRBInfrastructureForNode(ctx, namespace, helmReleaseName, "leader", leaderPVCName); err != nil {
		return nil, fmt.Errorf("failed to deploy DRB infrastructure for leader node: %w", err)
	}
	t.logger.Infof("⏳ Waiting for leader PVC to be bound: %s", leaderPVCName)
	if err := t.waitForPVCBound(ctx, namespace, leaderPVCName); err != nil {
		return nil, fmt.Errorf("failed to wait for leader PVC to be bound: %w", err)
	}
	t.logger.Info("✅ Leader PVC is bound and ready")

	// Deploy leader node
	return t.deployDRBLeaderNodeWithHelm(ctx, inputs, contracts, leaderDBURL, leaderPeerID, dbConfig.Password, helmReleaseName, leaderPVCName)
}

// deploySingleDatabase deploys a single database instance for the leader node
func (t *ThanosStack) deploySingleDatabase(ctx context.Context, dbType string, password string) (string, error) {
	dbConfig := &types.DRBDatabaseConfig{
		Type:         dbType,
		Username:     "postgres",
		Password:     password,
		DatabaseName: "drb",
	}

	switch dbType {
	case "rds":
		return t.deployDRBDatabaseRDS(ctx, dbConfig)
	case "local":
		return t.deployDRBDatabaseLocal(ctx, dbConfig)
	default:
		return "", fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// deployDRBLeaderNodeWithHelm updates Helm values and deploys only the leader node
func (t *ThanosStack) deployDRBLeaderNodeWithHelm(ctx context.Context, inputs *types.DeployDRBInput, contracts *types.DeployDRBContractsOutput, leaderDBURL string, leaderPeerID string, leaderDBPassword string, helmReleaseName string, leaderPVCName string) (*types.DeployDRBApplicationOutput, error) {
	namespace := constants.DRBNamespace

	// Check if values.yaml exists
	valuesFilePath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/drb-node/values.yaml", t.deploymentPath)
	if _, err := os.Stat(valuesFilePath); err != nil {
		return nil, fmt.Errorf("DRB node values.yaml not found at %s: %w", valuesFilePath, err)
	}

	// Update leader configuration
	t.logger.Info("Updating DRB leader node Helm values...")

	// Extract leader EOA from private key
	leaderPrivateKey := strings.TrimPrefix(inputs.LeaderNodeInput.PrivateKey, "0x")
	leaderPrivateKeyECDSA, err := crypto.HexToECDSA(leaderPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid leader private key: %w", err)
	}
	leaderEOA := crypto.PubkeyToAddress(leaderPrivateKeyECDSA.PublicKey).Hex()

	// Helper function to extract hostname from PostgreSQL connection URL
	extractHostFromURL := func(dbURL string) string {
		if strings.Contains(dbURL, "@") {
			parts := strings.Split(dbURL, "@")
			if len(parts) == 2 {
				hostPort := strings.Split(parts[1], "/")[0]
				return strings.Split(hostPort, ":")[0]
			}
		}
		return dbURL
	}

	leaderPostgresHost := extractHostFromURL(leaderDBURL)

	// Update leader node environment variables
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.POSTGRES_HOST", leaderPostgresHost); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.POSTGRES_HOST: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.POSTGRES_PASSWORD", leaderDBPassword); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.POSTGRES_PASSWORD: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.CHAIN_ID", fmt.Sprintf("%d", inputs.ChainID)); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.CHAIN_ID: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.ETH_RPC_URLS", inputs.RPC); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.ETH_RPC_URLS: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.CONTRACT_ADDRESS", contracts.ContractAddress); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.CONTRACT_ADDRESS: %w", err)
	}
	// LEADER_PRIVATE_KEY and EOA_PRIVATE_KEY should be without "0x" prefix for DRB application
	leaderPrivateKeyValue := strings.TrimPrefix(inputs.LeaderNodeInput.PrivateKey, "0x")
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.LEADER_PRIVATE_KEY", leaderPrivateKeyValue); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.LEADER_PRIVATE_KEY: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.env.LEADER_EOA", leaderEOA); err != nil {
		return nil, fmt.Errorf("failed to update leader.env.LEADER_EOA: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.EOA_PRIVATE_KEY", leaderPrivateKeyValue); err != nil {
		return nil, fmt.Errorf("failed to update leader.EOA_PRIVATE_KEY: %w", err)
	}
	if err := utils.UpdateYAMLField(valuesFilePath, "leader.persistence.existingClaim", leaderPVCName); err != nil {
		return nil, fmt.Errorf("failed to update leader.persistence.existingClaim: %w", err)
	}

	// Deploy using Helm
	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/drb-node", t.deploymentPath)
	args := []string{
		"install",
		helmReleaseName,
		chartPath,
		"--values", valuesFilePath,
		"--namespace", namespace,
		"--create-namespace",
	}

	output, err := utils.ExecuteCommand(ctx, "helm", args...)
	if err != nil {
		t.logger.Error("Error installing DRB leader node Helm chart", "err", err, "output", output)
		return nil, fmt.Errorf("failed to install DRB leader node Helm chart: %w", err)
	}

	t.logger.Info("✅ DRB leader node Helm chart installed successfully")
	t.logger.Info("⏳ Waiting for LoadBalancer address to become available...")

	// Wait for leader node LoadBalancer service to be ready
	var leaderNodeURL string
	for {
		leaderServices, err := utils.GetAddressByService(ctx, namespace, fmt.Sprintf("%s-leader", helmReleaseName))
		if err != nil {
			t.logger.Warn("Error retrieving leader service addresses", "err", err)
		} else if len(leaderServices) > 0 {
			leaderNodeURL = "http://" + leaderServices[0]
			break
		}
		time.Sleep(10 * time.Second)
	}
	t.logger.Infof("✅ Leader node is up and running. URL: %s", leaderNodeURL)

	return &types.DeployDRBApplicationOutput{
		LeaderNodeURL: leaderNodeURL,
	}, nil
}
