package thanos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

const opBridgeImage = "tokamaknetwork/trh-op-bridge-app:latest"

func (t *ThanosStack) InstallBridge(ctx context.Context) (string, error) {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
		chainName = t.deployConfig.ChainName
		l1ChainID = t.deployConfig.L1ChainID
		l1RPC     = t.deployConfig.L1RPCURL
	)

	opBridgePods, err := utils.GetPodsByName(ctx, namespace, "op-bridge")
	if err != nil {
		t.logger.Error("Error to get op bridge pods", "err", err)
		return "", err
	}
	if len(opBridgePods) > 0 {
		t.logger.Info("OP Bridge is running: \n")
		bridgeUrl, err := t.waitForBridgeURL(ctx, namespace, "op-bridge")
		if err != nil {
			return "", err
		}
		return bridgeUrl, nil
	}

	t.logger.Info("Installing a bridge component...")

	var contracts *types.Contracts

	contracts, err = utils.ReadDeployementConfigFromJSONFile(t.deploymentPath, l1ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to read deployment config: %w", err)
	}

	// make yaml file at {cwd}/tokamak-thanos-stack/terraform/thanos-stack/op-bridge-values.yaml
	opBridgeConfig := types.OpBridgeConfig{}

	opBridgeConfig.OpBridge.Env.L1ChainName = constants.L1ChainConfigurations[l1ChainID].ChainName
	opBridgeConfig.OpBridge.Env.L1ChainID = fmt.Sprintf("%d", l1ChainID)
	opBridgeConfig.OpBridge.Env.L1RPC = l1RPC

	opBridgeConfig.OpBridge.Env.L1NativeCurrencyName = constants.L1ChainConfigurations[l1ChainID].NativeTokenName
	opBridgeConfig.OpBridge.Env.L1NativeCurrencySymbol = constants.L1ChainConfigurations[l1ChainID].NativeTokenSymbol
	opBridgeConfig.OpBridge.Env.L1NativeCurrencyDecimals = constants.L1ChainConfigurations[l1ChainID].NativeTokenDecimals

	feeTokenConfig := constants.GetFeeTokenConfig(t.deployConfig.FeeToken, l1ChainID)
	opBridgeConfig.OpBridge.Env.NativeTokenL1Address = feeTokenConfig.L1Address

	opBridgeConfig.OpBridge.Env.L1BlockExplorer = constants.L1ChainConfigurations[l1ChainID].BlockExplorer
	opBridgeConfig.OpBridge.Env.L1USDTAddress = constants.L1ChainConfigurations[l1ChainID].USDTAddress
	opBridgeConfig.OpBridge.Env.L1USDCAddress = constants.L1ChainConfigurations[l1ChainID].USDCAddress

	opBridgeConfig.OpBridge.Env.L2ChainName = chainName
	opBridgeConfig.OpBridge.Env.L2ChainID = fmt.Sprintf("%d", t.deployConfig.L2ChainID)
	opBridgeConfig.OpBridge.Env.L2RPC = t.deployConfig.L2RpcUrl
	opBridgeConfig.OpBridge.Env.L2NativeCurrencyName = feeTokenConfig.Name
	opBridgeConfig.OpBridge.Env.L2NativeCurrencySymbol = feeTokenConfig.Symbol
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

	if t.isLocal() {
		// Local: disable ingress (no ALB), use ClusterIP service
		opBridgeConfig.OpBridge.Ingress = struct {
			Enabled     bool              `yaml:"enabled"`
			ClassName   string            `yaml:"className"`
			Annotations map[string]string `yaml:"annotations"`
			TLS         struct {
				Enabled bool `yaml:"enabled"`
			} `yaml:"tls"`
		}{Enabled: false}
	} else {
		// Cloud: ALB ingress
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
	}

	data, err := yaml.Marshal(&opBridgeConfig)
	if err != nil {
		t.logger.Error("Error marshalling op-bridge values YAML file", "err", err)
		return "", err
	}

	configFileDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	if err := os.MkdirAll(configFileDir, os.ModePerm); err != nil {
		t.logger.Error("Error creating directory", "err", err)
		return "", err
	}

	// Write to file
	filePath := filepath.Join(configFileDir, "/op-bridge-values.yaml")
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		t.logger.Error("Error writing file", "err", err)
		return "", nil
	}

	// For local kind clusters, pre-load the bridge image and set pullPolicy to IfNotPresent
	if t.isLocal() {
		if err := t.loadImageToKind(ctx, opBridgeImage); err != nil {
			t.logger.Warnf("Failed to pre-load bridge image to kind: %v (will attempt pull from registry)", err)
		}
	}

	helmReleaseName := fmt.Sprintf("op-bridge-%d", time.Now().Unix())
	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/op-bridge", t.deploymentPath)

	valueFiles := []string{filePath}
	// For local: override imagePullPolicy so kind uses the pre-loaded image
	if t.isLocal() {
		localOverride := filepath.Join(configFileDir, "op-bridge-local-override.yaml")
		overrideContent := "op_bridge:\n  spec:\n    imagePullPolicy: IfNotPresent\n"
		if writeErr := os.WriteFile(localOverride, []byte(overrideContent), 0644); writeErr == nil {
			valueFiles = append(valueFiles, localOverride)
		}
	}

	if err = t.helmInstallWithFiles(ctx, helmReleaseName, chartPath, namespace, valueFiles); err != nil {
		t.logger.Error("Error installing Helm charts", "err", err)
		return "", err
	}

	t.logger.Info("✅ Bridge component installed successfully and is being initialized...")
	bridgeUrl, err := t.waitForBridgeURL(ctx, namespace, helmReleaseName)
	if err != nil {
		return "", err
	}
	t.logger.Infof("✅ Bridge component is up and running. You can access it at: %s", bridgeUrl)

	return bridgeUrl, nil
}

// waitForBridgeURL returns the bridge URL, using different strategies for local vs cloud.
func (t *ThanosStack) waitForBridgeURL(ctx context.Context, namespace, releaseName string) (string, error) {
	if t.isLocal() {
		// Local: wait for pods to be ready, then return localhost URL.
		// The bridge service is accessible via kubectl port-forward.
		t.logger.Info("Local deployment: waiting for bridge pods to be ready...")
		for i := 0; i < 60; i++ {
			pods, err := utils.GetPodsByName(ctx, namespace, "op-bridge")
			if err == nil && len(pods) > 0 {
				bridgeUrl := "http://localhost:3100"
				t.logger.Infof("Local bridge ready. Access via: kubectl port-forward -n %s svc/%s 3100:3000", namespace, releaseName)
				return bridgeUrl, nil
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
		return "", fmt.Errorf("bridge pods did not become ready within timeout")
	}

	// Cloud: wait for ALB ingress address
	t.logger.Info("Waiting for ingress address to become available...")
	for {
		k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, releaseName)
		if err != nil {
			t.logger.Error("Error retrieving ingress addresses", "err", err, "details", k8sIngresses)
			return "", err
		}

		if len(k8sIngresses) > 0 {
			return "http://" + k8sIngresses[0], nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(15 * time.Second):
		}
	}
}

func (t *ThanosStack) UninstallBridge(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	// AWS config check: only required for cloud deployments
	if !t.isLocal() {
		if t.deployConfig.AWS == nil {
			t.logger.Error("AWS configuration is not set. Please run the deploy command first")
			return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
		}
	}

	releases, err := t.helmFilterReleases(ctx, namespace, "op-bridge")
	if err != nil {
		t.logger.Error("Error to filter helm releases", "err", err)
		return err
	}

	for _, release := range releases {
		if err = t.helmUninstall(ctx, release, namespace); err != nil {
			t.logger.Error("❌ Error uninstalling op-bridge helm chart", "err", err)
			return err
		}
	}

	t.logger.Info("✅ Uninstall a bridge component successfully!")

	return nil
}
