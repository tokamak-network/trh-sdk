package thanos

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const localConfigVolume = "trh-local-config"

//go:embed templates/local-compose.yml.tmpl
var localComposeTmpl string

type localComposeData struct {
	OpGethImage               string
	OpNodeImage               string
	OpBatcherImage            string
	OpProposerImage           string
	OpChallengerImage         string
	L1RpcUrl                  string
	L1BeaconUrl               string
	SequencerKey              string
	BatcherKey                string
	ProposerKey               string
	ChallengerKey             string
	ConfigVolume              string
	L2ChainID                 uint64
	MaxChannelDuration        uint64
	L2OutputOracleAddress     string
	DisputeGameFactoryAddress string
	UseBlobs                  bool
	EnableFraudProof          bool
	Preset                    string
	DRBNodeImage              string
	DRBLeaderPrivateKey       string
	DRBLeaderEOA              string
}

func (t *ThanosStack) deployLocalNetwork(ctx context.Context) error {
	if t.deployConfig == nil {
		return fmt.Errorf("deployment config not found; run deploy-contracts first")
	}

	if t.deployConfig.DeployContractState == nil ||
		t.deployConfig.DeployContractState.Status != types.DeployContractStatusCompleted {
		return fmt.Errorf("contracts are not deployed successfully; run deploy-contracts first")
	}

	if t.deployConfig.EnableFraudProof && t.deployConfig.ChallengerPrivateKey == "" {
		return fmt.Errorf("fault proof is enabled but challenger private key is not set; re-run deploy-contracts with --enable-fault-proof")
	}

	// Generate compose file
	composePath := filepath.Join(t.deploymentPath, "docker-compose.local.yml")
	if err := t.generateLocalComposeFile(ctx, composePath); err != nil {
		return fmt.Errorf("failed to generate docker compose file: %w", err)
	}

	// Initialize op-geth genesis
	if err := t.initLocalOpGeth(ctx, composePath); err != nil {
		return fmt.Errorf("failed to initialize op-geth: %w", err)
	}

	// Start core services (proposer always, challenger if fraud proof enabled)
	if err := t.startLocalCoreServices(ctx, composePath); err != nil {
		return fmt.Errorf("failed to start core services: %w", err)
	}

	// Initialize AnchorStateRegistry for fault proof chains
	if t.deployConfig.EnableFraudProof {
		deployedContracts, contractsErr := t.readDeploymentContracts()
		if contractsErr != nil {
			t.logger.Warnf("⚠️ Could not read deployed contracts (skipping anchor init): %v", contractsErr)
		} else if deployedContracts.AnchorStateRegistryProxy == "" {
			t.logger.Warn("⚠️ AnchorStateRegistryProxy address not found (skipping anchor init)")
		} else {
			anchorErr := initGenesisAnchorState(
				ctx,
				t.logger,
				t.deployConfig.L1RPCURL,
				"http://localhost:8545",
				t.deployConfig.AdminPrivateKey,
				deployedContracts.AnchorStateRegistryProxy,
				t.deployConfig.L1ChainID,
				0, // gameType 0 = CANNON
			)
			if anchorErr != nil {
				t.logger.Warnf("⚠️ Failed to initialize genesis anchor state: %v", anchorErr)
				t.logger.Warn("Dispute games may fail with AnchorRootNotFound until anchor state is set manually")
			} else {
				t.logger.Info("✅ Genesis anchor state initialized in AnchorStateRegistry")
			}
		}
	}

	// Start preset module services
	modules := constants.PresetModules[t.deployConfig.Preset]
	if err := t.startLocalModules(ctx, composePath, modules); err != nil {
		return fmt.Errorf("failed to start preset modules: %w", err)
	}

	t.logger.Info("✅ Local L2 network started successfully!")
	t.printLocalServiceURLs(modules)
	return nil
}

func (t *ThanosStack) generateLocalComposeFile(ctx context.Context, composePath string) error {
	imageTags := constants.DockerImageTag[t.network]

	contracts, err := t.readDeploymentContracts()
	if err != nil {
		t.logger.Warnf("Failed to read deployment contracts, some addresses may be empty: %v", err)
		contracts = &types.Contracts{}
	}

	l1ChainConfig := constants.L1ChainConfigurations[t.deployConfig.L1ChainID]

	genesisPath := filepath.Join(t.deploymentPath, "tokamak-thanos/build/genesis.json")
	rollupPath := filepath.Join(t.deploymentPath, "tokamak-thanos/build/rollup.json")
	prestatePath := filepath.Join(t.deploymentPath, "tokamak-thanos/op-program/bin/prestate.json")
	jwtPath := filepath.Join(t.deploymentPath, "jwt.txt")

	// Verify required files exist before populating the config volume.
	for _, f := range []string{genesisPath, rollupPath} {
		info, err := os.Stat(f)
		if err != nil {
			return fmt.Errorf("required file missing: %s (run deploy-contracts first)", f)
		}
		if info.IsDir() {
			if rmErr := os.Remove(f); rmErr != nil {
				return fmt.Errorf("%s is a directory (stale Docker mount); remove it manually: %w", f, rmErr)
			}
			return fmt.Errorf("%s was a stale directory (removed); re-run deploy-contracts to regenerate it", f)
		}
	}
	if t.deployConfig.EnableFraudProof {
		if info, err := os.Stat(prestatePath); err != nil || info.IsDir() {
			return fmt.Errorf("prestate.json missing or invalid: %s (run deploy-contracts with fault proof enabled)", prestatePath)
		}
	}

	// Generate JWT secret if it doesn't exist yet (initLocalOpGeth also does this,
	// but we need jwt.txt present before copying files into the config volume).
	if _, err := os.Stat(jwtPath); os.IsNotExist(err) {
		if err := generateJWTSecret(jwtPath); err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
	}

	// Copy config files into a named Docker volume so services can mount them
	// without bind-mount restrictions (required in DinD environments).
	configFiles := map[string]string{
		"genesis.json": genesisPath,
		"rollup.json":  rollupPath,
		"jwt.txt":      jwtPath,
	}
	if t.deployConfig.EnableFraudProof {
		configFiles["prestate.json"] = prestatePath
	}
	if err := t.copyFilesToVolume(ctx, configFiles); err != nil {
		return fmt.Errorf("failed to populate config volume: %w", err)
	}

	data := localComposeData{
		OpGethImage:               fmt.Sprintf("tokamaknetwork/thanos-op-geth:%s", imageTags.OpGethImageTag),
		OpNodeImage:               fmt.Sprintf("tokamaknetwork/thanos-op-node:%s", imageTags.ThanosStackImageTag),
		OpBatcherImage:            fmt.Sprintf("tokamaknetwork/thanos-op-batcher:%s", imageTags.ThanosStackImageTag),
		OpProposerImage:           fmt.Sprintf("tokamaknetwork/thanos-op-proposer:%s", imageTags.ThanosStackImageTag),
		OpChallengerImage:         fmt.Sprintf("tokamaknetwork/thanos-op-challenger:%s", imageTags.ThanosStackImageTag),
		L1RpcUrl:                  t.deployConfig.L1RPCURL,
		L1BeaconUrl:               t.deployConfig.L1BeaconURL,
		SequencerKey:              t.deployConfig.SequencerPrivateKey,
		BatcherKey:                t.deployConfig.BatcherPrivateKey,
		ProposerKey:               t.deployConfig.ProposerPrivateKey,
		ChallengerKey:             t.deployConfig.ChallengerPrivateKey,
		ConfigVolume:              localConfigVolume,
		L2ChainID:                 t.deployConfig.L2ChainID,
		MaxChannelDuration:        l1ChainConfig.MaxChannelDuration,
		L2OutputOracleAddress:     contracts.L2OutputOracleProxy,
		DisputeGameFactoryAddress: contracts.DisputeGameFactoryProxy,
		UseBlobs:                  t.network != constants.LocalDevnet,
		EnableFraudProof:          t.deployConfig.EnableFraudProof,
		Preset:                    t.deployConfig.Preset,
		DRBNodeImage:              fmt.Sprintf("tokamaknetwork/drb-node:%s", imageTags.DRBNodeImageTag),
		DRBLeaderPrivateKey:       t.deployConfig.AdminPrivateKey,
	}

	// Derive DRB leader EOA from admin private key for gaming/full presets
	if t.deployConfig.Preset == constants.PresetGaming || t.deployConfig.Preset == constants.PresetFull {
		if t.deployConfig.AdminPrivateKey != "" {
			addr, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
			if err == nil {
				data.DRBLeaderEOA = addr.Hex()
			}
		}
	}

	tmpl, err := template.New("local-compose").Parse(localComposeTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse compose template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render compose template: %w", err)
	}

	return os.WriteFile(composePath, buf.Bytes(), 0644)
}

func (t *ThanosStack) initLocalOpGeth(ctx context.Context, composePath string) error {
	// Generate JWT secret if it doesn't exist
	jwtPath := filepath.Join(t.deploymentPath, "jwt.txt")
	if _, err := os.Stat(jwtPath); os.IsNotExist(err) {
		if err := generateJWTSecret(jwtPath); err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
	}

	genesisPath := filepath.Join(t.deploymentPath, "tokamak-thanos/build/genesis.json")
	genesisHashFile := filepath.Join(t.deploymentPath, "op-geth-data", ".genesis-hash")
	chainDataPath := filepath.Join(t.deploymentPath, "op-geth-data", "chaindata")

	// Compute current genesis hash
	currentHash, err := hashFile(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to hash genesis.json: %w", err)
	}

	// Check if chaindata exists and genesis hash matches
	if _, err := os.Stat(chainDataPath); err == nil {
		prevHash, readErr := os.ReadFile(genesisHashFile)
		if readErr == nil && string(prevHash) == currentHash {
			t.logger.Info("op-geth data directory already exists with matching genesis, skipping init")
			return nil
		}

		// Genesis changed — wipe stale chaindata to prevent hash mismatch
		t.logger.Warn("genesis.json changed since last init, reinitializing op-geth data...")
		if err := t.resetOpGethVolume(ctx, composePath); err != nil {
			return fmt.Errorf("failed to reset op-geth volume: %w", err)
		}
	}

	t.logger.Info("Initializing op-geth genesis...")
	// Run genesis init using the config volume (avoids bind-mount path issues in DinD).
	if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
		"-f", composePath,
		"run", "--rm", "--no-deps",
		"op-geth",
		"--datadir=/data", "init", "/config/genesis.json"); err != nil {
		return err
	}

	// Persist genesis hash for future change detection
	hashDir := filepath.Dir(genesisHashFile)
	if err := os.MkdirAll(hashDir, 0755); err != nil {
		t.logger.Warnf("Failed to create directory for genesis hash: %v", err)
	}
	if err := os.WriteFile(genesisHashFile, []byte(currentHash), 0644); err != nil {
		t.logger.Warnf("Failed to save genesis hash (init will repeat next run): %v", err)
	}
	return nil
}

// copyFilesToVolume copies files from the container filesystem into a named Docker volume
// using a temporary Alpine container. This avoids bind-mount restrictions in DinD environments
// where container-internal paths are not accessible to the host Docker daemon.
func (t *ThanosStack) copyFilesToVolume(ctx context.Context, files map[string]string) error {
	const helperName = "trh-config-init"

	// Remove any stale helper container from a previous run.
	_, _ = utils.ExecuteCommand(ctx, "docker", "rm", "-f", helperName)

	containerID, err := utils.ExecuteCommand(ctx, "docker", "run", "-d",
		"--name", helperName,
		"-v", localConfigVolume+":/config",
		"alpine", "sleep", "infinity")
	if err != nil {
		return fmt.Errorf("failed to start config helper container: %w", err)
	}
	containerID = strings.TrimSpace(containerID)
	defer utils.ExecuteCommand(ctx, "docker", "rm", "-f", containerID)

	for destName, srcPath := range files {
		if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "cp",
			srcPath, containerID+":/config/"+destName); err != nil {
			return fmt.Errorf("failed to copy %s into config volume: %w", destName, err)
		}
	}
	return nil
}

// resetOpGethVolume stops op-geth and removes its data volume so it can be reinitialized.
func (t *ThanosStack) resetOpGethVolume(ctx context.Context, composePath string) error {
	// Stop op-geth and dependent services
	_ = utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
		"-f", composePath, "stop", "op-geth", "op-node", "op-batcher")

	// Remove the op-geth container to release the volume
	_ = utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
		"-f", composePath, "rm", "-f", "op-geth")

	// Remove the named volume
	projectName := filepath.Base(t.deploymentPath)
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", "volume", "rm", "-f",
		projectName+"_op-geth-data")
}

// hashFile returns the hex-encoded SHA-256 hash of a file's contents.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (t *ThanosStack) startLocalCoreServices(ctx context.Context, composePath string) error {
	args := []string{"compose", "-f", composePath}
	if t.deployConfig.EnableFraudProof {
		args = append(args, "--profile", "challenger")
	} else {
		args = append(args, "--profile", "proposer")
	}
	args = append(args, "up", "-d", "--remove-orphans")
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", args...)
}

func (t *ThanosStack) startLocalModules(ctx context.Context, composePath string, modules map[string]bool) error {
	var profiles []string
	for module, enabled := range modules {
		if !enabled || module == "crossTrade" || module == "drb" {
			// crossTrade requires additional contract deployment; drb is started inline via compose
			continue
		}
		profiles = append(profiles, module)
	}
	if len(profiles) == 0 {
		return nil
	}

	// Run blockscout DB migration before starting (idempotent, no-op if already migrated)
	for _, p := range profiles {
		if p == "blockExplorer" {
			t.logger.Info("Running blockscout database migration...")
			// Start blockscout-db first
			if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
				"-f", composePath, "--profile", "blockExplorer",
				"up", "-d", "blockscout-db"); err != nil {
				t.logger.Warnf("Failed to start blockscout-db: %v", err)
			}
			// Run migration
			if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
				"-f", composePath, "--profile", "blockExplorer",
				"run", "--rm", "blockscout",
				"bin/blockscout", "eval", "Elixir.Explorer.ReleaseTasks.create_and_migrate()"); err != nil {
				t.logger.Warnf("Blockscout migration warning (may be already migrated): %v", err)
			}
			break
		}
	}

	// Re-up with all profiles (already running services are skipped)
	args := []string{"compose", "-f", composePath}
	if t.deployConfig.EnableFraudProof {
		args = append(args, "--profile", "challenger")
	} else {
		args = append(args, "--profile", "proposer")
	}
	for _, p := range profiles {
		args = append(args, "--profile", p)
	}
	args = append(args, "up", "-d", "--remove-orphans")
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", args...)
}

func (t *ThanosStack) destroyLocalNetwork(ctx context.Context) error {
	composePath := filepath.Join(t.deploymentPath, "docker-compose.local.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.logger.Warn("Local compose file not found, nothing to destroy")
		return nil
	}
	t.logger.Info("Stopping local L2 network...")
	allProfiles := []string{"proposer", "challenger", "bridge", "blockExplorer", "monitoring", "uptimeService"}
	args := []string{"compose", "-f", composePath}
	for _, p := range allProfiles {
		args = append(args, "--profile", p)
	}
	args = append(args, "down", "-v", "--remove-orphans")
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", args...)
}

func (t *ThanosStack) printLocalServiceURLs(modules map[string]bool) {
	t.logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	t.logger.Info("  Local L2 Network Services")
	t.logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	t.logger.Infof("  L2 RPC (HTTP): http://localhost:8545")
	t.logger.Infof("  L2 RPC (WS):   ws://localhost:8546")
	t.logger.Infof("  op-node RPC:   http://localhost:9545")
	if modules["bridge"] {
		t.logger.Infof("  Bridge UI:     http://localhost:3001")
	}
	if modules["blockExplorer"] {
		t.logger.Infof("  Block Explorer: http://localhost:4001")
	}
	if modules["monitoring"] {
		t.logger.Infof("  Grafana:       http://localhost:3002  (admin/admin)")
		t.logger.Infof("  Prometheus:    http://localhost:9090")
	}
	if modules["uptimeService"] {
		t.logger.Infof("  Uptime Kuma:   http://localhost:3003")
	}
	t.logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func generateJWTSecret(path string) error {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(secret)), 0600)
}
