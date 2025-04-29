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

func (t *ThanosStack) installBlockExplorer(ctx context.Context, deployConfig *types.Config) error {
	if deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}
	_, _, err := t.loginAWS(ctx, deployConfig)
	if err != nil {
		fmt.Println("Error to login in AWS:", err)
		return err
	}
	var (
		namespace = deployConfig.K8s.Namespace
		vpcId     = deployConfig.AWS.VpcID
	)

	blockExplorerPods, err := utils.GetPodsByName(namespace, "block-explorer")
	if err != nil {
		fmt.Println("Error to get block explorer pods:", err)
		return err
	}
	if len(blockExplorerPods) > 0 {
		fmt.Printf("Block Explorer is running: \n")
		return nil
	}

	err = t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return err
	}

	fmt.Println("Installing a block explorer component...")

	// Make .envrc file
	installBlockExplorerInput, err := t.inputInstallBlockExplorer()
	if err != nil || installBlockExplorerInput == nil {
		fmt.Println("Error installing block explorer:", err)
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
		fmt.Println("Error creating block explorer environments file:", err)
		return err
	}

	chainReleaseName, err := utils.FilterHelmReleases(namespace, namespace)
	if err != nil {
		fmt.Println("Error filtering helm releases:", err)
		return err
	}
	if len(chainReleaseName) == 0 {
		fmt.Println("No helm releases found")
		return nil
	}

	releaseName := chainReleaseName[0]

	err = utils.ExecuteCommandStream("bash", []string{
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
		fmt.Println("Error initializing Terraform backend:", err)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
	}

	rdsConnectionUrl, err := utils.ExecuteCommand("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd block-explorer &&	
		terraform output -json rds_connection_url`,
	}...)
	if err != nil {
		return fmt.Errorf("failed to get terraform output for %s: %w", "vpc_id", err)
	}

	rdsConnectionUrl = strings.Trim(rdsConnectionUrl, `"`)

	var opGethSVC string
	for {
		k8sSvc, err := utils.GetServiceNames(namespace, "op-geth")
		if err != nil {
			fmt.Println("Error retrieving svc:", err, "details:", k8sSvc)
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
		k8sIngresses, err := utils.GetAddressByIngress(namespace, "op-geth")
		if err != nil {
			fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
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
		"-c",
		fmt.Sprintf("cd tokamak-thanos-stack/charts/blockscout-stack && echo '%s' > .env", envValues),
	)
	if err != nil {
		fmt.Print("\r❌ Make .env file failed!\n")
		return err
	}

	_, err = utils.ExecuteCommand(
		"bash",
		"-c",
		"cd tokamak-thanos-stack/charts/blockscout-stack && source .env && bash ./scripts/generate-blockscout.sh",
	)
	if err != nil {
		fmt.Print("\r❌ Make helm values failed!\n")
		return err
	}
	cwd, err = os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
	}

	// Install backend first
	blockExplorerBackendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-be", time.Now().Unix())
	fileValue := fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", cwd)
	_, err = utils.ExecuteCommand("helm", []string{
		"install",
		blockExplorerBackendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", cwd),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing block explorer backend component:", err)
		return err
	}
	fmt.Println("✅ Install block explorer backend component successfully")

	// Install the frontend
	// Get the ingress
	var blockExplorerUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, blockExplorerBackendReleaseName)
		if err != nil {
			fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
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
		fmt.Println("Error updating blockscout.enabled field:", err)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.enabled", true)
	if err != nil {
		fmt.Println("Error updating frontend.enabled field:", err)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_API_HOST", blockExplorerUrl)
	if err != nil {
		fmt.Println("Error updating NEXT_PUBLIC_API_HOST field:", err)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_APP_HOST", blockExplorerUrl)
	if err != nil {
		fmt.Println("Error updating NEXT_PUBLIC_APP_HOST field:", err)
		return err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.ingress.hostname", blockExplorerUrl)
	if err != nil {
		fmt.Println("Error updating frontend.ingress.hostname field:", err)
		return err
	}

	blockExplorerFrontendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-fe", time.Now().Unix())
	_, err = utils.ExecuteCommand("helm", []string{
		"install",
		blockExplorerFrontendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", cwd),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing block explorer front-end component:", err)
	}

	fmt.Printf("✅ Block Explorer frontend component installed successfully. Accessible at: %s\n", fmt.Sprintf("http://%s", blockExplorerUrl))

	return nil
}

func (t *ThanosStack) uninstallBlockExplorer(ctx context.Context, deployConfig *types.Config) error {
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
	releases, err := utils.FilterHelmReleases(namespace, "block-explorer")
	if err != nil {
		fmt.Println("Error to filter helm releases:", err)
		return err
	}

	for _, release := range releases {
		_, err = utils.ExecuteCommand("helm", []string{
			"uninstall",
			release,
			"--namespace",
			namespace,
		}...)
		if err != nil {
			fmt.Println("Error uninstalling op-bridge helm chart:", err)
			return err
		}
	}
	// 2. Destroy terraform resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/block-explorer")
	if err != nil {
		fmt.Println("Error running block-explorer terraform destroy", err)
		return err
	}

	fmt.Println("✅ Uninstall block explorer components successfully")
	return nil
}
