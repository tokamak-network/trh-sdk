package types

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

type CrossTradeChainConfig struct {
	Name        string              `yaml:"name" json:"name"`
	DisplayName string              `yaml:"displayName" json:"displayName"`
	Contracts   CrossTradeContracts `yaml:"contracts" json:"contracts"`
	Tokens      CrossTradeTokens    `yaml:"tokens" json:"tokens"`
	RPCURL      string              `yaml:"rpcURL" json:"rpcURL"`
}

type CrossTradeConfigs map[string]CrossTradeChainConfig

type CrossTradeEnvConfig struct {
	NextPublicProjectID   string `yaml:"NEXT_PUBLIC_PROJECT_ID" json:"NEXT_PUBLIC_PROJECT_ID"`
	NextPublicChainConfig string `yaml:"NEXT_PUBLIC_CHAIN_CONFIG" json:"NEXT_PUBLIC_CHAIN_CONFIG"`
}
type CrossTradeConfig struct {
	CrossTrade struct {
		Ingress Ingress             `yaml:"ingress" json:"ingress"`
		Env     CrossTradeEnvConfig `yaml:"env" json:"env"`
	} `yaml:"cross_trade"`
}
