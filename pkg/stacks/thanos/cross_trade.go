package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

type BlockExplorerConfig struct {
	APIKey string                      `json:"api_key"`
	URL    string                      `json:"url"`
	Type   constants.BlockExplorerType `json:"type"`
}

type L1CrossTradeChainInput struct {
	RPC                    string               `json:"rpc"`
	ChainID                uint64               `json:"chain_id"`
	PrivateKey             string               `json:"private_key"`
	IsDeployedNew          bool                 `json:"is_deployed_new"`
	DeploymentScriptPath   string               `json:"deployment_script_path"`
	ContractName           string               `json:"contract_name"`
	BlockExplorerConfig    *BlockExplorerConfig `json:"block_explorer_config"`
	CrossTradeProxyAddress string               `json:"cross_trade_proxy_address"`
	CrossTradeAddress      string               `json:"cross_trade_address"`
}

type L2CrossTradeChainInput struct {
	RPC                     string               `json:"rpc"`
	ChainID                 uint64               `json:"chain_id"`
	PrivateKey              string               `json:"private_key"`
	IsDeployedNew           bool                 `json:"is_deployed_new"`
	DeploymentScriptPath    string               `json:"deployment_script_path"`
	ContractName            string               `json:"contract_name"`
	BlockExplorerConfig     *BlockExplorerConfig `json:"block_explorer_config"`
	CrossDomainMessenger    string               `json:"cross_domain_messenger"`
	CrossTradeProxyAddress  string               `json:"cross_trade_proxy_address"`
	CrossTradeAddress       string               `json:"cross_trade_address"`
	L2Tokens                map[string]string    `json:"l2_tokens"`
	L1Tokens                map[string]string    `json:"l1_tokens"`
	NativeTokenAddressOnL1  string               `json:"native_token_address"`
	L1StandardBridgeAddress string               `json:"l1_standard_bridge_address"`
	L1USDCBridgeAddress     string               `json:"l1_usdc_bridge_address"`
	L1CrossDomainMessenger  string               `json:"l1_cross_domain_messenger"`
}

type DeployCrossTradeInputs struct {
	Mode          constants.CrossTradeDeployMode `json:"mode"`
	ProjectID     string                         `json:"project_id"`
	L1ChainConfig *L1CrossTradeChainInput        `json:"l1_chain_config"`
	L2ChainConfig []*L2CrossTradeChainInput      `json:"l2_chain_config"`
}

type DeployCrossTradeContractsOutput struct {
	Mode                       constants.CrossTradeDeployMode `json:"mode"`
	L1CrossTradeProxyAddress   string                         `json:"l1_cross_trade_proxy_address"`
	L1CrossTradeAddress        string                         `json:"l1_cross_trade_address"`
	L2CrossTradeProxyAddresses map[uint64]string              `json:"l2_cross_trade_proxy_addresses"`
	L2CrossTradeAddresses      map[uint64]string              `json:"l2_l2_cross_trade_addresses"`
}

type DeployCrossTradeApplicationOutput struct {
	URL string `json:"url"`
}

type DeployCrossTradeOutput struct {
	DeployCrossTradeContractsOutput   *DeployCrossTradeContractsOutput   `json:"deploy_cross_trade_contracts_output"`
	DeployCrossTradeApplicationOutput *DeployCrossTradeApplicationOutput `json:"deploy_cross_trade_application_output"`
}

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

// getTokenInputsFromUser interactively collects L1Tokens and L2Tokens from user input
func getTokenInputsFromUser() (map[string]string, map[string]string, error) {
	l1Tokens := make(map[string]string)
	l2Tokens := make(map[string]string)

	fmt.Println("\n--------------------------------")
	fmt.Println("Token Configuration")
	fmt.Println("--------------------------------")

	for {
		fmt.Print("\nDo you want to add a token? (Y/n): ")
		addMore, err := scanner.ScanBool(true)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read add token option: %w", err)
		}

		if !addMore {
			break
		}

		var tokenName string
		for {
			fmt.Print("Please enter the token name: ")
			tokenName, err = scanner.ScanString()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read token name: %w", err)
			}
			if tokenName == "" {
				fmt.Println("Token name cannot be empty")
				continue
			}
			// Check if token name already exists
			if _, exists := l1Tokens[tokenName]; exists {
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
				return nil, nil, fmt.Errorf("failed to read L1 token address: %w", err)
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

		var l2Address string
		for {
			fmt.Print("Please enter the L2 token address: ")
			l2Address, err = scanner.ScanString()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read L2 token address: %w", err)
			}
			if l2Address == "" {
				fmt.Println("L2 token address cannot be empty")
				continue
			}
			if !common.IsHexAddress(l2Address) {
				fmt.Println("Invalid L2 token address")
				continue
			}
			l2Address = common.HexToAddress(l2Address).Hex()
			break
		}

		l1Tokens[tokenName] = l1Address
		l2Tokens[tokenName] = l2Address
		fmt.Printf("✅ Added token '%s' - L1: %s, L2: %s\n", tokenName, l1Address, l2Address)
	}

	return l1Tokens, l2Tokens, nil
}

// getL1BridgeConfigFromUser interactively collects L1 bridge configuration from user input
func getL1BridgeConfigFromUser() (string, string, string, string, error) {
	var nativeTokenAddressOnL1 string
	var l1StandardBridgeAddress string
	var l1USDCBridgeAddress string
	var l1CrossDomainMessenger string
	var err error

	fmt.Println("\n--------------------------------")
	fmt.Println("L1 Bridge Configuration")
	fmt.Println("--------------------------------")

	for {
		fmt.Print("Please enter the Native Token Address on L1: ")
		nativeTokenAddressOnL1, err = scanner.ScanString()
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to read native token address on L1: %w", err)
		}
		if nativeTokenAddressOnL1 == "" {
			fmt.Println("Native token address on L1 cannot be empty")
			continue
		}
		if !common.IsHexAddress(nativeTokenAddressOnL1) {
			fmt.Println("Invalid native token address on L1")
			continue
		}
		nativeTokenAddressOnL1 = common.HexToAddress(nativeTokenAddressOnL1).Hex()
		break
	}

	for {
		fmt.Print("Please enter the L1 Standard Bridge Address: ")
		l1StandardBridgeAddress, err = scanner.ScanString()
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to read L1 standard bridge address: %w", err)
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
			return "", "", "", "", fmt.Errorf("failed to read L1 USDC bridge address: %w", err)
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
			return "", "", "", "", fmt.Errorf("failed to read L1 cross domain messenger address: %w", err)
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

	return nativeTokenAddressOnL1, l1StandardBridgeAddress, l1USDCBridgeAddress, l1CrossDomainMessenger, nil
}

func (t *ThanosStack) GetCrossTradeContractsInputs(ctx context.Context, mode constants.CrossTradeDeployMode) (*DeployCrossTradeInputs, error) {
	var (
		l1ContractFileName, l2ContractFileName  string
		l1CrossTradeProxyName, l1CrossTradeName string
		l2CrossTradeProxyName, l2CrossTradeName string
		deploymentScriptPath                    string
	)

	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		t.logger.Infof("Deploying the cross-trade contracts for L2 to L1")
		l1ContractFileName = DeployL1CrossTradeL2L1
		l2ContractFileName = DeployL2CrossTradeL2L1
		l1CrossTradeProxyName = L1L2CrossTradeProxyL1ContractName
		l1CrossTradeName = L1L2CrossTradeL1ContractName
		l2CrossTradeProxyName = L1L2CrossTradeProxyL2ContractName
		l2CrossTradeName = L1L2CrossTradeL2ContractName
		deploymentScriptPath = L1L2ScriptPath
	case constants.CrossTradeDeployModeL2ToL2:
		t.logger.Infof("Deploying the cross-trade contracts for L2 to L2")
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

	var l1ChainConfig *L1CrossTradeChainInput

	fmt.Println("Please enter your configuration to deploy the L1 contracts to your L1 chain")

	// Ask if user wants to deploy new L1 contracts
	l1RPC := t.deployConfig.L1RPCURL
	l1ChainID := t.deployConfig.L1ChainID

	l1PrivateKey := t.deployConfig.AdminPrivateKey

	// Get Etherscan API key (optional)
	fmt.Print("Do you want to deploy new L1 cross-trade contracts? (Y/n): ")
	deployNewL1, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read L1 deployment option: %w", err)
	}

	fmt.Print("Do you want to verify the L1 cross-trade contracts? (Y/n): ")
	verifyL1, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read L1 verification option: %w", err)
	}

	var l1BlockExplorerConfig *BlockExplorerConfig
	if verifyL1 {
		fmt.Print("Please enter Etherscan API key (optional, press Enter to skip): ")
		l1EtherscanAPIKey, err := scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to read Etherscan API key: %w", err)
		}
		l1BlockExplorerConfig = &BlockExplorerConfig{
			APIKey: l1EtherscanAPIKey,
			Type:   constants.BlockExplorerTypeEtherscan,
		}
	}

	if deployNewL1 {
		l1ChainConfig = &L1CrossTradeChainInput{
			RPC:                  l1RPC,
			ChainID:              l1ChainID,
			ContractName:         l1ContractFileName,
			PrivateKey:           "0x" + l1PrivateKey,
			IsDeployedNew:        true,
			BlockExplorerConfig:  l1BlockExplorerConfig,
			DeploymentScriptPath: deploymentScriptPath,
		}
	} else {
		contracts, err := t.getContractAddressFromOutput(ctx, l1ContractFileName, t.deployConfig.L1ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %w", err)
		}

		l1CrossTradeProxyAddress := contracts[l1CrossTradeProxyName]
		l1CrossTradeAddress := contracts[l1CrossTradeName]
		if l1CrossTradeProxyAddress == "" {
			for {
				fmt.Print("Please enter the L1 cross-trade proxy address: ")
				l1CrossTradeProxyAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L1 cross-trade proxy address: %w", err)
				}
				if l1CrossTradeProxyAddress == "" {
					fmt.Println("L1 cross-trade proxy address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l1CrossTradeProxyAddress) {
					fmt.Println("Invalid L1 cross-trade proxy address")
					continue
				}
				break
			}
			l1CrossTradeProxyAddress = common.HexToAddress(l1CrossTradeProxyAddress).Hex()
		}

		if l1CrossTradeAddress == "" {
			for {
				fmt.Print("Please enter the L1 cross-trade address: ")
				l1CrossTradeAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L1 cross-trade address: %w", err)
				}
				if l1CrossTradeAddress == "" {
					fmt.Println("L1 cross-trade address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l1CrossTradeAddress) {
					fmt.Println("Invalid L1 cross-trade address")
					continue
				}
				break
			}
			l1CrossTradeAddress = common.HexToAddress(l1CrossTradeAddress).Hex()
		}

		// Read for the deployment file
		l1ChainConfig = &L1CrossTradeChainInput{
			RPC:                    l1RPC,
			ChainID:                l1ChainID,
			ContractName:           l1ContractFileName,
			PrivateKey:             "0x" + l1PrivateKey,
			IsDeployedNew:          false,
			CrossTradeProxyAddress: l1CrossTradeProxyAddress,
			CrossTradeAddress:      l1CrossTradeAddress,
			DeploymentScriptPath:   deploymentScriptPath,
			BlockExplorerConfig:    l1BlockExplorerConfig,
		}
	}

	l2ChainConfigs := make([]*L2CrossTradeChainInput, 0)
	// Get current running L2 chain
	l2RPC := t.deployConfig.L2RpcUrl
	l2RpcClient, err := ethclient.Dial(l2RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}
	l2ChainID := t.deployConfig.L2ChainID

	fmt.Println("\n--------------------------------")
	fmt.Println("\nPlease enter your configuration to deploy the L2 contracts to your L2 chain")
	fmt.Print("Do you want to deploy the L2 cross-trade contracts to the current L2 chain? (Y/n): ")
	deployNewL2, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read L2 deployment option: %w", err)
	}

	// Get block explorer URL
	var l2BlockExplorerConfig *BlockExplorerConfig
	l2BlockExplorerURL, err := t.GetBlockExplorerURL(ctx)
	if err != nil {
		fmt.Println("No block explorer URL found, skip verifying L2 cross-trade contracts")
	} else {
		l2BlockExplorerConfig = &BlockExplorerConfig{
			URL:  l2BlockExplorerURL,
			Type: constants.BlockExplorerTypeBlockscout,
		}
	}

	var privateKey string
	if deployNewL2 {
		for {
			fmt.Print("Please enter the private key: ")
			privateKey, err = scanner.ScanString()
			if err != nil {
				fmt.Println("Failed to read L2 private key: ", err)
				continue
			}
			if privateKey == "" {
				fmt.Println("L2 private key cannot be empty")
				continue
			}

			// Validate this private key is valid
			privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
			if err != nil {
				fmt.Println("Invalid private key: ", err)
				continue
			}

			if !strings.HasPrefix(privateKey, "0x") {
				privateKey = "0x" + privateKey
			}

			address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
			balance, err := l2RpcClient.BalanceAt(ctx, address, nil)
			if err != nil {
				fmt.Println("Failed to get balance: ", err)
				continue
			}

			if balance.Cmp(big.NewInt(0)) == 0 {
				fmt.Println("Balance is 0, please enter a valid private key")
				continue
			}

			break
		}

		// Get token inputs from user
		l1Tokens, l2Tokens, err := getTokenInputsFromUser()
		if err != nil {
			return nil, fmt.Errorf("failed to get token inputs: %w", err)
		}

		// Get L1 bridge configuration from user
		nativeTokenAddressOnL1, l1StandardBridgeAddress, l1USDCBridgeAddress, l1CrossDomainMessenger, err := getL1BridgeConfigFromUser()
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
		}

		l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
			RPC:                     l2RPC,
			ChainID:                 l2ChainID,
			ContractName:            l2ContractFileName,
			PrivateKey:              privateKey,
			IsDeployedNew:           deployNewL2,
			BlockExplorerConfig:     l2BlockExplorerConfig,
			L1CrossDomainMessenger:  l1CrossDomainMessenger,
			CrossDomainMessenger:    constants.L2CrossDomainMessenger,
			DeploymentScriptPath:    deploymentScriptPath,
			L1Tokens:                l1Tokens,
			L2Tokens:                l2Tokens,
			NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
			L1StandardBridgeAddress: l1StandardBridgeAddress,
			L1USDCBridgeAddress:     l1USDCBridgeAddress,
		})
	} else {
		contracts, err := t.getContractAddressFromOutput(ctx, l2ContractFileName, t.deployConfig.L2ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %w", err)
		}

		l2CrossTradeProxyAddress := contracts[l2CrossTradeProxyName]
		l2CrossTradeAddress := contracts[l2CrossTradeName]
		if l2CrossTradeProxyAddress == "" {
			for {
				fmt.Print("Please enter the L2 cross-trade proxy address: ")
				l2CrossTradeProxyAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 cross-trade proxy address: %w", err)
				}
				if l2CrossTradeProxyAddress == "" {
					fmt.Println("L2 cross-trade proxy address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l2CrossTradeProxyAddress) {
					fmt.Println("Invalid L1 cross-trade proxy address")
					continue
				}
				break
			}
			l2CrossTradeProxyAddress = common.HexToAddress(l2CrossTradeProxyAddress).Hex()
		}

		if l2CrossTradeAddress == "" {
			for {
				fmt.Print("Please enter the L2 cross-trade address: ")
				l2CrossTradeAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 cross-trade address: %w", err)
				}
				if l2CrossTradeAddress == "" {
					fmt.Println("L2 cross-trade address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l2CrossTradeAddress) {
					fmt.Println("Invalid L1 cross-trade address")
					continue
				}
				break
			}
			l2CrossTradeAddress = common.HexToAddress(l2CrossTradeAddress).Hex()
		}

		// Get token inputs from user
		l1Tokens, l2Tokens, err := getTokenInputsFromUser()
		if err != nil {
			return nil, fmt.Errorf("failed to get token inputs: %w", err)
		}

		// Get L1 bridge configuration from user
		nativeTokenAddressOnL1, l1StandardBridgeAddress, l1USDCBridgeAddress, l1CrossDomainMessenger, err := getL1BridgeConfigFromUser()
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
		}

		// Read for the deployment file
		l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
			RPC:                     l2RPC,
			ChainID:                 l2ChainID,
			ContractName:            l2ContractFileName,
			PrivateKey:              "0x" + privateKey,
			IsDeployedNew:           false,
			BlockExplorerConfig:     l2BlockExplorerConfig,
			L1CrossDomainMessenger:  l1CrossDomainMessenger,
			CrossDomainMessenger:    constants.L2CrossDomainMessenger,
			CrossTradeProxyAddress:  l2CrossTradeProxyAddress,
			CrossTradeAddress:       l2CrossTradeAddress,
			DeploymentScriptPath:    deploymentScriptPath,
			L1Tokens:                l1Tokens,
			L2Tokens:                l2Tokens,
			NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
			L1StandardBridgeAddress: l1StandardBridgeAddress,
			L1USDCBridgeAddress:     l1USDCBridgeAddress,
		})
	}

	if mode == constants.CrossTradeDeployModeL2ToL2 {
		fmt.Println("Please enter your configuration to deploy the L2 contracts to other L2 chain")
		fmt.Print("Do you want to deploy contracts to other L2 chain? (Y/n): ")
		addOtherL2, err := scanner.ScanBool(true)
		if err != nil {
			return nil, fmt.Errorf("failed to read other L2 chain option: %w", err)
		}

		fmt.Print("Do you want to verify the L2 cross-trade contracts? (Y/n): ")
		verifyOtherL2, err := scanner.ScanBool(true)
		if err != nil {
			return nil, fmt.Errorf("failed to read L2 verification option: %w", err)
		}

		var otherL2BlockExplorerConfig *BlockExplorerConfig
		if verifyOtherL2 {
			fmt.Print("Please enter the Etherscan API key: ")
			otherL2EtherscanAPIKey, err := scanner.ScanString()
			if err != nil {
				return nil, fmt.Errorf("failed to read Etherscan API key: %w", err)
			}
			otherL2BlockExplorerConfig = &BlockExplorerConfig{
				APIKey: otherL2EtherscanAPIKey,
				Type:   constants.BlockExplorerTypeEtherscan,
			}
		}

		var otherChainID *big.Int
		if addOtherL2 {
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

			var otherPrivateKey string
			for {
				fmt.Print("Please enter the private key: ")
				otherPrivateKey, err = scanner.ScanString()
				if err != nil {
					fmt.Println("Failed to read private key: ", err)
					continue
				}
				if otherPrivateKey == "" {
					fmt.Println("Private key cannot be empty")
					continue
				}

				// Validate this private key is valid
				privateKeyECDSA, err := crypto.HexToECDSA(otherPrivateKey)
				if err != nil {
					fmt.Println("Invalid private key: ", err)
					continue
				}

				if !strings.HasPrefix(otherPrivateKey, "0x") {
					otherPrivateKey = "0x" + otherPrivateKey
				}

				// Get balance of this private key
				address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
				balance, err := l2RpcClient.BalanceAt(ctx, address, nil)
				if err != nil {
					fmt.Println("Failed to get balance: ", err)
					continue
				}

				if balance.Cmp(big.NewInt(0)) == 0 {
					fmt.Println("Balance is 0, please enter a valid private key")
					continue
				}

				break
			}

			otherChainID, err = l2RpcClient.ChainID(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get chain ID: %w", err)
			}

			// Get token inputs from user
			l1Tokens, l2Tokens, err := getTokenInputsFromUser()
			if err != nil {
				return nil, fmt.Errorf("failed to get token inputs: %w", err)
			}

			// Get L1 bridge configuration from user
			nativeTokenAddressOnL1, l1StandardBridgeAddress, l1USDCBridgeAddress, l1CrossDomainMessenger, err := getL1BridgeConfigFromUser()
			if err != nil {
				return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
			}

			l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
				RPC:                     otherRpc,
				ChainID:                 otherChainID.Uint64(),
				ContractName:            l2ContractFileName,
				PrivateKey:              otherPrivateKey,
				IsDeployedNew:           addOtherL2,
				BlockExplorerConfig:     otherL2BlockExplorerConfig,
				L1CrossDomainMessenger:  l1CrossDomainMessenger,
				CrossDomainMessenger:    constants.L2CrossDomainMessenger,
				DeploymentScriptPath:    deploymentScriptPath,
				L1Tokens:                l1Tokens,
				L2Tokens:                l2Tokens,
				NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
				L1StandardBridgeAddress: l1StandardBridgeAddress,
				L1USDCBridgeAddress:     l1USDCBridgeAddress,
			})
		} else {
			var l2CrossTradeProxyAddress string
			var l2CrossTradeAddress string

			var l2ChainID int
			for {
				fmt.Print("Please enter the L2 cross-trade chain ID: ")
				l2ChainID, err = scanner.ScanInt()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 cross-trade chain ID: %w", err)
				}
				if l2ChainID == 0 {
					fmt.Println("L2 cross-trade chain ID cannot be empty")
					continue
				}
				break
			}

			for {
				fmt.Print("Please enter the L2 cross-trade proxy address: ")
				l2CrossTradeProxyAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 cross-trade proxy address: %w", err)
				}
				if l2CrossTradeProxyAddress == "" {
					fmt.Println("L2 cross-trade proxy address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l2CrossTradeProxyAddress) {
					fmt.Println("Invalid L1 cross-trade proxy address")
					continue
				}
				break
			}

			for {
				fmt.Print("Please enter the L2 cross-trade address: ")
				l2CrossTradeAddress, err = scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read L2 cross-trade address: %w", err)
				}
				if l2CrossTradeAddress == "" {
					fmt.Println("L2 cross-trade address cannot be empty")
					continue
				}
				if !common.IsHexAddress(l2CrossTradeAddress) {
					fmt.Println("Invalid L1 cross-trade address")
					continue
				}
				break
			}

			// Get token inputs from user
			l1Tokens, l2Tokens, err := getTokenInputsFromUser()
			if err != nil {
				return nil, fmt.Errorf("failed to get token inputs: %w", err)
			}

			// Get L1 bridge configuration from user
			nativeTokenAddressOnL1, l1StandardBridgeAddress, l1USDCBridgeAddress, l1CrossDomainMessenger, err := getL1BridgeConfigFromUser()
			if err != nil {
				return nil, fmt.Errorf("failed to get L1 bridge configuration: %w", err)
			}

			l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
				RPC:                     l2RPC,
				ChainID:                 uint64(l2ChainID),
				ContractName:            l2ContractFileName,
				PrivateKey:              "0x" + privateKey,
				IsDeployedNew:           false,
				BlockExplorerConfig:     l2BlockExplorerConfig,
				L1CrossDomainMessenger:  l1CrossDomainMessenger,
				CrossDomainMessenger:    constants.L2CrossDomainMessenger,
				CrossTradeProxyAddress:  l2CrossTradeProxyAddress,
				CrossTradeAddress:       l2CrossTradeAddress,
				DeploymentScriptPath:    deploymentScriptPath,
				L1Tokens:                l1Tokens,
				L2Tokens:                l2Tokens,
				NativeTokenAddressOnL1:  nativeTokenAddressOnL1,
				L1StandardBridgeAddress: l1StandardBridgeAddress,
				L1USDCBridgeAddress:     l1USDCBridgeAddress,
			})
		}
	}

	return &DeployCrossTradeInputs{
		Mode:          mode,
		L1ChainConfig: l1ChainConfig,
		L2ChainConfig: l2ChainConfigs,
	}, nil
}

func (t *ThanosStack) DeployCrossTrade(ctx context.Context, input *DeployCrossTradeInputs) (*DeployCrossTradeOutput, error) {
	deployCrossTradeContractsOutput, err := t.DeployCrossTradeContracts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy cross trade contracts: %s", err)
	}

	deployCrossTradeApplicationOutput, err := t.DeployCrossTradeApplication(ctx, input, deployCrossTradeContractsOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy cross trade application: %s", err)
	}

	return &DeployCrossTradeOutput{
		DeployCrossTradeContractsOutput:   deployCrossTradeContractsOutput,
		DeployCrossTradeApplicationOutput: deployCrossTradeApplicationOutput,
	}, nil
}

func (t *ThanosStack) DeployCrossTradeContracts(ctx context.Context, input *DeployCrossTradeInputs) (*DeployCrossTradeContractsOutput, error) {
	if input.L1ChainConfig == nil {
		return nil, fmt.Errorf("l1 chain config is required")
	}

	// Clone cross trade repository
	err := t.cloneSourcecode(ctx, "crossTrade", "https://github.com/tokamak-network/crossTrade.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone cross trade repository: %s", err)
	}

	// Set current working directory
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %s", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(t.deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to change directory to crossTrade: %s", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd crossTrade && git checkout L2toL2Implementation")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout L2toL2Implementation: %s", err)
	}

	t.logger.Info("Start to build cross-trade contracts")

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
		// PRIVATE_KEY=0X1233 forge script script/foundry_scripts/DeployL1CrossTrade.s.sol:DeployL1CrossTrade --rpc-url https://sepolia.infura.io/v3/1234567890 --broadcast --chain sepolia
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
		l1ContractAddresses, err = t.getContractAddressFromOutput(ctx, input.L1ChainConfig.ContractName, input.L1ChainConfig.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %s", err)
		}

		for contractName, address := range l1ContractAddresses {
			t.logger.Infof("L1 contract address %s with address %s", contractName, address)
			switch contractName {
			case L2L2CrossTradeProxyL1ContractName, L1L2CrossTradeProxyL1ContractName:
				l1CrossTradeProxyAddress = address
			case L2L2CrossTradeL1ContractName, L1L2CrossTradeL1ContractName:
				l1CrossTradeAddress = address
			default:
				t.logger.Infof("Unknown contract %s", contractName)
			}
		}
	} else {
		l1CrossTradeProxyAddress = input.L1ChainConfig.CrossTradeProxyAddress
		l1CrossTradeAddress = input.L1ChainConfig.CrossTradeAddress
	}
	// Verify the contracts
	//
	if input.L1ChainConfig.BlockExplorerConfig != nil {
		for contractName, address := range l1ContractAddresses {
			if address == "" {
				continue
			}
			t.logger.Infof("Verifying L1 contract %s with address %s", contractName, address)
			script := fmt.Sprintf(
				"cd crossTrade && forge verify-contract %s contracts/L1/%s.sol:%s --etherscan-api-key %s --chain %s",
				address,
				contractName,
				contractName,
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
			addresses, err := t.getContractAddressFromOutput(ctx, l2ChainConfig.ContractName, l2ChainConfig.ChainID)
			if err != nil {
				return nil, fmt.Errorf("failed to get the contract address: %s", err)
			}

			t.logger.Infof("Contract addresses: %v", addresses)

			for contractName, address := range addresses {
				t.logger.Infof("L2 contract address %s with address %s", contractName, address)
				switch contractName {
				case L2L2CrossTradeProxyL2ContractName:
					l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = address
				case L1L2CrossTradeProxyL2ContractName:
					l1l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = address
				case L2L2CrossTradeL2ContractName:
					l2l2CrossTradeAddresses[l2ChainConfig.ChainID] = address
				case L1L2CrossTradeL2ContractName:
					l1l2CrossTradeAddresses[l2ChainConfig.ChainID] = address
				default:
					t.logger.Infof("Unknown contract %s", contractName)
				}
			}

			// Verify the contracts
			//
			if l2ChainConfig.BlockExplorerConfig != nil {
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
				l1l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = l2ChainConfig.CrossTradeProxyAddress
				l1l2CrossTradeAddresses[l2ChainConfig.ChainID] = l2ChainConfig.CrossTradeAddress
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

	if input.Mode == constants.CrossTradeDeployModeL2ToL1 {
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

		// Register tokens
		for name, l2TokenAddress := range l2ChainConfig.L2Tokens {
			l1TokenAddress, ok := l2ChainConfig.L1Tokens[name]
			if !ok {
				continue
			}
			if l1TokenAddress == "" || l2TokenAddress == "" {
				continue
			}
			script := fmt.Sprintf(
				`cd crossTrade && PRIVATE_KEY=%s L2_CROSS_TRADE_PROXY=%s L1_TOKEN=%s L2_TOKEN=%s L1_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
				l2ChainConfig.PrivateKey,
				l1l2CrossTradeProxyAddresses[l2ChainID],
				l1TokenAddress,
				l2TokenAddress,
				input.L1ChainConfig.ChainID,
				"scripts/foundry_scripts/L2L1/RegisterToken_L2L1.sol:RegisterToken_L2L1",
				l2ChainConfig.RPC,
			)
			t.logger.Infof("Registering token %s on L1 %s", l2TokenAddress, script)
			err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
			if err != nil {
				return nil, fmt.Errorf("failed to deploy the contracts: %s", err)
			}

		}
	} else if input.Mode == constants.CrossTradeDeployModeL2ToL2 {
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

			// If the l2 chain is the current running chain, set USES_SIMPLIFIED_BRIDGE by false
			usesSimplifiedBridge := true
			if l2ChainConfig.ChainID == t.deployConfig.L2ChainID {
				usesSimplifiedBridge = false
			}
			script = fmt.Sprintf(
				`cd crossTrade && PRIVATE_KEY=%s L1_CROSS_TRADE_PROXY=%s L1_CROSS_DOMAIN_MESSENGER=%s L2_CROSS_TRADE_PROXY=%s L2_NATIVE_TOKEN_ADDRESS_ON_L1=%s L1_STANDARD_BRIDGE=%s L1_USDC_BRIDGE=%s L2_CHAIN_ID=%d USES_SIMPLIFIED_BRIDGE=%t forge script %s --rpc-url %s --broadcast`,
				input.L1ChainConfig.PrivateKey,
				l1CrossTradeProxyAddress,
				l2ChainConfig.L1CrossDomainMessenger,
				l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID],
				l2ChainConfig.NativeTokenAddressOnL1,
				l2ChainConfig.L1StandardBridgeAddress,
				l2ChainConfig.L1USDCBridgeAddress,
				l2ChainConfig.ChainID,
				usesSimplifiedBridge,
				"scripts/foundry_scripts/SetChainInfoL1_L2L2.sol:SetChainInfoL1_L2L2",
				input.L1ChainConfig.RPC,
			)
			t.logger.Infof("Setting chain information for L1 %s", script)
			err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
			if err != nil {
				return nil, fmt.Errorf("failed to set chain information for L1: %s", err)
			}

			// Get other L2 chain config
			otherL2ChainsConfig := make([]*L2CrossTradeChainInput, 0)
			for _, otherL2ChainConfig := range input.L2ChainConfig {
				if otherL2ChainConfig.ChainID != l2ChainConfig.ChainID {
					otherL2ChainsConfig = append(otherL2ChainsConfig, otherL2ChainConfig)
				}
			}

			for _, otherL2ChainConfig := range otherL2ChainsConfig {
				// Register tokens
				for name, l2TokenAddress := range l2ChainConfig.L2Tokens {
					l1TokenAddress, ok := l2ChainConfig.L1Tokens[name]
					if !ok {
						continue
					}

					if l1TokenAddress == "" || l2TokenAddress == "" {
						continue
					}

					if otherL2ChainConfig.L2Tokens[name] == "" {
						continue
					}

					script := fmt.Sprintf(
						`cd crossTrade && PRIVATE_KEY=%s L2_CROSS_TRADE_PROXY=%s L1_TOKEN=%s L2_SOURCE_TOKEN=%s L2_DESTINATION_TOKEN=%s L1_CHAIN_ID=%d L2_SOURCE_CHAIN_ID=%d L2_DESTINATION_CHAIN_ID=%d forge script %s --rpc-url %s --broadcast`,
						l2ChainConfig.PrivateKey,
						l2l2CrossTradeProxyAddresses[l2ChainConfig.ChainID],
						l1TokenAddress,
						l2TokenAddress,
						otherL2ChainConfig.L2Tokens[name],
						input.L1ChainConfig.ChainID,
						l2ChainConfig.ChainID,      // Source chain ID
						otherL2ChainConfig.ChainID, // Destination chain ID
						"scripts/foundry_scripts/RegisterToken_L2L2.sol:RegisterToken_L2L2",
						l2ChainConfig.RPC,
					)
					t.logger.Infof("Registering token %s on L2 %s", l2TokenAddress, script)
					err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", script)
					if err != nil {
						return nil, fmt.Errorf("failed to register token on L2: %s", err)
					}
				}
			}
		}
	}

	if input.Mode == constants.CrossTradeDeployModeL2ToL1 {
		return &DeployCrossTradeContractsOutput{
			Mode:                       input.Mode,
			L1CrossTradeProxyAddress:   l1CrossTradeProxyAddress,
			L1CrossTradeAddress:        l1CrossTradeAddress,
			L2CrossTradeProxyAddresses: l1l2CrossTradeProxyAddresses,
			L2CrossTradeAddresses:      l1l2CrossTradeAddresses,
		}, nil
	} else {
		return &DeployCrossTradeContractsOutput{
			Mode:                       input.Mode,
			L1CrossTradeProxyAddress:   l1CrossTradeProxyAddress,
			L1CrossTradeAddress:        l1CrossTradeAddress,
			L2CrossTradeProxyAddresses: l2l2CrossTradeProxyAddresses,
			L2CrossTradeAddresses:      l2l2CrossTradeAddresses,
		}, nil
	}
}

func (t *ThanosStack) DeployCrossTradeApplication(ctx context.Context, input *DeployCrossTradeInputs, contracts *DeployCrossTradeContractsOutput) (*DeployCrossTradeApplicationOutput, error) {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
		l1ChainID = t.deployConfig.L1ChainID
		mode      = strings.ReplaceAll(string(input.Mode), "_", "-")
	)

	crossTradePods, err := utils.GetPodsByName(ctx, namespace, fmt.Sprintf("cross-trade-%s", mode))
	if err != nil {
		t.logger.Error("Error to get cross trade pods", "err", err)
		return nil, err
	}
	if len(crossTradePods) > 0 {
		t.logger.Info("Cross trade is running: \n")
		var bridgeUrl string
		for {
			k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, fmt.Sprintf("cross-trade-%s", mode))
			if err != nil {
				t.logger.Error("Error retrieving ingress addresses", "err", err, "details", k8sIngresses)
				return nil, err
			}

			if len(k8sIngresses) > 0 {
				bridgeUrl = "http://" + k8sIngresses[0]
				break
			}

			time.Sleep(15 * time.Second)
		}
		return &DeployCrossTradeApplicationOutput{
			URL: bridgeUrl,
		}, nil
	}

	t.logger.Info("Installing a cross trade component...")

	// make yaml file at {cwd}/tokamak-thanos-stack/terraform/thanos-stack/cross-trade-values.yaml
	crossTradeConfig := types.CrossTradeConfig{}

	// Add L1 chain config
	crossTradeConfig.CrossTrade.Env.NextPublicProjectID = "568b8d3d0528e743b0e2c6c92f54d721"

	chainConfig := make(map[string]types.CrossTradeChainConfig)
	chainConfig[fmt.Sprintf("%d", l1ChainID)] = types.CrossTradeChainConfig{
		Name:        constants.L1ChainConfigurations[l1ChainID].ChainName,
		DisplayName: constants.L1ChainConfigurations[l1ChainID].ChainName,
		Contracts: types.CrossTradeContracts{
			L1CrossTrade: &contracts.L1CrossTradeProxyAddress,
		},
		RPCURL: input.L1ChainConfig.RPC,
		Tokens: types.CrossTradeTokens{
			ETH:  "0x0000000000000000000000000000000000000000",
			USDC: constants.L1ChainConfigurations[l1ChainID].USDCAddress,
			USDT: constants.L1ChainConfigurations[l1ChainID].USDTAddress,
			TON:  constants.L1ChainConfigurations[l1ChainID].TON,
		},
	}

	l2ChainRPCs := make(map[uint64]string)
	usdcAddresses := make(map[uint64]string)
	usdtAddresses := make(map[uint64]string)
	tonAddresses := make(map[uint64]string)
	ethAddresses := make(map[uint64]string)
	for _, l2ChainConfig := range input.L2ChainConfig {
		l2ChainRPCs[l2ChainConfig.ChainID] = l2ChainConfig.RPC
		usdcAddresses[l2ChainConfig.ChainID] = l2ChainConfig.L2Tokens["USDC"]
		usdtAddresses[l2ChainConfig.ChainID] = l2ChainConfig.L2Tokens["USDT"]
		tonAddresses[l2ChainConfig.ChainID] = l2ChainConfig.L2Tokens["TON"]
		ethAddresses[l2ChainConfig.ChainID] = l2ChainConfig.L2Tokens["ETH"]
	}

	// Add L2 chain config
	for l2ChainID, l2ChainConfig := range contracts.L2CrossTradeProxyAddresses {
		usdcAddress := constants.L2ChainConfigurations[l2ChainID].USDCAddress
		usdtAddress := constants.L2ChainConfigurations[l2ChainID].USDTAddress
		tonAddress := constants.L2ChainConfigurations[l2ChainID].TONAddress
		ethAddress := constants.L2ChainConfigurations[l2ChainID].ETHAddress
		nativeTokenName := constants.L2ChainConfigurations[l2ChainID].NativeTokenName
		nativeTokenSymbol := constants.L2ChainConfigurations[l2ChainID].NativeTokenSymbol

		if usdcAddresses[l2ChainID] != "" {
			usdcAddress = usdcAddresses[l2ChainID]
		}
		if usdtAddresses[l2ChainID] != "" {
			usdtAddress = usdtAddresses[l2ChainID]
		}
		if tonAddresses[l2ChainID] != "" {
			tonAddress = tonAddresses[l2ChainID]
		}
		if ethAddresses[l2ChainID] != "" {
			ethAddress = ethAddresses[l2ChainID]
		}

		if l2ChainID == t.deployConfig.L2ChainID {
			usdcAddress = constants.USDCAddress
			tonAddress = constants.TON
			ethAddress = constants.ETH
			nativeTokenName = "Tokamak Network Token"
			nativeTokenSymbol = "TON"
		}

		chainConfig[fmt.Sprintf("%d", l2ChainID)] = types.CrossTradeChainConfig{
			Name:        fmt.Sprintf("%d", l2ChainID),
			DisplayName: fmt.Sprintf("%d", l2ChainID),
			Contracts: types.CrossTradeContracts{
				L2CrossTrade: &l2ChainConfig,
			},
			RPCURL: l2ChainRPCs[l2ChainID],
			Tokens: types.CrossTradeTokens{
				ETH:  ethAddress,
				USDC: usdcAddress,
				USDT: usdtAddress,
				TON:  tonAddress,
			},
			NativeTokenName:   nativeTokenName,
			NativeTokenSymbol: nativeTokenSymbol,
		}
	}

	chainConfigJSON, err := json.Marshal(chainConfig)
	if err != nil {
		t.logger.Error("Error marshalling chain config", "err", err)
		return nil, err
	}

	switch input.Mode {
	case constants.CrossTradeDeployModeL2ToL1:
		crossTradeConfig.CrossTrade.Env.L2L1Config = string(chainConfigJSON)
	case constants.CrossTradeDeployModeL2ToL2:
		crossTradeConfig.CrossTrade.Env.L2L2Config = string(chainConfigJSON)
	}

	// input from users

	crossTradeConfig.CrossTrade.Ingress = types.Ingress{Enabled: true, ClassName: "alb", Annotations: map[string]string{
		"alb.ingress.kubernetes.io/target-type":  "ip",
		"alb.ingress.kubernetes.io/scheme":       "internet-facing",
		"alb.ingress.kubernetes.io/listen-ports": "[{\"HTTP\": 80}]",
		"alb.ingress.kubernetes.io/group.name":   fmt.Sprintf("cross-trade-%s", mode),
	}, TLS: types.TLS{
		Enabled: false,
	}}

	data, err := yaml.Marshal(&crossTradeConfig)
	if err != nil {
		t.logger.Error("Error marshalling cross-trade values YAML file", "err", err)
		return nil, err
	}

	configFileDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	if err := os.MkdirAll(configFileDir, os.ModePerm); err != nil {
		t.logger.Error("Error creating directory", "err", err)
		return nil, err
	}

	// Write to file
	filePath := filepath.Join(configFileDir, "/cross-trade-values.yaml")
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		t.logger.Error("Error writing file", "err", err)
		return nil, nil
	}

	helmReleaseName := fmt.Sprintf("cross-trade-%s", mode)
	_, err = utils.ExecuteCommand(ctx, "helm", []string{
		"install",
		helmReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/cross-trade", t.deploymentPath),
		"--values",
		filePath,
		"--namespace",
		namespace,
	}...)
	if err != nil {
		t.logger.Error("Error installing Helm charts", "err", err)
		return nil, err
	}

	t.logger.Info("✅ Cross trade component installed successfully and is being initialized. Please wait for the ingress address to become available...")
	var bridgeUrl string
	for {
		k8sIngresses, err := utils.GetAddressByIngress(ctx, namespace, helmReleaseName)
		if err != nil {
			t.logger.Error("Error retrieving ingress addresses", "err", err, "details", k8sIngresses)
			return nil, err
		}

		if len(k8sIngresses) > 0 {
			bridgeUrl = "http://" + k8sIngresses[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	t.logger.Infof("✅ Cross trade component is up and running. You can access it at: %s", bridgeUrl)

	return &DeployCrossTradeApplicationOutput{
		URL: bridgeUrl,
	}, nil
}

func (t *ThanosStack) UninstallCrossTrade(ctx context.Context, mode constants.CrossTradeDeployMode) error {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)
	modeString := strings.ReplaceAll(string(mode), "_", "-")

	if t.deployConfig.AWS == nil {
		t.logger.Error("AWS configuration is not set. Please run the deploy command first")
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	releases, err := utils.FilterHelmReleases(ctx, namespace, fmt.Sprintf("cross-trade-%s", modeString))
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

	return nil
}

func (t *ThanosStack) getContractAddressFromOutput(_ context.Context, deployFile string, chainID uint64) (map[string]string, error) {
	// Construct the file path
	filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", t.deploymentPath, deployFile, chainID)

	// Open and read the file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to open deployment file %s: %w", filePath, err)
	}
	defer file.Close()

	// Parse the JSON structure
	var deploymentData map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&deploymentData); err != nil {
		return nil, fmt.Errorf("failed to decode deployment JSON: %w", err)
	}

	// Extract the transactions array
	transactions, ok := deploymentData["transactions"].([]any)
	if !ok {
		return nil, fmt.Errorf("transactions field not found or not an array in deployment file")
	}

	// Collect all contract addresses from CREATE transactions
	contractAddresses := make(map[string]string)

	// Loop through transactions to find CREATE type
	for _, tx := range transactions {
		txMap, ok := tx.(map[string]any)
		if !ok {
			continue
		}

		// Check if transaction type is CREATE
		txType, ok := txMap["transactionType"].(string)
		if !ok || txType != "CREATE" {
			continue
		}

		// Extract contract address
		contractAddress, ok := txMap["contractAddress"].(string)
		if !ok || contractAddress == "" {
			continue
		}

		contractAddresses[txMap["contractName"].(string)] = contractAddress
	}

	if len(contractAddresses) == 0 {
		return nil, fmt.Errorf("no CREATE transaction found in deployment file")
	}

	return contractAddresses, nil
}
