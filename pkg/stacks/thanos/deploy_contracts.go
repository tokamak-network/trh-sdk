package thanos

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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
		return fmt.Errorf("network %s does not require contract deployment, please run `trh-sdk deploy` instead", constants.LocalDevnet)
	}
	if t.network != constants.Testnet && t.network != constants.Mainnet {
		return fmt.Errorf("network %s does not support", t.network)
	}

	if deployContractsConfig == nil {
		return fmt.Errorf("deployContractsConfig is required")
	}

	if deployContractsConfig.ChainConfiguration == nil {
		return fmt.Errorf("chainConfiguration is required")
	}

	l1ChainID := deployContractsConfig.l1ChainId

	if err := deployContractsConfig.ChainConfiguration.Validate(l1ChainID); err != nil {
		return err
	}

	var (
		err      error
		isResume bool
	)

	if t.deployConfig == nil {
		t.deployConfig = &types.Config{
			Stack:   constants.ThanosStack,
			Network: t.network,
		}
	}

	if t.deployConfig.DeployContractState != nil {
		if t.deployConfig.DeployContractState.Status == types.DeployContractStatusCompleted {
			fmt.Println("The contracts have already been deployed successfully.")
			if !t.ignorePromptInput {
				fmt.Print("Do you want to deploy the contracts again? (y/N): ")
				isDeployAgain, err := scanner.ScanBool(false)
				if err != nil {
					fmt.Println("Error reading the deploy again input:", err)
					return err
				}

				if !isDeployAgain {
					return nil
				}
			}
		} else if t.deployConfig.DeployContractState.Status == types.DeployContractStatusInProgress {
			if !t.ignorePromptInput {
				fmt.Print("The contracts deployment is in progress. Do you want to resume? (Y/n): ")
				isResume, err = scanner.ScanBool(true)
				if err != nil {
					fmt.Println("Error reading the resume input:", err)
					return err
				}
			}
		}
	}

	if isResume {
		l1Rpc := t.deployConfig.L1RPCURL
		l1Client, err := ethclient.DialContext(ctx, l1Rpc)
		if err != nil {
			fmt.Printf("Failed to connect to L1 RPC: %s", err)
			return err
		}

		err = t.deployContracts(ctx, l1Client, true)
		if err != nil {
			fmt.Print("\r❌ Resume the contracts deployment failed!       \n")
			return err
		}
	} else {
		l1Client, err := ethclient.DialContext(ctx, deployContractsConfig.l1RPCurl)
		if err != nil {
			return err
		}

		l2ChainID, err := utils.GenerateL2ChainId()
		if err != nil {
			fmt.Printf("Failed to generate L2ChainID: %s", err)
			return err
		}

		deployContractsTemplate := initDeployConfigTemplate(deployContractsConfig, l1ChainID, l2ChainID)

		operators := deployContractsConfig.Operators

		if len(operators) == 0 || len(operators) < 4 {
			return fmt.Errorf("at least 5 operators are required for deploying contracts")
		}

		if !t.ignorePromptInput {
			fmt.Print("🔎 The SDK is ready to deploy the contracts to the L1 network. Do you want to proceed(Y/n)? ")
			confirmation, err := scanner.ScanBool(true)
			if err != nil {
				return err
			}
			if !confirmation {
				return nil
			}
		}

		shellConfigFile := utils.GetShellConfigDefault()

		// Check dependencies
		if !dependencies.CheckPnpmInstallation() {
			fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
			return nil
		}

		if !dependencies.CheckFoundryInstallation() {
			fmt.Printf("Try running `source %s` to set up your environment \n", shellConfigFile)
			return nil
		}

		// STEP 2. Clone the repository
		err = t.cloneSourcecode("tokamak-thanos", "https://github.com/tokamak-network/tokamak-thanos.git")
		if err != nil {
			return err
		}

		t.deployConfig.AdminPrivateKey = operators[0].PrivateKey
		t.deployConfig.SequencerPrivateKey = operators[1].PrivateKey
		t.deployConfig.BatcherPrivateKey = operators[2].PrivateKey
		t.deployConfig.ProposerPrivateKey = operators[3].PrivateKey
		if deployContractsConfig.fraudProof {
			if operators[4] == nil {
				return fmt.Errorf("challenger operator is required for fault proof but was not found")
			}
			t.deployConfig.ChallengerPrivateKey = operators[4].PrivateKey
		}
		t.deployConfig.DeploymentFilePath = fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/deployments/%d-deploy.json", t.deploymentPath, deployContractsTemplate.L1ChainID)
		t.deployConfig.L1RPCProvider = utils.DetectRPCKind(deployContractsConfig.l1RPCurl)
		t.deployConfig.L1ChainID = deployContractsTemplate.L1ChainID
		t.deployConfig.L2ChainID = l2ChainID
		t.deployConfig.L1RPCURL = deployContractsConfig.l1RPCurl
		t.deployConfig.EnableFraudProof = deployContractsConfig.fraudProof
		t.deployConfig.ChainConfiguration = deployContractsConfig.ChainConfiguration

		deployConfigFilePath := fmt.Sprintf("%s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts/deploy-config.json", t.deploymentPath)

		err = makeDeployContractConfigJsonFile(ctx, l1Client, operators, deployContractsTemplate, deployConfigFilePath)
		if err != nil {
			return err
		}

		// STEP 3. Build the contracts
		fmt.Println("Building smart contracts...")
		err = utils.ExecuteCommandStream(t.l, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh build", t.deploymentPath))
		if err != nil {
			fmt.Print("\r❌ Build the contracts failed!       \n")
			return err
		}
		fmt.Print("\r✅ Build the contracts completed!       \n")

		// STEP 4. Deploy the contracts
		// Check admin balance and estimated deployment cost
		adminAddress := operators[0].Address
		balance, err := l1Client.BalanceAt(ctx, common.HexToAddress(adminAddress), nil)
		if err != nil {
			fmt.Printf("❌ Failed to retrieve admin account balance: %v\n", err)
			return err
		}
		fmt.Printf("Admin account balance: %.2f ETH\n", utils.WeiToEther(balance))

		// Estimate gas price
		gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
		if err != nil {
			fmt.Printf("❌ Failed to get gas price: %v\n", err)
			return err
		}
		fmt.Printf("⛽ Current gas price: %.4f Gwei\n", new(big.Float).Quo(new(big.Float).SetInt(gasPriceWei), big.NewFloat(1e9)))

		// Estimate deployment cost
		estimatedCost := new(big.Int).Mul(gasPriceWei, estimatedDeployContracts)
		estimatedCost.Mul(estimatedCost, big.NewInt(2))
		fmt.Printf("💰 Estimated deployment cost: %.4f ETH\n", utils.WeiToEther(estimatedCost))

		// Check if balance is sufficient
		if balance.Cmp(estimatedCost) < 0 {
			fmt.Println("❌ Insufficient balance for deployment.")
			return fmt.Errorf("admin account balance (%.4f ETH) is less than estimated deployment cost (%.4f  ETH)", utils.WeiToEther(balance), utils.WeiToEther(estimatedCost))
		} else {
			fmt.Println("✅ The admin account has sufficient balance to proceed with deployment.")
		}

		t.deployConfig.DeployContractState = &types.DeployContractState{
			Status: types.DeployContractStatusInProgress,
		}
		err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
		if err != nil {
			fmt.Println("Failed to write settings file:", err)
			return err
		}

		err = t.deployContracts(ctx, l1Client, false)
		if err != nil {
			fmt.Print("\r❌ Deploy the contracts failed!       \n")
		}
	}

	// STEP 5: Generate the genesis and rollup files
	err = utils.ExecuteCommandStream(t.l, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh generate -e .env -c deploy-config.json", t.deploymentPath))
	fmt.Println("Generating the rollup and genesis files...")
	if err != nil {
		fmt.Print("\r❌ Failed to generate rollup and genesis files!       \n")
		return err
	}
	fmt.Print("\r✅ Successfully generated rollup and genesis files!       \n")
	fmt.Printf("\r Genesis file path: %s/tokamak-thanos/build/genesis.json\n", t.deploymentPath)
	fmt.Printf("\r Rollup file path: %s/tokamak-thanos/build/rollup.json\n", t.deploymentPath)

	fmt.Printf("✅ Configuration successfully saved to: %s/settings.json \n", t.deploymentPath)
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

	fmt.Println("Deploying the contracts...")

	gasPriceWei, err := l1Client.SuggestGasPrice(ctx)
	if err != nil {
		fmt.Printf("Failed to get gas price: %v\n", err)
	}

	envValues := fmt.Sprintf("export GS_ADMIN_PRIVATE_KEY=%s\nexport L1_RPC_URL=%s\n", adminPrivateKey, l1RPC)
	if gasPriceWei != nil && gasPriceWei.Uint64() > 0 {
		// double gas price
		envValues += fmt.Sprintf("export GAS_PRICE=%d\n", gasPriceWei.Uint64()*2)
	}

	// STEP 4.1. Generate the .env file
	_, err = utils.ExecuteCommand(
		"bash",
		"-c",
		fmt.Sprintf("cd %s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && echo '%s' > .env", t.deploymentPath, envValues),
	)
	if err != nil {
		fmt.Print("\r❌ Make .env file failed!       \n")
		return err
	}

	// STEP 4.3. Deploy contracts
	if isResume {
		err = utils.ExecuteCommandStream(t.l, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh redeploy -e .env -c deploy-config.json", t.deploymentPath))
		if err != nil {
			fmt.Print("\r❌ Contract deployment failed!       \n")
			return err
		}
	} else {
		err = utils.ExecuteCommandStream(t.l, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh deploy -e .env -c deploy-config.json", t.deploymentPath))
		if err != nil {
			fmt.Print("\r❌ Contract deployment failed!       \n")
			return err
		}
	}
	fmt.Print("\r✅ Contract deployment completed successfully!       \n")

	t.deployConfig.DeployContractState.Status = types.DeployContractStatusCompleted
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		fmt.Println("Failed to write settings file:", err)
		return err
	}
	return nil
}
