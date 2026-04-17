package thanos

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos/bindings/commitreveal2"
)

func activateRegularOperatorsSequentially(
	ctx context.Context,
	accounts *DRBAccounts,
	threshold *big.Int,
	newTransactor func(regular DRBRegular) (*bind.TransactOpts, error),
	submitActivation func(regular DRBRegular, auth *bind.TransactOpts) (*types.Transaction, error),
	waitReceipt func(regular DRBRegular, tx *types.Transaction) (*types.Receipt, error),
) error {
	for _, regular := range accounts.Regulars {
		auth, err := newTransactor(regular)
		if err != nil {
			return fmt.Errorf("create transactor for regular %d: %w", regular.Index, err)
		}

		auth.Value = threshold

		tx, err := submitActivation(regular, auth)
		if err != nil {
			return fmt.Errorf("regular %d depositAndActivate submission failed: %w", regular.Index, err)
		}

		receipt, err := waitReceipt(regular, tx)
		if err != nil {
			return fmt.Errorf("regular %d transaction failed: %w", regular.Index, err)
		}

		if receipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("regular %d transaction reverted: status=%d", regular.Index, receipt.Status)
		}
	}

	return nil
}

// ActivateRegularOperators calls depositAndActivate() for each Regular operator sequentially.
// Must be called after all DRB nodes (Leader + Regular 1/2/3) are healthy and running.
// Activation is sequential (not concurrent) to avoid nonce collisions.
func ActivateRegularOperators(ctx context.Context, rpcURL string, contractAddr string, accounts *DRBAccounts, threshold *big.Int) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("dial RPC %s: %w", rpcURL, err)
	}
	defer client.Close()

	// Load contract binding
	contract, err := commitreveal2.NewCommitReveal2L2(common.HexToAddress(contractAddr), client)
	if err != nil {
		return fmt.Errorf("load CommitReveal2L2 contract: %w", err)
	}

	// Get chain ID for signing
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("get chain ID: %w", err)
	}

	return activateRegularOperatorsSequentially(
		ctx,
		accounts,
		threshold,
		func(regular DRBRegular) (*bind.TransactOpts, error) {
			privKey, err := crypto.HexToECDSA(regular.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("parse regular %d private key: %w", regular.Index, err)
			}

			auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
			if err != nil {
				return nil, fmt.Errorf("create transactor for regular %d: %w", regular.Index, err)
			}

			return auth, nil
		},
		func(regular DRBRegular, auth *bind.TransactOpts) (*types.Transaction, error) {
			return contract.DepositAndActivate(auth)
		},
		func(regular DRBRegular, tx *types.Transaction) (*types.Receipt, error) {
			return bind.WaitMined(ctx, client, tx)
		},
	)
}
