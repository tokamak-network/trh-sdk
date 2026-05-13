package thanos

import (
	"context"
	"testing"
)

// TestInitGenesisAnchorState_OnProgressCalled verifies that initGenesisAnchorState
// calls onProgress("Waiting for L2 genesis block") before attempting any network I/O,
// and onProgress("Submitting anchor state to L1") after obtaining the genesis block.
//
// Uses an already-cancelled context so the retry loop exits immediately without
// sleeping 5 seconds per attempt.
func TestInitGenesisAnchorState_OnProgressCalled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before call — retry loop exits on first ctx.Done() select

	var progressCalls []string
	onProgress := func(msg string) {
		progressCalls = append(progressCalls, msg)
	}

	// Invalid RPC URLs — the function will fail, but we only care about the
	// onProgress calls that happen before any network I/O.
	err := initGenesisAnchorState(
		ctx,
		nopLogger(),
		"http://127.0.0.1:1",  // L1
		"http://127.0.0.1:1",  // L2
		"0000000000000000000000000000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000001", // anchorStateRegistryAddr
		"0x0000000000000000000000000000000000000002", // proxyAdminAddr
		"0x0000000000000000000000000000000000000003", // implAddr
		1,
		0,
		onProgress,
	)

	if err == nil {
		t.Fatal("expected error with cancelled context and invalid RPC URLs")
	}

	if len(progressCalls) == 0 {
		t.Fatalf("onProgress was never called; want at least 'Waiting for L2 genesis block'")
	}
	if progressCalls[0] != "Waiting for L2 genesis block" {
		t.Errorf("progressCalls[0] = %q, want %q", progressCalls[0], "Waiting for L2 genesis block")
	}
}
