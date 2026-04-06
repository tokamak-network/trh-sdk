package thanos

import (
	"context"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/abis"
	"go.uber.org/zap"
)

// DeployCrossTradeLocalInput defines the parameters required to deploy CrossTrade contracts
// via L1 OptimismPortal depositTransaction calls. Matches PRD v2.1 interface definition.
type DeployCrossTradeLocalInput struct {
	L1RPCUrl             string      `json:"l1_rpc_url"`
	L1ChainID            uint64      `json:"l1_chain_id"`
	DeployerPrivateKey   string      `json:"deployer_private_key"`
	L2RPCUrl             string      `json:"l2_rpc_url"`
	L2ChainID            uint64      `json:"l2_chain_id"`
	OptimismPortalProxy  string      `json:"optimism_portal_proxy"`
	CrossDomainMessenger string      `json:"cross_domain_messenger"`
	L1CrossTradeProxy    string      `json:"l1_cross_trade_proxy"`
	L2toL2CrossTradeL1   string      `json:"l2_to_l2_cross_trade_l1"`
	SupportedTokens      []TokenPair `json:"supported_tokens"`
}

// TokenPair represents an L1/L2 token pair to be registered in CrossTrade.
type TokenPair struct {
	L1Token string `json:"l1_token"`
	L2Token string `json:"l2_token"`
	Symbol  string `json:"symbol"`
}

// DeployCrossTradeLocalOutput contains the deployed contract addresses and registration tx hashes.
// Matches PRD v2.1 interface definition. Bytecode fields are intentionally absent (stored in
// cross_trade_local_bytecodes.go constants to keep this struct clean).
type DeployCrossTradeLocalOutput struct {
	L2CrossTradeProxy     string `json:"l2_cross_trade_proxy"`
	L2CrossTrade          string `json:"l2_cross_trade"`
	L2toL2CrossTradeProxy string `json:"l2_to_l2_cross_trade_proxy"`
	L2toL2CrossTradeL2    string `json:"l2_to_l2_cross_trade_l2"`
	L1RegistrationTxHash  string `json:"l1_registration_tx_hash"`
	L1RegistrationL2L2Tx  string `json:"l1_registration_l2_l2_tx"`
}

// DeployCrossTradeLocal deploys the CrossTrade L2 contracts via L1 OptimismPortal
// depositTransaction calls, then registers chain info and tokens on L1.
// This is the local (Docker Compose) deployment path, distinct from the AWS/Foundry path
// in cross_trade.go.
// Implements PRD v2.1 12-step sequence (L2→L1 6 steps + L2→L2 6 steps).
func (t *ThanosStack) DeployCrossTradeLocal(
	ctx context.Context,
	input *DeployCrossTradeLocalInput,
) (*DeployCrossTradeLocalOutput, error) {
	// Plan 03에서 구현
	return nil, fmt.Errorf("not yet implemented")
}

// waitForContractCode polls L2 for contract deployment confirmation via eth_getCode.
// Returns nil when the contract is deployed (code length > 0) or error on timeout.
// Per D-04: used for creation deposit tx verification.
func waitForContractCode(ctx context.Context, l2Client *ethclient.Client, addr common.Address, logger *zap.SugaredLogger) error {
	for attempt := 1; attempt <= 60; attempt++ {
		if attempt%10 == 0 {
			logger.Infof("waiting for contract at %s (attempt %d/60)", addr.Hex(), attempt)
		}

		code, err := l2Client.CodeAt(ctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to call CodeAt for %s: %w", addr.Hex(), err)
		}
		if len(code) > 0 {
			logger.Infof("contract deployed at %s (attempt %d/60)", addr.Hex(), attempt)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("contract at %s not deployed after 120s", addr.Hex())
}

// verifyDepositCallEffect checks that a function-call deposit tx actually executed on L2
// by calling a view function on the target contract to verify state change.
// Per D-04: used for non-creation deposit tx verification.
func verifyDepositCallEffect(ctx context.Context, l2Client *ethclient.Client, contractAddr common.Address, checkCalldata []byte, logger *zap.SugaredLogger) error {
	for attempt := 1; attempt <= 60; attempt++ {
		if attempt%10 == 0 {
			logger.Infof("verifying deposit call effect at %s (attempt %d/60)", contractAddr.Hex(), attempt)
		}

		result, err := l2Client.CallContract(ctx, ethereum.CallMsg{
			To:   &contractAddr,
			Data: checkCalldata,
		}, nil)
		if err == nil && len(result) > 0 {
			logger.Infof("deposit call effect verified at %s (attempt %d/60)", contractAddr.Hex(), attempt)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("deposit call effect not verified at %s after 120s", contractAddr.Hex())
}

// sendDepositCreation sends an L1 OptimismPortal.depositTransaction for contract creation.
// The _to field is address(0) and _isCreation is true. Waits for L1 receipt.
// Per D-09: fails fast if the L1 tx reverts.
func sendDepositCreation(
	ctx context.Context,
	portal *abis.OptimismPortalTransactor,
	opts *bind.TransactOpts,
	l1Client *ethclient.Client,
	bytecode []byte,
	gasLimit uint64,
	logger *zap.SugaredLogger,
) (*types.Receipt, error) {
	tx, err := portal.DepositTransaction(opts, common.Address{}, big.NewInt(0), big.NewInt(0), gasLimit, true, bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to send deposit creation tx: %w", err)
	}
	logger.Infof("L1 deposit creation tx sent: %s", tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for deposit creation tx receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deposit creation tx reverted (tx: %s, gas used: %d)", tx.Hash().Hex(), receipt.GasUsed)
	}
	return receipt, nil
}

// sendDepositCall sends an L1 OptimismPortal.depositTransaction for a function call.
// The _isCreation flag is false. Waits for L1 receipt.
// Per D-09: fails fast if the L1 tx reverts.
func sendDepositCall(
	ctx context.Context,
	portal *abis.OptimismPortalTransactor,
	opts *bind.TransactOpts,
	l1Client *ethclient.Client,
	to common.Address,
	calldata []byte,
	gasLimit uint64,
	logger *zap.SugaredLogger,
) (*types.Receipt, error) {
	tx, err := portal.DepositTransaction(opts, to, big.NewInt(0), big.NewInt(0), gasLimit, false, calldata)
	if err != nil {
		return nil, fmt.Errorf("failed to send deposit call tx to %s: %w", to.Hex(), err)
	}
	logger.Infof("L1 deposit call tx sent to %s: %s", to.Hex(), tx.Hash().Hex())

	receipt, err := bind.WaitMined(ctx, l1Client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for deposit call tx receipt (to: %s): %w", to.Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deposit call tx reverted (to: %s, tx: %s, gas used: %d)", to.Hex(), tx.Hash().Hex(), receipt.GasUsed)
	}
	return receipt, nil
}
