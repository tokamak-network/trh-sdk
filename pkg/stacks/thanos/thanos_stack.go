package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"os"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type ThanosStack struct {
	network string
	stack   string

	s3Client *s3.Client
}

func NewThanosStack(network string, stack string) *ThanosStack {
	return &ThanosStack{
		network: network,
		stack:   stack,
	}
}

// ----------------------------------------- Deploy contracts command  ----------------------------- //

func (t *ThanosStack) DeployContracts(ctx context.Context, deployConfig *types.Config) error {
	fileName := fmt.Sprintf("logs/deploy_thanos_%s_%s_%d.log", t.stack, t.network, time.Now().Unix())
	logging.InitLogger(fileName)
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet)
	}
	if t.network != constants.Testnet && t.network != constants.Mainnet {
		return fmt.Errorf("network %s does not support", t.network)
	}

	var (
		err      error
		isResume bool
	)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error obtaining current working directory:", err)
		return err
	}

	if deployConfig == nil {
		deployConfig = &types.Config{
			Stack:   t.stack,
			Network: t.network,
		}
	}

	if deployConfig.DeployContractState != nil {
		if deployConfig.DeployContractState.Status == types.DeployContractStatusCompleted {
			fmt.Println("The contracts have already been deployed successfully.")
			fmt.Print("Do you want to deploy the contracts again? (y/N): ")
			isDeployAgain, err := scanner.ScanBool(false)
			if err != nil {
				fmt.Println("Error reading the deploy again input:", err)
				return err
			}
			if !isDeployAgain {
				return nil
			}
		} else if deployConfig.DeployContractState.Status == types.DeployContractStatusInProgress {
			fmt.Print("The contracts deployment is in progress. Do you want to resume? (Y/n): ")
			isResume, err = scanner.ScanBool(true)
			if err != nil {
				fmt.Println("Error reading the resume input:", err)
				return err
			}
		}
	}

	if isResume {
		l1Rpc := deployConfig.L1RPCURL
		l1Client, err := ethclient.DialContext(ctx, l1Rpc)
		if err != nil {
			fmt.Printf("Failed to connect to L1 RPC: %s", err)
			return err
		}

		err = t.deployContracts(ctx, l1Client, deployConfig, true)
		if err != nil {
			fmt.Print("\r❌ Resume the contracts deployment failed!       \n")
			return err
		}
	} else {
		// STEP 1. Input the parameters
		fmt.Println("You are about to deploy the L1 contracts.")
		deployContractsConfig, err := t.inputDeployContracts(ctx)
		if err != nil {
			return err
		}

		// Download testnet dependencies file
		err = utils.ExecuteCommandStream("bash", "-c", "curl -o ./install-testnet-packages.sh https://raw.githubusercontent.com/tokamak-network/trh-sdk/refs/heads/main/scripts/install-testnet-packages.sh && chmod +x ./install-testnet-packages.sh")
		if err != nil {
			fmt.Println("\r❌ Failed to download testnet dependencies file!")
		}

		// Install the dependencies
		err = utils.ExecuteCommandStream("bash", "-c", "bash ./install-testnet-packages.sh")
		if err != nil {
			fmt.Println("\r❌ Failed to install testnet dependencies!")
		}

		shellConfigFile := utils.GetShellConfigDefault()

		// Check dependencies
		if !dependencies.CheckPnpmInstallation() {
			fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
			return nil
		}

		if !dependencies.CheckFoundryInstallation() {
			fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
			return nil
		}

		l1Client, err := ethclient.DialContext(ctx, deployContractsConfig.l1RPCurl)
		if err != nil {
			return err
		}

		l2ChainID, err := utils.GenerateL2ChainId()
		if err != nil {
			fmt.Printf("Failed to generate L2ChainID: %s", err)
			return err
		}

		deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig, l2ChainID)

		// Select operators Accounts
		operators, err := selectAccounts(ctx, l1Client, deployContractsConfig.fraudProof, deployContractsConfig.seed)
		if err != nil {
			return err
		}

		if len(operators) == 0 {
			return fmt.Errorf("no operators were found")
		}

		deployConfig.AdminPrivateKey = operators[0].PrivateKey
		deployConfig.SequencerPrivateKey = operators[1].PrivateKey
		deployConfig.BatcherPrivateKey = operators[2].PrivateKey
		deployConfig.ProposerPrivateKey = operators[3].PrivateKey
		if deployContractsConfig.fraudProof {
			if operators[4] == nil {
				return fmt.Errorf("challenger operator is required for fault proof but was not found")
			}
			deployConfig.ChallengerPrivateKey = operators[4].PrivateKey
		}
		deployConfig.DeploymentPath = fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%d-deploy.json", cwd, deployContractsTemplate.L1ChainID)
		deployConfig.L1RPCProvider = deployContractsConfig.l1Provider
		deployConfig.L1ChainID = deployContractsTemplate.L1ChainID
		deployConfig.L2ChainID = l2ChainID
		deployConfig.L1RPCURL = deployContractsConfig.l1RPCurl
		deployConfig.EnableFraudProof = deployContractsConfig.fraudProof
		deployConfig.ChainConfiguration = deployContractsConfig.ChainConfiguration

		err = makeDeployContractConfigJsonFile(ctx, l1Client, operators, deployContractsTemplate)
		if err != nil {
			return err
		}

		// STEP 2. Clone the repository
		err = t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
		if err != nil {
			return err
		}

		// STEP 3. Build the contracts
		fmt.Println("Building smart contracts...")
		err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh build")
		if err != nil {
			fmt.Print("\r❌ Build the contracts failed!       \n")
			return err
		}
		fmt.Print("\r✅ Build the contracts completed!       \n")

		// STEP 4. Deploy the contracts
		// Check admin balance and estimated deployment cost
		adminAddress := operators[0].Address
		balance, err := l1Client.BalanceAt(ctx, common.HexToAddress(adminAddress), nil)
		if err != nil {
			fmt.Printf("❌ Failed to retrieve admin account balance: %v\n", err)
			return err
		}
		fmt.Printf("Admin account balance: %.2f ETH\n", utils.WeiToEther(balance))

		// Estimate gas price
		gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
		if err != nil {
			fmt.Printf("❌ Failed to get gas price: %v\n", err)
			return err
		}
		fmt.Printf("⛽ Current gas price: %.4f Gwei\n", new(big.Float).Quo(new(big.Float).SetInt(gasPriceWei), big.NewFloat(1e9)))

		// Estimate deployment cost
		estimatedCost := new(big.Int).Mul(gasPriceWei, estimatedDeployContracts)
		estimatedCost.Mul(estimatedCost, big.NewInt(2))
		fmt.Printf("💰 Estimated deployment cost: %.4f ETH\n", utils.WeiToEther(estimatedCost))

		// Check if balance is sufficient
		if balance.Cmp(estimatedCost) < 0 {
			fmt.Println("❌ Insufficient balance for deployment.")
			return fmt.Errorf("admin account balance (%.4f ETH) is less than estimated deployment cost (%.4f  ETH)", utils.WeiToEther(balance), utils.WeiToEther(estimatedCost))
		} else {
			fmt.Println("✅ The admin account has sufficient balance to proceed with deployment.")
		}

		fmt.Print("🔎 The SDK is ready to deploy the contracts to the L1 network. Do you want to proceed(Y/n)? ")
		confirmation, err := scanner.ScanBool(true)
		if err != nil {
			return err
		}
		if !confirmation {
			return nil
		}

		deployConfig.DeployContractState = &types.DeployContractState{
			Status: types.DeployContractStatusInProgress,
		}
		err = deployConfig.WriteToJSONFile()
		if err != nil {
			fmt.Println("Failed to write settings file:", err)
			return err
		}

		err = t.deployContracts(ctx, l1Client, deployConfig, false)
		if err != nil {
			fmt.Print("\r❌ Deploy the contracts failed!       \n")
		}
	}

	// STEP 5: Generate the genesis and rollup files
	err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh generate -e .env -c deploy-config.json")
	fmt.Println("Generating the rollup and genesis files...")
	if err != nil {
		fmt.Print("\r❌ Failed to generate rollup and genesis files!       \n")
		return err
	}
	fmt.Print("\r✅ Successfully generated rollup and genesis files!       \n")
	fmt.Printf("\r Genesis file path: %s/tokamak-thanos/build/genesis.json\n", cwd)
	fmt.Printf("\r Rollup file path: %s/tokamak-thanos/build/rollup.json\n", cwd)

	fmt.Printf("✅ Configuration successfully saved to: %s/settings.json \n", cwd)
	return nil
}

func (t *ThanosStack) deployContracts(ctx context.Context,
	l1Client *ethclient.Client, deployConfig *types.Config,
	isResume bool,
) error {
	var (
		adminPrivateKey = deployConfig.AdminPrivateKey
		l1RPC           = deployConfig.L1RPCURL
	)

	fmt.Println("Deploying the contracts...")

	gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		fmt.Printf("Failed to get gas price: %v\n", err)
	}

	envValues := fmt.Sprintf("export GS_ADMIN_PRIVATE_KEY=%s\nexport L1_RPC_URL=%s\n", adminPrivateKey, l1RPC)
	if gasPriceWei != nil && gasPriceWei.Uint64() > 0 {
		// double gas price
		envValues += fmt.Sprintf("export GAS_PRICE=%d\n", gasPriceWei.Uint64()*2)
	}

	// STEP 4.1. Generate the .env file
	_, err = utils.ExecuteCommand(
		"bash",
		"-c",
		fmt.Sprintf("cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && echo '%s' > .env", envValues),
	)
	if err != nil {
		fmt.Print("\r❌ Make .env file failed!       \n")
		return err
	}

	// STEP 4.2. Copy the config file into the scripts folder
	err = utils.ExecuteCommandStream("bash", "-c", "cp ./deploy-config.json tokamak-thanos/packages/tokamak/contracts-bedrock/scripts")
	if err != nil {
		fmt.Print("\r❌ Copy the config file successfully!       \n")
		return err
	}

	// STEP 4.3. Deploy contracts
	if isResume {
		err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh redeploy -e .env -c deploy-config.json")
		if err != nil {
			fmt.Print("\r❌ Contract deployment failed!       \n")
			return err
		}
	} else {
		err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh deploy -e .env -c deploy-config.json")
		if err != nil {
			fmt.Print("\r❌ Contract deployment failed!       \n")
			return err
		}
	}
	fmt.Print("\r✅ Contract deployment completed successfully!       \n")

	deployConfig.DeployContractState.Status = types.DeployContractStatusCompleted
	err = deployConfig.WriteToJSONFile()
	if err != nil {
		fmt.Println("Failed to write settings file:", err)
		return err
	}
	return nil
}

// ----------------------------------------- Deploy command  ----------------------------- //

func (t *ThanosStack) Deploy(ctx context.Context, deployConfig *types.Config) error {
	fileName := fmt.Sprintf("logs/deploy_thanos_%s_%s_%d.log", t.stack, t.network, time.Now().Unix())
	logging.InitLogger(fileName)
	switch t.network {
	case constants.LocalDevnet:
		err := t.deployLocalDevnet()
		if err != nil {
			fmt.Printf("Error deploying local devnet: %s", err)
			return t.destroyDevnet()
		}
	case constants.Testnet, constants.Mainnet:
		// Check L1 RPC URL
		if deployConfig.L1RPCURL == "" {
			return fmt.Errorf("L1 RPC URL is not set. Please run the deploy-contracts command first")
		}

		var (
			blockNo uint64
			err     error
		)

		ctxTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		client, err := ethclient.DialContext(ctxTimeout, deployConfig.L1RPCURL)
		if client != nil {
			blockNo, err = client.BlockNumber(ctxTimeout)
			if err != nil {
				fmt.Printf("❌ Failed to retrieve block number: %s \n", err)
			} else {
				fmt.Printf("✅ Successfully connected to L1 RPC, current block number: %d \n", blockNo)
			}
		}
		if err != nil {
			fmt.Printf("❌ Can't connect to L1 RPC. Please try again: %s \n", err)
			l1RPC, l1RPCKind, _, err := t.inputL1RPC(ctx)
			if err != nil {
				fmt.Printf("Error while getting L1 RPC URL: %s", err)
				return err
			}
			deployConfig.L1RPCURL = l1RPC
			deployConfig.L1RPCProvider = l1RPCKind
			err = deployConfig.WriteToJSONFile()
			if err != nil {
				fmt.Println("Failed to write settings file after getting L1 RPC", err)
				return err
			}
		}

		fmt.Print("Please select your infrastructure provider [AWS] (default: AWS): ")
		input, err := scanner.ScanString()
		if err != nil {
			fmt.Printf("Error reading infrastructure selection: %s", err)
			return err
		}
		infraOpt := strings.ToLower(input)
		if infraOpt == "" {
			infraOpt = constants.AWS
		}

		switch infraOpt {
		case constants.AWS:
			err = t.deployNetworkToAWS(ctx, deployConfig)
			if err != nil {
				return t.destroyInfraOnAWS(ctx, deployConfig)
			}
			return nil
		default:
			return fmt.Errorf("infrastructure provider %s is not supported", infraOpt)
		}
	default:
		return fmt.Errorf("network %s is not supported", t.network)
	}

	return nil
}

func (t *ThanosStack) deployLocalDevnet() error {
	// Download testnet dependencies file
	err := utils.ExecuteCommandStream("bash", "-c", "curl -o ./install-devnet-packages.sh https://raw.githubusercontent.com/tokamak-network/trh-sdk/refs/heads/main/scripts/install-devnet-packages.sh && chmod +x ./install-devnet-packages.sh")
	if err != nil {
		fmt.Println("\r❌ Failed to download devnet dependencies file!")
	}

	// Install the dependencies
	err = utils.ExecuteCommandStream("bash", "-c", "bash ./install-devnet-packages.sh")
	if err != nil {
		fmt.Println("\r❌ Failed to install devnet dependencies!")
	}

	err = t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
	if err != nil {
		return err
	}

	// STEP 2. Source the config file
	shellConfigFile := utils.GetShellConfigDefault()

	// Source the shell configuration file
	err = utils.ExecuteCommandStream("bash", "-c", fmt.Sprintf("source %s", shellConfigFile))
	if err != nil {
		return err
	}

	// STEP 3. Start the devnet
	fmt.Println("Starting the devnet...")
	fmt.Print("\r✅ Package installation completed successfully!       \n")

	err = utils.ExecuteCommandStream("bash", "-l", "-c", "cd tokamak-thanos && export DEVNET_L2OO=true && make devnet-up")
	if err != nil {
		fmt.Print("\r❌ Failed to start devnet!       \n")
		return err
	}

	fmt.Print("\r✅ Devnet started successfully!       \n")

	return nil
}

func (t *ThanosStack) deployNetworkToAWS(ctx context.Context, deployConfig *types.Config) error {
	shellConfigFile := utils.GetShellConfigDefault()

	// Check dependencies
	// STEP 1. Verify required dependencies
	if !dependencies.CheckTerraformInstallation() {
		fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
		return nil
	}

	if !dependencies.CheckHelmInstallation() {
		fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
		return nil
	}

	if !dependencies.CheckAwsCLIInstallation() {
		fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
		return nil
	}

	if !dependencies.CheckK8sInstallation() {
		fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
		return nil
	}

	// STEP 1. Clone the charts repository
	err := t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return err
	}

	// STEP 2. AWS Authentication
	awsProfile, awsLoginInputs, err := t.loginAWS(ctx, deployConfig)
	if err != nil {
		fmt.Println("Error authenticating with AWS:", err)
		return err
	}

	deployConfig.AWS = awsLoginInputs
	if err := deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	fmt.Println("⚡️Removing the previous deployment state...")
	err = t.clearTerraformState(ctx)
	if err != nil {
		fmt.Printf("Failed to clear the existing terraform state, err: %s", err.Error())
		return err
	}

	fmt.Println("✅ Removed the previous deployment state...")

	inputs, err := t.inputDeployInfra(deployConfig.L1ChainID)
	if err != nil {
		fmt.Println("Error collecting infrastructure deployment parameters:", err)
		return err
	}

	var (
		chainConfiguration = deployConfig.ChainConfiguration
	)

	if chainConfiguration == nil {
		return fmt.Errorf("chain configuration is not set")
	}

	// STEP 3. Create .envrc file
	err = makeTerraformEnvFile("tokamak-thanos-stack/terraform", types.TerraformEnvConfig{
		ThanosStackName:     inputs.ChainName,
		AwsRegion:           awsLoginInputs.Region,
		SequencerKey:        deployConfig.SequencerPrivateKey,
		BatcherKey:          deployConfig.BatcherPrivateKey,
		ProposerKey:         deployConfig.ProposerPrivateKey,
		ChallengerKey:       deployConfig.ChallengerPrivateKey,
		EksClusterAdmins:    awsProfile.Arn,
		DeploymentsPath:     deployConfig.DeploymentPath,
		L1BeaconUrl:         inputs.L1BeaconURL,
		L1RpcUrl:            deployConfig.L1RPCURL,
		L1RpcProvider:       deployConfig.L1RPCProvider,
		Azs:                 awsProfile.AvailabilityZones,
		ThanosStackImageTag: constants.DockerImageTag[deployConfig.Network].ThanosStackImageTag,
		OpGethImageTag:      constants.DockerImageTag[deployConfig.Network].OpGethImageTag,
		MaxChannelDuration:  chainConfiguration.GetMaxChannelDuration(),
	})
	if err != nil {
		fmt.Println("Error generating Terraform environment configuration:", err)
		return err
	}

	// STEP 4. Initialize Terraform backend
	err = utils.ExecuteCommandStream("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd backend &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve
		`,
	}...)
	if err != nil {
		fmt.Println("Error initializing Terraform backend:", err)
		return err
	}

	// STEP 5. Copy configuration files
	err = utils.CopyFile("tokamak-thanos/build/rollup.json", "tokamak-thanos-stack/terraform/thanos-stack/config-files/rollup.json")
	if err != nil {
		fmt.Println("Error copying rollup configuration:", err)
		return err
	}
	err = utils.CopyFile("tokamak-thanos/build/genesis.json", "tokamak-thanos-stack/terraform/thanos-stack/config-files/genesis.json")
	if err != nil {
		fmt.Println("Error copying genesis configuration:", err)
		return err
	}

	fmt.Println("Deploying Thanos stack infrastructure")
	// STEP 6. Deploy Thanos stack infrastructure
	err = utils.ExecuteCommandStream("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve`,
	}...)
	if err != nil {
		fmt.Println("Error deploying Thanos stack infrastructure:", err)
		return err
	}

	// Get VPC ID
	vpcIdOutput, err := utils.ExecuteCommand("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform output -json vpc_id`,
	}...)
	if err != nil {
		return fmt.Errorf("failed to get terraform output for %s: %w", "vpc_id", err)
	}

	deployConfig.AWS.VpcID = strings.Trim(vpcIdOutput, `"`)
	if err := deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	thanosStackValueFileExist := utils.CheckFileExists("tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml")
	if !thanosStackValueFileExist {
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	deployConfig.ChainName = inputs.ChainName
	if err := deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Step 7. Configure EKS access
	eksSetup, err := utils.ExecuteCommand("aws", []string{
		"eks",
		"update-kubeconfig",
		"--region", awsLoginInputs.Region,
		"--name", namespace,
	}...)
	if err != nil {
		fmt.Println("Error configuring EKS access:", err, "details:", eksSetup)
		return err
	}

	fmt.Println("EKS configuration updated:", eksSetup)

	// Step 7.1. Check if K8s cluster is ready
	fmt.Println("Checking if K8s cluster is ready...")
	k8sReady, err := utils.CheckK8sReady(namespace)
	if err != nil {
		fmt.Println("❌ Error checking K8s cluster readiness:", err)
		return err
	}
	fmt.Printf("✅ K8s cluster is ready: %t\n", k8sReady)

	// ---------------------------------------- Deploy chain --------------------------//
	// Step 8. Add Helm repository
	helmAddOuput, err := utils.ExecuteCommand("helm", []string{
		"repo",
		"add",
		"thanos-stack",
		"https://tokamak-network.github.io/tokamak-thanos-stack",
	}...)
	if err != nil {
		fmt.Println("Error adding Helm repository:", err, "details:", helmAddOuput)
		return err
	}

	// Step 8.1 Search available Helm charts
	helmSearchOutput, err := utils.ExecuteCommand("helm", []string{
		"search",
		"repo",
		"thanos-stack",
	}...)
	if err != nil {
		fmt.Println("Error searching Helm charts:", err, "details:", helmSearchOutput)
		return err
	}
	fmt.Println("Helm repository added successfully: \n", helmSearchOutput)

	// Step 8.2. Install Helm charts
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())
	_, err = utils.ExecuteCommand("helm", []string{
		"install",
		helmReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", cwd),
		"--values",
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", cwd),
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing Helm charts:", err)
		return err
	}

	fmt.Println("✅ Helm charts installed successfully")

	var l2RPCUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, helmReleaseName)
		if err != nil {
			fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
			return err
		}

		if len(k8sIngresses) > 0 {
			l2RPCUrl = "http://" + k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	fmt.Printf("✅ Network deployment completed successfully!\n")
	fmt.Printf("🌐 RPC endpoint: %s\n", l2RPCUrl)

	deployConfig.K8s = &types.K8sConfig{
		Namespace: namespace,
	}
	deployConfig.L2RpcUrl = l2RPCUrl
	deployConfig.L1BeaconURL = inputs.L1BeaconURL

	err = deployConfig.WriteToJSONFile()
	if err != nil {
		fmt.Println("Error saving configuration file:", err)
		return err
	}
	fmt.Printf("Configuration saved successfully to: %s/settings.json \n", cwd)

	// After installing the infra successfully, we install the bridge
	err = t.installBridge(ctx, deployConfig)
	if err != nil {
		fmt.Println("Error installing bridge:", err)
	}
	fmt.Println("🎉 Thanos Stack installation completed successfully!")
	fmt.Println("🚀 Your network is now up and running.")
	fmt.Println("🔧 You can start interacting with your deployed infrastructure.")

	return nil
}

// --------------------------------------------- Destroy command -------------------------------------//

func (t *ThanosStack) Destroy(ctx context.Context, deployConfig *types.Config) error {
	fileName := fmt.Sprintf("logs/destroy_thanos_%s_%s_%d.log", t.stack, t.network, time.Now().Unix())
	logging.InitLogger(fileName)
	switch t.network {
	case constants.LocalDevnet:
		return t.destroyDevnet()
	case constants.Testnet, constants.Mainnet:
		return t.destroyInfraOnAWS(ctx, deployConfig)
	}
	return nil
}

func (t *ThanosStack) destroyDevnet() error {
	output, err := utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && make nuke")
	if err != nil {
		fmt.Printf("\r❌ Devnet cleanup failed!       \n Details: %s", output)
		return err
	}

	fmt.Print("\r✅ Devnet network destroyed successfully!       \n")

	return nil
}

func (t *ThanosStack) destroyInfraOnAWS(ctx context.Context, deployConfig *types.Config) error {
	var (
		err error
	)

	_, _, err = t.loginAWS(ctx, deployConfig)
	if err != nil {
		fmt.Println("Error getting AWS profile:", err)
		return err
	}

	var namespace string
	if deployConfig.K8s != nil {
		namespace = deployConfig.K8s.Namespace
	}

	helmReleases, err := utils.GetHelmReleases(namespace)
	if err != nil {
		fmt.Println("Error retrieving Helm releases:", err)
	}

	if len(helmReleases) > 0 {
		for _, release := range helmReleases {
			if strings.Contains(release, namespace) || strings.Contains(release, "op-bridge") || strings.Contains(release, "block-explorer") {
				fmt.Printf("Uninstalling Helm release: %s in namespace: %s...\n", release, namespace)
				_, err := utils.ExecuteCommand("helm", "uninstall", release, "--namespace", namespace)
				if err != nil {
					fmt.Println("Error removing Helm release:", err)
					return err
				}
			}
		}

		fmt.Println("Helm release removed successfully")
	}

	// Delete namespace before destroying the infrastructure
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err = t.tryToDeleteK8sNamespace(ctxTimeout, namespace)
	if err != nil {
		fmt.Println("Error deleting namespace:", err)
	} else {
		fmt.Println("✅ Namespace destroyed successfully!")
	}

	return t.clearTerraformState(ctx)
}

// ------------------------------------------ Install plugins ---------------------------

func (t *ThanosStack) InstallPlugins(ctx context.Context, pluginNames []string, deployConfig *types.Config) error {
	fileName := fmt.Sprintf("logs/install_plugins_%s_%s_%d.log", t.stack, t.network, time.Now().Unix())
	logging.InitLogger(fileName)
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s does not support plugin installation", constants.LocalDevnet)
	}

	if deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = deployConfig.K8s.Namespace
	)

	for _, pluginName := range pluginNames {
		if !constants.SupportedPlugins[pluginName] {
			fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
			continue
		}

		fmt.Printf("Installing plugin: %s in namespace: %s...\n", pluginName, namespace)

		switch pluginName {
		case constants.PluginBlockExplorer:
			err := t.installBlockExplorer(ctx, deployConfig)
			if err != nil {
				return t.uninstallBlockExplorer(ctx, deployConfig)
			}
			return nil
		case constants.PluginBridge:
			err := t.installBridge(ctx, deployConfig)
			if err != nil {
				return t.uninstallBridge(ctx, deployConfig)
			}
			return nil
		}
	}
	return nil
}

// ------------------------------------------ Uninstall plugins ---------------------------

func (t *ThanosStack) UninstallPlugins(ctx context.Context, pluginNames []string, deployConfig *types.Config) error {
	fileName := fmt.Sprintf("logs/uninstall_plugins_%s_%s_%d.log", t.stack, t.network, time.Now().Unix())
	logging.InitLogger(fileName)
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s does not support plugin installation", constants.LocalDevnet)
	}

	if deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = deployConfig.K8s.Namespace
	)

	for _, pluginName := range pluginNames {
		if !constants.SupportedPlugins[pluginName] {
			fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
			continue
		}

		fmt.Printf("Uninstalling plugin: %s in namespace: %s...\n", pluginName, namespace)

		switch pluginName {
		case constants.PluginBridge:
			return t.uninstallBridge(ctx, deployConfig)
		case constants.PluginBlockExplorer:
			return t.uninstallBlockExplorer(ctx, deployConfig)
		}
	}
	return nil
}
