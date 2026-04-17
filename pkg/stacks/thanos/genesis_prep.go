package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
)

// prepareL2GenesisInputs stages the files tokamak-deployer's generate-genesis needs
// before op-node can run, and returns the paths to use when invoking the binary.
//
// Why this exists:
//
//   - op-node's "genesis l2" subcommand requires --l2-allocs (L2 genesis state dump
//     produced by forge scripts/L2Genesis.s.sol), which neither trh-sdk nor
//     tokamak-deployer used to run. This helper fills that gap.
//   - forge's FFI sandbox only allows reads under the contracts-bedrock project
//     root, so the deploy-config and deploy-output must be staged inside it before
//     the script runs (/tmp paths are rejected).
//   - forge's L2Genesis.s.sol parses every key in CONTRACT_ADDRESSES_PATH as an
//     Ethereum address, so the l1ChainId / l2ChainId metadata tokamak-deployer
//     writes into deploy-output.json must be stripped first.
//
// Inputs:
//   - tokamakThanosDir: the cloned tokamak-thanos project root
//   - deployOutputPath: absolute path to deploy-output.json (with metadata)
//   - deployConfigPath: absolute path to deploy-config.json
//   - l2ChainID: used to name staged files so concurrent deployments don't collide
//
// Outputs:
//   - stagedAddrPath: addresses-only JSON under contracts-bedrock/deployments/
//   - stagedConfigPath: deploy-config copy under contracts-bedrock/deploy-config/
func prepareL2GenesisInputs(tokamakThanosDir, deployOutputPath, deployConfigPath string, l2ChainID uint64) (stagedAddrPath, stagedConfigPath string, err error) {
	contractsDir := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock")
	deploymentsDir := filepath.Join(contractsDir, "deployments")
	deployConfigDir := filepath.Join(contractsDir, "deploy-config")
	if err := os.MkdirAll(deploymentsDir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir %s: %w", deploymentsDir, err)
	}
	if err := os.MkdirAll(deployConfigDir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir %s: %w", deployConfigDir, err)
	}

	stagedAddrPath = filepath.Join(deploymentsDir, fmt.Sprintf("%d-addresses.json", l2ChainID))
	if err := writeAddressesOnly(deployOutputPath, stagedAddrPath); err != nil {
		return "", "", err
	}

	stagedConfigPath = filepath.Join(deployConfigDir, fmt.Sprintf("%d.json", l2ChainID))
	if err := copyFile(deployConfigPath, stagedConfigPath); err != nil {
		return "", "", fmt.Errorf("stage deploy-config: %w", err)
	}
	return stagedAddrPath, stagedConfigPath, nil
}

// writeAddressesOnly reads a deploy-output JSON (which includes l1ChainId /
// l2ChainId metadata plus contract-name → address fields) and writes a new
// JSON containing only the address fields, which is what forge's
// L2Genesis.s.sol expects under CONTRACT_ADDRESSES_PATH.
func writeAddressesOnly(srcPath, dstPath string) error {
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPath, err)
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return fmt.Errorf("parse %s: %w", srcPath, err)
	}
	out := make(map[string]json.RawMessage, len(all))
	for k, v := range all {
		// Drop known metadata keys. forge L2Genesis parses every remaining key as
		// an address, so numbers / non-address values must not be present.
		if k == "l1ChainId" || k == "l2ChainId" {
			continue
		}
		out[k] = v
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal addresses-only: %w", err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// runForgeL2GenesisScript runs forge scripts/L2Genesis.s.sol inside the
// contracts-bedrock directory to produce state-dump-<l2ChainID>.json, the L2
// allocs file op-node will need.
func runForgeL2GenesisScript(
	ctx context.Context,
	logger *zap.SugaredLogger,
	tokamakThanosDir, stagedAddrPath, stagedConfigPath, l1RPCURL string,
) (stateDumpPath string, err error) {
	contractsDir := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock")

	cmd := exec.CommandContext(ctx, "forge", "script",
		"scripts/L2Genesis.s.sol:L2Genesis",
		"--rpc-url", l1RPCURL,
	)
	cmd.Dir = contractsDir
	cmd.Env = append(os.Environ(),
		"CONTRACT_ADDRESSES_PATH="+stagedAddrPath,
		"DEPLOY_CONFIG_PATH="+stagedConfigPath,
	)

	logger.Info("Running forge L2Genesis.s.sol to produce state-dump",
		"dir", contractsDir,
		"addresses", stagedAddrPath,
		"config", stagedConfigPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("forge L2Genesis failed: %w\n%s", err, out)
	}

	// forge derives the filename from l2ChainID in deploy-config; we infer it
	// from the staged config name (which is <l2ChainID>.json).
	l2ChainIDStr := filepath.Base(stagedConfigPath)
	l2ChainIDStr = l2ChainIDStr[:len(l2ChainIDStr)-len(filepath.Ext(l2ChainIDStr))]
	stateDumpPath = filepath.Join(contractsDir, "state-dump-"+l2ChainIDStr+".json")
	if _, err := os.Stat(stateDumpPath); err != nil {
		return "", fmt.Errorf("forge L2Genesis completed but state dump missing at %s: %w", stateDumpPath, err)
	}
	logger.Info("✅ L2 state dump generated", "path", stateDumpPath)
	return stateDumpPath, nil
}

// ensureOpNodeBinary returns the absolute path to the op-node binary inside
// tokamakThanosDir, building it on demand with `go build` if it does not yet
// exist. The built binary is placed at op-node/bin/op-node (same path
// start-deploy.sh and op-node/justfile use).
func ensureOpNodeBinary(ctx context.Context, logger *zap.SugaredLogger, tokamakThanosDir string) (string, error) {
	opNodeDir := filepath.Join(tokamakThanosDir, "op-node")
	binPath := filepath.Join(opNodeDir, "bin", "op-node")

	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	logger.Info("op-node binary not found, building it", "dir", opNodeDir, "out", binPath)
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(binPath), err)
	}
	if err := utils.ExecuteCommandStreamInDir(ctx, logger, opNodeDir,
		"go", "build", "-o", "./bin/op-node", "./cmd",
	); err != nil {
		return "", fmt.Errorf("build op-node: %w", err)
	}
	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("op-node build finished but binary missing at %s: %w", binPath, err)
	}
	return binPath, nil
}
