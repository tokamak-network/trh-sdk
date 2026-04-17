package thanos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	cacheDir := filepath.Join(homeDir, ".trh", "bin")
	binaryPath, err := ensureTokamakDeployer(cacheDir)
	if err != nil {
		return fmt.Errorf("tokamak-deployer binary unavailable: %w", err)
	}
	deployOutputPath := filepath.Join(t.deploymentPath, "deploy-output.json")
	deployConfigFilePath := filepath.Join(t.deploymentPath, "deploy-config.json")

	if isResume {
		if err = runDeployContracts(ctx, binaryPath, deployContractsOpts{
			L1RPCURL:   t.deployConfig.L1RPCURL,
			PrivateKey: t.deployConfig.AdminPrivateKey,
			L2ChainID:  t.deployConfig.L2ChainID,
			OutPath:    deployOutputPath,
		}, t.output); err != nil {
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

		var prestateHash string

		// Fault proof requires tokamak-thanos clone for cannon prestate build
		if deployContractsConfig.EnableFaultProof {
			err = t.cloneSourcecode(ctx, "tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
			if err != nil {
				t.logger.Error("failed to clone the repository", "err", err)
				return err
			}
			tokamakThanosDir := filepath.Join(t.deploymentPath, "tokamak-thanos")
			if patchErr := patchAnchorStateRegistry(tokamakThanosDir); patchErr != nil {
				return fmt.Errorf("failed to patch AnchorStateRegistry.sol: %w", patchErr)
			}
			t.logger.Info("✅ AnchorStateRegistry.sol patched with setInitialAnchorState")

			g, gctx := errgroup.WithContext(ctx)
			g.Go(func() error {
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
				prestateHash = hash
				t.logger.Info("Cannon prestate hash loaded", "hash", prestateHash)
				return nil
			})
			if err = g.Wait(); err != nil {
				return err
			}
		}

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
		t.deployConfig.DeploymentFilePath = deployOutputPath
		t.deployConfig.L1RPCProvider = utils.DetectRPCKind(deployContractsConfig.L1RPCurl)
		t.deployConfig.L1ChainID = deployContractsTemplate.L1ChainID
		t.deployConfig.L2ChainID = l2ChainID
		t.deployConfig.L1RPCURL = deployContractsConfig.L1RPCurl
		t.deployConfig.EnableFraudProof = deployContractsConfig.EnableFaultProof
		t.deployConfig.ChainConfiguration = deployContractsConfig.ChainConfiguration
		t.deployConfig.Preset = deployContractsConfig.Preset
		t.deployConfig.FeeToken = deployContractsConfig.FeeToken
		t.deployConfig.Mnemonic = deployContractsConfig.Mnemonic

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

		// Estimate gas price. The same suggested price is reused as a fixed
		// gas price passed to tokamak-deployer (× deployGasPriceMultiplier)
		// so the deployer avoids 26-32 per-TX SuggestGasPrice round-trips.
		gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
		if err != nil {
			t.logger.Error("❌ Failed to get gas price", "err", err)
			return err
		}
		t.logger.Infof("⛽ Current gas price: %.4f Gwei", new(big.Float).Quo(new(big.Float).SetInt(gasPriceWei), big.NewFloat(1e9)))
		fixedGasPrice := new(big.Int).Mul(gasPriceWei, deployGasPriceMultiplier)
		t.logger.Infof("⛽ Fixed gas price for deploy: %.4f Gwei (suggested × %d)",
			new(big.Float).Quo(new(big.Float).SetInt(fixedGasPrice), big.NewFloat(1e9)),
			deployGasPriceMultiplier)

		// Estimate deployment cost.
		// 3× margin remains on the raw suggested price to keep the balance
		// precheck wider than the 2× actually charged via fixedGasPrice.
		estimatedCost := new(big.Int).Mul(gasPriceWei, estimatedDeployContracts)
		estimatedCost.Mul(estimatedCost, big.NewInt(3))
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

		// When fault proof is enabled, remove the cached AnchorStateRegistry implementation
		// address from the Sepolia address file before running Deploy.s.sol.
		if deployContractsConfig.EnableFaultProof {
			tokamakThanosDir := filepath.Join(t.deploymentPath, "tokamak-thanos")
			contractsDir := filepath.Join(tokamakThanosDir, "packages", "tokamak", "contracts-bedrock")
			if cleared := clearAnchorStateRegistryFromAddressFile(contractsDir); cleared {
				t.logger.Info("✅ Cleared cached AnchorStateRegistry implementation address — will deploy patched version")
			}
		}

		// Deploy contracts via tokamak-deployer binary
		if err = runDeployContracts(ctx, binaryPath, deployContractsOpts{
			L1RPCURL:    deployContractsConfig.L1RPCurl,
			PrivateKey:  operators.AdminPrivateKey,
			L2ChainID:   uint64(l2ChainID),
			OutPath:     deployOutputPath,
			GasPriceWei: fixedGasPrice,
		}, t.output); err != nil {
			t.logger.Error("failed to deploy contracts", "err", err)
			return err
		}
	}

	// STEP 5: Generate genesis and rollup files.
	//
	// op-node's "genesis l2" subcommand requires two inputs that tokamak-deployer
	// does not produce itself:
	//   - --l2-allocs: the L2 state dump written by forge scripts/L2Genesis.s.sol
	//   - --l1-rpc:    an L1 JSON-RPC endpoint for block lookups
	//
	// We stage the deploy-output (addresses-only) and deploy-config under the
	// contracts-bedrock project root (forge's FFI sandbox rejects /tmp reads),
	// run the forge script, ensure op-node is built, then hand all paths to
	// tokamak-deployer which calls op-node and applies the tokamak-specific
	// post-processing (DRB / USDC / MultiTokenPaymaster / L1Block / rollup hash).
	tokamakThanosDirForGenesis := filepath.Join(t.deploymentPath, "tokamak-thanos")
	stagedAddrPath, stagedConfigPath, err := prepareL2GenesisInputs(
		tokamakThanosDirForGenesis,
		deployOutputPath,
		deployConfigFilePath,
		t.deployConfig.L2ChainID,
	)
	if err != nil {
		t.logger.Error("❌ Failed to stage L2 genesis inputs", "err", err)
		return err
	}

	stateDumpPath, err := runForgeL2GenesisScript(
		ctx,
		t.logger,
		tokamakThanosDirForGenesis,
		stagedAddrPath,
		stagedConfigPath,
		t.deployConfig.L1RPCURL,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			t.logger.Warn("Deployment canceled")
			return err
		}
		t.logger.Error("❌ Failed to run forge L2Genesis.s.sol", "err", err)
		return err
	}

	opNodeBin, err := ensureOpNodeBinary(ctx, t.logger, tokamakThanosDirForGenesis)
	if err != nil {
		t.logger.Error("❌ Failed to obtain op-node binary", "err", err)
		return err
	}

	genesisPath := filepath.Join(t.deploymentPath, "genesis.json")
	t.logger.Info("Generating the rollup and genesis files...")
	if err = runGenerateGenesis(ctx, binaryPath, genesisOpts{
		DeployOutputPath: stagedAddrPath, // addresses-only; op-node & deployer both accept this
		ConfigPath:       deployConfigFilePath,
		OutPath:          genesisPath,
		L1RPCURL:         t.deployConfig.L1RPCURL,
		L2AllocsPath:     stateDumpPath,
		OpNodeBinary:     opNodeBin,
	}, t.output); err != nil {
		if errors.Is(err, context.Canceled) {
			t.logger.Warn("Deployment canceled")
			return err
		}
		t.logger.Error("❌ Failed to generate rollup and genesis files!")
		return err
	}
	t.logger.Info("✅ Successfully generated rollup and genesis files!")
	t.logger.Infof("Genesis file path: %s", genesisPath)
	t.logger.Infof("Rollup file path: %s/rollup.json", t.deploymentPath)

	// Inject DRB predeploy if enabled for Gaming/Full preset
	if err := maybeInjectDRB(ctx, t.logger, genesisPath, t.deployConfig.Preset, &defaultArtifactFetcher{}); err != nil {
		t.logger.Error("❌ Failed to inject DRB predeploy", "err", err)
		return err
	}
	if err := maybeFundDRBRegulars(genesisPath, t.deployConfig.Preset, t.deployConfig.Mnemonic); err != nil {
		t.logger.Error("❌ Failed to fund DRB regular operators in genesis", "err", err)
		return err
	}

	// Mark deployment as complete so subsequent steps (e.g. local_network start,
	// deploy-aws-infra) pass the `Status == Completed` gate in their preflight
	// checks. This persist was accidentally dropped in df52538 when the forge
	// pipeline was replaced by the tokamak-deployer binary.
	t.deployConfig.DeployContractState.Status = types.DeployContractStatusCompleted
	if err = t.deployConfig.WriteToJSONFile(t.deploymentPath); err != nil {
		t.logger.Error("Failed to persist Completed deploy-contract state", "err", err)
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

// clearAnchorStateRegistryFromAddressFile removes the "AnchorStateRegistry" implementation
// address from thanos-stack-sepolia/address.json inside contractsDir.
// This forces Deploy.s.sol to deploy the freshly compiled (patched) implementation when
// fault proof is enabled, instead of reusing the pre-deployed Sepolia implementation which
// lacks setInitialAnchorState.
// Returns true if the key was found and removed, false if the file is absent or the key
// was not present (both are no-ops — not error conditions).
func clearAnchorStateRegistryFromAddressFile(contractsDir string) bool {
	sepoliaAddressFile := filepath.Join(contractsDir, "deployments", "thanos-stack-sepolia", "address.json")
	rawJSON, readErr := os.ReadFile(sepoliaAddressFile)
	if readErr != nil {
		return false
	}
	var addresses map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(rawJSON, &addresses); unmarshalErr != nil {
		return false
	}
	existing, exists := addresses["AnchorStateRegistry"]
	if !exists {
		return false
	}
	// Already cleared (zero address) — idempotent no-op.
	zeroAddr := `"0x0000000000000000000000000000000000000000"`
	if string(existing) == zeroAddr {
		return false
	}
	// Set to zero address instead of deleting the key. Deploy.s.sol calls
	// readAddress(".AnchorStateRegistry") unconditionally when reuseDeployment is
	// true, so the key must exist in the JSON. A zero address triggers the
	// "deploy new" branch in Solidity (savedAddress != address(0) check).
	addresses["AnchorStateRegistry"] = json.RawMessage(zeroAddr)
	updated, marshalErr := json.MarshalIndent(addresses, "", "  ")
	if marshalErr != nil {
		return false
	}
	return os.WriteFile(sepoliaAddressFile, updated, 0644) == nil
}
