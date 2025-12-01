package constants

type CrossTradeDeployMode string

type L1ContractAddress struct {
	NativeTokenAddress            string `json:"native_token_address"`
	L1StandardBridgeAddress       string `json:"l1_standard_bridge_address"`
	L1USDCBridgeAddress           string `json:"l1_usdc_bridge_address"`
	L1CrossDomainMessengerAddress string `json:"l1_cross_domain_messenger_address"`
}

const (
	CrossTradeDeployModeL2ToL1 CrossTradeDeployMode = "l2_to_l1"
	CrossTradeDeployModeL2ToL2 CrossTradeDeployMode = "l2_to_l2"
)

var L1ContractAddresses = map[uint64]L1ContractAddress{
	8453: { // Base Mainnet
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0x3154Cf16ccdb4C6d922629664174b904d80F2C35",
		L1USDCBridgeAddress:           "0x3154Cf16ccdb4C6d922629664174b904d80F2C35",
		L1CrossDomainMessengerAddress: "0x866E82a600A1414e583f7F13623F1aC5d58b0Afa",
	},
	84532: { // Base Sepolia
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0xfd0Bf71F60660E2f608ed56e1659C450eB113120",
		L1USDCBridgeAddress:           "0xfd0Bf71F60660E2f608ed56e1659C450eB113120",
		L1CrossDomainMessengerAddress: "0xC34855F4De64F1840e5686e64278da901e261f20",
	},
	10: { // Optimism Mainnet
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		L1USDCBridgeAddress:           "0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		L1CrossDomainMessengerAddress: "0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1",
	},
	11155420: { // Optimism Sepolia
		NativeTokenAddress:            "0x0000000000000000000000000000000000000000",
		L1StandardBridgeAddress:       "0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1",
		L1USDCBridgeAddress:           "0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1",
		L1CrossDomainMessengerAddress: "0x58Cc85b8D04EA49cC6DBd3CbFFd00B4B8D6cb3ef",
	},
}
