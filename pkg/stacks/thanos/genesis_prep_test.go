package thanos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteAddressesOnly_stripsNonAddressFields verifies that writeAddressesOnly
// drops metadata numbers AND nested JSON objects from tokamak-deployer's
// deploy-output.json before staging the file for forge L2Genesis.
//
// tokamak-deployer v0.0.10 added an "implementations" key whose value is a
// nested JSON object. forge's vm.parseJsonAddress iterates every key and
// reverts with "expected address, found JSON object" when it hits that field.
func TestWriteAddressesOnly_stripsNonAddressFields(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "deploy-output.json")
	dst := filepath.Join(dir, "addresses.json")

	// Mirrors tokamak-deployer v0.0.10 output structure.
	input := map[string]any{
		"l1ChainId": 11155111,
		"l2ChainId": 111551194428,
		"AddressManager":                  "0x07bb3be4b784f20503150cBBCfb65b7d0325afD4",
		"OptimismPortalProxy":             "0xE1d1d1d9D8e76F97Db95120Adeb64de7080731E4",
		"SystemConfigProxy":               "0x7b3688ec09bC077AD7A9b5fd3c2E5Bb3df3a51bB",
		"implementations": map[string]string{
			"OptimismPortal": "0x968aaf1A6010dc9d97A3dBA5d176dE7671F4abEA",
			"SystemConfig":   "0x531aC70DCa1934E8Ed870FBa52326f300C876480",
		},
	}

	raw, _ := json.Marshal(input)
	if err := os.WriteFile(src, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeAddressesOnly(src, dst); err != nil {
		t.Fatalf("writeAddressesOnly returned error: %v", err)
	}

	outRaw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(outRaw, &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Metadata numbers must be stripped.
	if _, ok := out["l1ChainId"]; ok {
		t.Error("l1ChainId should be stripped but is present")
	}
	if _, ok := out["l2ChainId"]; ok {
		t.Error("l2ChainId should be stripped but is present")
	}

	// Nested JSON object must be stripped (forge vm.parseJsonAddress rejects it).
	if _, ok := out["implementations"]; ok {
		t.Error("implementations (nested object) should be stripped but is present")
	}

	// Address strings must be preserved.
	for _, key := range []string{"AddressManager", "OptimismPortalProxy", "SystemConfigProxy"} {
		if _, ok := out[key]; !ok {
			t.Errorf("address key %q should be present but is missing", key)
		}
	}
}
