package thanos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethclient"
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

		// STEP 3. Build the contracts
		scriptsDir := filepath.Join(t.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "scripts")
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
	t.logger.Infof("Genesis file path: %s/tokamak-thanos/build/genesis.json", t.deploymentPath)
	t.logger.Infof("Rollup file path: %s/tokamak-thanos/build/rollup.json", t.deploymentPath)

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

	// Patch 4: limit forge parallelism and reduce non-essential build output to prevent OOM.
	// Forge compiles 6 Solc versions in parallel by default, exhausting Docker Desktop memory.
	// --jobs 1 forces sequential compilation. We strip extra_output and build_info to reduce
	// memory, but keep optimizer_runs=999999 as the deploy script's chain assertions validate
	// bytecode compiled with that exact setting.
	forgeCleanOld := `forge clean && forge build`
	forgeCleanNew := `sed -i '/^extra_output/d' foundry.toml 2>/dev/null || true && sed -i 's/build_info = true/build_info = false/' foundry.toml 2>/dev/null || true && forge clean && forge build --jobs 1`
	if bytes.Contains(content, []byte(forgeCleanOld)) {
		content = bytes.Replace(content, []byte(forgeCleanOld), []byte(forgeCleanNew), 1)
	}

	return os.WriteFile(scriptPath, content, 0755)
}
