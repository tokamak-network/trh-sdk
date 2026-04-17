package thanos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDerivePeerID_Deterministic(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	role := "leader"

	// Invoke DerivePeerID twice, assert outputs are identical
	peerID1, bytes1, err1 := DerivePeerID(mnemonic, role)
	require.NoError(t, err1)

	peerID2, bytes2, err2 := DerivePeerID(mnemonic, role)
	require.NoError(t, err2)

	// Both string and bytes must be identical
	require.Equal(t, peerID1, peerID2, "peer ID string differs on redrive")
	require.Equal(t, bytes1, bytes2, "peer ID bytes differ on redrive")
}

func TestDerivePeerID_DifferentRoles(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	roles := []string{"leader", "regular-1", "regular-2", "regular-3"}

	peerIDs := make(map[string]bool)
	for _, role := range roles {
		peerID, _, err := DerivePeerID(mnemonic, role)
		require.NoError(t, err)
		require.NotContains(t, peerIDs, peerID, "duplicate peer ID for role %s", role)
		peerIDs[peerID] = true
	}
	require.Equal(t, 4, len(peerIDs), "must have 4 unique peer IDs")
}

func TestDerivePeerID_MarshaledBinaryFormat(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	_, bytes1, err := DerivePeerID(mnemonic, "leader")
	require.NoError(t, err)

	// Protobuf marshaled Ed25519 is 36-40 bytes (4-byte length prefix + 32-byte key)
	require.GreaterOrEqual(t, len(bytes1), 36, "marshaled peer ID too short")
	require.LessOrEqual(t, len(bytes1), 40, "marshaled peer ID too long")

	// Must NOT be ASCII hex string (would be 64+ bytes)
	require.True(t, len(bytes1) < 50, "looks like hex string, not protobuf binary")
}

func TestDerivePeerID_InvalidMnemonicReturnsError(t *testing.T) {
	_, _, err := DerivePeerID("invalid mnemonic words", "leader")
	require.Error(t, err)
}
