package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
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
	RPC                    string               `json:"rpc"`
	ChainID                uint64               `json:"chain_id"`
	PrivateKey             string               `json:"private_key"`
	IsDeployedNew          bool                 `json:"is_deployed_new"`
	DeploymentScriptPath   string               `json:"deployment_script_path"`
	ContractName           string               `json:"contract_name"`
	BlockExplorerConfig    *BlockExplorerConfig `json:"block_explorer_config"`
	CrossDomainMessenger   string               `json:"cross_domain_messenger"`
	CrossTradeProxyAddress string               `json:"cross_trade_proxy_address"`
	CrossTradeAddress      string               `json:"cross_trade_address"`
}

type DeployCrossTradeContractsInputs struct {
	Mode          constants.CrossTradeDeployMode `json:"mode"`
	L1ChainConfig *L1CrossTradeChainInput        `json:"l1_chain_config"`
	L2ChainConfig []*L2CrossTradeChainInput      `json:"l2_chain_config"`
}

type DeployCrossTradeContractsOutput struct {
	Mode                       constants.CrossTradeDeployMode `json:"mode"`
	L1CrossTradeProxyAddress   string                         `json:"l1_cross_trade_proxy_address"`
	L1CrossTradeAddress        string                         `json:"l1_cross_trade_address"`
	L2CrossTradeProxyAddresses map[uint64][]string            `json:"l2_cross_trade_proxy_addresses"`
	L2CrossTradeAddresses      map[uint64][]string            `json:"l2_cross_trade_addresses"`
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

func (t *ThanosStack) GetCrossTradeContractsInputs(ctx context.Context, mode constants.CrossTradeDeployMode) (*DeployCrossTradeContractsInputs, error) {
	// Step 1: Ask the user want to deploy new contracts for L1
	// If yes, ask to input the private key and RPC URL
	// Get chain id from this rpc
	// if no, set IsDeployNew to false and ask to input the cross trade proxy address

	// Same as L2s, but we have multiple l2 chains

	var (
		l1ContractFileName, l2ContractFileName  string
		l1CrossTradeProxyName, l1CrossTradeName string
		l2CrossTradeProxyName, l2CrossTradeName string
		deploymentScriptPath                    string
	)

	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		l1ContractFileName = DeployL1CrossTradeL2L1
		l2ContractFileName = DeployL2CrossTradeL2L1
		l1CrossTradeProxyName = L1L2CrossTradeProxyL1ContractName
		l1CrossTradeName = L1L2CrossTradeL1ContractName
		l2CrossTradeProxyName = L1L2CrossTradeProxyL2ContractName
		l2CrossTradeName = L1L2CrossTradeL2ContractName
		deploymentScriptPath = L1L2ScriptPath
	case constants.CrossTradeDeployModeL2ToL2:
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

	fmt.Println("=== L1 Chain Configuration ===")

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

	if deployNewL1 {
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
		}
	}

	l2ChainConfigs := make([]*L2CrossTradeChainInput, 0)
	// Get current running L2 chain
	l2RPC := t.deployConfig.L2RpcUrl
	l2ChainID := t.deployConfig.L2ChainID

	fmt.Println("=== Your L2 Chain Configuration ===")
	fmt.Print("Do you want to deploy the L2 cross-trade contracts to the current L2 chain? (Y/n): ")
	deployNewL2, err := scanner.ScanBool(true)
	if err != nil {
		return nil, fmt.Errorf("failed to read L2 deployment option: %w", err)
	}

	var privateKey string
	if deployNewL2 {
		fmt.Print("Please enter the private key: ")
		privateKey, err = scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("failed to read L2 private key: %w", err)
		}
		if privateKey == "" {
			return nil, fmt.Errorf("L2 private key cannot be empty")
		}

		if !strings.HasPrefix(privateKey, "0x") {
			privateKey = "0x" + privateKey
		}

		// Get block explorer URL
		var l2BlockExplorerConfig *BlockExplorerConfig
		l2BlockExplorerURL, err := t.GetBlockExplorerURL(ctx)
		if err != nil {
			t.l.Warnf("No block explorer URL found, skip verifying L2 cross-trade contracts")
		} else {
			l2BlockExplorerConfig = &BlockExplorerConfig{
				URL:  l2BlockExplorerURL,
				Type: constants.BlockExplorerTypeBlockscout,
			}
		}

		l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
			RPC:                  l2RPC,
			ChainID:              l2ChainID,
			ContractName:         l2ContractFileName,
			PrivateKey:           privateKey,
			IsDeployedNew:        deployNewL2,
			BlockExplorerConfig:  l2BlockExplorerConfig,
			CrossDomainMessenger: constants.L2CrossDomainMessenger,
			DeploymentScriptPath: deploymentScriptPath,
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

		var l2BlockExplorerConfig *BlockExplorerConfig
		l2BlockExplorerURL, err := t.GetBlockExplorerURL(ctx)
		if err != nil {
			t.l.Warnf("No block explorer URL found, skip verifying L2 cross-trade contracts")
		} else {
			l2BlockExplorerConfig = &BlockExplorerConfig{
				URL:  l2BlockExplorerURL,
				Type: constants.BlockExplorerTypeBlockscout,
			}
		}

		// Read for the deployment file
		l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
			RPC:                    l2RPC,
			ChainID:                l2ChainID,
			ContractName:           l2ContractFileName,
			PrivateKey:             "0x" + privateKey,
			IsDeployedNew:          false,
			BlockExplorerConfig:    l2BlockExplorerConfig,
			CrossDomainMessenger:   constants.L2CrossDomainMessenger,
			CrossTradeProxyAddress: l2CrossTradeProxyAddress,
			CrossTradeAddress:      l2CrossTradeAddress,
			DeploymentScriptPath:   deploymentScriptPath,
		})
	}

	if mode == constants.CrossTradeDeployModeL2ToL2 {
		fmt.Println("=== Other L2 Chain Configuration ===")
		fmt.Print("Do you want to deploy contracts to other L2 chain? (Y/n): ")
		addOtherL2, err := scanner.ScanBool(true)
		if err != nil {
			return nil, fmt.Errorf("failed to read other L2 chain option: %w", err)
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

			fmt.Print("Please enter the private key: ")
			otherPrivateKey, err := scanner.ScanString()
			if err != nil {
				return nil, fmt.Errorf("failed to read private key: %w", err)
			}
			if otherPrivateKey == "" {
				return nil, fmt.Errorf("private key cannot be empty")
			}

			if !strings.HasPrefix(otherPrivateKey, "0x") {
				otherPrivateKey = "0x" + otherPrivateKey
			}

			otherChainID, err = l2RpcClient.ChainID(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get chain ID: %w", err)
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

			l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
				RPC:                  otherRpc,
				ChainID:              otherChainID.Uint64(),
				ContractName:         l2ContractFileName,
				PrivateKey:           otherPrivateKey,
				IsDeployedNew:        addOtherL2,
				BlockExplorerConfig:  otherL2BlockExplorerConfig,
				CrossDomainMessenger: constants.L2CrossDomainMessenger,
				DeploymentScriptPath: deploymentScriptPath,
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

			var l2BlockExplorerConfig *BlockExplorerConfig
			fmt.Print("Do you want to verify the L2 cross-trade contracts? (Y/n): ")
			verifyL2, err := scanner.ScanBool(true)
			if err != nil {
				return nil, fmt.Errorf("failed to read L2 verification option: %w", err)
			}

			if verifyL2 {
				fmt.Print("Please enter the Etherscan API key: ")
				l2EtherscanAPIKey, err := scanner.ScanString()
				if err != nil {
					return nil, fmt.Errorf("failed to read Etherscan API key: %w", err)
				}
				l2BlockExplorerConfig = &BlockExplorerConfig{
					APIKey: l2EtherscanAPIKey,
					Type:   constants.BlockExplorerTypeEtherscan,
				}
			}

			l2ChainConfigs = append(l2ChainConfigs, &L2CrossTradeChainInput{
				RPC:                    l2RPC,
				ChainID:                uint64(l2ChainID),
				ContractName:           l2ContractFileName,
				PrivateKey:             "0x" + privateKey,
				IsDeployedNew:          false,
				BlockExplorerConfig:    l2BlockExplorerConfig,
				CrossDomainMessenger:   constants.L2CrossDomainMessenger,
				CrossTradeProxyAddress: l2CrossTradeProxyAddress,
				CrossTradeAddress:      l2CrossTradeAddress,
				DeploymentScriptPath:   deploymentScriptPath,
			})
		}
	}

	return &DeployCrossTradeContractsInputs{
		Mode:          mode,
		L1ChainConfig: l1ChainConfig,
		L2ChainConfig: l2ChainConfigs,
	}, nil
}

func (t *ThanosStack) DeployCrossTradeContracts(ctx context.Context, input *DeployCrossTradeContractsInputs) (*DeployCrossTradeContractsOutput, error) {
	if input.L1ChainConfig == nil {
		return nil, fmt.Errorf("l1 chain config is required")
	}

	// Clone cross trade repository
	err := t.cloneSourcecode(ctx, "crossTrade", "https://github.com/tokamak-network/crossTrade.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone cross trade repository: %s", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", "cd crossTrade && git checkout L2toL2Implementation")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout L2toL2Implementation: %s", err)
	}

	t.l.Info("Start to build cross-trade contracts")

	// Build the contracts
	err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", "cd crossTrade && pnpm install && forge clean && forge build")
	if err != nil {
		return nil, fmt.Errorf("failed to build the contracts: %s", err)
	}

	// Step 1: Check if the L1 contracts are deployed
	var (
		l1CrossTradeProxyAddress   string
		l1CrossTradeAddress        string
		l1ContractAddresses        = make(map[string]string)
		l2CrossTradeProxyAddresses = make(map[uint64][]string)
		l2CrossTradeAddresses      = make(map[uint64][]string)
	)
	if input.L1ChainConfig.IsDeployedNew {
		t.l.Info("L1 contracts are not deployed. Deploying new L1 contracts")
		// PRIVATE_KEY=0X1233 forge script script/foundry_scripts/DeployL1CrossTrade.s.sol:DeployL1CrossTrade --rpc-url https://sepolia.infura.io/v3/1234567890 --broadcast --chain sepolia
		script := fmt.Sprintf(
			"cd crossTrade && PRIVATE_KEY=%s forge script %s/%s --rpc-url %s --broadcast --chain %s",
			input.L1ChainConfig.PrivateKey,
			input.L1ChainConfig.DeploymentScriptPath,
			input.L1ChainConfig.ContractName,
			input.L1ChainConfig.RPC,
			"sepolia",
		)
		t.l.Infof("Deploying L1 contracts %s", script)
		err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", script)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy the contracts: %s", err)
		}
		// Get the contract address from the output
		l1ContractAddresses, err = t.getContractAddressFromOutput(ctx, input.L1ChainConfig.ContractName, input.L1ChainConfig.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %s", err)
		}

		for contractName, address := range l1ContractAddresses {
			t.l.Infof("L1 contract address %s with address %s", contractName, address)
			switch contractName {
			case L2L2CrossTradeProxyL1ContractName, L1L2CrossTradeProxyL1ContractName:
				l1CrossTradeProxyAddress = address
			case L2L2CrossTradeL1ContractName, L1L2CrossTradeL1ContractName:
				l1CrossTradeAddress = address
			default:
				t.l.Infof("Unknown contract %s", contractName)
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
			t.l.Infof("Verifying L1 contract %s with address %s", contractName, address)
			script := fmt.Sprintf(
				"cd crossTrade && forge verify-contract %s contracts/L1/%s.sol:%s --etherscan-api-key %s --chain %s",
				address,
				contractName,
				contractName,
				input.L1ChainConfig.BlockExplorerConfig.APIKey,
				constants.ChainIDToForgeChainName[input.L1ChainConfig.ChainID],
			)
			t.l.Infof("Verifying L1 contract %s", script)
			err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", script)
			if err != nil {
				// Skip if the contract is not verified
				t.l.Errorf("failed to verify the contracts: %s", err)
				continue
			}
			t.l.Infof("Verified L1 contract %s with address %s", contractName, address)
		}
	}

	for _, l2ChainConfig := range input.L2ChainConfig {
		if !l2ChainConfig.IsDeployedNew {
			continue
		}

		script := fmt.Sprintf(
			`cd crossTrade && PRIVATE_KEY=%s CHAIN_ID=%d NATIVE_TOKEN=%s CROSS_DOMAIN_MESSENGER=%s L1_CROSS_TRADE=%s forge script %s/%s --rpc-url %s --broadcast`,
			l2ChainConfig.PrivateKey,
			l2ChainConfig.ChainID,
			constants.NativeToken,
			l2ChainConfig.CrossDomainMessenger,
			l1CrossTradeProxyAddress,
			l2ChainConfig.DeploymentScriptPath,
			l2ChainConfig.ContractName,
			l2ChainConfig.RPC,
		)
		t.l.Infof("Deploying L2 contracts %s", script)
		err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", script)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy the contracts: %s", err)
		}

		// Get the contract address from the output
		addresses, err := t.getContractAddressFromOutput(ctx, l2ChainConfig.ContractName, l2ChainConfig.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get the contract address: %s", err)
		}

		for contractName, address := range addresses {
			t.l.Infof("L2 contract address %s with address %s", contractName, address)
			switch contractName {
			case L2L2CrossTradeProxyL2ContractName, L1L2CrossTradeProxyL2ContractName:
				l2CrossTradeProxyAddresses[l2ChainConfig.ChainID] = append(l2CrossTradeProxyAddresses[l2ChainConfig.ChainID], address)
			case L2L2CrossTradeL2ContractName, L1L2CrossTradeL2ContractName:
				l2CrossTradeAddresses[l2ChainConfig.ChainID] = append(l2CrossTradeAddresses[l2ChainConfig.ChainID], address)
			default:
				t.l.Infof("Unknown contract %s", contractName)
			}
		}

		// Verify the contracts
		//
		if l2ChainConfig.BlockExplorerConfig != nil {
			for contractName, address := range addresses {
				t.l.Infof("Verifying L2 contract %s with address %s", contractName, address)
				if l2ChainConfig.BlockExplorerConfig.Type == constants.BlockExplorerTypeEtherscan {
					script = fmt.Sprintf(
						"cd crossTrade && forge verify-contract %s contracts/L2/%s.sol:%s --etherscan-api-key %s --chain %s",
						address,
						contractName,
						contractName,
						l2ChainConfig.BlockExplorerConfig.APIKey,
						constants.ChainIDToForgeChainName[l2ChainConfig.ChainID],
					)
					t.l.Infof("Verifying L2 contract %s", script)
					err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", script)
					if err != nil {
						t.l.Errorf("failed to verify the contracts: %s", err)
						continue
					}
					t.l.Infof("Verified L2 contract %s with address %s", contractName, address)
				} else if l2ChainConfig.BlockExplorerConfig.Type == constants.BlockExplorerTypeBlockscout {
					script = fmt.Sprintf(
						"cd crossTrade && forge verify-contract --rpc-url %s %s contracts/L2/%s.sol:%s --verifier blockscout --verifier-url %s/api",
						l2ChainConfig.RPC,
						address,
						contractName,
						contractName,
						l2ChainConfig.BlockExplorerConfig.URL,
					)
					t.l.Infof("Verifying L2 contract %s", script)
					err = utils.ExecuteCommandStream(ctx, t.l, "bash", "-c", script)
					if err != nil {
						t.l.Errorf("failed to verify the contracts: %s", err)
						continue
					}
					t.l.Infof("Verified L2 contract %s with address %s", contractName, address)
				}
			}
		}

	}

	t.l.Infof("L1 cross trade proxy address %s", l1CrossTradeProxyAddress)
	t.l.Infof("L1 cross trade address %s", l1CrossTradeAddress)
	t.l.Infof("L2 cross trade proxy addresses %v", l2CrossTradeProxyAddresses)
	t.l.Infof("L2 cross trade addresses %v", l2CrossTradeAddresses)

	return &DeployCrossTradeContractsOutput{
		Mode:                       input.Mode,
		L1CrossTradeProxyAddress:   l1CrossTradeProxyAddress,
		L1CrossTradeAddress:        l1CrossTradeAddress,
		L2CrossTradeProxyAddresses: l2CrossTradeProxyAddresses,
		L2CrossTradeAddresses:      l2CrossTradeAddresses,
	}, nil
}

func (t *ThanosStack) getContractAddressFromOutput(_ context.Context, deployFile string, chainID uint64) (map[string]string, error) {
	// Construct the file path
	filePath := fmt.Sprintf("%s/crossTrade/broadcast/%s/%d/run-latest.json", t.deploymentPath, deployFile, chainID)

	// Open and read the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open deployment file %s: %w", filePath, err)
	}
	defer file.Close()

	// Parse the JSON structure
	var deploymentData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&deploymentData); err != nil {
		return nil, fmt.Errorf("failed to decode deployment JSON: %w", err)
	}

	// Extract the transactions array
	transactions, ok := deploymentData["transactions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("transactions field not found or not an array in deployment file")
	}

	// Collect all contract addresses from CREATE transactions
	contractAddresses := make(map[string]string)

	// Loop through transactions to find CREATE type
	for _, tx := range transactions {
		txMap, ok := tx.(map[string]interface{})
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
