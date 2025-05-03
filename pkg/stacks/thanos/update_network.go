package thanos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) UpdateNetwork(ctx context.Context, deployConfig *types.Config) error {
	if deployConfig == nil || deployConfig.K8s == nil {
		return errors.New("your chain hasn't deployed yet. Please run 'trh-sdk deploy' first")
	}

	_, _, err := t.loginAWS(ctx, deployConfig)
	if err != nil {
		fmt.Println("Error to login in AWS:", err)
		return err
	}

	var (
		namespace = deployConfig.K8s.Namespace
		chainName = deployConfig.ChainName
	)

	// Step 1. Check the network status
	chainPods, err := utils.GetPodsByName(namespace, namespace)
	if len(chainPods) == 0 || err != nil {
		fmt.Printf("No pods found for chain %s in namespace %s\n", chainName, namespace)
		return nil
	}

	// Step 2. Get the input from users
	// Step 2.1. Get L1 RPC
	fmt.Print("Do you want to update the L1 RPC? (Y/n): ")
	wantUpdateL1RPC, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Println("Error scanning the L1 RPC option", err)
		return err
	}
	if wantUpdateL1RPC {
		l1RPC, l1Kind, _, err := t.inputL1RPC(ctx)
		if err != nil {
			fmt.Println("Error scanning the L1 RPC URL", err)
			return err
		}

		deployConfig.L1RPCURL = l1RPC
		deployConfig.L1RPCProvider = l1Kind
	}

	// Step 2.2. Get the Beacon RPC
	fmt.Print("Do you want to update the L1 Beacon RPC? (Y/n): ")
	wantUpdateL1BeaconRPC, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Println("Error scanning the L1 Beacon RPC option", err)
		return err
	}
	if wantUpdateL1BeaconRPC {
		l1BeaconRPC, err := t.inputL1BeaconURL()
		if err != nil {
			fmt.Println("Error scanning the L1 Beacon RPC URL", err)
		}
		deployConfig.L1BeaconURL = l1BeaconRPC
	}

	fmt.Print("Do you want to update the network? (Y/n): ")
	wantUpdate, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Println("Error scanning input:", err)
		return err
	}

	if !wantUpdate {
		fmt.Println("Skip to update the network")
		return nil
	}

	// Step 3. Update the network
	// Step 3.1. Regenerate the values file
	err = updateTerraformEnvFile("tokamak-thanos-stack/terraform", types.UpdateTerraformEnvConfig{
		L1BeaconUrl:         deployConfig.L1BeaconURL,
		L1RpcUrl:            deployConfig.L1RPCURL,
		L1RpcProvider:       deployConfig.L1RPCProvider,
		ThanosStackImageTag: constants.DockerImageTag[deployConfig.Network].ThanosStackImageTag,
		OpGethImageTag:      constants.DockerImageTag[deployConfig.Network].OpGethImageTag,
	})
	if err != nil {
		fmt.Println("Error generating Terraform environment configuration:", err)
		return err
	}

	var (
		thanosValuesFilePath        = "tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml"
		bridgeValuesFilePath        = "tokamak-thanos-stack/terraform/thanos-stack/op-bridge-values.yaml"
		blockExplorerValuesFilePath = "tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml"
		thanosChartPath             = "tokamak-thanos-stack/charts/thanos-stack"
		bridgeChartPath             = "tokamak-thanos-stack/charts/op-bridge"
		blockExplorerChartPath      = "tokamak-thanos-stack/charts/blockscout-stack"
	)

	// Generate the values file
	thanosStackValueFileExist := utils.CheckFileExists(thanosValuesFilePath)
	if !thanosStackValueFileExist {
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	// Step 3.2. Update the thanos-stack-values.yaml file
	if utils.CheckFileExists(thanosValuesFilePath) {
		err = utils.UpdateYAMLField(thanosValuesFilePath, "l1_rpc.url", deployConfig.L1RPCURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}
		err = utils.UpdateYAMLField(thanosValuesFilePath, "l1_rpc.kind", deployConfig.L1RPCProvider)
		if err != nil {
			fmt.Println("Error updating L1_RPC_PROVIDER field:", err)
			return err
		}
		err = utils.UpdateYAMLField(thanosValuesFilePath, "op_node.env.l1_beacon", deployConfig.L1BeaconURL)
		if err != nil {
			fmt.Println("Error updating L1_BEACON_RPC field:", err)
			return err
		}
	}

	// Update the L1 RPC URL in the op-bridge-values.yaml file
	if utils.CheckFileExists(bridgeValuesFilePath) {
		err = utils.UpdateYAMLField(bridgeValuesFilePath, "op_bridge.env.l1_rpc", deployConfig.L1RPCURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}
	}

	// Update the L1 RPC URL in the block-explorer-values.yaml file
	if utils.CheckFileExists(blockExplorerValuesFilePath) {
		// Update the L1 RPC URL in the block-explorer-values.yaml file
		if utils.CheckFileExists(blockExplorerValuesFilePath) {
			err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.env.INDEXER_OPTIMISM_L1_RPC", deployConfig.L1RPCURL)
			if err != nil {
				fmt.Println("Error updating L1_RPC field:", err)
				return err
			}
		}

		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.env.INDEXER_BEACON_RPC_URL", deployConfig.L1BeaconURL)
		if err != nil {
			fmt.Println("Error updating L1_RPC field:", err)
			return err
		}
	}

	// Step 3.3. Update the network
	helmReleases, err := utils.GetHelmReleases(deployConfig.K8s.Namespace)
	if err != nil {
		fmt.Println("Error getting helm releases:", err)
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
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
		} else if strings.Contains(release, "block-explorer") {
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

	if err = deployConfig.WriteToJSONFile(); err != nil {
		fmt.Println("Error writing to settings.json", err)
		return err
	}

	fmt.Println("✅ The network is updated successfully. Please wait for the ingress address to become available...")

	return nil
}
