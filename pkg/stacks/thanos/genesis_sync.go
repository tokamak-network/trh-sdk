package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"go.uber.org/zap"
)

// genesisRegenerateFn regenerates rollup.json using --base-genesis, writing the
// result to rollupOutPath. Injected as a dependency so callers can stub it in tests.
type genesisRegenerateFn func(ctx context.Context, rollupOutPath string) error

// computeGenesisBlockHash computes block 0's hash from genesis.json using
// core.Genesis.ToBlock().Hash(). Verified empirically to match what
// tokamak-deployer writes into rollup.json genesis.l2.hash (3 separate deployments).
func computeGenesisBlockHash(genesisPath string) (common.Hash, error) {
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return common.Hash{}, fmt.Errorf("read genesis file: %w", err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return common.Hash{}, fmt.Errorf("parse genesis JSON: %w", err)
	}
	return genesis.ToBlock().Hash(), nil
}

// readRollupGenesisHash extracts genesis.l2.hash from rollup.json.
func readRollupGenesisHash(rollupPath string) (common.Hash, error) {
	data, err := os.ReadFile(rollupPath)
	if err != nil {
		return common.Hash{}, fmt.Errorf("read rollup file: %w", err)
	}
	var rollup struct {
		Genesis struct {
			L2 struct {
				Hash string `json:"hash"`
			} `json:"l2"`
		} `json:"genesis"`
	}
	if err := json.Unmarshal(data, &rollup); err != nil {
		return common.Hash{}, fmt.Errorf("parse rollup JSON: %w", err)
	}
	hashStr := rollup.Genesis.L2.Hash
	if !strings.HasPrefix(hashStr, "0x") || len(hashStr) < 3 {
		return common.Hash{}, fmt.Errorf("genesis.l2.hash invalid or missing: %q", hashStr)
	}
	return common.HexToHash(hashStr), nil
}

// ensureRollupGenesisHashSync verifies that rollup.json's genesis.l2.hash matches
// genesis.json's computed block 0 hash. On mismatch, calls regenerate to auto-correct.
//
// Root cause this guards against: maybeFundAAAdmin patches genesis.json alloc before
// Stage B, changing block 0's hash. If rollup.json is not re-synced (e.g., stacks
// deployed before this check was added), op-node CrashLoopBackOff with
// "l2_genesis_block_hash mismatch" at startup.
func ensureRollupGenesisHashSync(
	ctx context.Context,
	genesisPath, rollupPath string,
	regenerate genesisRegenerateFn,
	logger *zap.SugaredLogger,
) error {
	genesisHash, err := computeGenesisBlockHash(genesisPath)
	if err != nil {
		return fmt.Errorf("compute genesis block hash: %w", err)
	}
	rollupHash, err := readRollupGenesisHash(rollupPath)
	if err != nil {
		return fmt.Errorf("read rollup genesis hash: %w", err)
	}

	if genesisHash == rollupHash {
		logger.Infow("genesis/rollup hash in sync", "hash", genesisHash.Hex())
		return nil
	}

	logger.Warnw("genesis hash mismatch — auto-correcting rollup.json",
		"genesis_hash", genesisHash.Hex(),
		"rollup_hash", rollupHash.Hex())

	if err := regenerate(ctx, rollupPath); err != nil {
		return fmt.Errorf("regenerate rollup.json after genesis hash mismatch: %w", err)
	}

	// Verify the fix took effect
	fixedHash, err := readRollupGenesisHash(rollupPath)
	if err != nil {
		return fmt.Errorf("re-read rollup.json after regeneration: %w", err)
	}
	if fixedHash != genesisHash {
		return fmt.Errorf("rollup.json genesis hash still mismatched after regeneration (want %s, got %s)", genesisHash.Hex(), fixedHash.Hex())
	}

	logger.Infow("genesis/rollup hash corrected", "hash", fixedHash.Hex())
	return nil
}
