package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// setupAAPaymaster configures the AA Paymaster infrastructure on L2 after the network starts.
// It is a no-op for TON fee token or non-AA presets (General/DeFi).
//
// Steps (all on L2):
//  1. EntryPoint.depositTo(MultiTokenPaymaster) — deposit fee token for gas sponsorship
//  2. SimplePriceOracle.updatePrice(initialPrice) — set initial TON/token exchange rate
//  3. MultiTokenPaymaster.addToken(tokenAddr, oracle, markupPct, decimals) — register fee token
//
// Note on USDT: USDT has no L2 predeploy address in the current tokamak-thanos release.
// AA setup is skipped for USDT with a warning. Manual configuration is required.
func (t *ThanosStack) setupAAPaymaster(ctx context.Context) error {
	if !constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
		return nil
	}

	feeToken := t.deployConfig.FeeToken

	// USDT has no L2 predeploy — cannot be registered automatically.
	if feeToken == constants.FeeTokenUSDT {
		t.logger.Warn("⚠️  USDT has no L2 predeploy address in this release.")
		t.logger.Warn("   AA Paymaster setup skipped for USDT.")
		t.logger.Warn("   To enable USDT as a paymaster fee token, deploy an OptimismMintableERC20")
		t.logger.Warn("   for USDT via the standard bridge, then call MultiTokenPaymaster.addToken() manually.")
		return nil
	}

	t.logger.Infof("🔧 Setting up AA Paymaster for fee token: %s", feeToken)

	// Resolve verified L2 predeploy address and paymaster parameters.
	tokenL2Addr, markupPct, decimals, initialPrice, err := resolveAATokenConfig(feeToken)
	if err != nil {
		return fmt.Errorf("unsupported fee token for AA setup: %w", err)
	}

	// Connect to L2.
	l2Client, err := ethclient.DialContext(ctx, localL2RPCURL())
	if err != nil {
		return fmt.Errorf("failed to connect to L2 RPC: %w", err)
	}
	defer l2Client.Close()

	// Wait for L2 to be responsive (up to ~30s).
	var l2ChainID *big.Int
	for attempt := 1; attempt <= 6; attempt++ {
		l2ChainID, err = l2Client.ChainID(ctx)
		if err == nil {
			break
		}
		t.logger.Warnf("Waiting for L2 RPC (attempt %d/6): %v", attempt, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("L2 RPC unavailable after retries: %w", err)
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(t.deployConfig.AdminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// Helper: build, sign, and send a raw transaction on L2.
	sendTx := func(toAddr common.Address, value *big.Int, calldata []byte) error {
		nonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
		if err != nil {
			return fmt.Errorf("failed to get nonce: %w", err)
		}
		gasPrice, err := l2Client.SuggestGasPrice(ctx)
		if err != nil {
			return fmt.Errorf("failed to get gas price: %w", err)
		}
		gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2)) // 2× for reliable inclusion

		tx := types.NewTransaction(nonce, toAddr, value, 300_000, gasPrice, calldata)
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(l2ChainID), privKey)
		if err != nil {
			return fmt.Errorf("failed to sign tx: %w", err)
		}
		return l2Client.SendTransaction(ctx, signedTx)
	}

	entryPoint := common.HexToAddress(constants.AAEntryPoint)
	oracle := common.HexToAddress(constants.SimplePriceOraclePredeploy)
	paymaster := common.HexToAddress(constants.MultiTokenPaymasterPredeploy)
	tokenAddr := common.HexToAddress(tokenL2Addr)

	// Step 1: EntryPoint.depositTo(MultiTokenPaymaster)
	// ABI: depositTo(address account) payable
	// Calldata: [4-byte selector][32-byte address (right-aligned)]
	selector1 := crypto.Keccak256([]byte("depositTo(address)"))[:4]
	calldata1 := make([]byte, 36)
	copy(calldata1[:4], selector1)
	copy(calldata1[16:36], paymaster.Bytes())

	if err := sendTx(entryPoint, constants.DefaultEntryPointDeposit, calldata1); err != nil {
		return fmt.Errorf("EntryPoint.depositTo failed: %w", err)
	}
	t.logger.Infof("✅ EntryPoint.depositTo(MultiTokenPaymaster): deposited %s wei", constants.DefaultEntryPointDeposit.String())

	// Step 2: SimplePriceOracle.updatePrice(newPrice)
	// ABI: updatePrice(uint256 newPrice)
	// Each fee token uses the single SimplePriceOraclePredeploy instance as its oracle.
	// Calldata: [4-byte selector][32-byte uint256 price]
	selector2 := crypto.Keccak256([]byte("updatePrice(uint256)"))[:4]
	calldata2 := make([]byte, 36)
	copy(calldata2[:4], selector2)
	priceBytes := initialPrice.Bytes()
	copy(calldata2[36-len(priceBytes):36], priceBytes) // uint256 right-aligned

	if err := sendTx(oracle, big.NewInt(0), calldata2); err != nil {
		return fmt.Errorf("SimplePriceOracle.updatePrice failed: %w", err)
	}
	t.logger.Infof("✅ SimplePriceOracle.updatePrice(%s): price set to %s", feeToken, initialPrice.String())

	// Step 3: MultiTokenPaymaster.addToken(token, oracle, markupPercent, decimals)
	// ABI: addToken(address token, address oracle, uint256 markupPercent, uint8 decimals)
	// markupPercent is a plain percentage (e.g. 5 = 5%), max 50. NOT basis points.
	// Calldata: [4][32 token][32 oracle][32 markupPct][32 decimals as uint256-padded uint8]
	selector3 := crypto.Keccak256([]byte("addToken(address,address,uint256,uint8)"))[:4]
	calldata3 := make([]byte, 132) // 4 + 32*4
	copy(calldata3[:4], selector3)
	copy(calldata3[16:36], tokenAddr.Bytes())  // token address, right-aligned
	copy(calldata3[48:68], oracle.Bytes())     // oracle address, right-aligned
	markupBig := new(big.Int).SetUint64(markupPct)
	markupBytes := markupBig.Bytes()
	copy(calldata3[100-len(markupBytes):100], markupBytes) // markupPercent uint256
	calldata3[131] = byte(decimals)                        // decimals uint8, right-aligned in 32 bytes

	if err := sendTx(paymaster, big.NewInt(0), calldata3); err != nil {
		return fmt.Errorf("MultiTokenPaymaster.addToken failed: %w", err)
	}
	t.logger.Infof("✅ MultiTokenPaymaster.addToken(%s, markup=%d%%, decimals=%d)", feeToken, markupPct, decimals)

	return nil
}

// aaMarkupForToken returns the markup percent for the given fee token (used for log messages).
func aaMarkupForToken(feeToken string) uint64 {
	switch feeToken {
	case constants.FeeTokenETH:
		return constants.ETHPaymasterMarkupPct
	case constants.FeeTokenUSDC:
		return constants.USDCPaymasterMarkupPct
	default:
		return 0
	}
}

// resolveAATokenConfig returns the verified L2 predeploy address and paymaster parameters
// for a given fee token.
//
// Verified predeploy addresses (from tokamak-thanos Predeploys.sol):
//   - ETH (WETH):  0x4200000000000000000000000000000000000486
//   - USDC:        0x4200000000000000000000000000000000000778
//
// USDT is intentionally absent — it has no L2 predeploy in the current release.
func resolveAATokenConfig(feeToken string) (l2Addr string, markupPct uint64, decimals uint8, initialPrice *big.Int, err error) {
	switch feeToken {
	case constants.FeeTokenETH:
		// L2 WETH predeploy — verified in Predeploys.sol line 77 and op-bindings/predeploys/addresses.go line 28.
		return "0x4200000000000000000000000000000000000486",
			constants.ETHPaymasterMarkupPct,
			18,
			constants.DefaultETHInitialPrice,
			nil
	case constants.FeeTokenUSDC:
		// L2 USDC predeploy (FiatTokenV2_2) — verified in Predeploys.sol line 119 and op-bindings/predeploys/addresses.go line 42.
		// DeFi and Full presets only.
		return "0x4200000000000000000000000000000000000778",
			constants.USDCPaymasterMarkupPct,
			6,
			constants.DefaultUSDCInitialPrice,
			nil
	default:
		return "", 0, 0, nil, fmt.Errorf("fee token %q has no L2 predeploy address — manual MultiTokenPaymaster.addToken() required", feeToken)
	}
}
