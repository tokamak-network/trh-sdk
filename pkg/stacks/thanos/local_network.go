package thanos

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	_ "embed"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

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
	GenesisPath               string
	RollupPath                string
	PrestatePath              string
	JWTPath                   string
	L2ChainID                 uint64
	MaxChannelDuration        uint64
	L2OutputOracleAddress     string
	DisputeGameFactoryAddress string
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
	if err := t.generateLocalComposeFile(composePath); err != nil {
		return fmt.Errorf("failed to generate docker compose file: %w", err)
	}

	// Initialize op-geth genesis
	if err := t.initLocalOpGeth(ctx, composePath); err != nil {
		return fmt.Errorf("failed to initialize op-geth: %w", err)
	}

	// Start core services + proposer or challenger
	if err := t.startLocalCoreServices(ctx, composePath); err != nil {
		return fmt.Errorf("failed to start core services: %w", err)
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

func (t *ThanosStack) generateLocalComposeFile(composePath string) error {
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

	data := localComposeData{
		OpGethImage:               fmt.Sprintf("tokamaknetwork/thanos-op-geth:nightly-%s", imageTags.OpGethImageTag),
		OpNodeImage:               fmt.Sprintf("tokamaknetwork/thanos-op-node:nightly-%s", imageTags.ThanosStackImageTag),
		OpBatcherImage:            fmt.Sprintf("tokamaknetwork/thanos-op-batcher:nightly-%s", imageTags.ThanosStackImageTag),
		OpProposerImage:           fmt.Sprintf("tokamaknetwork/thanos-op-proposer:nightly-%s", imageTags.ThanosStackImageTag),
		OpChallengerImage:         fmt.Sprintf("tokamaknetwork/thanos-op-challenger:nightly-%s", imageTags.ThanosStackImageTag),
		L1RpcUrl:                  t.deployConfig.L1RPCURL,
		L1BeaconUrl:               t.deployConfig.L1BeaconURL,
		SequencerKey:              t.deployConfig.SequencerPrivateKey,
		BatcherKey:                t.deployConfig.BatcherPrivateKey,
		ProposerKey:               t.deployConfig.ProposerPrivateKey,
		ChallengerKey:             t.deployConfig.ChallengerPrivateKey,
		GenesisPath:               genesisPath,
		RollupPath:                rollupPath,
		PrestatePath:              prestatePath,
		JWTPath:                   jwtPath,
		L2ChainID:                 t.deployConfig.L2ChainID,
		MaxChannelDuration:        l1ChainConfig.MaxChannelDuration,
		L2OutputOracleAddress:     contracts.L2OutputOracleProxy,
		DisputeGameFactoryAddress: contracts.DisputeGameFactoryProxy,
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

	// Check if genesis already initialized (data directory has chaindata)
	chainDataPath := filepath.Join(t.deploymentPath, "op-geth-data", "chaindata")
	if _, err := os.Stat(chainDataPath); err == nil {
		t.logger.Info("op-geth data directory already exists, skipping genesis init")
		return nil
	}

	t.logger.Info("Initializing op-geth genesis...")
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
		"-f", composePath,
		"run", "--rm", "op-geth",
		"--datadir=/data", "init", "/config/genesis.json")
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
		if !enabled || module == "crossTrade" {
			// crossTrade requires additional contract deployment; skip for local
			continue
		}
		profiles = append(profiles, module)
	}
	if len(profiles) == 0 {
		return nil
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
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
		"-f", composePath, "down", "-v", "--remove-orphans")
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
