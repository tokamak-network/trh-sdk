// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abis

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// Layer2ManagerV1MetaData contains all meta data concerning the Layer2ManagerV1 contract.
var Layer2ManagerV1MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"ExcludeError\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"OnApproveError\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"RegisterError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"StatusError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddressError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroBytesError\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"candidateAddOn\",\"type\":\"address\"}],\"name\":\"PausedCandidateAddOn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"wtonAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"memo\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"candidateAddOn\",\"type\":\"address\"}],\"name\":\"RegisteredCandidateAddOn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_l2Register\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_operatorManagerFactory\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_ton\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_wton\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_dao\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_depositManager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_seigManager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_swapProxy\",\"type\":\"address\"}],\"name\":\"SetAddresses\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_minimumInitialDepositAmount\",\"type\":\"uint256\"}],\"name\":\"SetMinimumInitialDepositAmount\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_operatorManagerFactory\",\"type\":\"address\"}],\"name\":\"SetOperatorManagerFactory\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"candidateAddOn\",\"type\":\"address\"}],\"name\":\"UnpausedCandidateAddOn\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MINTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"_checkL1BridgeDetail\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"result\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"l1Bridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"portal\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l2Ton\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bool\",\"name\":\"rejectedSeigs\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"addAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"aliveImplementation\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"availableRegister\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"result\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oper\",\"type\":\"address\"}],\"name\":\"candidateAddOnOfOperator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"checkL1Bridge\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"result\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"l1Bridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"portal\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l2Ton\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"checkL1BridgeDetail\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"result\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"l1Bridge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"portal\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l2Ton\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"uint8\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"bool\",\"name\":\"rejectedSeigs\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"checkLayer2TVL\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"result\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"dao\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"depositManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"isAdmin\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BridgeRegistry\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minimumInitialDepositAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"onApprove\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"operatorInfo\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"candidateAddOn\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"operatorManagerFactory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"operatorOfRollupConfig\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"pauseCandidateAddOn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pauseProxy\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"proxyImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"flagTon\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"memo\",\"type\":\"string\"}],\"name\":\"registerCandidateAddOn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"removeAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"rollupConfigInfo\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"status\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"operatorManager\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_oper\",\"type\":\"address\"}],\"name\":\"rollupConfigOfOperator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"seigManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"selectorImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1BridgeRegistry\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_operatorManagerFactory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_ton\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_wton\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_dao\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_depositManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_seigManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_swapProxy\",\"type\":\"address\"}],\"name\":\"setAddresses\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minimumInitialDepositAmount\",\"type\":\"uint256\"}],\"name\":\"setMinimumInitialDepositAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_operatorManagerFactory\",\"type\":\"address\"}],\"name\":\"setOperatorManagerFactory\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollupConfig\",\"type\":\"address\"}],\"name\":\"statusLayer2\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"swapProxy\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ton\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"transferAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"unpauseCandidateAddOn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"updateSeigniorage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wton\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// Layer2ManagerV1ABI is the input ABI used to generate the binding from.
// Deprecated: Use Layer2ManagerV1MetaData.ABI instead.
var Layer2ManagerV1ABI = Layer2ManagerV1MetaData.ABI

// Layer2ManagerV1 is an auto generated Go binding around an Ethereum contract.
type Layer2ManagerV1 struct {
	Layer2ManagerV1Caller     // Read-only binding to the contract
	Layer2ManagerV1Transactor // Write-only binding to the contract
	Layer2ManagerV1Filterer   // Log filterer for contract events
}

// Layer2ManagerV1Caller is an auto generated read-only Go binding around an Ethereum contract.
type Layer2ManagerV1Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Layer2ManagerV1Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Layer2ManagerV1Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Layer2ManagerV1Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Layer2ManagerV1Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Layer2ManagerV1Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Layer2ManagerV1Session struct {
	Contract     *Layer2ManagerV1  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Layer2ManagerV1CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Layer2ManagerV1CallerSession struct {
	Contract *Layer2ManagerV1Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// Layer2ManagerV1TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Layer2ManagerV1TransactorSession struct {
	Contract     *Layer2ManagerV1Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// Layer2ManagerV1Raw is an auto generated low-level Go binding around an Ethereum contract.
type Layer2ManagerV1Raw struct {
	Contract *Layer2ManagerV1 // Generic contract binding to access the raw methods on
}

// Layer2ManagerV1CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Layer2ManagerV1CallerRaw struct {
	Contract *Layer2ManagerV1Caller // Generic read-only contract binding to access the raw methods on
}

// Layer2ManagerV1TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Layer2ManagerV1TransactorRaw struct {
	Contract *Layer2ManagerV1Transactor // Generic write-only contract binding to access the raw methods on
}

// NewLayer2ManagerV1 creates a new instance of Layer2ManagerV1, bound to a specific deployed contract.
func NewLayer2ManagerV1(address common.Address, backend bind.ContractBackend) (*Layer2ManagerV1, error) {
	contract, err := bindLayer2ManagerV1(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1{Layer2ManagerV1Caller: Layer2ManagerV1Caller{contract: contract}, Layer2ManagerV1Transactor: Layer2ManagerV1Transactor{contract: contract}, Layer2ManagerV1Filterer: Layer2ManagerV1Filterer{contract: contract}}, nil
}

// NewLayer2ManagerV1Caller creates a new read-only instance of Layer2ManagerV1, bound to a specific deployed contract.
func NewLayer2ManagerV1Caller(address common.Address, caller bind.ContractCaller) (*Layer2ManagerV1Caller, error) {
	contract, err := bindLayer2ManagerV1(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1Caller{contract: contract}, nil
}

// NewLayer2ManagerV1Transactor creates a new write-only instance of Layer2ManagerV1, bound to a specific deployed contract.
func NewLayer2ManagerV1Transactor(address common.Address, transactor bind.ContractTransactor) (*Layer2ManagerV1Transactor, error) {
	contract, err := bindLayer2ManagerV1(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1Transactor{contract: contract}, nil
}

// NewLayer2ManagerV1Filterer creates a new log filterer instance of Layer2ManagerV1, bound to a specific deployed contract.
func NewLayer2ManagerV1Filterer(address common.Address, filterer bind.ContractFilterer) (*Layer2ManagerV1Filterer, error) {
	contract, err := bindLayer2ManagerV1(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1Filterer{contract: contract}, nil
}

// bindLayer2ManagerV1 binds a generic wrapper to an already deployed contract.
func bindLayer2ManagerV1(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Layer2ManagerV1MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Layer2ManagerV1 *Layer2ManagerV1Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Layer2ManagerV1.Contract.Layer2ManagerV1Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Layer2ManagerV1 *Layer2ManagerV1Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.Layer2ManagerV1Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Layer2ManagerV1 *Layer2ManagerV1Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.Layer2ManagerV1Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Layer2ManagerV1 *Layer2ManagerV1CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Layer2ManagerV1.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) DEFAULTADMINROLE() ([32]byte, error) {
	return _Layer2ManagerV1.Contract.DEFAULTADMINROLE(&_Layer2ManagerV1.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Layer2ManagerV1.Contract.DEFAULTADMINROLE(&_Layer2ManagerV1.CallOpts)
}

// MINTERROLE is a free data retrieval call binding the contract method 0xd5391393.
//
// Solidity: function MINTER_ROLE() view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) MINTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "MINTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MINTERROLE is a free data retrieval call binding the contract method 0xd5391393.
//
// Solidity: function MINTER_ROLE() view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) MINTERROLE() ([32]byte, error) {
	return _Layer2ManagerV1.Contract.MINTERROLE(&_Layer2ManagerV1.CallOpts)
}

// MINTERROLE is a free data retrieval call binding the contract method 0xd5391393.
//
// Solidity: function MINTER_ROLE() view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) MINTERROLE() ([32]byte, error) {
	return _Layer2ManagerV1.Contract.MINTERROLE(&_Layer2ManagerV1.CallOpts)
}

// CheckL1BridgeDetail is a free data retrieval call binding the contract method 0x4cdf7b16.
//
// Solidity: function _checkL1BridgeDetail(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton, uint8 _type, uint8 status, bool rejectedSeigs, bool rejectedL2Deposit)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) CheckL1BridgeDetail(opts *bind.CallOpts, _rollupConfig common.Address) (struct {
	Result            bool
	L1Bridge          common.Address
	Portal            common.Address
	L2Ton             common.Address
	Type              uint8
	Status            uint8
	RejectedSeigs     bool
	RejectedL2Deposit bool
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "_checkL1BridgeDetail", _rollupConfig)

	outstruct := new(struct {
		Result            bool
		L1Bridge          common.Address
		Portal            common.Address
		L2Ton             common.Address
		Type              uint8
		Status            uint8
		RejectedSeigs     bool
		RejectedL2Deposit bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Result = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.L1Bridge = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Portal = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.L2Ton = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.Type = *abi.ConvertType(out[4], new(uint8)).(*uint8)
	outstruct.Status = *abi.ConvertType(out[5], new(uint8)).(*uint8)
	outstruct.RejectedSeigs = *abi.ConvertType(out[6], new(bool)).(*bool)
	outstruct.RejectedL2Deposit = *abi.ConvertType(out[7], new(bool)).(*bool)

	return *outstruct, err

}

// CheckL1BridgeDetail is a free data retrieval call binding the contract method 0x4cdf7b16.
//
// Solidity: function _checkL1BridgeDetail(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton, uint8 _type, uint8 status, bool rejectedSeigs, bool rejectedL2Deposit)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) CheckL1BridgeDetail(_rollupConfig common.Address) (struct {
	Result            bool
	L1Bridge          common.Address
	Portal            common.Address
	L2Ton             common.Address
	Type              uint8
	Status            uint8
	RejectedSeigs     bool
	RejectedL2Deposit bool
}, error) {
	return _Layer2ManagerV1.Contract.CheckL1BridgeDetail(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CheckL1BridgeDetail is a free data retrieval call binding the contract method 0x4cdf7b16.
//
// Solidity: function _checkL1BridgeDetail(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton, uint8 _type, uint8 status, bool rejectedSeigs, bool rejectedL2Deposit)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) CheckL1BridgeDetail(_rollupConfig common.Address) (struct {
	Result            bool
	L1Bridge          common.Address
	Portal            common.Address
	L2Ton             common.Address
	Type              uint8
	Status            uint8
	RejectedSeigs     bool
	RejectedL2Deposit bool
}, error) {
	return _Layer2ManagerV1.Contract.CheckL1BridgeDetail(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// AliveImplementation is a free data retrieval call binding the contract method 0x550d01a3.
//
// Solidity: function aliveImplementation(address ) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) AliveImplementation(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "aliveImplementation", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AliveImplementation is a free data retrieval call binding the contract method 0x550d01a3.
//
// Solidity: function aliveImplementation(address ) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) AliveImplementation(arg0 common.Address) (bool, error) {
	return _Layer2ManagerV1.Contract.AliveImplementation(&_Layer2ManagerV1.CallOpts, arg0)
}

// AliveImplementation is a free data retrieval call binding the contract method 0x550d01a3.
//
// Solidity: function aliveImplementation(address ) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) AliveImplementation(arg0 common.Address) (bool, error) {
	return _Layer2ManagerV1.Contract.AliveImplementation(&_Layer2ManagerV1.CallOpts, arg0)
}

// AvailableRegister is a free data retrieval call binding the contract method 0x6a909247.
//
// Solidity: function availableRegister(address _rollupConfig) view returns(bool result, uint256 amount)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) AvailableRegister(opts *bind.CallOpts, _rollupConfig common.Address) (struct {
	Result bool
	Amount *big.Int
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "availableRegister", _rollupConfig)

	outstruct := new(struct {
		Result bool
		Amount *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Result = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.Amount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// AvailableRegister is a free data retrieval call binding the contract method 0x6a909247.
//
// Solidity: function availableRegister(address _rollupConfig) view returns(bool result, uint256 amount)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) AvailableRegister(_rollupConfig common.Address) (struct {
	Result bool
	Amount *big.Int
}, error) {
	return _Layer2ManagerV1.Contract.AvailableRegister(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// AvailableRegister is a free data retrieval call binding the contract method 0x6a909247.
//
// Solidity: function availableRegister(address _rollupConfig) view returns(bool result, uint256 amount)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) AvailableRegister(_rollupConfig common.Address) (struct {
	Result bool
	Amount *big.Int
}, error) {
	return _Layer2ManagerV1.Contract.AvailableRegister(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CandidateAddOnOfOperator is a free data retrieval call binding the contract method 0x7bae05f1.
//
// Solidity: function candidateAddOnOfOperator(address _oper) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) CandidateAddOnOfOperator(opts *bind.CallOpts, _oper common.Address) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "candidateAddOnOfOperator", _oper)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CandidateAddOnOfOperator is a free data retrieval call binding the contract method 0x7bae05f1.
//
// Solidity: function candidateAddOnOfOperator(address _oper) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) CandidateAddOnOfOperator(_oper common.Address) (common.Address, error) {
	return _Layer2ManagerV1.Contract.CandidateAddOnOfOperator(&_Layer2ManagerV1.CallOpts, _oper)
}

// CandidateAddOnOfOperator is a free data retrieval call binding the contract method 0x7bae05f1.
//
// Solidity: function candidateAddOnOfOperator(address _oper) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) CandidateAddOnOfOperator(_oper common.Address) (common.Address, error) {
	return _Layer2ManagerV1.Contract.CandidateAddOnOfOperator(&_Layer2ManagerV1.CallOpts, _oper)
}

// CheckL1Bridge is a free data retrieval call binding the contract method 0x58bf884f.
//
// Solidity: function checkL1Bridge(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) CheckL1Bridge(opts *bind.CallOpts, _rollupConfig common.Address) (struct {
	Result   bool
	L1Bridge common.Address
	Portal   common.Address
	L2Ton    common.Address
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "checkL1Bridge", _rollupConfig)

	outstruct := new(struct {
		Result   bool
		L1Bridge common.Address
		Portal   common.Address
		L2Ton    common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Result = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.L1Bridge = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Portal = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.L2Ton = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// CheckL1Bridge is a free data retrieval call binding the contract method 0x58bf884f.
//
// Solidity: function checkL1Bridge(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) CheckL1Bridge(_rollupConfig common.Address) (struct {
	Result   bool
	L1Bridge common.Address
	Portal   common.Address
	L2Ton    common.Address
}, error) {
	return _Layer2ManagerV1.Contract.CheckL1Bridge(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CheckL1Bridge is a free data retrieval call binding the contract method 0x58bf884f.
//
// Solidity: function checkL1Bridge(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) CheckL1Bridge(_rollupConfig common.Address) (struct {
	Result   bool
	L1Bridge common.Address
	Portal   common.Address
	L2Ton    common.Address
}, error) {
	return _Layer2ManagerV1.Contract.CheckL1Bridge(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CheckL1BridgeDetail1 is a free data retrieval call binding the contract method 0x2e0f1775.
//
// Solidity: function checkL1BridgeDetail(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton, uint8 _type, uint8 status, bool rejectedSeigs, bool rejectedL2Deposit)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) CheckL1BridgeDetail1(opts *bind.CallOpts, _rollupConfig common.Address) (struct {
	Result            bool
	L1Bridge          common.Address
	Portal            common.Address
	L2Ton             common.Address
	Type              uint8
	Status            uint8
	RejectedSeigs     bool
	RejectedL2Deposit bool
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "checkL1BridgeDetail", _rollupConfig)

	outstruct := new(struct {
		Result            bool
		L1Bridge          common.Address
		Portal            common.Address
		L2Ton             common.Address
		Type              uint8
		Status            uint8
		RejectedSeigs     bool
		RejectedL2Deposit bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Result = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.L1Bridge = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Portal = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.L2Ton = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.Type = *abi.ConvertType(out[4], new(uint8)).(*uint8)
	outstruct.Status = *abi.ConvertType(out[5], new(uint8)).(*uint8)
	outstruct.RejectedSeigs = *abi.ConvertType(out[6], new(bool)).(*bool)
	outstruct.RejectedL2Deposit = *abi.ConvertType(out[7], new(bool)).(*bool)

	return *outstruct, err

}

// CheckL1BridgeDetail1 is a free data retrieval call binding the contract method 0x2e0f1775.
//
// Solidity: function checkL1BridgeDetail(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton, uint8 _type, uint8 status, bool rejectedSeigs, bool rejectedL2Deposit)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) CheckL1BridgeDetail1(_rollupConfig common.Address) (struct {
	Result            bool
	L1Bridge          common.Address
	Portal            common.Address
	L2Ton             common.Address
	Type              uint8
	Status            uint8
	RejectedSeigs     bool
	RejectedL2Deposit bool
}, error) {
	return _Layer2ManagerV1.Contract.CheckL1BridgeDetail1(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CheckL1BridgeDetail1 is a free data retrieval call binding the contract method 0x2e0f1775.
//
// Solidity: function checkL1BridgeDetail(address _rollupConfig) view returns(bool result, address l1Bridge, address portal, address l2Ton, uint8 _type, uint8 status, bool rejectedSeigs, bool rejectedL2Deposit)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) CheckL1BridgeDetail1(_rollupConfig common.Address) (struct {
	Result            bool
	L1Bridge          common.Address
	Portal            common.Address
	L2Ton             common.Address
	Type              uint8
	Status            uint8
	RejectedSeigs     bool
	RejectedL2Deposit bool
}, error) {
	return _Layer2ManagerV1.Contract.CheckL1BridgeDetail1(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CheckLayer2TVL is a free data retrieval call binding the contract method 0x89c0db62.
//
// Solidity: function checkLayer2TVL(address _rollupConfig) view returns(bool result, uint256 amount)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) CheckLayer2TVL(opts *bind.CallOpts, _rollupConfig common.Address) (struct {
	Result bool
	Amount *big.Int
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "checkLayer2TVL", _rollupConfig)

	outstruct := new(struct {
		Result bool
		Amount *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Result = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.Amount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// CheckLayer2TVL is a free data retrieval call binding the contract method 0x89c0db62.
//
// Solidity: function checkLayer2TVL(address _rollupConfig) view returns(bool result, uint256 amount)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) CheckLayer2TVL(_rollupConfig common.Address) (struct {
	Result bool
	Amount *big.Int
}, error) {
	return _Layer2ManagerV1.Contract.CheckLayer2TVL(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// CheckLayer2TVL is a free data retrieval call binding the contract method 0x89c0db62.
//
// Solidity: function checkLayer2TVL(address _rollupConfig) view returns(bool result, uint256 amount)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) CheckLayer2TVL(_rollupConfig common.Address) (struct {
	Result bool
	Amount *big.Int
}, error) {
	return _Layer2ManagerV1.Contract.CheckLayer2TVL(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// Dao is a free data retrieval call binding the contract method 0x4162169f.
//
// Solidity: function dao() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) Dao(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "dao")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Dao is a free data retrieval call binding the contract method 0x4162169f.
//
// Solidity: function dao() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) Dao() (common.Address, error) {
	return _Layer2ManagerV1.Contract.Dao(&_Layer2ManagerV1.CallOpts)
}

// Dao is a free data retrieval call binding the contract method 0x4162169f.
//
// Solidity: function dao() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) Dao() (common.Address, error) {
	return _Layer2ManagerV1.Contract.Dao(&_Layer2ManagerV1.CallOpts)
}

// DepositManager is a free data retrieval call binding the contract method 0x6c7ac9d8.
//
// Solidity: function depositManager() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) DepositManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "depositManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DepositManager is a free data retrieval call binding the contract method 0x6c7ac9d8.
//
// Solidity: function depositManager() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) DepositManager() (common.Address, error) {
	return _Layer2ManagerV1.Contract.DepositManager(&_Layer2ManagerV1.CallOpts)
}

// DepositManager is a free data retrieval call binding the contract method 0x6c7ac9d8.
//
// Solidity: function depositManager() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) DepositManager() (common.Address, error) {
	return _Layer2ManagerV1.Contract.DepositManager(&_Layer2ManagerV1.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Layer2ManagerV1.Contract.GetRoleAdmin(&_Layer2ManagerV1.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Layer2ManagerV1.Contract.GetRoleAdmin(&_Layer2ManagerV1.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Layer2ManagerV1.Contract.HasRole(&_Layer2ManagerV1.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Layer2ManagerV1.Contract.HasRole(&_Layer2ManagerV1.CallOpts, role, account)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address account) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) IsAdmin(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "isAdmin", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address account) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) IsAdmin(account common.Address) (bool, error) {
	return _Layer2ManagerV1.Contract.IsAdmin(&_Layer2ManagerV1.CallOpts, account)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address account) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) IsAdmin(account common.Address) (bool, error) {
	return _Layer2ManagerV1.Contract.IsAdmin(&_Layer2ManagerV1.CallOpts, account)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "isOwner")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) IsOwner() (bool, error) {
	return _Layer2ManagerV1.Contract.IsOwner(&_Layer2ManagerV1.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) IsOwner() (bool, error) {
	return _Layer2ManagerV1.Contract.IsOwner(&_Layer2ManagerV1.CallOpts)
}

// L1BridgeRegistry is a free data retrieval call binding the contract method 0xcf8f9110.
//
// Solidity: function l1BridgeRegistry() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) L1BridgeRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "l1BridgeRegistry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeRegistry is a free data retrieval call binding the contract method 0xcf8f9110.
//
// Solidity: function l1BridgeRegistry() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) L1BridgeRegistry() (common.Address, error) {
	return _Layer2ManagerV1.Contract.L1BridgeRegistry(&_Layer2ManagerV1.CallOpts)
}

// L1BridgeRegistry is a free data retrieval call binding the contract method 0xcf8f9110.
//
// Solidity: function l1BridgeRegistry() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) L1BridgeRegistry() (common.Address, error) {
	return _Layer2ManagerV1.Contract.L1BridgeRegistry(&_Layer2ManagerV1.CallOpts)
}

// MinimumInitialDepositAmount is a free data retrieval call binding the contract method 0xe9d6ec4f.
//
// Solidity: function minimumInitialDepositAmount() view returns(uint256)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) MinimumInitialDepositAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "minimumInitialDepositAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinimumInitialDepositAmount is a free data retrieval call binding the contract method 0xe9d6ec4f.
//
// Solidity: function minimumInitialDepositAmount() view returns(uint256)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) MinimumInitialDepositAmount() (*big.Int, error) {
	return _Layer2ManagerV1.Contract.MinimumInitialDepositAmount(&_Layer2ManagerV1.CallOpts)
}

// MinimumInitialDepositAmount is a free data retrieval call binding the contract method 0xe9d6ec4f.
//
// Solidity: function minimumInitialDepositAmount() view returns(uint256)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) MinimumInitialDepositAmount() (*big.Int, error) {
	return _Layer2ManagerV1.Contract.MinimumInitialDepositAmount(&_Layer2ManagerV1.CallOpts)
}

// OperatorInfo is a free data retrieval call binding the contract method 0x50c246d6.
//
// Solidity: function operatorInfo(address ) view returns(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) OperatorInfo(opts *bind.CallOpts, arg0 common.Address) (struct {
	RollupConfig   common.Address
	CandidateAddOn common.Address
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "operatorInfo", arg0)

	outstruct := new(struct {
		RollupConfig   common.Address
		CandidateAddOn common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.RollupConfig = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.CandidateAddOn = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// OperatorInfo is a free data retrieval call binding the contract method 0x50c246d6.
//
// Solidity: function operatorInfo(address ) view returns(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) OperatorInfo(arg0 common.Address) (struct {
	RollupConfig   common.Address
	CandidateAddOn common.Address
}, error) {
	return _Layer2ManagerV1.Contract.OperatorInfo(&_Layer2ManagerV1.CallOpts, arg0)
}

// OperatorInfo is a free data retrieval call binding the contract method 0x50c246d6.
//
// Solidity: function operatorInfo(address ) view returns(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) OperatorInfo(arg0 common.Address) (struct {
	RollupConfig   common.Address
	CandidateAddOn common.Address
}, error) {
	return _Layer2ManagerV1.Contract.OperatorInfo(&_Layer2ManagerV1.CallOpts, arg0)
}

// OperatorManagerFactory is a free data retrieval call binding the contract method 0xaba8e577.
//
// Solidity: function operatorManagerFactory() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) OperatorManagerFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "operatorManagerFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OperatorManagerFactory is a free data retrieval call binding the contract method 0xaba8e577.
//
// Solidity: function operatorManagerFactory() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) OperatorManagerFactory() (common.Address, error) {
	return _Layer2ManagerV1.Contract.OperatorManagerFactory(&_Layer2ManagerV1.CallOpts)
}

// OperatorManagerFactory is a free data retrieval call binding the contract method 0xaba8e577.
//
// Solidity: function operatorManagerFactory() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) OperatorManagerFactory() (common.Address, error) {
	return _Layer2ManagerV1.Contract.OperatorManagerFactory(&_Layer2ManagerV1.CallOpts)
}

// OperatorOfRollupConfig is a free data retrieval call binding the contract method 0xff600b69.
//
// Solidity: function operatorOfRollupConfig(address _rollupConfig) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) OperatorOfRollupConfig(opts *bind.CallOpts, _rollupConfig common.Address) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "operatorOfRollupConfig", _rollupConfig)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OperatorOfRollupConfig is a free data retrieval call binding the contract method 0xff600b69.
//
// Solidity: function operatorOfRollupConfig(address _rollupConfig) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) OperatorOfRollupConfig(_rollupConfig common.Address) (common.Address, error) {
	return _Layer2ManagerV1.Contract.OperatorOfRollupConfig(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// OperatorOfRollupConfig is a free data retrieval call binding the contract method 0xff600b69.
//
// Solidity: function operatorOfRollupConfig(address _rollupConfig) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) OperatorOfRollupConfig(_rollupConfig common.Address) (common.Address, error) {
	return _Layer2ManagerV1.Contract.OperatorOfRollupConfig(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// PauseProxy is a free data retrieval call binding the contract method 0x63a8fd89.
//
// Solidity: function pauseProxy() view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) PauseProxy(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "pauseProxy")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PauseProxy is a free data retrieval call binding the contract method 0x63a8fd89.
//
// Solidity: function pauseProxy() view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) PauseProxy() (bool, error) {
	return _Layer2ManagerV1.Contract.PauseProxy(&_Layer2ManagerV1.CallOpts)
}

// PauseProxy is a free data retrieval call binding the contract method 0x63a8fd89.
//
// Solidity: function pauseProxy() view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) PauseProxy() (bool, error) {
	return _Layer2ManagerV1.Contract.PauseProxy(&_Layer2ManagerV1.CallOpts)
}

// ProxyImplementation is a free data retrieval call binding the contract method 0xb911135f.
//
// Solidity: function proxyImplementation(uint256 ) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) ProxyImplementation(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "proxyImplementation", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProxyImplementation is a free data retrieval call binding the contract method 0xb911135f.
//
// Solidity: function proxyImplementation(uint256 ) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) ProxyImplementation(arg0 *big.Int) (common.Address, error) {
	return _Layer2ManagerV1.Contract.ProxyImplementation(&_Layer2ManagerV1.CallOpts, arg0)
}

// ProxyImplementation is a free data retrieval call binding the contract method 0xb911135f.
//
// Solidity: function proxyImplementation(uint256 ) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) ProxyImplementation(arg0 *big.Int) (common.Address, error) {
	return _Layer2ManagerV1.Contract.ProxyImplementation(&_Layer2ManagerV1.CallOpts, arg0)
}

// RollupConfigInfo is a free data retrieval call binding the contract method 0x72898c23.
//
// Solidity: function rollupConfigInfo(address ) view returns(uint8 status, address operatorManager)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) RollupConfigInfo(opts *bind.CallOpts, arg0 common.Address) (struct {
	Status          uint8
	OperatorManager common.Address
}, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "rollupConfigInfo", arg0)

	outstruct := new(struct {
		Status          uint8
		OperatorManager common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Status = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.OperatorManager = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// RollupConfigInfo is a free data retrieval call binding the contract method 0x72898c23.
//
// Solidity: function rollupConfigInfo(address ) view returns(uint8 status, address operatorManager)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RollupConfigInfo(arg0 common.Address) (struct {
	Status          uint8
	OperatorManager common.Address
}, error) {
	return _Layer2ManagerV1.Contract.RollupConfigInfo(&_Layer2ManagerV1.CallOpts, arg0)
}

// RollupConfigInfo is a free data retrieval call binding the contract method 0x72898c23.
//
// Solidity: function rollupConfigInfo(address ) view returns(uint8 status, address operatorManager)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) RollupConfigInfo(arg0 common.Address) (struct {
	Status          uint8
	OperatorManager common.Address
}, error) {
	return _Layer2ManagerV1.Contract.RollupConfigInfo(&_Layer2ManagerV1.CallOpts, arg0)
}

// RollupConfigOfOperator is a free data retrieval call binding the contract method 0xa37d6aa7.
//
// Solidity: function rollupConfigOfOperator(address _oper) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) RollupConfigOfOperator(opts *bind.CallOpts, _oper common.Address) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "rollupConfigOfOperator", _oper)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RollupConfigOfOperator is a free data retrieval call binding the contract method 0xa37d6aa7.
//
// Solidity: function rollupConfigOfOperator(address _oper) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RollupConfigOfOperator(_oper common.Address) (common.Address, error) {
	return _Layer2ManagerV1.Contract.RollupConfigOfOperator(&_Layer2ManagerV1.CallOpts, _oper)
}

// RollupConfigOfOperator is a free data retrieval call binding the contract method 0xa37d6aa7.
//
// Solidity: function rollupConfigOfOperator(address _oper) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) RollupConfigOfOperator(_oper common.Address) (common.Address, error) {
	return _Layer2ManagerV1.Contract.RollupConfigOfOperator(&_Layer2ManagerV1.CallOpts, _oper)
}

// SeigManager is a free data retrieval call binding the contract method 0x6fb7f558.
//
// Solidity: function seigManager() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) SeigManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "seigManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SeigManager is a free data retrieval call binding the contract method 0x6fb7f558.
//
// Solidity: function seigManager() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SeigManager() (common.Address, error) {
	return _Layer2ManagerV1.Contract.SeigManager(&_Layer2ManagerV1.CallOpts)
}

// SeigManager is a free data retrieval call binding the contract method 0x6fb7f558.
//
// Solidity: function seigManager() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) SeigManager() (common.Address, error) {
	return _Layer2ManagerV1.Contract.SeigManager(&_Layer2ManagerV1.CallOpts)
}

// SelectorImplementation is a free data retrieval call binding the contract method 0x50d2a276.
//
// Solidity: function selectorImplementation(bytes4 ) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) SelectorImplementation(opts *bind.CallOpts, arg0 [4]byte) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "selectorImplementation", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SelectorImplementation is a free data retrieval call binding the contract method 0x50d2a276.
//
// Solidity: function selectorImplementation(bytes4 ) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SelectorImplementation(arg0 [4]byte) (common.Address, error) {
	return _Layer2ManagerV1.Contract.SelectorImplementation(&_Layer2ManagerV1.CallOpts, arg0)
}

// SelectorImplementation is a free data retrieval call binding the contract method 0x50d2a276.
//
// Solidity: function selectorImplementation(bytes4 ) view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) SelectorImplementation(arg0 [4]byte) (common.Address, error) {
	return _Layer2ManagerV1.Contract.SelectorImplementation(&_Layer2ManagerV1.CallOpts, arg0)
}

// StatusLayer2 is a free data retrieval call binding the contract method 0x79a0e6ea.
//
// Solidity: function statusLayer2(address _rollupConfig) view returns(uint8)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) StatusLayer2(opts *bind.CallOpts, _rollupConfig common.Address) (uint8, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "statusLayer2", _rollupConfig)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// StatusLayer2 is a free data retrieval call binding the contract method 0x79a0e6ea.
//
// Solidity: function statusLayer2(address _rollupConfig) view returns(uint8)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) StatusLayer2(_rollupConfig common.Address) (uint8, error) {
	return _Layer2ManagerV1.Contract.StatusLayer2(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// StatusLayer2 is a free data retrieval call binding the contract method 0x79a0e6ea.
//
// Solidity: function statusLayer2(address _rollupConfig) view returns(uint8)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) StatusLayer2(_rollupConfig common.Address) (uint8, error) {
	return _Layer2ManagerV1.Contract.StatusLayer2(&_Layer2ManagerV1.CallOpts, _rollupConfig)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Layer2ManagerV1.Contract.SupportsInterface(&_Layer2ManagerV1.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Layer2ManagerV1.Contract.SupportsInterface(&_Layer2ManagerV1.CallOpts, interfaceId)
}

// SwapProxy is a free data retrieval call binding the contract method 0x6ec4be90.
//
// Solidity: function swapProxy() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) SwapProxy(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "swapProxy")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SwapProxy is a free data retrieval call binding the contract method 0x6ec4be90.
//
// Solidity: function swapProxy() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SwapProxy() (common.Address, error) {
	return _Layer2ManagerV1.Contract.SwapProxy(&_Layer2ManagerV1.CallOpts)
}

// SwapProxy is a free data retrieval call binding the contract method 0x6ec4be90.
//
// Solidity: function swapProxy() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) SwapProxy() (common.Address, error) {
	return _Layer2ManagerV1.Contract.SwapProxy(&_Layer2ManagerV1.CallOpts)
}

// Ton is a free data retrieval call binding the contract method 0xcc48b947.
//
// Solidity: function ton() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) Ton(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "ton")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Ton is a free data retrieval call binding the contract method 0xcc48b947.
//
// Solidity: function ton() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) Ton() (common.Address, error) {
	return _Layer2ManagerV1.Contract.Ton(&_Layer2ManagerV1.CallOpts)
}

// Ton is a free data retrieval call binding the contract method 0xcc48b947.
//
// Solidity: function ton() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) Ton() (common.Address, error) {
	return _Layer2ManagerV1.Contract.Ton(&_Layer2ManagerV1.CallOpts)
}

// Wton is a free data retrieval call binding the contract method 0x8d62d949.
//
// Solidity: function wton() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Caller) Wton(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Layer2ManagerV1.contract.Call(opts, &out, "wton")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Wton is a free data retrieval call binding the contract method 0x8d62d949.
//
// Solidity: function wton() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) Wton() (common.Address, error) {
	return _Layer2ManagerV1.Contract.Wton(&_Layer2ManagerV1.CallOpts)
}

// Wton is a free data retrieval call binding the contract method 0x8d62d949.
//
// Solidity: function wton() view returns(address)
func (_Layer2ManagerV1 *Layer2ManagerV1CallerSession) Wton() (common.Address, error) {
	return _Layer2ManagerV1.Contract.Wton(&_Layer2ManagerV1.CallOpts)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) AddAdmin(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "addAdmin", account)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) AddAdmin(account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.AddAdmin(&_Layer2ManagerV1.TransactOpts, account)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) AddAdmin(account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.AddAdmin(&_Layer2ManagerV1.TransactOpts, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.GrantRole(&_Layer2ManagerV1.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.GrantRole(&_Layer2ManagerV1.TransactOpts, role, account)
}

// OnApprove is a paid mutator transaction binding the contract method 0x4273ca16.
//
// Solidity: function onApprove(address owner, address spender, uint256 amount, bytes data) returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) OnApprove(opts *bind.TransactOpts, owner common.Address, spender common.Address, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "onApprove", owner, spender, amount, data)
}

// OnApprove is a paid mutator transaction binding the contract method 0x4273ca16.
//
// Solidity: function onApprove(address owner, address spender, uint256 amount, bytes data) returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1Session) OnApprove(owner common.Address, spender common.Address, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.OnApprove(&_Layer2ManagerV1.TransactOpts, owner, spender, amount, data)
}

// OnApprove is a paid mutator transaction binding the contract method 0x4273ca16.
//
// Solidity: function onApprove(address owner, address spender, uint256 amount, bytes data) returns(bool)
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) OnApprove(owner common.Address, spender common.Address, amount *big.Int, data []byte) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.OnApprove(&_Layer2ManagerV1.TransactOpts, owner, spender, amount, data)
}

// PauseCandidateAddOn is a paid mutator transaction binding the contract method 0x32d89cc6.
//
// Solidity: function pauseCandidateAddOn(address rollupConfig) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) PauseCandidateAddOn(opts *bind.TransactOpts, rollupConfig common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "pauseCandidateAddOn", rollupConfig)
}

// PauseCandidateAddOn is a paid mutator transaction binding the contract method 0x32d89cc6.
//
// Solidity: function pauseCandidateAddOn(address rollupConfig) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) PauseCandidateAddOn(rollupConfig common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.PauseCandidateAddOn(&_Layer2ManagerV1.TransactOpts, rollupConfig)
}

// PauseCandidateAddOn is a paid mutator transaction binding the contract method 0x32d89cc6.
//
// Solidity: function pauseCandidateAddOn(address rollupConfig) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) PauseCandidateAddOn(rollupConfig common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.PauseCandidateAddOn(&_Layer2ManagerV1.TransactOpts, rollupConfig)
}

// RegisterCandidateAddOn is a paid mutator transaction binding the contract method 0xc04a0a42.
//
// Solidity: function registerCandidateAddOn(address rollupConfig, uint256 amount, bool flagTon, string memo) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) RegisterCandidateAddOn(opts *bind.TransactOpts, rollupConfig common.Address, amount *big.Int, flagTon bool, memo string) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "registerCandidateAddOn", rollupConfig, amount, flagTon, memo)
}

// RegisterCandidateAddOn is a paid mutator transaction binding the contract method 0xc04a0a42.
//
// Solidity: function registerCandidateAddOn(address rollupConfig, uint256 amount, bool flagTon, string memo) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RegisterCandidateAddOn(rollupConfig common.Address, amount *big.Int, flagTon bool, memo string) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RegisterCandidateAddOn(&_Layer2ManagerV1.TransactOpts, rollupConfig, amount, flagTon, memo)
}

// RegisterCandidateAddOn is a paid mutator transaction binding the contract method 0xc04a0a42.
//
// Solidity: function registerCandidateAddOn(address rollupConfig, uint256 amount, bool flagTon, string memo) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) RegisterCandidateAddOn(rollupConfig common.Address, amount *big.Int, flagTon bool, memo string) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RegisterCandidateAddOn(&_Layer2ManagerV1.TransactOpts, rollupConfig, amount, flagTon, memo)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) RemoveAdmin(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "removeAdmin", account)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RemoveAdmin(account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RemoveAdmin(&_Layer2ManagerV1.TransactOpts, account)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) RemoveAdmin(account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RemoveAdmin(&_Layer2ManagerV1.TransactOpts, account)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RenounceOwnership() (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RenounceOwnership(&_Layer2ManagerV1.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RenounceOwnership(&_Layer2ManagerV1.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RenounceRole(&_Layer2ManagerV1.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RenounceRole(&_Layer2ManagerV1.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RevokeRole(&_Layer2ManagerV1.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.RevokeRole(&_Layer2ManagerV1.TransactOpts, role, account)
}

// SetAddresses is a paid mutator transaction binding the contract method 0xd733cfd0.
//
// Solidity: function setAddresses(address _l1BridgeRegistry, address _operatorManagerFactory, address _ton, address _wton, address _dao, address _depositManager, address _seigManager, address _swapProxy) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) SetAddresses(opts *bind.TransactOpts, _l1BridgeRegistry common.Address, _operatorManagerFactory common.Address, _ton common.Address, _wton common.Address, _dao common.Address, _depositManager common.Address, _seigManager common.Address, _swapProxy common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "setAddresses", _l1BridgeRegistry, _operatorManagerFactory, _ton, _wton, _dao, _depositManager, _seigManager, _swapProxy)
}

// SetAddresses is a paid mutator transaction binding the contract method 0xd733cfd0.
//
// Solidity: function setAddresses(address _l1BridgeRegistry, address _operatorManagerFactory, address _ton, address _wton, address _dao, address _depositManager, address _seigManager, address _swapProxy) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SetAddresses(_l1BridgeRegistry common.Address, _operatorManagerFactory common.Address, _ton common.Address, _wton common.Address, _dao common.Address, _depositManager common.Address, _seigManager common.Address, _swapProxy common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.SetAddresses(&_Layer2ManagerV1.TransactOpts, _l1BridgeRegistry, _operatorManagerFactory, _ton, _wton, _dao, _depositManager, _seigManager, _swapProxy)
}

// SetAddresses is a paid mutator transaction binding the contract method 0xd733cfd0.
//
// Solidity: function setAddresses(address _l1BridgeRegistry, address _operatorManagerFactory, address _ton, address _wton, address _dao, address _depositManager, address _seigManager, address _swapProxy) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) SetAddresses(_l1BridgeRegistry common.Address, _operatorManagerFactory common.Address, _ton common.Address, _wton common.Address, _dao common.Address, _depositManager common.Address, _seigManager common.Address, _swapProxy common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.SetAddresses(&_Layer2ManagerV1.TransactOpts, _l1BridgeRegistry, _operatorManagerFactory, _ton, _wton, _dao, _depositManager, _seigManager, _swapProxy)
}

// SetMinimumInitialDepositAmount is a paid mutator transaction binding the contract method 0x4318dc1d.
//
// Solidity: function setMinimumInitialDepositAmount(uint256 _minimumInitialDepositAmount) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) SetMinimumInitialDepositAmount(opts *bind.TransactOpts, _minimumInitialDepositAmount *big.Int) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "setMinimumInitialDepositAmount", _minimumInitialDepositAmount)
}

// SetMinimumInitialDepositAmount is a paid mutator transaction binding the contract method 0x4318dc1d.
//
// Solidity: function setMinimumInitialDepositAmount(uint256 _minimumInitialDepositAmount) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SetMinimumInitialDepositAmount(_minimumInitialDepositAmount *big.Int) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.SetMinimumInitialDepositAmount(&_Layer2ManagerV1.TransactOpts, _minimumInitialDepositAmount)
}

// SetMinimumInitialDepositAmount is a paid mutator transaction binding the contract method 0x4318dc1d.
//
// Solidity: function setMinimumInitialDepositAmount(uint256 _minimumInitialDepositAmount) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) SetMinimumInitialDepositAmount(_minimumInitialDepositAmount *big.Int) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.SetMinimumInitialDepositAmount(&_Layer2ManagerV1.TransactOpts, _minimumInitialDepositAmount)
}

// SetOperatorManagerFactory is a paid mutator transaction binding the contract method 0xc8452265.
//
// Solidity: function setOperatorManagerFactory(address _operatorManagerFactory) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) SetOperatorManagerFactory(opts *bind.TransactOpts, _operatorManagerFactory common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "setOperatorManagerFactory", _operatorManagerFactory)
}

// SetOperatorManagerFactory is a paid mutator transaction binding the contract method 0xc8452265.
//
// Solidity: function setOperatorManagerFactory(address _operatorManagerFactory) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) SetOperatorManagerFactory(_operatorManagerFactory common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.SetOperatorManagerFactory(&_Layer2ManagerV1.TransactOpts, _operatorManagerFactory)
}

// SetOperatorManagerFactory is a paid mutator transaction binding the contract method 0xc8452265.
//
// Solidity: function setOperatorManagerFactory(address _operatorManagerFactory) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) SetOperatorManagerFactory(_operatorManagerFactory common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.SetOperatorManagerFactory(&_Layer2ManagerV1.TransactOpts, _operatorManagerFactory)
}

// TransferAdmin is a paid mutator transaction binding the contract method 0x75829def.
//
// Solidity: function transferAdmin(address newAdmin) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) TransferAdmin(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "transferAdmin", newAdmin)
}

// TransferAdmin is a paid mutator transaction binding the contract method 0x75829def.
//
// Solidity: function transferAdmin(address newAdmin) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) TransferAdmin(newAdmin common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.TransferAdmin(&_Layer2ManagerV1.TransactOpts, newAdmin)
}

// TransferAdmin is a paid mutator transaction binding the contract method 0x75829def.
//
// Solidity: function transferAdmin(address newAdmin) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) TransferAdmin(newAdmin common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.TransferAdmin(&_Layer2ManagerV1.TransactOpts, newAdmin)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newAdmin) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) TransferOwnership(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "transferOwnership", newAdmin)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newAdmin) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) TransferOwnership(newAdmin common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.TransferOwnership(&_Layer2ManagerV1.TransactOpts, newAdmin)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newAdmin) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) TransferOwnership(newAdmin common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.TransferOwnership(&_Layer2ManagerV1.TransactOpts, newAdmin)
}

// UnpauseCandidateAddOn is a paid mutator transaction binding the contract method 0xd523c077.
//
// Solidity: function unpauseCandidateAddOn(address rollupConfig) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) UnpauseCandidateAddOn(opts *bind.TransactOpts, rollupConfig common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "unpauseCandidateAddOn", rollupConfig)
}

// UnpauseCandidateAddOn is a paid mutator transaction binding the contract method 0xd523c077.
//
// Solidity: function unpauseCandidateAddOn(address rollupConfig) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) UnpauseCandidateAddOn(rollupConfig common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.UnpauseCandidateAddOn(&_Layer2ManagerV1.TransactOpts, rollupConfig)
}

// UnpauseCandidateAddOn is a paid mutator transaction binding the contract method 0xd523c077.
//
// Solidity: function unpauseCandidateAddOn(address rollupConfig) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) UnpauseCandidateAddOn(rollupConfig common.Address) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.UnpauseCandidateAddOn(&_Layer2ManagerV1.TransactOpts, rollupConfig)
}

// UpdateSeigniorage is a paid mutator transaction binding the contract method 0xc5f16b89.
//
// Solidity: function updateSeigniorage(address rollupConfig, uint256 amount) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Transactor) UpdateSeigniorage(opts *bind.TransactOpts, rollupConfig common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Layer2ManagerV1.contract.Transact(opts, "updateSeigniorage", rollupConfig, amount)
}

// UpdateSeigniorage is a paid mutator transaction binding the contract method 0xc5f16b89.
//
// Solidity: function updateSeigniorage(address rollupConfig, uint256 amount) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1Session) UpdateSeigniorage(rollupConfig common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.UpdateSeigniorage(&_Layer2ManagerV1.TransactOpts, rollupConfig, amount)
}

// UpdateSeigniorage is a paid mutator transaction binding the contract method 0xc5f16b89.
//
// Solidity: function updateSeigniorage(address rollupConfig, uint256 amount) returns()
func (_Layer2ManagerV1 *Layer2ManagerV1TransactorSession) UpdateSeigniorage(rollupConfig common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Layer2ManagerV1.Contract.UpdateSeigniorage(&_Layer2ManagerV1.TransactOpts, rollupConfig, amount)
}

// Layer2ManagerV1PausedCandidateAddOnIterator is returned from FilterPausedCandidateAddOn and is used to iterate over the raw logs and unpacked data for PausedCandidateAddOn events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1PausedCandidateAddOnIterator struct {
	Event *Layer2ManagerV1PausedCandidateAddOn // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1PausedCandidateAddOnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1PausedCandidateAddOn)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1PausedCandidateAddOn)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1PausedCandidateAddOnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1PausedCandidateAddOnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1PausedCandidateAddOn represents a PausedCandidateAddOn event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1PausedCandidateAddOn struct {
	RollupConfig   common.Address
	CandidateAddOn common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterPausedCandidateAddOn is a free log retrieval operation binding the contract event 0x14280f6eab1c847215d9714b0aa6711ad26e21304f06a85a46ad07a1f11e8370.
//
// Solidity: event PausedCandidateAddOn(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterPausedCandidateAddOn(opts *bind.FilterOpts) (*Layer2ManagerV1PausedCandidateAddOnIterator, error) {

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "PausedCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1PausedCandidateAddOnIterator{contract: _Layer2ManagerV1.contract, event: "PausedCandidateAddOn", logs: logs, sub: sub}, nil
}

// WatchPausedCandidateAddOn is a free log subscription operation binding the contract event 0x14280f6eab1c847215d9714b0aa6711ad26e21304f06a85a46ad07a1f11e8370.
//
// Solidity: event PausedCandidateAddOn(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchPausedCandidateAddOn(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1PausedCandidateAddOn) (event.Subscription, error) {

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "PausedCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1PausedCandidateAddOn)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "PausedCandidateAddOn", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePausedCandidateAddOn is a log parse operation binding the contract event 0x14280f6eab1c847215d9714b0aa6711ad26e21304f06a85a46ad07a1f11e8370.
//
// Solidity: event PausedCandidateAddOn(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParsePausedCandidateAddOn(log types.Log) (*Layer2ManagerV1PausedCandidateAddOn, error) {
	event := new(Layer2ManagerV1PausedCandidateAddOn)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "PausedCandidateAddOn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1RegisteredCandidateAddOnIterator is returned from FilterRegisteredCandidateAddOn and is used to iterate over the raw logs and unpacked data for RegisteredCandidateAddOn events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RegisteredCandidateAddOnIterator struct {
	Event *Layer2ManagerV1RegisteredCandidateAddOn // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1RegisteredCandidateAddOnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1RegisteredCandidateAddOn)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1RegisteredCandidateAddOn)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1RegisteredCandidateAddOnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1RegisteredCandidateAddOnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1RegisteredCandidateAddOn represents a RegisteredCandidateAddOn event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RegisteredCandidateAddOn struct {
	RollupConfig   common.Address
	WtonAmount     *big.Int
	Memo           string
	Operator       common.Address
	CandidateAddOn common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterRegisteredCandidateAddOn is a free log retrieval operation binding the contract event 0x3685763fd42e5061ff53b001782ad759eafbc93a460b368a4ce42228ec45da98.
//
// Solidity: event RegisteredCandidateAddOn(address rollupConfig, uint256 wtonAmount, string memo, address operator, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterRegisteredCandidateAddOn(opts *bind.FilterOpts) (*Layer2ManagerV1RegisteredCandidateAddOnIterator, error) {

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "RegisteredCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1RegisteredCandidateAddOnIterator{contract: _Layer2ManagerV1.contract, event: "RegisteredCandidateAddOn", logs: logs, sub: sub}, nil
}

// WatchRegisteredCandidateAddOn is a free log subscription operation binding the contract event 0x3685763fd42e5061ff53b001782ad759eafbc93a460b368a4ce42228ec45da98.
//
// Solidity: event RegisteredCandidateAddOn(address rollupConfig, uint256 wtonAmount, string memo, address operator, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchRegisteredCandidateAddOn(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1RegisteredCandidateAddOn) (event.Subscription, error) {

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "RegisteredCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1RegisteredCandidateAddOn)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "RegisteredCandidateAddOn", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRegisteredCandidateAddOn is a log parse operation binding the contract event 0x3685763fd42e5061ff53b001782ad759eafbc93a460b368a4ce42228ec45da98.
//
// Solidity: event RegisteredCandidateAddOn(address rollupConfig, uint256 wtonAmount, string memo, address operator, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseRegisteredCandidateAddOn(log types.Log) (*Layer2ManagerV1RegisteredCandidateAddOn, error) {
	event := new(Layer2ManagerV1RegisteredCandidateAddOn)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "RegisteredCandidateAddOn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1RoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RoleAdminChangedIterator struct {
	Event *Layer2ManagerV1RoleAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1RoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1RoleAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1RoleAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1RoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1RoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1RoleAdminChanged represents a RoleAdminChanged event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*Layer2ManagerV1RoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1RoleAdminChangedIterator{contract: _Layer2ManagerV1.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1RoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1RoleAdminChanged)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseRoleAdminChanged(log types.Log) (*Layer2ManagerV1RoleAdminChanged, error) {
	event := new(Layer2ManagerV1RoleAdminChanged)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1RoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RoleGrantedIterator struct {
	Event *Layer2ManagerV1RoleGranted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1RoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1RoleGranted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1RoleGranted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1RoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1RoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1RoleGranted represents a RoleGranted event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*Layer2ManagerV1RoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1RoleGrantedIterator{contract: _Layer2ManagerV1.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1RoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1RoleGranted)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "RoleGranted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseRoleGranted(log types.Log) (*Layer2ManagerV1RoleGranted, error) {
	event := new(Layer2ManagerV1RoleGranted)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1RoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RoleRevokedIterator struct {
	Event *Layer2ManagerV1RoleRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1RoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1RoleRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1RoleRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1RoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1RoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1RoleRevoked represents a RoleRevoked event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1RoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*Layer2ManagerV1RoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1RoleRevokedIterator{contract: _Layer2ManagerV1.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1RoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1RoleRevoked)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseRoleRevoked(log types.Log) (*Layer2ManagerV1RoleRevoked, error) {
	event := new(Layer2ManagerV1RoleRevoked)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1SetAddressesIterator is returned from FilterSetAddresses and is used to iterate over the raw logs and unpacked data for SetAddresses events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1SetAddressesIterator struct {
	Event *Layer2ManagerV1SetAddresses // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1SetAddressesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1SetAddresses)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1SetAddresses)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1SetAddressesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1SetAddressesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1SetAddresses represents a SetAddresses event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1SetAddresses struct {
	L2Register             common.Address
	OperatorManagerFactory common.Address
	Ton                    common.Address
	Wton                   common.Address
	Dao                    common.Address
	DepositManager         common.Address
	SeigManager            common.Address
	SwapProxy              common.Address
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterSetAddresses is a free log retrieval operation binding the contract event 0x617f1cba5d7b9d5d3c81f0d5c78720ec2be227878bac84c0851b70e4cf61cb08.
//
// Solidity: event SetAddresses(address _l2Register, address _operatorManagerFactory, address _ton, address _wton, address _dao, address _depositManager, address _seigManager, address _swapProxy)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterSetAddresses(opts *bind.FilterOpts) (*Layer2ManagerV1SetAddressesIterator, error) {

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "SetAddresses")
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1SetAddressesIterator{contract: _Layer2ManagerV1.contract, event: "SetAddresses", logs: logs, sub: sub}, nil
}

// WatchSetAddresses is a free log subscription operation binding the contract event 0x617f1cba5d7b9d5d3c81f0d5c78720ec2be227878bac84c0851b70e4cf61cb08.
//
// Solidity: event SetAddresses(address _l2Register, address _operatorManagerFactory, address _ton, address _wton, address _dao, address _depositManager, address _seigManager, address _swapProxy)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchSetAddresses(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1SetAddresses) (event.Subscription, error) {

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "SetAddresses")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1SetAddresses)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "SetAddresses", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSetAddresses is a log parse operation binding the contract event 0x617f1cba5d7b9d5d3c81f0d5c78720ec2be227878bac84c0851b70e4cf61cb08.
//
// Solidity: event SetAddresses(address _l2Register, address _operatorManagerFactory, address _ton, address _wton, address _dao, address _depositManager, address _seigManager, address _swapProxy)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseSetAddresses(log types.Log) (*Layer2ManagerV1SetAddresses, error) {
	event := new(Layer2ManagerV1SetAddresses)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "SetAddresses", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1SetMinimumInitialDepositAmountIterator is returned from FilterSetMinimumInitialDepositAmount and is used to iterate over the raw logs and unpacked data for SetMinimumInitialDepositAmount events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1SetMinimumInitialDepositAmountIterator struct {
	Event *Layer2ManagerV1SetMinimumInitialDepositAmount // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1SetMinimumInitialDepositAmountIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1SetMinimumInitialDepositAmount)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1SetMinimumInitialDepositAmount)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1SetMinimumInitialDepositAmountIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1SetMinimumInitialDepositAmountIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1SetMinimumInitialDepositAmount represents a SetMinimumInitialDepositAmount event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1SetMinimumInitialDepositAmount struct {
	MinimumInitialDepositAmount *big.Int
	Raw                         types.Log // Blockchain specific contextual infos
}

// FilterSetMinimumInitialDepositAmount is a free log retrieval operation binding the contract event 0x50e01964a144ee7e40d80d0792ff2641558654ef906fd81502421f22bf6ca92c.
//
// Solidity: event SetMinimumInitialDepositAmount(uint256 _minimumInitialDepositAmount)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterSetMinimumInitialDepositAmount(opts *bind.FilterOpts) (*Layer2ManagerV1SetMinimumInitialDepositAmountIterator, error) {

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "SetMinimumInitialDepositAmount")
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1SetMinimumInitialDepositAmountIterator{contract: _Layer2ManagerV1.contract, event: "SetMinimumInitialDepositAmount", logs: logs, sub: sub}, nil
}

// WatchSetMinimumInitialDepositAmount is a free log subscription operation binding the contract event 0x50e01964a144ee7e40d80d0792ff2641558654ef906fd81502421f22bf6ca92c.
//
// Solidity: event SetMinimumInitialDepositAmount(uint256 _minimumInitialDepositAmount)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchSetMinimumInitialDepositAmount(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1SetMinimumInitialDepositAmount) (event.Subscription, error) {

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "SetMinimumInitialDepositAmount")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1SetMinimumInitialDepositAmount)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "SetMinimumInitialDepositAmount", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSetMinimumInitialDepositAmount is a log parse operation binding the contract event 0x50e01964a144ee7e40d80d0792ff2641558654ef906fd81502421f22bf6ca92c.
//
// Solidity: event SetMinimumInitialDepositAmount(uint256 _minimumInitialDepositAmount)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseSetMinimumInitialDepositAmount(log types.Log) (*Layer2ManagerV1SetMinimumInitialDepositAmount, error) {
	event := new(Layer2ManagerV1SetMinimumInitialDepositAmount)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "SetMinimumInitialDepositAmount", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1SetOperatorManagerFactoryIterator is returned from FilterSetOperatorManagerFactory and is used to iterate over the raw logs and unpacked data for SetOperatorManagerFactory events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1SetOperatorManagerFactoryIterator struct {
	Event *Layer2ManagerV1SetOperatorManagerFactory // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1SetOperatorManagerFactoryIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1SetOperatorManagerFactory)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1SetOperatorManagerFactory)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1SetOperatorManagerFactoryIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1SetOperatorManagerFactoryIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1SetOperatorManagerFactory represents a SetOperatorManagerFactory event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1SetOperatorManagerFactory struct {
	OperatorManagerFactory common.Address
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterSetOperatorManagerFactory is a free log retrieval operation binding the contract event 0x2342c0559c82fec746d047e08479a8b46bf45d4265f1db1c3a685b542c06f6a2.
//
// Solidity: event SetOperatorManagerFactory(address _operatorManagerFactory)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterSetOperatorManagerFactory(opts *bind.FilterOpts) (*Layer2ManagerV1SetOperatorManagerFactoryIterator, error) {

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "SetOperatorManagerFactory")
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1SetOperatorManagerFactoryIterator{contract: _Layer2ManagerV1.contract, event: "SetOperatorManagerFactory", logs: logs, sub: sub}, nil
}

// WatchSetOperatorManagerFactory is a free log subscription operation binding the contract event 0x2342c0559c82fec746d047e08479a8b46bf45d4265f1db1c3a685b542c06f6a2.
//
// Solidity: event SetOperatorManagerFactory(address _operatorManagerFactory)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchSetOperatorManagerFactory(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1SetOperatorManagerFactory) (event.Subscription, error) {

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "SetOperatorManagerFactory")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1SetOperatorManagerFactory)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "SetOperatorManagerFactory", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSetOperatorManagerFactory is a log parse operation binding the contract event 0x2342c0559c82fec746d047e08479a8b46bf45d4265f1db1c3a685b542c06f6a2.
//
// Solidity: event SetOperatorManagerFactory(address _operatorManagerFactory)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseSetOperatorManagerFactory(log types.Log) (*Layer2ManagerV1SetOperatorManagerFactory, error) {
	event := new(Layer2ManagerV1SetOperatorManagerFactory)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "SetOperatorManagerFactory", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Layer2ManagerV1UnpausedCandidateAddOnIterator is returned from FilterUnpausedCandidateAddOn and is used to iterate over the raw logs and unpacked data for UnpausedCandidateAddOn events raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1UnpausedCandidateAddOnIterator struct {
	Event *Layer2ManagerV1UnpausedCandidateAddOn // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Layer2ManagerV1UnpausedCandidateAddOnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Layer2ManagerV1UnpausedCandidateAddOn)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Layer2ManagerV1UnpausedCandidateAddOn)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Layer2ManagerV1UnpausedCandidateAddOnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Layer2ManagerV1UnpausedCandidateAddOnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Layer2ManagerV1UnpausedCandidateAddOn represents a UnpausedCandidateAddOn event raised by the Layer2ManagerV1 contract.
type Layer2ManagerV1UnpausedCandidateAddOn struct {
	RollupConfig   common.Address
	CandidateAddOn common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUnpausedCandidateAddOn is a free log retrieval operation binding the contract event 0xaf0a20fb16d60a15e38b6cd70fa0cce0bd2a744265663dfcbc90a02b360caf80.
//
// Solidity: event UnpausedCandidateAddOn(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) FilterUnpausedCandidateAddOn(opts *bind.FilterOpts) (*Layer2ManagerV1UnpausedCandidateAddOnIterator, error) {

	logs, sub, err := _Layer2ManagerV1.contract.FilterLogs(opts, "UnpausedCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return &Layer2ManagerV1UnpausedCandidateAddOnIterator{contract: _Layer2ManagerV1.contract, event: "UnpausedCandidateAddOn", logs: logs, sub: sub}, nil
}

// WatchUnpausedCandidateAddOn is a free log subscription operation binding the contract event 0xaf0a20fb16d60a15e38b6cd70fa0cce0bd2a744265663dfcbc90a02b360caf80.
//
// Solidity: event UnpausedCandidateAddOn(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) WatchUnpausedCandidateAddOn(opts *bind.WatchOpts, sink chan<- *Layer2ManagerV1UnpausedCandidateAddOn) (event.Subscription, error) {

	logs, sub, err := _Layer2ManagerV1.contract.WatchLogs(opts, "UnpausedCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Layer2ManagerV1UnpausedCandidateAddOn)
				if err := _Layer2ManagerV1.contract.UnpackLog(event, "UnpausedCandidateAddOn", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpausedCandidateAddOn is a log parse operation binding the contract event 0xaf0a20fb16d60a15e38b6cd70fa0cce0bd2a744265663dfcbc90a02b360caf80.
//
// Solidity: event UnpausedCandidateAddOn(address rollupConfig, address candidateAddOn)
func (_Layer2ManagerV1 *Layer2ManagerV1Filterer) ParseUnpausedCandidateAddOn(log types.Log) (*Layer2ManagerV1UnpausedCandidateAddOn, error) {
	event := new(Layer2ManagerV1UnpausedCandidateAddOn)
	if err := _Layer2ManagerV1.contract.UnpackLog(event, "UnpausedCandidateAddOn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
