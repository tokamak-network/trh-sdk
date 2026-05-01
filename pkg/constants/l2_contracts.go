package constants

const (
	L2CrossDomainMessenger = "0x4200000000000000000000000000000000000007"
	NativeToken            = "0x0000000000000000000000000000000000000000"

	// AA predeploys (Gaming/Full preset)
	AAEntryPoint                = "0x4200000000000000000000000000000000000063"
	VerifyingPaymasterPredeploy = "0x4200000000000000000000000000000000000064"
	Simple7702Account           = "0x4200000000000000000000000000000000000065"
	SimplePriceOraclePredeploy  = "0x4200000000000000000000000000000000000066"
	MultiTokenPaymasterPredeploy = "0x4200000000000000000000000000000000000067"

	// DRB predeploy (Gaming/Full preset) — Commit2RevealDRB replaces VRFCoordinator + VRFPredeploy
	Commit2RevealDRB = "0x4200000000000000000000000000000000000060"

	// Uniswap V3 predeploys (Gaming/Full preset)
	// Verified in tokamak-thanos Predeploys.sol lines 81-93.
	UniswapV3FactoryPredeploy        = "0x4200000000000000000000000000000000000502"
	UniswapV3PositionManagerPredeploy = "0x4200000000000000000000000000000000000504"

	// WTONPredeploy is the wrapped native token (TON) on L2, used as one side
	// of the Uniswap V3 pool for the AA oracle.
	// Verified in tokamak-thanos Predeploys.sol line 27 (WETH = wrapped native).
	WTONPredeploy = "0x4200000000000000000000000000000000000006"
)
