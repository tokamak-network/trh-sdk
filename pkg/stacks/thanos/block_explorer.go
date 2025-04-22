package thanos

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) installBlockExplorer(ctx context.Context, deployConfig *types.Config, logFileName string) error {
	if deployConfig.K8s == nil {
		utils.LogToFile(logFileName, "K8s configuration is not set. Please run the deploy command first", true)
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}
	_, _, err := t.loginAWS(ctx, deployConfig)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error to login in AWS: %s", err), true)
		return err
	}
	var (
		namespace = deployConfig.K8s.Namespace
		vpcId     = deployConfig.AWS.VpcID
	)

	blockExplorerPods, err := utils.GetPodsByName(namespace, "block-explorer")
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error to get block explorer pods: %s", err), true)
		return err
	}
	if len(blockExplorerPods) > 0 {
		utils.LogToFile(logFileName, "Block Explorer is running: \n", true)
		for _, pod := range blockExplorerPods {
			utils.LogToFile(logFileName, pod, true)
		}
		return nil
	}

	err = t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error cloning repository: %s", err), true)
		return err
	}

	utils.LogToFile(logFileName, "Installing a block explorer component...", true)

	// Make .envrc file
	installBlockExplorerInput, err := t.inputInstallBlockExplorer()
	if err != nil || installBlockExplorerInput == nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error installing block explorer: %s", err), true)
		return err
	}
	var (
		databasePassword     = installBlockExplorerInput.DatabasePassword
		databaseUserName     = installBlockExplorerInput.DatabaseUsername
		coinmarketcapKey     = installBlockExplorerInput.CoinmarketcapKey
		coinmarketcapTokenID = installBlockExplorerInput.CoinmarketcapTokenID
		walletConnectID      = installBlockExplorerInput.WalletConnectProjectID
	)
	err = makeBlockExplorerEnvs(
		"tokamak-thanos-stack/terraform",
		".envrc",
		types.BlockExplorerEnvs{
			BlockExplorerDatabasePassword: databasePassword,
			BlockExplorerDatabaseUserName: databaseUserName,
			BlockExplorerDatabaseName:     "blockscout",
			VpcId:                         vpcId,
		},
	)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error creating block explorer environments file: %s", err), true)
		return err
	}

	chainReleaseName, err := utils.FilterHelmReleases(namespace, namespace, logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error filtering helm releases: %s", err), true)
		return err
	}
	if len(chainReleaseName) == 0 {
		utils.LogToFile(logFileName, "No helm releases found", true)
		return nil
	}

	releaseName := chainReleaseName[0]

	err = utils.ExecuteCommandStream("bash", logFileName, []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd block-explorer &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve
		`,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error initializing Terraform backend: %s", err), true)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error getting current working directory: %s", err), true)
		return err
	}

	rdsConnectionUrl, err := utils.ExecuteCommand("bash", logFileName, []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd block-explorer &&	
		terraform output -json rds_connection_url`,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("failed to get terraform output for %s: %s", "rds_connection_url", err), true)
		return err
	}

	rdsConnectionUrl = strings.Trim(rdsConnectionUrl, `"`)

	var opGethSVC string
	for {
		k8sSvc, err := utils.GetServiceNames(namespace, "op-geth", logFileName)
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error retrieving svc: %s, details: %s", err, k8sSvc), true)
			return err
		}

		if len(k8sSvc) > 0 {
			opGethSVC = k8sSvc[0]
			break
		}

		time.Sleep(15 * time.Second)
	}

	var opGethPublicUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, "op-geth", logFileName)
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error retrieving ingress addresses: %s, details: %s", err, k8sIngresses), true)
			return err
		}

		if len(k8sIngresses) > 0 {
			opGethPublicUrl = k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}

	// generate the helm chart value file
	envValues := fmt.Sprintf(`
		export stack_deployments_path=%s
		export stack_l1_rpc_url=%s
		export stack_chain_id=%d
		export stack_coinmarketcap_api_key=%s
		export stack_coinmarketcap_coin_id=%s
		export stack_helm_release_name=%s
		export stack_network_name=%s
		export stack_wallet_connect_project_id=%s
		export rollup_path=%s
		export rds_connection_url=%s
		export l1_beacon_rpc_url=%s
		export op_geth_svc=%s
		export op_geth_public_url=%s
		`,
		deployConfig.DeploymentPath,
		deployConfig.L1RPCURL,
		deployConfig.L2ChainID,
		coinmarketcapKey,
		coinmarketcapTokenID,
		releaseName,
		deployConfig.ChainName,
		walletConnectID,
		fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", cwd),
		rdsConnectionUrl,
		deployConfig.L1BeaconURL,
		opGethSVC,
		opGethPublicUrl,
	)
	_, err = utils.ExecuteCommand(
		"bash",
		logFileName,
		"-c",
		fmt.Sprintf("cd tokamak-thanos-stack/charts/blockscout-stack && echo '%s' > .env", envValues),
	)
	if err != nil {
		utils.LogToFile(logFileName, "❌ Make .env file failed!\n", true)
		return err
	}

	_, err = utils.ExecuteCommand(
		"bash",
		logFileName,
		"-c",
		"cd tokamak-thanos-stack/charts/blockscout-stack && source .env && bash ./scripts/generate-blockscout.sh",
	)
	if err != nil {
		utils.LogToFile(logFileName, "❌ Make helm values failed!\n", true)
		return err
	}
	cwd, err = os.Getwd()
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error getting current working directory: %s", err), true)
		return err
	}

	// Install backend first
	blockExplorerBackendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-be", time.Now().Unix())
	fileValue := fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", cwd)
	_, err = utils.ExecuteCommand("helm", logFileName, []string{
		"install",
		blockExplorerBackendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", cwd),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error installing block explorer backend component: %s", err), true)
		return err
	}
	utils.LogToFile(logFileName, "✅ Install block explorer backend component successfully", true)

	// Install the frontend
	// Get the ingress
	var blockExplorerUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, blockExplorerBackendReleaseName, logFileName)
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error retrieving ingress addresses: %s, details: %s", err, k8sIngresses), true)
			return err
		}

		if len(k8sIngresses) > 0 {
			blockExplorerUrl = k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}

	// update the values file
	err = utils.UpdateYAMLField(fileValue, "blockscout.enabled", false)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error updating blockscout.enabled field: %s", err), true)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.enabled", true)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error updating frontend.enabled field: %s", err), true)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_API_HOST", blockExplorerUrl)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error updating NEXT_PUBLIC_API_HOST field: %s", err), true)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_APP_HOST", blockExplorerUrl)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error updating NEXT_PUBLIC_APP_HOST field: %s", err), true)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.ingress.hostname", blockExplorerUrl)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error updating frontend.ingress.hostname field: %s", err), true)
		return err
	}

	blockExplorerFrontendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-fe", time.Now().Unix())
	_, err = utils.ExecuteCommand("helm", logFileName, []string{
		"install",
		blockExplorerFrontendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", cwd),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error installing block explorer front-end component: %s", err), true)
		return err
	}

	utils.LogToFile(logFileName, fmt.Sprintf("✅ Block Explorer frontend component installed successfully. Accessible at: %s\n", fmt.Sprintf("http://%s", blockExplorerUrl)), true)

	return nil
}

func (t *ThanosStack) uninstallBlockExplorer(ctx context.Context, deployConfig *types.Config, logFileName string) error {
	if deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	if deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = deployConfig.K8s.Namespace
	)

	// 1. Uninstall helm charts
	releases, err := utils.FilterHelmReleases(namespace, "block-explorer", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error to filter helm releases: %s", err), true)
		return err
	}

	for _, release := range releases {
		_, err = utils.ExecuteCommand("helm", logFileName, []string{
			"uninstall",
			release,
			"--namespace",
			namespace,
		}...)
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error uninstalling op-bridge helm chart: %s", err), true)
			return err
		}
	}
	// 2. Destroy terraform resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/block-explorer", logFileName)
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error running block-explorer terraform destroy: %s", err), true)
		return err
	}

	utils.LogToFile(logFileName, "✅ Uninstall block explorer components successfully", true)
	return nil
}
