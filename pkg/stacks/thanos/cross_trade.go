package thanos

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	crosstrade "github.com/tokamak-network/trh-sdk/pkg/stacks/thanos/cross-trade"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) DeployCrossTradeContracts(ctx context.Context, input *types.CrossTrade, scratch bool) (*types.DeployCrossTradeOutput, error) {
	if input.L1ChainConfig == nil {
		return nil, fmt.Errorf("l1 chain config is required")
	}

	err := os.Chdir(t.deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to change directory to deployment path: %w", err)
	}

	// Clone cross trade repository
	err = t.cloneSourcecode(ctx, "crossTrade", "https://github.com/tokamak-network/crossTrade.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone cross trade repository: %s", err)
	}

	if scratch {
		t.deployConfig.CrossTrade[input.Mode] = nil
		err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to write cross trade config to file: %s", err)
		}
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd crossTrade && git checkout main")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout main: %s", err)
	}

	t.logger.Info("Start to build cross-trade contracts")

	// Before deploying contracts, fill the missing fields in the input
	l1ContractFileName, l2ContractFileName, err := crosstrade.GetL1L2ContractFileName(input.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 and L2 contract file names: %s", err)
	}

	deploymentScriptPath, err := crosstrade.GetDeploymentScriptPath(input.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment script path: %s", err)
	}

	if input.L1ChainConfig.ContractName == "" {
		input.L1ChainConfig.ContractName = l1ContractFileName
	}
	if input.L1ChainConfig.DeploymentScriptPath == "" {
		input.L1ChainConfig.DeploymentScriptPath = deploymentScriptPath
	}

	if !strings.HasPrefix(input.L1ChainConfig.PrivateKey, "0x") {
		input.L1ChainConfig.PrivateKey = "0x" + input.L1ChainConfig.PrivateKey
	}

	for _, l2ChainConfig := range input.L2ChainConfig {
		if l2ChainConfig.ContractName == "" {
			l2ChainConfig.ContractName = l2ContractFileName
		}
		if l2ChainConfig.DeploymentScriptPath == "" {
			l2ChainConfig.DeploymentScriptPath = deploymentScriptPath
		}

		if !strings.HasPrefix(l2ChainConfig.PrivateKey, "0x") {
			l2ChainConfig.PrivateKey = "0x" + l2ChainConfig.PrivateKey
		}
	}

	// Build the contracts
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd crossTrade && pnpm install && forge clean && forge build")
	if err != nil {
		return nil, fmt.Errorf("failed to build the contracts: %s", err)
	}

	// Step 1: Check if the L1 contracts are deployed
	var (
		l1CrossTradeProxyAddress     string
		l1CrossTradeAddress          string
		l1ContractAddresses          = make(map[string]string)
		l2l2CrossTradeProxyAddresses = make(map[uint64]string)
		l1l2CrossTradeProxyAddresses = make(map[uint64]string)
		l2l2CrossTradeAddresses      = make(map[uint64]string)
		l1l2CrossTradeAddresses      = make(map[uint64]string)
	)
	if input.L1ChainConfig.IsDeployedNew {
		t.logger.Info("L1 contracts are not deployed. Deploying new L1 contracts")

		script := fmt.Sprintf(
			"cd crossTrade && PRIVATE_KEY=%s forge script %s/%s --rpc-url %s --broadcast --chain %s",
			input.L1ChainConfig.PrivateKey,
			input.L1ChainConfig.DeploymentScriptPath,
			input.L1ChainConfig.ContractName,
			input.L1ChainConfig.RPC,
			"sepolia",
		)
		t.logger.Infof("Deploying L1 contracts %s", script)
		err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy the contracts: %s", err)
		}
		// Get the contract address from the output
		filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", t.deploymentPath, input.L1ChainConfig.ContractName, input.L1ChainConfig.ChainID)
		l1ContractAddresses, err := utils.GetContractAddressFromOutput(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %w", err)
		}

		for contractName, address := range l1ContractAddresses {
			t.logger.Infof("L1 contract address %s with address %s", contractName, address)
			switch contractName {
			case crosstrade.L2L2CrossTradeProxyL1ContractName, crosstrade.L1L2CrossTradeProxyL1ContractName:
				l1CrossTradeProxyAddress = address
			case crosstrade.L2L2CrossTradeL1ContractName, crosstrade.L1L2CrossTradeL1ContractName:
				l1CrossTradeAddress = address
			default:
				t.logger.Infof("Unknown contract %s", contractName)
			}
		}
	} else {
		if input.L1ChainConfig.CrossTradeProxyAddress == "" || input.L1ChainConfig.CrossTradeAddress == "" {
			// Get L1 cross trade proxy and address from the output
			output := t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput
			if output == nil {
				return nil, fmt.Errorf("output is not set. Please run the deploy command first")
			}
			if output.L1CrossTradeProxyAddress == "" {
				return nil, fmt.Errorf("l1 cross trade proxy address is required")
			}

			l1CrossTradeProxyAddress = output.L1CrossTradeProxyAddress
			l1CrossTradeAddress = output.L1CrossTradeAddress
		} else {
			l1CrossTradeProxyAddress = input.L1ChainConfig.CrossTradeProxyAddress
			l1CrossTradeAddress = input.L1ChainConfig.CrossTradeAddress
		}
	}
	// Verify the contracts
	//
	if input.L1ChainConfig.BlockExplorerConfig != nil && input.L1ChainConfig.BlockExplorerConfig.APIKey != "" {
		for contractName, address := range l1ContractAddresses {
			if address == "" {
				continue
			}
			t.logger.Infof("Verifying L1 contract %s with address %s", contractName, address)
			functionName := contractName
			script := fmt.Sprintf(
				"cd crossTrade && forge verify-contract %s contracts/L1/%s.sol:%s --etherscan-api-key %s --chain %s",
				address,
				contractName,
				functionName,
				input.L1ChainConfig.BlockExplorerConfig.APIKey,
				constants.ChainIDToForgeChainName[input.L1ChainConfig.ChainID],
			)
			t.logger.Infof("Verifying L1 contract %s", script)
			err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
			if err != nil {
				// Skip if the contract is not verified
				t.logger.Errorf("failed to verify the contracts: %s", err)
				continue
			}
			t.logger.Infof("Verified L1 contract %s with address %s", contractName, address)
		}
	}

	for _, l2ChainConfig := range input.L2ChainConfig {
		if l2ChainConfig.IsDeployedNew {
			if l2ChainConfig.ContractName == "" {
			}
			if l2ChainConfig.DeploymentScriptPath == "" {
			}
			script := fmt.Sprintf(
				`cd crossTrade && PRIVATE_KEY=%s CHAIN_ID=%d NATIVE_TOKEN=%s L2_CROSS_DOMAIN_MESSENGER=%s L1_CROSS_TRADE=%s forge script %s/%s --rpc-url %s --broadcast`,
				l2ChainConfig.PrivateKey,
				l2ChainConfig.ChainID,
				constants.NativeToken,
				l2ChainConfig.CrossDomainMessenger,
				l1CrossTradeProxyAddress,
				l2ChainConfig.DeploymentScriptPath,
				l2ChainConfig.ContractName,
				l2ChainConfig.RPC,
			)
			t.logger.Infof("Deploying L2 contracts %s", script)
			err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
			if err != nil {
				return nil, fmt.Errorf("failed to deploy the contracts: %s", err)
			}

			// Get the contract address from the output
			t.logger.Infof("Getting contract address from output for %s on chain %d", l2ChainConfig.ContractName, l2ChainConfig.ChainID)
			filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", t.deploymentPath, l2ChainConfig.ContractName, l2ChainConfig.ChainID)
			addresses, err := utils.GetContractAddressFromOutput(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get the contract address: %s", err)
			}

			t.logger.Infof("Contract addresses: %v", addresses)

			for contractName, address := range addresses {
				t.logger.Infof("L2 contract address %s with address %s", contractName, address)
				switch contractName {
				case crosstrade.L2L2CrossTradeProxyL2ContractName:
					l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = address
				case crosstrade.L1L2CrossTradeProxyL2ContractName:
					l1l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = address
				case crosstrade.L2L2CrossTradeL2ContractName:
					l2l2CrossTradeAddresses[l2ChainConfig.ChainID] = address
				case crosstrade.L1L2CrossTradeL2ContractName:
					l1l2CrossTradeAddresses[l2ChainConfig.ChainID] = address
				default:
					t.logger.Infof("Unknown contract %s", contractName)
				}
			}

			// Verify the contracts
			//
			if l2ChainConfig.BlockExplorerConfig != nil && l2ChainConfig.BlockExplorerConfig.APIKey != "" {
				for contractName, address := range addresses {
					t.logger.Infof("Verifying L2 contract %s with address %s", contractName, address)
					if l2ChainConfig.BlockExplorerConfig.Type == constants.BlockExplorerTypeEtherscan {
						script = fmt.Sprintf(
							"cd crossTrade && forge verify-contract %s contracts/L2/%s.sol:%s --etherscan-api-key %s --chain %s",
							address,
							contractName,
							contractName,
							l2ChainConfig.BlockExplorerConfig.APIKey,
							constants.ChainIDToForgeChainName[l2ChainConfig.ChainID],
						)
						t.logger.Infof("Verifying L2 contract %s", script)
						err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
						if err != nil {
							t.logger.Errorf("failed to verify the contracts: %s", err)
							continue
						}
						t.logger.Infof("Verified L2 contract %s with address %s", contractName, address)
					} else if l2ChainConfig.BlockExplorerConfig.Type == constants.BlockExplorerTypeBlockscout {
						script = fmt.Sprintf(
							"cd crossTrade && forge verify-contract --rpc-url %s %s contracts/L2/%s.sol:%s --verifier blockscout --verifier-url %s/api",
							l2ChainConfig.RPC,
							address,
							contractName,
							contractName,
							l2ChainConfig.BlockExplorerConfig.URL,
						)
						t.logger.Infof("Verifying L2 contract %s", script)
						err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
						if err != nil {
							t.logger.Errorf("failed to verify the contracts: %s", err)
							continue
						}
						t.logger.Infof("Verified L2 contract %s with address %s", contractName, address)
					}
				}
			}
		} else {
			if input.Mode == constants.CrossTradeDeployModeL2ToL1 {
				// Get L1 cross trade proxy and address from the output
				output := t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput
				if output == nil {
					return nil, fmt.Errorf("output is not set. Please run the deploy command first")
				}
				if output.L2CrossTradeProxyAddresses[l2ChainConfig.ChainID] == "" {
					return nil, fmt.Errorf("l2 cross trade proxy address is required")
				}
				if output.L2CrossTradeAddresses[l2ChainConfig.ChainID] == "" {
					return nil, fmt.Errorf("l2 cross trade address is required")
				}

				l1l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = output.L2CrossTradeProxyAddresses[l2ChainConfig.ChainID]
				l1l2CrossTradeAddresses[l2ChainConfig.ChainID] = output.L2CrossTradeAddresses[l2ChainConfig.ChainID]
			} else {
				l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = l2ChainConfig.CrossTradeProxyAddress
				l2l2CrossTradeAddresses[l2ChainConfig.ChainID] = l2ChainConfig.CrossTradeAddress
			}
		}
	}

	t.logger.Infof("L1 cross trade proxy address %s", l1CrossTradeProxyAddress)
	t.logger.Infof("L1 cross trade address %s", l1CrossTradeAddress)
	t.logger.Infof("L2 <> L2 cross trade proxy addresses %v", l2l2CrossTradeProxyAddresses)
	t.logger.Infof("L1 <> L2 cross trade addresses %v", l1l2CrossTradeAddresses)

	switch input.Mode {
	case constants.CrossTradeDeployModeL2ToL1:
		l2ChainConfig := input.L2ChainConfig[0]
		l2ChainID := l2ChainConfig.ChainID
		// Set chain information for L1
		script := fmt.Sprintf(
			`cd crossTrade && PRIVATE_KEY=%s L1_CROSS_TRADE_PROXY=%s L1_CROSS_DOMAIN_MESSENGER=%s L2_CROSS_TRADE_PROXY=%s L2_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
			input.L1ChainConfig.PrivateKey,
			l1CrossTradeProxyAddress,
			l2ChainConfig.L1CrossDomainMessenger,
			l1l2CrossTradeProxyAddresses[l2ChainID],
			l2ChainID,
			"scripts/foundry_scripts/L2L1/SetChainInfoL1_L2L1.sol:SetChainInfoL1_L2L1",
			input.L1ChainConfig.RPC,
		)
		t.logger.Infof("Setting chain information for L1 %s", script)
		err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
		if err != nil {
			return nil, fmt.Errorf("failed to set chain information for L1: %s", err)
		}

		// Set chain information for L2
		script = fmt.Sprintf(
			`cd crossTrade && PRIVATE_KEY=%s L2_CROSS_TRADE_PROXY=%s L1_CROSS_TRADE_PROXY=%s L1_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
			l2ChainConfig.PrivateKey,
			l1l2CrossTradeProxyAddresses[l2ChainID],
			l1CrossTradeProxyAddress,
			input.L1ChainConfig.ChainID,
			"scripts/foundry_scripts/L2L1/SetChainInfoL2_L2L1.sol:SetChainInfoL2_L2L1",
			l2ChainConfig.RPC,
		)
		t.logger.Infof("Setting chain information for L2 %s", script)
		err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
		if err != nil {
			return nil, fmt.Errorf("failed to set chain information for L2: %s", err)
		}

	case constants.CrossTradeDeployModeL2ToL2:
		t.logger.Infof("Setting chain information for L2")
		for _, l2ChainConfig := range input.L2ChainConfig {

			// Set chain information for L2
			script := fmt.Sprintf(
				`cd crossTrade && PRIVATE_KEY=%s L2_CROSS_TRADE_PROXY=%s L1_CROSS_TRADE_PROXY=%s L1_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
				l2ChainConfig.PrivateKey,
				l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID],
				l1CrossTradeProxyAddress,
				input.L1ChainConfig.ChainID,
				"scripts/foundry_scripts/SetChainInfoL2_L2L2.sol:SetChainInfoL2_L2L2",
				l2ChainConfig.RPC,
			)
			t.logger.Infof("Setting chain information for L2 %s", script)
			err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
			if err != nil {
				return nil, fmt.Errorf("failed to set chain information for L2: %s", err)
			}

			// If the l2 chain is the current running chain, set USE_CUSTOM_BRIDGE by true
			useCustomBridge := false
			if l2ChainConfig.ChainID == t.deployConfig.L2ChainID {
				useCustomBridge = true
			}
			script = fmt.Sprintf(
				`cd crossTrade && PRIVATE_KEY=%s L1_CROSS_TRADE_PROXY=%s L1_CROSS_DOMAIN_MESSENGER=%s L2_CROSS_TRADE_PROXY=%s L2_NATIVE_TOKEN_ADDRESS_ON_L1=%s L1_STANDARD_BRIDGE=%s L1_USDC_BRIDGE=%s L2_CHAIN_ID=%d USE_CUSTOM_BRIDGE=%t forge script %s --rpc-url %s --broadcast`,
				input.L1ChainConfig.PrivateKey,
				l1CrossTradeProxyAddress,
				l2ChainConfig.L1CrossDomainMessenger,
				l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID],
				l2ChainConfig.NativeTokenAddressOnL1,
				l2ChainConfig.L1StandardBridgeAddress,
				l2ChainConfig.L1USDCBridgeAddress,
				l2ChainConfig.ChainID,
				useCustomBridge,
				"scripts/foundry_scripts/SetChainInfoL1_L2L2.sol:SetChainInfoL1_L2L2",
				input.L1ChainConfig.RPC,
			)
			t.logger.Infof("Setting chain information for L1 %s", script)
			err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
			if err != nil {
				return nil, fmt.Errorf("failed to set chain information for L1: %s", err)
			}
		}
	}

	var deployCrossTradeContractsOutput *types.DeployCrossTradeContractsOutput
	if input.Mode == constants.CrossTradeDeployModeL2ToL1 {
		deployCrossTradeContractsOutput = &types.DeployCrossTradeContractsOutput{
			Mode:                       input.Mode,
			L1CrossTradeProxyAddress:   l1CrossTradeProxyAddress,
			L1CrossTradeAddress:        l1CrossTradeAddress,
			L2CrossTradeProxyAddresses: l1l2CrossTradeProxyAddresses,
			L2CrossTradeAddresses:      l1l2CrossTradeAddresses,
		}
	} else {
		deployCrossTradeContractsOutput = &types.DeployCrossTradeContractsOutput{
			Mode:                       input.Mode,
			L1CrossTradeProxyAddress:   l1CrossTradeProxyAddress,
			L1CrossTradeAddress:        l1CrossTradeAddress,
			L2CrossTradeProxyAddresses: l2l2CrossTradeProxyAddresses,
			L2CrossTradeAddresses:      l2l2CrossTradeAddresses,
		}
	}

	// Save the inputs to the setting file
	if t.deployConfig.CrossTrade == nil {
		t.deployConfig.CrossTrade = make(map[constants.CrossTradeDeployMode]*types.CrossTrade)
	}

	if scratch {
		t.deployConfig.CrossTrade[input.Mode] = input
	} else { // Deploy new chain
		t.deployConfig.CrossTrade[input.Mode].L2ChainConfig = append(t.deployConfig.CrossTrade[input.Mode].L2ChainConfig, input.L2ChainConfig...)
	}

	if t.deployConfig.CrossTrade[input.Mode].Output == nil {
		t.deployConfig.CrossTrade[input.Mode].Output = &types.DeployCrossTradeOutput{
			DeployCrossTradeContractsOutput: deployCrossTradeContractsOutput,
		}
	} else {
		t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput.L1CrossTradeAddress = deployCrossTradeContractsOutput.L1CrossTradeAddress
		t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput.L1CrossTradeProxyAddress = deployCrossTradeContractsOutput.L1CrossTradeProxyAddress

		// Add new chain to the output
		for chainID, proxyAddress := range deployCrossTradeContractsOutput.L2CrossTradeProxyAddresses {
			t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput.L2CrossTradeProxyAddresses[chainID] = proxyAddress
		}
		for chainID, address := range deployCrossTradeContractsOutput.L2CrossTradeAddresses {
			t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput.L2CrossTradeAddresses[chainID] = address
		}
	}

	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save the inputs to the setting file: %s", err)
	}
	t.logger.Infof("Saved the cross-trade inputs to the setting file: %s", t.deploymentPath)

	_, err = t.DeployCrossTradeApplication(ctx, input.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy cross trade application: %s", err)
	}

	return &types.DeployCrossTradeOutput{
		DeployCrossTradeContractsOutput:   t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeContractsOutput,
		DeployCrossTradeApplicationOutput: t.deployConfig.CrossTrade[input.Mode].Output.DeployCrossTradeApplicationOutput,
		RegisterTokens:                    t.deployConfig.CrossTrade[input.Mode].RegisterTokens,
	}, nil
}

func (t *ThanosStack) DeployCrossTradeApplication(ctx context.Context, mode constants.CrossTradeDeployMode) (*types.DeployCrossTradeApplicationOutput, error) {
	// if t.deployConfig.K8s == nil {
	// 	t.logger.Error("K8s configuration is not set. Please run the deploy command first")
	// 	return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	// }

	// // STEP 1. Clone the charts repository
	// err := t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	// if err != nil {
	// 	t.logger.Error("Error cloning repository", "err", err)
	// 	return nil, err
	// }

	// input := t.deployConfig.CrossTrade[mode]
	// if input == nil {
	// 	return nil, fmt.Errorf("cross trade input is not set. Please run the deploy command first")
	// }

	// contracts := t.deployConfig.CrossTrade[mode].Output.DeployCrossTradeContractsOutput
	// if contracts == nil {
	// 	return nil, fmt.Errorf("contracts are not set. Please run the deploy command first")
	// }

	// var (
	// 	namespace = t.deployConfig.K8s.Namespace
	// 	l1ChainID = t.deployConfig.L1ChainID
	// )

	// t.logger.Info("Installing a cross trade component...")

	// // make yaml file at {cwd}/tokamak-thanos-stack/terraform/thanos-stack/cross-trade-values.yaml
	// crossTradeConfig := types.CrossTradeConfig{}

	// // Add L1 chain config
	// crossTradeConfig.CrossTrade.Env.NextPublicProjectID = "568b8d3d0528e743b0e2c6c92f54d721"

	// chainConfig := make(map[string]types.CrossTradeChainConfig)

	// l1Tokens := make([]*types.RegisterToken, 0)         // Token name -> Token address
	// l2Tokens := make(map[uint64][]*types.RegisterToken) // Chain ID -> Token name -> Token address
	// for _, tokenInput := range input.RegisterTokens {
	// 	registerL1Token := &types.RegisterToken{
	// 		Name:              tokenInput.TokenName,
	// 		Address:           tokenInput.L1TokenAddress,
	// 		DestinationChains: make([]uint64, 0),
	// 	}
	// 	if !slices.ContainsFunc(l1Tokens, func(token *types.RegisterToken) bool {
	// 		return token.Name == registerL1Token.Name && token.Address == registerL1Token.Address
	// 	}) {
	// 		l1Tokens = append(l1Tokens, registerL1Token)
	// 	}

	// 	for _, l2TokenInput := range tokenInput.L2TokenInputs {
	// 		if l2Tokens[l2TokenInput.ChainID] == nil {
	// 			l2Tokens[l2TokenInput.ChainID] = make([]*types.RegisterToken, 0)
	// 		}

	// 		// Get other destination chains
	// 		destinationChains := make([]uint64, 0)
	// 		for _, otherL2TokenInput := range tokenInput.L2TokenInputs {
	// 			if otherL2TokenInput.ChainID != l2TokenInput.ChainID {
	// 				destinationChains = append(destinationChains, otherL2TokenInput.ChainID)
	// 			}
	// 		}

	// 		// Check if a token with the same name and address already exists
	// 		var existingToken *types.RegisterToken
	// 		for _, existing := range l2Tokens[l2TokenInput.ChainID] {
	// 			if existing.Name == tokenInput.TokenName && existing.Address == l2TokenInput.TokenAddress {
	// 				existingToken = existing
	// 				break
	// 			}
	// 		}

	// 		if existingToken != nil {
	// 			// Merge destination chains, avoiding duplicates
	// 			for _, chainID := range destinationChains {
	// 				if !slices.Contains(existingToken.DestinationChains, chainID) {
	// 					existingToken.DestinationChains = append(existingToken.DestinationChains, chainID)
	// 				}
	// 			}
	// 		} else {
	// 			// Create a new token entry
	// 			registerL2Token := &types.RegisterToken{
	// 				Name:              tokenInput.TokenName,
	// 				Address:           l2TokenInput.TokenAddress,
	// 				DestinationChains: destinationChains,
	// 			}
	// 			l2Tokens[l2TokenInput.ChainID] = append(l2Tokens[l2TokenInput.ChainID], registerL2Token)
	// 		}
	// 	}
	// }

	// chainConfig[fmt.Sprintf("%d", l1ChainID)] = types.CrossTradeChainConfig{
	// 	Name:        constants.L1ChainConfigurations[l1ChainID].ChainName,
	// 	DisplayName: constants.L1ChainConfigurations[l1ChainID].ChainName,
	// 	Contracts: types.CrossTradeContracts{
	// 		L1CrossTrade: &contracts.L1CrossTradeProxyAddress,
	// 	},
	// 	RPCURL:            input.L1ChainConfig.RPC,
	// 	Tokens:            l1Tokens,
	// 	NativeTokenName:   constants.L1ChainConfigurations[l1ChainID].NativeTokenName,
	// 	NativeTokenSymbol: constants.L1ChainConfigurations[l1ChainID].NativeTokenSymbol,
	// }

	// l2ChainRPCs := make(map[uint64]string)

	// // Add L2 chain config
	// for _, chain := range input.L2ChainConfig {
	// 	l2ChainID := chain.ChainID
	// 	nativeTokenName := constants.L2ChainConfigurations[l2ChainID].NativeTokenName
	// 	nativeTokenSymbol := constants.L2ChainConfigurations[l2ChainID].NativeTokenSymbol
	// 	if l2ChainID == t.deployConfig.L2ChainID {
	// 		nativeTokenName = "Tokamak Network Token"
	// 		nativeTokenSymbol = "TON"
	// 	} else if nativeTokenName == "" || nativeTokenSymbol == "" {
	// 		nativeTokenName = "Ether"
	// 		nativeTokenSymbol = "ETH"
	// 	}
	// 	l2Tokens := l2Tokens[l2ChainID]
	// 	l2CrossTradeProxyAddress := contracts.L2CrossTradeProxyAddresses[l2ChainID]

	// 	chainConfig[fmt.Sprintf("%d", l2ChainID)] = types.CrossTradeChainConfig{
	// 		Name:        chain.ChainName,
	// 		DisplayName: chain.ChainName,
	// 		Contracts: types.CrossTradeContracts{
	// 			L2CrossTrade: &l2CrossTradeProxyAddress,
	// 		},
	// 		RPCURL:            l2ChainRPCs[l2ChainID],
	// 		Tokens:            l2Tokens,
	// 		NativeTokenName:   nativeTokenName,
	// 		NativeTokenSymbol: nativeTokenSymbol,
	// 	}
	// }

	// chainConfigJSON, err := json.Marshal(chainConfig)
	// if err != nil {
	// 	t.logger.Error("Error marshalling chain config", "err", err)
	// 	return nil, err
	// }

	// switch input.Mode {
	// case constants.CrossTradeDeployModeL2ToL1:
	// 	crossTradeConfig.CrossTrade.Env.L2L1Config = string(chainConfigJSON)
	// case constants.CrossTradeDeployModeL2ToL2:
	// 	crossTradeConfig.CrossTrade.Env.L2L2Config = string(chainConfigJSON)
	// }

	// // input from users

	// crossTradeConfig.CrossTrade.Ingress = types.Ingress{Enabled: true, ClassName: "alb", Annotations: map[string]string{
	// 	"alb.ingress.kubernetes.io/target-type":  "ip",
	// 	"alb.ingress.kubernetes.io/scheme":       "internet-facing",
	// 	"alb.ingress.kubernetes.io/listen-ports": "[{\"HTTP\": 80}]",
	// 	"alb.ingress.kubernetes.io/group.name":   "cross-trade",
	// }, TLS: types.TLS{
	// 	Enabled: false,
	// }}

	// data, err := yaml.Marshal(&crossTradeConfig)
	// if err != nil {
	// 	t.logger.Error("Error marshalling cross-trade values YAML file", "err", err)
	// 	return nil, err
	// }

	// configFileDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	// if err := os.MkdirAll(configFileDir, os.ModePerm); err != nil {
	// 	t.logger.Error("Error creating directory", "err", err)
	// 	return nil, err
	// }

	// // Write to file
	// filePath := filepath.Join(configFileDir, "/cross-trade-values.yaml")
	// err = os.WriteFile(filePath, data, 0644)
	// if err != nil {
	// 	t.logger.Error("Error writing file", "err", err)
	// 	return nil, nil
	// }

	// helmReleaseName := "cross-trade"

	// releases, err := utils.FilterHelmReleases(ctx, namespace, helmReleaseName)
	// if err != nil {
	// 	t.logger.Error("Error to filter helm releases", "err", err)
	// 	return nil, err
	// }

	// command := "install"
	// if len(releases) > 0 {
	// 	command = "upgrade"
	// }
	// _, err = utils.ExecuteCommand(ctx, "helm", []string{
	// 	command,
	// 	helmReleaseName,
	// 	fmt.Sprintf("%s/tokamak-thanos-stack/charts/cross-trade", t.deploymentPath),
	// 	"--values",
	// 	filePath,
	// 	"--namespace",
	// 	namespace,
	// }...)
	// if err != nil {
	// 	t.logger.Error("Error installing Helm charts", "err", err)
	// 	return nil, err
	// }

	// t.logger.Info("✅ Cross trade component installed successfully and is being initialized. Please wait for the ingress address to become available...")
	// var bridgeUrl string
	// for {
	// 	k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, helmReleaseName)
	// 	if err != nil {
	// 		t.logger.Error("Error retrieving ingress addresses", "err", err, "details", k8sIngresses)
	// 		return nil, err
	// 	}

	// 	if len(k8sIngresses) > 0 {
	// 		bridgeUrl = "http://" + k8sIngresses[0]
	// 		break
	// 	}

	// 	time.Sleep(15 * time.Second)
	// }
	// t.logger.Infof("✅ Cross trade component is up and running. You can access it at: %s", bridgeUrl)

	bridgeUrl := "http://localhost:8080"
	var err error
	output := &types.DeployCrossTradeApplicationOutput{
		URL: bridgeUrl,
	}
	t.deployConfig.CrossTrade[mode].Output.DeployCrossTradeApplicationOutput = output
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Error("Error saving configuration file", "err", err)
		return nil, err
	}

	return output, nil
}

func (t *ThanosStack) UninstallCrossTrade(ctx context.Context, mode constants.CrossTradeDeployMode) error {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	if t.deployConfig.AWS == nil {
		t.logger.Error("AWS configuration is not set. Please run the deploy command first")
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	releases, err := utils.FilterHelmReleases(ctx, namespace, "cross-trade")
	if err != nil {
		t.logger.Error("Error to filter helm releases", "err", err)
		return err
	}

	for _, release := range releases {
		_, err = utils.ExecuteCommand(ctx, "helm", []string{
			"uninstall",
			release,
			"--namespace",
			namespace,
		}...)
		if err != nil {
			t.logger.Error("❌ Error uninstalling cross-trade helm chart", "err", err)
			return err
		}
	}

	t.logger.Info("✅ Uninstall a cross-trade component successfully!")

	// Delete the cross trade config
	t.deployConfig.CrossTrade[mode] = nil
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Error("Error saving configuration file", "err", err)
		return err
	}

	return nil
}

func (t *ThanosStack) RegisterNewTokensOnExistingCrossTrade(
	ctx context.Context, mode constants.CrossTradeDeployMode,
	inputs []*types.RegisterTokenInput,
) (*types.DeployCrossTradeOutput, error) {
	crossTradeInput := t.deployConfig.CrossTrade[mode]
	if crossTradeInput == nil {
		return nil, fmt.Errorf("cross trade input is not set. Please run the deploy command first")
	}
	err := os.Chdir(t.deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to change directory to deployment path: %w", err)
	}

	l2ChainConfigs := make(map[uint64]*types.L2CrossTradeChainInput)
	for _, l2ChainConfig := range crossTradeInput.L2ChainConfig {
		l2ChainConfigs[l2ChainConfig.ChainID] = l2ChainConfig
	}
	// Fill out the missing fields in the input
	for _, tokenInput := range inputs {
		for _, l2TokenInput := range tokenInput.L2TokenInputs {
			chainID := l2TokenInput.ChainID
			l2ChainConfig, ok := l2ChainConfigs[chainID]
			if !ok {
				return nil, fmt.Errorf("l2 chain config is not set for chain ID: %d", chainID)
			}

			if l2TokenInput.L1L2CrossTradeProxyAddress == "" && mode == constants.CrossTradeDeployModeL2ToL1 {
				l2TokenInput.L1L2CrossTradeProxyAddress = crossTradeInput.Output.DeployCrossTradeContractsOutput.L2CrossTradeProxyAddresses[chainID]
			} else if l2TokenInput.L2L2CrossTradeProxyAddress == "" && mode == constants.CrossTradeDeployModeL2ToL2 {
				l2TokenInput.L2L2CrossTradeProxyAddress = crossTradeInput.Output.DeployCrossTradeContractsOutput.L2CrossTradeProxyAddresses[chainID]
			}

			if l2TokenInput.RPC == "" {
				l2TokenInput.RPC = l2ChainConfig.RPC
			}

			if l2TokenInput.PrivateKey == "" {
				l2TokenInput.PrivateKey = l2ChainConfig.PrivateKey
			}
		}
	}

	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		t.logger.Infof("Registering new tokens on existing cross-trade contracts for L2 to L1")
		for _, tokenInput := range inputs {
			l1TokenAddress := tokenInput.L1TokenAddress
			for _, l2TokenInput := range tokenInput.L2TokenInputs {
				l2TokenAddress := l2TokenInput.TokenAddress
				script := fmt.Sprintf(
					`cd crossTrade && PRIVATE_KEY=%s L2_CROSS_TRADE_PROXY=%s L1_TOKEN=%s L2_TOKEN=%s L1_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
					l2TokenInput.PrivateKey,
					l2TokenInput.L1L2CrossTradeProxyAddress,
					l1TokenAddress,
					l2TokenAddress,
					t.deployConfig.L1ChainID,
					"scripts/foundry_scripts/L2L1/RegisterToken_L2L1.sol:RegisterToken_L2L1",
					l2TokenInput.RPC,
				)
				t.logger.Infof("Registering token %s on L1 %s", l2TokenAddress, script)
				err := utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
				if err != nil {
					return nil, fmt.Errorf("failed to deploy the contracts: %s", err)

				}
			}
		}

	case constants.CrossTradeDeployModeL2ToL2:
		t.logger.Infof("Registering new tokens on existing cross-trade contracts for L2 to L2")

		for _, tokenInput := range inputs {
			l1TokenAddress := tokenInput.L1TokenAddress
			for _, l2TokenInput := range tokenInput.L2TokenInputs {
				otherL2Tokens := make([]*types.L2TokenInput, 0)
				for _, otherL2TokenInput := range tokenInput.L2TokenInputs {
					if otherL2TokenInput.ChainID != l2TokenInput.ChainID {
						otherL2Tokens = append(otherL2Tokens, otherL2TokenInput)
					}
				}

				l2TokenAddress := l2TokenInput.TokenAddress
				for _, otherL2Token := range otherL2Tokens {
					script := fmt.Sprintf(
						`cd crossTrade && PRIVATE_KEY=%s L2_CROSS_TRADE_PROXY=%s L1_TOKEN=%s L2_SOURCE_TOKEN=%s L2_DESTINATION_TOKEN=%s L1_CHAIN_ID=%d L2_SOURCE_CHAIN_ID=%d L2_DESTINATION_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
						l2TokenInput.PrivateKey,
						l2TokenInput.L2L2CrossTradeProxyAddress,
						l1TokenAddress,
						l2TokenAddress,
						otherL2Token.TokenAddress,
						t.deployConfig.L1ChainID,
						l2TokenInput.ChainID, // Source chain ID
						otherL2Token.ChainID, // Destination chain ID
						"scripts/foundry_scripts/RegisterToken_L2L2.sol:RegisterToken_L2L2",
						l2TokenInput.RPC,
					)
					t.logger.Infof("Registering token %s on L2 %s", l2TokenAddress, script)
					err := utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
					if err != nil {
						return nil, fmt.Errorf("failed to register token on L2: %s", err)
					}
				}
			}
		}
	}

	if t.deployConfig.CrossTrade[mode].RegisterTokens == nil {
		t.deployConfig.CrossTrade[mode].RegisterTokens = inputs
	} else {
		t.deployConfig.CrossTrade[mode].RegisterTokens = append(t.deployConfig.CrossTrade[mode].RegisterTokens, inputs...)
	}

	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save the inputs to the setting file: %s", err)
	}
	t.logger.Infof("Saved the cross-trade inputs to the setting file: %s", t.deploymentPath)

	contracts := t.deployConfig.CrossTrade[mode].Output.DeployCrossTradeContractsOutput
	if contracts == nil {
		return nil, fmt.Errorf("contracts are not set. Please run the deploy command first")
	}

	_, err = t.DeployCrossTradeApplication(ctx, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy cross trade application: %s", err)
	}

	return &types.DeployCrossTradeOutput{
		DeployCrossTradeContractsOutput:   t.deployConfig.CrossTrade[mode].Output.DeployCrossTradeContractsOutput,
		DeployCrossTradeApplicationOutput: t.deployConfig.CrossTrade[mode].Output.DeployCrossTradeApplicationOutput,
		RegisterTokens:                    t.deployConfig.CrossTrade[mode].RegisterTokens,
	}, nil
}

func (t *ThanosStack) GetCrossTradeConfiguration(ctx context.Context, mode constants.CrossTradeDeployMode) (*types.CrossTrade, error) {
	crossTradeInput := t.deployConfig.CrossTrade[mode]
	if crossTradeInput == nil {
		return nil, fmt.Errorf("cross trade input is not set. Please run the deploy command first")
	}

	return crossTradeInput, nil
}
