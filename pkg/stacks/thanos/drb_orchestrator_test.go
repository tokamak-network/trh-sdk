package thanos

import (
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
