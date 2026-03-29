package thanos

import (
	"context"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

// AAOperatorConfig holds the runtime configuration for the aa-operator process.
type AAOperatorConfig struct {
	// FeeToken is the non-TON fee token (ETH, USDT, USDC) the paymaster is configured for.
	FeeToken string
	// AdminPrivateKey is the L2 admin wallet private key used to push oracle prices and
	// refill the EntryPoint deposit.
	AdminPrivateKey string
}

// RunAAOperator starts the AA operator background services and blocks until ctx is cancelled.
//
// It is the entrypoint for the aa-operator Docker service which keeps the AA paymaster
// infrastructure healthy after the initial setupAAPaymaster call:
//
//   - Price updater: fetches TON market price from CoinGecko every 10 minutes and pushes it
//     to SimplePriceOracle on L2, preventing the oracle from going stale (24h threshold).
//   - EntryPoint refill monitor: checks the EntryPoint deposit balance for MultiTokenPaymaster
//     every 5 minutes and tops it up from the admin wallet when it falls below 0.5 TON.
//
// The L2 RPC URL is resolved via localL2RPCURL(), which checks the L2_RPC_URL env var first
// so the Docker service can point directly at the op-geth container (http://op-geth:8545).
func RunAAOperator(ctx context.Context, cfg AAOperatorConfig) {
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck
	sugar := logger.Sugar()

	t := &ThanosStack{
		deployConfig: &types.Config{
			FeeToken:        cfg.FeeToken,
			AdminPrivateKey: cfg.AdminPrivateKey,
		},
		logger: sugar,
	}

	sugar.Infof("aa-operator starting (feeToken=%s)", cfg.FeeToken)
	t.startPriceUpdater(ctx)
	t.startEntryPointRefillMonitor(ctx)

	<-ctx.Done()
	sugar.Infof("aa-operator stopped")
}
