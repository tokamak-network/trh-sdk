package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) InstallBlockExplorer(ctx context.Context, inputs *InstallBlockExplorerInput) (string, error) {
	if t.deployConfig.K8s == nil {
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	if inputs == nil {
		return "", fmt.Errorf("inputs are not set. Please provide the inputs")
	}

	if err := inputs.Validate(ctx); err != nil {
		return "", err
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
		vpcId     = t.deployConfig.AWS.VpcID
	)

	blockExplorerPods, err := utils.GetPodsByName(ctx, namespace, "block-explorer")
	if err != nil {
		fmt.Println("Error to get block explorer pods:", err)
		return "", err
	}
	if len(blockExplorerPods) > 0 {
		fmt.Printf("Block Explorer is running: \n")
		var blockExplorerURL string
		for {
			k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, "block-explorer")
			if err != nil {
				fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
				return "", err
			}

			if len(k8sIngresses) > 0 {
				blockExplorerURL = "http://" + k8sIngresses[0]
				break
			}

			time.Sleep(15 * time.Second)
		}
		return blockExplorerURL, nil
	}

	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return "", err
	}

	fmt.Println("Installing a block explorer component...")

	// Make .envrc file

	var (
		databasePassword     = inputs.DatabasePassword
		databaseUserName     = inputs.DatabaseUsername
		coinmarketcapKey     = inputs.CoinmarketcapKey
		coinmarketcapTokenID = inputs.CoinmarketcapTokenID
		walletConnectID      = inputs.WalletConnectProjectID
	)
	err = makeBlockExplorerEnvs(
		fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath),
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
		return "", err
	}

	chainReleaseName, err := utils.FilterHelmReleases(ctx, namespace, namespace)
	if err != nil {
		fmt.Println("Error filtering helm releases:", err)
		return "", err
	}
	if len(chainReleaseName) == 0 {
		fmt.Println("No helm releases found")
		return "", nil
	}

	releaseName := chainReleaseName[0]

	err = utils.ExecuteCommandStream(ctx, t.l, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd block-explorer &&
		terraform init &&
		terraform plan &&
		terraform apply -auto-approve
		`, t.deploymentPath),
	}...)
	if err != nil {
		fmt.Println("Error initializing Terraform backend:", err)
		return "", err
	}

	rdsConnectionUrl, err := utils.ExecuteCommand(ctx, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform &&
		source .envrc &&
		cd block-explorer &&	
		terraform output -json rds_connection_url`, t.deploymentPath),
	}...)
	if err != nil {
		return "", fmt.Errorf("failed to get terraform output for %s: %w", "vpc_id", err)
	}

	rdsConnectionUrl = strings.Trim(rdsConnectionUrl, `"`)

	var opGethSVC string
	for {
		k8sSvc, err := utils.GetServiceNames(ctx, namespace, "op-geth")
		if err != nil {
			fmt.Println("Error retrieving svc:", err, "details:", k8sSvc)
			return "", err
		}

		if len(k8sSvc) > 0 {
			opGethSVC = k8sSvc[0]
			break
		}

		time.Sleep(15 * time.Second)
	}

	var opGethPublicUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, "op-geth")
		if err != nil {
			fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
			return "", err
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
		t.deployConfig.DeploymentFilePath,
		t.deployConfig.L1RPCURL,
		t.deployConfig.L2ChainID,
		coinmarketcapKey,
		coinmarketcapTokenID,
		releaseName,
		t.deployConfig.ChainName,
		walletConnectID,
		fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", t.deploymentPath),
		rdsConnectionUrl,
		t.deployConfig.L1BeaconURL,
		opGethSVC,
		opGethPublicUrl,
	)
	_, err = utils.ExecuteCommand(ctx,
		"bash",
		"-c",
		fmt.Sprintf("cd %s/tokamak-thanos-stack/charts/blockscout-stack && echo '%s' > .env", t.deploymentPath, envValues),
	)
	if err != nil {
		fmt.Print("\r❌ Make .env file failed!\n")
		return "", err
	}

	_, err = utils.ExecuteCommand(ctx,
		"bash",
		"-c",
		fmt.Sprintf("cd %s/tokamak-thanos-stack/charts/blockscout-stack && source .env && bash ./scripts/generate-blockscout.sh", t.deploymentPath),
	)
	if err != nil {
		fmt.Print("\r❌ Make helm values failed!\n")
		return "", err
	}

	// Install backend first
	blockExplorerBackendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-be", time.Now().Unix())
	fileValue := fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", t.deploymentPath)
	_, err = utils.ExecuteCommand(ctx, "helm", []string{
		"install",
		blockExplorerBackendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", t.deploymentPath),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing block explorer backend component:", err)
		return "", err
	}
	fmt.Println("✅ Install block explorer backend component successfully")

	// Install the frontend
	// Get the ingress
	var blockExplorerUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, blockExplorerBackendReleaseName)
		if err != nil {
			fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
			return "", err
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
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.enabled", true)
	if err != nil {
		fmt.Println("Error updating frontend.enabled field:", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_API_HOST", blockExplorerUrl)
	if err != nil {
		fmt.Println("Error updating NEXT_PUBLIC_API_HOST field:", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.env.NEXT_PUBLIC_APP_HOST", blockExplorerUrl)
	if err != nil {
		fmt.Println("Error updating NEXT_PUBLIC_APP_HOST field:", err)
		return "", err
	}

	err = utils.UpdateYAMLField(fileValue, "frontend.ingress.hostname", blockExplorerUrl)
	if err != nil {
		fmt.Println("Error updating frontend.ingress.hostname field:", err)
		return "", err
	}

	blockExplorerFrontendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-fe", time.Now().Unix())
	_, err = utils.ExecuteCommand(ctx, "helm", []string{
		"install",
		blockExplorerFrontendReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", t.deploymentPath),
		"--values",
		fileValue,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing block explorer front-end component:", err)
		return "", err
	}

	fmt.Printf("✅ Block Explorer frontend component installed successfully. Accessible at: %s\n", fmt.Sprintf("http://%s", blockExplorerUrl))

	return "http://" + blockExplorerUrl, nil
}

func (t *ThanosStack) UninstallBlockExplorer(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	// 1. Uninstall helm charts
	releases, err := utils.FilterHelmReleases(ctx, namespace, "block-explorer")
	if err != nil {
		fmt.Println("Error to filter helm releases:", err)
		return err
	}

	for _, release := range releases {
		_, err = utils.ExecuteCommand(ctx, "helm", []string{
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
	err = t.destroyTerraform(ctx, fmt.Sprintf("%s/tokamak-thanos-stack/terraform/block-explorer", t.deploymentPath))
	if err != nil {
		fmt.Println("Error running block-explorer terraform destroy", err)
		return err
	}

	fmt.Println("✅ Uninstall block explorer components successfully")
	return nil
}
