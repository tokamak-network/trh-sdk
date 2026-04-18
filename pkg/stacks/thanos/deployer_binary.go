package thanos

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// TokamakDeployerVersion is the pinned version of the tokamak-deployer binary.
//
// v0.0.6: --fault-proof bool flag added. Without it cfg.EnableFaultProof stays
// false and steps 27-32 (DisputeGameFactory + AnchorStateRegistry) are silently
// skipped, breaking DRB stacks that expect those addresses (Bug #8).
//
// v0.0.5: --gas-price / --gas-price-multiplier flags introduced. trh-sdk now
// resolves a conservative fixed gas price once and passes it via --gas-price,
// so tokamak-deployer no longer calls SuggestGasPrice per TX.
const TokamakDeployerVersion = "v0.0.6"

const tokamakDeployerRepo = "tokamak-network/tokamak-thanos"

// tokamakDeployerTagPrefix is the monorepo tag prefix used for tokamak-deployer
// releases (full tag: tokamak-deployer/vX.Y.Z). GitHub release asset URLs
// include this prefix: releases/download/tokamak-deployer/vX.Y.Z/<asset>.
const tokamakDeployerTagPrefix = "tokamak-deployer/"

// ensureTokamakDeployer checks the cache dir for the binary; downloads from GitHub Releases if missing.
// Returns the path to the executable binary.
func ensureTokamakDeployer(cacheDir string) (string, error) {
	return ensureTokamakDeployerWithVersion(cacheDir, TokamakDeployerVersion)
}

// ensureTokamakDeployerWithVersion is the internal helper that supports version override for tests.
func ensureTokamakDeployerWithVersion(cacheDir, version string) (string, error) {
	binaryName := fmt.Sprintf("tokamak-deployer-%s", version)
	binaryPath := filepath.Join(cacheDir, binaryName)

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil // cache hit
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("create tokamak-deployer cache dir: %w", err)
	}

	osName := runtime.GOOS // "linux", "darwin"
	arch := runtime.GOARCH // "amd64", "arm64"
	assetName := fmt.Sprintf("tokamak-deployer-%s-%s.tar.gz", osName, arch)
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s%s/%s",
		tokamakDeployerRepo, tokamakDeployerTagPrefix, version, assetName,
	)

	tarPath := binaryPath + ".tar.gz"
	if err := downloadFile(downloadURL, tarPath); err != nil {
		return "", fmt.Errorf("failed to download tokamak-deployer %s: %w\nCheck network connectivity or retry.", version, err)
	}
	defer os.Remove(tarPath)

	if err := extractTarGz(tarPath, binaryPath); err != nil {
		return "", fmt.Errorf("failed to extract tokamak-deployer: %w", err)
	}
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}
	return binaryPath, nil
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(destPath) // clean up partial download
		return err
	}
	return nil
}

func extractTarGz(tarGzPath, destPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for tokamak-deployer executable
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "tokamak-deployer" {
			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			return nil
		}
	}
	return fmt.Errorf("tokamak-deployer binary not found in archive")
}

// deployContractsOpts holds inputs for the deploy-contracts subcommand.
type deployContractsOpts struct {
	L1RPCURL   string
	PrivateKey string
	L2ChainID  uint64
	OutPath    string

	// EnableFaultProof, when true, adds --fault-proof so tokamak-deployer
	// runs steps 27-32 (DisputeGameFactory + AnchorStateRegistry). Required
	// for DRB / fault-proof stacks; the CLI flag has been present since
	// tokamak-deployer v0.0.6 (Bug #8).
	EnableFaultProof bool

	// GasPriceWei, when non-nil, is forwarded to tokamak-deployer via
	// --gas-price. The deployer reuses this exact price for every one of
	// its 26-32 transactions instead of calling SuggestGasPrice per-TX.
	// trh-sdk sets this to (current suggested) × 2 so the bump-on-timeout
	// retry path inside the deployer rarely activates.
	GasPriceWei *big.Int
}

// genesisOpts holds inputs for the generate-genesis subcommand.
type genesisOpts struct {
	DeployOutputPath string
	ConfigPath       string
	OutPath          string
	RollupOutPath    string
	Preset           string

	// L1RPCURL is passed to op-node via --l1-rpc. Required unless BaseGenesisPath is set.
	L1RPCURL string

	// L2AllocsPath is the state dump produced by forge L2Genesis.s.sol. Passed to
	// op-node via --l2-allocs. Required unless BaseGenesisPath is set.
	L2AllocsPath string

	// OpNodeBinary overrides the op-node binary lookup. Useful when op-node is
	// built locally at tokamak-thanos/op-node/bin/op-node rather than $PATH.
	OpNodeBinary string

	// BaseGenesisPath, when set, skips tokamak-deployer's internal op-node call
	// and uses this file as the base genesis. Only for testing / advanced flows.
	BaseGenesisPath string
}

// buildDeployContractsArgs constructs the argv for tokamak-deployer
// deploy-contracts. Extracted so args can be exercised in unit tests
// without spawning a subprocess.
func buildDeployContractsArgs(opts deployContractsOpts) []string {
	args := []string{
		"deploy-contracts",
		"--l1-rpc", opts.L1RPCURL,
		"--private-key", opts.PrivateKey,
		"--chain-id", fmt.Sprintf("%d", opts.L2ChainID),
		"--out", opts.OutPath,
	}
	if opts.EnableFaultProof {
		args = append(args, "--fault-proof")
	}
	if opts.GasPriceWei != nil && opts.GasPriceWei.Sign() > 0 {
		args = append(args, "--gas-price", opts.GasPriceWei.String())
	}
	return args
}

// runDeployContracts executes tokamak-deployer deploy-contracts.
func runDeployContracts(ctx context.Context, binaryPath string, opts deployContractsOpts, w io.Writer) error {
	return runBinaryCommand(ctx, binaryPath, buildDeployContractsArgs(opts), w)
}

// runGenerateGenesis executes tokamak-deployer generate-genesis.
func runGenerateGenesis(ctx context.Context, binaryPath string, opts genesisOpts, w io.Writer) error {
	args := []string{
		"generate-genesis",
		"--deploy-output", opts.DeployOutputPath,
		"--config", opts.ConfigPath,
		"--out", opts.OutPath,
	}
	if opts.RollupOutPath != "" {
		args = append(args, "--rollup-out", opts.RollupOutPath)
	}
	if opts.Preset != "" {
		args = append(args, "--preset", opts.Preset)
	}
	if opts.BaseGenesisPath != "" {
		args = append(args, "--base-genesis", opts.BaseGenesisPath)
	} else {
		// --l1-rpc and --l2-allocs are required by op-node genesis l2 (as of
		// tokamak-deployer v0.0.3). If the caller forgot them and also didn't
		// pass --base-genesis, let the binary surface the same error.
		if opts.L1RPCURL != "" {
			args = append(args, "--l1-rpc", opts.L1RPCURL)
		}
		if opts.L2AllocsPath != "" {
			args = append(args, "--l2-allocs", opts.L2AllocsPath)
		}
	}
	if opts.OpNodeBinary != "" {
		args = append(args, "--op-node-bin", opts.OpNodeBinary)
	}
	return runBinaryCommand(ctx, binaryPath, args, w)
}

func runBinaryCommand(ctx context.Context, binaryPath string, args []string, w io.Writer) error {
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	if w == nil {
		w = os.Stdout
	}
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tokamak-deployer %s: %w", args[0], err)
	}
	return nil
}
