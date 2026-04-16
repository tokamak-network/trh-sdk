package thanos

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// TokamakDeployerVersion is the pinned version of the tokamak-deployer binary.
const TokamakDeployerVersion = "v1.0.1"

const tokamakDeployerRepo = "tokamak-network/tokamak-thanos"

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

	osName := runtime.GOOS // "linux", "darwin"
	arch := runtime.GOARCH // "amd64", "arm64"
	assetName := fmt.Sprintf("tokamak-deployer-%s-%s", osName, arch)
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		tokamakDeployerRepo, version, assetName,
	)

	if err := downloadFile(downloadURL, binaryPath); err != nil {
		return "", fmt.Errorf("failed to download tokamak-deployer %s: %w\nCheck network connectivity or retry.", version, err)
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

// deployContractsOpts holds inputs for the deploy-contracts subcommand.
type deployContractsOpts struct {
	L1RPCURL   string
	PrivateKey string
	L2ChainID  uint64
	OutPath    string
}

// genesisOpts holds inputs for the generate-genesis subcommand.
type genesisOpts struct {
	DeployOutputPath string
	ConfigPath       string
	OutPath          string
	RollupOutPath    string
	Preset           string
}

// runDeployContracts executes tokamak-deployer deploy-contracts.
func runDeployContracts(ctx context.Context, binaryPath string, opts deployContractsOpts, w io.Writer) error {
	return runBinaryCommand(ctx, binaryPath, []string{
		"deploy-contracts",
		"--l1-rpc", opts.L1RPCURL,
		"--private-key", opts.PrivateKey,
		"--chain-id", fmt.Sprintf("%d", opts.L2ChainID),
		"--out", opts.OutPath,
	}, w)
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
