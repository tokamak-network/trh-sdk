package constants

const EthereumMainnetChainID uint64 = 1
const EthereumSepoliaChainID uint64 = 11155111
const EthereumHoleskyChainID uint64 = 17000

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
	L2ManagerAddress                 string `json:"l2_manager_address"`
	L1BridgeRegistry                 string `json:"l1_bridge_registry"`
	TON                              string `json:"ton"`
	MaxChannelDuration               uint64 `json:"max_channel_duration"`
	StakingURL                       string `json:"staking_url"`
	TxmgrCellProofTime               uint64 `json:"txmgr_cell_proof_time"`
	NextPublicRollupL1BaseUrl        string `json:"next_public_rollup_l1_base_url"`
}{
	//TODO: Updated the addresses for L1VerificationContractAddress, L2TonAddress, L2ManagerAddress and L1BridgeRegistry for different chains
	EthereumMainnetChainID: {
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
		L2ManagerAddress:                 "0x0000000000000000000000000000000000000000",
		L1BridgeRegistry:                 "0x0000000000000000000000000000000000000000",
		TON:                              "0x0000000000000000000000000000000000000000",
		MaxChannelDuration:               1500,
		TxmgrCellProofTime:               1764798551,
		NextPublicRollupL1BaseUrl:        "https://eth.blockscout.com",
	},
	EthereumSepoliaChainID: {
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
		L2ManagerAddress:                 "0x58B4C2FEf19f5CDdd944AadD8DC99cCC71bfeFDc",
		L1BridgeRegistry:                 "0x2D47fa57101203855b336e9E61BC9da0A6dd0Dbc",
		TON:                              "0xa30fe40285B8f5c0457DbC3B7C8A280373c40044",
		MaxChannelDuration:               120,
		StakingURL:                       "https://sepolia.staking.tokamak.network/staking",
		TxmgrCellProofTime:               1760427360,
		NextPublicRollupL1BaseUrl:        "https://eth-sepolia.blockscout.com",
	},
	EthereumHoleskyChainID: {
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
		L2ManagerAddress:                 "0x0000000000000000000000000000000000000000",
		L1BridgeRegistry:                 "0x0000000000000000000000000000000000000000",
		TON:                              "0x0000000000000000000000000000000000000000",
		MaxChannelDuration:               120,
	},
}

var BaseBatchInboxAddress = "0xff00000000000000000000000000000000000000"

var DefaultL2BlockTimeInSeconds uint64 = 2

const (
	OptimismSepoliaChainID = 10
	OptimismChainID        = 11155420
	BaseSepoliaChainID     = 84532
	BaseChainID            = 8453
	UnichainSepoliaChainID = 1301
	UnichainChainID        = 130
)

var L2ChainConfigurations = map[uint64]struct {
	ETHAddress        string `json:"eth_address"`
	NativeTokenName   string `json:"native_token_name"`
	NativeTokenSymbol string `json:"native_token_symbol"`
	USDCAddress       string `json:"usdc_address"`
	USDTAddress       string `json:"usdt_address"`
	TONAddress        string `json:"ton_address"`
}{
	OptimismSepoliaChainID: {
		ETHAddress:        "0x0000000000000000000000000000000000000000",
		USDCAddress:       "0x5fd84259d66cd46123540766be93dfe6d43130d7",
		NativeTokenName:   "ETH",
		NativeTokenSymbol: "ETH",
	},
	OptimismChainID: {
		ETHAddress:        "0x0000000000000000000000000000000000000000",
		USDCAddress:       "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
		USDTAddress:       "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
		NativeTokenName:   "ETH",
		NativeTokenSymbol: "ETH",
	},
	BaseSepoliaChainID: {
		ETHAddress:        "0x0000000000000000000000000000000000000000",
		USDCAddress:       "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
		USDTAddress:       "0x323e78f944A9a1FcF3a10efcC5319DBb0bB6e673",
		NativeTokenName:   "ETH",
		NativeTokenSymbol: "ETH",
	},
	BaseChainID: {
		ETHAddress:        "0x0000000000000000000000000000000000000000",
		USDCAddress:       "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		USDTAddress:       "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2",
		NativeTokenName:   "ETH",
		NativeTokenSymbol: "ETH",
	},
}

const (
	USDCAddress = "0x4200000000000000000000000000000000000778"
	ETH         = "0x4200000000000000000000000000000000000486"
	TON         = "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000"
)
