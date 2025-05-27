package constants

var L1ChainConfigurations = map[uint64]struct {
	BlockTimeInSeconds   uint64 `json:"block_time_in_seconds"`
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
	L1VerificationContractAddress    string `json:"l1_verification_contract_address"`
	L2TonAddress                     string `json:"l2_ton_address"`
	L2ManagerAddress                 string `json:"l2_manager_address"`
	L1BridgeRegistry                 string `json:"l1_bridge_registry"`
	TON                              string `json:"ton"`
	MaxChannelDuration               uint64 `json:"max_channel_duration"`
}{
	//TODO: Updated the addresses for L1VerificationContractAddress, L2TonAddress, L2ManagerAddress and L1BridgeRegistry for different chains
	1: {
		BlockTimeInSeconds:   12,
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
		L1VerificationContractAddress:    "0x0000000000000000000000000000000000000000",
		L2TonAddress:                     "0x0000000000000000000000000000000000000000",
		L2ManagerAddress:                 "0x0000000000000000000000000000000000000000",
		L1BridgeRegistry:                 "0x0000000000000000000000000000000000000000",
		TON:                              "0x0000000000000000000000000000000000000000",
		MaxChannelDuration:               1500,
	},
	11155111: {
		BlockTimeInSeconds:   12,
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
		L1VerificationContractAddress:    "0xe18a97CD99056A790E5153d554C58a32c5D596Ce",
		L2TonAddress:                     "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		L2ManagerAddress:                 "0x58B4C2FEf19f5CDdd944AadD8DC99cCC71bfeFDc",
		L1BridgeRegistry:                 "0x2D47fa57101203855b336e9E61BC9da0A6dd0Dbc",
		TON:                              "0xa30fe40285B8f5c0457DbC3B7C8A280373c40044",
		MaxChannelDuration:               120,
	},
	17000: {
		BlockTimeInSeconds:   12,
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
		L1VerificationContractAddress:    "0x0000000000000000000000000000000000000000",
		L2TonAddress:                     "0x0000000000000000000000000000000000000000",
		L2ManagerAddress:                 "0x0000000000000000000000000000000000000000",
		L1BridgeRegistry:                 "0x0000000000000000000000000000000000000000",
		TON:                              "0x0000000000000000000000000000000000000000",
		MaxChannelDuration:               120,
	},
}

var BaseBatchInboxAddress = "0xff00000000000000000000000000000000000000"

var DefaultL2BlockTimeInSeconds uint64 = 2
