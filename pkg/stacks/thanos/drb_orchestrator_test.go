package thanos

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDeriveDRBAccounts_Deterministic(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	accounts1, err1 := DeriveDRBAccounts(mnemonic)
	require.NoError(t, err1)

	accounts2, err2 := DeriveDRBAccounts(mnemonic)
	require.NoError(t, err2)

	// All fields must be identical
	require.Equal(t, accounts1.LeaderPeerID, accounts2.LeaderPeerID)
	require.Equal(t, accounts1.LeaderEOA, accounts2.LeaderEOA)
	require.Equal(t, accounts1.LeaderPrivateKey, accounts2.LeaderPrivateKey)

	for i := 0; i < 3; i++ {
		require.Equal(t, accounts1.Regulars[i].Index, accounts2.Regulars[i].Index)
		require.Equal(t, accounts1.Regulars[i].PrivateKey, accounts2.Regulars[i].PrivateKey)
		require.Equal(t, accounts1.Regulars[i].Address, accounts2.Regulars[i].Address)
		require.Equal(t, accounts1.Regulars[i].PeerID, accounts2.Regulars[i].PeerID)
	}
}

func TestDeriveDRBAccounts_BIP44Indices(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	accounts, err := DeriveDRBAccounts(mnemonic)
	require.NoError(t, err)

	// Verify Regular indices
	require.Equal(t, 1, accounts.Regulars[0].Index)
	require.Equal(t, 2, accounts.Regulars[1].Index)
	require.Equal(t, 3, accounts.Regulars[2].Index)

	// Verify addresses are non-zero and unique
	require.NotEqual(t, common.Address{}, accounts.Regulars[0].Address)
	require.NotEqual(t, common.Address{}, accounts.Regulars[1].Address)
	require.NotEqual(t, common.Address{}, accounts.Regulars[2].Address)
	require.NotEqual(t, accounts.Regulars[0].Address, accounts.Regulars[1].Address)
	require.NotEqual(t, accounts.Regulars[1].Address, accounts.Regulars[2].Address)

	// Leader address must differ from all Regulars
	require.NotEqual(t, accounts.LeaderEOA, accounts.Regulars[0].Address)
	require.NotEqual(t, accounts.LeaderEOA, accounts.Regulars[1].Address)
	require.NotEqual(t, accounts.LeaderEOA, accounts.Regulars[2].Address)
}

// Test: Peer ID file bootstrap (Phase 7-02 Wave 1 RED)
// This test will fail during Wave 1 because BootstrapDRBPeerIDFiles is not yet implemented.
// It documents the expected behavior and will pass after Wave 1→2 (GREEN).
func TestBootstrapDRBPeerIDFiles_WritesLeaderAndRegularBinaries(t *testing.T) {
	original := executeDRBCommand
	defer func() { executeDRBCommand = original }()

	accounts := &DRBAccounts{
		LeaderPeerIDBytes: []byte{0x01, 0x02, 0x03},
		Regulars: [3]DRBRegular{
			{Index: 1, PeerIDBytes: []byte{0x11}},
			{Index: 2, PeerIDBytes: []byte{0x22}},
			{Index: 3, PeerIDBytes: []byte{0x33}},
		},
	}

	var calls []string
	executeDRBCommand = func(_ context.Context, name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if len(args) >= 2 && args[0] == "run" && args[1] == "-d" {
			return "helper-container-id\n", nil
		}
		return "", nil
	}

	err := BootstrapDRBPeerIDFiles(context.Background(), "demo", accounts)
	require.NoError(t, err)
	require.Len(t, calls, 7)
	require.Contains(t, calls[1], "demo_drb-leader-keys:/peer-id-leader")
	require.Contains(t, calls[1], "demo_drb-regular-1-keys:/peer-id-regular-1")
	require.Contains(t, calls[1], "demo_drb-regular-2-keys:/peer-id-regular-2")
	require.Contains(t, calls[1], "demo_drb-regular-3-keys:/peer-id-regular-3")
	require.Contains(t, calls[2], "leadernode.bin")
	require.Contains(t, calls[3], "regularnode.bin")
	require.Contains(t, calls[4], "regularnode.bin")
	require.Contains(t, calls[5], "regularnode.bin")
	require.Equal(t, "docker rm -f helper-container-id", calls[6])
}
