package thanos

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// test private key (abandon×11 about, BIP44 index 0)
const testAdminPrivKey = "1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727"

func TestMaybeFundAAAdmin_SkipsForTONFeeToken(t *testing.T) {
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	beforeData, _ := os.ReadFile(genesisPath)

	if err := maybeFundAAAdmin(genesisPath, constants.PresetGeneral, constants.FeeTokenTON, testAdminPrivKey); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	afterData, _ := os.ReadFile(genesisPath)
	if string(beforeData) != string(afterData) {
		t.Error("genesis should be unchanged for TON fee token")
	}
}

func TestMaybeFundAAAdmin_PatchesGenesis_WhenNeedsAASetup(t *testing.T) {
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	if err := maybeFundAAAdmin(genesisPath, constants.PresetGeneral, "USDT", testAdminPrivKey); err != nil {
		t.Fatalf("maybeFundAAAdmin failed: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var genesis map[string]json.RawMessage
	json.Unmarshal(data, &genesis)
	var alloc map[string]json.RawMessage
	json.Unmarshal(genesis["alloc"], &alloc)

	adminAddr, err := utils.GetAddressFromPrivateKey(testAdminPrivKey)
	if err != nil {
		t.Fatalf("derive admin addr: %v", err)
	}

	addrHex := strings.ToLower(adminAddr.Hex())
	raw, ok := alloc[addrHex]
	if !ok {
		t.Fatalf("admin address %s not found in alloc", addrHex)
	}

	var entry map[string]string
	json.Unmarshal(raw, &entry)

	// Expected: 2 × DefaultEntryPointDeposit = 2e18 = 0x1bc16d674ec80000
	expected := "0x" + new(big.Int).Mul(constants.DefaultEntryPointDeposit, big.NewInt(2)).Text(16)
	if entry["balance"] != expected {
		t.Errorf("admin balance = %s, want %s", entry["balance"], expected)
	}
}

func TestMaybeFundAAAdmin_UpdatesExistingAllocEntry(t *testing.T) {
	adminAddr, err := utils.GetAddressFromPrivateKey(testAdminPrivKey)
	if err != nil {
		t.Fatalf("derive admin addr: %v", err)
	}
	addrHex := strings.ToLower(adminAddr.Hex())

	genesis := map[string]interface{}{
		"config": map[string]interface{}{"chainId": 12345},
		"alloc": map[string]interface{}{
			addrHex: map[string]interface{}{
				"balance": "0x1000",
				"nonce":   "0x1",
			},
		},
	}
	genesisJSON, _ := json.MarshalIndent(genesis, "", "  ")
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	os.WriteFile(genesisPath, genesisJSON, 0644)

	if err := maybeFundAAAdmin(genesisPath, constants.PresetFull, "USDC", testAdminPrivKey); err != nil {
		t.Fatalf("maybeFundAAAdmin failed: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var alloc map[string]json.RawMessage
	json.Unmarshal(parsed["alloc"], &alloc)

	raw, ok := alloc[addrHex]
	if !ok {
		t.Fatalf("admin alloc entry missing after update")
	}

	var entry map[string]json.RawMessage
	json.Unmarshal(raw, &entry)

	// balance should be updated
	var balance string
	json.Unmarshal(entry["balance"], &balance)
	expected := "0x" + new(big.Int).Mul(constants.DefaultEntryPointDeposit, big.NewInt(2)).Text(16)
	if balance != expected {
		t.Errorf("balance = %s, want %s", balance, expected)
	}

	// nonce should be preserved
	var nonce string
	json.Unmarshal(entry["nonce"], &nonce)
	if nonce != "0x1" {
		t.Errorf("nonce lost: got %s, want 0x1", nonce)
	}
}

func TestMaybeFundAAAdmin_PreservesOtherAllocEntries(t *testing.T) {
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	copyFixture(t, genesisPath)

	if err := maybeFundAAAdmin(genesisPath, constants.PresetDeFi, "ETH", testAdminPrivKey); err != nil {
		t.Fatalf("maybeFundAAAdmin failed: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var genesis map[string]json.RawMessage
	json.Unmarshal(data, &genesis)
	var alloc map[string]json.RawMessage
	json.Unmarshal(genesis["alloc"], &alloc)

	// copyFixture includes 0x4200...0060 — must still be there
	if _, ok := alloc["0x4200000000000000000000000000000000000060"]; !ok {
		t.Error("pre-existing alloc entry was removed after AA admin funding patch")
	}
}

func TestMaybeFundAAAdmin_HandleChecksumCaseKey(t *testing.T) {
	adminAddr, err := utils.GetAddressFromPrivateKey(testAdminPrivKey)
	if err != nil {
		t.Fatalf("derive admin addr: %v", err)
	}
	// Use checksum (EIP-55 mixed-case) address as existing alloc key
	checksumKey := adminAddr.Hex() // go-ethereum Hex() returns checksum address

	genesis := map[string]interface{}{
		"config": map[string]interface{}{"chainId": 12345},
		"alloc": map[string]interface{}{
			checksumKey: map[string]interface{}{
				"balance": "0x1000",
				"nonce":   "0x2",
			},
		},
	}
	genesisJSON, _ := json.MarshalIndent(genesis, "", "  ")
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	os.WriteFile(genesisPath, genesisJSON, 0644)

	if err := maybeFundAAAdmin(genesisPath, constants.PresetGeneral, "ETH", testAdminPrivKey); err != nil {
		t.Fatalf("maybeFundAAAdmin failed: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var alloc map[string]json.RawMessage
	json.Unmarshal(parsed["alloc"], &alloc)

	// Must NOT create a second entry — only one key for this address
	count := 0
	for key := range alloc {
		normalized := strings.ToLower(strings.TrimPrefix(key, "0x"))
		if normalized == strings.ToLower(strings.TrimPrefix(adminAddr.Hex(), "0x")) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 alloc entry for admin address, got %d", count)
	}

	// Existing key should be updated, not a new lowercase key added
	raw, ok := alloc[checksumKey]
	if !ok {
		t.Fatalf("original checksum key %s missing; got keys: %v", checksumKey, allocKeys(alloc))
	}

	var entry map[string]json.RawMessage
	json.Unmarshal(raw, &entry)
	var nonce string
	json.Unmarshal(entry["nonce"], &nonce)
	if nonce != "0x2" {
		t.Errorf("nonce lost or wrong: got %s", nonce)
	}
}

func TestMaybeFundAAAdmin_DoesNotLowerExistingBalance(t *testing.T) {
	adminAddr, err := utils.GetAddressFromPrivateKey(testAdminPrivKey)
	if err != nil {
		t.Fatalf("derive admin addr: %v", err)
	}
	addrHex := strings.ToLower(adminAddr.Hex())

	// Set balance higher than 2e18
	higherBalance := new(big.Int).Mul(constants.DefaultEntryPointDeposit, big.NewInt(10)) // 10e18
	genesis := map[string]interface{}{
		"config": map[string]interface{}{"chainId": 12345},
		"alloc": map[string]interface{}{
			addrHex: map[string]interface{}{
				"balance": "0x" + higherBalance.Text(16),
			},
		},
	}
	genesisJSON, _ := json.MarshalIndent(genesis, "", "  ")
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	os.WriteFile(genesisPath, genesisJSON, 0644)

	if err := maybeFundAAAdmin(genesisPath, constants.PresetFull, "USDT", testAdminPrivKey); err != nil {
		t.Fatalf("maybeFundAAAdmin failed: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var alloc map[string]json.RawMessage
	json.Unmarshal(parsed["alloc"], &alloc)

	var entry map[string]json.RawMessage
	json.Unmarshal(alloc[addrHex], &entry)
	var balance string
	json.Unmarshal(entry["balance"], &balance)

	expected := "0x" + higherBalance.Text(16)
	if balance != expected {
		t.Errorf("balance was lowered: got %s, want %s (should preserve higher balance)", balance, expected)
	}
}

func TestPatchGenesisWithAAAdminFunding_EmptyAlloc(t *testing.T) {
	adminAddr, err := utils.GetAddressFromPrivateKey(testAdminPrivKey)
	if err != nil {
		t.Fatalf("derive admin addr: %v", err)
	}

	genesis := map[string]interface{}{
		"config": map[string]interface{}{"chainId": 12345},
		"alloc":  map[string]interface{}{},
	}
	genesisJSON, _ := json.MarshalIndent(genesis, "", "  ")
	tmpDir := t.TempDir()
	genesisPath := filepath.Join(tmpDir, "genesis.json")
	os.WriteFile(genesisPath, genesisJSON, 0644)

	// Must not panic on empty alloc
	if err := patchGenesisWithAAAdminFunding(genesisPath, adminAddr); err != nil {
		t.Fatalf("patchGenesisWithAAAdminFunding failed on empty alloc: %v", err)
	}

	data, _ := os.ReadFile(genesisPath)
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var alloc map[string]json.RawMessage
	json.Unmarshal(parsed["alloc"], &alloc)

	if len(alloc) != 1 {
		t.Errorf("expected 1 alloc entry, got %d", len(alloc))
	}
}
