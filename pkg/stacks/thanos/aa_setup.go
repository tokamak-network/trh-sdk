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
//  1. EntryPoint.depositTo(MultiTokenPaymaster) — deposit native fee token for gas sponsorship
//  2. SimplePriceOracle.setPrice(feeTokenL2Addr, initialPrice) — set initial TON/token price
//  3. MultiTokenPaymaster.addToken(feeTokenL2Addr, oracle, markup, decimals) — register fee token
func (t *ThanosStack) setupAAPaymaster(ctx context.Context) error {
	if !constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
		return nil
	}

	feeToken := t.deployConfig.FeeToken
	t.logger.Infof("🔧 Setting up AA Paymaster for fee token: %s", feeToken)

	// Resolve fee token L2 address and paymaster parameters.
	tokenL2Addr, markup, decimals, initialPrice, err := resolveAATokenConfig(feeToken)
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
	selector1 := crypto.Keccak256([]byte("depositTo(address)"))[:4]
	calldata1 := make([]byte, 36)
	copy(calldata1[:4], selector1)
	copy(calldata1[16:36], paymaster.Bytes()) // address padded to 32 bytes (right-aligned)

	if err := sendTx(entryPoint, constants.DefaultEntryPointDeposit, calldata1); err != nil {
		return fmt.Errorf("EntryPoint.depositTo failed: %w", err)
	}
	t.logger.Infof("✅ EntryPoint.depositTo(MultiTokenPaymaster): deposited %s wei", constants.DefaultEntryPointDeposit.String())

	// Step 2: SimplePriceOracle.setPrice(token, price)
	// ABI: setPrice(address token, uint256 price)
	selector2 := crypto.Keccak256([]byte("setPrice(address,uint256)"))[:4]
	calldata2 := make([]byte, 68)
	copy(calldata2[:4], selector2)
	copy(calldata2[16:36], tokenAddr.Bytes()) // address padded
	priceBytes := initialPrice.Bytes()
	copy(calldata2[68-len(priceBytes):68], priceBytes) // uint256 right-aligned

	if err := sendTx(oracle, big.NewInt(0), calldata2); err != nil {
		return fmt.Errorf("SimplePriceOracle.setPrice failed: %w", err)
	}
	t.logger.Infof("✅ SimplePriceOracle.setPrice(%s, %s)", feeToken, initialPrice.String())

	// Step 3: MultiTokenPaymaster.addToken(token, oracle, markup, decimals)
	// ABI: addToken(address token, address oracle, uint256 markup, uint8 decimals)
	selector3 := crypto.Keccak256([]byte("addToken(address,address,uint256,uint8)"))[:4]
	calldata3 := make([]byte, 132) // 4 + 32*4
	copy(calldata3[:4], selector3)
	copy(calldata3[16:36], tokenAddr.Bytes())     // token address
	copy(calldata3[48:68], oracle.Bytes())        // oracle address
	markupBig := new(big.Int).SetUint64(markup)
	markupBytes := markupBig.Bytes()
	copy(calldata3[100-len(markupBytes):100], markupBytes) // markup uint256
	calldata3[131] = byte(decimals)                        // decimals uint8

	if err := sendTx(paymaster, big.NewInt(0), calldata3); err != nil {
		return fmt.Errorf("MultiTokenPaymaster.addToken failed: %w", err)
	}
	t.logger.Infof("✅ MultiTokenPaymaster.addToken(%s, markup=%d BPS, decimals=%d)", feeToken, markup, decimals)

	return nil
}

// aaMarkupForToken returns the markup BPS for the given fee token (used for log messages).
func aaMarkupForToken(feeToken string) uint64 {
	switch feeToken {
	case constants.FeeTokenETH:
		return constants.ETHPaymasterMarkupBPS
	case constants.FeeTokenUSDT:
		return constants.USDTPaymasterMarkupBPS
	case constants.FeeTokenUSDC:
		return constants.USDCPaymasterMarkupBPS
	default:
		return 0
	}
}

// resolveAATokenConfig returns L2 token address and paymaster parameters for the given fee token.
// L2 fee token addresses follow the OP Stack predeploy pattern.
func resolveAATokenConfig(feeToken string) (l2Addr string, markup uint64, decimals uint8, initialPrice *big.Int, err error) {
	switch feeToken {
	case constants.FeeTokenETH:
		// L2 ETH is represented as a wrapped ERC20 predeploy.
		return "0x4200000000000000000000000000000000000486",
			constants.ETHPaymasterMarkupBPS,
			18,
			constants.DefaultETHInitialPrice,
			nil
	case constants.FeeTokenUSDT:
		// L2 USDT predeploy address (mirrors the L1 USDT configuration).
		return "0x4200000000000000000000000000000000000487",
			constants.USDTPaymasterMarkupBPS,
			6,
			constants.DefaultUSDTInitialPrice,
			nil
	case constants.FeeTokenUSDC:
		return "0x4200000000000000000000000000000000000778",
			constants.USDCPaymasterMarkupBPS,
			6,
			constants.DefaultUSDCInitialPrice,
			nil
	default:
		return "", 0, 0, nil, fmt.Errorf("fee token %q not supported for AA paymaster", feeToken)
	}
}
