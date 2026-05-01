//go:build live

// Live integration test for AA Paymaster async setup.
// Verifies that alto-bundler becomes available after deployLocalNetwork() returns.
// Run with: go test -v -tags live -run TestAASetupAsync -timeout 60m ./pkg/stacks/thanos/
package thanos

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAASetupAsync(t *testing.T) {
	t.Log("Testing AA Paymaster async setup — waiting for alto-bundler to be ready...")
	waitForAltoBundler(t, 10*time.Minute)
}

func waitForAltoBundler(t *testing.T, maxWait time.Duration) {
	t.Helper()
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp, err := http.Post("http://localhost:4337",
			"application/json",
			strings.NewReader(`{"jsonrpc":"2.0","method":"eth_supportedEntryPoints","params":[],"id":1}`),
		)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			t.Log("✅ alto-bundler ready")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(5 * time.Second)
	}
	t.Errorf("alto-bundler not ready after %v", maxWait)
}
