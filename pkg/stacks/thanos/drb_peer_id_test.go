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

	// Protobuf marshaled Ed25519 is typically 36-68 bytes depending on libp2p version
	// The format is a protobuf message with type info, so larger than raw key bytes
	require.GreaterOrEqual(t, len(bytes1), 36, "marshaled peer ID too short")

	// Must be binary (not ASCII hex string which would be 64+ bytes for raw key, but we check format)
	// Just verify it's a reasonable binary size
	require.Greater(t, len(bytes1), 0, "marshaled bytes should not be empty")
}

func TestDerivePeerID_InvalidMnemonicReturnsError(t *testing.T) {
	_, _, err := DerivePeerID("invalid mnemonic words", "leader")
	require.Error(t, err)
}
