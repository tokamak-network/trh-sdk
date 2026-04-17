package thanos

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// DRBRegular holds a single Regular operator's deterministically-derived keys.
type DRBRegular struct {
	Index      int            // 1, 2, 3
	PrivateKey string         // Hex string (without 0x prefix)
	Address    common.Address // Derived from PrivateKey
	PeerID     string         // libp2p Ed25519 peer ID (string form)
}

// DRBAccounts holds all deterministically-derived DRB accounts.
type DRBAccounts struct {
	LeaderPrivateKey string         // Reuses admin key (index 0)
	LeaderEOA        common.Address // Derived from LeaderPrivateKey
	LeaderPeerID     string         // libp2p Ed25519 peer ID
	Regulars         [3]DRBRegular
}

// DeriveDRBAccounts derives all DRB accounts deterministically from a mnemonic.
// Returns DRBAccounts containing Leader + 3 Regular operator keys and peer IDs.
func DeriveDRBAccounts(mnemonic string) (*DRBAccounts, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	// Derive leader peer ID (reuses admin key index 0)
	leaderPeerID, _, err := DerivePeerID(mnemonic, "leader")
	if err != nil {
		return nil, fmt.Errorf("failed to derive leader peer ID: %w", err)
	}

	// Get leader private key and address (index 0)
	leaderPrivKey, leaderAddr, err := getAccountFromIndex(mnemonic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive leader account: %w", err)
	}

	accounts := &DRBAccounts{
		LeaderPrivateKey: leaderPrivKey,
		LeaderEOA:        leaderAddr,
		LeaderPeerID:     leaderPeerID,
	}

	// Derive 3 Regular operators at BIP44 indices 5, 6, 7
	for i, index := range []int{5, 6, 7} {
		privKey, addr, err := getAccountFromIndex(mnemonic, index)
		if err != nil {
			return nil, fmt.Errorf("failed to derive regular %d account at index %d: %w", i+1, index, err)
		}

		role := fmt.Sprintf("regular-%d", i+1)
		peerID, _, err := DerivePeerID(mnemonic, role)
		if err != nil {
			return nil, fmt.Errorf("failed to derive regular %d peer ID: %w", i+1, err)
		}

		accounts.Regulars[i] = DRBRegular{
			Index:      i + 1,
			PrivateKey: privKey,
			Address:    addr,
			PeerID:     peerID,
		}
	}

	return accounts, nil
}

// getAccountFromIndex derives a BIP44 account at a specific index.
// Path: m/44'/60'/0'/0/{index}
func getAccountFromIndex(mnemonic string, index int) (string, common.Address, error) {
	seed := bip39.NewSeed(mnemonic, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return "", common.Address{}, fmt.Errorf("master key: %w", err)
	}

	hardened := func(i uint32) uint32 {
		return i + 0x80000000
	}

	purposeKey, err := masterKey.NewChildKey(hardened(44))
	if err != nil {
		return "", common.Address{}, fmt.Errorf("purpose key: %w", err)
	}

	coinTypeKey, err := purposeKey.NewChildKey(hardened(60)) // 60 = Ethereum
	if err != nil {
		return "", common.Address{}, fmt.Errorf("coin type key: %w", err)
	}

	accountKey, err := coinTypeKey.NewChildKey(hardened(0))
	if err != nil {
		return "", common.Address{}, fmt.Errorf("account key: %w", err)
	}

	changeKey, err := accountKey.NewChildKey(0) // 0 = external
	if err != nil {
		return "", common.Address{}, fmt.Errorf("change key: %w", err)
	}

	childKey, err := changeKey.NewChildKey(uint32(index))
	if err != nil {
		return "", common.Address{}, fmt.Errorf("child key at index %d: %w", index, err)
	}

	// Extract ECDSA private key
	childPrivKey, err := crypto.HexToECDSA(fmt.Sprintf("%064x", childKey.Key))
	if err != nil {
		return "", common.Address{}, fmt.Errorf("ECDSA parse: %w", err)
	}

	// Derive address from private key
	childAddr := crypto.PubkeyToAddress(childPrivKey.PublicKey)

	// Return hex string (no 0x prefix, matches Phase 6 convention)
	privKeyHex := fmt.Sprintf("%064x", childKey.Key)

	return privKeyHex, childAddr, nil
}
