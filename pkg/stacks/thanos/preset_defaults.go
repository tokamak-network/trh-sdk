package thanos

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// BuildDefaultMonitoringInput returns a non-interactive monitoring configuration
// suitable for preset-based auto-deployment. Alerts are disabled by default;
// logging is enabled. Admin password is randomly generated.
func BuildDefaultMonitoringInput() (*InstallMonitoringInput, error) {
	password, err := generateRandomPassword(16)
	if err != nil {
		return nil, err
	}
	return &InstallMonitoringInput{
		AdminPassword:  password,
		AlertManager:   types.AlertManagerConfig{}, // alerts disabled — configure via UI after deploy
		LoggingEnabled: true,
	}, nil
}

// generateRandomPassword returns a hex-encoded random password of the given byte length.
func generateRandomPassword(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
