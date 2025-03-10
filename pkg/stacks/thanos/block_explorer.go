package thanos

import (
	"fmt"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) installBlockExplorer(deployConfig *types.Config) error {
	var (
		namespace = deployConfig.K8s.Namespace
		vpcId     = deployConfig.AWS.VpcID
		awsRegion = deployConfig.AWS.Region
		chainName = deployConfig.ChainName
	)

	awsConfig := deployConfig.AWS
	if awsConfig == nil {
		return fmt.Errorf("AWS configuration is missing")
	}

	_, err := loginAWS(awsConfig)
	if err != nil {
		fmt.Println("Error to login in AWS:", err)
		return err
	}

	blockExplorerPods, err := utils.GetPodsByName(namespace, "block-explorer")
	if err != nil {
		fmt.Println("Error to get block explorer pods:", err)
		return err
	}
	if len(blockExplorerPods) > 0 {
		fmt.Printf("Block Explorer is running: \n")
		for _, pod := range blockExplorerPods {
			fmt.Println(pod)
		}
		return nil
	}

	err = t.cloneSourcecode("tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		fmt.Println("Error cloning repository:", err)
		return err
	}

	fmt.Println("Installing a block explorer component...")

	// Make .envrc file
	fmt.Print("Please input your database username: ")
	databaseUserName, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning database name: ", err)
		return err
	}

	fmt.Print("Please input your database password: ")
	databasePassword, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning database name:", err)
		return err
	}

	fmt.Print("Please input your CoinMarketCap key(read more):")
	coinmarketcapKey, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning CoinMarketCap key:", err)
		return err
	}
	fmt.Print("Please input your CoinMarketCap token id(read more):")
	coinmarketcapTokenId, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning CoinMarketCap token id:", err)
		return err
	}

	fmt.Print("Please input your wallet connect id(read more):")
	walletConnectID, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning wallet connect id:", err)
		return err
	}

	err = makeBlockExplorerEnvs(
		"tokamak-thanos-stack/terraform/block-explorer",
		".envrc",
		types.BlockExplorerEnvs{
			ThanosStackName:               chainName,
			BlockExplorerDatabasePassword: databasePassword,
			BlockExplorerDatabaseUserName: databaseUserName,
			BlockExplorerDatabaseName:     "blockscout",
			VpcId:                         vpcId,
			AwsRegion:                     awsRegion,
		},
	)
	if err != nil {
		fmt.Println("Error creating block explorer environments file:", err)
		return err
	}

	//err = utils.ExecuteCommandStream("bash", []string{
	//	"-c",
	//	`cd tokamak-thanos-stack/terraform/block-explorer &&
	//	source .envrc &&
	//	terraform init &&
	//	terraform plan &&
	//	terraform apply -auto-approve
	//	`,
	//}...)
	//if err != nil {
	//	fmt.Println("Error initializing Terraform backend:", err)
	//	return err
	//}

	//chainReleaseName, err := utils.FilterHelmReleases(namespace, namespace)
	//if err != nil {
	//	fmt.Println("Error filtering helm releases:", err)
	//	return err
	//}
	//if len(chainReleaseName) == 0 {
	//	fmt.Println("No helm releases found")
	//	return nil
	//}
	//
	//releaseName := chainReleaseName[0]
	releaseName := "nam-sdk-sepolia"
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
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
		`,
		deployConfig.DeploymentPath,
		deployConfig.L1RPCURL,
		constants.L2ChainId,
		coinmarketcapKey,
		coinmarketcapTokenId,
		releaseName,
		deployConfig.ChainName,
		walletConnectID,
		fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", cwd),
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
		"cd tokamak-thanos-stack/charts/blockscout-stack && source .env && sh ./scripts/generate-blockscout.sh",
	)
	if err != nil {
		fmt.Print("\r❌ Make helm values failed!\n")
		return err
	}

	//// Install backend first
	//blockExplorerBackendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-be", time.Now().Unix())
	fileValue := fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", cwd)
	//_, err = utils.ExecuteCommand("helm", []string{
	//	"install",
	//	blockExplorerBackendReleaseName,
	//	"thanos-stack/block-explorer",
	//	"--values",
	//	fileValue,
	//	"--namespace",
	//	namespace,
	//}...)
	//if err != nil {
	//	fmt.Println("Error installing block explorer backend component:", err)
	//	return err
	//}
	//
	//// Install the frontend
	//// Get the ingress
	//var blockExplorerUrl string
	//for {
	//	k8sIngresses, err := utils.GetAddressByIngress(namespace, blockExplorerBackendReleaseName)
	//	if err != nil {
	//		fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
	//		return err
	//	}
	//
	//	if len(k8sIngresses) > 0 {
	//		blockExplorerUrl = "http://" + k8sIngresses[0]
	//		break
	//	}
	//
	//	time.Sleep(15 * time.Second)
	//}

	blockExplorerUrl := "backend.com"

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

	//blockExplorerFrontendReleaseName := fmt.Sprintf("%s-%d", "block-explorer-fe", time.Now().Unix())
	//_, err = utils.ExecuteCommand("helm", []string{
	//	"install",
	//	blockExplorerFrontendReleaseName,
	//	"thanos-stack/block-explorer",
	//	"--values",
	//	fileValue,
	//	"--namespace",
	//	namespace,
	//}...)
	//if err != nil {
	//	fmt.Println("Error installing block explorer front-end component:", err)
	//}

	return nil
}
