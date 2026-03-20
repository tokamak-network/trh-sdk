package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
)

// skipIfNoKind skips the test if no kind cluster is available.
func skipIfNoKind(t *testing.T) string {
	t.Helper()

	kubeconfig := os.Getenv("TRH_TEST_KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = "/tmp/trh-test.kubeconfig"
	}
	if _, err := os.Stat(kubeconfig); err != nil {
		t.Skipf("skipping E2E: kubeconfig not found at %s (set TRH_TEST_KUBECONFIG)", kubeconfig)
	}

	// Verify cluster is reachable
	out, err := utils.ExecuteCommand(context.Background(), "kubectl", "--kubeconfig", kubeconfig, "cluster-info")
	if err != nil {
		t.Skipf("skipping E2E: kind cluster not reachable: %v", err)
	}
	if !strings.Contains(out, "running") && !strings.Contains(out, "control plane") {
		t.Skipf("skipping E2E: unexpected cluster-info output: %s", out)
	}

	return kubeconfig
}

// setupE2EStack creates a ThanosStack with a minimal settings.json for E2E testing.
// Returns the stack, temp deployment dir, and cleanup function.
func setupE2EStack(t *testing.T, kubeconfig string) (*ThanosStack, string, func()) {
	t.Helper()

	deployDir := t.TempDir()

	// Write a minimal settings.json that looks like a completed contract deployment
	config := &types.Config{
		Stack:   "thanos",
		Network: "LocalTestnet",
		DeployContractState: &types.DeployContractState{
			Status: types.DeployContractStatusCompleted,
		},
		L1RPCURL:    "https://eth-sepolia.g.alchemy.com/v2/test",
		L1RPCProvider: "alchemy",
		L2ChainID:   111551190241,
		L1ChainID:   11155111,
		SequencerPrivateKey: "0000000000000000000000000000000000000000000000000000000000000001",
		BatcherPrivateKey:   "0000000000000000000000000000000000000000000000000000000000000002",
		ProposerPrivateKey:  "0000000000000000000000000000000000000000000000000000000000000003",
		DeploymentFilePath:  filepath.Join(deployDir, "deploy.json"),
	}
	if err := config.WriteToJSONFile(deployDir); err != nil {
		t.Fatalf("failed to write settings.json: %v", err)
	}

	// Write a dummy deployment file with contract addresses
	deployJSON := `{
		"L2OutputOracleProxy": "0xB54527Ac1a8744C2a72a078256d8Cb8bcf438499",
		"L1StandardBridgeProxy": "0x0000000000000000000000000000000000000001",
		"AddressManager": "0x0000000000000000000000000000000000000002",
		"L1CrossDomainMessengerProxy": "0x0000000000000000000000000000000000000003",
		"OptimismPortalProxy": "0x0000000000000000000000000000000000000004",
		"L1UsdcBridgeProxy": "0x0000000000000000000000000000000000000005",
		"DisputeGameFactoryProxy": "0x0000000000000000000000000000000000000006"
	}`
	os.WriteFile(config.DeploymentFilePath, []byte(deployJSON), 0644)

	// Write dummy genesis.json and rollup.json (needed by deploy)
	buildDir := filepath.Join(deployDir, "tokamak-thanos", "build")
	os.MkdirAll(buildDir, 0755)
	os.WriteFile(filepath.Join(buildDir, "genesis.json"), []byte(`{"config":{}}`), 0644)
	os.WriteFile(filepath.Join(buildDir, "rollup.json"), []byte(`{"genesis":{}}`), 0644)

	logger := zap.NewNop().Sugar()
	stack, err := NewLocalTestnetThanosStack(context.Background(), logger, deployDir, kubeconfig)
	if err != nil {
		t.Fatalf("failed to create stack: %v", err)
	}

	cleanup := func() {
		// Clean up any namespaces we created
		if stack.deployConfig != nil && stack.deployConfig.K8s != nil {
			ns := stack.deployConfig.K8s.Namespace
			if ns != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				stack.kubectl(ctx, "delete", "namespace", ns, "--ignore-not-found=true")
			}
		}
	}

	return stack, deployDir, cleanup
}

func TestE2E_DeployLocalInfra_NamespaceCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	kubeconfig := skipIfNoKind(t)

	stack, _, cleanup := setupE2EStack(t, kubeconfig)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	chainName := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	namespace := utils.ConvertChainNameToNamespace(chainName)

	// Create namespace via our wrapper
	if _, err := stack.kubectl(ctx, "create", "namespace", namespace); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	defer stack.kubectl(ctx, "delete", "namespace", namespace, "--ignore-not-found=true")

	// Verify namespace exists
	out, err := stack.kubectl(ctx, "get", "namespace", namespace, "-o", "jsonpath={.metadata.name}")
	if err != nil {
		t.Fatalf("namespace not found: %v", err)
	}
	if out != namespace {
		t.Errorf("expected namespace %s, got %s", namespace, out)
	}
}

func TestE2E_DeployLocalInfra_ConfigServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	kubeconfig := skipIfNoKind(t)

	stack, deployDir, cleanup := setupE2EStack(t, kubeconfig)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	namespace := fmt.Sprintf("e2e-cfg-%d", time.Now().Unix())
	if _, err := stack.kubectl(ctx, "create", "namespace", namespace); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	defer stack.kubectl(ctx, "delete", "namespace", namespace, "--ignore-not-found=true")

	// Prepare config files
	configFilesDir := filepath.Join(deployDir, "config-files")
	os.MkdirAll(configFilesDir, 0755)
	os.WriteFile(filepath.Join(configFilesDir, "genesis.json"), []byte(`{"test": true}`), 0644)
	os.WriteFile(filepath.Join(configFilesDir, "rollup.json"), []byte(`{"test": true}`), 0644)

	releaseName := fmt.Sprintf("e2e-%d", time.Now().Unix())
	url, err := stack.ensureConfigServer(ctx, releaseName, namespace, configFilesDir)
	if err != nil {
		t.Fatalf("ensureConfigServer failed: %v", err)
	}

	if !strings.HasPrefix(url, "http://") {
		t.Errorf("expected http:// URL, got %s", url)
	}

	// Verify deployment exists
	out, err := stack.kubectl(ctx, "get", "deployment", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		t.Fatalf("failed to get deployments: %v", err)
	}
	if !strings.Contains(out, "cfg-srv") {
		t.Errorf("config server deployment not found, got: %s", out)
	}
}

func TestE2E_DeployLocalInfra_ESOCRDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	kubeconfig := skipIfNoKind(t)

	stack, _, _ := setupE2EStack(t, kubeconfig)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := stack.ensureESOCRDs(ctx); err != nil {
		t.Fatalf("ensureESOCRDs failed: %v", err)
	}

	// Verify CRD exists
	out, err := stack.kubectl(ctx, "get", "crd", "externalsecrets.external-secrets.io", "-o", "jsonpath={.metadata.name}")
	if err != nil {
		t.Fatalf("CRD not found: %v", err)
	}
	if out != "externalsecrets.external-secrets.io" {
		t.Errorf("unexpected CRD name: %s", out)
	}
}

func TestE2E_DeployLocalInfra_PVCCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	kubeconfig := skipIfNoKind(t)

	stack, _, cleanup := setupE2EStack(t, kubeconfig)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	namespace := fmt.Sprintf("e2e-pvc-%d", time.Now().Unix())
	if _, err := stack.kubectl(ctx, "create", "namespace", namespace); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	defer stack.kubectl(ctx, "delete", "namespace", namespace, "--ignore-not-found=true")

	fullname := fmt.Sprintf("e2e-%d-thanos-stack", time.Now().Unix())
	if err := stack.ensureLocalPVCs(ctx, fullname, namespace); err != nil {
		t.Fatalf("ensureLocalPVCs failed: %v", err)
	}

	// Verify PVCs exist
	out, err := stack.kubectl(ctx, "get", "pvc", "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		t.Fatalf("failed to get PVCs: %v", err)
	}
	if !strings.Contains(out, "op-geth") || !strings.Contains(out, "op-node") {
		t.Errorf("expected PVCs for op-geth and op-node, got: %s", out)
	}

	// Cleanup PVs (cluster-scoped)
	defer func() {
		stack.kubectl(ctx, "delete", "pv", fullname+"-op-geth", "--ignore-not-found=true")
		stack.kubectl(ctx, "delete", "pv", fullname+"-op-node", "--ignore-not-found=true")
	}()
}

func TestE2E_DeployLocalInfra_SecretCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	kubeconfig := skipIfNoKind(t)

	stack, _, cleanup := setupE2EStack(t, kubeconfig)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	namespace := fmt.Sprintf("e2e-sec-%d", time.Now().Unix())
	if _, err := stack.kubectl(ctx, "create", "namespace", namespace); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	defer stack.kubectl(ctx, "delete", "namespace", namespace, "--ignore-not-found=true")

	secretName := "test-thanos-stack-secret"
	if err := stack.ensureStackSecret(ctx, secretName, namespace); err != nil {
		t.Fatalf("ensureStackSecret failed: %v", err)
	}

	// Verify secret exists with expected keys
	out, err := stack.kubectl(ctx, "get", "secret", secretName, "-n", namespace, "-o", "jsonpath={.data}")
	if err != nil {
		t.Fatalf("secret not found: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("failed to parse secret data: %v", err)
	}
	for _, key := range []string{"OP_NODE_P2P_SEQUENCER_KEY", "OP_BATCHER_PRIVATE_KEY", "OP_PROPOSER_PRIVATE_KEY"} {
		if _, ok := data[key]; !ok {
			t.Errorf("secret missing key: %s", key)
		}
	}
}

func TestE2E_DeployLocalInfra_SettingsJsonPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	kubeconfig := skipIfNoKind(t)

	stack, deployDir, _ := setupE2EStack(t, kubeconfig)

	// Simulate what DeployLocalInfrastructure does at the end: save K8s config
	stack.deployConfig.K8s = &types.K8sConfig{Namespace: "test-namespace"}
	stack.deployConfig.L2RpcUrl = "http://localhost:8545"
	stack.deployConfig.ChainName = "test-chain"

	if err := stack.deployConfig.WriteToJSONFile(deployDir); err != nil {
		t.Fatalf("WriteToJSONFile failed: %v", err)
	}

	// Re-read and verify
	reloaded, err := utils.ReadConfigFromJSONFile(deployDir)
	if err != nil {
		t.Fatalf("ReadConfigFromJSONFile failed: %v", err)
	}

	if reloaded.K8s == nil {
		t.Fatal("K8s config not persisted")
	}
	if reloaded.K8s.Namespace != "test-namespace" {
		t.Errorf("expected namespace test-namespace, got %s", reloaded.K8s.Namespace)
	}
	if reloaded.L2RpcUrl != "http://localhost:8545" {
		t.Errorf("expected L2RpcUrl http://localhost:8545, got %s", reloaded.L2RpcUrl)
	}
	if reloaded.ChainName != "test-chain" {
		t.Errorf("expected ChainName test-chain, got %s", reloaded.ChainName)
	}
}
