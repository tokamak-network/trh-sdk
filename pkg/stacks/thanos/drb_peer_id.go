package thanos

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/tyler-smith/go-bip39"
)

// DerivePeerID derives a libp2p Ed25519 peer ID deterministically from a mnemonic and role.
// Seed is computed as sha256(mnemonic + "|drb-peer-id-v1|" + role) to ensure reproducibility.
// Returns the peer ID as a string, the marshaled private key as protobuf bytes, and any error.
//
// Roles: "leader", "regular-1", "regular-2", "regular-3"
func DerivePeerID(mnemonic, role string) (string, []byte, error) {
	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return "", nil, fmt.Errorf("invalid mnemonic")
	}

	// Deterministic seed from hash of mnemonic + role
	h := sha256.New()
	h.Write([]byte(mnemonic + "|drb-peer-id-v1|" + role))
	seed := h.Sum(nil) // 32 bytes

	// Generate Ed25519 key from deterministic reader
	reader := bytes.NewReader(seed)
	privKey, _, err := crypto.GenerateEd25519Key(reader)
	if err != nil {
		return "", nil, fmt.Errorf("Ed25519 key generation failed: %w", err)
	}

	// Derive peer ID string from private key
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return "", nil, fmt.Errorf("peer ID from key failed: %w", err)
	}

	// Marshal private key to protobuf bytes for storage in volume
	peerIDBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return "", nil, fmt.Errorf("marshal peer ID private key failed: %w", err)
	}

	return peerID.String(), peerIDBytes, nil
}
