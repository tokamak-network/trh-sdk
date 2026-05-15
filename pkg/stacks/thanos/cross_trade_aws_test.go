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
	out, err := stack.AutoInstallCrossTradeAWS(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported L1 chain, got nil")
	}
	if out != nil {
		t.Errorf("expected nil output on error, got: %+v", out)
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
	out, err := stack.AutoInstallCrossTradeAWS(context.Background())
	if err == nil {
		t.Fatal("expected error for empty L2RpcUrl, got nil")
	}
	if out != nil {
		t.Errorf("expected nil output on error, got: %+v", out)
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
	out, err := stack.AutoInstallCrossTradeAWS(context.Background())
	if err == nil {
		t.Fatal("expected error for missing deploy-output.json, got nil")
	}
	if out != nil {
		t.Errorf("expected nil output on error, got: %+v", out)
	}
	if !strings.Contains(err.Error(), "deployed contracts") && !strings.Contains(err.Error(), "deploy-output") {
		t.Errorf("error should mention deploy-output / deployed contracts, got: %v", err)
	}
}

// TestAutoInstallCrossTradeAWS_OutputFields verifies that AutoInstallCrossTradeAWSOutput
// has all required fields for backend consumption: contract addresses and a single dApp URL.
func TestAutoInstallCrossTradeAWS_OutputFields(t *testing.T) {
	// Compile-time check: AutoInstallCrossTradeAWSOutput must have all contract/URL fields.
	var out AutoInstallCrossTradeAWSOutput
	_ = out.L2CrossTradeProxy
	_ = out.L2toL2CrossTradeProxy
	_ = out.L1CrossTradeProxy
	_ = out.L2toL2CrossTradeL1
	_ = out.DAppURL
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
// Address source: crossTrade broadcast/DeployL1CrossTrade_L2L1.s.sol/11155111 and
// broadcast/DeployL1CrossTrade_L2L2.s.sol/11155111 — deployed by admin key
// 0x7220c734653ae8Ca014d4D82A84041EE4169499c so setChainInfo succeeds.
func TestL1CrossTradeAddresses_SepoliaValues(t *testing.T) {
	const (
		wantL1CTProxy  = "0xfea37d39bec823d503ed6fb9d3a6e151190821fb"
		wantL2toL2CTL1 = "0xd038d89655f106d88c5bd56a9442d9ecee675c1c"
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

// TestCrossTradeL2L2ReleaseName verifies the deterministic L2→L2 release name format.
func TestCrossTradeL2L2ReleaseName(t *testing.T) {
	got := crossTradeL2L2ReleaseName(12345)
	want := "cross-trade-l2l2-12345"
	if got != want {
		t.Errorf("crossTradeL2L2ReleaseName(12345) = %q, want %q", got, want)
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

// --- Chain config builder tests ---

const (
	testL1ChainID = constants.EthereumSepoliaChainID
	testL2ChainID = uint64(111551187746)
	testL2RPC     = "http://l2.example.com"
	testL1RPC     = "http://l1.example.com"
	testL2Name    = "My L2"
)

// TestBuildCrossTradeL2L1Config_BothChainsPresent verifies that L1 and L2 entries exist.
func TestBuildCrossTradeL2L1Config_BothChainsPresent(t *testing.T) {
	cfg := buildCrossTradeL2L1Config(testL1ChainID, testL1RPC, "l1proxy", testL2ChainID, testL2Name, testL2RPC, "l2proxy")
	l1Key := "11155111"
	l2Key := "111551187746"
	if _, ok := cfg[l1Key]; !ok {
		t.Errorf("L1 chain key %q missing from L2→L1 config", l1Key)
	}
	if _, ok := cfg[l2Key]; !ok {
		t.Errorf("L2 chain key %q missing from L2→L1 config", l2Key)
	}
}

// TestBuildCrossTradeL2L1Config_L2TokensPointToL1 verifies L2 tokens have DestinationChains=[l1ChainID],
// which makes the L2 appear as a selectable source in the dApp's chain dropdown.
func TestBuildCrossTradeL2L1Config_L2TokensPointToL1(t *testing.T) {
	cfg := buildCrossTradeL2L1Config(testL1ChainID, testL1RPC, "l1proxy", testL2ChainID, testL2Name, testL2RPC, "l2proxy")
	l2 := cfg["111551187746"]
	if len(l2.Tokens) == 0 {
		t.Fatal("L2 chain must have at least one token in L2→L1 config")
	}
	for _, tok := range l2.Tokens {
		if len(tok.DestinationChains) == 0 {
			t.Errorf("L2 token %q: DestinationChains must not be empty (L2 is a source chain)", tok.Name)
		}
		found := false
		for _, dst := range tok.DestinationChains {
			if dst == testL1ChainID {
				found = true
			}
		}
		if !found {
			t.Errorf("L2 token %q: DestinationChains must include L1 chainID %d", tok.Name, testL1ChainID)
		}
	}
}

// TestBuildCrossTradeL2L1Config_NativeTokenFields verifies NativeTokenName/Symbol are set on both chains.
func TestBuildCrossTradeL2L1Config_NativeTokenFields(t *testing.T) {
	cfg := buildCrossTradeL2L1Config(testL1ChainID, testL1RPC, "l1proxy", testL2ChainID, testL2Name, testL2RPC, "l2proxy")
	for key, chain := range cfg {
		if chain.NativeTokenName == "" {
			t.Errorf("chain %q: NativeTokenName must not be empty", key)
		}
		if chain.NativeTokenSymbol == "" {
			t.Errorf("chain %q: NativeTokenSymbol must not be empty", key)
		}
	}
}

// TestBuildCrossTradeL2L2Config_ThanosSepolia verifies the fixed Thanos Sepolia chain entry
// is always included (chainID 111551119090). Without it the dApp has no destination for L2→L2.
func TestBuildCrossTradeL2L2Config_ThanosSepolia(t *testing.T) {
	cfg := buildCrossTradeL2L2Config(testL1ChainID, testL1RPC, "l2tol2l1", testL2ChainID, testL2Name, testL2RPC, "l2tol2proxy")
	thanosSepKey := "111551119090"
	entry, ok := cfg[thanosSepKey]
	if !ok {
		t.Fatalf("Thanos Sepolia key %q missing from L2→L2 config — dApp has no destination chain", thanosSepKey)
	}
	if entry.RPCURL != crossTradeThanosSepRPCURL {
		t.Errorf("Thanos Sepolia RPCURL: got %q, want %q", entry.RPCURL, crossTradeThanosSepRPCURL)
	}
}

// TestBuildCrossTradeL2L2Config_L2TokensPointToThanosSepolia verifies L2 tokens point to Thanos Sepolia,
// making the custom L2 appear as a source chain in L2→L2 mode.
func TestBuildCrossTradeL2L2Config_L2TokensPointToThanosSepolia(t *testing.T) {
	cfg := buildCrossTradeL2L2Config(testL1ChainID, testL1RPC, "l2tol2l1", testL2ChainID, testL2Name, testL2RPC, "l2tol2proxy")
	l2 := cfg["111551187746"]
	if len(l2.Tokens) == 0 {
		t.Fatal("L2 chain must have at least one token in L2→L2 config")
	}
	for _, tok := range l2.Tokens {
		found := false
		for _, dst := range tok.DestinationChains {
			if dst == crossTradeThanosSepolia {
				found = true
			}
		}
		if !found {
			t.Errorf("L2 token %q: DestinationChains must include Thanos Sepolia (%d)", tok.Name, crossTradeThanosSepolia)
		}
	}
}

// TestBuildCrossTradeL2L2Config_ThanosTokensEmptyDestination verifies Thanos Sepolia tokens
// have empty DestinationChains — prevents the dApp from routing Thanos→newL2 (unregistered direction).
func TestBuildCrossTradeL2L2Config_ThanosTokensEmptyDestination(t *testing.T) {
	cfg := buildCrossTradeL2L2Config(testL1ChainID, testL1RPC, "l2tol2l1", testL2ChainID, testL2Name, testL2RPC, "l2tol2proxy")
	thanos := cfg["111551119090"]
	for _, tok := range thanos.Tokens {
		if len(tok.DestinationChains) != 0 {
			t.Errorf("Thanos Sepolia token %q: DestinationChains must be empty (reverse direction not registered)", tok.Name)
		}
	}
}

// TestBuildCrossTradeL2L2Config_AllChainsHaveTokens verifies every chain entry has at least one token,
// otherwise the dApp's source/destination dropdowns will be empty.
func TestBuildCrossTradeL2L2Config_AllChainsHaveTokens(t *testing.T) {
	cfg := buildCrossTradeL2L2Config(testL1ChainID, testL1RPC, "l2tol2l1", testL2ChainID, testL2Name, testL2RPC, "l2tol2proxy")
	for key, chain := range cfg {
		if len(chain.Tokens) == 0 {
			t.Errorf("chain %q: Tokens must not be empty (dApp uses Tokens to populate dropdowns)", key)
		}
	}
}

// TestBuildCrossTradeL2L2Config_ContractAddresses verifies contract addresses are plumbed correctly.
func TestBuildCrossTradeL2L2Config_ContractAddresses(t *testing.T) {
	cfg := buildCrossTradeL2L2Config(testL1ChainID, testL1RPC, "l2tol2l1addr", testL2ChainID, testL2Name, testL2RPC, "l2tol2proxyaddr")
	l1 := cfg["11155111"]
	if l1.Contracts.L1CrossTrade == nil || *l1.Contracts.L1CrossTrade != "l2tol2l1addr" {
		t.Errorf("L1 chain: L1CrossTrade contract mismatch, got %v", l1.Contracts.L1CrossTrade)
	}
	l2 := cfg["111551187746"]
	if l2.Contracts.L2CrossTrade == nil || *l2.Contracts.L2CrossTrade != "l2tol2proxyaddr" {
		t.Errorf("L2 chain: L2CrossTrade contract mismatch, got %v", l2.Contracts.L2CrossTrade)
	}
}
