//go:build live

// Live integration test for DeployCrossTradeLocal.
// Requires a running L2 node at localhost:8545 (from the deployed ect-defi-crosstrade stack).
// Run with: go test -v -tags live -run TestDeployCrossTradeLocalLive -timeout 60m ./pkg/stacks/thanos/
package thanos

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDeployCrossTradeLocalLive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Minute)
	defer cancel()

	logger, _ := zap.NewDevelopment()
	stack := &ThanosStack{logger: logger.Sugar()}

	input := &DeployCrossTradeLocalInput{
		L1RPCUrl:             "https://eth-sepolia.g.alchemy.com/v2/zPJeUK2LKGg4LjvHPGXYl1Ef4FJ_u7Gn",
		L1ChainID:            11155111,
		DeployerPrivateKey:   "6544462e611a9040a74d0fdfe3f00ed4b3c3e924a6f29165059c76e6e587e4ff",
		L2RPCUrl:             "http://localhost:8545",
		L2ChainID:            111551215120,
		OptimismPortalProxy:  "0xC0337869d225FDb7F7e31e6761bcc9c9F6Ac2fc7",
		CrossDomainMessenger: "0x58060E91B14a5E7c7e5069f6330b4e11a9D00558",
		L1CrossTradeProxy:    "0xf3473E20F1d9EB4468C72454a27aA1C65B67AB35",
		L2toL2CrossTradeL1:   "0xDa2CbF69352cB46d9816dF934402b421d93b6BC2",
		SupportedTokens:      []TokenPair{},
	}

	t.Log("Starting DeployCrossTradeLocal — this takes 20-40 minutes")

	output, err := stack.DeployCrossTradeLocal(ctx, input)
	if err != nil {
		t.Fatalf("DeployCrossTradeLocal failed: %v", err)
	}

	t.Logf("SUCCESS!")
	t.Logf("L2CrossTrade:          %s", output.L2CrossTrade)
	t.Logf("L2CrossTradeProxy:     %s", output.L2CrossTradeProxy)
	t.Logf("L2toL2CrossTradeL2:    %s", output.L2toL2CrossTradeL2)
	t.Logf("L2toL2CrossTradeProxy: %s", output.L2toL2CrossTradeProxy)

	// Verify all 4 addresses are non-empty
	if output.L2CrossTrade == "" {
		t.Error("L2CrossTrade address is empty")
	}
	if output.L2CrossTradeProxy == "" {
		t.Error("L2CrossTradeProxy address is empty")
	}
	if output.L2toL2CrossTradeL2 == "" {
		t.Error("L2toL2CrossTradeL2 address is empty")
	}
	if output.L2toL2CrossTradeProxy == "" {
		t.Error("L2toL2CrossTradeProxy address is empty")
	}
}
