package thanos

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// genesisMinimalFixturePath is a test genesis.json with known fields and a deterministic hash.
const genesisMinimalFixturePath = "testdata/genesis-minimal.json"

// computedHashForGenesisMinimal is core.Genesis.ToBlock().Hash() for testdata/genesis-minimal.json.
// Verified empirically — update here if the fixture changes.
const computedHashForGenesisMinimal = "0x9f5e5efdb5606df88602ce316714dcd0745f1c42fd762c1311fabc83bc6c5deb"

func TestComputeGenesisBlockHash_KnownFixture(t *testing.T) {
	got, err := computeGenesisBlockHash(genesisMinimalFixturePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hex() != computedHashForGenesisMinimal {
		t.Errorf("hash = %s, want %s", got.Hex(), computedHashForGenesisMinimal)
	}
}

func TestComputeGenesisBlockHash_Deterministic(t *testing.T) {
	h1, err := computeGenesisBlockHash(genesisMinimalFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := computeGenesisBlockHash(genesisMinimalFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("hash is not deterministic")
	}
}

func TestComputeGenesisBlockHash_MissingFile(t *testing.T) {
	_, err := computeGenesisBlockHash("/nonexistent/genesis.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestReadRollupGenesisHash_ValidJSON(t *testing.T) {
	rollup := map[string]interface{}{
		"genesis": map[string]interface{}{
			"l2": map[string]interface{}{
				"hash": computedHashForGenesisMinimal,
			},
		},
	}
	data, _ := json.Marshal(rollup)
	tmp := filepath.Join(t.TempDir(), "rollup.json")
	os.WriteFile(tmp, data, 0644)

	got, err := readRollupGenesisHash(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hex() != computedHashForGenesisMinimal {
		t.Errorf("got %s, want %s", got.Hex(), computedHashForGenesisMinimal)
	}
}

func TestReadRollupGenesisHash_InvalidJSON(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "rollup.json")
	os.WriteFile(tmp, []byte("not json"), 0644)

	_, err := readRollupGenesisHash(tmp)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestReadRollupGenesisHash_MissingHash(t *testing.T) {
	rollup := map[string]interface{}{
		"genesis": map[string]interface{}{
			"l2": map[string]interface{}{},
		},
	}
	data, _ := json.Marshal(rollup)
	tmp := filepath.Join(t.TempDir(), "rollup.json")
	os.WriteFile(tmp, data, 0644)

	_, err := readRollupGenesisHash(tmp)
	if err == nil {
		t.Error("expected error for missing hash")
	}
}

func nopLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}

func writeGenesisAndRollup(t *testing.T, dir, rollupHash string) (genesisPath, rollupPath string) {
	t.Helper()
	src, err := os.ReadFile(genesisMinimalFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	genesisPath = filepath.Join(dir, "genesis.json")
	if err := os.WriteFile(genesisPath, src, 0644); err != nil {
		t.Fatal(err)
	}

	rollup := map[string]interface{}{
		"genesis": map[string]interface{}{
			"l1": map[string]interface{}{"hash": "0xaabbcc", "number": 0},
			"l2": map[string]interface{}{
				"hash":   rollupHash,
				"number": 0,
			},
		},
	}
	rollupData, _ := json.Marshal(rollup)
	rollupPath = filepath.Join(dir, "rollup.json")
	os.WriteFile(rollupPath, rollupData, 0644)
	return
}

func TestEnsureRollupGenesisHashSync_InSync_RegenNotCalled(t *testing.T) {
	dir := t.TempDir()
	genesisPath, rollupPath := writeGenesisAndRollup(t, dir, computedHashForGenesisMinimal)

	called := false
	regenerate := func(_ context.Context, _ string) error {
		called = true
		return nil
	}

	if err := ensureRollupGenesisHashSync(context.Background(), genesisPath, rollupPath, regenerate, nopLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("regenerate should NOT be called when hashes are in sync")
	}
}

func TestEnsureRollupGenesisHashSync_Mismatch_CallsRegenerate(t *testing.T) {
	dir := t.TempDir()
	wrongHash := "0x0000000000000000000000000000000000000000000000000000000000000001"
	genesisPath, rollupPath := writeGenesisAndRollup(t, dir, wrongHash)

	regenerateCalled := false
	regenerate := func(_ context.Context, outPath string) error {
		regenerateCalled = true
		// Write the correct hash so the post-check passes
		rollup := map[string]interface{}{
			"genesis": map[string]interface{}{
				"l2": map[string]interface{}{"hash": computedHashForGenesisMinimal},
			},
		}
		data, _ := json.Marshal(rollup)
		return os.WriteFile(outPath, data, 0644)
	}

	if err := ensureRollupGenesisHashSync(context.Background(), genesisPath, rollupPath, regenerate, nopLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !regenerateCalled {
		t.Error("regenerate should be called on hash mismatch")
	}
}

func TestEnsureRollupGenesisHashSync_RegenerateError_Propagates(t *testing.T) {
	dir := t.TempDir()
	wrongHash := "0x0000000000000000000000000000000000000000000000000000000000000002"
	genesisPath, rollupPath := writeGenesisAndRollup(t, dir, wrongHash)

	sentinel := errors.New("binary not found")
	regenerate := func(_ context.Context, _ string) error {
		return sentinel
	}

	err := ensureRollupGenesisHashSync(context.Background(), genesisPath, rollupPath, regenerate, nopLogger())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}
}

func TestEnsureRollupGenesisHashSync_RegenerateProducesWrongHash(t *testing.T) {
	dir := t.TempDir()
	wrongHash := "0x0000000000000000000000000000000000000000000000000000000000000003"
	genesisPath, rollupPath := writeGenesisAndRollup(t, dir, wrongHash)

	regenerate := func(_ context.Context, outPath string) error {
		// Write STILL wrong hash after regeneration — should return error
		rollup := map[string]interface{}{
			"genesis": map[string]interface{}{
				"l2": map[string]interface{}{"hash": wrongHash},
			},
		}
		data, _ := json.Marshal(rollup)
		return os.WriteFile(outPath, data, 0644)
	}

	err := ensureRollupGenesisHashSync(context.Background(), genesisPath, rollupPath, regenerate, nopLogger())
	if err == nil {
		t.Error("expected error when regeneration produces wrong hash")
	}
}
