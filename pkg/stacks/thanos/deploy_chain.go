package thanos

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ----------------------------------------- Deploy command  ----------------------------- //

func (t *ThanosStack) Deploy(ctx context.Context, infraOpt string) error {
	switch t.network {
	case constants.LocalDevnet:
		err := t.deployLocalDevnet()
		if err != nil {
			fmt.Printf("Error deploying local devnet: %s", err)
			return t.destroyDevnet()
		}
	case constants.Testnet, constants.Mainnet:
		// Check L1 RPC URL
		if t.deployConfig.L1RPCURL == "" {
			return fmt.Errorf("L1 RPC URL is not set. Please run the deploy-contracts command first")
		}

		var (
			blockNo uint64
			err     error
		)

		ctxTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		client, err := ethclient.DialContext(ctxTimeout, t.deployConfig.L1RPCURL)
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
			l1RPC, l1RPCKind, l1ChainId, err := t.inputL1RPC(ctx)
			if err != nil {
				fmt.Printf("Error while getting L1 RPC URL: %s", err)
				return err
			}

			t.deployConfig.L1RPCURL = l1RPC
			t.deployConfig.L1RPCProvider = l1RPCKind
			t.deployConfig.L1ChainID = l1ChainId

			err = t.deployConfig.WriteToJSONFile()
			if err != nil {
				fmt.Println("Failed to write settings file after getting L1 RPC", err)
				return err
			}
		}

		switch infraOpt {
		case constants.AWS:
			err = t.deployNetworkToAWS(ctx)
			if err != nil {
				return t.destroyInfraOnAWS(ctx)
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
	err := t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
	}
	deploymentPath := fmt.Sprintf("%s/%s", cwd, t.deploymentPath)

	// Start the devnet
	fmt.Println("Starting the devnet...")

	err = utils.ExecuteCommandStream(t.l, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos && export DEVNET_L2OO=true && make devnet-up", deploymentPath))
	if err != nil {
		fmt.Print("\r❌ Failed to start devnet!       \n")
		return err
	}

	fmt.Print("\r✅ Devnet started successfully!       \n")

	return nil
}

func (t *ThanosStack) deployNetworkToAWS(ctx context.Context) error {
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
	if t.awsConfig == nil {
		return fmt.Errorf("AWS configuration is not set")
	}
	awsAccountProfile := t.awsConfig.AccountProfile
	awsLoginInputs := t.awsConfig.AwsConfig

	t.deployConfig.AWS = awsLoginInputs
	if err := t.deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

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

	var (
		chainConfiguration = t.deployConfig.ChainConfiguration
	)

	if chainConfiguration == nil {
		return fmt.Errorf("chain configuration is not set")
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
	}
	deploymentPath := fmt.Sprintf("%s/%s", cwd, t.deploymentPath)

	// STEP 3. Create .envrc file
	err = makeTerraformEnvFile(fmt.Sprintf("%s/tokamak-thanos-stack/terraform", deploymentPath), types.TerraformEnvConfig{
		ThanosStackName:     inputs.ChainName,
		AwsRegion:           awsLoginInputs.Region,
		SequencerKey:        t.deployConfig.SequencerPrivateKey,
		BatcherKey:          t.deployConfig.BatcherPrivateKey,
		ProposerKey:         t.deployConfig.ProposerPrivateKey,
		ChallengerKey:       t.deployConfig.ChallengerPrivateKey,
		EksClusterAdmins:    awsAccountProfile.Arn,
		DeploymentsPath:     t.deployConfig.DeploymentPath,
		L1BeaconUrl:         inputs.L1BeaconURL,
		L1RpcUrl:            t.deployConfig.L1RPCURL,
		L1RpcProvider:       t.deployConfig.L1RPCProvider,
		Azs:                 awsAccountProfile.AvailabilityZones,
		ThanosStackImageTag: constants.DockerImageTag[t.deployConfig.Network].ThanosStackImageTag,
		OpGethImageTag:      constants.DockerImageTag[t.deployConfig.Network].OpGethImageTag,
		MaxChannelDuration:  chainConfiguration.GetMaxChannelDuration(),
	})
	if err != nil {
		fmt.Println("Error generating Terraform environment configuration:", err)
		return err
	}

	// STEP 4. Initialize Terraform backend
	err = utils.ExecuteCommandStream(t.l, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd backend &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve
		`, deploymentPath),
	}...)
	if err != nil {
		fmt.Println("Error initializing Terraform backend:", err)
		return err
	}

	// STEP 5. Copy configuration files
	err = utils.CopyFile(
		fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", deploymentPath),
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files/rollup.json", deploymentPath),
	)
	if err != nil {
		fmt.Println("Error copying rollup configuration:", err)
		return err
	}

	err = utils.CopyFile(
		fmt.Sprintf("%s/tokamak-thanos/build/genesis.json", deploymentPath),
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files/genesis.json", deploymentPath),
	)
	if err != nil {
		fmt.Println("Error copying genesis configuration:", err)
		return err
	}

	fmt.Println("Deploying Thanos stack infrastructure")
	// STEP 6. Deploy Thanos stack infrastructure
	err = utils.ExecuteCommandStream(t.l, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve`, deploymentPath),
	}...)
	if err != nil {
		fmt.Println("Error deploying Thanos stack infrastructure:", err)
		return err
	}

	// Get VPC ID
	vpcIdOutput, err := utils.ExecuteCommand("bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd thanos-stack &&
		terraform output -json vpc_id`, deploymentPath),
	}...)
	if err != nil {
		return fmt.Errorf("failed to get terraform output for %s: %w", "vpc_id", err)
	}

	t.deployConfig.AWS.VpcID = strings.Trim(vpcIdOutput, `"`)
	if err := t.deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	thanosStackValueFileExist := utils.CheckFileExists(fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", deploymentPath))
	if !thanosStackValueFileExist {
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	t.deployConfig.ChainName = inputs.ChainName
	if err := t.deployConfig.WriteToJSONFile(); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Sleep for 30 seconds to allow the infrastructure to be fully deployed
	time.Sleep(30 * time.Second)

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
	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())
	chartFile := fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", cwd)
	valueFile := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", cwd)

	// Install the PVC first
	err = utils.UpdateYAMLField(valueFile, "enable_vpc", true)
	if err != nil {
		fmt.Println("Error updating `enable_vpc` configuration:", err)
		return err
	}
	err = utils.InstallHelmRelease(helmReleaseName, chartFile, valueFile, namespace)
	if err != nil {
		fmt.Println("Error installing Helm charts:", err)
		return err
	}

	fmt.Println("Wait for the VPCs to be created...")
	err = utils.WaitPVCReady(namespace)
	if err != nil {
		fmt.Println("Error waiting for PVC to be ready:", err)
		return err
	}

	// Install the rest of the charts
	err = utils.UpdateYAMLField(valueFile, "enable_deployment", true)
	if err != nil {
		fmt.Println("Error updating `enable_deployment` configuration:", err)
	}

	err = utils.InstallHelmRelease(helmReleaseName, chartFile, valueFile, namespace)
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

	t.deployConfig.K8s = &types.K8sConfig{
		Namespace: namespace,
	}
	t.deployConfig.L2RpcUrl = l2RPCUrl
	t.deployConfig.L1BeaconURL = inputs.L1BeaconURL

	err = t.deployConfig.WriteToJSONFile()
	if err != nil {
		fmt.Println("Error saving configuration file:", err)
		return err
	}
	fmt.Printf("Configuration saved successfully to: %s/settings.json \n", cwd)

	// After installing the infra successfully, we install the bridge
	err = t.installBridge(ctx)
	if err != nil {
		fmt.Println("Error installing bridge:", err)
	}
	fmt.Println("🎉 Thanos Stack installation completed successfully!")
	fmt.Println("🚀 Your network is now up and running.")
	fmt.Println("🔧 You can start interacting with your deployed infrastructure.")

	return nil
}
