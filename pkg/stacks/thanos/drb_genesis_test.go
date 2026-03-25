package thanos

import (
	"encoding/json"
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

	result, err := deployDRBSimulated(simpleCreation)
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
