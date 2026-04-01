package thanos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClearAnchorStateRegistryFromAddressFile(t *testing.T) {
	writeAddressFile := func(t *testing.T, dir string, content map[string]string) {
		t.Helper()
		deploymentsDir := filepath.Join(dir, "deployments", "thanos-stack-sepolia")
		if err := os.MkdirAll(deploymentsDir, 0755); err != nil {
			t.Fatal(err)
		}
		raw := make(map[string]json.RawMessage, len(content))
		for k, v := range content {
			b, _ := json.Marshal(v)
			raw[k] = b
		}
		data, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(deploymentsDir, "address.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	readAddressFile := func(t *testing.T, dir string) map[string]json.RawMessage {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "deployments", "thanos-stack-sepolia", "address.json"))
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}

	t.Run("removes AnchorStateRegistry key and returns true", func(t *testing.T) {
		dir := t.TempDir()
		writeAddressFile(t, dir, map[string]string{
			"AnchorStateRegistry":      "0xd987136472d8B51b98384dcCc17Fe6Cb9264d1C3",
			"AnchorStateRegistryProxy": "0x3536dcC25B70Fb2EB9FB2D42dFc4eF6f7fb218A9",
			"OptimismPortalProxy":      "0xabcd1234abcd1234abcd1234abcd1234abcd1234",
		})

		got := clearAnchorStateRegistryFromAddressFile(dir)

		if !got {
			t.Fatal("expected true, got false")
		}
		addrs := readAddressFile(t, dir)
		if _, exists := addrs["AnchorStateRegistry"]; exists {
			t.Error("AnchorStateRegistry key should have been removed")
		}
		if _, exists := addrs["AnchorStateRegistryProxy"]; !exists {
			t.Error("AnchorStateRegistryProxy should still be present")
		}
		if _, exists := addrs["OptimismPortalProxy"]; !exists {
			t.Error("other keys should still be present")
		}
	})

	t.Run("returns false when AnchorStateRegistry key is absent", func(t *testing.T) {
		dir := t.TempDir()
		writeAddressFile(t, dir, map[string]string{
			"AnchorStateRegistryProxy": "0x3536dcC25B70Fb2EB9FB2D42dFc4eF6f7fb218A9",
		})

		got := clearAnchorStateRegistryFromAddressFile(dir)

		if got {
			t.Fatal("expected false when key is absent, got true")
		}
	})

	t.Run("returns false when address.json does not exist", func(t *testing.T) {
		dir := t.TempDir()
		// No file created.
		got := clearAnchorStateRegistryFromAddressFile(dir)
		if got {
			t.Fatal("expected false for missing file, got true")
		}
	})

	t.Run("returns false when address.json contains invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		deploymentsDir := filepath.Join(dir, "deployments", "thanos-stack-sepolia")
		if err := os.MkdirAll(deploymentsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(deploymentsDir, "address.json"), []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}

		got := clearAnchorStateRegistryFromAddressFile(dir)
		if got {
			t.Fatal("expected false for invalid JSON, got true")
		}
	})

	t.Run("idempotent: second call returns false (key already removed)", func(t *testing.T) {
		dir := t.TempDir()
		writeAddressFile(t, dir, map[string]string{
			"AnchorStateRegistry":      "0xd987136472d8B51b98384dcCc17Fe6Cb9264d1C3",
			"AnchorStateRegistryProxy": "0x3536dcC25B70Fb2EB9FB2D42dFc4eF6f7fb218A9",
		})

		first := clearAnchorStateRegistryFromAddressFile(dir)
		second := clearAnchorStateRegistryFromAddressFile(dir)

		if !first {
			t.Fatal("first call should return true")
		}
		if second {
			t.Fatal("second call should return false (key already removed)")
		}
	})
}
