package constants

type CrossTradeDeployMode string

func IsSupportedCrossTradeDeployMode(mode CrossTradeDeployMode) bool {
	return mode == CrossTradeDeployModeL2ToL1 || mode == CrossTradeDeployModeL2ToL2
}

type DefaultContractAddress struct {
	L2CrossDomainMessengerAddress string `json:"l2_cross_domain_messenger_address"`
	NativeTokenAddress            string `json:"native_token_address"`
	L1StandardBridgeAddress       string `json:"l1_standard_bridge_address"`
	L1USDCBridgeAddress           string `json:"l1_usdc_bridge_address"`
	L1CrossDomainMessengerAddress string `json:"l1_cross_domain_messenger_address"`
}

const (
	CrossTradeDeployModeL2ToL1 CrossTradeDeployMode = "l2_to_l1"
	CrossTradeDeployModeL2ToL2 CrossTradeDeployMode = "l2_to_l2"
)

var DefaultContractAddresses = map[uint64]DefaultContractAddress{
	BaseChainID: { // Base Mainnet
		L2CrossDomainMessengerAddress: "0x4200000000000000000000000000000000000007",
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0x3154Cf16ccdb4C6d922629664174b904d80F2C35",
		L1USDCBridgeAddress:           "0x3154Cf16ccdb4C6d922629664174b904d80F2C35",
		L1CrossDomainMessengerAddress: "0x866E82a600A1414e583f7F13623F1aC5d58b0Afa",
	},
	BaseSepoliaChainID: { // Base Sepolia
		L2CrossDomainMessengerAddress: "0x4200000000000000000000000000000000000007",
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0xfd0Bf71F60660E2f608ed56e1659C450eB113120",
		L1USDCBridgeAddress:           "0xfd0Bf71F60660E2f608ed56e1659C450eB113120",
		L1CrossDomainMessengerAddress: "0xC34855F4De64F1840e5686e64278da901e261f20",
	},
	OptimismChainID: { // Optimism Mainnet
		L2CrossDomainMessengerAddress: "0x4200000000000000000000000000000000000007",
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		L1USDCBridgeAddress:           "0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		L1CrossDomainMessengerAddress: "0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1",
	},
	OptimismSepoliaChainID: { // Optimism Sepolia
		L2CrossDomainMessengerAddress: "0x4200000000000000000000000000000000000007",
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1",
		L1USDCBridgeAddress:           "0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1",
		L1CrossDomainMessengerAddress: "0x58Cc85b8D04EA49cC6DBd3CbFFd00B4B8D6cb3ef",
	},
	UnichainChainID: {
		L2CrossDomainMessengerAddress: "0x4200000000000000000000000000000000000007",
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0x81014f44b0a345033bb2b3b21c7a1a308b35feea",
		L1USDCBridgeAddress:           "0x81014f44b0a345033bb2b3b21c7a1a308b35feea",
		L1CrossDomainMessengerAddress: "0x9a3d64e386c18cb1d6d5179a9596a4b5736e98a6",
	},
	UnichainSepoliaChainID: {
		L2CrossDomainMessengerAddress: "0x4200000000000000000000000000000000000007",
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0xea58fca6849d79ead1f26608855c2d6407d54ce2",
		L1USDCBridgeAddress:           "0xea58fca6849d79ead1f26608855c2d6407d54ce2",
		L1CrossDomainMessengerAddress: "0x448a37330a60494e666f6dd60ad48d930aeba381",
	},
}
