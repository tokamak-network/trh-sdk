package constants

var L1ChainConfigurations = map[uint64]struct {
	L2NativeTokenAddress string `json:"native_token_address"`

	NativeTokenSymbol   string `json:"native_token_symbol"`
	NativeTokenDecimals int    `json:"native_token_decimals"`
	NativeTokenName     string `json:"native_token_name"`

	FinalizationPeriodSeconds        uint64 `json:"finalization_period_seconds"`
	L2OutputOracleSubmissionInterval uint64 `json:"l2_output_oracle_submission_interval"`
	USDCAddress                      string `json:"usdc_address"`
	USDTAddress                      string `json:"usdt_address"`
	ChainName                        string `json:"chain_name"`
	BlockExplorer                    string `json:"block_explorer"`
}{
	1: {
		L2NativeTokenAddress: "0x2be5e8c109e2197D077D13A82dAead6a9b3433C5",
		NativeTokenSymbol:    "ETH",
		NativeTokenDecimals:  18,
		NativeTokenName:      "Ether",

		FinalizationPeriodSeconds:        604800,
		L2OutputOracleSubmissionInterval: 10800,
		USDCAddress:                      "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		USDTAddress:                      "0xdac17f958d2ee523a2206206994597c13d831ec7",
		ChainName:                        "Ethereum Mainnet",
		BlockExplorer:                    "https://etherscan.io",
	},
	11155111: {
		L2NativeTokenAddress: "0xa30fe40285b8f5c0457dbc3b7c8a280373c40044",

		NativeTokenSymbol:   "ETH",
		NativeTokenDecimals: 18,
		NativeTokenName:     "Ether",

		FinalizationPeriodSeconds:        12,
		L2OutputOracleSubmissionInterval: 120,
		USDCAddress:                      "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238",
		USDTAddress:                      "0xaa8e23fb1079ea71e0a56f48a2aa51851d8433d0",
		ChainName:                        "Ethereum Sepolia",
		BlockExplorer:                    "https://sepolia.etherscan.io/",
	},
	17000: {
		L2NativeTokenAddress: "0xe11Ad6B761D175042340a784640d3A6e373E52A5",

		NativeTokenSymbol:   "ETH",
		NativeTokenDecimals: 18,
		NativeTokenName:     "Ether",

		FinalizationPeriodSeconds:        12,
		L2OutputOracleSubmissionInterval: 120,
		USDCAddress:                      "0xd718826bbc28e61dc93aacae04711c8e755b4915",
		USDTAddress:                      "0xD6e9Cd5ef382b0830653d1b2007D5Ca6987FaA26", // use USDT from morph: https://docs.morphl2.io/docs/quick-start/faucet/#erc20-usdt
		ChainName:                        "Ethereum Holesky",
		BlockExplorer:                    "https://holesky.etherscan.io/",
	},
}

var BaseBatchInboxAddress = "0xff00000000000000000000000000000000000000"
