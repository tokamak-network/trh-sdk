package thanos

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

func TestReadPrestateHash(t *testing.T) {
	t.Run("valid prestate json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prestate.json")
		content := `{"pre":"0xabc123def456abc123def456abc123def456abc123def456abc123def456abc123","other":"ignored"}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		hash, err := readPrestateHash(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash != "0xabc123def456abc123def456abc123def456abc123def456abc123def456abc123" {
			t.Errorf("got %q, want expected hash", hash)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readPrestateHash("/nonexistent/prestate.json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty pre field", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prestate.json")
		if err := os.WriteFile(path, []byte(`{"pre":""}`), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readPrestateHash(path)
		if err == nil {
			t.Fatal("expected error for empty pre field")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "prestate.json")
		if err := os.WriteFile(path, []byte(`not json`), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readPrestateHash(path)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestDeployNetworkToAWSFaultProofValidation(t *testing.T) {
	logger := zap.NewNop().Sugar()

	t.Run("fault proof enabled with empty challenger key returns error", func(t *testing.T) {
		stack := &ThanosStack{
			deployConfig: &types.Config{
				EnableFraudProof:     true,
				ChallengerPrivateKey: "",
			},
			logger: logger,
		}
		err := stack.deployNetworkToAWS(context.Background(), &DeployInfraInput{ChainName: "test"})
		if err == nil {
			t.Fatal("expected error when fault proof enabled without challenger key")
		}
		if !strings.Contains(err.Error(), "challenger private key is not set") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("fault proof disabled with empty challenger key does not error on that check", func(t *testing.T) {
		stack := &ThanosStack{
			deployConfig: &types.Config{
				EnableFraudProof:     false,
				ChallengerPrivateKey: "",
			},
			logger: logger,
		}
		// The function will fail on inputs.Validate (L1BeaconURL is required),
		// but should NOT fail due to the challenger key check.
		err := stack.deployNetworkToAWS(context.Background(), &DeployInfraInput{ChainName: "test"})
		if err != nil && strings.Contains(err.Error(), "challenger private key is not set") {
			t.Errorf("should not fail on challenger key check when fault proof disabled: %v", err)
		}
	})
}
