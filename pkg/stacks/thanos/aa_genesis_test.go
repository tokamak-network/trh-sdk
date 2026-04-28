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
