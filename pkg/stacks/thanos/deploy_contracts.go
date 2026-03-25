package thanos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shirou/gopsutil/mem"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ----------------------------------------- Deploy contracts command  ----------------------------- //

func (t *ThanosStack) DeployContracts(ctx context.Context, deployContractsConfig *DeployContractsInput) error {
	if t.network == constants.LocalDevnet {
		t.logger.Error("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet)
		return fmt.Errorf("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet)
	}
	if t.network != constants.Testnet && t.network != constants.Mainnet {
		t.logger.Error("network %s does not support", t.network)
		return fmt.Errorf("network %s does not support", t.network)
	}

	if deployContractsConfig == nil {
		t.logger.Error("deployContractsConfig is required")
		return fmt.Errorf("deployContractsConfig is required")
	}

	if deployContractsConfig.ChainConfiguration == nil {
		l1Client, err := ethclient.DialContext(ctx, deployContractsConfig.L1RPCurl)
		if err != nil {
			t.logger.Error("Failed to get the L1 ChainID", "err", err)
			return err
		}

		l1ChainID, err := l1Client.ChainID(ctx)
		if err != nil {
			t.logger.Error("Failed to get the L1 ChainID", "err", err)
			return err
		}

		finalzationPeriodSeconds := constants.L1ChainConfigurations[l1ChainID.Uint64()].FinalizationPeriodSeconds
		l2OutputSubmissionInterval := constants.L1ChainConfigurations[l1ChainID.Uint64()].L2OutputOracleSubmissionInterval
		maxChannelDuration := constants.L1ChainConfigurations[l1ChainID.Uint64()].MaxChannelDuration
		l2BlockTime := constants.DefaultL2BlockTimeInSeconds
		l1BlockTime := constants.L1ChainConfigurations[l1ChainID.Uint64()].BlockTimeInSeconds

		deployContractsConfig.ChainConfiguration = &types.ChainConfiguration{
			BatchSubmissionFrequency: maxChannelDuration * l1BlockTime,
			ChallengePeriod:          finalzationPeriodSeconds,
			OutputRootFrequency:      l2BlockTime * l2OutputSubmissionInterval,
			L2BlockTime:              l2BlockTime,
			L1BlockTime:              l1BlockTime,
		}
	}

	err := deployContractsConfig.Validate(ctx, t.registerCandidate)
	if err != nil {
		t.logger.Error("Error validating deployContractsConfig, err:", err)
		return err
	}

	var (
		isResume bool
	)

	if t.deployConfig == nil {
		t.deployConfig = &types.Config{
			Stack:   constants.ThanosStack,
			Network: t.network,
		}

		networkToChainID := map[string]uint64{
			constants.Testnet: constants.EthereumSepoliaChainID,
			constants.Mainnet: constants.EthereumMainnetChainID,
		}
		if chainID, ok := networkToChainID[t.network]; ok {
			chainConfig := constants.L1ChainConfigurations[chainID]
			t.deployConfig.TxmgrCellProofTime = chainConfig.TxmgrCellProofTime
			t.deployConfig.NextPublicRollupL1BaseUrl = chainConfig.NextPublicRollupL1BaseUrl
		}
	}

	l1Client, err := ethclient.DialContext(ctx, deployContractsConfig.L1RPCurl)
	if err != nil {
		t.logger.Error("Failed to get the L1 client", "err", err)
		return err
	}

	if t.deployConfig.DeployContractState != nil {
		switch t.deployConfig.DeployContractState.Status {
		case types.DeployContractStatusCompleted:
			t.logger.Info("The contracts have already been deployed successfully.")
			if t.usePromptInput {
				fmt.Print("Do you want to deploy the contracts again? (y/N): ")
				isDeployAgain, err := scanner.ScanBool(false)
				if err != nil {
					t.logger.Error("Error reading the deploy again input:", err)
					return err
				}

				if !isDeployAgain {
					return nil
				}
			} else {
				return nil
			}
		case types.DeployContractStatusInProgress:
			if t.usePromptInput {
				fmt.Print("The contracts deployment is in progress. Do you want to resume? (Y/n): ")
				isResume, err = scanner.ScanBool(true)
				if err != nil {
					t.logger.Error("Error reading the resume input:", err)
					return err
				}
			} else {
				isResume = true
			}
		}
	}

	registerCandidate := deployContractsConfig.RegisterCandidate

	if isResume {
		// Clone is always required so the source and submodules are available
		err = t.cloneSourcecode(ctx, "tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
		if err != nil {
			t.logger.Error("failed to clone the repository", "err", err)
			return err
		}

		err = t.deployContracts(ctx, l1Client, true)
		if err != nil {
			t.logger.Error("❌ Resume the contracts deployment failed!", "err", err)
			return err
		}
		if t.registerCandidate {
			if registerCandidate == nil {
				t.logger.Error("register candidate is required")
				return fmt.Errorf("register candidate is required")
			}

			adminAddress, err := utils.GetAddressFromPrivateKey(t.deployConfig.AdminPrivateKey)
			if err != nil {
				t.logger.Error("failed to get admin address from private key", "err", err)
				return fmt.Errorf("failed to get admin address from private key: %s", err)
			}
			err = t.checkAdminBalance(ctx, adminAddress, registerCandidate.Amount, l1Client)
			if err != nil {
				t.logger.Error("failed to check admin balance", "err", err)
				return fmt.Errorf("failed to check admin balance: %s", err)
			}
		}
	} else {
		l2ChainID, err := utils.GenerateL2ChainId()
		if err != nil {
			t.logger.Error("Failed to generate L2ChainID", "err", err)
			return err
		}

		l1ChainId, err := l1Client.ChainID(ctx)
		if err != nil {
			t.logger.Error("Failed to get L1 ChainID", "err", err)
			return err
		}

		operators := deployContractsConfig.Operators

		if operators == nil ||
			operators.AdminPrivateKey == "" ||
			operators.SequencerPrivateKey == "" ||
			operators.BatcherPrivateKey == "" ||
			operators.ProposerPrivateKey == "" {
			t.logger.Error("at least 5 operators are required for deploying contracts")
			return fmt.Errorf("at least 5 operators are required for deploying contracts")
		}

		adminAccount, err := utils.GetAddressFromPrivateKey(operators.AdminPrivateKey)
		if err != nil {
			t.logger.Error("failed to get admin address from private key", "err", err)
			return fmt.Errorf("failed to get admin address from private key: %s", err)
		}

		if t.registerCandidate {
			err = t.checkAdminBalance(ctx, adminAccount, registerCandidate.Amount, l1Client)
			if err != nil {
				t.logger.Error("failed to check admin balance", "err", err)
				return err
			}
		}

		if t.usePromptInput {
			fmt.Print("🔎 The SDK is ready to deploy the contracts to the L1 network. Do you want to proceed(Y/n)? ")
			confirmation, err := scanner.ScanBool(true)
			if err != nil {
				t.logger.Error("failed to read the confirmation input", "err", err)
				return err
			}
			if !confirmation {
				return nil
			}
		}

		shellConfigFile := utils.GetShellConfigDefault()

		// Check dependencies
		if !dependencies.CheckPnpmInstallation(ctx) {
			t.logger.Warn("pnpm is not installed, try running `source %s` to set up your environment", shellConfigFile)
			return nil
		}

		if !dependencies.CheckFoundryInstallation(ctx) {
			t.logger.Warn("foundry is not installed, try running `source %s` to set up your environment", shellConfigFile)
			return nil
		}

		// STEP 2. Clone the repository (always required regardless of ReuseDeployment)
		err = t.cloneSourcecode(ctx, "tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
		if err != nil {
			t.logger.Error("failed to clone the repository", "err", err)
			return err
		}

		// STEP 2.1. Patch AnchorStateRegistry.sol to add setInitialAnchorState.
		// The upstream contract requires a resolved FaultDisputeGame to set anchor state,
		// which is impossible on a brand-new chain (bootstrapping problem). This patch
		// adds a guardian-only function that accepts an OutputRoot directly.
		if deployContractsConfig.EnableFaultProof {
			tokamakThanosDir := filepath.Join(t.deploymentPath, "tokamak-thanos")
			if patchErr := patchAnchorStateRegistry(tokamakThanosDir); patchErr != nil {
				t.logger.Error("Failed to patch AnchorStateRegistry.sol", "err", patchErr)
				return fmt.Errorf("failed to patch AnchorStateRegistry.sol: %w", patchErr)
			}
			t.logger.Info("✅ AnchorStateRegistry.sol patched with setInitialAnchorState")
		}

		// STEP 2.5. When fault proof is enabled, build cannon-prestate and extract the
		// prestate hash BEFORE generating the deploy config. This ensures FaultGameAbsolutePrestate
		// in the contract config matches the actual op-program binary hash, not a stale hardcoded value.
		var prestateHash string
		if deployContractsConfig.EnableFaultProof {
			tokamakThanosDir := filepath.Join(t.deploymentPath, "tokamak-thanos")
			prestatePath := filepath.Join(tokamakThanosDir, "op-program", "bin", "prestate.json")
			if _, statErr := os.Stat(prestatePath); os.IsNotExist(statErr) {
				t.logger.Info("Building cannon prestate before contract deployment...")
				if buildErr := buildCannonPrestate(ctx, t.logger, tokamakThanosDir); buildErr != nil {
					t.logger.Error("Failed to build cannon prestate", "err", buildErr)
					return fmt.Errorf("failed to build cannon prestate: %w", buildErr)
				}
				t.logger.Info("✅ Cannon prestate built successfully")
			} else {
				t.logger.Info("Cannon prestate already exists, reading hash", "path", prestatePath)
			}
			prestateHash, err = readPrestateHash(prestatePath)
			if err != nil {
				t.logger.Error("Failed to read cannon prestate hash", "err", err)
				return err
			}
			t.logger.Info("Cannon prestate hash loaded", "hash", prestateHash)
		}

		deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig, l1ChainId.Uint64(), l2ChainID, prestateHash)

		t.deployConfig.AdminPrivateKey = operators.AdminPrivateKey
		t.deployConfig.SequencerPrivateKey = operators.SequencerPrivateKey
		t.deployConfig.BatcherPrivateKey = operators.BatcherPrivateKey
		t.deployConfig.ProposerPrivateKey = operators.ProposerPrivateKey
		if deployContractsConfig.EnableFaultProof {
			if operators.ChallengerPrivateKey == "" {
				return fmt.Errorf("challenger operator is required for fault proof but was not found")
			}
			t.deployConfig.ChallengerPrivateKey = operators.ChallengerPrivateKey
		}
		t.deployConfig.DeploymentFilePath = fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%d-deploy.json", t.deploymentPath, deployContractsTemplate.L1ChainID)
		t.deployConfig.L1RPCProvider = utils.DetectRPCKind(deployContractsConfig.L1RPCurl)
		t.deployConfig.L1ChainID = deployContractsTemplate.L1ChainID
		t.deployConfig.L2ChainID = l2ChainID
		t.deployConfig.L1RPCURL = deployContractsConfig.L1RPCurl
		t.deployConfig.EnableFraudProof = deployContractsConfig.EnableFaultProof
		t.deployConfig.ChainConfiguration = deployContractsConfig.ChainConfiguration
		t.deployConfig.Preset = deployContractsConfig.Preset
		t.deployConfig.FeeToken = deployContractsConfig.FeeToken

		deployConfigFilePath := fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts/deploy-config.json", t.deploymentPath)

		err = makeDeployContractConfigJsonFile(ctx, l1Client, operators, deployContractsTemplate, deployConfigFilePath)
		if err != nil {
			t.logger.Error("failed to make deploy contract config json file", "err", err)
			return err
		}

		// STEP 2.9. Patch start-deploy.sh for known build issues.
		// (a) cannon prestate: upstream buildSource() always runs `make cannon-prestate`, but
		//     the cannon binary requires --type flag that the Makefile doesn't pass. Skip when
		//     fault proof is disabled.
		// (b) op-node: the tokamak-thanos repo migrated from Makefile to justfiles; the
		//     `make op-node` target is now deprecated and fails. Replace with direct go build.
		tokamakThanosDir := filepath.Join(t.deploymentPath, "tokamak-thanos")
		if patchErr := patchStartDeployScript(tokamakThanosDir); patchErr != nil {
			t.logger.Warn("Failed to patch start-deploy.sh, build may fail", "err", patchErr)
		} else {
			t.logger.Info("✅ start-deploy.sh patched: cannon prestate and op-node build fixed")
		}

		// Patch workspace dependency: pnpm overrides don't resolve workspace:* references.
		// Replace the self-referencing devDependency with the actual package name.
		if patchErr := patchContractsPkgJSON(tokamakThanosDir); patchErr != nil {
			t.logger.Warn("Failed to patch package.json workspace dep", "err", patchErr)
		}

		// STEP 3. Build the contracts
		contractsDir := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock")

		// Remove broken test files that prevent forge build (upstream constructor mismatch).
		removeBrokenTestFiles(t.logger, contractsDir)
		scriptsDir := filepath.Join(contractsDir, "scripts")
		forgeCacheDir := filepath.Join(t.deploymentPath, ".forge-cache")

		// Try downloading pre-built artifacts from npm to skip forge build (~5min → ~10s)
		if dlErr := downloadPrebuiltArtifacts(ctx, t.logger, contractsDir); dlErr != nil {
			t.logger.Warn("Pre-built artifacts unavailable, will build from source", "err", dlErr)
			// Restore cached forge artifacts if available (survives re-clones)
			restoreForgeCache(t.logger, forgeCacheDir, contractsDir)
		} else {
			t.logger.Info("✅ Pre-built artifacts downloaded, forge build will be skipped")
			os.Setenv("SKIP_FORGE_BUILD", "true")
			defer os.Unsetenv("SKIP_FORGE_BUILD")

			// Invalidate cache for patched files so forge recompiles only those contracts
			if deployContractsConfig.EnableFaultProof {
				if cacheErr := invalidateCacheEntry(contractsDir, "src/dispute/AnchorStateRegistry.sol"); cacheErr != nil {
					t.logger.Warn("Failed to invalidate cache for patched contract", "err", cacheErr)
				}
			}
		}

		t.logger.Info("Building smart contracts...")
		err = utils.ExecuteCommandStreamInDir(ctx, t.logger, scriptsDir, "bash", "./start-deploy.sh", "build")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				t.logger.Warn("Deployment canceled")
				return err
			}
			t.logger.Error("❌ Build the contracts failed!")
			return err
		}
		t.logger.Info("✅ Build the contracts completed!")

		// Cache forge artifacts for future builds
		saveForgeCache(t.logger, forgeCacheDir, contractsDir)

		// STEP 4. Deploy the contracts
		// Check admin balance and estimated deployment cost
		balance, err := l1Client.BalanceAt(ctx, adminAccount, nil)
		if err != nil {
			t.logger.Error("❌ Failed to retrieve admin account balance", "err", err)
			return err
		}
		t.logger.Infof("Admin account balance: %.2f ETH", utils.WeiToEther(balance))

		// Estimate gas price
		gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
		if err != nil {
			t.logger.Error("❌ Failed to get gas price", "err", err)
			return err
		}
		t.logger.Infof("⛽ Current gas price: %.4f Gwei", new(big.Float).Quo(new(big.Float).SetInt(gasPriceWei), big.NewFloat(1e9)))

		// Estimate deployment cost
		estimatedCost := new(big.Int).Mul(gasPriceWei, estimatedDeployContracts)
		estimatedCost.Mul(estimatedCost, big.NewInt(2))
		t.logger.Infof("💰 Estimated deployment cost: %.4f ETH", utils.WeiToEther(estimatedCost))

		// Check if balance is sufficient
		if balance.Cmp(estimatedCost) < 0 {
			t.logger.Error("❌ Insufficient balance for deployment.")
			return fmt.Errorf("admin account balance (%.4f ETH) is less than estimated deployment cost (%.4f  ETH)", utils.WeiToEther(balance), utils.WeiToEther(estimatedCost))
		} else {
			t.logger.Info("✅ The admin account has sufficient balance to proceed with deployment.")
		}

		t.deployConfig.DeployContractState = &types.DeployContractState{
			Status: types.DeployContractStatusInProgress,
		}
		err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
		if err != nil {
			t.logger.Error("Failed to write settings file", "err", err)
			return err
		}

		if deployContractsConfig.ReuseDeployment {
			t.logger.Info("ℹ️ ReuseDeployment: Deploying with existing implementation contracts...")
		}

		// Deploy contracts. If ReuseDeployment is true, deploy-config.json instructs the deploy
		// script to reuse existing implementation contracts on the network.
		err = t.deployContracts(ctx, l1Client, false)
		if err != nil {
			t.logger.Error("failed to deploy contracts", "err", err)
			return err
		}
	}

	// STEP 5: Generate the genesis and rollup files
	scriptsDir := filepath.Join(t.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "scripts")
	err = utils.ExecuteCommandStreamInDir(ctx, t.logger, scriptsDir, "bash", "./start-deploy.sh", "generate", "-e", ".env", "-c", "deploy-config.json")
	t.logger.Info("Generating the rollup and genesis files...")
	if err != nil {
		if errors.Is(err, context.Canceled) {
			t.logger.Warn("Deployment canceled")
			return err
		}
		t.logger.Error("❌ Failed to generate rollup and genesis files!")
		return err
	}
	t.logger.Info("✅ Successfully generated rollup and genesis files!")
	genesisPath := filepath.Join(t.deploymentPath, "tokamak-thanos", "build", "genesis.json")
	t.logger.Infof("Genesis file path: %s", genesisPath)
	t.logger.Infof("Rollup file path: %s/tokamak-thanos/build/rollup.json", t.deploymentPath)

	// STEP 5.1: Inject DRB predeploy into genesis for Gaming/Full presets
	if t.deployConfig.Preset == constants.PresetGaming || t.deployConfig.Preset == constants.PresetFull {
		t.logger.Info("Injecting DRB (CommitReveal2L2) predeploy into genesis...")
		drbConfig := DefaultDRBGenesisConfig()
		if err := injectDRBIntoGenesis(ctx, t.logger, genesisPath, drbConfig); err != nil {
			t.logger.Error("❌ Failed to inject DRB into genesis!", "err", err)
			return err
		}
	}

	t.logger.Infof("✅ Configuration successfully saved to: %s/settings.json", t.deploymentPath)

	// If --no-candidate flag is NOT provided, register the candidate
	if t.registerCandidate {
		t.logger.Info("Setting up the safe wallet...")

		if err := t.setupSafeWallet(ctx, t.deploymentPath); err != nil {
			return err
		}
		t.logger.Info("🔍 Verifying and registering candidate...")
		verifyRegisterError := t.verifyRegisterCandidates(ctx, registerCandidate)
		if verifyRegisterError != nil {
			return fmt.Errorf("candidate registration failed: %v", verifyRegisterError)
		}

		// Display additional registration information
		t.DisplayRegistrationAdditionalInfo(ctx, registerCandidate)
	} else {
		t.logger.Info("ℹ️ Skipping candidate registration (--no-candidate flag provided)")
	}

	return nil
}

func (t *ThanosStack) deployContracts(ctx context.Context,
	l1Client *ethclient.Client,
	isResume bool,
) error {
	var (
		adminPrivateKey = t.deployConfig.AdminPrivateKey
		l1RPC           = t.deployConfig.L1RPCURL
	)

	t.logger.Info("Deploying the contracts...")

	gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		t.logger.Error("Failed to get gas price", "err", err)
	}

	envValues := fmt.Sprintf("export GS_ADMIN_PRIVATE_KEY=%s\nexport L1_RPC_URL=%s\n", adminPrivateKey, l1RPC)
	if gasPriceWei != nil && gasPriceWei.Uint64() > 0 {
		// double gas price
		envValues += fmt.Sprintf("export GAS_PRICE=%d\n", gasPriceWei.Uint64()*2)
	}

	// STEP 4.1. Generate the .env file using native Go file write (avoids shell injection)
	scriptsDir := filepath.Join(t.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "scripts")
	envFilePath := filepath.Join(scriptsDir, ".env")
	err = os.WriteFile(envFilePath, []byte(envValues), 0600)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			t.logger.Warn("Deployment canceled")
			return err
		}
		t.logger.Error("❌ Make .env file failed!")
		return err
	}

	// STEP 4.3. Deploy contracts
	if isResume {
		err = utils.ExecuteCommandStreamInDir(ctx, t.logger, scriptsDir, "bash", "./start-deploy.sh", "redeploy", "-e", ".env", "-c", "deploy-config.json")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				t.logger.Warn("Deployment canceled")
				return err
			}
			t.logger.Error("❌ Contract deployment failed!")
			return err
		}
	} else {
		err = utils.ExecuteCommandStreamInDir(ctx, t.logger, scriptsDir, "bash", "./start-deploy.sh", "deploy", "-e", ".env", "-c", "deploy-config.json")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				t.logger.Warn("Deployment canceled")
				return err
			}
			t.logger.Error("❌ Contract deployment failed!")
			return err
		}
	}
	t.logger.Info("✅ Contract deployment completed successfully!")

	t.deployConfig.DeployContractState.Status = types.DeployContractStatusCompleted
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Error("Failed to write settings file", "err", err)
		return err
	}
	return nil
}

// patchStartDeployScript patches the start-deploy.sh script in the cloned tokamak-thanos
// repository to fix known build issues:
//  1. Cannon prestate: skip when fault proof is disabled (cannon CLI changed to require --type).
//  2. op-node: replace deprecated `make op-node` with direct go build (Makefile migrated to justfiles).
func patchStartDeployScript(tokamakThanosDir string) error {
	scriptPath := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock", "scripts", "start-deploy.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read start-deploy.sh: %w", err)
	}

	// Patch 1: cannon prestate — skip when fault proof is disabled.
	cannonOld := `  # Build cannon prestate
  echo "Building cannon prestate..."
  if ! retryCommand "make cannon-prestate" "Building cannon prestate"; then
    echo "❌ Error: Failed to build cannon prestate after $MAX_RETRIES attempts"
    return 1
  fi`

	cannonNew := `  # Build cannon prestate (only when fault proof is enabled)
  if [ "${ENABLE_FAULT_PROOF:-false}" = "true" ]; then
    echo "Building cannon prestate..."
    if ! retryCommand "make cannon-prestate" "Building cannon prestate"; then
      echo "❌ Error: Failed to build cannon prestate after $MAX_RETRIES attempts"
      return 1
    fi
  else
    echo "ℹ️ Skipping cannon prestate build (ENABLE_FAULT_PROOF not set)"
  fi`

	if bytes.Contains(content, []byte(cannonOld)) {
		content = bytes.Replace(content, []byte(cannonOld), []byte(cannonNew), 1)
	}

	// Patch 2: op-node build — replace deprecated `make op-node` with direct go build.
	// The tokamak-thanos repo migrated from Makefile to justfiles; `make op-node` now errors.
	opNodeOld := `  if ! retryCommand "make op-node" "Building op-node"; then`
	opNodeNew := `  if ! retryCommand "(cd $projectRoot/op-node && env GO111MODULE=on CGO_ENABLED=0 go build -v -o ./bin/op-node ./cmd)" "Building op-node"; then`

	if bytes.Contains(content, []byte(opNodeOld)) {
		content = bytes.Replace(content, []byte(opNodeOld), []byte(opNodeNew), 1)
	}

	// Patch 3: forge artifacts symlinks — newer forge versions name artifacts with solidity version
	// suffix (e.g. L1UsdcBridge.0.8.15.json) but the TypeScript SDK imports the unversioned name
	// (e.g. L1UsdcBridge.json). Create symlinks before the SDK TypeScript build.
	sdkBuildOld := `  # Build SDK with retry logic
  if ! retryCommand "pnpm build" "Building SDK"; then`
	sdkBuildNew := `  # Create non-versioned symlinks for versioned forge artifacts (needed for TypeScript resolution)
  echo "Creating artifact symlinks for TypeScript resolution..."
  find $projectRoot/packages/tokamak/contracts-bedrock/forge-artifacts -name "*.json" | while read f; do
    dir=$(dirname "$f")
    base=$(basename "$f")
    nonversioned=$(echo "$base" | sed 's/\(\.[0-9][0-9]*\)\+\.json$/.json/')
    if [ "$base" != "$nonversioned" ] && [ ! -f "$dir/$nonversioned" ]; then
      ln -sf "$base" "$dir/$nonversioned"
    fi
  done
  echo "✅ Artifact symlinks created"

  # Build SDK with retry logic
  if ! retryCommand "pnpm build" "Building SDK"; then`

	if bytes.Contains(content, []byte(sdkBuildOld)) {
		content = bytes.Replace(content, []byte(sdkBuildOld), []byte(sdkBuildNew), 1)
	}

	// Patch 4: incremental builds + memory-based parallelism.
	// - Skip `forge clean` when forge-artifacts already exist to enable incremental builds
	//   (Forge uses solidity-files-cache.json to recompile only changed files).
	// - Dynamically set --jobs based on available system memory to balance speed vs OOM risk.
	//   Docker Desktop defaults to 7.75GB which only supports --jobs 1.
	// Replace the entire retryCommand block (3 lines) to avoid nested-quote shell issues.
	jobs := getForgeJobs()
	jobsStr := strconv.Itoa(jobs)
	forgeBuildBlockOld := `  if ! retryCommand "forge clean && forge build" "Building contracts"; then
    echo "❌ Error: Failed to build contracts after $MAX_RETRIES attempts"
    return 1
  fi`
	forgeBuildBlockNew := `  # Incremental build with memory-based parallelism (patched by trh-sdk)
  if [ "${SKIP_FORGE_BUILD:-false}" = "true" ] && [ -d "forge-artifacts" ] && [ -n "$(ls -A forge-artifacts 2>/dev/null)" ]; then
    echo "Pre-built artifacts found, skipping forge build"
  elif [ -d "forge-artifacts" ] && [ -n "$(ls -A forge-artifacts 2>/dev/null)" ]; then
    echo "Incremental build (reusing cached artifacts)..."
    if ! retryCommand "forge build --jobs ` + jobsStr + `" "Building contracts (incremental)"; then
      echo "❌ Error: Failed to build contracts after $MAX_RETRIES attempts"
      return 1
    fi
  else
    echo "Clean build (no cached artifacts found)..."
    if ! retryCommand "forge clean && forge build --jobs ` + jobsStr + `" "Building contracts (clean)"; then
      echo "❌ Error: Failed to build contracts after $MAX_RETRIES attempts"
      return 1
    fi
  fi`
	if bytes.Contains(content, []byte(forgeBuildBlockOld)) {
		content = bytes.Replace(content, []byte(forgeBuildBlockOld), []byte(forgeBuildBlockNew), 1)
	}

	// Patch 5: remove redundant waitForFileSystem between TypeScript builds.
	// Modern filesystems (ext4, overlay2) don't need sleep+sync between sequential builds.
	// Keep only the one after forge build for artifact verification.
	// There are exactly 2 occurrences (before core-utils and before SDK builds).
	waitOld := []byte(`  # Additional wait to ensure modules are properly synced
  waitForFileSystem`)
	waitNew := []byte(`  # filesystem sync skipped (not needed between sequential builds)`)
	content = bytes.Replace(content, waitOld, waitNew, 2)

	// Patch 6: shallow submodule update instead of full history clone.
	submoduleOld := `make submodules`
	submoduleNew := `git submodule update --init --recursive --depth 1`
	if bytes.Contains(content, []byte(submoduleOld)) {
		content = bytes.Replace(content, []byte(submoduleOld), []byte(submoduleNew), 1)
	}

	// Patch 7: skip TypeScript builds if already built (enables fast re-deploys).
	coreUtilsBuildOld := `  if ! retryCommand "pnpm build" "Building core-utils"; then`
	coreUtilsBuildNew := `  if [ -f "dist/index.js" ]; then
    echo "core-utils already built, skipping"
  elif ! retryCommand "pnpm build" "Building core-utils"; then`
	if bytes.Contains(content, []byte(coreUtilsBuildOld)) {
		content = bytes.Replace(content, []byte(coreUtilsBuildOld), []byte(coreUtilsBuildNew), 1)
	}

	return os.WriteFile(scriptPath, content, 0755)
}

// removeBrokenTestFiles deletes test files that have compilation errors (e.g. constructor
// argument mismatches) so they don't block `forge build`. These are test-only files that
// are not needed for contract deployment.
func removeBrokenTestFiles(logger interface{ Info(args ...interface{}) }, contractsDir string) {
	brokenTests := []string{
		"test/AA/MultiTokenPaymaster.t.sol",
		"test/AA/SimplePriceOracle.t.sol",
	}
	for _, f := range brokenTests {
		path := filepath.Join(contractsDir, f)
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err == nil {
				logger.Info("Removed broken test file: " + f)
			}
		}
	}
}

// patchContractsPkgJSON fixes the workspace self-reference in contracts-bedrock/package.json.
// The package declares "@eth-optimism/contracts-bedrock": "workspace:*" in devDependencies,
// but pnpm overrides don't resolve workspace: protocol references. Replace with the actual
// tokamak package name so pnpm can resolve it within the workspace.
func patchContractsPkgJSON(tokamakThanosDir string) error {
	pkgPath := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock", "package.json")
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}

	old := []byte(`"@eth-optimism/contracts-bedrock": "workspace:*"`)
	replacement := []byte(`"@tokamak-network/thanos-contracts": "workspace:*"`)
	if bytes.Contains(content, old) {
		content = bytes.Replace(content, old, replacement, 1)
		return os.WriteFile(pkgPath, content, 0644)
	}
	return nil
}

// getForgeJobs returns the number of parallel Solc compilation jobs based on available memory.
// Forge compiles up to 6 Solc versions in parallel by default, which can exhaust memory
// on resource-constrained hosts like Docker Desktop (7.75GB default).
func getForgeJobs() int {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 1
	}
	totalGB := v.Total / (1024 * 1024 * 1024)
	switch {
	case totalGB >= 16:
		return 4
	case totalGB >= 10:
		return 2
	default:
		return 1
	}
}

// restoreForgeCache restores cached forge build artifacts from the deployment-level cache
// directory into the contracts-bedrock directory. This enables incremental builds even after
// re-cloning the tokamak-thanos repository, since the cache lives outside the repo directory.
// Cache errors are logged but never fail the build — caching is best-effort.
func restoreForgeCache(logger interface{ Info(args ...interface{}) }, cacheDir, contractsDir string) {
	forgeArtifactsCache := filepath.Join(cacheDir, "forge-artifacts")
	forgeSolcCache := filepath.Join(cacheDir, "cache")

	if _, err := os.Stat(forgeArtifactsCache); os.IsNotExist(err) {
		return
	}

	destArtifacts := filepath.Join(contractsDir, "forge-artifacts")
	destCache := filepath.Join(contractsDir, "cache")

	// Only restore if the destination doesn't already have artifacts
	if _, err := os.Stat(destArtifacts); !os.IsNotExist(err) {
		return
	}

	logger.Info("Restoring cached forge artifacts for incremental build...")

	// Always copy (not rename) to preserve the cache for future use
	if err := copyDir(forgeArtifactsCache, destArtifacts); err != nil {
		logger.Info("Failed to restore forge artifacts cache, will do clean build", err)
		os.RemoveAll(destArtifacts) // Clean up partial copy
		return
	}
	if _, err := os.Stat(forgeSolcCache); !os.IsNotExist(err) {
		if err := copyDir(forgeSolcCache, destCache); err != nil {
			logger.Info("Failed to restore forge solc cache", err)
			os.RemoveAll(destCache)
		}
	}
}

// saveForgeCache saves forge build artifacts to the deployment-level cache directory.
// Cache errors are logged but never fail the build — caching is best-effort.
func saveForgeCache(logger interface{ Info(args ...interface{}) }, cacheDir, contractsDir string) {
	srcArtifacts := filepath.Join(contractsDir, "forge-artifacts")
	srcCache := filepath.Join(contractsDir, "cache")

	if _, err := os.Stat(srcArtifacts); os.IsNotExist(err) {
		return
	}

	logger.Info("Caching forge artifacts for future builds...")

	// Clean old cache and replace
	os.RemoveAll(filepath.Join(cacheDir, "forge-artifacts"))
	os.RemoveAll(filepath.Join(cacheDir, "cache"))
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logger.Info("Failed to create forge cache directory", err)
		return
	}

	if err := copyDir(srcArtifacts, filepath.Join(cacheDir, "forge-artifacts")); err != nil {
		logger.Info("Failed to save forge artifacts cache", err)
		os.RemoveAll(filepath.Join(cacheDir, "forge-artifacts")) // Clean up partial copy
		return
	}
	if _, err := os.Stat(srcCache); !os.IsNotExist(err) {
		if err := copyDir(srcCache, filepath.Join(cacheDir, "cache")); err != nil {
			logger.Info("Failed to save forge solc cache", err)
			os.RemoveAll(filepath.Join(cacheDir, "cache"))
		}
	}
}

// copyDir recursively copies a directory tree, preserving symlinks as symlinks.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// Check if the original entry is a symlink (filepath.Walk follows symlinks,
		// so we need to Lstat the original path to detect them)
		linfo, lstatErr := os.Lstat(path)
		if lstatErr != nil {
			return lstatErr
		}
		if linfo.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(path)
			if readErr != nil {
				return readErr
			}
			return os.Symlink(target, dstPath)
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}
