package thanos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestInjectL1BlockBytecode(t *testing.T) {
	// Simulate old Ecotone-only bytecode (no setL1BlockValuesIsthmus selector)
	oldCode := "0x608060405260043610610041576000357c010000000000000000000000000000000000000000000000000000000090048063440a5e2014610046575b600080fd"
	// Simulate correct Isthmus-capable bytecode (contains 098999be selector)
	newCode := "0x608060405260043610610041576000357c010000000000000000000000000000000000000000000000000000000090048063098999be14610046575b600080fd"

	// Compute expected code namespace address for L1Block proxy
	proxyAddr := common.HexToAddress(l1BlockProxyAddress)
	codeAddr := predeployToCodeNamespace(proxyAddr)
	codeAddrKey := strings.ToLower(codeAddr.Hex())

	// Build minimal genesis.json: L1Block implementation at code namespace with old bytecode,
	// a separate storage slot to verify preservation, and an unrelated alloc entry.
	genesis := map[string]interface{}{
		"config": map[string]interface{}{"chainId": 111551180224},
		"alloc": map[string]interface{}{
			codeAddrKey: map[string]interface{}{
				"code":    oldCode,
				"balance": "0x0",
				"storage": map[string]string{
					"0x0000000000000000000000000000000000000000000000000000000000000000": "0x0000000000000000000000000000000000000000000000000000000000000001",
				},
			},
			"0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef": map[string]interface{}{
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

	// Build fake forge artifact tree: tokamak-thanos/packages/tokamak/contracts-bedrock/forge-artifacts/L1Block.sol/L1Block.json
	artifactDir := filepath.Join(tmpDir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "forge-artifacts", "L1Block.sol")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifact := map[string]interface{}{
		"deployedBytecode": map[string]interface{}{
			"object": newCode,
		},
	}
	artifactJSON, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "L1Block.json"), artifactJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// Run the function under test
	if err := injectL1BlockBytecode(genesisPath, tmpDir); err != nil {
		t.Fatalf("injectL1BlockBytecode failed: %v", err)
	}

	// Read back patched genesis
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

	// Verify code at implementation address is updated to new bytecode
	implRaw, ok := alloc[codeAddrKey]
	if !ok {
		t.Fatalf("implementation entry missing at %s after patch", codeAddrKey)
	}
	var implEntry map[string]json.RawMessage
	if err := json.Unmarshal(implRaw, &implEntry); err != nil {
		t.Fatal(err)
	}
	var gotCode string
	if err := json.Unmarshal(implEntry["code"], &gotCode); err != nil {
		t.Fatal(err)
	}
	if gotCode != newCode {
		t.Errorf("code not updated: got %s, want %s", gotCode, newCode)
	}

	// Verify existing storage slot is preserved (not wiped by the patch)
	if implEntry["storage"] == nil {
		t.Error("storage field was lost during patch")
	}
	var storage map[string]string
	if err := json.Unmarshal(implEntry["storage"], &storage); err != nil {
		t.Fatal(err)
	}
	slotZero := "0x0000000000000000000000000000000000000000000000000000000000000000"
	if storage[slotZero] != "0x0000000000000000000000000000000000000000000000000000000000000001" {
		t.Errorf("storage slot 0 lost: got %s", storage[slotZero])
	}

	// Verify balance field is preserved
	if implEntry["balance"] == nil {
		t.Error("balance field was lost during patch")
	}

	// Verify unrelated alloc entry is not disturbed
	if _, ok := alloc["0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"]; !ok {
		t.Error("unrelated alloc entry was removed during patch")
	}

	// Verify top-level config is preserved
	if patched["config"] == nil {
		t.Error("config section lost during patch")
	}
}

func TestInjectL1BlockBytecodeIdempotent(t *testing.T) {
	// Calling injectL1BlockBytecode twice should leave the file in the same state as calling it once.
	proxyAddr := common.HexToAddress(l1BlockProxyAddress)
	codeAddr := predeployToCodeNamespace(proxyAddr)
	codeAddrKey := strings.ToLower(codeAddr.Hex())

	correctCode := "0x098999be_fake_isthmus_bytecode"

	genesis := map[string]interface{}{
		"alloc": map[string]interface{}{
			codeAddrKey: map[string]interface{}{
				"code":    "0xold_ecotone_only",
				"balance": "0x0",
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

	artifactDir := filepath.Join(tmpDir, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "forge-artifacts", "L1Block.sol")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifact := map[string]interface{}{
		"deployedBytecode": map[string]interface{}{"object": correctCode},
	}
	artifactJSON, _ := json.MarshalIndent(artifact, "", "  ")
	if err := os.WriteFile(filepath.Join(artifactDir, "L1Block.json"), artifactJSON, 0644); err != nil {
		t.Fatal(err)
	}

	if err := injectL1BlockBytecode(genesisPath, tmpDir); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	firstResult, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := injectL1BlockBytecode(genesisPath, tmpDir); err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	secondResult, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(firstResult) != string(secondResult) {
		t.Error("injectL1BlockBytecode is not idempotent: file differs after second call")
	}
}
