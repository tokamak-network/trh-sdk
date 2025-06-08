package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

func (t *ThanosStack) InstallBridge(_ context.Context) (string, error) {
	if t.deployConfig.K8s == nil {
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
		chainName = t.deployConfig.ChainName
		l1ChainID = t.deployConfig.L1ChainID
		l1RPC     = t.deployConfig.L1RPCURL
	)

	opBridgePods, err := utils.GetPodsByName(namespace, "op-bridge")
	if err != nil {
		fmt.Println("Error to get op bridge pods:", err)
		return "", err
	}
	if len(opBridgePods) > 0 {
		fmt.Printf("OP Bridge is running: \n")
		var bridgeUrl string
		for {
			k8sIngresses, err := utils.GetAddressByIngress(namespace, "op-bridge")
			if err != nil {
				fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
				return "", err
			}

			if len(k8sIngresses) > 0 {
				bridgeUrl = "http://" + k8sIngresses[0]
				break
			}

			time.Sleep(15 * time.Second)
		}
		return bridgeUrl, nil
	}

	fmt.Println("Installing a bridge component...")

	file, err := os.Open(fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%s", t.deploymentPath, fmt.Sprintf("%d-deploy.json", l1ChainID)))
	if err != nil {
		fmt.Println("Error opening deployment file:", err)
		return "", err
	}
	defer file.Close()

	// Decode JSON
	var contracts types.Contracts
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&contracts); err != nil {
		fmt.Println("Error decoding deployment JSON file:", err)
		return "", err
	}

	// make yaml file at {cwd}/tokamak-thanos-stack/terraform/thanos-stack/op-bridge-values.yaml
	opBridgeConfig := types.OpBridgeConfig{}

	opBridgeConfig.OpBridge.Env.L1ChainName = constants.L1ChainConfigurations[l1ChainID].ChainName
	opBridgeConfig.OpBridge.Env.L1ChainID = fmt.Sprintf("%d", l1ChainID)
	opBridgeConfig.OpBridge.Env.L1RPC = l1RPC

	opBridgeConfig.OpBridge.Env.L1NativeCurrencyName = constants.L1ChainConfigurations[l1ChainID].NativeTokenName
	opBridgeConfig.OpBridge.Env.L1NativeCurrencySymbol = constants.L1ChainConfigurations[l1ChainID].NativeTokenSymbol
	opBridgeConfig.OpBridge.Env.L1NativeCurrencyDecimals = constants.L1ChainConfigurations[l1ChainID].NativeTokenDecimals

	opBridgeConfig.OpBridge.Env.NativeTokenL1Address = constants.L1ChainConfigurations[l1ChainID].L2NativeTokenAddress

	opBridgeConfig.OpBridge.Env.L1BlockExplorer = constants.L1ChainConfigurations[l1ChainID].BlockExplorer
	opBridgeConfig.OpBridge.Env.L1USDTAddress = constants.L1ChainConfigurations[l1ChainID].USDTAddress
	opBridgeConfig.OpBridge.Env.L1USDCAddress = constants.L1ChainConfigurations[l1ChainID].USDCAddress

	opBridgeConfig.OpBridge.Env.L2ChainName = chainName
	opBridgeConfig.OpBridge.Env.L2ChainID = fmt.Sprintf("%d", t.deployConfig.L2ChainID)
	opBridgeConfig.OpBridge.Env.L2RPC = t.deployConfig.L2RpcUrl
	opBridgeConfig.OpBridge.Env.L2NativeCurrencyName = "Tokamak Network Token"
	opBridgeConfig.OpBridge.Env.L2NativeCurrencySymbol = "TON"
	opBridgeConfig.OpBridge.Env.L2NativeCurrencyDecimals = 18
	opBridgeConfig.OpBridge.Env.L2USDTAddress = ""

	opBridgeConfig.OpBridge.Env.StandardBridgeAddress = contracts.L1StandardBridgeProxy
	opBridgeConfig.OpBridge.Env.AddressManagerAddress = contracts.AddressManager
	opBridgeConfig.OpBridge.Env.L1CrossDomainMessengerAddress = contracts.L1CrossDomainMessengerProxy
	opBridgeConfig.OpBridge.Env.OptimismPortalAddress = contracts.OptimismPortalProxy
	opBridgeConfig.OpBridge.Env.L2OutputOracleAddress = contracts.L2OutputOracleProxy
	opBridgeConfig.OpBridge.Env.L1USDCBridgeAddress = contracts.L1UsdcBridgeProxy
	opBridgeConfig.OpBridge.Env.DisputeGameFactoryAddress = contracts.DisputeGameFactoryProxy
	opBridgeConfig.OpBridge.Env.BatchSubmissionFrequency = t.deployConfig.ChainConfiguration.BatchSubmissionFrequency
	opBridgeConfig.OpBridge.Env.L1BlockTime = t.deployConfig.ChainConfiguration.L1BlockTime
	opBridgeConfig.OpBridge.Env.L2BlockTime = t.deployConfig.ChainConfiguration.L2BlockTime
	opBridgeConfig.OpBridge.Env.OutputRootFrequency = t.deployConfig.ChainConfiguration.OutputRootFrequency
	opBridgeConfig.OpBridge.Env.ChallengePeriod = t.deployConfig.ChainConfiguration.ChallengePeriod

	// input from users

	opBridgeConfig.OpBridge.Ingress = struct {
		Enabled     bool              `yaml:"enabled"`
		ClassName   string            `yaml:"className"`
		Annotations map[string]string `yaml:"annotations"`
		TLS         struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"tls"`
	}{Enabled: true, ClassName: "alb", Annotations: map[string]string{
		"alb.ingress.kubernetes.io/target-type":  "ip",
		"alb.ingress.kubernetes.io/scheme":       "internet-facing",
		"alb.ingress.kubernetes.io/listen-ports": "[{\"HTTP\": 80}]",
		"alb.ingress.kubernetes.io/group.name":   "bridge",
	}, TLS: struct {
		Enabled bool `yaml:"enabled"`
	}{
		Enabled: false,
	}}

	data, err := yaml.Marshal(&opBridgeConfig)
	if err != nil {
		fmt.Println("Error marshalling op-bridge values YAML file:", err)
		return "", err
	}

	configFileDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	if err := os.MkdirAll(configFileDir, os.ModePerm); err != nil {
		fmt.Println("Error creating directory:", err)
		return "", err
	}

	// Write to file
	filePath := filepath.Join(configFileDir, "/op-bridge-values.yaml")
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return "", nil
	}

	helmReleaseName := fmt.Sprintf("op-bridge-%d", time.Now().Unix())
	_, err = utils.ExecuteCommand("helm", []string{
		"install",
		helmReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/op-bridge", t.deploymentPath),
		"--values",
		filePath,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		fmt.Println("Error installing Helm charts:", err)
		return "", err
	}

	fmt.Println("✅ Bridge component installed successfully and is being initialized. Please wait for the ingress address to become available...")
	var bridgeUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(namespace, helmReleaseName)
		if err != nil {
			fmt.Println("Error retrieving ingress addresses:", err, "details:", k8sIngresses)
			return "", err
		}

		if len(k8sIngresses) > 0 {
			bridgeUrl = "http://" + k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	fmt.Printf("✅ Bridge component is up and running. You can access it at: %s\n", bridgeUrl)

	return bridgeUrl, nil
}

func (t *ThanosStack) UninstallBridge(_ context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	releases, err := utils.FilterHelmReleases(namespace, "op-bridge")
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

	fmt.Println("Uninstall a bridge component successfully")

	return nil
}
