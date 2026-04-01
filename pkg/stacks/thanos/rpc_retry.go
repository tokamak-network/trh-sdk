package thanos

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	rpcRetryMaxAttempts  = 5
	rpcRetryInitialDelay = 3 * time.Second
)

// is429Error reports whether err is an HTTP 429 rate-limit response from an RPC provider.
func is429Error(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "429") ||
		strings.Contains(s, "Too Many Requests") ||
		strings.Contains(s, "rate limit")
}

// rpcCallWithRetry retries fn with exponential back-off when it returns a 429 error.
// Non-429 errors are returned immediately. Max 5 attempts, starting at 3s delay.
func rpcCallWithRetry(ctx context.Context, fn func() error) error {
	delay := rpcRetryInitialDelay
	for attempt := 1; attempt <= rpcRetryMaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !is429Error(err) || attempt == rpcRetryMaxAttempts {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
	}
	return nil
}

// sendTxWithRetry wraps ethclient.Client.SendTransaction with exponential back-off on 429.
func sendTxWithRetry(ctx context.Context, client *ethclient.Client, tx *types.Transaction) error {
	return rpcCallWithRetry(ctx, func() error {
		return client.SendTransaction(ctx, tx)
	})
}
