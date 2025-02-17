package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"os"
)

type ThanosStack struct {
	network             string
	stack               string
	defaultDeployConfig *types.DeployConfigTemplate
	l1Client            *ethclient.Client
	deployConfig        *types.Config
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

func NewThanosStack(network string, stack string) *ThanosStack {
	return &ThanosStack{
		network: network,
		stack:   stack,
	}
}

func (t *ThanosStack) Deploy(deployConfig *types.Config) error {
	switch t.network {
	case constants.LocalDevnet:
		return t.deployLocalDevnet()
	case constants.Testnet, constants.Mainnet:
		fmt.Print("Please choose your infrastructure [AWS] (default AWS): ")
		input, err := scanner.ScanString()
		if err != nil {
			fmt.Printf("Error scanning L1 RPC URL: %s", err)
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
			return fmt.Errorf("%s not supported", infraOpt)
		}
	default:
		return fmt.Errorf("network %s is not supported", t.network)
	}
}

func (t *ThanosStack) DeployContracts() error {
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s doesn't need to deploy the contracts, please running `tokamak-cli-sdk deploy` instead", constants.LocalDevnet)
	}
	var err error
	// STEP 1. Input the parameters
	deployContractsConfig, err := t.inputDeployContracts()
	if err != nil {
		return err
	}

	l1Client, err := ethclient.DialContext(context.Background(), deployContractsConfig.l1RPCurl)
	if err != nil {
		return err
	}

	deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig.falutProof, t.network)

	// Select operators Accounts
	operators, err := selectAccounts(l1Client, deployContractsConfig.falutProof, deployContractsConfig.seed)
	if err != nil {
		return err
	}

	if len(operators) == 0 {
		return fmt.Errorf("no operator found")
	}

	for k, v := range operators {
		fmt.Printf("%d account: %d, address: %s\n", k, types.Operator(v.Index), v.Address)
	}

	err = makeDeployContractConfigJsonFile(l1Client, operators, deployContractsTemplate)
	if err != nil {
		return err
	}

	// STEP 2. Clone the repository
	err = t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
	if err != nil {
		return err
	}

	// STEP 3. Build the contracts
	doneCh := make(chan bool)
	go utils.ShowLoadingAnimation(doneCh, "Building the contracts...")
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh build")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Build the contracts failed!       \n")
		return err
	}
	doneCh <- true
	fmt.Print("\r✅ Build the contracts completed!       \n")

	// STEP 4. Deploy the contracts
	go utils.ShowLoadingAnimation(doneCh, "Deploying the contracts...")
	// STEP 4.1. Generate the .env file
	_, err = utils.ExecuteCommand("bash", "-c", fmt.Sprintf("cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && echo 'export GS_ADMIN_PRIVATE_KEY=%s' > .env && echo 'export L1_RPC_URL=%s' >> .env", operators[0].PrivateKey, deployContractsConfig.l1RPCurl))
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Make .env file failed!       \n")
		return err
	}

	// STEP 4.2. Copy the config file into the scripts folder
	_, err = utils.ExecuteCommand("bash", "-c", "cp ./deploy-config.json tokamak-thanos/packages/tokamak/contracts-bedrock/scripts")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Copy the config file successfully!       \n")
		return err
	}

	// STEP 4.3. Deploy contracts
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh deploy -e .env -c deploy-config.json")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Build the contracts failed!       \n")
		return err
	}
	fmt.Print("\r✅ Deploy the contracts completed!       \n")
	doneCh <- true

	// STEP 5: Generate the genesis and rollup files
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh generate -e .env -c deploy-config.json")
	go utils.ShowLoadingAnimation(doneCh, "Generating the rollup and genesis files...")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Generate the rollup and genesis files!       \n")
		return err
	}
	doneCh <- true
	fmt.Print("\r✅ Generated the rollup and genesis files!       \n")
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return err
	}
	fmt.Printf("\r Genesis file located at: %s/tokamak-thanos/build/genesis.json\n", cwd)
	fmt.Printf("\r Rollup file located at: %s/tokamak-thanos/build/rollup.json\n", cwd)

	var challengerPrivateKey string
	if deployContractsConfig.falutProof {
		if operators[4] == nil {
			return fmt.Errorf("no challenger operator found")
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
		L1RPCURL:             deployContractsConfig.l1RPCurl,
		Stack:                t.stack,
		Network:              t.network,
		EnableFraudProof:     deployContractsConfig.falutProof,
	}
	err = cfg.WriteToJSONFile("settings.json")
	if err != nil {
		fmt.Println("Error writing settings file:", err)
		return err
	}
	fmt.Printf("✅ The configuration has been saved in: %s/settings.json", cwd)
	return nil
}

func (t *ThanosStack) deployLocalDevnet() error {
	err := t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
	if err != nil {
		return err
	}

	doneCh := make(chan bool)
	go utils.ShowLoadingAnimation(doneCh, "Installing the devnet packages...")
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && bash ./install-devnet-packages.sh")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Installation failed!       \n")
		return err
	}
	fmt.Print("\r✅ Installation completed!       \n")
	doneCh <- true

	go utils.ShowLoadingAnimation(doneCh, "Making the devnet up...")
	output, err := utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && make devnet-up")
	if err != nil {
		doneCh <- true
		fmt.Printf("\r❌ Make devnet failed!       \n Detail: %s", output)

		return err
	}

	fmt.Print("\r✅ Devnet up!       \n")

	return nil
}

func (t *ThanosStack) deployNetworkToAWS(deployConfig *types.Config) error {
	// STEP 1. Check the required dependencies
	if !dependencies.CheckTerraformInstallation() {
		return fmt.Errorf("terraform is not available")
	}

	if !dependencies.CheckHelmInstallation() {
		return fmt.Errorf("helm is not available")
	}

	if !dependencies.CheckAwsCLIInstallation() {
		return fmt.Errorf("aws is not available")
	}

	if !dependencies.CheckK8sInstallation() {
		return fmt.Errorf("kubectl is not available")
	}

	// STEP 1. Clone the charts repository
	err := t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		fmt.Println("Error cloning sourcecode:", err)
		return err
	}

	// STEP 2. Login AWS
	awsLoginInputs, err := t.inputAWSLogin()
	if err != nil {
		fmt.Println("Error getting AWS login inputs:", err)
		return err
	}

	awsProfile, err := loginAWS(awsLoginInputs)
	if err != nil {
		fmt.Println("Error getting AWS profile:", err)
		return err
	}
	fmt.Println("AWS Profile:", awsProfile)

	inputs, err := t.inputDeployInfra()
	if err != nil {
		fmt.Println("Error getting deploy infrastructure inputs:", err)
		return err
	}

	// STEP 3. Make .envrc file
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
	})
	if err != nil {
		fmt.Println("Error creating Terraform environment:", err)
		return err
	}
	// STEP 4. Make terraform backend up
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
		fmt.Println("Error running the terraform backend up:", err)
		return err
	}

	// STEP 5. copy rollup and genesis files
	err = utils.CopyFile("tokamak-thanos/build/rollup.json", "tokamak-thanos-stack/terraform/thanos-stack/config-files/rollup.json")
	if err != nil {
		fmt.Println("Error copying rollup file:", err)
		return err
	}
	err = utils.CopyFile("tokamak-thanos/build/genesis.json", "tokamak-thanos-stack/terraform/thanos-stack/config-files/genesis.json")
	if err != nil {
		fmt.Println("Error copying genesis file:", err)
		return err
	}

	fmt.Println("Make thanos stack terraform up")
	// STEP 6. Make terraform thanos-stack up
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
		fmt.Println("Error running thanos-stack terraform:", err)
		return err
	}

	thanosStackValueFileExist := utils.CheckFileExists("tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml")
	if !thanosStackValueFileExist {
		return fmt.Errorf("thanos-stack-values.yaml not found")
	}

	namespace := inputs.ChainName

	// Step 7. Interact with EKS
	eksSetup, err := utils.ExecuteCommand("aws", []string{
		"eks",
		"update-kubeconfig",
		"--region", awsLoginInputs.Region,
		"--name", namespace,
	}...)
	if err != nil {
		fmt.Println("Error running eks update-kubeconfig:", err, "details:", eksSetup)
		return err
	}

	fmt.Println("eks update-kubeconfig:", eksSetup)

	k8sPods, err := utils.GetK8sPods(namespace)
	if err != nil {
		fmt.Println("Error getting k8s pods:", err, "details:", k8sPods)
		return err
	}
	fmt.Println("kubectl get pods: \n", k8sPods)

	// ---------------------------------------- Deploy chain --------------------------//
	// Step 8. Add helm chart
	helmAddOuput, err := utils.ExecuteCommand("helm", []string{
		"repo",
		"add",
		"thanos-stack",
		"https://tokamak-network.github.io/tokamak-thanos-stack",
	}...)
	if err != nil {
		fmt.Println("Error running helm add:", err, "details:", helmAddOuput)
		return err
	}

	// Step 8.1 Search helm charts
	helmSearchOutput, err := utils.ExecuteCommand("helm", []string{
		"search",
		"repo",
		"thanos-stack",
	}...)
	if err != nil {
		fmt.Println("Error running helm search:", err, "details:", helmSearchOutput)
		return err
	}
	fmt.Println("Helm added successfully: \n", helmSearchOutput)

	// Step 8.2. Install helm charts
	var helmReleaseNameInput string
	for {
		fmt.Print("Please enter the Helm chart release name: ")
		helmReleaseNameInput, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning Helm chart release name:", err)
			return err
		}

		if helmReleaseNameInput == "" {
			fmt.Println("Error: Release name cannot be empty. Please try again.")
			continue
		}

		releaseNameExist, err := utils.HelmReleaseExists(namespace, helmReleaseNameInput)
		if err != nil {
			fmt.Println("Error checking if Helm chart release exists:", helmReleaseNameInput)
			return err
		}
		if releaseNameExist {
			fmt.Println("Error: Helm release name already exists. Please choose a different name.")
			continue
		}

		break
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	_, err = utils.ExecuteCommand("helm", []string{
		"install",
		helmReleaseNameInput,
		"thanos-stack/thanos-stack",
		"--values",
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", cwd),
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error running helm search:", err, "details:", helmSearchOutput)
		return err
	}

	fmt.Println("✅ Helm charts installed successfully")
	k8sPods, err = utils.GetK8sPods(namespace)
	if err != nil {
		fmt.Println("Error getting k8s pods:", err, "details:", k8sPods)
		return err
	}
	fmt.Println("Pods installed: \n", k8sPods)

	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, helmReleaseNameInput)
		if err != nil {
			fmt.Println("Error getting k8s ingresses:", err, "details:", k8sIngresses)
			return err
		}

		if len(k8sIngresses) > 0 {
			fmt.Printf("Your chain deployed successfully: %s", k8sIngresses[0])
			return nil
		}

		time.Sleep(15 * time.Second)
	}
}

func (t *ThanosStack) cloneSourcecode(repositoryName, url string) error {
	doneCh := make(chan bool)
	defer close(doneCh)
	existingSourcecode, err := utils.CheckExistingSourceCode(repositoryName)
	if err != nil {
		fmt.Println("Error checking existing source code")
		return err
	}

	if !existingSourcecode {
		go utils.ShowLoadingAnimation(doneCh, fmt.Sprintf("Cloning the %s repository...", repositoryName))
		err := utils.CloneRepo(url, repositoryName)
		doneCh <- true
		if err != nil {
			fmt.Println("Error cloning the repo")
			return err
		}
	}
	fmt.Printf("\r✅ Clone the %s repository successfully!       \n", repositoryName)

	return nil
}

func (t *ThanosStack) inputDeployContracts() (*DeployContractsInput, error) {
	fmt.Println("You are deploying the L1 contracts.")

	fmt.Print("Please input your L1 RPC URL: ")
	l1RPCUrl, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning L1 RPC URL: %s", err)
		return nil, err
	}

	fmt.Print("Please input your L1 provider: ")
	l1Provider, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning L1 provider: %s", err)
		return nil, err
	}

	fmt.Print("Please input your admin seed phrase: ")
	seed, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning the seed phrase: %s", err)
		return nil, err
	}

	faultProof := false
	fmt.Print("Do you want to enable the fault-proof system on your chain? [Y or N] (default: N): ")
	faultProof, err = scanner.ScanBool()
	if err != nil {
		fmt.Printf("Error scanning the fault-proof system setting: %s", err)
		return nil, err
	}

	return &DeployContractsInput{
		l1RPCurl:   l1RPCUrl,
		l1Provider: l1Provider,
		seed:       seed,
		falutProof: faultProof,
	}, nil
}

func (t *ThanosStack) inputAWSLogin() (*types.AWSLogin, error) {
	fmt.Print("Please enter the AWS access key(read more): ")
	awsAccessKeyID, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning AWS access key")
		return nil, err
	}

	if awsAccessKeyID == "" {
		return nil, fmt.Errorf("AWS access key ID cannot be empty")
	}

	fmt.Print("Please enter the AWS secret key(read more): ")
	awsSecretKey, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning AWS secret key")
		return nil, err
	}
	if awsSecretKey == "" {
		return nil, fmt.Errorf("AWS secret key cannot be empty")
	}

	fmt.Print("Please enter the AWS region(default ap-northeast-2): ")
	awsRegion, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning AWS region")
		return nil, err
	}
	if awsRegion == "" {
		awsRegion = "ap-northeast-2"
	}

	fmt.Print("Please enter the format file(default Json): ")
	defaultFormatFile, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning AWS format file")
		return nil, err
	}
	if defaultFormatFile == "" {
		defaultFormatFile = "json"
	}

	return &types.AWSLogin{
		SecretKey:     awsSecretKey,
		Region:        awsRegion,
		AccessKey:     awsAccessKeyID,
		DefaultFormat: defaultFormatFile,
	}, nil
}

func (t *ThanosStack) inputDeployInfra() (*DeployInfraInput, error) {
	fmt.Print("Please input your chain name: ")
	chainName, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning chain name: %s", err)
		return nil, err
	}

	fmt.Print("Please input your L1 beacon URL(read more): ")
	l1BeaconUrl, err := scanner.ScanString()
	if err != nil {
		return nil, err
	}

	return &DeployInfraInput{
		ChainName:   chainName,
		L1BeaconURL: l1BeaconUrl,
	}, nil
}
