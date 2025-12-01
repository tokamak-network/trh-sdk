package crosstrade

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
)

const (
	DeployL1CrossTradeL2L1 = "DeployL1CrossTrade_L2L1.s.sol"
	DeployL2CrossTradeL2L1 = "DeployL2CrossTrade_L2L1.s.sol"
	DeployL1CrossTradeL2L2 = "DeployL1CrossTrade_L2L2.s.sol"
	DeployL2CrossTradeL2L2 = "DeployL2CrossTrade_L2L2.s.sol"
)

const (
	L2L2CrossTradeProxyL1ContractName = "L2toL2CrossTradeProxyL1"
	L2L2CrossTradeL1ContractName      = "L2toL2CrossTradeL1"
	L1L2CrossTradeProxyL1ContractName = "L1CrossTradeProxy"
	L1L2CrossTradeL1ContractName      = "L1CrossTrade"

	L2L2CrossTradeProxyL2ContractName = "L2toL2CrossTradeProxy"
	L2L2CrossTradeL2ContractName      = "L2toL2CrossTradeL2"
	L1L2CrossTradeProxyL2ContractName = "L2CrossTradeProxy"
	L1L2CrossTradeL2ContractName      = "L2CrossTrade"
)

const (
	L2L2ScriptPath = "scripts/foundry_scripts"
	L1L2ScriptPath = "scripts/foundry_scripts/L2L1"
)

// GetRegisterTokensFromPrompt interactively collects RegisterTokenInput entries
func GetRegisterTokensFromPrompt(
	ctx context.Context,
	logger *zap.SugaredLogger,
	deploymentPath string,
	mode constants.CrossTradeDeployMode,
	crossTradeConfig *types.CrossTrade,
) ([]*types.RegisterTokenInput, error) {
	registerTokenInputs := make([]*types.RegisterTokenInput, 0)
	seenTokenNames := make(map[string]struct{})

	if crossTradeConfig == nil {
		return nil, fmt.Errorf("cross trade config is required")
	}

	l2ChainConfigs := make(map[string]*types.L2CrossTradeChainInput)
	existingL2ChainNames := make([]string, 0)
	for _, l2ChainConfig := range crossTradeConfig.L2ChainConfig {
		l2ChainConfigs[l2ChainConfig.ChainName] = l2ChainConfig
		existingL2ChainNames = append(existingL2ChainNames, l2ChainConfig.ChainName)
	}

	fmt.Println("Please enter the token configuration")

	for {
		var err error

		var tokenName string
		for {
			fmt.Print("Please enter the token name: ")
			tokenName, err = scanner.ScanString()
			if err != nil {
				return nil, fmt.Errorf("failed to read token name: %w", err)
			}
			if tokenName == "" {
				fmt.Println("Token name cannot be empty")
				continue
			}
			if _, exists := seenTokenNames[tokenName]; exists {
				fmt.Printf("Token '%s' already exists. Please use a different name.\n", tokenName)
				continue
			}
			break
		}

		var l1Address string
		for {
			fmt.Print("Please enter the L1 token address: ")
			l1Address, err = scanner.ScanString()
			if err != nil {
				return nil, fmt.Errorf("failed to read L1 token address: %w", err)
			}
			if l1Address == "" {
				fmt.Println("L1 token address cannot be empty")
				continue
			}
			if !common.IsHexAddress(l1Address) {
				fmt.Println("Invalid L1 token address")
				continue
			}
			l1Address = common.HexToAddress(l1Address).Hex()
			break
		}
		fmt.Printf("   ↳ Added L1 token - address: %s\n", l1Address)

		l2TokenInputs := make([]*types.L2TokenInput, 0)
		seenL2Chains := make(map[uint64]struct{})
		seenL2ChainNames := make(map[string]struct{})
		l2TokenAddresses := make(map[string]string)

		for {
			if len(l2TokenInputs) >= 2 {
				fmt.Print("\nDo you want to add an L2 chain mapping for this token? (Y/n): ")
				addL2, err := scanner.ScanBool(true)
				if err != nil {
					return nil, fmt.Errorf("failed to read add L2 chain option: %w", err)
				}
				if !addL2 {
					if len(l2TokenInputs) < 2 {
						fmt.Println("At least two L2 chain mappings are required for each token.")
						continue
					}
					break
				}
			}

			var l2ChainConfig *types.L2CrossTradeChainInput
			for {
				fmt.Printf("Please enter the L2 chain name (choose from the following list: %s): ", strings.Join(existingL2ChainNames, ", "))
				l2ChainName, err := scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 chain name: %w", err)
				}
				if l2ChainName == "" {
					fmt.Println("L2 chain name cannot be empty")
					continue
				}
				if _, exists := l2ChainConfigs[l2ChainName]; !exists {
					fmt.Printf("L2 chain '%s' not found. Please enter a valid L2 chain name.\n", l2ChainName)
					continue
				}
				if _, exists := seenL2ChainNames[l2ChainName]; exists {
					fmt.Printf("L2 chain '%s' already added. Please enter a different L2 chain name.\n", l2ChainName)
					continue
				}
				l2ChainConfig = l2ChainConfigs[l2ChainName]
				seenL2ChainNames[l2ChainName] = struct{}{}
				break
			}

			if l2ChainConfig == nil {
				return nil, fmt.Errorf("L2 chain config not found")
			}

			l2RPC := l2ChainConfig.RPC
			l2RpcClient, err := ethclient.Dial(l2RPC)
			if err != nil {
				return nil, fmt.Errorf("failed to dial L2 RPC URL: %w", err)
			}

			chainID, err := l2RpcClient.ChainID(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get chain ID: %w", err)
			}
			l2ChainID := chainID.Uint64()

			var l2TokenAddress string
			for {
				fmt.Print("Please enter the L2 token address: ")
				l2TokenAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 token address: %w", err)
				}
				if l2TokenAddress == "" {
					fmt.Println("L2 token address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l2TokenAddress) {
					fmt.Println("Invalid L2 token address")
					continue
				}
				l2TokenAddress = common.HexToAddress(l2TokenAddress).Hex()
				break
			}

			privateKey := l2ChainConfig.PrivateKey
			if privateKey == "" {
				privateKey, err = getPrivateKeyFromPrompt(ctx, l2RpcClient)
				if err != nil {
					return nil, fmt.Errorf("failed to get private key: %w", err)
				}
			}

			var l2ContractNameFileName string
			var l2ContractName string
			var l2CrossTradeProxyAddress string
			switch mode {
			case constants.CrossTradeDeployModeL2ToL1:
				l2ContractNameFileName = DeployL2CrossTradeL2L1
				l2ContractName = L1L2CrossTradeProxyL1ContractName
			case constants.CrossTradeDeployModeL2ToL2:
				l2ContractNameFileName = DeployL2CrossTradeL2L2
				l2ContractName = L2L2CrossTradeProxyL2ContractName
			}

			filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", deploymentPath, l2ContractNameFileName, l2ChainID)
			contracts, err := utils.GetContractAddressFromOutput(filePath)
			if err != nil {
				logger.Error("Failed to get the contract address", "err", err)
			}
			if len(contracts) == 0 || contracts[l2ContractName] == "" {
				logger.Error("No contract address found", "contract_name", l2ContractName, "chain_id", l2ChainID)
				fmt.Println("Please enter the L2 cross-trade proxy address: ")
				l2CrossTradeProxyAddress, err = getContractAddressFromPrompt("L2 cross-trade proxy")
				if err != nil {
					return nil, fmt.Errorf("failed to get L2 cross-trade proxy address: %w", err)
				}
			} else {
				l2CrossTradeProxyAddress = contracts[l2ContractName]
			}

			var l1l2CrossTradeProxyAddress string
			var l2l2CrossTradeProxyAddress string
			if mode == constants.CrossTradeDeployModeL2ToL1 {
				l1l2CrossTradeProxyAddress = l2CrossTradeProxyAddress
				l2l2CrossTradeProxyAddress = ""
			} else {
				l1l2CrossTradeProxyAddress = ""
				l2l2CrossTradeProxyAddress = l2CrossTradeProxyAddress
			}

			seenL2Chains[l2ChainID] = struct{}{}
			l2TokenInputs = append(l2TokenInputs, &types.L2TokenInput{
				ChainID:                    l2ChainID,
				TokenAddress:               l2TokenAddress,
				RPC:                        l2RPC,
				PrivateKey:                 privateKey,
				L1L2CrossTradeProxyAddress: l1l2CrossTradeProxyAddress,
				L2L2CrossTradeProxyAddress: l2l2CrossTradeProxyAddress,
			})
			l2TokenAddresses[l2ChainConfig.ChainName] = l2TokenAddress
			fmt.Printf("   ↳ Added L2 chain '%s' - address: %s\n", l2ChainConfig.ChainName, l2TokenAddress)
		}

		seenTokenNames[tokenName] = struct{}{}
		registerTokenInputs = append(registerTokenInputs, &types.RegisterTokenInput{
			TokenName:      tokenName,
			L1TokenAddress: l1Address,
			L2TokenInputs:  l2TokenInputs,
		})

		l2TokenAddressesStr := make([]string, 0)
		for chainName, address := range l2TokenAddresses {
			l2TokenAddressesStr = append(l2TokenAddressesStr, fmt.Sprintf("%s: %s", chainName, address))
		}
		fmt.Printf("✅ Added token '%s' with %d L2 chain(s)\n\t↳ L1: %s: %s \n\t↳ L2: %s\n", tokenName, len(l2TokenInputs), crossTradeConfig.L1ChainConfig.ChainName, l1Address, strings.Join(l2TokenAddressesStr, ", "))
		fmt.Print("\nDo you want to add a token? (Y/n): ")
		addMore, err := scanner.ScanBool(true)
		if err != nil {
			return nil, fmt.Errorf("failed to read add token option: %w", err)
		}

		if !addMore {
			break
		}
	}

	return registerTokenInputs, nil
}

// GetL1ContractAddressesFromPrompt interactively collects L1 bridge configuration from user input
func GetL1ContractAddressesFromPrompt(chainID uint64) (*types.L1ContractAddressConfig, error) {
	var nativeTokenAddressOnL1 string
	var l1StandardBridgeAddress string
	var l1USDCBridgeAddress string
	var l1CrossDomainMessenger string
	var err error

	l1ContractAddress, exist := constants.L1ContractAddresses[chainID]
	if exist {
		return &types.L1ContractAddressConfig{
			NativeTokenAddress:            l1ContractAddress.NativeTokenAddress,
			L1StandardBridgeAddress:       l1ContractAddress.L1StandardBridgeAddress,
			L1USDCBridgeAddress:           l1ContractAddress.L1USDCBridgeAddress,
			L1CrossDomainMessengerAddress: l1ContractAddress.L1CrossDomainMessengerAddress,
		}, nil
	}

	fmt.Println("Please enter the contract addresses")

	for {
		fmt.Print("Please enter the native token address on L1: ")
		nativeTokenAddressOnL1, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to read native token address on L1: %w", err)
		}
		if nativeTokenAddressOnL1 == "" {
			fmt.Println("The native token address cannot be empty")
			continue
		}
		if !common.IsHexAddress(nativeTokenAddressOnL1) {
			fmt.Println("The native token address is invalid")
			continue
		}
		nativeTokenAddressOnL1 = common.HexToAddress(nativeTokenAddressOnL1).Hex()
		break
	}

	for {
		fmt.Print("Please enter the L1 Standard Bridge Address: ")
		l1StandardBridgeAddress, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to read L1 standard bridge address: %w", err)
		}
		if l1StandardBridgeAddress == "" {
			fmt.Println("L1 standard bridge address cannot be empty")
			continue
		}
		if !common.IsHexAddress(l1StandardBridgeAddress) {
			fmt.Println("Invalid L1 standard bridge address")
			continue
		}
		l1StandardBridgeAddress = common.HexToAddress(l1StandardBridgeAddress).Hex()
		break
	}

	for {
		fmt.Print("Please enter the L1 USDC Bridge Address: ")
		l1USDCBridgeAddress, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to read L1 USDC bridge address: %w", err)
		}
		if l1USDCBridgeAddress == "" {
			fmt.Println("L1 USDC bridge address cannot be empty")
			continue
		}
		if !common.IsHexAddress(l1USDCBridgeAddress) {
			fmt.Println("Invalid L1 USDC bridge address")
			continue
		}
		l1USDCBridgeAddress = common.HexToAddress(l1USDCBridgeAddress).Hex()
		break
	}

	for {
		fmt.Print("Please enter the L1 Cross Domain Messenger Address: ")
		l1CrossDomainMessenger, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to read L1 cross domain messenger address: %w", err)
		}
		if l1CrossDomainMessenger == "" {
			fmt.Println("L1 cross domain messenger address cannot be empty")
			continue
		}
		if !common.IsHexAddress(l1CrossDomainMessenger) {
			fmt.Println("Invalid L1 cross domain messenger address")
			continue
		}
		l1CrossDomainMessenger = common.HexToAddress(l1CrossDomainMessenger).Hex()
		break
	}

	return &types.L1ContractAddressConfig{
		NativeTokenAddress:            nativeTokenAddressOnL1,
		L1StandardBridgeAddress:       l1StandardBridgeAddress,
		L1USDCBridgeAddress:           l1USDCBridgeAddress,
		L1CrossDomainMessengerAddress: l1CrossDomainMessenger,
	}, nil
}

func GetCrossTradeContractsInputs(
	ctx context.Context,
	logger *zap.SugaredLogger,
	deploymentPath string,
	mode constants.CrossTradeDeployMode,
	deployConfig *types.Config,
) (*types.CrossTrade, error) {
	var (
		l1ContractFileName, l2ContractFileName  string
		l1CrossTradeProxyName, l1CrossTradeName string
		l2CrossTradeProxyName, l2CrossTradeName string
		deploymentScriptPath                    string
	)

	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		logger.Infof("Deploying the cross-trade contracts for L2 to L1")
		l1ContractFileName = DeployL1CrossTradeL2L1
		l2ContractFileName = DeployL2CrossTradeL2L1
		l1CrossTradeProxyName = L1L2CrossTradeProxyL1ContractName
		l1CrossTradeName = L1L2CrossTradeL1ContractName
		l2CrossTradeProxyName = L1L2CrossTradeProxyL2ContractName
		l2CrossTradeName = L1L2CrossTradeL2ContractName
		deploymentScriptPath = L1L2ScriptPath
	case constants.CrossTradeDeployModeL2ToL2:
		logger.Infof("Deploying the cross-trade contracts for L2 to L2")
		l1ContractFileName = DeployL1CrossTradeL2L2
		l2ContractFileName = DeployL2CrossTradeL2L2
		l1CrossTradeProxyName = L2L2CrossTradeProxyL1ContractName
		l1CrossTradeName = L2L2CrossTradeL1ContractName
		l2CrossTradeProxyName = L2L2CrossTradeProxyL2ContractName
		l2CrossTradeName = L2L2CrossTradeL2ContractName
		deploymentScriptPath = L2L2ScriptPath
	default:
		return nil, fmt.Errorf("invalid cross trade deploy mode: %s", mode)
	}

	var l1ChainConfig *types.L1CrossTradeChainInput

	fmt.Println("Please enter your configuration to deploy the L1 contracts to your L1 chain")

	// Ask if user wants to deploy new L1 contracts
	l1RPC := deployConfig.L1RPCURL
	l1ChainID := deployConfig.L1ChainID

	l1PrivateKey := deployConfig.AdminPrivateKey
	if !strings.HasPrefix(l1PrivateKey, "0x") {
		l1PrivateKey = "0x" + l1PrivateKey
	}

	// Get Etherscan API key (optional)

	// Get chain name
	fmt.Print("Please enter the L1 chain name: ")
	l1ChainName, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read L1 chain name: %w", err)
	}
	if l1ChainName == "" {
		return nil, fmt.Errorf("L1 chain name cannot be empty")
	}

	fmt.Print("Do you want to deploy new L1 cross-trade contracts? (Y/n): ")
	deployNewL1, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read L1 deployment option: %w", err)
	}

	var l1BlockExplorerConfig *types.BlockExplorerConfig
	fmt.Print("Please enter Etherscan API key to verify contracts (optional, press Enter to skip): ")
	l1EtherscanAPIKey, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read Etherscan API key: %w", err)
	}
	l1BlockExplorerConfig = &types.BlockExplorerConfig{
		APIKey: l1EtherscanAPIKey,
		Type:   constants.BlockExplorerTypeEtherscan,
	}

	if deployNewL1 {
		l1ChainConfig = &types.L1CrossTradeChainInput{
			RPC:                  l1RPC,
			ChainID:              l1ChainID,
			ContractName:         l1ContractFileName,
			PrivateKey:           l1PrivateKey,
			IsDeployedNew:        true,
			BlockExplorerConfig:  l1BlockExplorerConfig,
			DeploymentScriptPath: deploymentScriptPath,
			ChainName:            l1ChainName,
		}
	} else {
		filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", deploymentPath, l1ContractFileName, deployConfig.L1ChainID)
		contracts, err := utils.GetContractAddressFromOutput(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %w", err)
		}

		l1CrossTradeProxyAddress := contracts[l1CrossTradeProxyName]
		l1CrossTradeAddress := contracts[l1CrossTradeName]
		if l1CrossTradeProxyAddress == "" {
			l1CrossTradeProxyAddress, err = getContractAddressFromPrompt("L1 cross-trade proxy")
			if err != nil {
				return nil, fmt.Errorf("failed to get L1 cross-trade proxy address: %w", err)
			}
		}

		if l1CrossTradeAddress == "" {
			l1CrossTradeAddress, err = getContractAddressFromPrompt("L1 cross-trade")
			if err != nil {
				return nil, fmt.Errorf("failed to get L1 cross-trade address: %w", err)
			}
		}

		// Read for the deployment file
		l1ChainConfig = &types.L1CrossTradeChainInput{
			RPC:                    l1RPC,
			ChainID:                l1ChainID,
			ContractName:           l1ContractFileName,
			PrivateKey:             l1PrivateKey,
			IsDeployedNew:          false,
			CrossTradeProxyAddress: l1CrossTradeProxyAddress,
			CrossTradeAddress:      l1CrossTradeAddress,
			DeploymentScriptPath:   deploymentScriptPath,
			BlockExplorerConfig:    l1BlockExplorerConfig,
			ChainName:              l1ChainName,
		}
	}

	l2ChainConfigs := make([]*types.L2CrossTradeChainInput, 0)
	// Get current running L2 chain
	l2RPC := deployConfig.L2RpcUrl
	l2RpcClient, err := ethclient.Dial(l2RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}
	l2ChainID := deployConfig.L2ChainID

	fmt.Println("\nPlease enter your configuration to deploy the L2 contracts to your L2 chain")

	// Get chain name
	fmt.Print("Please enter the L2 chain name: ")
	l2ChainName, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read L2 chain name: %w", err)
	}
	if l2ChainName == "" {
		return nil, fmt.Errorf("L2 chain name cannot be empty")
	}

	fmt.Print("Do you want to deploy the L2 cross-trade contracts to the current L2 chain? (Y/n): ")
	deployNewL2, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read L2 deployment option: %w", err)
	}

	// Get block explorer URL
	var l2BlockExplorerConfig *types.BlockExplorerConfig
	l2BlockExplorerURL := deployConfig.BlockExplorerURL
	if l2BlockExplorerURL == "" {
		fmt.Println("No block explorer URL found, skip verifying L2 cross-trade contracts")
	} else {
		l2BlockExplorerConfig = &types.BlockExplorerConfig{
			URL:  l2BlockExplorerURL,
			Type: constants.BlockExplorerTypeBlockscout,
		}
	}

	var privateKey string
	if deployNewL2 {
		privateKey, err = getPrivateKeyFromPrompt(ctx, l2RpcClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get private key: %w", err)
		}

		// Get L1 bridge configuration from user
		l1ContractAddressConfig, err := GetL1ContractAddressesFromPrompt(l2ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
		}
		nativeTokenAddressOnL1 := l1ContractAddressConfig.NativeTokenAddress
		l1StandardBridgeAddress := l1ContractAddressConfig.L1StandardBridgeAddress
		l1USDCBridgeAddress := l1ContractAddressConfig.L1USDCBridgeAddress
		l1CrossDomainMessenger := l1ContractAddressConfig.L1CrossDomainMessengerAddress

		l2ChainConfigs = append(l2ChainConfigs, &types.L2CrossTradeChainInput{
			RPC:                     l2RPC,
			ChainID:                 l2ChainID,
			ContractName:            l2ContractFileName,
			PrivateKey:              privateKey,
			IsDeployedNew:           deployNewL2,
			BlockExplorerConfig:     l2BlockExplorerConfig,
			L1CrossDomainMessenger:  l1CrossDomainMessenger,
			CrossDomainMessenger:    constants.L2CrossDomainMessenger,
			DeploymentScriptPath:    deploymentScriptPath,
			NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
			L1StandardBridgeAddress: l1StandardBridgeAddress,
			L1USDCBridgeAddress:     l1USDCBridgeAddress,
			ChainName:               l2ChainName,
		})
	} else {
		filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", deploymentPath, l2ContractFileName, deployConfig.L2ChainID)
		contracts, err := utils.GetContractAddressFromOutput(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %w", err)
		}

		l2CrossTradeProxyAddress := contracts[l2CrossTradeProxyName]
		l2CrossTradeAddress := contracts[l2CrossTradeName]
		if l2CrossTradeProxyAddress == "" {
			l2CrossTradeProxyAddress, err = getContractAddressFromPrompt("L2 cross-trade proxy")
			if err != nil {
				return nil, fmt.Errorf("failed to get L2 cross-trade proxy address: %w", err)
			}
		}

		if l2CrossTradeAddress == "" {
			l2CrossTradeAddress, err = getContractAddressFromPrompt("L2 cross-trade")
			if err != nil {
				return nil, fmt.Errorf("failed to get L2 cross-trade address: %w", err)
			}
		}

		// Get L1 bridge configuration from user
		l1ContractAddressConfig, err := GetL1ContractAddressesFromPrompt(deployConfig.L2ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
		}
		nativeTokenAddressOnL1 := l1ContractAddressConfig.NativeTokenAddress
		l1StandardBridgeAddress := l1ContractAddressConfig.L1StandardBridgeAddress
		l1USDCBridgeAddress := l1ContractAddressConfig.L1USDCBridgeAddress
		l1CrossDomainMessenger := l1ContractAddressConfig.L1CrossDomainMessengerAddress

		// Read for the deployment file
		l2ChainConfigs = append(l2ChainConfigs, &types.L2CrossTradeChainInput{
			RPC:                     l2RPC,
			ChainID:                 l2ChainID,
			ContractName:            l2ContractFileName,
			PrivateKey:              privateKey,
			IsDeployedNew:           false,
			BlockExplorerConfig:     l2BlockExplorerConfig,
			L1CrossDomainMessenger:  l1CrossDomainMessenger,
			CrossDomainMessenger:    constants.L2CrossDomainMessenger,
			CrossTradeProxyAddress:  l2CrossTradeProxyAddress,
			CrossTradeAddress:       l2CrossTradeAddress,
			DeploymentScriptPath:    deploymentScriptPath,
			NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
			L1StandardBridgeAddress: l1StandardBridgeAddress,
			L1USDCBridgeAddress:     l1USDCBridgeAddress,
			ChainName:               l2ChainName,
		})
	}

	if mode == constants.CrossTradeDeployModeL2ToL2 {
		l2ChainConfig, err := getNewL2ChainRegistrationInputs(ctx, l2ContractFileName, deploymentScriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get new L2 chain registration inputs: %w", err)
		}
		l2ChainConfigs = append(l2ChainConfigs, l2ChainConfig)
	}

	return &types.CrossTrade{
		Mode:          mode,
		L1ChainConfig: l1ChainConfig,
		L2ChainConfig: l2ChainConfigs,
	}, nil
}

func GetNewChainRegistrationInputs(
	ctx context.Context,
	logger *zap.SugaredLogger,
	deploymentPath string,
	mode constants.CrossTradeDeployMode,
	deployConfig *types.Config,
) (*types.CrossTrade, error) {
	var (
		l1ContractFileName, l2ContractFileName  string
		l1CrossTradeProxyName, l1CrossTradeName string
		deploymentScriptPath                    string
	)

	if deployConfig.CrossTrade == nil || deployConfig.CrossTrade[mode] == nil {
		return nil, fmt.Errorf("cross trade config for mode %s is required", mode)
	}

	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		logger.Infof("Deploying the cross-trade contracts for L2 to L1")
		l1ContractFileName = DeployL1CrossTradeL2L1
		l2ContractFileName = DeployL2CrossTradeL2L1
		l1CrossTradeProxyName = L1L2CrossTradeProxyL1ContractName
		l1CrossTradeName = L1L2CrossTradeL1ContractName
		deploymentScriptPath = L1L2ScriptPath
	case constants.CrossTradeDeployModeL2ToL2:
		logger.Infof("Deploying the cross-trade contracts for L2 to L2")
		l1ContractFileName = DeployL1CrossTradeL2L2
		l2ContractFileName = DeployL2CrossTradeL2L2
		l1CrossTradeProxyName = L2L2CrossTradeProxyL1ContractName
		l1CrossTradeName = L2L2CrossTradeL1ContractName
		deploymentScriptPath = L2L2ScriptPath
	default:
		return nil, fmt.Errorf("invalid cross trade deploy mode: %s", mode)
	}

	// Ask if user wants to deploy new L1 contracts
	l1RPC := deployConfig.L1RPCURL
	l1ChainID := deployConfig.L1ChainID

	l1PrivateKey := deployConfig.AdminPrivateKey
	if !strings.HasPrefix(l1PrivateKey, "0x") {
		l1PrivateKey = "0x" + l1PrivateKey
	}

	filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", deploymentPath, l1ContractFileName, deployConfig.L1ChainID)
	contracts, err := utils.GetContractAddressFromOutput(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get the contract address: %w", err)
	}

	l1CrossTradeProxyAddress := contracts[l1CrossTradeProxyName]
	l1CrossTradeAddress := contracts[l1CrossTradeName]
	if l1CrossTradeProxyAddress == "" {
		l1CrossTradeProxyAddress, err = getContractAddressFromPrompt("L1 cross-trade proxy")
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 cross-trade proxy address: %w", err)
		}
	}

	if l1CrossTradeAddress == "" {
		l1CrossTradeAddress, err = getContractAddressFromPrompt("L1 cross-trade")
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 cross-trade address: %w", err)
		}
	}

	// Read for the deployment file
	l1ChainConfig := &types.L1CrossTradeChainInput{
		RPC:                    l1RPC,
		ChainID:                l1ChainID,
		ContractName:           l1ContractFileName,
		PrivateKey:             l1PrivateKey,
		IsDeployedNew:          false,
		CrossTradeProxyAddress: l1CrossTradeProxyAddress,
		CrossTradeAddress:      l1CrossTradeAddress,
		DeploymentScriptPath:   deploymentScriptPath,
		BlockExplorerConfig:    nil,
		ChainName:              deployConfig.CrossTrade[mode].L1ChainConfig.ChainName,
	}

	l2ChainConfigs := make([]*types.L2CrossTradeChainInput, 0)

	if mode == constants.CrossTradeDeployModeL2ToL2 {
		l2ChainConfig, err := getNewL2ChainRegistrationInputs(ctx, l2ContractFileName, deploymentScriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get new L2 chain registration inputs: %w", err)
		}
		l2ChainConfigs = append(l2ChainConfigs, l2ChainConfig)
	}

	return &types.CrossTrade{
		Mode:          mode,
		L1ChainConfig: l1ChainConfig,
		L2ChainConfig: l2ChainConfigs,
	}, nil
}

func getNewL2ChainRegistrationInputs(ctx context.Context, l2ContractFileName string, deploymentScriptPath string) (*types.L2CrossTradeChainInput, error) {
	fmt.Println("Please enter your configuration to deploy the L2 contracts to the new L2 chain")

	fmt.Print("Please enter the new L2 chain name: ")
	newL2ChainName, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read new L2 chain name: %w", err)
	}
	if newL2ChainName == "" {
		return nil, fmt.Errorf("new L2 chain name cannot be empty")
	}

	fmt.Print("Do you want to deploy contracts to the new L2 chain? (Y/n): ")
	addOtherL2, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read other L2 chain option: %w", err)
	}

	var otherL2BlockExplorerConfig *types.BlockExplorerConfig
	fmt.Print("Please enter Etherscan API key to verify contracts (optional, press Enter to skip): ")
	otherL2EtherscanAPIKey, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read Etherscan API key: %w", err)
	}
	otherL2BlockExplorerConfig = &types.BlockExplorerConfig{
		APIKey: otherL2EtherscanAPIKey,
		Type:   constants.BlockExplorerTypeEtherscan,
	}

	var otherChainID *big.Int
	fmt.Print("Please enter the RPC URL: ")
	otherRpc, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("failed to read RPC URL: %w", err)
	}
	if otherRpc == "" {
		return nil, fmt.Errorf("RPC URL cannot be empty")
	}

	l2RpcClient, err := ethclient.Dial(otherRpc)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}

	otherChainID, err = l2RpcClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if addOtherL2 {
		otherPrivateKey, err := getPrivateKeyFromPrompt(ctx, l2RpcClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get private key: %w", err)
		}
		// Get L1 bridge configuration from user
		l1ContractAddressConfig, err := GetL1ContractAddressesFromPrompt(otherChainID.Uint64())
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
		}
		nativeTokenAddressOnL1 := l1ContractAddressConfig.NativeTokenAddress
		l1StandardBridgeAddress := l1ContractAddressConfig.L1StandardBridgeAddress
		l1USDCBridgeAddress := l1ContractAddressConfig.L1USDCBridgeAddress
		l1CrossDomainMessenger := l1ContractAddressConfig.L1CrossDomainMessengerAddress

		return &types.L2CrossTradeChainInput{
			RPC:                     otherRpc,
			ChainName:               newL2ChainName,
			ChainID:                 otherChainID.Uint64(),
			ContractName:            l2ContractFileName,
			PrivateKey:              otherPrivateKey,
			IsDeployedNew:           addOtherL2,
			BlockExplorerConfig:     otherL2BlockExplorerConfig,
			L1CrossDomainMessenger:  l1CrossDomainMessenger,
			CrossDomainMessenger:    constants.L2CrossDomainMessenger,
			DeploymentScriptPath:    deploymentScriptPath,
			NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
			L1StandardBridgeAddress: l1StandardBridgeAddress,
			L1USDCBridgeAddress:     l1USDCBridgeAddress,
		}, nil
	} else {
		var l2CrossTradeProxyAddress string
		var l2CrossTradeAddress string

		l2CrossTradeProxyAddress, err = getContractAddressFromPrompt("L2 cross-trade proxy")
		if err != nil {
			return nil, fmt.Errorf("failed to get L2 cross-trade proxy address: %w", err)
		}

		l2CrossTradeAddress, err = getContractAddressFromPrompt("L2 cross-trade")
		if err != nil {
			return nil, fmt.Errorf("failed to get L2 cross-trade address: %w", err)
		}

		otherPrivateKey, err := getPrivateKeyFromPrompt(ctx, l2RpcClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get private key: %w", err)
		}

		// Get L1 bridge configuration from user
		l1ContractAddressConfig, err := GetL1ContractAddressesFromPrompt(otherChainID.Uint64())
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
		}
		nativeTokenAddressOnL1 := l1ContractAddressConfig.NativeTokenAddress
		l1StandardBridgeAddress := l1ContractAddressConfig.L1StandardBridgeAddress
		l1USDCBridgeAddress := l1ContractAddressConfig.L1USDCBridgeAddress
		l1CrossDomainMessenger := l1ContractAddressConfig.L1CrossDomainMessengerAddress

		return &types.L2CrossTradeChainInput{
			RPC:                     otherRpc,
			ChainID:                 otherChainID.Uint64(),
			ContractName:            l2ContractFileName,
			PrivateKey:              otherPrivateKey,
			IsDeployedNew:           false,
			BlockExplorerConfig:     otherL2BlockExplorerConfig,
			L1CrossDomainMessenger:  l1CrossDomainMessenger,
			CrossDomainMessenger:    constants.L2CrossDomainMessenger,
			CrossTradeProxyAddress:  l2CrossTradeProxyAddress,
			CrossTradeAddress:       l2CrossTradeAddress,
			DeploymentScriptPath:    deploymentScriptPath,
			NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
			L1StandardBridgeAddress: l1StandardBridgeAddress,
			L1USDCBridgeAddress:     l1USDCBridgeAddress,
		}, nil
	}
}

func getContractAddressFromPrompt(name string) (string, error) {
	var contractAddress string
	var err error
	for {
		fmt.Printf("Please enter the %s address: ", name)
		contractAddress, err = scanner.ScanString()
		if err != nil {
			return "", fmt.Errorf("failed to read %s address: %w", name, err)
		}
		if contractAddress == "" {
			fmt.Printf("%s address cannot be empty", name)
			continue
		}
		if !common.IsHexAddress(contractAddress) {
			fmt.Printf("Invalid %s address", name)
			continue
		}
		break
	}
	contractAddress = common.HexToAddress(contractAddress).Hex()
	return contractAddress, nil
}

func getPrivateKeyFromPrompt(ctx context.Context, client *ethclient.Client) (string, error) {
	var privateKey string
	var err error
	for {
		fmt.Print("Please enter the private key: ")
		privateKey, err = scanner.ScanString()
		if err != nil {
			return "", fmt.Errorf("failed to read private key: %w", err)
		}
		if privateKey == "" {
			fmt.Println("Private key cannot be empty")
			continue
		}

		// Validate this private key is valid
		privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
		if err != nil {
			fmt.Println("Invalid private key: ", err)
		}
		address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
		balance, err := client.BalanceAt(ctx, address, nil)
		if err != nil {
			fmt.Println("Failed to get balance: ", err)

		}
		if balance.Cmp(big.NewInt(0)) == 0 {
			fmt.Println("Balance is 0, please enter a valid private key")
			continue
		}
		break
	}
	if !strings.HasPrefix(privateKey, "0x") {
		privateKey = "0x" + privateKey
	}
	return privateKey, nil
}
