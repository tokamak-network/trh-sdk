package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/ethclient"

	"os"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
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

func (t *ThanosStack) DeployContracts(ctx context.Context) error {
	logFileName := fmt.Sprintf("logs/deploy_contracts_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
	if t.network == constants.LocalDevnet {
		utils.LogToFile(logFileName, fmt.Sprintf("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet), false)
		return fmt.Errorf("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet)
	}
	if t.network != constants.Testnet && t.network != constants.Mainnet {
		utils.LogToFile(logFileName, fmt.Sprintf("network %s does not support", t.network), false)
		return fmt.Errorf("network %s does not support", t.network)
	}
	var err error

	// STEP 1. Input the parameters
	utils.LogToFile(logFileName, "You are about to deploy the L1 contracts.", true)
	deployContractsConfig, err := t.inputDeployContracts(ctx)
	if err != nil {
		return err
	}

	// Download testnet dependencies file
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "curl -o ./install-testnet-packages.sh https://raw.githubusercontent.com/tokamak-network/trh-sdk/refs/heads/main/scripts/install-testnet-packages.sh && chmod +x ./install-testnet-packages.sh")
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("\r❌ Failed to download testnet dependencies file: %v", err), true)
	}

	// Install the dependencies
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "bash ./install-testnet-packages.sh")
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("\r❌ Failed to install testnet dependencies: %v", err), true)
	}

	shellConfigFile := utils.GetShellConfigDefault()

	// Check dependencies
	if !dependencies.CheckPnpmInstallation(logFileName) {
		utils.LogToFile(logFileName, fmt.Sprintf("Try running `source %s` to set up your environment", shellConfigFile), true)
		return nil
	}

	if !dependencies.CheckFoundryInstallation(logFileName) {
		utils.LogToFile(logFileName, fmt.Sprintf("Try running `source %s` to set up your environment", shellConfigFile), true)
		return nil
	}

	l1Client, err := ethclient.DialContext(ctx, deployContractsConfig.l1RPCurl)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to connect to L1 client: %v", err), true)
		return err
	}
	l1ChainID, err := l1Client.ChainID(ctx)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to get chain id: %s", err), true)
		return err
	}

	l2ChainID, err := utils.GenerateL2ChainId()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to generate L2ChainID: %s", err), true)
		return err
	}

	deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig.fraudProof, l1ChainID, l2ChainID)

	// Select operators Accounts
	operators, err := selectAccounts(ctx, l1Client, deployContractsConfig.fraudProof, deployContractsConfig.seed)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to select accounts: %v", err), true)
		return err
	}

	if len(operators) == 0 {
		utils.LogToFile(logFileName, "no operators were found", true)
		return fmt.Errorf("no operators were found")
	}

	utils.LogToFile(logFileName, "The SDK is ready to deploy the contracts to the L1 environment. Do you want to proceed(Y/n)? ", true)
	confirmation, err := scanner.ScanBool(true)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error getting confirmation: %v", err), true)
		return err
	}
	if !confirmation {
		utils.LogToFile(logFileName, "User did not confirm the deployment", true)
		return nil
	}

	err = makeDeployContractConfigJsonFile(ctx, l1Client, operators, deployContractsTemplate)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to make deploy contract config file: %v", err), true)
		return err
	}

	// STEP 2. Clone the repository
	err = t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to clone repository: %v", err), true)
		return err
	}

	// STEP 3. Build the contracts
	utils.LogToFile(logFileName, "Building smart contracts...", true)
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh build")
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Build the contracts failed!", true)
		return err
	}
	utils.LogToFile(logFileName, "\r✅ Build the contracts completed!", true)

	// STEP 4. Deploy the contracts
	utils.LogToFile(logFileName, "Deploying the contracts...", true)

	gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to get gas price: %v", err), true)
	}

	envValues := fmt.Sprintf("export GS_ADMIN_PRIVATE_KEY=%s\nexport L1_RPC_URL=%s\n", operators[0].PrivateKey, deployContractsConfig.l1RPCurl)
	if gasPriceWei != nil && gasPriceWei.Uint64() > 0 {
		// double gas price
		envValues += fmt.Sprintf("export GAS_PRICE=%d\n", gasPriceWei.Uint64()*2)
	}

	// STEP 4.1. Generate the .env file
	_, err = utils.ExecuteCommand(
		"bash",
		logFileName,
		"-c",
		fmt.Sprintf("cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && echo '%s' > .env", envValues),
	)
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Make .env file failed!", true)
		return err
	}

	// STEP 4.2. Copy the config file into the scripts folder
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "cp ./deploy-config.json tokamak-thanos/packages/tokamak/contracts-bedrock/scripts")
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Copy the config file successfully!", true)
		return err
	}

	// STEP 4.3. Deploy contracts
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh deploy -e .env -c deploy-config.json")
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Contract deployment failed!", true)
		return err
	}
	utils.LogToFile(logFileName, "\r✅ Contract deployment completed successfully!", true)

	// STEP 5: Generate the genesis and rollup files
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh generate -e .env -c deploy-config.json")
	utils.LogToFile(logFileName, "Generating the rollup and genesis files...", true)
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Failed to generate rollup and genesis files!", true)
		return err
	}
	utils.LogToFile(logFileName, "\r✅ Successfully generated rollup and genesis files!", true)
	cwd, err := os.Getwd()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error obtaining current working directory: %v", err), true)
		return err
	}
	utils.LogToFile(logFileName, fmt.Sprintf("\r Genesis file path: %s/tokamak-thanos/build/genesis.json\n", cwd), true)
	utils.LogToFile(logFileName, fmt.Sprintf("\r Rollup file path: %s/tokamak-thanos/build/rollup.json\n", cwd), true)

	var challengerPrivateKey string
	if deployContractsConfig.fraudProof {
		if operators[4] == nil {
			return fmt.Errorf("challenger operator is required for fault proof but was not found")
		}
		challengerPrivateKey = operators[4].PrivateKey
	}
	cfg := &types.Config{
		AdminPrivateKey:      operators[0].PrivateKey,
		SequencerPrivateKey:  operators[1].PrivateKey,
		BatcherPrivateKey:    operators[2].PrivateKey,
		ProposerPrivateKey:   operators[3].PrivateKey,
		ChallengerPrivateKey: challengerPrivateKey,
		DeploymentPath:       fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%d-deploy.json", cwd, deployContractsTemplate.L1ChainID),
		L1RPCProvider:        deployContractsConfig.l1Provider,
		L1ChainID:            l1ChainID.Uint64(),
		L2ChainID:            l2ChainID,
		L1RPCURL:             deployContractsConfig.l1RPCurl,
		Stack:                t.stack,
		Network:              t.network,
		EnableFraudProof:     deployContractsConfig.fraudProof,
	}
	err = cfg.WriteToJSONFile()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to write settings file: %v", err), true)
		return err
	}
	utils.LogToFile(logFileName, fmt.Sprintf("✅ Configuration successfully saved to: %s/settings.json \n", cwd), true)
	return nil
}

// ----------------------------------------- Deploy command  ----------------------------- //

func (t *ThanosStack) Deploy(ctx context.Context, deployConfig *types.Config) error {
	logFileName := fmt.Sprintf("logs/deploy_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
	switch t.network {
	case constants.LocalDevnet:
		err := t.deployLocalDevnet()
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error deploying local devnet: %s", err), true)
			return t.destroyDevnet()
		}
	case constants.Testnet, constants.Mainnet:
		// Check L1 RPC URL
		if deployConfig.L1RPCURL == "" {
			utils.LogToFile(logFileName, "L1 RPC URL is not set. Please run the deploy-contracts command first", true)
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
				utils.LogToFile(logFileName, fmt.Sprintf("❌ Failed to retrieve block number: %s \n", err), true)
			} else {
				utils.LogToFile(logFileName, fmt.Sprintf("✅ Successfully connected to L1 RPC, current block number: %d \n", blockNo), true)
			}
		}
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("❌ Can't connect to L1 RPC. Please try again: %s \n", err), true)
			l1RPC, l1RPCKind, err := t.inputL1RPC(ctx)
			if err != nil {
				utils.LogToFile(logFileName, fmt.Sprintf("Error while getting L1 RPC URL: %s", err), true)
				return err
			}
			deployConfig.L1RPCURL = l1RPC
			deployConfig.L1RPCProvider = l1RPCKind
			err = deployConfig.WriteToJSONFile()
			if err != nil {
				utils.LogToFile(logFileName, fmt.Sprintf("Failed to write settings file after getting L1 RPC: %s", err), true)
				return err
			}
		}

		utils.LogToFile(logFileName, "Please select your infrastructure provider [AWS] (default: AWS): ", true)
		input, err := scanner.ScanString()
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error reading infrastructure selection: %s", err), true)
			return err
		}
		infraOpt := strings.ToLower(input)
		if infraOpt == "" {
			infraOpt = constants.AWS
		}

		switch infraOpt {
		case constants.AWS:
			err = t.deployNetworkToAWS(ctx, deployConfig, logFileName)
			if err != nil {
				return t.destroyInfraOnAWS(ctx, deployConfig)
			}
			return nil
		default:
			utils.LogToFile(logFileName, fmt.Sprintf("infrastructure provider %s is not supported", infraOpt), true)
			return fmt.Errorf("infrastructure provider %s is not supported", infraOpt)
		}
	default:
		utils.LogToFile(logFileName, fmt.Sprintf("network %s is not supported", t.network), true)
		return fmt.Errorf("network %s is not supported", t.network)
	}

	return nil
}

func (t *ThanosStack) deployLocalDevnet() error {
	logFileName := fmt.Sprintf("logs/deploy_local_devnet_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
	// Download testnet dependencies file
	err := utils.ExecuteCommandStream("bash", logFileName, "-c", "curl -o ./install-devnet-packages.sh https://raw.githubusercontent.com/tokamak-network/trh-sdk/refs/heads/main/scripts/install-devnet-packages.sh && chmod +x ./install-devnet-packages.sh")
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Failed to download devnet dependencies file!", true)
	}

	// Install the dependencies
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", "bash ./install-devnet-packages.sh")
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Failed to install devnet dependencies!", true)
	}

	err = t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to clone repository: %s", err), true)
		return err
	}

	// STEP 2. Source the config file
	shellConfigFile := utils.GetShellConfigDefault()

	// Source the shell configuration file
	err = utils.ExecuteCommandStream("bash", logFileName, "-c", fmt.Sprintf("source %s", shellConfigFile))
	if err != nil {
		return err
	}

	// STEP 3. Start the devnet
	utils.LogToFile(logFileName, "Starting the devnet...", true)
	utils.LogToFile(logFileName, "\r✅ Package installation completed successfully!       \n", true)

	err = utils.ExecuteCommandStream("bash", logFileName, "-l", "-c", "cd tokamak-thanos && export DEVNET_L2OO=true && make devnet-up")
	if err != nil {
		utils.LogToFile(logFileName, "\r❌ Failed to start devnet!", true)
		return err
	}

	utils.LogToFile(logFileName, "\r✅ Devnet started successfully!       \n", true)

	return nil
}

func (t *ThanosStack) deployNetworkToAWS(ctx context.Context, deployConfig *types.Config, logFileName string) error {
	shellConfigFile := utils.GetShellConfigDefault()

	// Check dependencies
	// STEP 1. Verify required dependencies
	if !dependencies.CheckTerraformInstallation(logFileName) {
		utils.LogToFile(logFileName, fmt.Sprintf("Try running `source %s` to set up your environment \n", shellConfigFile), true)
		return nil
	}

	if !dependencies.CheckHelmInstallation(logFileName) {
		utils.LogToFile(logFileName, fmt.Sprintf("Try running `source %s` to set up your environment \n", shellConfigFile), true)
		return nil
	}

	if !dependencies.CheckAwsCLIInstallation(logFileName) {
		utils.LogToFile(logFileName, fmt.Sprintf("Try running `source %s` to set up your environment \n", shellConfigFile), true)
		return nil
	}

	if !dependencies.CheckK8sInstallation(logFileName) {
		utils.LogToFile(logFileName, fmt.Sprintf("Try running `source %s` to set up your environment \n", shellConfigFile), true)
		return nil
	}

	// STEP 1. Clone the charts repository
	err := t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error cloning repository: %s", err), true)
		return err
	}

	// STEP 2. AWS Authentication
	awsProfile, awsLoginInputs, err := t.loginAWS(ctx, deployConfig)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error authenticating with AWS: %s", err), true)
		return err
	}

	deployConfig.AWS = awsLoginInputs
	if err := deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	utils.LogToFile(logFileName, "⚡️Removing the previous deployment state...", true)
	err = t.clearTerraformState(ctx, logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Failed to clear the existing terraform state, err: %s", err.Error()), true)
		return err
	}

	utils.LogToFile(logFileName, "✅ Removed the previous deployment state...", true)

	inputs, err := t.inputDeployInfra()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error collecting infrastructure deployment parameters: %s", err), true)
		return err
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
	})
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error generating Terraform environment configuration: %s", err), true)
		return err
	}
	// STEP 4. Initialize Terraform backend
	err = utils.ExecuteCommandStream("bash", logFileName, []string{
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
		utils.LogToFile(logFileName, fmt.Sprintf("Error initializing Terraform backend: %s", err), true)
		return err
	}

	// STEP 5. Copy configuration files
	err = utils.CopyFile("tokamak-thanos/build/rollup.json", "tokamak-thanos-stack/terraform/thanos-stack/config-files/rollup.json", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error copying rollup configuration: %s", err), true)
		return err
	}
	err = utils.CopyFile("tokamak-thanos/build/genesis.json", "tokamak-thanos-stack/terraform/thanos-stack/config-files/genesis.json", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error copying genesis configuration: %s", err), true)
		return err
	}

	utils.LogToFile(logFileName, "Deploying Thanos stack infrastructure", true)
	// STEP 6. Deploy Thanos stack infrastructure
	err = utils.ExecuteCommandStream("bash", logFileName, []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve`,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error deploying Thanos stack infrastructure: %s", err), true)
		return err
	}

	// Get VPC ID
	vpcIdOutput, err := utils.ExecuteCommand("bash", logFileName, []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform output -json vpc_id`,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("failed to get terraform output for %s: %s", "vpc_id", err), true)
		return err
	}

	deployConfig.AWS.VpcID = strings.Trim(vpcIdOutput, `"`)
	if err := deployConfig.WriteToJSONFile(); err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("failed to write settings file: %s", err), true)
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	thanosStackValueFileExist := utils.CheckFileExists("tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml")
	if !thanosStackValueFileExist {
		utils.LogToFile(logFileName, "configuration file thanos-stack-values.yaml not found", true)
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	deployConfig.ChainName = inputs.ChainName
	if err := deployConfig.WriteToJSONFile(); err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("failed to write settings file: %s", err), true)
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Step 7. Configure EKS access
	eksSetup, err := utils.ExecuteCommand("aws", logFileName, []string{
		"eks",
		"update-kubeconfig",
		"--region", awsLoginInputs.Region,
		"--name", namespace,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error configuring EKS access: %s", err), true)
		return err
	}

	utils.LogToFile(logFileName, fmt.Sprintf("EKS configuration updated: %s", eksSetup), true)

	// ---------------------------------------- Deploy chain --------------------------//
	// Step 8. Add Helm repository
	helmAddOuput, err := utils.ExecuteCommand("helm", logFileName, []string{
		"repo",
		"add",
		"thanos-stack",
		"https://tokamak-network.github.io/tokamak-thanos-stack",
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error adding Helm repository: %s, details: %s", err, helmAddOuput), true)
		return err
	}

	// Step 8.1 Search available Helm charts
	helmSearchOutput, err := utils.ExecuteCommand("helm", logFileName, []string{
		"search",
		"repo",
		"thanos-stack",
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error searching Helm charts: %s, details: %s", err, helmSearchOutput), true)
		return err
	}
	utils.LogToFile(logFileName, fmt.Sprintf("Helm repository added successfully: \n%s", helmSearchOutput), true)

	// Step 8.2. Install Helm charts
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())
	_, err = utils.ExecuteCommand("helm", logFileName, []string{
		"install",
		helmReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", cwd),
		"--values",
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", cwd),
		"--namespace",
		namespace,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error installing Helm charts: %s", err), true)
		return err
	}

	utils.LogToFile(logFileName, "✅ Helm charts installed successfully", true)

	var l2RPCUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, helmReleaseName, logFileName)
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error retrieving ingress addresses: %s, details: %s", err, k8sIngresses), true)
			return err
		}

		if len(k8sIngresses) > 0 {
			l2RPCUrl = "http://" + k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	utils.LogToFile(logFileName, "✅ Network deployment completed successfully!\n", true)
	utils.LogToFile(logFileName, fmt.Sprintf("🌐 RPC endpoint: %s\n", l2RPCUrl), true)

	deployConfig.K8s = &types.K8sConfig{
		Namespace: namespace,
	}
	deployConfig.L2RpcUrl = l2RPCUrl
	deployConfig.L1BeaconURL = inputs.L1BeaconURL

	err = deployConfig.WriteToJSONFile()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error saving configuration file: %s", err), true)
		return err
	}
	utils.LogToFile(logFileName, fmt.Sprintf("Configuration saved successfully to: %s/settings.json \n", cwd), true)

	// After installing the infra successfully, we install the bridge
	err = t.installBridge(ctx, deployConfig, logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error installing bridge: %s", err), true)
	}
	utils.LogToFile(logFileName, "🎉 Thanos Stack installation completed successfully!", true)
	utils.LogToFile(logFileName, "🚀 Your network is now up and running.", true)
	utils.LogToFile(logFileName, "🔧 You can start interacting with your deployed infrastructure.", true)

	return nil
}

// --------------------------------------------- Destroy command -------------------------------------//

func (t *ThanosStack) Destroy(ctx context.Context, deployConfig *types.Config) error {
	switch t.network {
	case constants.LocalDevnet:
		return t.destroyDevnet()
	case constants.Testnet, constants.Mainnet:
		return t.destroyInfraOnAWS(ctx, deployConfig)
	}
	return nil
}

func (t *ThanosStack) destroyDevnet() error {
	logFileName := fmt.Sprintf("logs/destroy_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
	output, err := utils.ExecuteCommand("bash", logFileName, []string{
		"-c",
		"cd tokamak-thanos && make nuke",
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("\r❌ Devnet cleanup failed!       \n Details: %s", output), true)
		return err
	}

	utils.LogToFile(logFileName, "\r✅ Devnet network destroyed successfully!       \n", true)

	return nil
}

func (t *ThanosStack) destroyInfraOnAWS(ctx context.Context, deployConfig *types.Config) error {
	logFileName := fmt.Sprintf("logs/destroy_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
	var (
		err error
	)

	_, _, err = t.loginAWS(ctx, deployConfig)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error getting AWS profile: %s", err), true)
		return err
	}

	var namespace string
	if deployConfig.K8s != nil {
		namespace = deployConfig.K8s.Namespace
	}

	helmReleases, err := utils.GetHelmReleases(namespace, logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error retrieving Helm releases: %s", err), true)
	}

	if len(helmReleases) > 0 {
		for _, release := range helmReleases {
			if strings.Contains(release, namespace) || strings.Contains(release, "op-bridge") || strings.Contains(release, "block-explorer") {
				utils.LogToFile(logFileName, fmt.Sprintf("Uninstalling Helm release: %s in namespace: %s...\n", release, namespace), true)
				_, err := utils.ExecuteCommand("helm", logFileName, []string{
					"uninstall",
					release,
					"--namespace",
					namespace,
				}...)
				if err != nil {
					utils.LogToFile(logFileName, fmt.Sprintf("Error removing Helm release: %s", err), true)
					return err
				}
			}
		}

		utils.LogToFile(logFileName, "Helm release removed successfully", true)
	}

	// Delete namespace before destroying the infrastructure
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err = t.tryToDeleteK8sNamespace(ctxTimeout, namespace, logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error deleting namespace: %s", err), true)
	} else {
		utils.LogToFile(logFileName, "✅ Namespace destroyed successfully!", true)
	}

	return t.clearTerraformState(ctx, logFileName)
}

// ------------------------------------------ Install plugins ---------------------------

func (t *ThanosStack) InstallPlugins(ctx context.Context, pluginNames []string, deployConfig *types.Config) error {
	logFileName := fmt.Sprintf("logs/install_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
	if t.network == constants.LocalDevnet {
		utils.LogToFile(logFileName, fmt.Sprintf("network %s does not support plugin installation", constants.LocalDevnet), true)
		return fmt.Errorf("network %s does not support plugin installation", constants.LocalDevnet)
	}

	if deployConfig.K8s == nil {
		utils.LogToFile(logFileName, "K8s configuration is not set. Please run the deploy command first", true)
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = deployConfig.K8s.Namespace
	)

	for _, pluginName := range pluginNames {
		if !constants.SupportedPlugins[pluginName] {
			utils.LogToFile(logFileName, fmt.Sprintf("Plugin %s is not supported for this stack.", pluginName), true)
			continue
		}

		utils.LogToFile(logFileName, fmt.Sprintf("Installing plugin: %s in namespace: %s...", pluginName, namespace), true)

		switch pluginName {
		case constants.PluginBlockExplorer:
			err := t.installBlockExplorer(ctx, deployConfig, logFileName)
			if err != nil {
				return t.uninstallBlockExplorer(ctx, deployConfig, logFileName)
			}
			utils.LogToFile(logFileName, fmt.Sprintf("Plugin %s installed successfully", pluginName), true)
			return nil
		case constants.PluginBridge:
			err := t.installBridge(ctx, deployConfig, logFileName)
			if err != nil {
				return t.uninstallBridge(ctx, deployConfig, logFileName)
			}
			utils.LogToFile(logFileName, fmt.Sprintf("Plugin %s installed successfully", pluginName), true)
			return nil
		}
	}
	return nil
}

// ------------------------------------------ Uninstall plugins ---------------------------

func (t *ThanosStack) UninstallPlugins(ctx context.Context, pluginNames []string, deployConfig *types.Config) error {
	logFileName := fmt.Sprintf("logs/uninstall_%s_%s_%s.log", t.network, t.stack, time.Now().Format("2006-01-02_15-04-05"))
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
			return t.uninstallBridge(ctx, deployConfig, logFileName)
		case constants.PluginBlockExplorer:
			return t.uninstallBlockExplorer(ctx, deployConfig, logFileName)
		}
	}
	return nil
}
