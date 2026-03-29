package constants

import "math/big"

// AA paymaster setup constants.
// These apply only to Gaming/Full presets when fee token is not TON.

// DefaultEntryPointDeposit is the amount of native token (fee token) deposited
// into EntryPoint on behalf of MultiTokenPaymaster during AA setup.
// 1 token unit (18 decimals) = 1e18 wei.
var DefaultEntryPointDeposit = new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

// Paymaster markup in basis points (1 BPS = 0.01%).
const (
	ETHPaymasterMarkupBPS  = uint64(500) // 5%
	USDTPaymasterMarkupBPS = uint64(300) // 3%
	USDCPaymasterMarkupBPS = uint64(300) // 3%
)

// Initial price values for SimplePriceOracle.
// Price is expressed as "amount of fee token per 1 TON, scaled to 18 decimals".
// These are conservative placeholder values — operator should update via setPrice.
//
// ETH:  1 TON ≈ 0.0005 ETH  → price = 5e14
// USDT: 1 TON ≈ 1.5 USDT    → price = 1.5e18 (18 decimal representation even for 6-decimal USDT)
// USDC: 1 TON ≈ 1.5 USDC    → price = 1.5e18
var (
	DefaultETHInitialPrice  = new(big.Int).Mul(big.NewInt(5), new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil)) // 5e14
	DefaultUSDTInitialPrice = new(big.Int).Mul(big.NewInt(15), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)) // 1.5e18
	DefaultUSDCInitialPrice = new(big.Int).Mul(big.NewInt(15), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)) // 1.5e18
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
