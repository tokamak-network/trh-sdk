package thanos

import (
	"context"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

func makeCrossTradeStack(t *testing.T, l1ChainID uint64, l2RpcUrl string) *ThanosStack {
	t.Helper()
	return &ThanosStack{
		deployConfig: &types.Config{
			L1ChainID: l1ChainID,
			L2ChainID: 12345,
			L2RpcUrl:  l2RpcUrl,
			L1RPCURL:  "http://localhost:8545",
			K8s:       &types.K8sConfig{Namespace: "test-ns"},
		},
		deploymentPath: t.TempDir(),
		logger:         zap.NewNop().Sugar(),
	}
}

// --- Guard condition tests ---

// TestAutoInstallCrossTradeAWS_UnsupportedL1Chain verifies that unsupported L1 chains
// return a clear error pointing to manual setup.
func TestAutoInstallCrossTradeAWS_UnsupportedL1Chain(t *testing.T) {
	stack := makeCrossTradeStack(t, 1234, "http://localhost:9545")
	err := stack.autoInstallCrossTradeAWS(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported L1 chain, got nil")
	}
	if !strings.Contains(err.Error(), "1234") {
		t.Errorf("error should mention unsupported chain ID 1234, got: %v", err)
	}
	if !strings.Contains(err.Error(), "manual") {
		t.Errorf("error should hint at manual setup, got: %v", err)
	}
}

// TestAutoInstallCrossTradeAWS_L2RpcUrlEmpty verifies that an empty L2RpcUrl is
// rejected before any deployment attempt.
func TestAutoInstallCrossTradeAWS_L2RpcUrlEmpty(t *testing.T) {
	stack := makeCrossTradeStack(t, constants.EthereumSepoliaChainID, "")
	err := stack.autoInstallCrossTradeAWS(context.Background())
	if err == nil {
		t.Fatal("expected error for empty L2RpcUrl, got nil")
	}
	if !strings.Contains(err.Error(), "L2RpcUrl") {
		t.Errorf("error should mention L2RpcUrl, got: %v", err)
	}
}

// TestAutoInstallCrossTradeAWS_MissingDeployOutput verifies that a missing
// deploy-output.json is caught and reported before any contract deployment.
func TestAutoInstallCrossTradeAWS_MissingDeployOutput(t *testing.T) {
	stack := makeCrossTradeStack(t, constants.EthereumSepoliaChainID, "http://localhost:9545")
	// deploymentPath is a TempDir with no deploy-output.json — read should fail.
	err := stack.autoInstallCrossTradeAWS(context.Background())
	if err == nil {
		t.Fatal("expected error for missing deploy-output.json, got nil")
	}
	if !strings.Contains(err.Error(), "deployed contracts") && !strings.Contains(err.Error(), "deploy-output") {
		t.Errorf("error should mention deploy-output / deployed contracts, got: %v", err)
	}
}

// --- Address regression tests ---

// TestL1CrossTradeAddresses_SepoliaExists verifies that Sepolia is registered
// in l1CrossTradeAddresses (required for all testnet deployments).
func TestL1CrossTradeAddresses_SepoliaExists(t *testing.T) {
	addrs, ok := l1CrossTradeAddresses[constants.EthereumSepoliaChainID]
	if !ok {
		t.Fatalf("l1CrossTradeAddresses missing entry for Sepolia (chainID=%d)", constants.EthereumSepoliaChainID)
	}
	if addrs.L1CrossTradeProxy == "" {
		t.Error("L1CrossTradeProxy address must not be empty for Sepolia")
	}
	if addrs.L2toL2CrossTradeL1 == "" {
		t.Error("L2toL2CrossTradeL1 address must not be empty for Sepolia")
	}
}

// TestL1CrossTradeAddresses_SepoliaValues is a regression test that pins the
// live-verified contract addresses for Sepolia. If these change, the AWS
// auto-install will reference the wrong contracts — this test will catch it.
//
// Address source: live test verified in cross_trade_local_live_test.go (April 2026).
func TestL1CrossTradeAddresses_SepoliaValues(t *testing.T) {
	const (
		wantL1CTProxy  = "0xf3473E20F1d9EB4468C72454a27aA1C65B67AB35"
		wantL2toL2CTL1 = "0xDa2CbF69352cB46d9816dF934402b421d93b6BC2"
	)

	addrs := l1CrossTradeAddresses[constants.EthereumSepoliaChainID]

	if !strings.EqualFold(addrs.L1CrossTradeProxy, wantL1CTProxy) {
		t.Errorf("L1CrossTradeProxy mismatch:\n  got  %s\n  want %s", addrs.L1CrossTradeProxy, wantL1CTProxy)
	}
	if !strings.EqualFold(addrs.L2toL2CrossTradeL1, wantL2toL2CTL1) {
		t.Errorf("L2toL2CrossTradeL1 mismatch:\n  got  %s\n  want %s", addrs.L2toL2CrossTradeL1, wantL2toL2CTL1)
	}
}

// TestCrossTradeReleaseName verifies the deterministic release name format.
func TestCrossTradeReleaseName(t *testing.T) {
	got := crossTradeReleaseName(12345)
	want := "cross-trade-12345"
	if got != want {
		t.Errorf("crossTradeReleaseName(12345) = %q, want %q", got, want)
	}
}

// TestUninstallCrossTradeAWS_NoK8s verifies that when K8s config is nil
// (no cluster deployed), UninstallCrossTradeAWS returns nil without errors.
// This is required by the best-effort destroy strategy — missing features are not errors.
func TestUninstallCrossTradeAWS_NoK8s(t *testing.T) {
	stack := &ThanosStack{
		deployConfig:   &types.Config{L2ChainID: 12345},
		logger:         zap.NewNop().Sugar(),
		deploymentPath: t.TempDir(),
	}
	err := stack.UninstallCrossTradeAWS(context.Background())
	if err != nil {
		t.Errorf("expected nil when K8s is nil, got: %v", err)
	}
}
