package types

import "github.com/tokamak-network/trh-sdk/pkg/constants"

type CrossTradeContracts struct {
	L2CrossTrade *string `yaml:"l2_cross_trade,omitempty" json:"l2_cross_trade,omitempty"`
	L1CrossTrade *string `yaml:"l1_cross_trade,omitempty" json:"l1_cross_trade,omitempty"`
}

type CrossTradeTokens struct {
	ETH  string `yaml:"eth" json:"eth"`
	USDC string `yaml:"usdc" json:"usdc"`
	USDT string `yaml:"usdt" json:"usdt"`
	TON  string `yaml:"ton" json:"ton"`
}

type RegisterToken struct {
	Name              string   `json:"name"`
	Address           string   `json:"address"`
	DestinationChains []uint64 `json:"destination_chains"`
}

type CrossTradeChainConfig struct {
	Name              string              `yaml:"name" json:"name"`
	DisplayName       string              `yaml:"display_name" json:"display_name"`
	Contracts         CrossTradeContracts `yaml:"contracts" json:"contracts"`
	Tokens            []*RegisterToken    `yaml:"tokens" json:"tokens"`
	RPCURL            string              `yaml:"rpc_url" json:"rpc_url"`
	NativeTokenName   string              `yaml:"native_token_name" json:"native_token_name"`
	NativeTokenSymbol string              `yaml:"native_token_symbol" json:"native_token_symbol"`
}

type CrossTradeConfigs map[string]CrossTradeChainConfig

type CrossTradeEnvConfig struct {
	NextPublicProjectID string `yaml:"NEXT_PUBLIC_PROJECT_ID" json:"NEXT_PUBLIC_PROJECT_ID"`
	L2L1Config          string `yaml:"NEXT_PUBLIC_CHAIN_CONFIG_L2_L1" json:"NEXT_PUBLIC_CHAIN_CONFIG_L2_L1"`
	L2L2Config          string `yaml:"NEXT_PUBLIC_CHAIN_CONFIG_L2_L2" json:"NEXT_PUBLIC_CHAIN_CONFIG_L2_L2"`
}
type CrossTradeConfig struct {
	CrossTrade struct {
		Ingress Ingress             `yaml:"ingress" json:"ingress"`
		Env     CrossTradeEnvConfig `yaml:"env" json:"env"`
	} `yaml:"cross_trade"`
}

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
	BlockExplorerConfig    *BlockExplorerConfig `json:"block_explorer_config,omitempty"`
	CrossTradeProxyAddress string               `json:"cross_trade_proxy_address,omitempty"`
	CrossTradeAddress      string               `json:"cross_trade_address,omitempty"`
	ChainName              string               `json:"chain_name"`
}

type L2CrossTradeChainInput struct {
	RPC                     string               `json:"rpc"`
	ChainID                 uint64               `json:"chain_id"`
	PrivateKey              string               `json:"private_key"`
	IsDeployedNew           bool                 `json:"is_deployed_new"`
	ChainName               string               `json:"chain_name"`
	BlockExplorerConfig     *BlockExplorerConfig `json:"block_explorer_config"`
	CrossDomainMessenger    string               `json:"cross_domain_messenger"`
	DeploymentScriptPath    string               `json:"deployment_script_path"`
	ContractName            string               `json:"contract_name"`
	CrossTradeProxyAddress  string               `json:"cross_trade_proxy_address,omitempty"`
	CrossTradeAddress       string               `json:"cross_trade_address,omitempty"`
	NativeTokenAddressOnL1  string               `json:"native_token_address"`
	L1StandardBridgeAddress string               `json:"l1_standard_bridge_address"`
	L1USDCBridgeAddress     string               `json:"l1_usdc_bridge_address"`
	L1CrossDomainMessenger  string               `json:"l1_cross_domain_messenger"`
}

type L2TokenInput struct {
	ChainID                    uint64 `json:"chain_id"`
	TokenAddress               string `json:"l2_token_address"`
	RPC                        string `json:"rpc"`
	PrivateKey                 string `json:"private_key"`
	L1L2CrossTradeProxyAddress string `json:"l1_l2_cross_trade_proxy_address,omitempty"`
	L2L2CrossTradeProxyAddress string `json:"l2_l2_cross_trade_proxy_address,omitempty"`
}

type RegisterTokenInput struct {
	TokenName      string          `json:"token_name"`
	L1TokenAddress string          `json:"l1_token_address"`
	L2TokenInputs  []*L2TokenInput `json:"l2_token_inputs"`
}

type L1ContractAddressConfig struct {
	NativeTokenAddress            string `json:"native_token_address"`
	L1StandardBridgeAddress       string `json:"l1_standard_bridge_address"`
	L1USDCBridgeAddress           string `json:"l1_usdc_bridge_address"`
	L1CrossDomainMessengerAddress string `json:"l1_cross_domain_messenger_address"`
}

type DeployCrossTradeContractsOutput struct {
	Mode                       constants.CrossTradeDeployMode `json:"mode"`
	L1CrossTradeProxyAddress   string                         `json:"l1_cross_trade_proxy_address"`
	L1CrossTradeAddress        string                         `json:"l1_cross_trade_address"`
	L2CrossTradeProxyAddresses map[uint64]string              `json:"l2_cross_trade_proxy_addresses"`
	L2CrossTradeAddresses      map[uint64]string              `json:"l2_cross_trade_addresses"`
}

type DeployCrossTradeApplicationOutput struct {
	URL string `json:"url"`
}

type DeployCrossTradeOutput struct {
	DeployCrossTradeContractsOutput   *DeployCrossTradeContractsOutput   `json:"deploy_cross_trade_contracts_output,omitempty"`
	DeployCrossTradeApplicationOutput *DeployCrossTradeApplicationOutput `json:"deploy_cross_trade_application_output,omitempty"`
	RegisterTokens                    []*RegisterTokenInput              `json:"register_tokens,omitempty"`
}

type CrossTrade struct {
	Mode           constants.CrossTradeDeployMode `json:"mode"`
	ProjectID      string                         `json:"project_id"`
	L1ChainConfig  *L1CrossTradeChainInput        `json:"l1_chain_config"`
	L2ChainConfig  []*L2CrossTradeChainInput      `json:"l2_chain_config"`
	RegisterTokens []*RegisterTokenInput          `json:"register_tokens,omitempty"`
	Output         *DeployCrossTradeOutput        `json:"output,omitempty"`
}
