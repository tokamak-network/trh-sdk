package thanos

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	_ "embed"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const localConfigVolume = "trh-local-config"
const localMonitoringVolume = "trh-local-monitoring"

// localL2RPCURL returns the L2 RPC URL reachable from the current process.
// When running inside a Docker container (trh-backend), localhost refers to
// the container itself, not the host where op-geth's port is mapped.
// In that case we use host.docker.internal which resolves to the Docker host.
func localL2RPCURL() string {
	if url := os.Getenv("L2_RPC_URL"); url != "" {
		return url
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "http://host.docker.internal:8545"
	}
	return "http://localhost:8545"
}

//go:embed templates/local-compose.yml.tmpl
var localComposeTmpl string

//go:embed templates/grafana-dashboard-application.json
var grafanaDashboardApplication string

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
	DataAvailabilityType      string // "blobs" or "calldata"
	BlobFeeCapMultiplier      uint64 // txmgr blob fee cap multiplier (default 4)
	MaxBlobBaseFeeGwei        string // txmgr max blob base fee threshold in gwei ("0" = disabled)
	EnableFraudProof          bool
	Preset                    string
	DRBNodeImage              string
	DRBLeaderPrivateKey       string
	DRBLeaderEOA              string
	DRBLeaderPeerID           string
	DRBRegulars               [3]DRBRegular
	// Bridge environment variables
	BridgeL1ChainName                   string
	BridgeL1ChainID                     string
	BridgeL1RPC                         string
	BridgeL1NativeCurrencyName          string
	BridgeL1NativeCurrencySymbol        string
	BridgeL1NativeCurrencyDecimals      int
	BridgeL1BlockExplorer               string
	BridgeL1USDCAddress                 string
	BridgeL1USDTAddress                 string
	BridgeL2ChainName                   string
	BridgeL2ChainID                     string
	BridgeL2RPC                         string
	BridgeL2NativeCurrencyName          string
	BridgeL2NativeCurrencySymbol        string
	BridgeNativeTokenL1Address          string
	BridgeStandardBridgeAddress         string
	BridgeAddressManagerAddress         string
	BridgeL1CrossDomainMessengerAddress string
	BridgeOptimismPortalAddress         string
	BridgeL2OutputOracleAddress         string
	BridgeL1USDCBridgeAddress           string
	BridgeDisputeGameFactoryAddress     string
	BridgeBatchSubmissionFrequency      uint64
	BridgeL1BlockTime                   uint64
	BridgeL2BlockTime                   uint64
	BridgeOutputRootFrequency           uint64
	BridgeChallengePeriod               uint64
	// Block Explorer environment variables
	BlockExplorerNetworkName         string
	BlockExplorerL1BaseURL           string
	BlockExplorerSystemConfigAddress string
	BlockExplorerBatchInboxAddress   string
	BlockExplorerL1StartBlock        uint64
	BlockExplorerCoinSymbol          string
	BlockExplorerCoinName            string
	BlockExplorerCoinDecimals        uint8
	BlockExplorerStableCoin          bool
	// Monitoring
	MonitoringConfigVolume string
	// AA operator — controls whether alto-bundler is included in the compose file.
	// Set to true for non-TON fee tokens; the aa-operator itself runs as a goroutine in trh-backend.
	AAOperatorEnabled bool
	AAAdminPrivateKey string
	// CrossTrade dApp — populated after DeployCrossTradeLocal succeeds.
	CrossTradeEnabled         bool
	CrossTradeProjectID       string
	CrossTradeChainConfigL2L1 string // NEXT_PUBLIC_CHAIN_CONFIG_L2_L1 (L2→L1 bridging)
	CrossTradeChainConfigL2L2 string // NEXT_PUBLIC_CHAIN_CONFIG_L2_L2 (L2→L2 bridging)
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
			return fmt.Errorf("failed to read deployed contracts for anchor init: %w", contractsErr)
		}
		if deployedContracts.AnchorStateRegistryProxy == "" {
			return fmt.Errorf("AnchorStateRegistryProxy address not found in deployed contracts — cannot initialize genesis anchor state")
		}
		anchorErr := initGenesisAnchorState(
			ctx,
			t.logger,
			t.deployConfig.L1RPCURL,
			localL2RPCURL(),
			t.deployConfig.AdminPrivateKey,
			deployedContracts.AnchorStateRegistryProxy,
			t.deployConfig.L1ChainID,
			0, // gameType 0 = CANNON
		)
		if anchorErr != nil {
			return fmt.Errorf("failed to initialize genesis anchor state (op-proposer will fail with AnchorRootNotFound): %w", anchorErr)
		}
		t.logger.Info("✅ Genesis anchor state initialized in AnchorStateRegistry")
	}

	// Setup AA Paymaster for non-TON fee tokens.
	// Runs after core services are healthy. Non-blocking: failure logs a warning
	// but does not prevent the L2 network from starting.
	if constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
		t.logger.Infof("🔧 AA Paymaster setup starting in background for fee token: %s", t.deployConfig.FeeToken)
		go func() {
			bgCtx := context.Background()
			// Auto-bridge admin TON from L1 to L2 if L2 balance is insufficient for the
			// EntryPoint deposit. On a fresh L2, the admin has zero L2 TON; this step
			// bridges 10 TON (covering the initial deposit + aa-operator refill cycles).
			bridgeOk := true
			if bridgeErr := t.bridgeAdminTONForAASetup(bgCtx); bridgeErr != nil {
				bridgeOk = false
				t.logger.Warnf("⚠️  Admin L2 TON bridge failed: %v", bridgeErr)
				t.logger.Warn("   Fund admin address on L2 manually and re-run `trh-sdk setup-aa`.")
			}

			if bridgeOk {
				if aaErr := t.setupAAPaymaster(bgCtx); aaErr != nil {
					t.logger.Warnf("⚠️  AA Paymaster setup failed: %v", aaErr)
					t.logger.Warn("   AA fee payment features may not work until paymaster is configured manually.")
					t.logger.Warn("   Re-run `trh-sdk setup-aa` or call setupAAPaymaster via the admin API.")
				} else {
					t.logger.Infof("✅ AA Paymaster configured for %s", t.deployConfig.FeeToken)
				}
			}

			// Start alto-bundler if admin has L2 funds (bridge succeeded).
			// Bundler is decoupled from paymaster setup: it processes UserOperations
			// independently and only needs gas funds (TON) in the executor account.
			if bridgeOk {
				t.logger.Info("🚀 Starting alto-bundler (admin funded on L2)...")
				if bundlerErr := utils.ExecuteCommandStream(bgCtx, t.logger, "docker", "compose",
					"-f", composePath, "--profile", "aa", "up", "-d", "alto-bundler"); bundlerErr != nil {
					t.logger.Warnf("⚠️  Failed to start alto-bundler: %v", bundlerErr)
					t.logger.Warn("   Run `docker compose --profile aa up -d alto-bundler` manually.")
				} else {
					t.logger.Info("✅ alto-bundler started successfully")
					// Persist aa profile in .env so bundler is included on restarts
					if err := t.persistAAProfile(composePath); err != nil {
						t.logger.Warnf("Failed to persist aa profile in .env: %v", err)
					}
				}
			}
		}()
	}

	// Start preset module services
	modules := constants.PresetModules[t.deployConfig.Preset]
	if err := t.startLocalModules(ctx, composePath, modules); err != nil {
		return fmt.Errorf("failed to start preset modules: %w", err)
	}

	// Post-startup DRB orchestration (Gaming/Full presets only)
	if modules["drb"] && t.deployConfig.Mnemonic != "" {
		if err := t.orchestrateDRBOperators(ctx, composePath); err != nil {
			return fmt.Errorf("failed to orchestrate DRB operators: %w", err)
		} else {
			t.logger.Info("✅ DRB operators orchestrated and activated successfully")
		}
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

	genesisPath := filepath.Join(t.deploymentPath, "genesis.json")
	rollupPath := filepath.Join(t.deploymentPath, "rollup.json")
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

	feeTokenConfig := constants.GetFeeTokenConfig(t.deployConfig.FeeToken, t.deployConfig.L1ChainID)

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
		DataAvailabilityType:      "blobs",
		BlobFeeCapMultiplier:      4,
		MaxBlobBaseFeeGwei:        "50", // 50 gwei threshold: pause submission during Sepolia spike
		EnableFraudProof:          t.deployConfig.EnableFraudProof,
		Preset:                    t.deployConfig.Preset,
		DRBNodeImage:              fmt.Sprintf("tokamaknetwork/drb-node:%s", imageTags.DRBNodeImageTag),
		DRBLeaderPrivateKey:       t.deployConfig.AdminPrivateKey,
		// Bridge environment variables
		BridgeL1ChainName:                   l1ChainConfig.ChainName,
		BridgeL1ChainID:                     fmt.Sprintf("%d", t.deployConfig.L1ChainID),
		BridgeL1RPC:                         t.deployConfig.L1RPCURL,
		BridgeL1NativeCurrencyName:          l1ChainConfig.NativeTokenName,
		BridgeL1NativeCurrencySymbol:        l1ChainConfig.NativeTokenSymbol,
		BridgeL1NativeCurrencyDecimals:      l1ChainConfig.NativeTokenDecimals,
		BridgeL1BlockExplorer:               l1ChainConfig.BlockExplorer,
		BridgeL1USDCAddress:                 l1ChainConfig.USDCAddress,
		BridgeL1USDTAddress:                 l1ChainConfig.USDTAddress,
		BridgeL2ChainName:                   t.deployConfig.ChainName,
		BridgeL2ChainID:                     fmt.Sprintf("%d", t.deployConfig.L2ChainID),
		BridgeL2RPC:                         "http://localhost:8545",
		BridgeL2NativeCurrencyName:          feeTokenConfig.Name,
		BridgeL2NativeCurrencySymbol:        feeTokenConfig.Symbol,
		BridgeNativeTokenL1Address:          feeTokenConfig.L1Address,
		BridgeStandardBridgeAddress:         contracts.L1StandardBridgeProxy,
		BridgeAddressManagerAddress:         contracts.AddressManager,
		BridgeL1CrossDomainMessengerAddress: contracts.L1CrossDomainMessengerProxy,
		BridgeOptimismPortalAddress:         contracts.OptimismPortalProxy,
		BridgeL2OutputOracleAddress:         contracts.L2OutputOracleProxy,
		BridgeL1USDCBridgeAddress:           contracts.L1UsdcBridgeProxy,
		BridgeDisputeGameFactoryAddress:     contracts.DisputeGameFactoryProxy,
		BridgeBatchSubmissionFrequency:      t.deployConfig.ChainConfiguration.BatchSubmissionFrequency,
		BridgeL1BlockTime:                   t.deployConfig.ChainConfiguration.L1BlockTime,
		BridgeL2BlockTime:                   t.deployConfig.ChainConfiguration.L2BlockTime,
		BridgeOutputRootFrequency:           t.deployConfig.ChainConfiguration.OutputRootFrequency,
		BridgeChallengePeriod:               t.deployConfig.ChainConfiguration.ChallengePeriod,
		// Block Explorer
		BlockExplorerNetworkName:         t.deployConfig.ChainName,
		BlockExplorerL1BaseURL:           l1ChainConfig.BlockExplorer,
		BlockExplorerSystemConfigAddress: contracts.SystemConfigProxy,
		BlockExplorerBatchInboxAddress:   utils.GenerateBatchInboxAddress(t.deployConfig.L2ChainID),
		BlockExplorerL1StartBlock:        readRollupL1GenesisBlock(rollupPath, t.logger),
		BlockExplorerCoinSymbol:          feeTokenConfig.Symbol,
		BlockExplorerCoinName:            feeTokenConfig.Name,
		BlockExplorerCoinDecimals:        feeTokenDecimals(t.deployConfig.FeeToken),
		BlockExplorerStableCoin:          t.deployConfig.FeeToken == constants.FeeTokenUSDC || t.deployConfig.FeeToken == constants.FeeTokenUSDT,
		MonitoringConfigVolume:           localMonitoringVolume,
	}

	// Derive DRB leader EOA and Regular accounts from mnemonic for gaming/full presets
	if t.deployConfig.Preset == constants.PresetGaming || t.deployConfig.Preset == constants.PresetFull {
		if t.deployConfig.AdminPrivateKey != "" {
			addr, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
			if err == nil {
				data.DRBLeaderEOA = addr.Hex()
			}
		}

		// Derive DRB accounts (Leader + 3 Regulars) deterministically from mnemonic
		if t.deployConfig.Mnemonic != "" {
			accounts, err := DeriveDRBAccounts(t.deployConfig.Mnemonic)
			if err != nil {
				t.logger.Warnf("Failed to derive DRB accounts from mnemonic: %v (DRB services will start but without derived peer IDs)", err)
			} else {
				// Populate compose data with Leader peer ID and Regular accounts
				data.DRBLeaderPeerID = accounts.LeaderPeerID
				for i := range accounts.Regulars {
					data.DRBRegulars[i] = accounts.Regulars[i]
				}
				t.logger.Infof("✅ Derived DRB accounts: Leader peer ID + 3 Regular operators")
			}
		}
	}

	// Populate CrossTrade dApp fields — only when local contracts have been deployed AND the preset
	// supports crossTrade (defi, full). Defense-in-depth: even if CrossTradeContracts were manually
	// set on a general/gaming config, we do not render the service.
	if constants.PresetModules[t.deployConfig.Preset]["crossTrade"] {
		if ct := t.deployConfig.CrossTradeContracts; ct != nil && ct.L1CrossTradeProxy != "" && ct.L2CrossTradeProxy != "" {
			l2l1JSON, err := t.buildCrossTradeChainConfigL2L1JSON(ct)
			if err != nil {
				t.logger.Warnf("Failed to build CrossTrade L2_L1 chain config, skipping crossTrade service: %v", err)
			} else {
				l2l2JSON, err := t.buildCrossTradeChainConfigL2L2JSON(ct)
				if err != nil {
					t.logger.Warnf("Failed to build CrossTrade L2_L2 chain config, skipping crossTrade service: %v", err)
				} else {
					data.CrossTradeEnabled = true
					data.CrossTradeProjectID = "568b8d3d0528e743b0e2c6c92f54d721"
					data.CrossTradeChainConfigL2L1 = l2l1JSON
					data.CrossTradeChainConfigL2L2 = l2l2JSON
				}
			}
		}
	}

	// Populate AA operator fields — alto-bundler is included in the compose file for non-TON fee tokens.
	// The aa-operator itself runs as a goroutine in trh-backend (not as a Docker container).
	if constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
		data.AAOperatorEnabled = true
		adminKey := t.deployConfig.AdminPrivateKey
		if !strings.HasPrefix(adminKey, "0x") {
			adminKey = "0x" + adminKey
		}
		data.AAAdminPrivateKey = adminKey
	}

	tmpl, err := template.New("local-compose").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(localComposeTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse compose template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render compose template: %w", err)
	}

	if err := os.WriteFile(composePath, buf.Bytes(), 0644); err != nil {
		return err
	}

	// Write .env file alongside docker-compose.local.yml so that Docker Compose
	// automatically applies the correct profiles on any `docker compose up` invocation,
	// including manual restarts that don't go through startLocalCoreServices.
	if err := t.writeComposeEnvFile(composePath); err != nil {
		t.logger.Warnf("Failed to write .env for COMPOSE_PROFILES (profiles may not persist on restart): %v", err)
	}

	// Generate prometheus.yml and copy into the monitoring volume
	if err := t.generatePrometheusConfig(ctx); err != nil {
		t.logger.Warnf("Failed to generate prometheus config (monitoring may not scrape L2 metrics): %v", err)
	}

	return nil
}

func (t *ThanosStack) generatePrometheusConfig(ctx context.Context) error {
	const promConfig = `global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: prometheus
    static_configs:
      - targets: ['localhost:9090']

  - job_name: op-node
    static_configs:
      - targets: ['op-node:7300']
        labels:
          service: op-node

  - job_name: op-geth
    metrics_path: /debug/metrics/prometheus
    static_configs:
      - targets: ['op-geth:6060']
        labels:
          service: op-geth

  - job_name: op-batcher
    static_configs:
      - targets: ['op-batcher:7302']
        labels:
          service: op-batcher

  - job_name: op-challenger
    static_configs:
      - targets: ['op-challenger:7304']
        labels:
          service: op-challenger
`

	const datasourceConfig = `apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    uid: prometheus
    url: http://prometheus:9090
    access: proxy
    isDefault: true
    editable: false
`

	const dashboardProviderConfig = `apiVersion: 1
providers:
  - name: default
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /monitoring/dashboards
`

	monitoringDir := filepath.Join(t.deploymentPath, "monitoring")
	dirs := []string{
		monitoringDir,
		filepath.Join(monitoringDir, "provisioning", "datasources"),
		filepath.Join(monitoringDir, "provisioning", "dashboards"),
		filepath.Join(monitoringDir, "dashboards"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", d, err)
		}
	}

	filesToWrite := map[string]string{
		filepath.Join(monitoringDir, "prometheus.yml"):                                 promConfig,
		filepath.Join(monitoringDir, "provisioning", "datasources", "prometheus.yaml"): datasourceConfig,
		filepath.Join(monitoringDir, "provisioning", "dashboards", "default.yaml"):     dashboardProviderConfig,
		filepath.Join(monitoringDir, "dashboards", "thanos-stack-application.json"):    grafanaDashboardApplication,
	}
	for path, content := range filesToWrite {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
		}
	}

	// Map host paths → destination names inside the monitoring volume
	monitoringFiles := map[string]string{
		"prometheus.yml": filepath.Join(monitoringDir, "prometheus.yml"),
		"provisioning/datasources/prometheus.yaml": filepath.Join(monitoringDir, "provisioning", "datasources", "prometheus.yaml"),
		"provisioning/dashboards/default.yaml":     filepath.Join(monitoringDir, "provisioning", "dashboards", "default.yaml"),
		"dashboards/thanos-stack-application.json": filepath.Join(monitoringDir, "dashboards", "thanos-stack-application.json"),
	}
	return t.copyFilesToMonitoringVolume(ctx, monitoringFiles)
}

func (t *ThanosStack) copyFilesToMonitoringVolume(ctx context.Context, files map[string]string) error {
	const helperName = "trh-monitoring-init"

	_, _ = utils.ExecuteCommand(ctx, "docker", "rm", "-f", helperName)

	containerID, err := utils.ExecuteCommand(ctx, "docker", "run", "-d",
		"--name", helperName,
		"-v", localMonitoringVolume+":/monitoring",
		"alpine", "sleep", "infinity")
	if err != nil {
		return fmt.Errorf("failed to start monitoring helper container: %w", err)
	}
	containerID = lastNonEmptyLine(containerID)
	defer utils.ExecuteCommand(ctx, "docker", "rm", "-f", containerID)

	// Collect unique parent directories and create them first.
	dirs := map[string]struct{}{}
	for destName := range files {
		if d := filepath.Dir(destName); d != "." {
			dirs[d] = struct{}{}
		}
	}
	for d := range dirs {
		if _, err := utils.ExecuteCommand(ctx, "docker", "exec", containerID,
			"mkdir", "-p", "/monitoring/"+d); err != nil {
			return fmt.Errorf("failed to create dir /monitoring/%s in volume: %w", d, err)
		}
	}

	for destName, srcPath := range files {
		if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "cp",
			srcPath, containerID+":/monitoring/"+destName); err != nil {
			return fmt.Errorf("failed to copy %s into monitoring volume: %w", destName, err)
		}
	}
	return nil
}

func (t *ThanosStack) initLocalOpGeth(ctx context.Context, composePath string) error {
	// Generate JWT secret if it doesn't exist
	jwtPath := filepath.Join(t.deploymentPath, "jwt.txt")
	if _, err := os.Stat(jwtPath); os.IsNotExist(err) {
		if err := generateJWTSecret(jwtPath); err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
	}

	genesisPath := filepath.Join(t.deploymentPath, "genesis.json")
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
	containerID = lastNonEmptyLine(containerID)
	defer utils.ExecuteCommand(ctx, "docker", "rm", "-f", containerID)

	for destName, srcPath := range files {
		if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "cp",
			srcPath, containerID+":/config/"+destName); err != nil {
			return fmt.Errorf("failed to copy %s into config volume: %w", destName, err)
		}
	}
	return nil
}

// lastNonEmptyLine extracts the last non-empty line from a string.
// This is used to parse the container ID from `docker run -d` output, which may
// include image pull progress lines before the actual 64-char container ID.
func lastNonEmptyLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return strings.TrimSpace(s)
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
		args = append(args, "--profile", "challenger", "--profile", "proposer")
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
		args = append(args, "--profile", "challenger", "--profile", "proposer")
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
		// Compose file is missing (e.g. backend container was restarted after deployment).
		// Fall back to removing known volumes directly so they don't block re-deployment.
		t.logger.Warn("Local compose file not found; removing volumes directly")
		return t.cleanLocalVolumesWithoutCompose(ctx)
	}
	t.logger.Info("Stopping local L2 network...")
	allProfiles := []string{"proposer", "challenger", "bridge", "blockExplorer", "monitoring", "uptimeService", "aa", "crossTrade"}
	args := []string{"compose", "-f", composePath}
	for _, p := range allProfiles {
		args = append(args, "--profile", p)
	}
	args = append(args, "down", "-v", "--remove-orphans")
	return utils.ExecuteCommandStream(ctx, t.logger, "docker", args...)
}

// cleanLocalVolumesWithoutCompose removes the well-known local L2 Docker volumes
// when the compose file is unavailable. Errors are logged but ignored because
// the volumes may already be absent.
func (t *ThanosStack) cleanLocalVolumesWithoutCompose(ctx context.Context) error {
	projectName := filepath.Base(t.deploymentPath)
	volumes := []string{
		localConfigVolume,
		localMonitoringVolume,
		projectName + "_op-geth-data",
	}
	t.logger.Infof("Removing orphaned local L2 volumes: %v", volumes)
	args := append([]string{"volume", "rm", "-f"}, volumes...)
	if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", args...); err != nil {
		t.logger.Warnf("Volume removal encountered errors (volumes may already be absent): %v", err)
	}
	return nil
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
	if t.deployConfig.CrossTradeContracts != nil {
		t.logger.Infof("  CrossTrade UI: http://localhost:3004")
	}
	t.logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// writeComposeEnvFile writes a .env file next to the docker-compose.local.yml
// with COMPOSE_PROFILES set based on the deployment configuration.
// Docker Compose reads .env from the same directory as the compose file,
// ensuring profiles (proposer, challenger, etc.) are automatically applied
// on any restart without needing explicit --profile flags.
func (t *ThanosStack) writeComposeEnvFile(composePath string) error {
	envDir := filepath.Dir(composePath)
	envPath := filepath.Join(envDir, ".env")

	// Determine required profiles
	profiles := []string{"proposer"}
	if t.deployConfig.EnableFraudProof {
		profiles = append(profiles, "challenger")
	}

	// Collect module profiles from preset
	modules := constants.PresetModules[t.deployConfig.Preset]
	for module, enabled := range modules {
		if !enabled || module == "crossTrade" || module == "drb" {
			continue
		}
		profiles = append(profiles, module)
	}

	content := fmt.Sprintf("COMPOSE_PROFILES=%s\n", strings.Join(profiles, ","))
	return os.WriteFile(envPath, []byte(content), 0644)
}

// persistAAProfile appends the "aa" profile to the COMPOSE_PROFILES in the .env
// file next to the compose file. Called only after AA setup + alto-bundler start
// succeeds, so that restarts (docker compose up) include alto-bundler only when
// the admin wallet is already funded and paymaster is configured.
func (t *ThanosStack) persistAAProfile(composePath string) error {
	envPath := filepath.Join(filepath.Dir(composePath), ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	content := string(data)
	// Check if aa is already present (idempotent)
	if strings.Contains(content, ",aa") || strings.Contains(content, "aa,") {
		return nil
	}
	// Find the COMPOSE_PROFILES line and append ,aa to its value
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "COMPOSE_PROFILES=") {
			lines[i] = strings.TrimRight(line, "\r") + ",aa"
			break
		}
	}
	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644)
}

// readRollupL1GenesisBlock reads genesis.l1.number from rollup.json so Blockscout
// knows which L1 block to start scanning from for deposits/withdrawals/batches.
// Returns 0 on any error (Blockscout will scan from genesis, which is slower but safe).
func readRollupL1GenesisBlock(rollupPath string, logger interface{ Warnf(string, ...any) }) uint64 {
	data, err := os.ReadFile(rollupPath)
	if err != nil {
		logger.Warnf("Could not read rollup.json for L1 start block: %v", err)
		return 0
	}
	var rollup struct {
		Genesis struct {
			L1 struct {
				Number uint64 `json:"number"`
			} `json:"l1"`
		} `json:"genesis"`
	}
	if err := json.Unmarshal(data, &rollup); err != nil {
		logger.Warnf("Could not parse rollup.json for L1 start block: %v", err)
		return 0
	}
	return rollup.Genesis.L1.Number
}

func generateJWTSecret(path string) error {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(secret)), 0600)
}

// crossTradeL2TokenEntry is the per-token JSON structure used in L2_L2 config's L2 chain entry.
// The CrossTrade dApp expects an array of these (not a flat map) for source L2 chains.
type crossTradeL2TokenEntry struct {
	Name              string   `json:"name"`
	Address           string   `json:"address"`
	DestinationChains []uint64 `json:"destination_chains"`
}

// Thanos Sepolia fixed partner constants for L2_L2 bridging.
const (
	crossTradeThanosSepolia        uint64 = 111551119090
	crossTradeThanosSepL2CTProxy          = "0x7BbEC445F9BDF6c579e81EAda5df86654184BcE3"
	crossTradeThanosSepRPCURL             = "https://rpc.thanos-sepolia.tokamak.network"
	crossTradeThanosSepExplorerURL        = "https://explorer.thanos-sepolia-test.tokamak.network"
	// crossTradeSepoliaL2toL2L1 is the L2toL2CrossTradeProxyL1 address on Sepolia deployed by
	// the Tokamak team. Used as the L1-side contract in NEXT_PUBLIC_CHAIN_CONFIG_L2_L2.
	crossTradeSepoliaL2toL2L1 = "0xF09Af74810010a0e9A452f71B3921641350c21D0"
)

// buildCrossTradeChainConfigL2L1JSON constructs the NEXT_PUBLIC_CHAIN_CONFIG_L2_L1 JSON value
// required by the CrossTrade dApp for L2→L1 bridging. Uses flat-map token format (old format)
// expected by the L2_L1 dApp path.
func (t *ThanosStack) buildCrossTradeChainConfigL2L1JSON(ct *types.CrossTradeLocalContracts) (string, error) {
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID
	l1Config := constants.L1ChainConfigurations[l1ChainID]

	l1CTProxy := ct.L1CrossTradeProxy
	l2CTProxy := ct.L2CrossTradeProxy

	chainConfig := map[string]interface{}{
		fmt.Sprintf("%d", l1ChainID): map[string]interface{}{
			"name":                l1Config.ChainName,
			"display_name":        "Ethereum",
			"native_token_name":   l1Config.NativeTokenName,
			"native_token_symbol": l1Config.NativeTokenSymbol,
			"rpc_url":             "",
			"contracts":           map[string]string{"l1_cross_trade": l1CTProxy},
			"tokens": map[string]string{
				"ETH":  "0x0000000000000000000000000000000000000000",
				"USDC": l1Config.USDCAddress,
				"USDT": l1Config.USDTAddress,
				"TON":  l1Config.TON,
			},
		},
		fmt.Sprintf("%d", l2ChainID): map[string]interface{}{
			"name":                fmt.Sprintf("%d", l2ChainID),
			"display_name":        t.deployConfig.ChainName,
			"native_token_name":   "Tokamak Network",
			"native_token_symbol": "TON",
			"rpc_url":             "http://host.docker.internal:8545",
			"contracts":           map[string]string{"l2_cross_trade": l2CTProxy},
			"tokens": map[string]string{
				"ETH":  "0x0000000000000000000000000000000000000000",
				"USDC": constants.USDCAddress,
				"USDT": "",
				"TON":  "",
			},
		},
	}

	b, err := json.Marshal(chainConfig)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// buildCrossTradeChainConfigL2L2JSON constructs the NEXT_PUBLIC_CHAIN_CONFIG_L2_L2 JSON value
// required by the CrossTrade dApp for L2→L2 bridging. Uses array-format tokens with
// destination_chains so the dApp can resolve allowed routing.
//
// Thanos Sepolia (111551119090) is always included as a fixed bridge partner.
// Its destination_chains is intentionally empty: the Thanos Sepolia L2toL2CrossTradeProxy
// has not registered the newly deployed L2's chainId, so Thanos→newL2 requests would fail
// at gas estimation (wallet refuses to sign). The reverse direction (newL2→Thanos) still works.
func (t *ThanosStack) buildCrossTradeChainConfigL2L2JSON(ct *types.CrossTradeLocalContracts) (string, error) {
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID
	l1Config := constants.L1ChainConfigurations[l1ChainID]

	l2l2CTProxy := ct.L2toL2CrossTradeProxy

	l2l2Tokens := []crossTradeL2TokenEntry{
		{Name: "ETH", Address: "0x0000000000000000000000000000000000000000", DestinationChains: []uint64{crossTradeThanosSepolia}},
		{Name: "USDC", Address: "", DestinationChains: []uint64{crossTradeThanosSepolia}},
	}

	thanosL2L2Tokens := []crossTradeL2TokenEntry{
		{Name: "ETH", Address: "0x4200000000000000000000000000000000000486", DestinationChains: []uint64{}},
		{Name: "TON", Address: "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000", DestinationChains: []uint64{}},
		{Name: "USDC", Address: "0x4200000000000000000000000000000000000778", DestinationChains: []uint64{}},
	}

	chainConfig := map[string]interface{}{
		fmt.Sprintf("%d", l1ChainID): map[string]interface{}{
			"name":                l1Config.ChainName,
			"display_name":        "Ethereum",
			"native_token_name":   l1Config.NativeTokenName,
			"native_token_symbol": l1Config.NativeTokenSymbol,
			"rpc_url":             "",
			"contracts":           map[string]string{"l1_cross_trade": crossTradeSepoliaL2toL2L1},
			"tokens": map[string]string{
				"ETH":  "0x0000000000000000000000000000000000000000",
				"USDC": l1Config.USDCAddress,
				"USDT": l1Config.USDTAddress,
				"TON":  l1Config.TON,
			},
		},
		fmt.Sprintf("%d", l2ChainID): map[string]interface{}{
			"name":                fmt.Sprintf("%d", l2ChainID),
			"display_name":        t.deployConfig.ChainName,
			"native_token_name":   "Tokamak Network",
			"native_token_symbol": "TON",
			"rpc_url":             "http://host.docker.internal:8545",
			"contracts":           map[string]string{"l2_cross_trade": l2l2CTProxy},
			"tokens":              l2l2Tokens,
		},
		fmt.Sprintf("%d", crossTradeThanosSepolia): map[string]interface{}{
			"name":                "Thanos Sepolia",
			"display_name":        "Thanos Sepolia",
			"native_token_name":   "Tokamak Network",
			"native_token_symbol": "TON",
			"rpc_url":             crossTradeThanosSepRPCURL,
			"block_explorer_url":  crossTradeThanosSepExplorerURL,
			"contracts":           map[string]string{"l2_cross_trade": crossTradeThanosSepL2CTProxy},
			"tokens":              thanosL2L2Tokens,
		},
	}

	b, err := json.Marshal(chainConfig)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// orchestrateDRBOperators executes the post-startup DRB orchestration:
// 1. Derives DRB accounts from mnemonic
// 2. Bootstraps peer ID files into Docker volumes
// 3. Activates Regular operators on-chain
func (t *ThanosStack) orchestrateDRBOperators(ctx context.Context, composePath string) error {
	// Derive DRB accounts from mnemonic
	accounts, err := DeriveDRBAccounts(t.deployConfig.Mnemonic)
	if err != nil {
		return fmt.Errorf("derive DRB accounts: %w", err)
	}

	// Inject peer ID files into static-key volumes
	projectName := filepath.Base(t.deploymentPath)
	t.logger.Infof("🔧 Bootstrapping DRB peer ID files into Docker volumes...")
	if err := BootstrapDRBPeerIDFiles(ctx, projectName, accounts); err != nil {
		return fmt.Errorf("bootstrap DRB peer ID files: %w", err)
	}
	t.logger.Info("✅ DRB peer ID files bootstrapped")

	// Wait for DRB containers to be healthy (simple implementation: just wait a bit)
	t.logger.Infof("⏳ Waiting for DRB containers to stabilize...")
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Activate Regular operators on-chain
	t.logger.Infof("📡 Activating Regular DRB operators on-chain...")
	l2RPC := localL2RPCURL()
	contractAddr := "0x4200000000000000000000000000000000000060" // CommitReveal2L2 predeploy address
	threshold := DefaultDRBGenesisConfig().ActivationThreshold
	if err := ActivateRegularOperators(ctx, l2RPC, contractAddr, accounts, threshold); err != nil {
		return fmt.Errorf("activate regular operators: %w", err)
	}
	t.logger.Info("✅ Regular DRB operators activated")

	return nil
}
