package thanos

import (
	"testing"
)

// TestBuildDefaultMonitoringInput verifies that the non-interactive monitoring
// config builder produces a valid config suitable for preset auto-deployment.
func TestBuildDefaultMonitoringInput(t *testing.T) {
	input, err := BuildDefaultMonitoringInput()
	if err != nil {
		t.Fatalf("BuildDefaultMonitoringInput() returned error: %v", err)
	}

	// Admin password must be non-empty and hex-encoded (32 hex chars = 16 bytes)
	if len(input.AdminPassword) != 32 {
		t.Errorf("AdminPassword length: got %d, want 32 (16 bytes hex-encoded)", len(input.AdminPassword))
	}
	for _, c := range input.AdminPassword {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("AdminPassword is not lowercase hex: %q", input.AdminPassword)
			break
		}
	}

	// Logging must be enabled by default
	if !input.LoggingEnabled {
		t.Error("LoggingEnabled: got false, want true")
	}

	// AlertManager must be zero-value (alerts disabled — configure via UI post-deploy)
	am := input.AlertManager
	if am.Telegram.Enabled {
		t.Errorf("AlertManager.Telegram.Enabled should be false, got %+v", am.Telegram)
	}
	if am.Email.Enabled {
		t.Errorf("AlertManager.Email.Enabled should be false, got %+v", am.Email)
	}
}

// TestBuildDefaultMonitoringInput_Randomness verifies that successive calls
// produce distinct passwords (not hardcoded or seeded with fixed value).
func TestBuildDefaultMonitoringInput_Randomness(t *testing.T) {
	a, err := BuildDefaultMonitoringInput()
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildDefaultMonitoringInput()
	if err != nil {
		t.Fatal(err)
	}
	if a.AdminPassword == b.AdminPassword {
		t.Error("Two consecutive calls returned identical passwords — randomness may be broken")
	}
}

func TestBuildDefaultBlockExplorerInput(t *testing.T) {
	input, err := BuildDefaultBlockExplorerInput()
	if err != nil {
		t.Fatalf("BuildDefaultBlockExplorerInput() returned error: %v", err)
	}

	if input.DatabaseUsername != "blockscout" {
		t.Errorf("DatabaseUsername: got %q, want %q", input.DatabaseUsername, "blockscout")
	}

	if len(input.DatabasePassword) != 32 {
		t.Errorf("DatabasePassword length: got %d, want 32 (16 bytes hex-encoded)", len(input.DatabasePassword))
	}
	for _, c := range input.DatabasePassword {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("DatabasePassword is not lowercase hex: %q", input.DatabasePassword)
			break
		}
	}

	if input.CoinmarketcapKey != "" {
		t.Errorf("CoinmarketcapKey: got %q, want empty string", input.CoinmarketcapKey)
	}
	if input.CoinmarketcapTokenID != "" {
		t.Errorf("CoinmarketcapTokenID: got %q, want empty string", input.CoinmarketcapTokenID)
	}
	if input.WalletConnectProjectID != "" {
		t.Errorf("WalletConnectProjectID: got %q, want empty string", input.WalletConnectProjectID)
	}
}

func TestBuildDefaultBlockExplorerInput_Randomness(t *testing.T) {
	a, err := BuildDefaultBlockExplorerInput()
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildDefaultBlockExplorerInput()
	if err != nil {
		t.Fatal(err)
	}
	if a.DatabasePassword == b.DatabasePassword {
		t.Error("Two consecutive calls returned identical passwords — randomness may be broken")
	}
}
