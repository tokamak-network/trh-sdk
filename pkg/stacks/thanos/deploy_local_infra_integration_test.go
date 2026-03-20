//go:build integration

package thanos

// deploy_local_infra_integration_test.go — DeployLocalInfrastructure 통합 테스트
//
// 실행:
//   kind create cluster --name trh-test --kubeconfig /tmp/trh-test.kubeconfig
//
//   KUBECONFIG=/tmp/trh-test.kubeconfig \
//   GOMODCACHE=/tmp/gomodcache \
//   go test -v -tags=integration -timeout=600s \
//       -run TestDeployLocalInfrastructure \
//       ./pkg/stacks/thanos/
//
// 전제 조건:
//   - kind 클러스터가 실행 중이어야 함 (KUBECONFIG 또는 기본 경로 /tmp/trh-test.kubeconfig)
//   - 네트워크 접근 가능 (tokamak-thanos-stack 리포 git clone 수행)
//   - rollup.json, genesis.json 픽스처 파일은 테스트 헬퍼가 생성함
//
// 각 테스트는 독립적이며, 테스트 종료 시 생성된 K8s 리소스를 정리합니다.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
)

// ─── 공통 헬퍼 ────────────────────────────────────────────────────────────────

// integrationKubeconfig returns the KUBECONFIG path.
// Skips the test when the file is not found.
func integrationKubeconfig(t *testing.T) string {
	t.Helper()
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		path = "/tmp/trh-test.kubeconfig"
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("kubeconfig 없음 (%s): kind create cluster --name trh-test --kubeconfig %s", path, path)
	}
	return path
}

// integrationLogger returns a development SugaredLogger.
func integrationLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()
	l, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("zap.NewDevelopment: %v", err)
	}
	return l.Sugar()
}

// integrationDeploymentPath creates a temp directory pre-populated with the
// minimal build artifacts that DeployLocalInfrastructure expects.
func integrationDeploymentPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	buildDir := filepath.Join(dir, "tokamak-thanos", "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("mkdir tokamak-thanos/build: %v", err)
	}

	// Minimal valid JSON fixtures — real content is not validated by the function.
	rollupJSON := []byte(`{"genesis":{"l1":{"hash":"0x0","number":0},"l2":{"hash":"0x0","number":0},"l2_time":0,"system_config":{"batcherAddr":"0x0","overhead":"0x0","scalar":"0x0","gasLimit":30000000}},"block_time":2,"max_sequencer_drift":600,"seq_window_size":3600,"channel_timeout":300,"l1_chain_id":11155111,"l2_chain_id":12345,"regolith_time":0,"batch_inbox_address":"0x0","deposit_contract_address":"0x0","l1_system_config_address":"0x0"}`)
	genesisJSON := []byte(`{"config":{"chainId":12345},"nonce":"0x0","timestamp":"0x0","gasLimit":"0x1c9c380","difficulty":"0x0","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","coinbase":"0x4200000000000000000000000000000000000011","alloc":{},"number":"0x0","gasUsed":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","baseFeePerGas":"0x3b9aca00"}`)

	if err := os.WriteFile(filepath.Join(buildDir, "rollup.json"), rollupJSON, 0644); err != nil {
		t.Fatalf("write rollup.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "genesis.json"), genesisJSON, 0644); err != nil {
		t.Fatalf("write genesis.json: %v", err)
	}

	return dir
}

// integrationStack builds a ThanosStack wired to the test kind cluster.
// Uses well-known dummy private keys (Hardhat account #0) that are safe for local testnets.
func integrationStack(t *testing.T, kubeconfigPath, deploymentPath string) *ThanosStack {
	t.Helper()
	logger := integrationLogger(t)

	tr, err := runner.New(runner.RunnerConfig{UseNative: true, KubeconfigPath: kubeconfigPath})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}

	// Hardhat account #0 private key — safe for local testnets, never used on mainnet.
	dummyKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	return &ThanosStack{
		network:        "LocalTestnet",
		logger:         logger,
		deploymentPath: deploymentPath,
		k8sRunner:      tr.K8s(),
		helmRunner:     tr.Helm(),
		deployConfig: &types.Config{
			SequencerPrivateKey: dummyKey,
			BatcherPrivateKey:   dummyKey,
			ProposerPrivateKey:  dummyKey,
			L1RPCURL:            "http://localhost:8545",
			DeployContractState: &types.DeployContractState{
				Status: types.DeployContractStatusCompleted,
			},
		},
	}
}

// cleanupDeployment uninstalls all Helm releases and deletes the namespace.
// Non-fatal: errors are logged.
func cleanupDeployment(t *testing.T, stack *ThanosStack, kubeconfigPath, namespace string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	releases, err := stack.helmList(ctx, namespace)
	if err != nil {
		t.Logf("cleanup helmList(%s): %v", namespace, err)
	}
	for _, rel := range releases {
		if err := stack.helmUninstall(ctx, rel, namespace); err != nil {
			t.Logf("cleanup helmUninstall %s/%s: %v", namespace, rel, err)
		}
	}

	// Delete namespace via K8s runner.
	if err := stack.k8sRunner.Delete(ctx, "namespaces", namespace, "", true); err != nil {
		t.Logf("cleanup delete namespace %s: %v", namespace, err)
	}
}

// ─── 1. InvalidKubeconfig テスト ─────────────────────────────────────────────

// TestDeployLocalInfrastructure_InvalidKubeconfig verifies that constructing a
// NativeRunner with a non-existent kubeconfig returns an error immediately.
// No cluster is required for this test.
func TestDeployLocalInfrastructure_InvalidKubeconfig(t *testing.T) {
	badKubeconfig := fmt.Sprintf("/tmp/trh-nonexistent-kubeconfig-%d.yaml", time.Now().UnixNano())

	_, err := runner.New(runner.RunnerConfig{
		UseNative:      true,
		KubeconfigPath: badKubeconfig,
	})
	if err == nil {
		t.Fatal("expected error for non-existent kubeconfig path, got nil")
	}
	t.Logf("correctly rejected non-existent kubeconfig: %v", err)
}

// ─── 2. Success テスト ────────────────────────────────────────────────────────

// TestDeployLocalInfrastructure_Success deploys the Thanos stack to a real kind
// cluster and verifies:
//   - The target namespace is created in the cluster.
//   - At least one Helm release is installed.
//   - deployConfig.K8s is populated with the namespace.
//   - deployConfig.L2RpcUrl is non-empty.
//
// Pre-condition: kind create cluster --name trh-test --kubeconfig /tmp/trh-test.kubeconfig
func TestDeployLocalInfrastructure_Success(t *testing.T) {
	kubeconfigPath := integrationKubeconfig(t)
	deploymentPath := integrationDeploymentPath(t)
	stack := integrationStack(t, kubeconfigPath, deploymentPath)

	inputs := &DeployLocalInfraInput{
		ChainName:   "trh-integ-success",
		L1BeaconURL: "http://localhost:5052",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := stack.DeployLocalInfrastructure(ctx, inputs); err != nil {
		t.Fatalf("DeployLocalInfrastructure failed: %v", err)
	}

	// The namespace is derived from ChainName by ConvertChainNameToNamespace.
	// The function appends a 5-char random suffix, so we use the value from config.
	namespace := stack.deployConfig.K8s.Namespace
	if namespace == "" {
		t.Fatal("deployConfig.K8s.Namespace is empty after deployment")
	}

	t.Cleanup(func() {
		cleanupDeployment(t, stack, kubeconfigPath, namespace)
	})

	// Verify namespace exists.
	exists, err := stack.k8sRunner.NamespaceExists(context.Background(), namespace)
	if err != nil {
		t.Fatalf("NamespaceExists(%s): %v", namespace, err)
	}
	if !exists {
		t.Fatalf("namespace %s not found after deployment", namespace)
	}
	t.Logf("namespace %s exists", namespace)

	// Verify L2 RPC URL.
	if stack.deployConfig.L2RpcUrl == "" {
		t.Fatal("deployConfig.L2RpcUrl is empty after deployment")
	}
	t.Logf("L2 RPC URL: %s", stack.deployConfig.L2RpcUrl)

	// Verify at least one Helm release in the namespace.
	releases, err := stack.helmList(context.Background(), namespace)
	if err != nil {
		t.Fatalf("helmList(%s): %v", namespace, err)
	}
	if len(releases) == 0 {
		t.Fatalf("no Helm releases found in namespace %s after deployment", namespace)
	}
	t.Logf("Helm releases in %s: %v", namespace, releases)
}

// ─── 3. NamespaceAlreadyExists — 멱등성 테스트 ────────────────────────────────

// TestDeployLocalInfrastructure_NamespaceAlreadyExists verifies idempotency:
// calling DeployLocalInfrastructure twice with the same ChainName must not return
// an error on the second call. The second call exercises the EnsureNamespace
// AlreadyExists-ignore path and the Helm upgrade path.
//
// Pre-condition: kind create cluster --name trh-test --kubeconfig /tmp/trh-test.kubeconfig
func TestDeployLocalInfrastructure_NamespaceAlreadyExists(t *testing.T) {
	kubeconfigPath := integrationKubeconfig(t)
	deploymentPath := integrationDeploymentPath(t)
	stack := integrationStack(t, kubeconfigPath, deploymentPath)

	inputs := &DeployLocalInfraInput{
		ChainName:   "trh-integ-idem",
		L1BeaconURL: "http://localhost:5052",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	// First deployment.
	if err := stack.DeployLocalInfrastructure(ctx, inputs); err != nil {
		t.Fatalf("first DeployLocalInfrastructure failed: %v", err)
	}

	namespace := stack.deployConfig.K8s.Namespace
	t.Cleanup(func() {
		cleanupDeployment(t, stack, kubeconfigPath, namespace)
	})

	// Second deployment — must succeed (idempotent).
	if err := stack.DeployLocalInfrastructure(ctx, inputs); err != nil {
		t.Fatalf("second DeployLocalInfrastructure (idempotency) failed: %v", err)
	}

	// Namespace must still exist.
	exists, err := stack.k8sRunner.NamespaceExists(context.Background(), namespace)
	if err != nil {
		t.Fatalf("NamespaceExists: %v", err)
	}
	if !exists {
		t.Fatalf("namespace %s missing after idempotent deployment", namespace)
	}
	t.Logf("idempotent deployment verified: namespace %s", namespace)
}

// ─── 4. EnsureNamespace 직접 검증 ─────────────────────────────────────────────

// TestDeployLocalInfrastructure_EnsureNamespaceCalledOnRunner is a focused
// integration test that verifies EnsureNamespace creates a real namespace in the
// kind cluster and that a second call is idempotent. It does not perform a full
// Helm deployment, isolating the k8s namespace-creation step.
//
// Pre-condition: kind create cluster --name trh-test --kubeconfig /tmp/trh-test.kubeconfig
func TestDeployLocalInfrastructure_EnsureNamespaceCalledOnRunner(t *testing.T) {
	kubeconfigPath := integrationKubeconfig(t)

	tr, err := runner.New(runner.RunnerConfig{UseNative: true, KubeconfigPath: kubeconfigPath})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	k8sRunner := tr.K8s()
	ctx := context.Background()

	namespace := fmt.Sprintf("trh-ns-check-%d", time.Now().Unix())
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		k8sRunner.Delete(cleanCtx, "namespaces", namespace, "", true) //nolint:errcheck
	})

	// First call — should create the namespace.
	if err := k8sRunner.EnsureNamespace(ctx, namespace); err != nil {
		t.Fatalf("EnsureNamespace(%s): %v", namespace, err)
	}

	exists, err := k8sRunner.NamespaceExists(ctx, namespace)
	if err != nil {
		t.Fatalf("NamespaceExists: %v", err)
	}
	if !exists {
		t.Fatalf("namespace %s not found after EnsureNamespace", namespace)
	}

	// Second call — must be idempotent (AlreadyExists is ignored).
	if err := k8sRunner.EnsureNamespace(ctx, namespace); err != nil {
		t.Fatalf("second EnsureNamespace (idempotency): %v", err)
	}
	t.Logf("EnsureNamespace idempotency verified for %s", namespace)
}

// Ensure utils is used (ConvertChainNameToNamespace is called indirectly through the stack).
var _ = utils.ConvertChainNameToNamespace
