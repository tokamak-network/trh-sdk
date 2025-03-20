package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

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
	network      string
	stack        string
	deployConfig *types.Config
}

type DeployContractsInput struct {
	l1Provider string
	l1RPCurl   string
	seed       string
	falutProof bool
}

type DeployInfraInput struct {
	ChainName   string
	L1BeaconURL string
}

type RegisterCandidateInput struct {
	rollupConfig string
	amount       float64
	useTon       bool
	memo         string
	nameInfo     string
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

	l1Client, err := ethclient.DialContext(ctx, deployContractsConfig.l1RPCurl)
	if err != nil {
		return err
	}
	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		fmt.Printf("Failed to get chain id: %s", err)
		return err
	}

	deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig.falutProof, chainID)

	// Select operators Accounts
	operators, err := selectAccounts(ctx, l1Client, deployContractsConfig.falutProof, deployContractsConfig.seed)
	if err != nil {
		return err
	}

	if len(operators) == 0 {
		return fmt.Errorf("no operators were found")
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
	if deployContractsConfig.falutProof {
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
		L1ChainID:            chainID.Uint64(),
		L1RPCURL:             deployContractsConfig.l1RPCurl,
		Stack:                t.stack,
		Network:              t.network,
		EnableFraudProof:     deployContractsConfig.falutProof,
	}
	err = cfg.WriteToJSONFile()
	if err != nil {
		fmt.Println("Failed to write settings file:", err)
		return err
	}
	fmt.Printf("✅ Configuration successfully saved to: %s/settings.json", cwd)

	verifyAndRegister, err := t.inputVerifyAndRegister()
	if err != nil {
		return err
	}

	if verifyAndRegister {
		err = t.VerifyRegisterCandidates(ctx)
		if err != nil {
			fmt.Println("Failed register candidate:", err)
			return err
		}
	}

	return nil
}

// ----------------------------------------- Deploy command  ----------------------------- //

func (t *ThanosStack) Deploy(ctx context.Context, deployConfig *types.Config) error {
	switch t.network {
	case constants.LocalDevnet:
		return t.deployLocalDevnet()
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
			return t.deployNetworkToAWS(deployConfig)
		default:
			return fmt.Errorf("Infrastructure provider %s is not supported", infraOpt)
		}
	default:
		return fmt.Errorf("Network %s is not supported", t.network)
	}
}

func (t *ThanosStack) deployLocalDevnet() error {
	err := t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
	if err != nil {
		return err
	}

	err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos && bash ./install-devnet-packages.sh")
	if err != nil {
		fmt.Print("\r❌ Package installation failed!       \n")
		return err
	}
	fmt.Print("\r✅ Package installation completed successfully!       \n")

	err = utils.ExecuteCommandStream("bash", "-c", "cd tokamak-thanos && export DEVNET_L2OO=true && make devnet-up")
	if err != nil {
		fmt.Print("\r❌ Failed to start devnet!       \n")
		return err
	}

	fmt.Print("\r✅ Devnet started successfully!       \n")

	return nil
}

func (t *ThanosStack) deployNetworkToAWS(deployConfig *types.Config) error {
	// STEP 1. Verify required dependencies
	if !dependencies.CheckTerraformInstallation() {
		return fmt.Errorf("terraform is not installed")
	}

	if !dependencies.CheckHelmInstallation() {
		return fmt.Errorf("helm is not installed")
	}

	if !dependencies.CheckAwsCLIInstallation() {
		return fmt.Errorf("AWS CLI is not installed")
	}

	if !dependencies.CheckK8sInstallation() {
		return fmt.Errorf("kubectl is not installed")
	}

	// STEP 1. Clone the charts repository
	err := t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return err
	}

	// STEP 2. AWS Authentication
	awsLoginInputs, err := t.inputAWSLogin()
	if err != nil {
		fmt.Println("Error collecting AWS credentials:", err)
		return err
	}

	awsProfile, err := loginAWS(awsLoginInputs)
	if err != nil {
		fmt.Println("Error authenticating with AWS:", err)
		return err
	}
	fmt.Println("Successfully authenticated with AWS Profile:", awsProfile)
	deployConfig.AWS = awsLoginInputs

	inputs, err := t.inputDeployInfra()
	if err != nil {
		fmt.Println("Error collecting infrastructure deployment parameters:", err)
		return err
	}

	// STEP 3. Create .envrc file
	err = makeTerraformEnvFile("tokamak-thanos-stack/terraform", types.TerraformEnvConfig{
		ThanosStackName:  inputs.ChainName,
		AwsRegion:        awsLoginInputs.Region,
		SequencerKey:     deployConfig.SequencerPrivateKey,
		BatcherKey:       deployConfig.BatcherPrivateKey,
		ProposerKey:      deployConfig.ProposerPrivateKey,
		ChallengerKey:    deployConfig.ChallengerPrivateKey,
		EksClusterAdmins: awsProfile.Arn,
		DeploymentsPath:  deployConfig.DeploymentPath,
		L1BeaconUrl:      inputs.L1BeaconURL,
		L1RpcUrl:         deployConfig.L1RPCURL,
		L1RpcProvider:    deployConfig.L1RPCProvider,
		Azs:              awsProfile.AvailabilityZones,
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

	thanosStackValueFileExist := utils.CheckFileExists("tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml")
	if !thanosStackValueFileExist {
		return fmt.Errorf("Configuration file thanos-stack-values.yaml not found")
	}

	namespace := types.ConvertChainNameToNamespace(inputs.ChainName)
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

	k8sPods, err := utils.GetK8sPods(namespace)
	if err != nil {
		fmt.Println("Error retrieving Kubernetes pods:", err, "details:", k8sPods)
		return err
	}
	fmt.Println("Current Kubernetes pods: \n", k8sPods)

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
		"thanos-stack/thanos-stack",
		"--values",
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", cwd),
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing Helm charts:", err, "details:", helmSearchOutput)
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
	fmt.Printf("Network deployment completed successfully. RPC endpoint: %s", l2RPCUrl)

	deployConfig.HelmReleaseName = helmReleaseName
	deployConfig.K8sNamespace = namespace
	deployConfig.L2RpcUrl = l2RPCUrl

	err = deployConfig.WriteToJSONFile()
	if err != nil {
		fmt.Println("Error saving configuration file:", err)
		return err
	}
	fmt.Printf("Configuration saved successfully to: %s/settings.json", cwd)

	// After installing the infra successfully, we install the bridge
	err = t.installBridge(deployConfig)
	if err != nil {
		fmt.Println("Error installing bridge:", err)
	}

	return nil
}

// --------------------------------------------- Destroy command -------------------------------------//

func (t *ThanosStack) Destroy(deployConfig *types.Config) error {
	switch t.network {
	case constants.LocalDevnet:
		return t.destroyDevnet()
	case constants.Testnet, constants.Mainnet:
		return t.destroyInfraOnAWS(deployConfig)
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

func (t *ThanosStack) destroyInfraOnAWS(deployConfig *types.Config) error {
	fmt.Printf("Removing Helm release: %s from namespace: %s...\n", deployConfig.HelmReleaseName, deployConfig.K8sNamespace)
	// login aws again because the session when logging in will be expired after a few time.
	_, err := loginAWS(t.deployConfig.AWS)
	if err != nil {
		fmt.Println("Error getting AWS profile:", err)
		return err
	}

	fmt.Printf("Uninstalling Helm release: %s in namespace: %s...\n", deployConfig.HelmReleaseName, deployConfig.K8sNamespace)

	output, err := utils.ExecuteCommand("helm", "uninstall", deployConfig.HelmReleaseName, "--namespace", deployConfig.K8sNamespace)
	if err != nil {
		fmt.Println("Error removing Helm release:", err, "details:", output)
		return err
	}

	fmt.Println("Helm release removed successfully:", output)

	err = utils.ExecuteCommandStream("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform destroy`,
	}...)
	if err != nil {
		fmt.Println("Error running thanos-stack terraform destroy:", err)
		return err
	}
	fmt.Println("Thanos stack terraform destroyed successfully.")

	err = utils.ExecuteCommandStream("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd backend &&
		terraform destroy`,
	}...)
	if err != nil {
		fmt.Println("Error running the terraform backend destroy:", err)
		return err
	}
	fmt.Println("Backend terraform destroyed successfully.")

	return nil
}

// ------------------------------------------ Install plugins ---------------------------

func (t *ThanosStack) InstallPlugins(pluginNames []string, deployConfig *types.Config) error {
	for _, pluginName := range pluginNames {
		if !constants.SupportedPlugins[pluginName] {
			fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
			continue
		}

		fmt.Printf("Installing plugin: %s in namespace: %s...\n", pluginName, deployConfig.K8sNamespace)

		switch pluginName {

		}
	}
	fmt.Println(pluginNames)
	return nil
}

// --------------------------------------------- Register Candidates ---------------------------

func (t *ThanosStack) VerifyRegisterCandidates(ctx context.Context) error {
	registerCandidate, err := t.inputRegisterCandidate()
	if err != nil {
		return err
	}

	// Get RPC URL from environment
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		return fmt.Errorf("RPC_URL environment variable is not set")
	}

	// Connect to Ethereum client
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to the Ethereum client: %v", err)
	}

	// Get private key from environment
	privateKeyString := os.Getenv("PRIVATE_KEY")
	if privateKeyString == "" {
		return fmt.Errorf("PRIVATE_KEY environment variable is not set")
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyString, "0x"))
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	// Get chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Create transaction auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transaction auth: %v", err)
	}

	// Get contract address from environment
	contractAddrStr := os.Getenv("L1_VERIFICATION_CONTRACT_ADDRESS")
	if contractAddrStr == "" {
		return fmt.Errorf("L1_VERIFICATION_CONTRACT_ADDRESS environment variable is not set")
	}
	contractAddr := common.HexToAddress(contractAddrStr)

	// Create contract instance
	contract, err := abis.NewL1ContractVerification(contractAddr, client)
	if err != nil {
		return fmt.Errorf("failed to create contract instance: %v", err)
	}

	//TODO: Need to check and update these functionality to get system config.

	systemConfigProxy := os.Getenv("SYSTEM_CONFIG_ADDRESS")
	if systemConfigProxy == "" {
		return fmt.Errorf("SYSTEM_CONFIG_ADDRESS environment variable is not set")
	}

	l2TonAddress := os.Getenv("L2_TON_ADDRESS")
	if l2TonAddress == "" {
		return fmt.Errorf("L2_TON_ADDRESS environment variable is not set")
	}

	// Verify and register config
	txVerifyAndRegisterConfig, err := contract.VerifyAndRegisterRollupConfig(
		auth,
		common.HexToAddress(systemConfigProxy),
		common.HexToAddress("0x33E6F5aa5A4cf5d0D2Cb68e43b15976D0E0234b1"), //TODO: Update this to fetch the proxy admin address from the deployed ones
		2, //TODO: Need to check and update this using TON
		common.HexToAddress(l2TonAddress),
		registerCandidate.nameInfo,
	)
	if err != nil {
		return fmt.Errorf("failed to register candidate: %v", err)
	}

	fmt.Printf("Transaction submitted: %s\n", txVerifyAndRegisterConfig.Hash().Hex())

	// Wait for transaction confirmation
	receiptVerifyRegisterConfig, err := bind.WaitMined(ctx, client, txVerifyAndRegisterConfig)
	if err != nil {
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	if receiptVerifyRegisterConfig.Status != 1 {
		return fmt.Errorf("transaction failed with status: %d", receiptVerifyRegisterConfig.Status)
	}

	fmt.Printf("Transaction confirmed in block %d\n", receiptVerifyRegisterConfig.BlockNumber.Uint64())

	// Convert amount to Wei
	amountInWei := new(big.Float).Mul(big.NewFloat(registerCandidate.amount), big.NewFloat(1e18))
	amountBigInt, _ := amountInWei.Int(nil)

	// Get contract address from environment
	l2ManagerAddressStr := os.Getenv("L2_MANAGER_ADDRESS")
	if l2ManagerAddressStr == "" {
		return fmt.Errorf("L2_MANAGER_ADDRESS environment variable is not set")
	}
	l2ManagerAddress := common.HexToAddress(l2ManagerAddressStr)

	// Create contract instance
	l2ManagerContract, err := abis.NewLayer2ManagerV1(l2ManagerAddress, client)
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

	fmt.Printf("Transaction submitted: %s\n", txRegisterCandidate.Hash().Hex())

	// Wait for transaction confirmation
	receiptRegisterCandidate, err := bind.WaitMined(ctx, client, txRegisterCandidate)
	if err != nil {
		return fmt.Errorf("failed waiting for transaction confirmation: %v", err)
	}

	if receiptRegisterCandidate.Status != 1 {
		return fmt.Errorf("transaction failed with status: %d", receiptRegisterCandidate.Status)
	}

	fmt.Printf("Transaction confirmed in block %d\n", receiptRegisterCandidate.BlockNumber.Uint64())

	return nil
}
