package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestPredeployToCodeNamespace(t *testing.T) {
	tests := []struct {
		name     string
		addr     common.Address
		expected common.Address
	}{
		{
			name:     "DRB predeploy 0x60",
			addr:     common.HexToAddress("0x4200000000000000000000000000000000000060"),
			expected: common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30060"),
		},
		{
			name:     "zero suffix",
			addr:     common.HexToAddress("0x4200000000000000000000000000000000000000"),
			expected: common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000"),
		},
		{
			name:     "L2CrossDomainMessenger 0x07",
			addr:     common.HexToAddress("0x4200000000000000000000000000000000000007"),
			expected: common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30007"),
		},
		{
			name:     "max predeploy 0x07FF",
			addr:     common.HexToAddress("0x42000000000000000000000000000000000007FF"),
			expected: common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d307FF"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := predeployToCodeNamespace(tc.addr)
			if result != tc.expected {
				t.Errorf("got %s, want %s", result.Hex(), tc.expected.Hex())
			}
		})
	}
}

func TestPatchGenesisWithDRB(t *testing.T) {
	// Create a minimal genesis.json with the proxy already present
	proxyAddr := "0x4200000000000000000000000000000000000060"
	adminSlot := "0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"
	proxyAdminAddr := "0x0000000000000000000000004200000000000000000000000000000000000018"

	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId": 12345,
		},
		"alloc": map[string]interface{}{
			proxyAddr: map[string]interface{}{
				"code":    "0x608060405234801561001057600080fd5b50",
				"balance": "0x0",
				"storage": map[string]string{
					adminSlot: proxyAdminAddr,
				},
			},
			// Some other account
			"0x1234567890abcdef1234567890abcdef12345678": map[string]interface{}{
				"balance": "0x1000",
			},
		},
	}

	genesisJSON, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	if err := os.WriteFile(genesisPath, genesisJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// Fake runtime bytecode
	fakeBytecode := []byte{0x60, 0x80, 0x60, 0x40, 0x52}

	// Patch genesis
	if err := patchGenesisWithDRB(genesisPath, fakeBytecode); err != nil {
		t.Fatalf("patchGenesisWithDRB failed: %v", err)
	}

	// Read back and verify
	patchedData, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}

	var patched map[string]json.RawMessage
	if err := json.Unmarshal(patchedData, &patched); err != nil {
		t.Fatal(err)
	}

	var alloc map[string]json.RawMessage
	if err := json.Unmarshal(patched["alloc"], &alloc); err != nil {
		t.Fatal(err)
	}

	// Verify code-namespace address has implementation bytecode
	codeAddr := predeployToCodeNamespace(common.HexToAddress(proxyAddr))
	codeAddrHex := strings.ToLower(codeAddr.Hex())

	implRaw, ok := alloc[codeAddrHex]
	if !ok {
		t.Fatalf("implementation not found at code-namespace address %s, alloc keys: %v", codeAddrHex, allocKeys(alloc))
	}

	var implData map[string]interface{}
	if err := json.Unmarshal(implRaw, &implData); err != nil {
		t.Fatal(err)
	}
	if implData["code"] != "0x6080604052" {
		t.Errorf("unexpected implementation code: %s", implData["code"])
	}

	// Verify proxy still has its original fields plus new implementation slot
	proxyRaw, ok := alloc[proxyAddr]
	if !ok {
		t.Fatal("proxy alloc entry missing after patch")
	}

	var proxyData map[string]json.RawMessage
	if err := json.Unmarshal(proxyRaw, &proxyData); err != nil {
		t.Fatal(err)
	}

	var storage map[string]string
	if err := json.Unmarshal(proxyData["storage"], &storage); err != nil {
		t.Fatal(err)
	}

	// Admin slot should be preserved
	if storage[adminSlot] != proxyAdminAddr {
		t.Errorf("admin slot lost: got %s, want %s", storage[adminSlot], proxyAdminAddr)
	}

	// Implementation slot should be set
	implSlotVal, ok := storage[erc1967ImplementationSlot]
	if !ok {
		t.Fatal("implementation slot not set")
	}
	expectedImplSlot := common.BytesToHash(codeAddr.Bytes()).Hex()
	if implSlotVal != expectedImplSlot {
		t.Errorf("implementation slot: got %s, want %s", implSlotVal, expectedImplSlot)
	}

	// Verify other alloc entries are preserved
	if _, ok := alloc["0x1234567890abcdef1234567890abcdef12345678"]; !ok {
		t.Error("other alloc entry was lost during patch")
	}

	// Verify config section is preserved
	if patched["config"] == nil {
		t.Error("config section was lost during patch")
	}
}

func TestDeployDRBSimulated(t *testing.T) {
	// This test requires the actual artifact bytecode.
	// Use a simple contract as smoke test: contract that returns 0x42.
	// PUSH1 0x42 PUSH1 0x00 MSTORE PUSH1 0x20 PUSH1 0x00 RETURN
	// = 60 42 60 00 52 60 20 60 00 f3
	simpleCreation := []byte{0x60, 0x42, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}

	result, err := deployDRBSimulated(simpleCreation, big.NewInt(0))
	if err != nil {
		t.Fatalf("deployDRBSimulated failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("deployDRBSimulated returned empty bytecode")
	}
}

func TestDefaultDRBGenesisConfig(t *testing.T) {
	config := DefaultDRBGenesisConfig()
	if config.ActivationThreshold.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("unexpected ActivationThreshold: %s", config.ActivationThreshold)
	}
	if config.Name != "Commit-Reveal2" {
		t.Errorf("unexpected Name: %s", config.Name)
	}
	if config.Version != "1" {
		t.Errorf("unexpected Version: %s", config.Version)
	}
}

func allocKeys(alloc map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(alloc))
	for k := range alloc {
		keys = append(keys, k)
	}
	return keys
}

// Test 1: Gaming preset calls fetcher
func TestMaybeInjectDRB_GamingPreset_CallsFetcher(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")

	// Create minimal genesis
	copyFixture(t, genesisPath)

	// Mock fetcher
	called := false
	mockFetcher := &mockFetcher{
		fn: func(c context.Context, pkg, tag string) ([]byte, error) {
			called = true
			// Return minimal valid artifact JSON with empty abi and bytecode
			return []byte(`{"abi": [], "bytecode": {"object": "0x6080"}}`), nil
		},
	}

	mockLogger := &mockLogger{}
	_ = maybeInjectDRB(ctx, mockLogger, genesisPath, "gaming", mockFetcher)

	// Even if there's an error in artifact parsing, we want to verify fetcher was called
	if !called {
		t.Error("fetcher.Fetch not called for gaming preset")
	}
}

// Test 2: General preset skips injection
func TestMaybeInjectDRB_GeneralPreset_SkipsInjection(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	called := false
	mockFetcher := &mockFetcher{
		fn: func(c context.Context, pkg, tag string) ([]byte, error) {
			called = true
			return nil, nil
		},
	}

	err := maybeInjectDRB(ctx, &mockLogger{}, genesisPath, "general", mockFetcher)

	if err != nil {
		t.Fatalf("maybeInjectDRB failed: %v", err)
	}
	if called {
		t.Error("fetcher.Fetch should not be called for general preset")
	}
}

// Test 3: DeFi preset skips injection
func TestMaybeInjectDRB_DeFiPreset_SkipsInjection(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	called := false
	mockFetcher := &mockFetcher{
		fn: func(c context.Context, pkg, tag string) ([]byte, error) {
			called = true
			return nil, nil
		},
	}

	err := maybeInjectDRB(ctx, &mockLogger{}, genesisPath, "defi", mockFetcher)

	if err != nil {
		t.Fatalf("maybeInjectDRB failed: %v", err)
	}
	if called {
		t.Error("fetcher.Fetch should not be called for defi preset")
	}
}

// Test 4: Fetcher error propagates
func TestMaybeInjectDRB_FetcherError_ReturnsWrappedError(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	mockFetcher := &mockFetcher{
		fn: func(c context.Context, pkg, tag string) ([]byte, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	err := maybeInjectDRB(ctx, &mockLogger{}, genesisPath, "gaming", mockFetcher)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to download DRB artifact") {
		t.Errorf("error message missing context: %v", err)
	}
}

// Test 5: Genesis patch includes proxy + impl
func TestPatchGenesisWithDRB_WritesProxyAndImpl(t *testing.T) {
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	// Use real patchGenesisWithDRB with fake bytecode
	fakeBytecode := []byte{0x60, 0x80, 0x60, 0x40, 0x52}
	err := patchGenesisWithDRB(genesisPath, fakeBytecode)
	if err != nil {
		t.Fatalf("patchGenesisWithDRB failed: %v", err)
	}

	// Read back and verify
	data, _ := os.ReadFile(genesisPath)
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var alloc map[string]json.RawMessage
	json.Unmarshal(parsed["alloc"], &alloc)

	proxyAddr := "0x4200000000000000000000000000000000000060"
	if _, ok := alloc[proxyAddr]; !ok {
		t.Errorf("proxy address not found in alloc")
	}

	implAddr := predeployToCodeNamespace(common.HexToAddress(proxyAddr))
	implAddrHex := strings.ToLower(implAddr.Hex())
	if _, ok := alloc[implAddrHex]; !ok {
		t.Errorf("impl address not found in alloc: %s", implAddrHex)
	}
}

// Test 6: ERC1967 slot set correctly
func TestPatchGenesisWithDRB_SetsERC1967Slot(t *testing.T) {
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	fakeBytecode := []byte{0x60, 0x80, 0x60, 0x40, 0x52}
	err := patchGenesisWithDRB(genesisPath, fakeBytecode)
	if err != nil {
		t.Fatalf("patchGenesisWithDRB failed: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var alloc map[string]json.RawMessage
	json.Unmarshal(parsed["alloc"], &alloc)

	proxyAddr := "0x4200000000000000000000000000000000000060"
	var proxyData map[string]json.RawMessage
	json.Unmarshal(alloc[proxyAddr], &proxyData)
	var storage map[string]string
	json.Unmarshal(proxyData["storage"], &storage)

	implSlotVal, ok := storage[erc1967ImplementationSlot]
	if !ok {
		t.Error("ERC1967 implementation slot not set in proxy storage")
	}

	implAddr := predeployToCodeNamespace(common.HexToAddress(proxyAddr))
	expectedSlot := common.BytesToHash(implAddr.Bytes()).Hex()
	if implSlotVal != expectedSlot {
		t.Errorf("implementation slot value mismatch: got %s, want %s", implSlotVal, expectedSlot)
	}
}

// Helper: copy fixture
func copyFixture(t *testing.T, dest string) {
	fixtureData := []byte(`{
  "config": {"chainId": 12345},
  "alloc": {
    "0x4200000000000000000000000000000000000060": {
      "code": "0x608060405234801561001057600080fd5b50",
      "balance": "0x0"
    }
  }
}`)
	os.WriteFile(dest, fixtureData, 0644)
}

// Mock fetcher
type mockFetcher struct {
	fn func(context.Context, string, string) ([]byte, error)
}

func (m *mockFetcher) Fetch(ctx context.Context, pkg, tag string) ([]byte, error) {
	return m.fn(ctx, pkg, tag)
}

// Mock logger
type mockLogger struct{}

func (ml *mockLogger) Info(args ...interface{}) {}

// Test 7: Regular balance allocation (Phase 7-02 Wave 1 RED)
func TestPatchGenesisWithDRB_IncludesRegularBalance(t *testing.T) {
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	err := maybeFundDRBRegulars(genesisPath, "gaming", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	if err != nil {
		t.Fatalf("maybeFundDRBRegulars failed: %v", err)
	}

	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatalf("read genesis: %v", err)
	}

	var genesis map[string]json.RawMessage
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatalf("unmarshal genesis: %v", err)
	}

	var alloc map[string]json.RawMessage
	if err := json.Unmarshal(genesis["alloc"], &alloc); err != nil {
		t.Fatalf("unmarshal alloc: %v", err)
	}

	accounts, err := DeriveDRBAccounts("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	if err != nil {
		t.Fatalf("derive accounts: %v", err)
	}

	for _, regular := range accounts.Regulars {
		raw, ok := alloc[strings.ToLower(regular.Address.Hex())]
		if !ok {
			t.Fatalf("regular operator alloc missing for %s", regular.Address.Hex())
		}

		var entry map[string]string
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Fatalf("unmarshal regular alloc: %v", err)
		}
		if entry["balance"] != "0xde0b6b3a7640000" {
			t.Fatalf("regular operator balance = %s, want 0xde0b6b3a7640000", entry["balance"])
		}
	}
}

// Test 8: resolveDRBNpmTag defaults to constant when env var not set
func TestResolveDRBNpmTag_DefaultsToConstant(t *testing.T) {
	// Ensure env var is NOT set
	t.Setenv("DRB_CONTRACTS_VERSION", "")

	result := resolveDRBNpmTag()

	if result != drbNpmTag {
		t.Errorf("expected %q, got %q", drbNpmTag, result)
	}
}

// Test 9: resolveDRBNpmTag is overridden by env var
func TestResolveDRBNpmTag_OverriddenByEnv(t *testing.T) {
	// Set env var to override default
	t.Setenv("DRB_CONTRACTS_VERSION", "2.0.0")

	result := resolveDRBNpmTag()

	if result != "2.0.0" {
		t.Errorf("expected %q, got %q", "2.0.0", result)
	}
}
