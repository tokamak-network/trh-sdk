package thanos

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPrestateHash(t *testing.T) {
	t.Run("valid prestate json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prestate.json")
		content := `{"pre":"0xabc123def456abc123def456abc123def456abc123def456abc123def456abc123","other":"ignored"}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		hash, err := readPrestateHash(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash != "0xabc123def456abc123def456abc123def456abc123def456abc123def456abc123" {
			t.Errorf("got %q, want expected hash", hash)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readPrestateHash("/nonexistent/prestate.json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty pre field", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prestate.json")
		if err := os.WriteFile(path, []byte(`{"pre":""}`), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readPrestateHash(path)
		if err == nil {
			t.Fatal("expected error for empty pre field")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prestate.json")
		if err := os.WriteFile(path, []byte(`not json`), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readPrestateHash(path)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}
