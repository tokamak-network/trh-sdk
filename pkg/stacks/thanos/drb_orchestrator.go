package thanos

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

var executeDRBCommand = utils.ExecuteCommand

// DRBRegular holds a single Regular operator's deterministically-derived keys.
type DRBRegular struct {
	Index       int            // 1, 2, 3
	PrivateKey  string         // Hex string (without 0x prefix)
	Address     common.Address // Derived from PrivateKey
	PeerID      string         // libp2p Ed25519 peer ID (string form)
	PeerIDBytes []byte         // libp2p Ed25519 peer ID as protobuf bytes (for volume injection)
}

// DRBAccounts holds all deterministically-derived DRB accounts.
type DRBAccounts struct {
	LeaderPrivateKey  string         // Reuses admin key (index 0)
	LeaderEOA         common.Address // Derived from LeaderPrivateKey
	LeaderPeerID      string         // libp2p Ed25519 peer ID (string form)
	LeaderPeerIDBytes []byte         // libp2p Ed25519 peer ID as protobuf bytes (for volume injection)
	Regulars          [3]DRBRegular
}

// DeriveDRBAccounts derives all DRB accounts deterministically from a mnemonic.
// Returns DRBAccounts containing Leader + 3 Regular operator keys and peer IDs.
func DeriveDRBAccounts(mnemonic string) (*DRBAccounts, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	// Derive leader peer ID (reuses admin key index 0)
	leaderPeerID, leaderPeerIDBytes, err := DerivePeerID(mnemonic, "leader")
	if err != nil {
		return nil, fmt.Errorf("failed to derive leader peer ID: %w", err)
	}

	// Get leader private key and address (index 0)
	leaderPrivKey, leaderAddr, err := getAccountFromIndex(mnemonic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive leader account: %w", err)
	}

	accounts := &DRBAccounts{
		LeaderPrivateKey:  leaderPrivKey,
		LeaderEOA:         leaderAddr,
		LeaderPeerID:      leaderPeerID,
		LeaderPeerIDBytes: leaderPeerIDBytes,
	}

	// Derive 3 Regular operators at BIP44 indices 5, 6, 7
	for i, index := range []int{5, 6, 7} {
		privKey, addr, err := getAccountFromIndex(mnemonic, index)
		if err != nil {
			return nil, fmt.Errorf("failed to derive regular %d account at index %d: %w", i+1, index, err)
		}

		role := fmt.Sprintf("regular-%d", i+1)
		peerID, peerIDBytes, err := DerivePeerID(mnemonic, role)
		if err != nil {
			return nil, fmt.Errorf("failed to derive regular %d peer ID: %w", i+1, err)
		}

		accounts.Regulars[i] = DRBRegular{
			Index:       i + 1,
			PrivateKey:  privKey,
			Address:     addr,
			PeerID:      peerID,
			PeerIDBytes: peerIDBytes,
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
	childPrivKey, err := ethcrypto.HexToECDSA(fmt.Sprintf("%064x", childKey.Key))
	if err != nil {
		return "", common.Address{}, fmt.Errorf("ECDSA parse: %w", err)
	}

	// Derive address from private key
	childAddr := ethcrypto.PubkeyToAddress(childPrivKey.PublicKey)

	// Return hex string (no 0x prefix, matches Phase 6 convention)
	privKeyHex := fmt.Sprintf("%064x", childKey.Key)

	return privKeyHex, childAddr, nil
}

// BootstrapDRBPeerIDFiles injects Leader and Regular peer ID binary files into their
// respective static-key Docker volumes. This must be called after postgres containers
// are healthy but before DRB nodes attempt to start (they expect these files to exist).
//
// Uses a temporary alpine container to write base64-encoded data into the volume,
// avoiding shell escaping issues and working correctly in DinD environments.
func BootstrapDRBPeerIDFiles(ctx context.Context, composeProject string, accounts *DRBAccounts) error {
	const helperName = "drb-peer-id-init"

	// Remove any stale helper container from a previous run.
	_, _ = executeDRBCommand(ctx, "docker", "rm", "-f", helperName)

	// Start temporary alpine container with volumes mounted
	volumeFlags := []string{"-v", composeProject + "_drb-leader-keys:/peer-id-leader"}
	for _, regular := range accounts.Regulars {
		volumeFlags = append(volumeFlags, "-v", fmt.Sprintf("%s_drb-regular-%d-keys:/peer-id-regular-%d",
			composeProject, regular.Index, regular.Index))
	}

	allArgs := append(
		[]string{"run", "-d", "--name", helperName},
		append(volumeFlags, "alpine", "sleep", "infinity")...,
	)
	containerIDOutput, err := executeDRBCommand(ctx, "docker", allArgs...)
	if err != nil {
		return fmt.Errorf("failed to start peer ID helper container: %w", err)
	}
	containerID := extractLastLine(containerIDOutput)
	defer executeDRBCommand(ctx, "docker", "rm", "-f", containerID)

	// Write Leader peer ID binary file
	leaderPeerIDB64 := base64.StdEncoding.EncodeToString(accounts.LeaderPeerIDBytes)
	cmd := fmt.Sprintf("echo %s | base64 -d > /peer-id-leader/leadernode.bin", leaderPeerIDB64)
	if _, err := executeDRBCommand(ctx, "docker", "exec", containerID, "sh", "-c", cmd); err != nil {
		return fmt.Errorf("failed to write leader peer ID file: %w", err)
	}

	// Write Regular peer ID binary files
	for _, regular := range accounts.Regulars {
		regularPeerIDB64 := base64.StdEncoding.EncodeToString(regular.PeerIDBytes)
		cmd := fmt.Sprintf("echo %s | base64 -d > /peer-id-regular-%d/regularnode.bin",
			regularPeerIDB64, regular.Index)
		if _, err := executeDRBCommand(ctx, "docker", "exec", containerID, "sh", "-c", cmd); err != nil {
			return fmt.Errorf("failed to write regular %d peer ID file: %w", regular.Index, err)
		}
	}

	return nil
}

// extractLastLine extracts the last non-empty line from a string.
// Used to parse container ID from docker run -d output, which may include pull progress.
func extractLastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return strings.TrimSpace(s)
}
