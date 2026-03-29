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
// For USDT (no L2 predeploy): OptimismMintableERC20Factory.createOptimismMintableERC20 is called
// first to deploy a bridged USDT token on L2, then steps 1–3 follow using the deployed address.
func (t *ThanosStack) setupAAPaymaster(ctx context.Context) error {
	if !constants.NeedsAASetup(t.deployConfig.Preset, t.deployConfig.FeeToken) {
		return nil
	}

	feeToken := t.deployConfig.FeeToken

	t.logger.Infof("🔧 Setting up AA Paymaster for fee token: %s", feeToken)

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

	// sendTxAndWait builds, signs, sends a transaction, and waits for its receipt.
	sendTxAndWait := func(toAddr common.Address, value *big.Int, calldata []byte) (*types.Receipt, error) {
		nonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}
		gasPrice, err := l2Client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas price: %w", err)
		}
		gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2)) // 2× for reliable inclusion

		tx := types.NewTransaction(nonce, toAddr, value, 300_000, gasPrice, calldata)
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(l2ChainID), privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign tx: %w", err)
		}
		if err := l2Client.SendTransaction(ctx, signedTx); err != nil {
			return nil, err
		}
		txHash := signedTx.Hash()
		for attempt := 1; attempt <= 30; attempt++ {
			receipt, err := l2Client.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			time.Sleep(2 * time.Second)
		}
		return nil, fmt.Errorf("tx %s not mined after 60s", txHash.Hex())
	}

	// sendTx sends a transaction without waiting for receipt (fire-and-forget).
	sendTx := func(toAddr common.Address, value *big.Int, calldata []byte) error {
		_, err := sendTxAndWait(toAddr, value, calldata)
		return err
	}

	// Resolve L2 token address and paymaster parameters.
	var tokenAddr common.Address
	var markupPct uint64
	var decimals uint8
	var initialPrice *big.Int

	if feeToken == constants.FeeTokenUSDT {
		// USDT has no L2 predeploy — deploy a bridged token via the Standard Bridge factory.
		l1USDTCfg := constants.GetFeeTokenConfig(constants.FeeTokenUSDT, t.deployConfig.L1ChainID)
		if l1USDTCfg.L1Address == "" || l1USDTCfg.L1Address == "0x0000000000000000000000000000000000000000" {
			return fmt.Errorf("L1 USDT address not configured for chain %d", t.deployConfig.L1ChainID)
		}
		l1USDTAddr := common.HexToAddress(l1USDTCfg.L1Address)

		t.logger.Infof("🏭 Deploying bridged USDT on L2 via OptimismMintableERC20Factory (L1: %s)...", l1USDTCfg.L1Address)
		deployedAddr, err := deployBridgedUSDT(l1USDTAddr, sendTxAndWait)
		if err != nil {
			return fmt.Errorf("failed to deploy bridged USDT on L2: %w", err)
		}
		t.logger.Infof("✅ Bridged USDT deployed at L2: %s", deployedAddr.Hex())

		tokenAddr = deployedAddr
		markupPct = constants.USDTPaymasterMarkupPct
		decimals = 6
		initialPrice = constants.DefaultUSDTInitialPrice
	} else {
		l2Addr, mp, d, ip, err := resolveAATokenConfig(feeToken)
		if err != nil {
			return fmt.Errorf("unsupported fee token for AA setup: %w", err)
		}
		tokenAddr = common.HexToAddress(l2Addr)
		markupPct = mp
		decimals = d
		initialPrice = ip
	}

	entryPoint := common.HexToAddress(constants.AAEntryPoint)
	oracle := common.HexToAddress(constants.SimplePriceOraclePredeploy)
	paymaster := common.HexToAddress(constants.MultiTokenPaymasterPredeploy)

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

// deployBridgedUSDT calls OptimismMintableERC20Factory.createOptimismMintableERC20 on L2
// to deploy a bridged USDT token, waits for the transaction receipt, and returns the
// deployed L2 token address parsed from the OptimismMintableERC20Created event.
func deployBridgedUSDT(
	l1USDTAddr common.Address,
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
) (common.Address, error) {
	factory := common.HexToAddress(constants.OptimismMintableERC20FactoryPredeploy)

	// ABI-encode: createOptimismMintableERC20(address _remoteToken, string _name, string _symbol)
	// Dynamic-type ABI layout:
	//   [0:4]    selector
	//   [4:36]   address (right-aligned)
	//   [36:68]  offset to _name from start of args = 96 (3×32)
	//   [68:100] offset to _symbol from start of args = 160 (96 + 32 len + 32 data)
	//   [100:132] len("Tether USD") = 10
	//   [132:164] "Tether USD" zero-padded to 32 bytes
	//   [164:196] len("USDT") = 4
	//   [196:228] "USDT" zero-padded to 32 bytes
	selector := crypto.Keccak256([]byte("createOptimismMintableERC20(address,string,string)"))[:4]

	name := []byte("Tether USD") // 10 bytes → pads to 32
	symbol := []byte("USDT")     // 4 bytes  → pads to 32

	padTo32 := func(b []byte) []byte {
		n := (len(b) + 31) / 32 * 32
		if n == 0 {
			n = 32
		}
		p := make([]byte, n)
		copy(p, b)
		return p
	}
	paddedName := padTo32(name)   // 32 bytes
	paddedSym := padTo32(symbol)  // 32 bytes

	nameOffset := uint64(3 * 32)                                        // 96
	symOffset := nameOffset + 32 + uint64(len(paddedName))              // 160

	totalLen := 4 + 3*32 + 32 + len(paddedName) + 32 + len(paddedSym) // 228
	calldata := make([]byte, totalLen)

	copy(calldata[:4], selector)

	// address right-aligned in [4:36]
	copy(calldata[16:36], l1USDTAddr.Bytes())

	// nameOffset right-aligned in [36:68]
	nameOffBytes := new(big.Int).SetUint64(nameOffset).Bytes()
	copy(calldata[68-len(nameOffBytes):68], nameOffBytes)

	// symOffset right-aligned in [68:100]
	symOffBytes := new(big.Int).SetUint64(symOffset).Bytes()
	copy(calldata[100-len(symOffBytes):100], symOffBytes)

	// name length right-aligned in [100:132]
	nameLenBytes := new(big.Int).SetUint64(uint64(len(name))).Bytes()
	copy(calldata[132-len(nameLenBytes):132], nameLenBytes)

	// name data
	copy(calldata[132:132+len(paddedName)], paddedName)

	// symbol length right-aligned in [164:196]
	symLenOff := 132 + len(paddedName) // 164
	symLenBytes := new(big.Int).SetUint64(uint64(len(symbol))).Bytes()
	copy(calldata[symLenOff+32-len(symLenBytes):symLenOff+32], symLenBytes)

	// symbol data
	copy(calldata[symLenOff+32:symLenOff+32+len(paddedSym)], paddedSym)

	receipt, err := sendTxAndWait(factory, big.NewInt(0), calldata)
	if err != nil {
		return common.Address{}, fmt.Errorf("createOptimismMintableERC20 tx failed: %w", err)
	}

	// Parse OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
	// Topics[0] = event sig, Topics[1] = localToken (L2 address, indexed)
	eventSigHash := common.BytesToHash(crypto.Keccak256([]byte("OptimismMintableERC20Created(address,address,address)")))
	for _, log := range receipt.Logs {
		if log.Address == factory && len(log.Topics) >= 2 && log.Topics[0] == eventSigHash {
			return common.BytesToAddress(log.Topics[1].Bytes()), nil
		}
	}
	return common.Address{}, fmt.Errorf("OptimismMintableERC20Created event not found in receipt (tx: %s)", receipt.TxHash.Hex())
}

// aaMarkupForToken returns the markup percent for the given fee token (used for log messages).
func aaMarkupForToken(feeToken string) uint64 {
	switch feeToken {
	case constants.FeeTokenETH:
		return constants.ETHPaymasterMarkupPct
	case constants.FeeTokenUSDC:
		return constants.USDCPaymasterMarkupPct
	case constants.FeeTokenUSDT:
		return constants.USDTPaymasterMarkupPct
	default:
		return 0
	}
}

// resolveAATokenConfig returns the verified L2 predeploy address and paymaster parameters
// for ETH and USDC fee tokens. USDT is handled separately via deployBridgedUSDT.
//
// Verified predeploy addresses (from tokamak-thanos Predeploys.sol):
//   - ETH (WETH):  0x4200000000000000000000000000000000000486
//   - USDC:        0x4200000000000000000000000000000000000778
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
		return "", 0, 0, nil, fmt.Errorf("fee token %q: use deployBridgedUSDT for USDT or check token configuration", feeToken)
	}
}
