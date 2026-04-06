package thanos

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
	// Plan 02에서 구현
	return nil, fmt.Errorf("not yet implemented")
}

// waitForContractCode polls L2 for contract deployment confirmation via eth_getCode.
// Returns nil when the contract is deployed (code length > 0) or error on timeout.
// Per D-04: used for creation deposit tx verification.
func waitForContractCode(ctx context.Context, l2Client *ethclient.Client, addr common.Address, logger *zap.SugaredLogger) error {
	return fmt.Errorf("not yet implemented")
}

// verifyDepositCallEffect checks that a function-call deposit tx actually executed on L2
// by calling a view function on the target contract to verify state change.
// Per D-04: used for non-creation deposit tx verification.
func verifyDepositCallEffect(ctx context.Context, l2Client *ethclient.Client, contractAddr common.Address, checkCalldata []byte, logger *zap.SugaredLogger) error {
	return fmt.Errorf("not yet implemented")
}
