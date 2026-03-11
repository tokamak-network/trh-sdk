package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// DeployLocalInfraInput holds the parameters needed to deploy the Thanos stack
// to a local kind cluster (LocalTestnet network).
type DeployLocalInfraInput struct {
	// ChainName is the human-readable chain name; converted to a k8s namespace.
	ChainName string
	// L1BeaconURL is the Sepolia beacon endpoint used by the sequencer.
	L1BeaconURL string
}

// DeployLocalInfrastructure deploys the Thanos stack Helm charts to a pre-existing
// kind cluster. It is the LocalTestnet equivalent of deployNetworkToAWS.
//
// Pre-condition: DeployContracts must have completed successfully so that
// settings.json contains DeployContractState.Status == DeployContractStatusCompleted
// and the rollup.json / genesis.json artifacts exist under deploymentPath.
func (t *ThanosStack) DeployLocalInfrastructure(ctx context.Context, inputs *DeployLocalInfraInput) error {
	if inputs == nil {
		return fmt.Errorf("inputs is required")
	}

	if t.deployConfig == nil {
		return fmt.Errorf("settings.json not found — deploy L1 contracts before deploying local infrastructure")
	}

	if t.deployConfig.DeployContractState == nil ||
		t.deployConfig.DeployContractState.Status != types.DeployContractStatusCompleted {
		return fmt.Errorf("contracts are not deployed successfully, please deploy the contracts first")
	}

	// STEP 1. Clone the Helm charts repository (idempotent).
	if err := t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git"); err != nil {
		t.logger.Error("Error cloning tokamak-thanos-stack repository", "err", err)
		return fmt.Errorf("clone tokamak-thanos-stack: %w", err)
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	t.logger.Infof("Using namespace: %s", namespace)

	// STEP 2. Create the Kubernetes namespace (idempotent via EnsureNamespace).
	if t.k8sRunner != nil {
		if err := t.k8sRunner.EnsureNamespace(ctx, namespace); err != nil {
			t.logger.Error("Error creating namespace", "namespace", namespace, "err", err)
			return fmt.Errorf("ensure namespace %s: %w", namespace, err)
		}
	} else {
		// Shell fallback: `kubectl create namespace` is idempotent with --dry-run=client | apply.
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "name"); err != nil {
			t.logger.Warnf("kubectl namespace dry-run failed: %v (continuing)", err)
		} else {
			if _, createErr := utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace); createErr != nil {
				t.logger.Warnf("kubectl create namespace %s: %v (may already exist)", namespace, createErr)
			}
		}
	}

	// STEP 3. Copy rollup.json and genesis.json from build output into chart config-files.
	configFilesDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files", t.deploymentPath)
	if err := os.MkdirAll(configFilesDir, 0755); err != nil {
		t.logger.Error("Error creating config-files directory", "err", err)
		return fmt.Errorf("mkdir config-files: %w", err)
	}

	rollupSrc := fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", t.deploymentPath)
	rollupDst := fmt.Sprintf("%s/rollup.json", configFilesDir)
	if err := utils.CopyFile(rollupSrc, rollupDst); err != nil {
		t.logger.Error("Error copying rollup.json", "err", err)
		return fmt.Errorf("copy rollup.json: %w", err)
	}

	genesisSrc := fmt.Sprintf("%s/tokamak-thanos/build/genesis.json", t.deploymentPath)
	genesisDst := fmt.Sprintf("%s/genesis.json", configFilesDir)
	if err := utils.CopyFile(genesisSrc, genesisDst); err != nil {
		t.logger.Error("Error copying genesis.json", "err", err)
		return fmt.Errorf("copy genesis.json: %w", err)
	}

	t.logger.Info("Configuration files copied successfully")

	// STEP 4. Prepare the values file for a local kind cluster.
	// The chart ships a thanos-stack-values.yaml that targets AWS; for kind we
	// write a local override into the same directory so existing AWS paths stay untouched.
	valueFile := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-local-values.yaml", t.deploymentPath)
	if err := t.writeLocalTestnetValuesFile(valueFile, inputs, namespace); err != nil {
		t.logger.Error("Error writing local values file", "err", err)
		return fmt.Errorf("write local values file: %w", err)
	}

	chartFile := fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", t.deploymentPath)
	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())

	// STEP 5a. Install with PVC enabled first (storage provisioning).
	t.logger.Info("Installing Helm release (PVC phase)...")
	if err := utils.UpdateYAMLField(valueFile, "enable_vpc", true); err != nil {
		t.logger.Warnf("UpdateYAMLField enable_vpc: %v (continuing with existing value)", err)
	}

	if err := t.helmInstallWithFiles(ctx, helmReleaseName, chartFile, namespace, []string{valueFile}); err != nil {
		t.logger.Error("Error installing Helm charts (PVC phase)", "err", err)
		return fmt.Errorf("helm install PVC phase: %w", err)
	}

	t.logger.Info("Waiting for PVCs to be ready...")
	if err := utils.WaitPVCReady(ctx, namespace); err != nil {
		t.logger.Error("Error waiting for PVC", "err", err)
		return fmt.Errorf("wait PVC ready: %w", err)
	}

	// STEP 5b. Enable the full deployment.
	if err := utils.UpdateYAMLField(valueFile, "enable_deployment", true); err != nil {
		t.logger.Warnf("UpdateYAMLField enable_deployment: %v (continuing with existing value)", err)
	}

	t.logger.Info("Upgrading Helm release (deployment phase)...")
	if err := t.helmUpgradeWithFiles(ctx, helmReleaseName, chartFile, namespace, []string{valueFile}); err != nil {
		t.logger.Error("Error upgrading Helm charts (deployment phase)", "err", err)
		return fmt.Errorf("helm upgrade deployment phase: %w", err)
	}

	t.logger.Info("Helm charts deployed, waiting for pods to become ready...")

	// STEP 5c. Wait for all L2 component pods to reach Running status.
	if err := t.waitForLocalPods(ctx, namespace); err != nil {
		t.logger.Error("Error waiting for pods", "err", err)
		return fmt.Errorf("wait pods running: %w", err)
	}

	t.logger.Info("All L2 component pods are running")

	// STEP 6. Discover the L2 RPC URL from the NodePort service.
	l2RPCUrl, err := t.discoverLocalL2RPC(ctx, namespace)
	if err != nil {
		t.logger.Warnf("Could not determine L2 RPC URL automatically: %v", err)
		// Non-fatal: operator can set up port-forwarding manually.
		l2RPCUrl = "http://localhost:8545"
	}
	t.logger.Infof("🌐 L2 RPC endpoint: %s", l2RPCUrl)

	// STEP 7. Persist metadata to settings.json.
	t.deployConfig.K8s = &types.K8sConfig{Namespace: namespace}
	t.deployConfig.L2RpcUrl = l2RPCUrl
	t.deployConfig.L1BeaconURL = inputs.L1BeaconURL
	t.deployConfig.ChainName = inputs.ChainName

	if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		t.logger.Error("Error saving settings.json", "err", err)
		return fmt.Errorf("save settings.json: %w", err)
	}

	t.logger.Info("🎉 LocalTestnet Thanos Stack deployed successfully!")
	return nil
}

// writeLocalTestnetValuesFile generates a Helm values YAML file suitable for a
// local kind cluster. AWS-specific fields (load balancers, EBS storage class, etc.)
// are replaced with kind-compatible equivalents (NodePort, standard storage class).
func (t *ThanosStack) writeLocalTestnetValuesFile(path string, inputs *DeployLocalInfraInput, namespace string) error {
	imgTags := constants.DockerImageTag[constants.LocalTestnet]

	content := fmt.Sprintf(`# Auto-generated by trh-sdk for LocalTestnet (kind cluster).
# Do not commit — this file is regenerated on each deploy.

namespace: %s

# Phase flags — updated during deployment
enable_vpc: false
enable_deployment: false

# Chain images
op_geth_image_tag: %q
thanos_stack_image_tag: %q

# Sequencer keys (from L1 contract deployment)
sequencer_private_key: %q
batcher_private_key: %q
proposer_private_key: %q

# L1 connectivity
l1_rpc_url: %q
l1_beacon_url: %q

# Storage: use the default (standard) storage class provided by kind
storage_class: standard

# Networking: NodePort instead of LoadBalancer for local clusters
service_type: NodePort

# Backup disabled for local clusters
backup_enabled: false
`,
		namespace,
		imgTags.OpGethImageTag,
		imgTags.ThanosStackImageTag,
		t.deployConfig.SequencerPrivateKey,
		t.deployConfig.BatcherPrivateKey,
		t.deployConfig.ProposerPrivateKey,
		t.deployConfig.L1RPCURL,
		inputs.L1BeaconURL,
	)

	return os.WriteFile(path, []byte(content), 0644)
}

// localInfraComponents lists the pod name prefixes that must be Running
// before the local deployment is considered successful.
var localInfraComponents = []string{"op-geth", "op-node", "op-batcher", "op-proposer"}

// waitForLocalPods polls until all expected L2 component pods reach the Running
// phase. It uses K8sRunner.List when available, falling back to kubectl.
func (t *ThanosStack) waitForLocalPods(ctx context.Context, namespace string) error {
	const (
		pollTimeout  = 5 * time.Minute
		pollInterval = 10 * time.Second
	)

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		running, err := t.listRunningPodNames(ctx, namespace)
		if err != nil {
			t.logger.Warnf("Error listing pods (will retry): %v", err)
		} else if allComponentsRunning(running) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timed out waiting for L2 component pods in namespace %s", namespace)
}

// listRunningPodNames returns the names of pods in the Running phase within the namespace.
func (t *ThanosStack) listRunningPodNames(ctx context.Context, namespace string) ([]string, error) {
	if t.k8sRunner != nil {
		return t.listRunningPodsNative(ctx, namespace)
	}
	return t.listRunningPodsShell(ctx, namespace)
}

// listRunningPodsNative uses K8sRunner.List to find Running pods.
func (t *ThanosStack) listRunningPodsNative(ctx context.Context, namespace string) ([]string, error) {
	data, err := t.k8sRunner.List(ctx, "pods", namespace, "")
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &podList); err != nil {
		return nil, fmt.Errorf("unmarshal pod list: %w", err)
	}

	names := make([]string, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if pod.Status.Phase == "Running" {
			names = append(names, pod.Metadata.Name)
		}
	}
	return names, nil
}

// listRunningPodsShell uses kubectl to list Running pods (fallback).
func (t *ThanosStack) listRunningPodsShell(ctx context.Context, namespace string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx,
		"kubectl", "get", "pods",
		"-n", namespace,
		"--field-selector=status.phase=Running",
		"-o", "jsonpath={.items[*].metadata.name}",
	)
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, " "), nil
}

// allComponentsRunning checks that each expected component has at least one Running pod.
func allComponentsRunning(podNames []string) bool {
	for _, component := range localInfraComponents {
		found := false
		for _, name := range podNames {
			if strings.HasPrefix(name, component) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// discoverLocalL2RPC attempts to find the L2 RPC URL from a NodePort service.
// It polls for up to 3 minutes before returning an error.
func (t *ThanosStack) discoverLocalL2RPC(ctx context.Context, namespace string) (string, error) {
	const (
		pollTimeout  = 3 * time.Minute
		pollInterval = 10 * time.Second
	)

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		// Try to get the NodePort for the op-geth RPC service.
		out, err := utils.ExecuteCommand(ctx,
			"kubectl", "get", "svc",
			"-n", namespace,
			"-o", "jsonpath={.items[?(@.spec.type==\"NodePort\")].spec.ports[?(@.name==\"rpc\")].nodePort}",
		)
		if err == nil && out != "" {
			// Kind exposes NodePorts on the node's internal IP; use localhost for the host machine.
			return fmt.Sprintf("http://localhost:%s", out), nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return "", fmt.Errorf("timed out waiting for NodePort service in namespace %s", namespace)
}
