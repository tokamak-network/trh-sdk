package thanos

import (
	"context"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"go.uber.org/zap"
)

func makePresetStack(t *testing.T, preset, feeToken string, k8s *types.K8sConfig) *ThanosStack {
	t.Helper()
	return &ThanosStack{
		deployConfig: &types.Config{
			Preset:   preset,
			FeeToken: feeToken,
			K8s:      k8s,
		},
		deploymentPath: t.TempDir(),
		logger:         zap.NewNop().Sugar(),
	}
}

// --- Group A: Preset × K8s nil routing ---

// TestInstallPresetModules_General_LocalNoError verifies that General preset
// does not make any external calls in installPresetModules — only hint logs.
// This is a regression guard: General preset must never acquire K8s/network deps.
func TestInstallPresetModules_General_LocalNoError(t *testing.T) {
	stack := makePresetStack(t, constants.PresetGeneral, constants.FeeTokenTON, nil)
	err := stack.installPresetModules(context.Background())
	if err != nil {
		t.Errorf("General preset on local should return nil, got: %v", err)
	}
}

// TestInstallPresetModules_DeFi_LocalErrors verifies that DeFi preset returns an
// error on local (K8s nil) because uptimeService and monitoring require K8s.
func TestInstallPresetModules_DeFi_LocalErrors(t *testing.T) {
	stack := makePresetStack(t, constants.PresetDeFi, constants.FeeTokenTON, nil)
	err := stack.installPresetModules(context.Background())
	if err == nil {
		t.Fatal("DeFi preset on local should return error (K8s required), got nil")
	}
}

// TestInstallPresetModules_Gaming_LocalErrors verifies that Gaming preset returns
// an error on local because uptimeService and drb require K8s.
func TestInstallPresetModules_Gaming_LocalErrors(t *testing.T) {
	stack := makePresetStack(t, constants.PresetGaming, constants.FeeTokenTON, nil)
	err := stack.installPresetModules(context.Background())
	if err == nil {
		t.Fatal("Gaming preset on local should return error (K8s required), got nil")
	}
}

// TestInstallPresetModules_Full_LocalErrors verifies that Full preset returns an
// error on local because multiple modules require K8s.
func TestInstallPresetModules_Full_LocalErrors(t *testing.T) {
	stack := makePresetStack(t, constants.PresetFull, constants.FeeTokenTON, nil)
	err := stack.installPresetModules(context.Background())
	if err == nil {
		t.Fatal("Full preset on local should return error (K8s required), got nil")
	}
}

// --- Group B: aaPaymaster FeeToken guard ---

// TestInstallPresetModules_Gaming_TONFeeToken_AASkipped verifies that the aaPaymaster
// block is skipped when FeeToken is TON. The final error comes from the DRB module
// (K8s nil), NOT from aaPaymaster attempting an L2 RPC connection.
//
// Execution order: uptimeService → monitoring → drb → [aaPaymaster skipped] → ...
// With TON: installErr = drb error ("K8s configuration is not set")
// With ETH: installErr = aaPaymaster error ("L2 RPC" / "failed to connect")
func TestInstallPresetModules_Gaming_TONFeeToken_AASkipped(t *testing.T) {
	stack := makePresetStack(t, constants.PresetGaming, constants.FeeTokenTON, nil)
	err := stack.installPresetModules(context.Background())
	if err == nil {
		t.Fatal("expected error from K8s-required modules, got nil")
	}
	// Final error must come from DRB (last K8s module before aaPaymaster).
	// If aaPaymaster ran, the error would mention RPC/dial/connection.
	if strings.Contains(strings.ToLower(err.Error()), "rpc") ||
		strings.Contains(strings.ToLower(err.Error()), "dial") ||
		strings.Contains(strings.ToLower(err.Error()), "paymaster") {
		t.Errorf("error suggests aaPaymaster ran despite TON fee token; got: %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "k8s") &&
		!strings.Contains(err.Error(), "K8s") {
		t.Errorf("expected K8s error from DRB module, got: %v", err)
	}
}

// TestInstallPresetModules_Gaming_ETHFeeToken_AAAttempted verifies that the aaPaymaster
// block IS entered when FeeToken is ETH (non-TON). The final error must come from
// setupAAPaymaster (not a K8s nil error), proving the FeeToken guard was passed.
//
// setupAAPaymaster validates the admin private key before dialing L2 RPC, so the
// error in this test context is "invalid admin private key" — that's acceptable.
// The key assertion is that the error is NOT "K8s configuration is not set".
func TestInstallPresetModules_Gaming_ETHFeeToken_AAAttempted(t *testing.T) {
	stack := makePresetStack(t, constants.PresetGaming, constants.FeeTokenETH, nil)
	err := stack.installPresetModules(context.Background())
	if err == nil {
		t.Fatal("expected error from aaPaymaster, got nil")
	}
	// Error must NOT be the K8s nil error — it must come from aaPaymaster itself.
	// (K8s nil error = "K8s configuration is not set", which is from drb/uptime/monitoring)
	if strings.Contains(err.Error(), "K8s configuration is not set") {
		t.Errorf("error should come from aaPaymaster (FeeToken guard passed), not K8s check; got: %v", err)
	}
}
