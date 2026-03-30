package thanos

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

const (
	// refillPollInterval is how often the monitor checks the EntryPoint balance.
	refillPollInterval = 5 * time.Minute

	// Base thresholds at 18 decimals — scaled at runtime by feeTokenDecimals().
	refillThresholdBase18   = uint64(5e17) // 0.5 token (18 dec)
	refillAmountBase18      = uint64(5e18) // 5 tokens (18 dec)
	adminWarnThresholdBase18 = uint64(2e18) // 2 tokens (18 dec)
)

// startEntryPointRefillMonitor starts a background goroutine that periodically checks
// the EntryPoint deposit balance for MultiTokenPaymaster and tops it up from the admin
// wallet when it falls below refillThresholdWei.
//
// The goroutine exits when ctx is cancelled (i.e., when the stack shuts down).
// Errors are logged but do not stop the monitor.
func (t *ThanosStack) startEntryPointRefillMonitor(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(refillPollInterval)
		defer ticker.Stop()

		// Run once immediately so the first check does not wait a full interval.
		if err := t.checkAndRefillEntryPoint(ctx); err != nil {
			t.logger.Warnf("EntryPoint refill initial check failed: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				t.logger.Infof("EntryPoint refill monitor stopped")
				return
			case <-ticker.C:
				if err := t.checkAndRefillEntryPoint(ctx); err != nil {
					t.logger.Warnf("EntryPoint refill check failed: %v", err)
				}
			}
		}
	}()
	dec := feeTokenDecimals(t.deployConfig.FeeToken)
	var sf *big.Int
	if dec < 18 {
		sf = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-dec)), nil)
	} else {
		sf = big.NewInt(1)
	}
	logThreshold := new(big.Int).Div(new(big.Int).SetUint64(refillThresholdBase18), sf)
	logRefill := new(big.Int).Div(new(big.Int).SetUint64(refillAmountBase18), sf)
	t.logger.Infof("EntryPoint refill monitor started (poll=%s, threshold=%s, refill=%s, token=%s)", refillPollInterval, logThreshold.String(), logRefill.String(), t.deployConfig.FeeToken)
}

// refillMu guards against concurrent refill transactions from multiple monitor ticks.
var refillMu sync.Mutex

// checkAndRefillEntryPoint checks the EntryPoint deposit balance for MultiTokenPaymaster
// and calls depositTo() from the admin wallet if the balance is below the threshold.
func (t *ThanosStack) checkAndRefillEntryPoint(ctx context.Context) error {
	l2Client, err := ethclient.DialContext(ctx, localL2RPCURL())
	if err != nil {
		return fmt.Errorf("dial L2: %w", err)
	}
	defer l2Client.Close()

	entryPoint := common.HexToAddress(constants.AAEntryPoint)
	paymaster := common.HexToAddress(constants.MultiTokenPaymasterPredeploy)

	// Query EntryPoint.balanceOf(paymaster).
	deposit, err := entryPointBalanceOf(ctx, l2Client, entryPoint, paymaster)
	if err != nil {
		return fmt.Errorf("balanceOf query failed: %w", err)
	}

	// Scale thresholds by fee token decimals.
	dec := feeTokenDecimals(t.deployConfig.FeeToken)
	var scaleFactor *big.Int
	if dec < 18 {
		scaleFactor = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-dec)), nil)
	} else {
		scaleFactor = big.NewInt(1)
	}
	threshold := new(big.Int).Div(new(big.Int).SetUint64(refillThresholdBase18), scaleFactor)
	refillValue := new(big.Int).Div(new(big.Int).SetUint64(refillAmountBase18), scaleFactor)
	warnThreshold := new(big.Int).Div(new(big.Int).SetUint64(adminWarnThresholdBase18), scaleFactor)

	if deposit.Cmp(threshold) >= 0 {
		// Enough balance — no action needed.
		t.logger.Infof("EntryPoint deposit OK: %s wei (threshold=%s)", deposit.String(), threshold.String())
		return nil
	}

	t.logger.Infof("EntryPoint deposit low: %s wei — triggering refill (%s wei)", deposit.String(), refillValue.String())

	// Guard against overlapping refill transactions.
	if !refillMu.TryLock() {
		t.logger.Infof("EntryPoint refill already in progress — skipping")
		return nil
	}
	defer refillMu.Unlock()

	// Derive admin address from private key.
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(t.deployConfig.AdminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin private key: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// Warn if admin wallet balance is running low.
	adminBalance, err := l2Client.BalanceAt(ctx, adminAddr, nil)
	if err == nil {
		if adminBalance.Cmp(warnThreshold) < 0 {
			t.logger.Warnf("admin wallet balance is low: %s wei", adminBalance.String())
		}
	}

	l2ChainID, err := l2Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("get chain ID: %w", err)
	}

	// Build sendTxAndWait closure (same pattern as setupAAPaymaster).
	sendTxAndWait := func(toAddr common.Address, value *big.Int, calldata []byte) (*types.Receipt, error) {
		nonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
		if err != nil {
			return nil, fmt.Errorf("get nonce: %w", err)
		}
		gasPrice, err := l2Client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("get gas price: %w", err)
		}
		gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

		tx := types.NewTransaction(nonce, toAddr, value, 100_000, gasPrice, calldata)
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(l2ChainID), privKey)
		if err != nil {
			return nil, fmt.Errorf("sign tx: %w", err)
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

	// ABI: depositTo(address account) payable
	selector := crypto.Keccak256([]byte("depositTo(address)"))[:4]
	calldata := make([]byte, 36)
	copy(calldata[:4], selector)
	copy(calldata[16:36], paymaster.Bytes())

	if _, err := sendTxAndWait(entryPoint, refillValue, calldata); err != nil {
		return fmt.Errorf("depositTo failed: %w", err)
	}

	t.logger.Infof("✅ EntryPoint refilled: deposited %s wei to paymaster %s", refillValue.String(), paymaster.Hex())
	return nil
}

// entryPointBalanceOf calls EntryPoint.balanceOf(account) via eth_call.
// ABI: balanceOf(address account) view returns (uint256)
func entryPointBalanceOf(ctx context.Context, l2Client *ethclient.Client, entryPoint, account common.Address) (*big.Int, error) {
	selector := crypto.Keccak256([]byte("balanceOf(address)"))[:4]
	calldata := make([]byte, 36)
	copy(calldata[:4], selector)
	copy(calldata[16:36], account.Bytes())

	result, err := l2Client.CallContract(ctx, ethereum.CallMsg{
		To:   &entryPoint,
		Data: calldata,
	}, nil)
	if err != nil {
		return nil, err
	}
	if len(result) < 32 {
		return nil, fmt.Errorf("unexpected balanceOf result length: %d", len(result))
	}
	return new(big.Int).SetBytes(result[:32]), nil
}
