package thanos

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// mockThresholdReader stubs the activationThresholdReader interface for tests.
type mockThresholdReader struct {
	threshold *big.Int
	err       error
}

func (m *mockThresholdReader) SActivationThreshold(opts *bind.CallOpts) (*big.Int, error) {
	return m.threshold, m.err
}

func TestReadActivationThreshold_ReturnsContractValue(t *testing.T) {
	expected := big.NewInt(100_000_000_000_000_000) // 0.1 ETH
	got, err := readActivationThreshold(context.Background(), &mockThresholdReader{threshold: expected})
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestReadActivationThreshold_PropagatesError(t *testing.T) {
	_, err := readActivationThreshold(context.Background(), &mockThresholdReader{err: errors.New("rpc error")})
	require.Error(t, err)
	require.Contains(t, err.Error(), "rpc error")
}

// Test: Sequential activation (Phase 7-02 Wave 1 RED)
// This test will fail during Wave 1 because ActivateRegularOperators is not yet implemented.
// It documents the expected behavior: sequential (not concurrent) transaction submission.
func TestActivateRegularOperators_Sequential(t *testing.T) {
	accounts := &DRBAccounts{
		Regulars: [3]DRBRegular{{Index: 1}, {Index: 2}, {Index: 3}},
	}

	var order []string
	err := activateRegularOperatorsSequentially(
		context.Background(),
		accounts,
		big.NewInt(7),
		func(regular DRBRegular) (*bind.TransactOpts, error) {
			order = append(order, fmt.Sprintf("auth-%d", regular.Index))
			return &bind.TransactOpts{From: common.BigToAddress(big.NewInt(int64(regular.Index)))}, nil
		},
		func(regular DRBRegular, auth *bind.TransactOpts) (*types.Transaction, error) {
			order = append(order, fmt.Sprintf("submit-%d", regular.Index))
			require.Equal(t, big.NewInt(7), auth.Value)
			return types.NewTx(&types.LegacyTx{Nonce: uint64(regular.Index)}), nil
		},
		func(regular DRBRegular, _ *types.Transaction) (*types.Receipt, error) {
			order = append(order, fmt.Sprintf("wait-%d", regular.Index))
			return &types.Receipt{Status: types.ReceiptStatusSuccessful}, nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, []string{
		"auth-1", "submit-1", "wait-1",
		"auth-2", "submit-2", "wait-2",
		"auth-3", "submit-3", "wait-3",
	}, order)
}

// Test: Error handling in activation (Phase 7-02 Wave 1 RED)
// This test will fail during Wave 1 because ActivateRegularOperators is not yet implemented.
// It documents the expected behavior: errors from contract calls are wrapped and propagated.
func TestActivateRegularOperators_ErrorHandling(t *testing.T) {
	accounts := &DRBAccounts{
		Regulars: [3]DRBRegular{{Index: 1}, {Index: 2}, {Index: 3}},
	}

	err := activateRegularOperatorsSequentially(
		context.Background(),
		accounts,
		big.NewInt(3),
		func(regular DRBRegular) (*bind.TransactOpts, error) {
			return &bind.TransactOpts{From: common.BigToAddress(big.NewInt(int64(regular.Index)))}, nil
		},
		func(regular DRBRegular, _ *bind.TransactOpts) (*types.Transaction, error) {
			if regular.Index == 2 {
				return nil, errors.New("boom")
			}
			return types.NewTx(&types.LegacyTx{Nonce: uint64(regular.Index)}), nil
		},
		func(regular DRBRegular, _ *types.Transaction) (*types.Receipt, error) {
			return &types.Receipt{Status: types.ReceiptStatusSuccessful}, nil
		},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "regular 2 depositAndActivate submission failed")
	require.Contains(t, err.Error(), "boom")
}
