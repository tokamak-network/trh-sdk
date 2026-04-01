package thanos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
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

		// STEP 2.1–3: Parallelize cannon prestate build with source build pipeline.
		// Track A: Patch AnchorStateRegistry + build cannon prestate (fault proof only)
		// Track B: Patch scripts + download artifacts + build source code
		// Both tracks run after clone completes; prestateHash is used after both finish.
		tokamakThanosDir := filepath.Join(t.deploymentPath, "tokamak-thanos")
		contractsDir := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock")
		scriptsDir := filepath.Join(contractsDir, "scripts")
		forgeCacheDir := filepath.Join(t.deploymentPath, ".forge-cache")

		var prestateHash string

		// Patch AnchorStateRegistry BEFORE the parallel tracks start.
		// Track B may build contracts, so the source must already be patched.
		if deployContractsConfig.EnableFaultProof {
			if patchErr := patchAnchorStateRegistry(tokamakThanosDir); patchErr != nil {
				return fmt.Errorf("failed to patch AnchorStateRegistry.sol: %w", patchErr)
			}
			t.logger.Info("✅ AnchorStateRegistry.sol patched with setInitialAnchorState")
		}

		g, gctx := errgroup.WithContext(ctx)

		// Track A: Cannon prestate (only when fault proof is enabled, ~34.5s)
		if deployContractsConfig.EnableFaultProof {
			g.Go(func() error {
				// Build cannon prestate to extract FaultGameAbsolutePrestate hash
				prestatePath := filepath.Join(tokamakThanosDir, "op-program", "bin", "prestate.json")
				if _, statErr := os.Stat(prestatePath); os.IsNotExist(statErr) {
					t.logger.Info("Building cannon prestate before contract deployment...")
					if buildErr := buildCannonPrestate(gctx, t.logger, tokamakThanosDir); buildErr != nil {
						return fmt.Errorf("failed to build cannon prestate: %w", buildErr)
					}
					t.logger.Info("✅ Cannon prestate built successfully")
				} else {
					t.logger.Info("Cannon prestate already exists, reading hash", "path", prestatePath)
				}

				hash, hashErr := readPrestateHash(prestatePath)
				if hashErr != nil {
					return hashErr
				}
				prestateHash = hash // Safe: only read after g.Wait()
				t.logger.Info("Cannon prestate hash loaded", "hash", prestateHash)
				return nil
			})
		}

		// Track B: Source build pipeline (~27s with caching, ~51s without)
		g.Go(func() error {
			// Patch start-deploy.sh for known build issues
			if patchErr := patchStartDeployScript(tokamakThanosDir); patchErr != nil {
				t.logger.Warn("Failed to patch start-deploy.sh, build may fail", "err", patchErr)
			} else {
				t.logger.Info("✅ start-deploy.sh patched: cannon prestate and op-node build fixed")
			}

			// Patch workspace dependency
			if patchErr := patchContractsPkgJSON(tokamakThanosDir); patchErr != nil {
				t.logger.Warn("Failed to patch package.json workspace dep", "err", patchErr)
			}

			// Remove broken test files that prevent forge build
			removeBrokenTestFiles(t.logger, contractsDir)

			// Try downloading pre-built artifacts from npm to skip forge build
			if dlErr := downloadPrebuiltArtifacts(gctx, t.logger, contractsDir); dlErr != nil {
				t.logger.Warn("Pre-built artifacts unavailable, will build from source", "err", dlErr)
				restoreForgeCache(t.logger, forgeCacheDir, contractsDir)
			} else {
				t.logger.Info("✅ Pre-built artifacts downloaded")

				// Create non-versioned symlinks in Go (replaces slow shell find loop: ~7.5s → <1s)
				if symlinkErr := createArtifactSymlinks(contractsDir); symlinkErr != nil {
					t.logger.Warn("Failed to create artifact symlinks in Go, shell fallback will run", "err", symlinkErr)
				} else {
					t.logger.Info("✅ Artifact symlinks created")
				}

				if deployContractsConfig.EnableFaultProof {
					// Fault proof enabled: invalidate cache for patched AnchorStateRegistry
					// and let forge do an incremental build to recompile it. The pre-built
					// artifacts don't contain setInitialAnchorState which is required for
					// bootstrapping the genesis anchor state on new chains.
					if cacheErr := invalidateCacheEntry(contractsDir, "src/dispute/AnchorStateRegistry.sol"); cacheErr != nil {
						t.logger.Warn("Failed to invalidate cache for patched contract", "err", cacheErr)
					}
					// Force recompile by removing compiled artifact directory.
					// invalidateCacheEntry uses a relative path key but the cache JSON may store
					// absolute path keys — making the cache invalidation a silent no-op.
					// Deleting the artifact dir guarantees forge must recompile the patched source.
					artifactDir := filepath.Join(contractsDir, "forge-artifacts", "AnchorStateRegistry.sol")
					if err := os.RemoveAll(artifactDir); err != nil {
						t.logger.Warnf("Failed to remove AnchorStateRegistry forge artifact: %v", err)
					}
					t.logger.Info("Forge incremental build will run to compile patched AnchorStateRegistry")
				} else {
					// No fault proof: safe to skip forge build entirely
					os.Setenv("SKIP_FORGE_BUILD", "true")
					t.logger.Info("Forge build will be skipped (no fault proof patches needed)")
				}
			}

			// Restore cached op-node binary to skip go build (~5.5s on re-deploys)
			restoreOpNodeBinary(t.logger, forgeCacheDir, tokamakThanosDir)

			t.logger.Info("Building smart contracts...")
			if buildErr := utils.ExecuteCommandStreamInDir(gctx, t.logger, scriptsDir, "bash", "./start-deploy.sh", "build"); buildErr != nil {
				if errors.Is(buildErr, context.Canceled) {
					return buildErr
				}
				t.logger.Error("❌ Build the contracts failed!")
				return buildErr
			}
			t.logger.Info("✅ Build the contracts completed!")
			if deployContractsConfig.EnableFaultProof {
				if verifyErr := verifyAnchorStateRegistryArtifact(contractsDir); verifyErr != nil {
					return fmt.Errorf("AnchorStateRegistry artifact verification failed — forge did not recompile the patched source: %w", verifyErr)
				}
				t.logger.Info("✅ AnchorStateRegistry artifact verified: setInitialAnchorState present")
			}
			return nil
		})

		// Wait for both tracks to complete
		if err = g.Wait(); err != nil {
			os.Unsetenv("SKIP_FORGE_BUILD")
			if errors.Is(err, context.Canceled) {
				t.logger.Warn("Deployment canceled")
			}
			return err
		}
		os.Unsetenv("SKIP_FORGE_BUILD")

		// Cache forge artifacts and op-node binary for future builds
		saveForgeCache(t.logger, forgeCacheDir, contractsDir)
		saveOpNodeBinary(t.logger, forgeCacheDir, tokamakThanosDir)

		// Generate deploy config (needs prestateHash from Track A)
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

		// Inform the operator about automatic AA Paymaster setup that will run after L2 starts.
		if constants.NeedsAASetup(deployContractsConfig.Preset, deployContractsConfig.FeeToken) {
			t.logger.Infof("ℹ️  Fee token: %s (non-TON)", deployContractsConfig.FeeToken)
			t.logger.Infof("ℹ️  AA Paymaster will be configured automatically after L2 network starts:")
			t.logger.Infof("    • %s native token deposited to EntryPoint for gas sponsorship", constants.DefaultEntryPointDeposit.String())
			t.logger.Infof("    • %s registered with MultiTokenPaymaster (markup: %d%%)", deployContractsConfig.FeeToken, aaMarkupForToken(deployContractsConfig.FeeToken))
			t.logger.Infof("    • SimplePriceOracle price set with initial placeholder value (update post-deployment)")
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
		// Lower activation threshold for testnet/local deployments where only 1 DRB node runs
		if t.network == constants.Testnet || t.network == constants.LocalDevnet {
			drbConfig.ActivationThreshold = big.NewInt(1)
			t.logger.Info("Using DRB ActivationThreshold=1 for testnet/local deployment")
		}
		if err := injectDRBIntoGenesis(ctx, t.logger, genesisPath, drbConfig); err != nil {
			t.logger.Error("❌ Failed to inject DRB into genesis!", "err", err)
			return err
		}
	}

	// STEP 5.2: Inject USDC (FiatTokenV2_2) predeploy into genesis for ALL presets
	// For DeFi/Full: tokamak-thanos already includes USDC — injectUSDCIntoGenesis skips if present
	t.logger.Info("Injecting USDC (FiatTokenV2_2) predeploy into genesis...")
	if err := injectUSDCIntoGenesis(genesisPath, t.deploymentPath); err != nil {
		t.logger.Error("❌ Failed to inject USDC into genesis!", "err", err)
		return err
	}

	// STEP 5.2b: Inject updated MultiTokenPaymaster implementation into genesis code namespace
	// Fixes paymasterAndData offset: [20:40] Phase 1 → [52:72] ERC-4337 v0.8 standard
	t.logger.Info("Injecting MultiTokenPaymaster (v0.8 paymasterAndData offset fix) into genesis...")
	if err := injectMultiTokenPaymasterBytecode(genesisPath, t.deploymentPath); err != nil {
		t.logger.Error("❌ Failed to inject MultiTokenPaymaster bytecode!", "err", err)
		return err
	}

	// STEP 5.3: Update rollup.json genesis hash after ALL genesis modifications
	rollupPath := filepath.Join(t.deploymentPath, "tokamak-thanos", "build", "rollup.json")
	if err := updateRollupGenesisHash(t.logger, genesisPath, rollupPath); err != nil {
		t.logger.Errorf("❌ Failed to update rollup.json genesis hash: %v", err)
		return err
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
	// Skip build entirely if a cached binary was restored by restoreOpNodeBinary().
	// Run from $projectRoot (where go.mod lives) with GOWORK=off to avoid workspace mode issues
	// in container environments where go.work + no per-package go.mod causes "go.mod not found".
	opNodeOld := `  if ! retryCommand "make op-node" "Building op-node"; then`
	opNodeNew := `  if [ -f "$projectRoot/op-node/bin/op-node" ]; then
    echo "op-node binary found, skipping build"
  elif ! retryCommand "mkdir -p $projectRoot/op-node/bin && (cd $projectRoot && env GOWORK=off GO111MODULE=on CGO_ENABLED=0 go build -v -o ./op-node/bin/op-node ./op-node/cmd)" "Building op-node"; then`

	if bytes.Contains(content, []byte(opNodeOld)) {
		content = bytes.Replace(content, []byte(opNodeOld), []byte(opNodeNew), 1)
	}

	// Patch 3: skip core-utils and SDK TypeScript builds entirely.
	// These packages are not used by the contract deployment pipeline (Deploy.s.sol,
	// L2Genesis.s.sol, op-node genesis). Removing them saves ~8s (core-utils 4.1s +
	// SDK reinstall 1.8s + SDK build 2.3s).
	coreUtilsBlockOld := `  # Build TypeScript packages in dependency order
  echo "Building core-utils..."`
	coreUtilsBlockNew := `  # TypeScript builds (core-utils, SDK) skipped — not needed for contract deployment
  echo "Skipping core-utils and SDK builds (not required for deployment pipeline)"
  cd $currentPWD
  echo "✅ All source code built successfully!"
  return 0

  # --- Below is unreachable (kept for reference) ---
  echo "Building core-utils..."`
	if bytes.Contains(content, []byte(coreUtilsBlockOld)) {
		content = bytes.Replace(content, []byte(coreUtilsBlockOld), []byte(coreUtilsBlockNew), 1)
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

	// Patch 5: (removed — core-utils/SDK builds skipped by Patch 3)

	// Patch 6: shallow, non-recursive submodule update instead of full history clone.
	// Only first-level submodules are needed; nested ones (e.g. automate→forge-std→ds-test)
	// are test-only dependencies of third-party libraries.
	submoduleOld := `make submodules`
	submoduleNew := `git submodule update --init --depth 1`
	if bytes.Contains(content, []byte(submoduleOld)) {
		content = bytes.Replace(content, []byte(submoduleOld), []byte(submoduleNew), 1)
	}

	// Patch 7: (removed — core-utils/SDK builds skipped by Patch 3)

	// Patch 8: remove --rpc-url from L2Genesis forge script.
	// L2Genesis generates L2 state locally and only reads L1 addresses from a JSON file
	// (CONTRACT_ADDRESSES_PATH). Using --rpc-url causes forge to fork L1, making 256+
	// RPC calls for vm.deal() on precompile addresses, which triggers Alchemy rate limits.
	l2GenesisOld := `    forge script scripts/L2Genesis.s.sol:L2Genesis \
    --rpc-url $L1_RPC_URL; then`
	l2GenesisNew := `    forge script scripts/L2Genesis.s.sol:L2Genesis; then`
	if bytes.Contains(content, []byte(l2GenesisOld)) {
		content = bytes.Replace(content, []byte(l2GenesisOld), []byte(l2GenesisNew), 1)
	}

	// Patch 9: add --fork-retries and --fork-retry-backoff to forge Deploy script.
	// Free-tier Alchemy endpoints return HTTP 429 under sustained forge RPC load.
	// Exponential backoff (starting at 3s, up to 10 retries) lets transient rate
	// limits resolve without failing the entire deployment.
	deployOld := `forge script scripts/Deploy.s.sol:Deploy --private-key $GS_ADMIN_PRIVATE_KEY --broadcast --rpc-url $L1_RPC_URL --slow --legacy --non-interactive`
	deployNew := `forge script scripts/Deploy.s.sol:Deploy --private-key $GS_ADMIN_PRIVATE_KEY --broadcast --rpc-url $L1_RPC_URL --slow --legacy --non-interactive --fork-retries 10 --fork-retry-backoff 3000`
	// Replace all occurrences (deploy, resume with --resume, and gas-price variants)
	content = bytes.ReplaceAll(content, []byte(deployOld), []byte(deployNew))

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

// versionedArtifactRe matches forge artifact filenames with Solidity version suffixes
// e.g. "L1UsdcBridge.0.8.15.json" → captures ".0.8.15"
var versionedArtifactRe = regexp.MustCompile(`(\.\d+)+\.json$`)

// createArtifactSymlinks creates non-versioned symlinks for versioned forge artifacts.
// Newer forge versions name artifacts with solidity version suffix (e.g. L1UsdcBridge.0.8.15.json)
// but the TypeScript SDK imports the unversioned name (L1UsdcBridge.json).
// This Go implementation replaces the slow shell find|while|sed loop (~7.5s → <1s).
func createArtifactSymlinks(contractsDir string) error {
	forgeDir := filepath.Join(contractsDir, "forge-artifacts")
	return filepath.WalkDir(forgeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		base := d.Name()
		nonversioned := versionedArtifactRe.ReplaceAllString(base, ".json")
		if nonversioned != base {
			target := filepath.Join(filepath.Dir(path), nonversioned)
			if _, statErr := os.Lstat(target); os.IsNotExist(statErr) {
				if linkErr := os.Symlink(base, target); linkErr != nil {
					return linkErr
				}
			}
		}
		return nil
	})
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

// restoreOpNodeBinary copies a cached op-node binary into the cloned repo to skip go build (~5.5s).
func restoreOpNodeBinary(logger interface{ Info(args ...interface{}) }, cacheDir, tokamakThanosDir string) {
	src := filepath.Join(cacheDir, "op-node")
	dst := filepath.Join(tokamakThanosDir, "op-node", "bin", "op-node")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	if err := os.WriteFile(dst, data, 0755); err != nil {
		logger.Info("Failed to restore cached op-node binary", err)
		return
	}
	logger.Info("✅ Restored cached op-node binary")
}

// saveOpNodeBinary saves the built op-node binary to the deployment-level cache.
func saveOpNodeBinary(logger interface{ Info(args ...interface{}) }, cacheDir, tokamakThanosDir string) {
	src := filepath.Join(tokamakThanosDir, "op-node", "bin", "op-node")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "op-node"), data, 0755); err != nil {
		logger.Info("Failed to cache op-node binary", err)
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

// updateRollupGenesisHash recomputes the L2 genesis block hash from genesis.json
// and updates rollup.json to match. This must be called after any post-generation
// modification to genesis.json (e.g., DRB injection) to prevent hash mismatches
// between op-node's rollup config and op-geth's actual genesis block.
func updateRollupGenesisHash(logger *zap.SugaredLogger, genesisPath, rollupPath string) error {
	// Parse genesis.json using go-ethereum's Genesis type
	genesisData, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to read genesis.json: %w", err)
	}

	var genesis core.Genesis
	if err := json.Unmarshal(genesisData, &genesis); err != nil {
		return fmt.Errorf("failed to parse genesis.json: %w", err)
	}

	// Compute the genesis block hash
	genesisBlock := genesis.ToBlock()
	newHash := genesisBlock.Hash().Hex()

	// Read and update rollup.json
	rollupData, err := os.ReadFile(rollupPath)
	if err != nil {
		return fmt.Errorf("failed to read rollup.json: %w", err)
	}

	var rollup map[string]interface{}
	if err := json.Unmarshal(rollupData, &rollup); err != nil {
		return fmt.Errorf("failed to parse rollup.json: %w", err)
	}

	genesisSection, ok := rollup["genesis"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("rollup.json missing 'genesis' section")
	}
	l2Section, ok := genesisSection["l2"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("rollup.json missing 'genesis.l2' section")
	}

	oldHash, _ := l2Section["hash"].(string)
	if oldHash == newHash {
		logger.Info("rollup.json L2 genesis hash already matches, no update needed")
		return nil
	}

	l2Section["hash"] = newHash
	logger.Infof("Updated rollup.json L2 genesis hash: %s → %s", oldHash, newHash)

	updatedData, err := json.MarshalIndent(rollup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rollup.json: %w", err)
	}

	return os.WriteFile(rollupPath, updatedData, 0644)
}

// verifyAnchorStateRegistryArtifact checks that the compiled AnchorStateRegistry artifact
// contains setInitialAnchorState in its ABI. Called after forge build when fault proof is
// enabled to detect early if patchAnchorStateRegistry did not take effect.
func verifyAnchorStateRegistryArtifact(contractsDir string) error {
	artifactDir := filepath.Join(contractsDir, "forge-artifacts", "AnchorStateRegistry.sol")
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return fmt.Errorf("forge-artifacts/AnchorStateRegistry.sol not found: %w", err)
	}
	var artifactPath string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".dbg.json") {
			artifactPath = filepath.Join(artifactDir, name)
			break
		}
	}
	if artifactPath == "" {
		return fmt.Errorf("no JSON artifact found in forge-artifacts/AnchorStateRegistry.sol/")
	}
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}
	var artifact struct {
		ABI []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"abi"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("failed to parse artifact JSON: %w", err)
	}
	for _, entry := range artifact.ABI {
		if entry.Type == "function" && entry.Name == "setInitialAnchorState" {
			return nil
		}
	}
	return fmt.Errorf("setInitialAnchorState not found in AnchorStateRegistry ABI — " +
		"forge did not recompile the patched source; ensure patchAnchorStateRegistry ran " +
		"or delete forge-artifacts/AnchorStateRegistry.sol/ and retry the build")
}
