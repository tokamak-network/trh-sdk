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

// configServerName is the release-relative suffix for the nginx config server
// that serves genesis.json and rollup.json inside the cluster.
const configServerSuffix = "-cfg-srv"

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

	chartFile := fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", t.deploymentPath)
	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())

	// STEP 4. Prepare the values file for a local kind cluster.
	// The chart ships a thanos-stack-values.yaml that targets AWS; for kind we
	// write a local override into the same directory so existing AWS paths stay untouched.

	// 4.1: Load L2OutputOracleProxy from the deployment JSON so op-proposer has its address.
	l2ooAddress, err := t.loadL2OutputOracleProxy()
	if err != nil {
		t.logger.Warnf("Could not load L2OutputOracleProxy (op-proposer may malfunction): %v", err)
		l2ooAddress = ""
	}

	// 4.2: Spin up a tiny nginx pod that serves genesis.json and rollup.json within the cluster.
	//      The container entrypoints use wget to fetch these files at startup.
	configServerBaseURL, err := t.ensureConfigServer(ctx, helmReleaseName, namespace, configFilesDir)
	if err != nil {
		t.logger.Error("Error creating config server", "err", err)
		return fmt.Errorf("ensure config server: %w", err)
	}

	// 4.3: Write the values file with the correct nested structure the chart expects.
	valueFile := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-local-values.yaml", t.deploymentPath)
	if err := t.writeLocalTestnetValuesFile(valueFile, inputs, namespace, configServerBaseURL, l2ooAddress); err != nil {
		t.logger.Error("Error writing local values file", "err", err)
		return fmt.Errorf("write local values file: %w", err)
	}

	// STEP 4b. Install External Secrets Operator CRDs.
	// The thanos-stack chart includes ExternalSecret and SecretStore resources that
	// require the ESO CRD group (external-secrets.io/v1). For a local kind cluster
	// the ESO operator itself is not needed; we only install minimal CRDs so the
	// chart API validation passes, and then pre-create the secret directly.
	t.logger.Info("Applying External Secrets Operator CRDs (local cluster)...")
	if err := t.ensureESOCRDs(ctx); err != nil {
		t.logger.Error("Error applying ESO CRDs", "err", err)
		return fmt.Errorf("ensure ESO CRDs: %w", err)
	}

	// STEP 4c. Pre-create the Kubernetes Secret that the thanos-stack chart expects.
	// On AWS the ExternalSecret operator would pull secrets from SecretsManager;
	// for local kind the operator is absent, so we inject the secret directly.
	secretName := fmt.Sprintf("%s-thanos-stack-secret", helmReleaseName)
	if len(secretName) > 63 {
		secretName = secretName[:63]
	}
	t.logger.Infof("Pre-creating Kubernetes secret %q in namespace %s...", secretName, namespace)
	if err := t.ensureStackSecret(ctx, secretName, namespace); err != nil {
		t.logger.Error("Error pre-creating stack secret", "err", err)
		return fmt.Errorf("pre-create stack secret: %w", err)
	}

	// STEP 4d. Pre-create PVCs using kind's dynamic provisioner.
	// The chart's PV templates use AWS EFS CSI (volumeHandle required) which is
	// incompatible with local kind clusters. We bypass enable_vpc entirely and
	// pre-create PVCs using the standard (local-path) storage class so they bind
	// via dynamic provisioning. The Helm release keeps enable_vpc: false so the
	// chart does not attempt to create conflicting static PVs.
	fullname := fmt.Sprintf("%s-thanos-stack", helmReleaseName)
	if len(fullname) > 63 {
		fullname = fullname[:63]
	}
	t.logger.Info("Pre-creating PVCs for op-geth and op-node (dynamic provisioning)...")
	if err := t.ensureLocalPVCs(ctx, fullname, namespace); err != nil {
		t.logger.Error("Error pre-creating PVCs", "err", err)
		return fmt.Errorf("pre-create PVCs: %w", err)
	}

	// STEP 5a. Helm install (enable_vpc stays false — no chart-managed PVs/PVCs).
	t.logger.Info("Installing Helm release (initial phase: ConfigMaps + ExternalSecret)...")
	if err := t.helmInstallWithFiles(ctx, helmReleaseName, chartFile, namespace, []string{valueFile}); err != nil {
		t.logger.Error("Error installing Helm charts (initial phase)", "err", err)
		return fmt.Errorf("helm install initial phase: %w", err)
	}

	t.logger.Info("Waiting for PVCs to be ready...")
	if err := t.waitForLocalPVCs(ctx, namespace); err != nil {
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
// local kind cluster using the nested structure that tokamak-thanos-stack expects.
//
// configServerBaseURL is the in-cluster HTTP base URL (e.g. "http://svc-name")
// for genesis.json and rollup.json.  l2ooAddress is the L2OutputOracleProxy address.
func (t *ThanosStack) writeLocalTestnetValuesFile(path string, inputs *DeployLocalInfraInput, namespace, configServerBaseURL, l2ooAddress string) error {
	img := constants.DockerImageTag[constants.LocalTestnet]

	opGethImage := fmt.Sprintf("tokamaknetwork/thanos-op-geth:nightly-%s", img.OpGethImageTag)
	opNodeImage := fmt.Sprintf("tokamaknetwork/thanos-op-node:nightly-%s", img.ThanosStackImageTag)
	opBatcherImage := fmt.Sprintf("tokamaknetwork/thanos-op-batcher:nightly-%s", img.ThanosStackImageTag)
	opProposerImage := fmt.Sprintf("tokamaknetwork/thanos-op-proposer:nightly-%s", img.ThanosStackImageTag)

	genesisURL := fmt.Sprintf("%s/genesis.json", configServerBaseURL)
	rollupURL := fmt.Sprintf("%s/rollup.json", configServerBaseURL)

	// l1_rpc.kind is derived from the provider stored in settings.json.
	l1RpcKind := t.deployConfig.L1RPCProvider
	if l1RpcKind == "" {
		l1RpcKind = "alchemy"
	}

	content := fmt.Sprintf(`# Auto-generated by trh-sdk for LocalTestnet (kind cluster).
# Do not commit — this file is regenerated on each deploy.

enable_vpc: false
enable_deployment: false

thanos_stack_infra:
  name: %q
  region: "local"

l1_rpc:
  url: %q
  kind: %q

op_geth:
  image: %q
  env:
    chain_id: %q
    geth_verbosity: 3
    geth_data_dir: "/db"
    rpc_port: 8545
    ws_port: 8546
    genesis_file_url: %q

op_node:
  image: %q
  env:
    l2_engine_auth: "/op-geth-auth/jwt.txt"
    sequencer_enabled: true
    sequencer_l1_confs: 5
    verifier_l1_confs: 4
    rollup_config_url: %q
    rpc_addr: "0.0.0.0"
    rpc_port: 8545
    p2p_disable: true
    metrics_enabled: true
    metrics_addr: "0.0.0.0"
    metrics_port: 7300
    pprof_enabled: true
    rpc_enable_admin: true
    l1_beacon: %q

op_batcher:
  image: %q

op_proposer:
  image: %q
  enabled: true
  env:
    poll_interval: 12s
    rpc_port: 8560
    metrics_enabled: true
    metrics_addr: "0.0.0.0"
    metrics_port: 7300
    l2oo_address: %q

op_challenger:
  enabled: false

graph_node:
  enabled: false

l1_proxyd:
  enabled: false

redis:
  enabled: false
`,
		namespace,
		t.deployConfig.L1RPCURL,
		l1RpcKind,
		opGethImage,
		fmt.Sprintf("%d", t.deployConfig.L2ChainID),
		genesisURL,
		opNodeImage,
		rollupURL,
		inputs.L1BeaconURL,
		opBatcherImage,
		opProposerImage,
		l2ooAddress,
	)

	return os.WriteFile(path, []byte(content), 0644)
}

// loadL2OutputOracleProxy reads the deployment JSON file and returns the
// L2OutputOracleProxy address needed by op-proposer.
func (t *ThanosStack) loadL2OutputOracleProxy() (string, error) {
	deployFilePath := t.deployConfig.DeploymentFilePath
	if deployFilePath == "" {
		return "", fmt.Errorf("deployment_file_path not set in settings.json")
	}

	data, err := os.ReadFile(deployFilePath)
	if err != nil {
		return "", fmt.Errorf("read deployment file %s: %w", deployFilePath, err)
	}

	var contracts types.Contracts
	if err := json.Unmarshal(data, &contracts); err != nil {
		return "", fmt.Errorf("parse deployment file: %w", err)
	}

	if contracts.L2OutputOracleProxy == "" {
		return "", fmt.Errorf("L2OutputOracleProxy not found in deployment file")
	}

	return contracts.L2OutputOracleProxy, nil
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
// Pod names include the Helm release prefix (e.g. "test-chain-abc123-thanos-stack-op-geth-0"),
// so we check for the component name as a substring rather than a prefix.
func allComponentsRunning(podNames []string) bool {
	for _, component := range localInfraComponents {
		found := false
		for _, name := range podNames {
			if strings.Contains(name, component) {
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

// discoverLocalL2RPC attempts to find the L2 RPC URL from the op-geth service.
// For local kind clusters the services are ClusterIP (not NodePort), so we return
// the cluster-internal URL.  If the K8sRunner is available we verify the service
// exists; otherwise we construct the expected DNS name from convention.
func (t *ThanosStack) discoverLocalL2RPC(ctx context.Context, namespace string) (string, error) {
	if t.k8sRunner == nil {
		return "", fmt.Errorf("k8sRunner not available, cannot discover L2 RPC")
	}

	const (
		pollTimeout  = 30 * time.Second
		pollInterval = 5 * time.Second
	)

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		data, err := t.k8sRunner.List(ctx, "services", namespace, "")
		if err == nil {
			var svcList struct {
				Items []struct {
					Metadata struct {
						Name string `json:"name"`
					} `json:"metadata"`
					Spec struct {
						Ports []struct {
							Name string `json:"name"`
							Port int    `json:"port"`
						} `json:"ports"`
					} `json:"spec"`
				} `json:"items"`
			}
			if json.Unmarshal(data, &svcList) == nil {
				for _, svc := range svcList.Items {
					if strings.Contains(svc.Metadata.Name, "op-geth") {
						// Use the cluster-internal service DNS name.
						return fmt.Sprintf("http://%s.%s.svc.cluster.local:8545", svc.Metadata.Name, namespace), nil
					}
				}
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return "", fmt.Errorf("timed out waiting for op-geth service in namespace %s", namespace)
}

// waitForLocalPVCs polls until all PVCs in the namespace reach "Bound" status.
// Uses K8sRunner when available, otherwise shells out to kubectl with KUBECONFIG.
func (t *ThanosStack) waitForLocalPVCs(ctx context.Context, namespace string) error {
	const (
		pollTimeout  = 3 * time.Minute
		pollInterval = 5 * time.Second
	)
	deadline := time.Now().Add(pollTimeout)

	for time.Now().Before(deadline) {
		bound, err := t.allPVCsBound(ctx, namespace)
		if err != nil {
			t.logger.Warnf("PVC status check error (will retry): %v", err)
		} else if bound {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("timed out waiting for PVCs in namespace %s to reach Bound status", namespace)
}

// allPVCsBound returns true when every PVC in the namespace is Bound.
func (t *ThanosStack) allPVCsBound(ctx context.Context, namespace string) (bool, error) {
	if t.k8sRunner != nil {
		data, err := t.k8sRunner.List(ctx, "persistentvolumeclaims", namespace, "")
		if err != nil {
			return false, err
		}
		var list struct {
			Items []struct {
				Status struct {
					Phase string `json:"phase"`
				} `json:"status"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &list); err != nil {
			return false, err
		}
		if len(list.Items) == 0 {
			return false, nil
		}
		for _, item := range list.Items {
			if item.Status.Phase != "Bound" {
				return false, nil
			}
		}
		return true, nil
	}
	// Shell fallback — uses KUBECONFIG env if available.
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc",
		"-n", namespace,
		"-o", "jsonpath={range .items[*]}{.status.phase}{\"\\n\"}{end}")
	if err != nil {
		return false, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return false, nil
	}
	for _, l := range lines {
		if l != "Bound" {
			return false, nil
		}
	}
	return true, nil
}

// ensureConfigServer creates a tiny nginx Deployment + ClusterIP Service in the
// namespace that serves genesis.json and rollup.json.
//
// Because genesis.json can be several MB (> Kubernetes ConfigMap 3 MB limit),
// we inject the files directly into the kind cluster node container via
// `docker exec` and use a hostPath volume.  This works because kind nodes ARE
// Docker containers and the node's filesystem is the container's filesystem.
//
// The returned URL (e.g. "http://<svc>") is used as the base for genesis_file_url
// and rollup_config_url in the Helm values.
func (t *ThanosStack) ensureConfigServer(ctx context.Context, releaseName, namespace, configFilesDir string) (string, error) {
	svcName := releaseName + configServerSuffix
	if len(svcName) > 63 {
		svcName = svcName[:63]
	}

	// Derive the kind node container name from the kubeconfig.
	// By convention for "trh-test" clusters the node is "trh-test-control-plane".
	// We discover it by listing kind containers.
	kindNodeName, err := t.findKindNodeName(ctx)
	if err != nil {
		return "", fmt.Errorf("find kind node: %w", err)
	}

	// Unique path inside the kind node so parallel deployments don't collide.
	nodePath := fmt.Sprintf("/tmp/trh-cfg-%s", svcName)

	// Create the directory inside the kind node.
	if _, err := utils.ExecuteCommand(ctx, "docker", "exec", kindNodeName, "mkdir", "-p", nodePath); err != nil {
		return "", fmt.Errorf("create dir in kind node: %w", err)
	}

	// Inject genesis.json and rollup.json by piping through docker exec stdin.
	for _, fname := range []string{"genesis.json", "rollup.json"} {
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", configFilesDir, fname))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fname, err)
		}
		if err := t.dockerExecWrite(ctx, kindNodeName, fmt.Sprintf("%s/%s", nodePath, fname), data); err != nil {
			return "", fmt.Errorf("inject %s into kind node: %w", fname, err)
		}
	}

	// nginx Deployment using a hostPath that maps to the injected files.
	deployManifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
        - name: nginx
          image: nginx:alpine
          ports:
            - containerPort: 80
          volumeMounts:
            - name: config-files
              mountPath: /usr/share/nginx/html
              readOnly: true
      volumes:
        - name: config-files
          hostPath:
            path: %q
            type: Directory
`, svcName, namespace, svcName, svcName, nodePath)

	svcManifest := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: %s
  ports:
    - port: 80
      targetPort: 80
`, svcName, namespace, svcName)

	combined := deployManifest + "---\n" + svcManifest

	if t.k8sRunner != nil {
		if err := t.k8sRunner.Apply(ctx, []byte(combined)); err != nil {
			return "", fmt.Errorf("apply config server manifests: %w", err)
		}
	} else {
		tmpFile, err := os.CreateTemp("", "cfg-srv-*.yaml")
		if err != nil {
			return "", fmt.Errorf("create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.WriteString(combined); err != nil {
			return "", fmt.Errorf("write config server manifest: %w", err)
		}
		tmpFile.Close()
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmpFile.Name()); err != nil {
			return "", fmt.Errorf("kubectl apply config server: %w", err)
		}
	}

	// In-cluster DNS: pods in the same namespace reach the service by its name.
	return fmt.Sprintf("http://%s", svcName), nil
}

// loadImageToKind pulls a Docker image and loads it into the kind cluster.
// It derives the kind cluster name from the node container name (e.g. "trh-test-control-plane" → "trh-test").
func (t *ThanosStack) loadImageToKind(ctx context.Context, image string) error {
	t.logger.Infof("Loading image %s into kind cluster...", image)

	// Pull image first
	if _, err := utils.ExecuteCommand(ctx, "docker", "pull", image); err != nil {
		return fmt.Errorf("docker pull %s: %w", image, err)
	}

	// Find kind cluster name from node container
	nodeName, err := t.findKindNodeName(ctx)
	if err != nil {
		return err
	}
	// "trh-test-control-plane" → "trh-test"
	clusterName := strings.TrimSuffix(nodeName, "-control-plane")

	if _, err := utils.ExecuteCommand(ctx, "kind", "load", "docker-image", image, "--name", clusterName); err != nil {
		return fmt.Errorf("kind load docker-image %s: %w", image, err)
	}
	t.logger.Infof("✅ Image %s loaded into kind cluster %s", image, clusterName)
	return nil
}

// findKindNodeName returns the Docker container name of the first kind cluster
// control-plane node.  It runs `docker ps` and looks for containers whose names
// end in "-control-plane".
func (t *ThanosStack) findKindNodeName(ctx context.Context) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "docker", "ps", "--format", "{{.Names}}")
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "-control-plane") {
			return line, nil
		}
	}
	return "", fmt.Errorf("no kind control-plane container found in docker ps output")
}

// dockerExecWrite pipes data into a file inside a Docker container using
// `docker exec -i <container> sh -c 'cat > <path>'`.
func (t *ThanosStack) dockerExecWrite(ctx context.Context, container, destPath string, data []byte) error {
	cmd := fmt.Sprintf("cat > %q", destPath)
	return utils.ExecuteCommandWithStdin(ctx, data, "docker", "exec", "-i", container, "sh", "-c", cmd)
}

// ensureLocalPVCs pre-creates static hostPath PersistentVolumes and their
// corresponding PersistentVolumeClaims for the local kind cluster.
//
// The chart's PV templates target AWS EFS CSI (volumeHandle required) which is
// incompatible with kind. We bypass enable_vpc entirely and create hostPath PVs
// with storageClassName "trh-local-manual" (no dynamic provisioner) so the PVCs
// bind immediately rather than waiting for a consumer pod.
func (t *ThanosStack) ensureLocalPVCs(ctx context.Context, fullname, namespace string) error {
	components := []string{"op-geth", "op-node"}

	for _, comp := range components {
		pvName := fmt.Sprintf("%s-%s", fullname, comp)
		pvcName := pvName
		hostPath := fmt.Sprintf("/tmp/trh-local/%s", pvName)

		// Build a combined PV + PVC manifest.
		manifest := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: %s
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Delete
  storageClassName: trh-local-manual
  hostPath:
    path: %q
    type: DirectoryOrCreate
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  storageClassName: trh-local-manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
`, pvName, hostPath, pvcName, namespace)

		if t.k8sRunner != nil {
			if err := t.k8sRunner.Apply(ctx, []byte(manifest)); err != nil {
				return fmt.Errorf("apply PV/PVC %s: %w", pvName, err)
			}
			continue
		}
		// Shell fallback.
		tmpFile, err := os.CreateTemp("", "pv-pvc-*.yaml")
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		if _, err := tmpFile.WriteString(manifest); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return fmt.Errorf("write PV/PVC manifest: %w", err)
		}
		tmpFile.Close()
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmpFile.Name()); err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("kubectl apply PV/PVC %s: %w", pvName, err)
		}
		os.Remove(tmpFile.Name())
	}
	return nil
}

// esoCRDManifest holds minimal CRD definitions for ExternalSecret and SecretStore
// from the external-secrets.io/v1 API group. These allow the thanos-stack Helm chart
// to succeed on a local kind cluster even though the ESO operator is not installed.
// x-kubernetes-preserve-unknown-fields: true is used so the spec need not enumerate
// every field — the cluster will accept any content.
const esoCRDManifest = `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: externalsecrets.external-secrets.io
spec:
  group: external-secrets.io
  names:
    kind: ExternalSecret
    listKind: ExternalSecretList
    plural: externalsecrets
    singular: externalsecret
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        x-kubernetes-preserve-unknown-fields: true
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: secretstores.external-secrets.io
spec:
  group: external-secrets.io
  names:
    kind: SecretStore
    listKind: SecretStoreList
    plural: secretstores
    singular: secretstore
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        x-kubernetes-preserve-unknown-fields: true
`

// ensureESOCRDs applies the minimal External Secrets Operator CRDs to the cluster.
// Uses K8sRunner when available, otherwise shells out to kubectl.
func (t *ThanosStack) ensureESOCRDs(ctx context.Context) error {
	if t.k8sRunner != nil {
		return t.k8sRunner.Apply(ctx, []byte(esoCRDManifest))
	}
	// Shell fallback: write to a temp file and apply.
	tmpFile, err := os.CreateTemp("", "eso-crds-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(esoCRDManifest); err != nil {
		return fmt.Errorf("write CRD manifest: %w", err)
	}
	tmpFile.Close()
	_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmpFile.Name())
	return err
}

// ensureStackSecret creates (or updates) the Kubernetes Secret that the thanos-stack
// Helm chart uses to inject private keys into op-node, op-batcher and op-proposer.
// On AWS this secret is populated by the External Secrets Operator from SecretsManager;
// on a local kind cluster we inject the keys directly.
func (t *ThanosStack) ensureStackSecret(ctx context.Context, secretName, namespace string) error {
	manifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  OP_NODE_P2P_SEQUENCER_KEY: %q
  OP_BATCHER_PRIVATE_KEY: %q
  OP_PROPOSER_PRIVATE_KEY: %q
  OP_CHALLENGER_PRIVATE_KEY: ""
`, secretName, namespace,
		t.deployConfig.SequencerPrivateKey,
		t.deployConfig.BatcherPrivateKey,
		t.deployConfig.ProposerPrivateKey,
	)

	if t.k8sRunner != nil {
		return t.k8sRunner.Apply(ctx, []byte(manifest))
	}
	// Shell fallback.
	tmpFile, err := os.CreateTemp("", "thanos-secret-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(manifest); err != nil {
		return fmt.Errorf("write secret manifest: %w", err)
	}
	tmpFile.Close()
	_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmpFile.Name())
	return err
}
