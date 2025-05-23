package thanos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type UpdateNetworkParams struct {
	L1RPCURL    string
	L1BeaconURL string
}

func (t *ThanosStack) UpdateNetwork(ctx context.Context) error {
	if t.deployConfig == nil || t.deployConfig.K8s == nil {
		return errors.New("your chain hasn't deployed yet. Please run 'trh-sdk deploy' first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
		chainName = t.deployConfig.ChainName
	)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return err
	}
	deploymentPath := fmt.Sprintf("%s/%s", cwd, t.deploymentPath)

	// Step 1. Check the network status
	chainPods, err := utils.GetPodsByName(namespace, namespace)
	if len(chainPods) == 0 || err != nil {
		fmt.Printf("No pods found for chain %s in namespace %s\n", chainName, namespace)
		return nil
	}

	// Step 3. Update the network
	// Step 3.1. Regenerate the values file
	err = updateTerraformEnvFile(fmt.Sprintf("%s/tokamak-thanos-stack/terraform", deploymentPath), types.UpdateTerraformEnvConfig{
		L1BeaconUrl:         t.deployConfig.L1BeaconURL,
		L1RpcUrl:            t.deployConfig.L1RPCURL,
		L1RpcProvider:       t.deployConfig.L1RPCProvider,
		ThanosStackImageTag: constants.DockerImageTag[t.deployConfig.Network].ThanosStackImageTag,
		OpGethImageTag:      constants.DockerImageTag[t.deployConfig.Network].OpGethImageTag,
	})
	if err != nil {
		fmt.Println("Error generating Terraform environment configuration:", err)
		return err
	}

	var (
		thanosValuesFilePath        = fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", deploymentPath)
		bridgeValuesFilePath        = fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/op-bridge-values.yaml", deploymentPath)
		blockExplorerValuesFilePath = fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", deploymentPath)
		thanosChartPath             = fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", deploymentPath)
		bridgeChartPath             = fmt.Sprintf("%s/tokamak-thanos-stack/charts/op-bridge", deploymentPath)
		blockExplorerChartPath      = fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", deploymentPath)
	)

	// Generate the values file
	thanosStackValueFileExist := utils.CheckFileExists(thanosValuesFilePath)
	if !thanosStackValueFileExist {
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	// Step 3.2. Update the thanos-stack-values.yaml file
	if utils.CheckFileExists(thanosValuesFilePath) {
		err = utils.UpdateYAMLField(thanosValuesFilePath, "l1_rpc.url", t.deployConfig.L1RPCURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}
		err = utils.UpdateYAMLField(thanosValuesFilePath, "l1_rpc.kind", t.deployConfig.L1RPCProvider)
		if err != nil {
			fmt.Println("Error updating L1_RPC_PROVIDER field:", err)
			return err
		}
		err = utils.UpdateYAMLField(thanosValuesFilePath, "op_node.env.l1_beacon", t.deployConfig.L1BeaconURL)
		if err != nil {
			fmt.Println("Error updating L1_BEACON_RPC field:", err)
			return err
		}
	}

	// Update the L1 RPC URL in the op-bridge-values.yaml file
	if utils.CheckFileExists(bridgeValuesFilePath) {
		err = utils.UpdateYAMLField(bridgeValuesFilePath, "op_bridge.env.l1_rpc", t.deployConfig.L1RPCURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}
	}

	// Update the L1 RPC URL in the block-explorer-values.yaml file
	if utils.CheckFileExists(blockExplorerValuesFilePath) {
		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "frontend.enabled", false)
		if err != nil {
			fmt.Println("Error updating frontend.enabled field:", err)
			return err
		}

		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.enabled", true)
		if err != nil {
			fmt.Println("Error updating blockscout.enabled field:", err)
			return err
		}

		// Update the L1 RPC URL in the block-explorer-values.yaml file
		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.env.INDEXER_OPTIMISM_L1_RPC", t.deployConfig.L1RPCURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}

		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.env.INDEXER_BEACON_RPC_URL", t.deployConfig.L1BeaconURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}
	}

	// Step 3.3. Update the network
	helmReleases, err := utils.GetHelmReleases(t.deployConfig.K8s.Namespace)
	if err != nil {
		fmt.Println("Error getting helm releases:", err)
		return err
	}

	for _, release := range helmReleases {
		var (
			fileValuesPath string
			chartPath      string
		)
		if strings.Contains(release, namespace) {
			fileValuesPath = thanosValuesFilePath
			chartPath = thanosChartPath
		} else if strings.Contains(release, "op-bridge") {
			fileValuesPath = bridgeValuesFilePath
			chartPath = bridgeChartPath
		} else if strings.Contains(release, "block-explorer-be") {
			fileValuesPath = blockExplorerValuesFilePath
			chartPath = blockExplorerChartPath
		}
		if fileValuesPath == "" || chartPath == "" {
			continue
		}
		// Update the helm release
		_, err = utils.ExecuteCommand("helm", []string{
			"upgrade",
			release,
			fmt.Sprintf("%s/%s", cwd, chartPath),
			"--values",
			fileValuesPath,
			"--namespace",
			namespace,
		}...)
		if err != nil {
			fmt.Printf("Error updating helm release: %s, err: %s \n", release, err.Error())
			return err
		}
	}

	if err = t.deployConfig.WriteToJSONFile(); err != nil {
		fmt.Println("Error writing to settings.json", err)
		return err
	}

	fmt.Println("✅ The network is updated successfully. Please wait for the ingress address to become available...")

	return nil
}
