package thanos

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type UpdateNetworkParams struct {
	L1RPCURL    string
	L1BeaconURL string
}

func (t *ThanosStack) UpdateNetwork(ctx context.Context, inputs *UpdateNetworkInput) error {
	if t.deployConfig == nil || t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return errors.New("your chain hasn't deployed yet. Please run 'trh-sdk deploy' first")
	}

	if inputs == nil {
		t.logger.Error("inputs can not be empty")
		return fmt.Errorf("inputs can not be empty")
	}

	if err := inputs.Validate(ctx); err != nil {
		t.logger.Error("failed to validate inputs", "err", err)
		return err
	}

	if inputs.L1RPC != "" {
		t.deployConfig.L1RPCURL = inputs.L1RPC
		t.deployConfig.L1RPCProvider = utils.DetectRPCKind(inputs.L1RPC)
	}

	if inputs.L1BeaconURL != "" {
		t.deployConfig.L1BeaconURL = inputs.L1BeaconURL
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
		chainName = t.deployConfig.ChainName
	)

	// Step 1. Check the network status
	chainPods, err := utils.GetPodsByName(ctx, namespace, namespace)
	if len(chainPods) == 0 || err != nil {
		t.logger.Error("No pods found for chain", "chainName", chainName, "namespace", namespace)
		return nil
	}

	// Step 3. Update the network
	// Step 3.1. Regenerate the values file
	err = updateTerraformEnvFile(fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath), types.UpdateTerraformEnvConfig{
		L1BeaconUrl:         t.deployConfig.L1BeaconURL,
		L1RpcUrl:            t.deployConfig.L1RPCURL,
		L1RpcProvider:       t.deployConfig.L1RPCProvider,
		ThanosStackImageTag: constants.DockerImageTag[t.deployConfig.Network].ThanosStackImageTag,
		OpGethImageTag:      constants.DockerImageTag[t.deployConfig.Network].OpGethImageTag,
	})
	if err != nil {
		t.logger.Error("Error generating Terraform environment configuration:", "err", err)
		return err
	}

	var (
		thanosValuesFilePath        = fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-values.yaml", t.deploymentPath)
		bridgeValuesFilePath        = fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/op-bridge-values.yaml", t.deploymentPath)
		blockExplorerValuesFilePath = fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack/block-explorer-value.yaml", t.deploymentPath)
		thanosChartPath             = fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", t.deploymentPath)
		bridgeChartPath             = fmt.Sprintf("%s/tokamak-thanos-stack/charts/op-bridge", t.deploymentPath)
		blockExplorerChartPath      = fmt.Sprintf("%s/tokamak-thanos-stack/charts/blockscout-stack", t.deploymentPath)
	)

	// Generate the values file
	thanosStackValueFileExist := utils.CheckFileExists(thanosValuesFilePath)
	if !thanosStackValueFileExist {
		t.logger.Error("configuration file thanos-stack-values.yaml not found")
		return fmt.Errorf("configuration file thanos-stack-values.yaml not found")
	}

	// Step 3.2. Update the thanos-stack-values.yaml file
	if utils.CheckFileExists(thanosValuesFilePath) {
		err = utils.UpdateYAMLField(thanosValuesFilePath, "l1_rpc.url", t.deployConfig.L1RPCURL)
		if err != nil {
			t.logger.Error("Error updating L1_RPC field:", "err", err)
			return err
		}
		err = utils.UpdateYAMLField(thanosValuesFilePath, "l1_rpc.kind", t.deployConfig.L1RPCProvider)
		if err != nil {
			t.logger.Error("Error updating L1_RPC_PROVIDER field:", "err", err)
			return err
		}
		err = utils.UpdateYAMLField(thanosValuesFilePath, "op_node.env.l1_beacon", t.deployConfig.L1BeaconURL)
		if err != nil {
			t.logger.Error("Error updating L1_BEACON_RPC field:", "err", err)
			return err
		}
	}

	// Update the L1 RPC URL in the op-bridge-values.yaml file
	if utils.CheckFileExists(bridgeValuesFilePath) {
		err = utils.UpdateYAMLField(bridgeValuesFilePath, "op_bridge.env.l1_rpc", t.deployConfig.L1RPCURL)
		if err != nil {
			t.logger.Error("Error updating L1_RPC field:", "err", err)
			return err
		}
	}

	// Update the L1 RPC URL in the block-explorer-values.yaml file
	if utils.CheckFileExists(blockExplorerValuesFilePath) {
		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "frontend.enabled", false)
		if err != nil {
			t.logger.Error("Error updating frontend.enabled field:", "err", err)
			return err
		}

		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.enabled", true)
		if err != nil {
			t.logger.Error("Error updating blockscout.enabled field:", "err", err)
			return err
		}

		// Update the L1 RPC URL in the block-explorer-values.yaml file
		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.env.INDEXER_OPTIMISM_L1_RPC", t.deployConfig.L1RPCURL)
		if err != nil {
			t.logger.Error("Error updating L1_RPC field:", "err", err)
			return err
		}

		err = utils.UpdateYAMLField(blockExplorerValuesFilePath, "blockscout.env.INDEXER_BEACON_RPC_URL", t.deployConfig.L1BeaconURL)
		if err != nil {
			t.logger.Error("Error updating L1_RPC field:", "err", err)
			return err
		}
	}

	// Step 3.3. Update the network
	helmReleases, err := utils.GetHelmReleases(ctx, t.deployConfig.K8s.Namespace)
	if err != nil {
		t.logger.Error("Error getting helm releases:", "err", err)
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
		output, err := utils.ExecuteCommand(ctx, "helm", []string{
			"upgrade",
			release,
			chartPath,
			"--values",
			fileValuesPath,
			"--namespace",
			namespace,
		}...)
		if err != nil {
			t.logger.Error("Error updating helm release", "release", release, "err", err, "output", output)
			return err
		}
	}

	if err = t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		t.logger.Error("Error writing to settings.json", "err", err)
		return err
	}

	t.logger.Info("✅ The network is updated successfully. Please wait for the ingress address to become available...")

	return nil
}
