package thanos

import (
	"context"
	"testing"
)

func TestInstallBlockExplorerInput_Validate_EmptyCMCAllowed(t *testing.T) {
	input := &InstallBlockExplorerInput{
		DatabaseUsername: "blockscout",
		DatabasePassword: "validpass123",
	}
	if err := input.Validate(context.Background()); err != nil {
		t.Errorf("empty CMC/WC fields should not cause validation error: %v", err)
	}
}

func TestInstallBlockExplorerInput_Validate_WithCMC(t *testing.T) {
	input := &InstallBlockExplorerInput{
		DatabaseUsername: "blockscout",
		DatabasePassword: "validpass123",
		CoinmarketcapKey: "somekey",
	}
	if err := input.Validate(context.Background()); err != nil {
		t.Errorf("CMC key present should not cause validation error: %v", err)
	}
	if input.CoinmarketcapTokenID == "" {
		t.Error("CoinmarketcapTokenID should be filled with default when key is set but ID is empty")
	}
}

func TestInstallBlockExplorerInput_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		input *InstallBlockExplorerInput
	}{
		{"empty username", &InstallBlockExplorerInput{DatabasePassword: "validpass123"}},
		{"empty password", &InstallBlockExplorerInput{DatabaseUsername: "blockscout"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.input.Validate(context.Background()); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}
