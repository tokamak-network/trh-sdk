package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const configServerSuffix = "-cfg-srv"

// DeployLocalInfraInput holds the parameters needed to deploy the Thanos stack
// to a local kind cluster (LocalTestnet network).
type DeployLocalInfraInput struct {
	ChainName   string
	L1BeaconURL string
}

// DeployLocalInfrastructure deploys the Thanos stack Helm charts to a pre-existing
// kind cluster. Pre-condition: DeployContracts must have completed successfully.
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
		return fmt.Errorf("clone tokamak-thanos-stack: %w", err)
	}

	namespace := utils.ConvertChainNameToNamespace(inputs.ChainName)
	t.logger.Infof("Using namespace: %s", namespace)

	// STEP 2. Create the Kubernetes namespace (idempotent).
	if _, err := t.kubectl(ctx, "create", "namespace", namespace, "--dry-run=client", "-o", "name"); err != nil {
		t.logger.Warnf("kubectl namespace dry-run failed: %v (continuing)", err)
	} else {
		if _, createErr := t.kubectl(ctx, "create", "namespace", namespace); createErr != nil {
			t.logger.Warnf("kubectl create namespace %s: %v (may already exist)", namespace, createErr)
		}
	}

	// STEP 3. Copy rollup.json and genesis.json from build output into chart config-files.
	configFilesDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/config-files", t.deploymentPath)
	if err := os.MkdirAll(configFilesDir, 0755); err != nil {
		return fmt.Errorf("mkdir config-files: %w", err)
	}
	if err := utils.CopyFile(fmt.Sprintf("%s/tokamak-thanos/build/rollup.json", t.deploymentPath), fmt.Sprintf("%s/rollup.json", configFilesDir)); err != nil {
		return fmt.Errorf("copy rollup.json: %w", err)
	}
	if err := utils.CopyFile(fmt.Sprintf("%s/tokamak-thanos/build/genesis.json", t.deploymentPath), fmt.Sprintf("%s/genesis.json", configFilesDir)); err != nil {
		return fmt.Errorf("copy genesis.json: %w", err)
	}
	t.logger.Info("Configuration files copied successfully")

	chartFile := fmt.Sprintf("%s/tokamak-thanos-stack/charts/thanos-stack", t.deploymentPath)
	helmReleaseName := fmt.Sprintf("%s-%d", namespace, time.Now().Unix())

	// STEP 4. Prepare the values file.
	l2ooAddress, err := t.loadL2OutputOracleProxy()
	if err != nil {
		t.logger.Warnf("Could not load L2OutputOracleProxy: %v", err)
		l2ooAddress = ""
	}

	configServerBaseURL, err := t.ensureConfigServer(ctx, helmReleaseName, namespace, configFilesDir)
	if err != nil {
		return fmt.Errorf("ensure config server: %w", err)
	}

	valueFile := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/thanos-stack-local-values.yaml", t.deploymentPath)
	if err := t.writeLocalTestnetValuesFile(valueFile, inputs, namespace, configServerBaseURL, l2ooAddress); err != nil {
		return fmt.Errorf("write local values file: %w", err)
	}

	// STEP 4b. External Secrets Operator CRDs.
	t.logger.Info("Applying External Secrets Operator CRDs (local cluster)...")
	if err := t.ensureESOCRDs(ctx); err != nil {
		return fmt.Errorf("ensure ESO CRDs: %w", err)
	}

	// STEP 4c. Pre-create the Kubernetes Secret.
	secretName := fmt.Sprintf("%s-thanos-stack-secret", helmReleaseName)
	if len(secretName) > 63 {
		secretName = secretName[:63]
	}
	t.logger.Infof("Pre-creating Kubernetes secret %q in namespace %s...", secretName, namespace)
	if err := t.ensureStackSecret(ctx, secretName, namespace); err != nil {
		return fmt.Errorf("pre-create stack secret: %w", err)
	}

	// STEP 4d. Pre-create PVCs.
	fullname := fmt.Sprintf("%s-thanos-stack", helmReleaseName)
	if len(fullname) > 63 {
		fullname = fullname[:63]
	}
	t.logger.Info("Pre-creating PVCs for op-geth and op-node...")
	if err := t.ensureLocalPVCs(ctx, fullname, namespace); err != nil {
		return fmt.Errorf("pre-create PVCs: %w", err)
	}

	// STEP 5a. Helm install (initial phase).
	t.logger.Info("Installing Helm release (initial phase)...")
	if _, err := t.helm(ctx, "upgrade", "--install", helmReleaseName, chartFile,
		"--values", valueFile, "--namespace", namespace, "--create-namespace"); err != nil {
		return fmt.Errorf("helm install initial phase: %w", err)
	}

	t.logger.Info("Waiting for PVCs to be ready...")
	if err := t.waitForLocalPVCs(ctx, namespace); err != nil {
		return fmt.Errorf("wait PVC ready: %w", err)
	}

	// STEP 5b. Enable deployment phase.
	if err := utils.UpdateYAMLField(valueFile, "enable_deployment", true); err != nil {
		t.logger.Warnf("UpdateYAMLField enable_deployment: %v", err)
	}

	t.logger.Info("Upgrading Helm release (deployment phase)...")
	if _, err := t.helm(ctx, "upgrade", helmReleaseName, chartFile,
		"--values", valueFile, "--namespace", namespace); err != nil {
		return fmt.Errorf("helm upgrade deployment phase: %w", err)
	}

	t.logger.Info("Waiting for pods to become ready...")
	if err := t.waitForLocalPods(ctx, namespace); err != nil {
		return fmt.Errorf("wait pods running: %w", err)
	}
	t.logger.Info("All L2 component pods are running")

	// STEP 6. Discover the L2 RPC URL.
	l2RPCUrl, err := t.discoverLocalL2RPC(ctx, namespace)
	if err != nil {
		t.logger.Warnf("Could not determine L2 RPC URL: %v", err)
		l2RPCUrl = "http://localhost:8545"
	}
	t.logger.Infof("🌐 L2 RPC endpoint: %s", l2RPCUrl)

	// STEP 7. Persist metadata to settings.json.
	t.deployConfig.K8s = &types.K8sConfig{Namespace: namespace}
	t.deployConfig.L2RpcUrl = l2RPCUrl
	t.deployConfig.L1BeaconURL = inputs.L1BeaconURL
	t.deployConfig.ChainName = inputs.ChainName

	if err := t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		return fmt.Errorf("save settings.json: %w", err)
	}

	t.logger.Info("🎉 LocalTestnet Thanos Stack deployed successfully!")
	return nil
}

// writeLocalTestnetValuesFile generates a Helm values YAML file for a local kind cluster.
func (t *ThanosStack) writeLocalTestnetValuesFile(path string, inputs *DeployLocalInfraInput, namespace, configServerBaseURL, l2ooAddress string) error {
	img := constants.DockerImageTag[constants.LocalTestnet]
	opGethImage := fmt.Sprintf("tokamaknetwork/thanos-op-geth:nightly-%s", img.OpGethImageTag)
	opNodeImage := fmt.Sprintf("tokamaknetwork/thanos-op-node:nightly-%s", img.ThanosStackImageTag)
	opBatcherImage := fmt.Sprintf("tokamaknetwork/thanos-op-batcher:nightly-%s", img.ThanosStackImageTag)
	opProposerImage := fmt.Sprintf("tokamaknetwork/thanos-op-proposer:nightly-%s", img.ThanosStackImageTag)

	l1RpcKind := t.deployConfig.L1RPCProvider
	if l1RpcKind == "" {
		l1RpcKind = "alchemy"
	}

	content := fmt.Sprintf(`# Auto-generated by trh-sdk for LocalTestnet (kind cluster).
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
		namespace, t.deployConfig.L1RPCURL, l1RpcKind,
		opGethImage, fmt.Sprintf("%d", t.deployConfig.L2ChainID),
		fmt.Sprintf("%s/genesis.json", configServerBaseURL),
		opNodeImage, fmt.Sprintf("%s/rollup.json", configServerBaseURL),
		inputs.L1BeaconURL, opBatcherImage, opProposerImage, l2ooAddress,
	)
	return os.WriteFile(path, []byte(content), 0644)
}

// loadL2OutputOracleProxy reads the L2OutputOracleProxy address from the deployment JSON.
func (t *ThanosStack) loadL2OutputOracleProxy() (string, error) {
	path := t.deployConfig.DeploymentFilePath
	if path == "" {
		return "", fmt.Errorf("deployment_file_path not set in settings.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read deployment file %s: %w", path, err)
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

var localInfraComponents = []string{"op-geth", "op-node", "op-batcher", "op-proposer"}

// waitForLocalPods polls until all expected L2 component pods reach Running.
func (t *ThanosStack) waitForLocalPods(ctx context.Context, namespace string) error {
	deadline := time.Now().Add(5 * time.Minute)
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
		case <-time.After(10 * time.Second):
		}
	}
	return fmt.Errorf("timed out waiting for L2 component pods in namespace %s", namespace)
}

func (t *ThanosStack) listRunningPodNames(ctx context.Context, namespace string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx,
		"kubectl", "get", "pods", "-n", namespace,
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

// discoverLocalL2RPC finds the L2 RPC URL from the op-geth service via kubectl.
func (t *ThanosStack) discoverLocalL2RPC(ctx context.Context, namespace string) (string, error) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		out, err := utils.ExecuteCommand(ctx,
			"kubectl", "get", "svc", "-n", namespace,
			"-o", "jsonpath={.items[*].metadata.name}",
		)
		if err == nil {
			for _, svcName := range strings.Split(strings.TrimSpace(out), " ") {
				if strings.Contains(svcName, "op-geth") {
					return fmt.Sprintf("http://%s.%s.svc.cluster.local:8545", svcName, namespace), nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return "", fmt.Errorf("timed out waiting for op-geth service in namespace %s", namespace)
}

// waitForLocalPVCs polls until all PVCs in the namespace reach Bound status.
func (t *ThanosStack) waitForLocalPVCs(ctx context.Context, namespace string) error {
	deadline := time.Now().Add(3 * time.Minute)
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
		case <-time.After(5 * time.Second):
		}
	}
	return fmt.Errorf("timed out waiting for PVCs in namespace %s", namespace)
}

func (t *ThanosStack) allPVCsBound(ctx context.Context, namespace string) (bool, error) {
	out, err := t.kubectl(ctx, "get", "pvc",
		"-n", namespace,
		"-o", `jsonpath={range .items[*]}{.status.phase}{"\n"}{end}`)
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

// ensureConfigServer creates a nginx Deployment + Service to serve genesis.json
// and rollup.json inside the cluster via hostPath on the kind node.
func (t *ThanosStack) ensureConfigServer(ctx context.Context, releaseName, namespace, configFilesDir string) (string, error) {
	svcName := releaseName + configServerSuffix
	if len(svcName) > 63 {
		svcName = svcName[:63]
	}

	kindNodeName, err := t.findKindNodeName(ctx)
	if err != nil {
		return "", fmt.Errorf("find kind node: %w", err)
	}

	nodePath := fmt.Sprintf("/tmp/trh-cfg-%s", svcName)
	if _, err := utils.ExecuteCommand(ctx, "docker", "exec", kindNodeName, "mkdir", "-p", nodePath); err != nil {
		return "", fmt.Errorf("create dir in kind node: %w", err)
	}

	// Inject genesis.json and rollup.json into kind node.
	for _, fname := range []string{"genesis.json", "rollup.json"} {
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", configFilesDir, fname))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fname, err)
		}
		if err := dockerExecWrite(ctx, kindNodeName, fmt.Sprintf("%s/%s", nodePath, fname), data); err != nil {
			return "", fmt.Errorf("inject %s into kind node: %w", fname, err)
		}
	}

	manifest := fmt.Sprintf(`apiVersion: apps/v1
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
---
apiVersion: v1
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
`, svcName, namespace, svcName, svcName, nodePath, svcName, namespace, svcName)

	if err := t.kubectlApplyManifest(ctx, manifest); err != nil {
		return "", fmt.Errorf("apply config server manifests: %w", err)
	}

	return fmt.Sprintf("http://%s", svcName), nil
}

// loadImageToKind pulls a Docker image and loads it into the kind cluster.
func (t *ThanosStack) loadImageToKind(ctx context.Context, image string) error {
	t.logger.Infof("Loading image %s into kind cluster...", image)
	if _, err := utils.ExecuteCommand(ctx, "docker", "pull", image); err != nil {
		return fmt.Errorf("docker pull %s: %w", image, err)
	}
	nodeName, err := t.findKindNodeName(ctx)
	if err != nil {
		return err
	}
	clusterName := strings.TrimSuffix(nodeName, "-control-plane")
	if _, err := utils.ExecuteCommand(ctx, "kind", "load", "docker-image", image, "--name", clusterName); err != nil {
		return fmt.Errorf("kind load docker-image %s: %w", image, err)
	}
	t.logger.Infof("✅ Image %s loaded into kind cluster %s", image, clusterName)
	return nil
}

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

// dockerExecWrite pipes data into a file inside a Docker container.
func dockerExecWrite(ctx context.Context, container, destPath string, data []byte) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container, "sh", "-c", fmt.Sprintf("cat > %q", destPath))
	cmd.Stdin = strings.NewReader(string(data))
	return cmd.Run()
}

// ensureLocalPVCs pre-creates hostPath PVs and PVCs for the local kind cluster.
func (t *ThanosStack) ensureLocalPVCs(ctx context.Context, fullname, namespace string) error {
	for _, comp := range []string{"op-geth", "op-node"} {
		pvName := fmt.Sprintf("%s-%s", fullname, comp)
		hostPath := fmt.Sprintf("/tmp/trh-local/%s", pvName)

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
`, pvName, hostPath, pvName, namespace)

		if err := t.kubectlApplyManifest(ctx, manifest); err != nil {
			return fmt.Errorf("apply PV/PVC %s: %w", pvName, err)
		}
	}
	return nil
}

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

func (t *ThanosStack) ensureESOCRDs(ctx context.Context) error {
	return t.kubectlApplyManifest(ctx, esoCRDManifest)
}

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
	return t.kubectlApplyManifest(ctx, manifest)
}

// kubectlApplyManifest writes a manifest to a temp file and runs kubectl apply
// with --kubeconfig if kubeconfigPath is set.
func (t *ThanosStack) kubectlApplyManifest(ctx context.Context, manifest string) error {
	tmpFile, err := os.CreateTemp("", "k8s-manifest-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(manifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	tmpFile.Close()
	_, err = t.kubectl(ctx, "apply", "-f", tmpFile.Name())
	return err
}
