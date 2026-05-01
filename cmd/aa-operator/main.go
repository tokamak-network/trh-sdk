// aa-operator is a long-running service that keeps the AA paymaster infrastructure
// healthy for L2 rollups using a non-TON native fee token.
//
// It is intended to run as a Docker Compose service alongside the L2 stack.
// The L2 RPC URL is read from L2_RPC_URL (defaults to host.docker.internal:8545).
//
// Required environment variables:
//
//	FEE_TOKEN        — non-TON fee token symbol (ETH, USDT, USDC)
//	ADMIN_PRIVATE_KEY — L2 admin wallet private key (hex, with or without 0x prefix)
//
// Optional:
//
//	COINGECKO_API_KEY — CoinGecko Pro API key; falls back to free-tier endpoint if unset
//	L2_RPC_URL        — L2 RPC URL; defaults to op-geth:8545 detection logic
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
)

func main() {
	feeToken := os.Getenv("FEE_TOKEN")
	if feeToken == "" {
		log.Fatal("FEE_TOKEN environment variable is required")
	}

	adminPrivKey := os.Getenv("ADMIN_PRIVATE_KEY")
	if adminPrivKey == "" {
		log.Fatal("ADMIN_PRIVATE_KEY environment variable is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	thanos.RunAAOperator(ctx, thanos.AAOperatorConfig{
		FeeToken:        feeToken,
		AdminPrivateKey: adminPrivKey,
	})
}
