package thanos

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// uniswapV3TwapOracleBytecode is the compiled creation bytecode of UniswapV3TwapOracle.sol.
// Source: tokamak-thanos/packages/tokamak/contracts-bedrock/src/AA/UniswapV3TwapOracle.sol
// Compiled with forge (solc 0.8.25, via-ir=false).
// Constructor: (address pool, address wton, uint8 feeTokenDecimals, address owner)
//
// Regenerate:
//
//	cd tokamak-thanos/packages/tokamak/contracts-bedrock
//	forge build --contracts src/AA/UniswapV3TwapOracle.sol
//	cat forge-artifacts/UniswapV3TwapOracle.sol/UniswapV3TwapOracle.json | \
//	  python3 -c "import json,sys; print(json.load(sys.stdin)['bytecode']['object'][2:])"
const uniswapV3TwapOracleBytecode = "60c060405234801561000f575f5ffd5b506040516110f13803806110f183398101604081905261002e916101a8565b6001600160a01b0383166100895760405162461bcd60e51b815260206004820152601e60248201527f556e69737761705633547761704f7261636c653a207a65726f2077746f6e000060448201526064015b60405180910390fd5b5f8260ff1611801561009f575060128260ff1611155b6100f95760405162461bcd60e51b815260206004820152602560248201527f556e69737761705633547761704f7261636c653a20696e76616c696420646563604482015264696d616c7360d81b6064820152608401610080565b6001600160a01b03811661014f5760405162461bcd60e51b815260206004820152601f60248201527f556e69737761705633547761704f7261636c653a207a65726f206f776e6572006044820152606401610080565b5f80546001600160a01b039586166001600160a01b03199182161790915592841660805260ff90911660a05260018054919093169116179055610200565b80516001600160a01b03811681146101a3575f5ffd5b919050565b5f5f5f5f608085870312156101bb575f5ffd5b6101c48561018d565b93506101d26020860161018d565b9250604085015160ff811681146101e7575f5ffd5b91506101f56060860161018d565b905092959194509250565b60805160a051610ebc6102355f395f818160d30152818161085e015261088d01525f818161015a01526106d60152610ebc5ff3fe608060405234801561000f575f5ffd5b506004361061009b575f3560e01c80638d62d949116100635780638d62d949146101555780638da5cb5b1461017c57806398d5fdca1461018f578063d0b06f5d146101a5578063f2fde38b146101ab575f5ffd5b806116f0115b1461009f57806336a2982f146100ce5780634437152a146101075780636cc42c671461011c5780637ca2518414610137575b5f5ffd5b5f546100b1906001600160a01b031681565b6040516001600160a01b0390911681526020015b60405180910390f35b6100f57f000000000000000000000000000000000000000000000000000000000000000081565b60405160ff90911681526020016100c5565b61011a6101153660046109ae565b6101be565b005b61012460c881565b60405160029190910b81526020016100c5565b61014061070881565b60405163ffffffff90911681526020016100c5565b6100b17f000000000000000000000000000000000000000000000000000000000000000081565b6001546100b1906001600160a01b031681565b610197610276565b6040519081526020016100c5565b42610197565b61011a6101b93660046109ae565b610442565b6001546001600160a01b0316331461021d5760405162461bcd60e51b815260206004820152601f60248201527f556e69737761705633547761704f7261636c653a206f6e6c79206f776e65720060448201526064015b60405180910390fd5b5f80546040516001600160a01b03808516939216917f90affc163f1a2dfedcd36aa02ed992eeeba8100a4014f0b4cdc20ea265a6662791a35f80546001600160a01b0319166001600160a01b0392909216919091179055565b5f80546001600160a01b03166102d85760405162461bcd60e51b815260206004820152602160248201527f556e69737761705633547761704f7261636c653a20706f6f6c206e6f742073656044820152601d60fa1b6064820152608401610214565b5f5f5f5f5f9054906101000a90046001600160a01b03166001600160a01b0316633850c7bd6040518163ffffffff1660e01b815260040160e060405180830381865afa15801561032a573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061034e91906109df565b505050935050925092505f836001600160a01b0316116103c25760405162461bcd60e51b815260206004820152602960248201527f556e69737761705633547761704f7261636c653a20706f6f6c206e6f7420696e6044820152681a5d1a585b1a5e995960ba1b6064820152608401610214565b60018161ffff161115610431575f5f6103d961054d565b91509150811561042e575f8460020b8260020b13610400576103fb8286610a8f565b61040a565b61040a8583610a8f565b905060c8600282900b1361042c57610421866106d2565b965050505050505090565b505b50505b61043a836106d2565b935050505090565b6001546001600160a01b0316331461049c5760405162461bcd60e51b815260206004820152601f60248201527f556e69737761705633547761704f7261636c653a206f6e6c79206f776e6572006044820152606401610214565b6001600160a01b0381166104f25760405162461bcd60e51b815260206004820152601f60248201527f556e69737761705633547761704f7261636c653a207a65726f206f776e6572006044820152606401610214565b6001546040516001600160a01b038084169216907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0905f90a3600180546001600160a01b0319166001600160a01b0392909216919091179055565b6040805160028082526060820183525f92839283929091602083019080368337019050509050610708815f8151811061058857610588610ace565b602002602001019063ffffffff16908163ffffffff16815250505f816001815181106105b6576105b6610ace565b63ffffffff909216602092830291909101909101525f5460405163883bdbfd60e01b81526001600160a01b039091169063883bdbfd906105fa908490600401610ae2565b5f60405180830381865afa92505050801561063657506040513d5f823e601f3d908101601f191682016040526106339190810190610bf1565b60015b61064457505f928392509050565b5f825f8151811061065757610657610ace565b60200260200101518360018151811061067257610672610ace565b60200260200101516106849190610cbd565b905061069261070882610cfe565b94505f8160060b1280156106b357506106ad61070882610d3a565b60060b15155b156106c657846106c281610d5b565b9550505b60019550505050509091565b5f5f7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03165f5f9054906101000a90046001600160a01b03166001600160a01b0316630dfe16816040518163ffffffff1660e01b8152600401602060405180830381865afa15801561074d573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906107719190610d7c565b6001600160a01b03161490505f6fffffffffffffffffffffffffffffffff8016846001600160a01b0316116107fa575f6107b46001600160a01b03861680610d97565b905082156107da576107d381670de0b6b3a7640000600160c01b6108d8565b91506107f4565b6107f1600160c01b670de0b6b3a7640000836108d8565b91505b5061085a565b5f6108186001600160a01b03861680680100000000000000006108d8565b9050821561083e5761083781670de0b6b3a7640000600160801b6108d8565b9150610858565b610855600160801b670de0b6b3a7640000836108d8565b91505b505b60127f000000000000000000000000000000000000000000000000000000000000000060ff1610156108d1576108b460ff7f0000000000000000000000000000000000000000000000000000000000000000166012610dae565b6108bf90600a610ea4565b6108c99082610d97565b949350505050565b9392505050565b5f838302815f1985870982811083820303915050805f0361090c5783828161090257610902610cea565b04925050506108d1565b80841161092c5760405163227bc15360e01b815260040160405180910390fd5b5f848688095f868103871696879004966002600389028118808a02820302808a02820302808a02820302808a02820302808a02820302808a02909103029181900381900460010186841190950394909402919094039290920491909117919091029150509392505050565b6001600160a01b03811681146109ab575f5ffd5b50565b5f602082840312156109be575f5ffd5b81356108d181610997565b805161ffff811681146109da575f5ffd5b919050565b5f5f5f5f5f5f5f60e0888a0312156109f5575f5ffd5b8751610a0081610997565b8097505060208801518060020b8114610a17575f5ffd5b9550610a25604089016109c9565b9450610a33606089016109c9565b9350610a41608089016109c9565b925060a088015160ff81168114610a56575f5ffd5b60c08901519092508015158114610a6b575f5ffd5b8091505092959891949750929550565b634e487b7160e01b5f52601160045260245ffd5b600282810b9082900b03627fffff198112627fffff82131715610ab457610ab4610a7b565b92915050565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b602080825282518282018190525f918401906040840190835b81811015610b1f57835163ffffffff16835260209384019390920191600101610afb565b509095945050505050565b604051601f8201601f1916810167ffffffffffffffff81118282101715610b5357610b53610aba565b604052919050565b5f67ffffffffffffffff821115610b7457610b74610aba565b5060051b60200190565b5f82601f830112610b8d575f5ffd5b8151610ba0610b9b82610b5b565b610b2a565b8082825260208201915060208360051b860101925085831115610bc1575f5ffd5b602085015b83811015610be7578051610bd981610997565b835260209283019201610bc6565b5095945050505050565b5f5f60408385031215610c02575f5ffd5b825167ffffffffffffffff811115610c18575f5ffd5b8301601f81018513610c28575f5ffd5b8051610c36610b9b82610b5b565b8082825260208201915060208360051b850101925087831115610c57575f5ffd5b6020840193505b82841015610c875783518060060b8114610c76575f5ffd5b825260209384019390910190610c5e565b80955050505050602083015167ffffffffffffffff811115610ca7575f5ffd5b610cb385828601610b7e565b9150509250929050565b600682810b9082900b03667fffffffffffff198112667fffffffffffff82131715610ab457610ab4610a7b565b634e487b7160e01b5f52601260045260245ffd5b5f8160060b8360060b80610d1457610d14610cea565b667fffffffffffff1982145f1982141615610d3157610d31610a7b565b90059392505050565b5f8260060b80610d4c57610d4c610cea565b808360060b0791505092915050565b5f8160020b627fffff198103610d7357610d73610a7b565b5f190192915050565b5f60208284031215610d8c575f5ffd5b81516108d181610997565b8082028115828204841417610ab457610ab4610a7b565b81810381811115610ab457610ab4610a7b565b6001815b6001841115610dfc57808504811115610de057610de0610a7b565b6001841615610dee57908102905b60019390931c928002610dc5565b935093915050565b5f82610e1257506001610ab4565b81610e1e57505f610ab4565b8160018114610e345760028114610e3e57610e5a565b6001915050610ab4565b60ff841115610e4f57610e4f610a7b565b50506001821b610ab4565b5060208310610133831016604e8410600b8410161715610e7d575081810a610ab4565b610e895f198484610dc1565b805f1904821115610e9c57610e9c610a7b565b029392505050565b5f6108d18383610e04"

// uniswapV3FeeDefault is the Uniswap V3 pool fee tier for TON/feeToken pairs.
// 3000 = 0.3%, the standard tier for most volatile pairs.
const uniswapV3FeeDefault = uint32(3000)

// setupUniswapV3Oracle sets up Phase 2 automated oracle for the AA Paymaster.
// It creates a Uniswap V3 pool (WTON/feeToken), initializes it with the initial price,
// deploys UniswapV3TwapOracle pointing to the pool, and updates MultiTokenPaymaster
// to use the new oracle instead of SimplePriceOracle.
//
// This is called after setupAAPaymaster (Phase 1) completes. Phase 1 leaves the
// paymaster functional with a manually-set SimplePriceOracle price. Phase 2 switches
// to a live pool-derived price.
//
// Parameters:
//   - tokenAddr: L2 fee token address (WETH/USDC/USDT)
//   - markupPct: token markup percentage (unchanged from Phase 1)
//   - decimals:  fee token decimals
//   - initialPrice: Phase 1 initial price (used to initialize the pool)
func (t *ThanosStack) setupUniswapV3Oracle(
	ctx context.Context,
	l2Client *ethclient.Client,
	l2ChainID *big.Int,
	adminAddr common.Address,
	tokenAddr common.Address,
	markupPct uint64,
	decimals uint8,
	initialPrice *big.Int,
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
) error {
	feeToken := t.deployConfig.FeeToken
	t.logger.Infof("🦄 Setting up Uniswap V3 oracle for %s (Phase 2)...", feeToken)

	wton := common.HexToAddress(constants.WTONPredeploy)
	factory := common.HexToAddress(constants.UniswapV3FactoryPredeploy)
	paymaster := common.HexToAddress(constants.MultiTokenPaymasterPredeploy)

	// Step 1: Create Uniswap V3 pool (WTON/feeToken, fee=0.3%).
	poolAddr, err := createOrGetUniswapV3Pool(ctx, l2Client, factory, wton, tokenAddr, uniswapV3FeeDefault, sendTxAndWait, t.logger.Infof)
	if err != nil {
		return fmt.Errorf("failed to create Uniswap V3 pool: %w", err)
	}

	// Step 2: Initialize pool with sqrtPriceX96 derived from initial price.
	sqrtPriceX96, err := computeSqrtPriceX96(initialPrice, wton, tokenAddr, uint8(decimals))
	if err != nil {
		return fmt.Errorf("failed to compute sqrtPriceX96: %w", err)
	}
	if err := initializeUniswapV3Pool(ctx, l2Client, poolAddr, sqrtPriceX96, sendTxAndWait, t.logger.Infof); err != nil {
		return fmt.Errorf("failed to initialize Uniswap V3 pool: %w", err)
	}

	// Step 3: Deploy UniswapV3TwapOracle pointing to the new pool.
	oracleAddr, err := deployUniswapV3TwapOracle(ctx, l2Client, l2ChainID, adminAddr, poolAddr, wton, decimals, sendTxAndWait, t.logger.Infof)
	if err != nil {
		return fmt.Errorf("failed to deploy UniswapV3TwapOracle: %w", err)
	}

	// Step 4: MultiTokenPaymaster.updateTokenConfig(token, uniswapOracle, markup)
	// This replaces the SimplePriceOracle set in Phase 1.
	if err := updatePaymasterOracle(tokenAddr, oracleAddr, markupPct, paymaster, sendTxAndWait, t.logger.Infof); err != nil {
		return fmt.Errorf("failed to update paymaster oracle: %w", err)
	}

	t.logger.Infof("✅ Phase 2 oracle live: %s uses Uniswap V3 pool %s", feeToken, poolAddr.Hex())
	return nil
}

// createOrGetUniswapV3Pool creates a WTON/feeToken pool on Uniswap V3 at fee tier 0.3%.
// If the pool already exists (idempotent re-run), the existing address is returned.
//
// ABI: createPool(address tokenA, address tokenB, uint24 fee) → address pool
func createOrGetUniswapV3Pool(
	ctx context.Context,
	l2Client *ethclient.Client,
	factory, wton, feeToken common.Address,
	fee uint32,
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
	logf func(format string, args ...interface{}),
) (common.Address, error) {
	// Check if pool already exists via getPool(tokenA, tokenB, fee).
	// ABI: getPool(address,address,uint24) → address
	getPoolSel := crypto.Keccak256([]byte("getPool(address,address,uint24)"))[:4]
	calldata := make([]byte, 4+3*32)
	copy(calldata[:4], getPoolSel)
	copy(calldata[16:36], wton.Bytes())
	copy(calldata[48:68], feeToken.Bytes())
	// fee as uint24, right-aligned in 32 bytes
	feeBig := new(big.Int).SetUint64(uint64(fee))
	feeBytes := feeBig.Bytes()
	copy(calldata[100-len(feeBytes):100], feeBytes)

	result, err := l2Client.CallContract(ctx, ethereum.CallMsg{
		To:   &factory,
		Data: calldata,
	}, nil)
	if err == nil && len(result) >= 32 {
		addr := common.BytesToAddress(result[12:32])
		if addr != (common.Address{}) {
			logf("ℹ️  Uniswap V3 WTON/%s pool already exists at %s (reusing)", feeToken.Hex(), addr.Hex())
			return addr, nil
		}
	}

	// Pool does not exist — create it.
	createPoolSel := crypto.Keccak256([]byte("createPool(address,address,uint24)"))[:4]
	createCalldata := make([]byte, 4+3*32)
	copy(createCalldata[:4], createPoolSel)
	copy(createCalldata[16:36], wton.Bytes())
	copy(createCalldata[48:68], feeToken.Bytes())
	copy(createCalldata[100-len(feeBytes):100], feeBytes)

	receipt, err := sendTxAndWait(factory, big.NewInt(0), createCalldata)
	if err != nil {
		return common.Address{}, fmt.Errorf("createPool tx failed: %w", err)
	}

	// Decode PoolCreated event: PoolCreated(address indexed token0, address indexed token1,
	//   uint24 indexed fee, int24 tickSpacing, address pool)
	poolCreatedSig := common.BytesToHash(crypto.Keccak256([]byte("PoolCreated(address,address,uint24,int24,address)")))
	for _, log := range receipt.Logs {
		if log.Address == factory && len(log.Topics) > 0 && log.Topics[0] == poolCreatedSig {
			if len(log.Data) >= 64 {
				// pool address is the second word of non-indexed data (after tickSpacing)
				poolAddr := common.BytesToAddress(log.Data[44:64])
				logf("✅ Uniswap V3 WTON/%s pool created at %s", feeToken.Hex(), poolAddr.Hex())
				return poolAddr, nil
			}
		}
	}
	return common.Address{}, fmt.Errorf("PoolCreated event not found in receipt (tx %s)", receipt.TxHash.Hex())
}

// initializeUniswapV3Pool calls pool.initialize(sqrtPriceX96) to set the initial price.
// If the pool is already initialized (sqrtPriceX96 != 0), this is a no-op.
//
// ABI: initialize(uint160 sqrtPriceX96)
func initializeUniswapV3Pool(
	ctx context.Context,
	l2Client *ethclient.Client,
	poolAddr common.Address,
	sqrtPriceX96 *big.Int,
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
	logf func(format string, args ...interface{}),
) error {
	// Check if already initialized via slot0().sqrtPriceX96 != 0.
	slot0Sel := crypto.Keccak256([]byte("slot0()"))[:4]
	result, err := l2Client.CallContract(ctx, ethereum.CallMsg{
		To:   &poolAddr,
		Data: slot0Sel,
	}, nil)
	if err == nil && len(result) >= 32 {
		existing := new(big.Int).SetBytes(result[:32])
		if existing.Sign() > 0 {
			logf("ℹ️  Uniswap V3 pool %s already initialized (sqrtPriceX96=%s)", poolAddr.Hex(), existing.String())
			return nil
		}
	}

	// ABI: initialize(uint160 sqrtPriceX96)
	initSel := crypto.Keccak256([]byte("initialize(uint160)"))[:4]
	calldata := make([]byte, 36)
	copy(calldata[:4], initSel)
	priceBytes := sqrtPriceX96.Bytes()
	copy(calldata[36-len(priceBytes):36], priceBytes)

	if _, err := sendTxAndWait(poolAddr, big.NewInt(0), calldata); err != nil {
		return fmt.Errorf("pool.initialize failed: %w", err)
	}
	logf("✅ Uniswap V3 pool initialized with sqrtPriceX96=%s", sqrtPriceX96.String())
	return nil
}

// deployUniswapV3TwapOracle deploys UniswapV3TwapOracle.sol on L2 via a CREATE transaction.
// The constructor takes (address pool, address wton, uint8 feeTokenDecimals, address owner).
//
// The oracle is deployed deterministically (nonce-based address), so re-runs at the same
// nonce are idempotent in practice. For true idempotency, check for an existing oracle
// in MultiTokenPaymaster's token config before calling this function.
func deployUniswapV3TwapOracle(
	ctx context.Context,
	l2Client *ethclient.Client,
	l2ChainID *big.Int,
	adminAddr common.Address,
	poolAddr common.Address,
	wton common.Address,
	feeTokenDecimals uint8,
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
	logf func(format string, args ...interface{}),
) (common.Address, error) {
	bytecode, err := hex.DecodeString(uniswapV3TwapOracleBytecode)
	if err != nil {
		return common.Address{}, fmt.Errorf("invalid oracle bytecode: %w", err)
	}

	// ABI-encode constructor args: (address pool, address wton, uint8 feeTokenDecimals, address owner)
	// Layout: [32 pool][32 wton][32 feeTokenDecimals][32 owner]
	args := make([]byte, 4*32)
	copy(args[12:32], poolAddr.Bytes())
	copy(args[44:64], wton.Bytes())
	args[95] = feeTokenDecimals // uint8 right-aligned in 32 bytes
	copy(args[108:128], adminAddr.Bytes())

	initCode := append(bytecode, args...)

	// Predict deployment address: nonce-based CREATE.
	nonce, err := l2Client.PendingNonceAt(ctx, adminAddr)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get nonce: %w", err)
	}
	oracleAddr := crypto.CreateAddress(adminAddr, nonce)

	// Deploy: to=nil signals CREATE; gas price is managed by sendTxAndWait.
	receipt, err := sendTxAndWait(common.Address{}, big.NewInt(0), initCode)
	if err != nil {
		return common.Address{}, fmt.Errorf("UniswapV3TwapOracle deploy failed: %w", err)
	}

	// Verify deployed address matches prediction.
	if receipt.ContractAddress != (common.Address{}) {
		oracleAddr = receipt.ContractAddress
	}

	logf("✅ UniswapV3TwapOracle deployed at %s (pool=%s)", oracleAddr.Hex(), poolAddr.Hex())
	return oracleAddr, nil
}

// updatePaymasterOracle calls MultiTokenPaymaster.updateTokenConfig(token, oracle, markup)
// to replace the Phase 1 SimplePriceOracle with the new UniswapV3TwapOracle.
//
// ABI: updateTokenConfig(address token, address oracle, uint256 markupPercent)
func updatePaymasterOracle(
	tokenAddr common.Address,
	oracleAddr common.Address,
	markupPct uint64,
	paymaster common.Address,
	sendTxAndWait func(common.Address, *big.Int, []byte) (*types.Receipt, error),
	logf func(format string, args ...interface{}),
) error {
	// ABI: updateTokenConfig(address,address,uint256)
	selector := crypto.Keccak256([]byte("updateTokenConfig(address,address,uint256)"))[:4]
	calldata := make([]byte, 4+3*32)
	copy(calldata[:4], selector)
	copy(calldata[16:36], tokenAddr.Bytes())
	copy(calldata[48:68], oracleAddr.Bytes())
	markupBytes := new(big.Int).SetUint64(markupPct).Bytes()
	copy(calldata[100-len(markupBytes):100], markupBytes)

	if _, err := sendTxAndWait(paymaster, big.NewInt(0), calldata); err != nil {
		return fmt.Errorf("updateTokenConfig failed: %w", err)
	}
	logf("✅ MultiTokenPaymaster oracle updated → UniswapV3TwapOracle (markup=%d%%)", markupPct)
	return nil
}

// computeSqrtPriceX96 computes the Uniswap V3 pool initialization price (sqrtPriceX96)
// from the oracle's initial price value.
//
// ITokenPriceOracle price format: "1 TON in feeToken, 18 decimals"
//   - ETH  (18 dec): initialPrice = 5e14 → 0.0005 ETH per 1 TON
//   - USDC (6 dec):  initialPrice = 1.5e18 → 1.5 USDC per 1 TON (18-dec fixed)
//   - USDT (6 dec):  initialPrice = 1.5e18 → 1.5 USDT per 1 TON (18-dec fixed)
//
// sqrtPriceX96 = sqrt(token1_per_token0) * 2^96
// where token0 < token1 by address (Uniswap convention).
//
// If WTON is token0:
//   - price_token1_per_token0 = feeTokenAmount / wtonAmount
//   - In lowest units: (initialPrice / 10^(18-feeDecimals)) / 1 = raw_fee_per_1e18_wton / 1e18
//
// The calculation uses integer square root of (price * 2^192) to get sqrtPriceX96.
func computeSqrtPriceX96(initialPrice *big.Int, wton, feeToken common.Address, feeDecimals uint8) (*big.Int, error) {
	if initialPrice == nil || initialPrice.Sign() <= 0 {
		return nil, fmt.Errorf("initial price must be positive")
	}

	// Determine token ordering (Uniswap sorts by address).
	tonIsToken0 := strings.ToLower(wton.Hex()) < strings.ToLower(feeToken.Hex())

	// Convert oracle price (18-dec fixed) to raw feeToken units per 1e18 WTON wei.
	// rawFee = initialPrice / 10^(18 - feeDecimals)
	// For ETH (18 dec): rawFee = initialPrice (already in wei)
	// For USDC/USDT (6 dec): rawFee = initialPrice / 1e12 (convert 18-dec to 6-dec)
	var rawFee *big.Int
	if feeDecimals < 18 {
		shift := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18-feeDecimals)), nil)
		rawFee = new(big.Int).Div(initialPrice, shift)
	} else {
		rawFee = new(big.Int).Set(initialPrice)
	}
	if rawFee.Sign() == 0 {
		return nil, fmt.Errorf("rawFee is zero after scaling (initialPrice=%s, feeDecimals=%d)", initialPrice, feeDecimals)
	}

	// price_ratio = rawFee / 1e18 (feeToken per WTON, in native dec units)
	// sqrtPriceX96 = sqrt(price_ratio) * 2^96
	//             = sqrt(rawFee / 1e18) * 2^96
	//             = sqrt(rawFee * 2^192 / 1e18)
	//
	// For WTON as token1 (feeToken is token0), invert: price = 1e18 / rawFee.

	var numerator, denominator *big.Int
	q192 := new(big.Int).Lsh(big.NewInt(1), 192) // 2^192

	if tonIsToken0 {
		// price = rawFee / 1e18 → sqrtPriceX96 = sqrt(rawFee * 2^192 / 1e18)
		numerator = new(big.Int).Mul(rawFee, q192)
		denominator = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1e18
	} else {
		// WTON is token1, feeToken is token0
		// price = 1e18 / rawFee → sqrtPriceX96 = sqrt(1e18 * 2^192 / rawFee)
		numerator = new(big.Int).Mul(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), q192)
		denominator = rawFee
	}

	// sqrtPriceX96 = floor(sqrt(numerator / denominator))
	//              = floor(sqrt(numerator) / sqrt(denominator))
	// Use integer sqrt of (numerator * precision^2 / denominator) then divide by precision.
	// We compute: sqrtPriceX96 = isqrt(numerator * 2^192 / denominator) to get 96 extra bits.
	// Since numerator already contains 2^192, we compute isqrt(numerator / denominator).
	sqrtArg := new(big.Int).Div(numerator, denominator)
	sqrtPriceX96 := new(big.Int).Sqrt(sqrtArg)

	if sqrtPriceX96.Sign() == 0 {
		return nil, fmt.Errorf("computed sqrtPriceX96 is zero")
	}
	return sqrtPriceX96, nil
}

