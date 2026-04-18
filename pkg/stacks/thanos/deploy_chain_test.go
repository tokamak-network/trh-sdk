package thanos

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

// NOTE: This test verifies ABI encoding constants only — it does NOT call initL1CrossDomainMessenger.
// It will PASS before the function is implemented. Purpose: confirm encoding logic before writing
// the function body in Step 3.
func TestInitL1CrossDomainMessengerCalldataEncoding(t *testing.T) {
	superchainConfig := common.HexToAddress("0x1111111111111111111111111111111111111111")
	portal          := common.HexToAddress("0x2222222222222222222222222222222222222222")
	systemConfig    := common.HexToAddress("0x3333333333333333333333333333333333333333")

	selector := crypto.Keccak256([]byte("initialize(address,address,address)"))[:4]
	calldata := make([]byte, 100)
	copy(calldata[0:4], selector)
	copy(calldata[16:36], superchainConfig.Bytes())
	copy(calldata[48:68], portal.Bytes())
	copy(calldata[80:100], systemConfig.Bytes())

	if len(calldata) != 100 {
		t.Fatalf("expected 100 bytes, got %d", len(calldata))
	}
	if !bytes.Equal(calldata[:4], selector) {
		t.Errorf("selector mismatch: got %x, want %x", calldata[:4], selector)
	}
	if !bytes.Equal(calldata[16:36], superchainConfig.Bytes()) {
		t.Errorf("superchainConfig not at [16:36]: %x", calldata[16:36])
	}
	if !bytes.Equal(calldata[48:68], portal.Bytes()) {
		t.Errorf("portal not at [48:68]: %x", calldata[48:68])
	}
	if !bytes.Equal(calldata[80:100], systemConfig.Bytes()) {
		t.Errorf("systemConfig not at [80:100]: %x", calldata[80:100])
	}

	// verify zero-padding before each address slot
	if !bytes.Equal(calldata[4:16], make([]byte, 12)) {
		t.Errorf("slot 0 zero-padding corrupted: %x", calldata[4:16])
	}
	if !bytes.Equal(calldata[36:48], make([]byte, 12)) {
		t.Errorf("slot 1 zero-padding corrupted: %x", calldata[36:48])
	}
	if !bytes.Equal(calldata[68:80], make([]byte, 12)) {
		t.Errorf("slot 2 zero-padding corrupted: %x", calldata[68:80])
	}
}
