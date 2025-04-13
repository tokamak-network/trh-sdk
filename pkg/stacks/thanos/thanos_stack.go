package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"os"

	"github.com/tokamak-network/trh-sdk/abis"
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

type RegisterCandidateInput struct {
	rollupConfig      string
	amount            float64
	useTon            bool
	memo              string
	nameInfo          string
	seed              string
	safeWalletAddress string
}

func NewThanosStack(network string, stack string) *ThanosStack {
	return &ThanosStack{
		network: network,
		stack:   stack,
	}
}

// ----------------------------------------- Deploy contracts command  ----------------------------- //

func (t *ThanosStack) DeployContracts(ctx context.Context) error {
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet)
	}
	if t.network != constants.Testnet && t.network != constants.Mainnet {
		return fmt.Errorf("network %s does not support", t.network)
	}
	var err error

	// STEP 1. Input the parameters
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
	l1ChainID, err := l1Client.ChainID(ctx)
	if err != nil {
		fmt.Printf("Failed to get chain id: %s", err)
		return err
	}

	l2ChainID, err := utils.GenerateL2ChainId()
	if err != nil {
		fmt.Printf("Failed to generate L2ChainID: %s", err)
		return err
	}

	deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig.fraudProof, l1ChainID, l2ChainID)

	// Select operators Accounts
	operators, err := selectAccounts(ctx, l1Client, deployContractsConfig.fraudProof, deployContractsConfig.seed)
	if err != nil {
		return err
	}

	if len(operators) == 0 {
		return fmt.Errorf("no operators were found")
	}

	fmt.Print("The SDK is ready to deploy the contracts to the L1 environment. Do you want to proceed(Y/n)? ")
	confirmation, err := scanner.ScanBool(true)
	if err != nil {
		return err
	}
	if !confirmation {
		return nil
	}

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
	fmt.Println("Deploying the contracts...")

	gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		fmt.Printf("Failed to get gas price: %v\n", err)
	}

	envValues := fmt.Sprintf("export GS_ADMIN_PRIVATE_KEY=%s\nexport L1_RPC_URL=%s\n", operators[0].PrivateKey, deployContractsConfig.l1RPCurl)
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
	err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh deploy -e .env -c deploy-config.json")
	if err != nil {
		fmt.Print("\r❌ Contract deployment failed!       \n")
		return err
	}
	fmt.Print("\r✅ Contract deployment completed successfully!       \n")

	// STEP 5: Generate the genesis and rollup files
	err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh generate -e .env -c deploy-config.json")
	fmt.Println("Generating the rollup and genesis files...")
	if err != nil {
		fmt.Print("\r❌ Failed to generate rollup and genesis files!       \n")
		return err
	}
	fmt.Print("\r✅ Successfully generated rollup and genesis files!       \n")
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error obtaining current working directory:", err)
		return err
	}
	fmt.Printf("\r Genesis file path: %s/tokamak-thanos/build/genesis.json\n", cwd)
	fmt.Printf("\r Rollup file path: %s/tokamak-thanos/build/rollup.json\n", cwd)

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
		fmt.Println("Failed to write settings file:", err)
		return err
	}
	fmt.Printf("✅ Configuration successfully saved to: %s/settings.json", cwd)

	noCandidate := ctx.Value("no-candidate").(bool)

	// If --no-candidate flag is NOT provided, register the candidate
	if !noCandidate {
		fmt.Println("🔍 Verifying and registering candidate...")
		err := t.VerifyRegisterCandidates(ctx, true)
		if err != nil {
			return fmt.Errorf("candidate registration failed: %v", err)
		}
		fmt.Println("✅ Candidate registration completed successfully!")
	} else {
		fmt.Println("ℹ️ Skipping candidate registration (--no-candidate flag provided)")
	}

	return nil
}

// ----------------------------------------- Deploy command  ----------------------------- //

func (t *ThanosStack) Deploy(ctx context.Context, deployConfig *types.Config) error {
	switch t.network {
	case constants.LocalDevnet:
		err := t.deployLocalDevnet()
		if err != nil {
			fmt.Printf("Error deploying local devnet: %s", err)
			return t.destroyDevnet()
		}
	case constants.Testnet, constants.Mainnet:
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

	fmt.Println("⚡️Removing the previous deployment state...")
	err = t.clearTerraformState(ctx)
	if err != nil {
		fmt.Printf("Failed to clear the existing terraform state, err: %s", err.Error())
		return err
	}

	fmt.Println("✅ Removed the previous deployment state...")

	inputs, err := t.inputDeployInfra()
	if err != nil {
		fmt.Println("Error collecting infrastructure deployment parameters:", err)
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

	thanosStackValueFileExist := utils.CheckFileExists("tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml")
	if !thanosStackValueFileExist {
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	deployConfig.ChainName = inputs.ChainName

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

// --------------------------------------------- Register Candidates ---------------------------

// fromDeployContract flag would be true if the function would be called from the deploy contract function
func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context, fromDeployContract bool) error {
	var privateKeyString string
	config, err := utils.ReadConfigFromJSONFile()
	if err != nil || config == nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	l1Client, err := ethclient.DialContext(ctx, config.L1RPCURL)
	if err != nil {
		return err
	}
	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		fmt.Printf("Failed to get chain id: %s", err)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error determining current directory:", err)
		return err
	}

	file, err := os.Open(fmt.Sprintf("%s/%s", cwd, "11155111-deploy.json"))
	if err != nil {
		fmt.Println("Error opening deployment file:", err)
		return err
	}

	// Decode JSON
	var contracts types.Contracts
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&contracts); err != nil {
		fmt.Println("Error decoding deployment JSON file:", err)
		return err
	}

	registerCandidate, err := t.inputRegisterCandidate(fromDeployContract)
	if err != nil {
		return err
	}

	if fromDeployContract {
		privateKeyString = config.AdminPrivateKey
	} else {
		operatorsSelected, err := selectAccounts(ctx, l1Client, config.EnableFraudProof, registerCandidate.seed)
		if err != nil {
			return err
		}
		privateKeyString = operatorsSelected[0].PrivateKey
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyString, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	// Create transaction auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transaction auth: %v", err)
	}

	callOpts := &bind.CallOpts{
		Context: ctx,
	}

	// Get contract address from environment
	contractAddrStr := constants.L1ChainConfigurations[chainID.Uint64()].L1VerificationContractAddress
	if contractAddrStr == "" {
		return fmt.Errorf("L1_VERIFICATION_CONTRACT_ADDRESS not set in constant")
	}
	contractAddr := common.HexToAddress(contractAddrStr)

	// Create contract instance
	contract, err := abis.NewL1ContractVerification(contractAddr, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	systemConfigProxy := contracts.SystemConfigProxy
	if systemConfigProxy == "" {
		return fmt.Errorf("SystemConfigProxy is not set")
	}

	proxyAdmin := contracts.ProxyAdmin
	if proxyAdmin == "" {
		return fmt.Errorf("ProxyAdmin is not set")
	}

	l2TonAddress := constants.L1ChainConfigurations[chainID.Uint64()].L2TonAddress
	if l2TonAddress == "" {
		return fmt.Errorf("L2TonAddress variable is not set")
	}

	isVerificationPossible, err := contract.IsVerificationPossible(callOpts)
	// Verify and register config
	if isVerificationPossible {
		txVerifyAndRegisterConfig, err := contract.VerifyAndRegisterRollupConfig(
			auth,
			common.HexToAddress(systemConfigProxy),
			common.HexToAddress(proxyAdmin),
			2, //TODO: Need to check and update this using TON
			common.HexToAddress(l2TonAddress),
			registerCandidate.nameInfo,
			common.HexToAddress(registerCandidate.safeWalletAddress),
		)
		if err != nil {
			return fmt.Errorf("failed to register candidate: %v", err)
		}

		fmt.Printf("Verification and register config transaction submitted: %s\n", txVerifyAndRegisterConfig.Hash().Hex())

		// Wait for transaction confirmation
		receiptVerifyRegisterConfig, err := bind.WaitMined(ctx, l1Client, txVerifyAndRegisterConfig)
		if err != nil {
			return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
		}

		if receiptVerifyRegisterConfig.Status != 1 {
			return fmt.Errorf("transaction failed with status: %d", receiptVerifyRegisterConfig.Status)
		}

		fmt.Printf("Transaction confirmed in block %d\n", receiptVerifyRegisterConfig.BlockNumber.Uint64())
	} else {
		contractAddrStrBridgeRegistry := constants.L1ChainConfigurations[chainID.Uint64()].L1BridgeRegistry
		if contractAddrStrBridgeRegistry == "" {
			return fmt.Errorf("L1BridgeRegistry variable not set in constant")
		}
		contractAddressBridgeRegistry := common.HexToAddress(contractAddrStrBridgeRegistry)

		// Create contract instance
		bridgeRegistryContract, err := abis.NewL1BridgeRegistry(contractAddressBridgeRegistry, l1Client)
		if err != nil {
			return fmt.Errorf("failed to create contract instance: %v", err)
		}

		txRegisterConfig, err := bridgeRegistryContract.RegisterRollupConfig(auth, common.HexToAddress(systemConfigProxy), 2, common.HexToAddress(l2TonAddress),
			registerCandidate.nameInfo)

		if err != nil {
			return fmt.Errorf("failed to register candidate: %v", err)
		}

		fmt.Printf("Register config transaction submitted: %s\n", txRegisterConfig.Hash().Hex())

		// Wait for transaction confirmation
		receiptRegisterConfig, err := bind.WaitMined(ctx, l1Client, txRegisterConfig)
		if err != nil {
			return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
		}

		if receiptRegisterConfig.Status != 1 {
			return fmt.Errorf("transaction failed with status: %d", receiptRegisterConfig.Status)
		}

		fmt.Printf("Transaction confirmed in block %d\n", receiptRegisterConfig.BlockNumber.Uint64())
	}

	// Convert amount to Wei
	amountInWei := new(big.Float).Mul(big.NewFloat(registerCandidate.amount), big.NewFloat(1e18))
	amountBigInt, _ := amountInWei.Int(nil)

	// Get contract address from environment
	l2ManagerAddressStr := constants.L1ChainConfigurations[chainID.Uint64()].L2ManagerAddress
	if l2ManagerAddressStr == "" {
		return fmt.Errorf("L2_MANAGER_ADDRESS variable is not set")
	}
	l2ManagerAddress := common.HexToAddress(l2ManagerAddressStr)

	// Create contract instance
	l2ManagerContract, err := abis.NewLayer2ManagerV1(l2ManagerAddress, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Call registerCandidateAddOn
	txRegisterCandidate, err := l2ManagerContract.RegisterCandidateAddOn(
		auth,
		common.HexToAddress(registerCandidate.rollupConfig),
		amountBigInt,
		registerCandidate.useTon,
		registerCandidate.memo,
	)
	if err != nil {
		return fmt.Errorf("failed to register candidate: %v", err)
	}

	fmt.Printf("Register Candidate transaction submitted: %s\n", txRegisterCandidate.Hash().Hex())

	// Wait for transaction confirmation
	receiptRegisterCandidate, err := bind.WaitMined(ctx, l1Client, txRegisterCandidate)
	if err != nil {
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	if receiptRegisterCandidate.Status != 1 {
		return fmt.Errorf("transaction failed with status: %d", receiptRegisterCandidate.Status)
	}

	fmt.Printf("Transaction confirmed in block %d\n", receiptRegisterCandidate.BlockNumber.Uint64())

	return nil
}
