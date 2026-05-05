package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

const (
	// priceUpdateInterval is how often the updater fetches and pushes a new price.
	priceUpdateInterval = 10 * time.Minute

	// priceUpdateThresholdPct is the minimum % change that triggers an on-chain update.
	// Updates also fire unconditionally after priceForceUpdateInterval regardless of change.
	priceUpdateThresholdPct = 2 // 2% change triggers update

	// priceForceUpdateInterval forces an update even if price hasn't moved, to prevent
	// SimplePriceOracle from going stale (STALE_THRESHOLD = 24h).
	priceForceUpdateInterval = 23 * time.Hour

	// coingeckoFreeEndpoint is the CoinGecko free-tier price endpoint (no key required).
	coingeckoFreeEndpoint = "https://api.coingecko.com/api/v3/simple/price?ids=tokamak-network&vs_currencies=usd,eth"

	// coingeckoProEndpoint is the CoinGecko Pro API endpoint (requires COINGECKO_API_KEY).
	coingeckoProEndpoint = "https://pro-api.coingecko.com/api/v3/simple/price?ids=tokamak-network&vs_currencies=usd,eth"

	// httpTimeout for external price fetch calls.
	httpTimeout = 10 * time.Second
)

// coingeckoURL returns the appropriate CoinGecko endpoint.
// If COINGECKO_API_KEY is set, it uses the Pro API endpoint with the key appended;
// otherwise it uses the free-tier endpoint (no key, ~30 req/min limit).
func coingeckoURL() string {
	if key := os.Getenv("COINGECKO_API_KEY"); key != "" {
		return coingeckoProEndpoint + "&x_cg_pro_api_key=" + key
	}
	return coingeckoFreeEndpoint
}

// coingeckoResponse is the JSON structure returned by the CoinGecko price endpoint.
type coingeckoResponse struct {
	TokamakNetwork struct {
		USD float64 `json:"usd"`
		ETH float64 `json:"eth"`
	} `json:"tokamak-network"`
}

// startPriceUpdater starts a background goroutine that periodically fetches the TON
// market price from CoinGecko and pushes it to SimplePriceOracle on L2.
//
// This keeps the oracle price current without requiring operator intervention, solving
// the 24-hour staleness problem in a simpler way than deploying a Uniswap V3 pool.
//
// Price source: CoinGecko free API (no key required).
//   - TON/USD for USDT and USDC fee tokens
//   - TON/ETH for ETH fee token
//
// An on-chain update is triggered when:
//   - Price moved more than priceUpdateThresholdPct (2%)
//   - OR more than priceForceUpdateInterval (23h) has elapsed since last update
func (t *ThanosStack) startPriceUpdater(ctx context.Context) {
	feeToken := t.deployConfig.FeeToken
	go func() {
		ticker := time.NewTicker(priceUpdateInterval)
		defer ticker.Stop()

		var lastPushed *big.Int
		var lastPushTime time.Time

		// Run immediately on start so the oracle has a fresh price right away.
		price, err := t.fetchOraclePrice(feeToken)
		if err != nil {
			t.logger.Warnf("PriceUpdater: initial fetch failed: %v", err)
		} else {
			if err := t.pushOraclePrice(ctx, price); err != nil {
				t.logger.Warnf("PriceUpdater: initial push failed: %v", err)
			} else {
				lastPushed = price
				lastPushTime = time.Now()
			}
		}

		for {
			select {
			case <-ctx.Done():
				t.logger.Infof("PriceUpdater: stopped")
				return
			case <-ticker.C:
				price, err := t.fetchOraclePrice(feeToken)
				if err != nil {
					t.logger.Warnf("PriceUpdater: fetch failed: %v", err)
					continue
				}

				needsUpdate := false
				if lastPushed == nil || lastPushed.Sign() == 0 {
					needsUpdate = true
				} else if priceDeltaPct(lastPushed, price) >= priceUpdateThresholdPct {
					needsUpdate = true
					t.logger.Infof("PriceUpdater: price moved %.1f%% — pushing update", priceDeltaPct(lastPushed, price))
				} else if time.Since(lastPushTime) >= priceForceUpdateInterval {
					needsUpdate = true
					t.logger.Infof("PriceUpdater: forcing update to prevent staleness")
				}

				if !needsUpdate {
					t.logger.Infof("PriceUpdater: price stable (%s), skipping on-chain update", price.String())
					continue
				}

				if err := t.pushOraclePrice(ctx, price); err != nil {
					t.logger.Warnf("PriceUpdater: push failed: %v", err)
					continue
				}
				lastPushed = price
				lastPushTime = time.Now()
			}
		}
	}()
	t.logger.Infof("PriceUpdater started (feeToken=%s, poll=%s, threshold=%d%%, force=%s)",
		feeToken, priceUpdateInterval, priceUpdateThresholdPct, priceForceUpdateInterval)
}

// fetchOraclePrice fetches the current TON market price from CoinGecko and converts it
// to ITokenPriceOracle format: "1 TON in feeToken, 18-decimal fixed-point".
//
// Format examples:
//
//	ETH  (18 dec): 0.0005 ETH/TON → 5e14
//	USDT (6 dec):  1.5 USDT/TON   → 1.5e18 (paymaster scales internally)
//	USDC (6 dec):  1.5 USDC/TON   → 1.5e18
func (t *ThanosStack) fetchOraclePrice(feeToken string) (*big.Int, error) {
	data, err := fetchCoinGeckoPrice()
	if err != nil {
		return nil, fmt.Errorf("CoinGecko fetch: %w", err)
	}

	switch feeToken {
	case constants.FeeTokenETH:
		if data.TokamakNetwork.ETH <= 0 {
			return nil, fmt.Errorf("CoinGecko returned zero ETH price")
		}
		// price = TON_in_ETH × 1e18
		return floatToWei(data.TokamakNetwork.ETH), nil

	case constants.FeeTokenUSDT, constants.FeeTokenUSDC:
		if data.TokamakNetwork.USD <= 0 {
			return nil, fmt.Errorf("CoinGecko returned zero USD price")
		}
		// price = TON_in_USD × 1e18 (USDT ≈ USD; paymaster scales 6-dec internally)
		return floatToWei(data.TokamakNetwork.USD), nil

	default:
		return nil, fmt.Errorf("unsupported fee token for price updater: %s", feeToken)
	}
}

// pushOraclePrice calls SimplePriceOracle.updatePrice(newPrice) on L2 from the admin wallet.
func (t *ThanosStack) pushOraclePrice(ctx context.Context, price *big.Int) error {
	l2URL := localL2RPCURL()
	if t.deployConfig.L2RpcUrl != "" {
		l2URL = t.deployConfig.L2RpcUrl
	}
	l2Client, err := ethclient.DialContext(ctx, l2URL)
	if err != nil {
		return fmt.Errorf("dial L2: %w", err)
	}
	defer l2Client.Close()

	l2ChainID, err := l2Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("get chain ID: %w", err)
	}

	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(t.deployConfig.AdminPrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("invalid admin key: %w", err)
	}
	adminAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	oracle := common.HexToAddress(constants.SimplePriceOraclePredeploy)

	// ABI: updatePrice(uint256 newPrice)
	selector := crypto.Keccak256([]byte("updatePrice(uint256)"))[:4]
	calldata := make([]byte, 36)
	copy(calldata[:4], selector)
	priceBytes := price.Bytes()
	copy(calldata[36-len(priceBytes):36], priceBytes)

	nonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return fmt.Errorf("get nonce: %w", err)
	}
	gasPrice, err := l2Client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("get gas price: %w", err)
	}
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(2))

	tx := types.NewTransaction(nonce, oracle, big.NewInt(0), 60_000, gasPrice, calldata)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(l2ChainID), privKey)
	if err != nil {
		return fmt.Errorf("sign tx: %w", err)
	}
	if err := l2Client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("send tx: %w", err)
	}

	// Wait for receipt (up to 60s).
	txHash := signedTx.Hash()
	for attempt := 1; attempt <= 30; attempt++ {
		receipt, err := l2Client.TransactionReceipt(ctx, txHash)
		if err == nil {
			if receipt.Status == 0 {
				return fmt.Errorf("updatePrice tx reverted (tx=%s)", txHash.Hex())
			}
			t.logger.Infof("✅ PriceUpdater: SimplePriceOracle.updatePrice(%s) mined (tx=%s)",
				price.String(), txHash.Hex())
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("updatePrice tx %s not mined after 60s", txHash.Hex())
}

// fetchCoinGeckoPrice fetches TON/USD and TON/ETH prices from CoinGecko free API.
// No API key required. Rate limit: ~10-50 req/min on free tier.
func fetchCoinGeckoPrice() (*coingeckoResponse, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(coingeckoURL())
	if err != nil {
		return nil, fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var data coingeckoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("JSON decode: %w", err)
	}
	return &data, nil
}

// floatToWei converts a float64 price (e.g. 0.0005 for ETH) to a *big.Int in 18-decimal
// fixed-point format (multiply by 1e18, truncate).
func floatToWei(f float64) *big.Int {
	// Use big.Float for precision; float64 has ~15 sig digits which is fine for oracle prices.
	bf := new(big.Float).SetFloat64(f)
	scale := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	bf.Mul(bf, scale)
	result, _ := bf.Int(nil) // truncate
	return result
}

// priceDeltaPct returns the absolute percentage change between old and new price.
// Returns 0 if old is zero.
func priceDeltaPct(old, newPrice *big.Int) float64 {
	if old == nil || old.Sign() == 0 {
		return 100
	}
	// |new - old| / old * 100
	diff := new(big.Int).Sub(newPrice, old)
	if diff.Sign() < 0 {
		diff.Neg(diff)
	}
	// Use float64 for the percentage (precision is fine for this use)
	oldF, _ := new(big.Float).SetInt(old).Float64()
	diffF, _ := new(big.Float).SetInt(diff).Float64()
	if oldF == 0 {
		return 100
	}
	return diffF / oldF * 100
}
