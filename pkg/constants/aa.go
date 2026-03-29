package constants

import "math/big"

// AA paymaster setup constants.
// These apply only to Gaming/Full presets when fee token is not TON.

// OptimismMintableERC20FactoryPredeploy is the L2 predeploy address of the factory used to
// deploy bridged ERC20 tokens (e.g. USDT) via the Standard Bridge.
// Verified in register_metadata.go and OP Stack Predeploys.sol.
const OptimismMintableERC20FactoryPredeploy = "0x4200000000000000000000000000000000000012"

// DefaultEntryPointDeposit is the amount of native token (fee token) deposited
// into EntryPoint on behalf of MultiTokenPaymaster during AA setup.
// 1 token unit (18 decimals) = 1e18 wei.
var DefaultEntryPointDeposit = new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

// Paymaster markup as a plain percentage (e.g. 5 = 5%).
// MultiTokenPaymaster.addToken() takes markupPercent directly, not basis points.
// Maximum allowed by the contract is 50 (50%).
const (
	ETHPaymasterMarkupPct  = uint64(5) // 5%
	USDCPaymasterMarkupPct = uint64(3) // 3%
	USDTPaymasterMarkupPct = uint64(3) // 3%
)

// Initial price values for SimplePriceOracle.
// Price is expressed as "amount of fee token per 1 TON, scaled to 18 decimals".
// SimplePriceOracle stores a single price value (no per-token mapping);
// one oracle instance serves one token.
// These are conservative placeholder values — operator must call updatePrice()
// post-deployment to set accurate market rates before stale threshold (24h) triggers.
//
// ETH:  1 TON ≈ 0.0005 ETH  → price = 5e14
// USDC: 1 TON ≈ 1.5 USDC    → price = 1.5e18 (MultiTokenPaymaster scales to 6 dec internally)
// USDT: 1 TON ≈ 1.5 USDT    → price = 1.5e18 (same scale as USDC)
var (
	DefaultETHInitialPrice  = new(big.Int).Mul(big.NewInt(5), new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil))  // 5e14
	DefaultUSDCInitialPrice = new(big.Int).Mul(big.NewInt(15), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)) // 1.5e18
	DefaultUSDTInitialPrice = new(big.Int).Mul(big.NewInt(15), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)) // 1.5e18
)

// AAPresetsWithPaymaster lists presets that include AA predeploy contracts.
var AAPresetsWithPaymaster = map[string]bool{
	PresetGaming: true,
	PresetFull:   true,
}

// IsAAPreset returns true if the given preset includes AA infrastructure.
func IsAAPreset(preset string) bool {
	return AAPresetsWithPaymaster[preset]
}

// NeedsAASetup returns true when AA paymaster configuration is required:
// the preset includes AA contracts AND the fee token is not TON (which is the L2 native token).
func NeedsAASetup(preset, feeToken string) bool {
	return IsAAPreset(preset) && feeToken != FeeTokenTON
}
