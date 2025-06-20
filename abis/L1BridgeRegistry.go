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

// L1BridgeRegistryMetaData contains all meta data concerning the L1BridgeRegistry contract.
var L1BridgeRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"BridgeError\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"ChangeError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NonRejectedError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyRejectedError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PortalError\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"RegisterError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddressError\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"bridge\",\"type\":\"address\"}],\"name\":\"AddedBridge\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"portal\",\"type\":\"address\"}],\"name\":\"AddedPortal\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"type_\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"l2TON\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"ChangedType\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"type_\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"l2TON\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"RegisteredRollupConfig\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"RejectedCandidateAddOn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"RestoredCandidateAddOn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_layer2Manager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_seigManager\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_ton\",\"type\":\"address\"}],\"name\":\"SetAddresses\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit\",\"type\":\"bool\"}],\"name\":\"SetBlockingL2Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_seigniorageCommittee\",\"type\":\"address\"}],\"name\":\"SetSeigniorageCommittee\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MANAGER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"REGISTRANT_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"addAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"addManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"addRegistrant\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"aliveImplementation\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"}],\"name\":\"availableForRegistration\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"valid\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"changeType\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"getRollupInfo\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"type_\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"l2TON_\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"rejectedSeigs_\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit_\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"isAdmin\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"isManager\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"isRegistrant\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"isRejectedL2Deposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"isRejectedSeigs\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"rejectedSeigs\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"l1Bridge\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"l2TON\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"l2TonAddress\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"layer2Manager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"layer2TVL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pauseProxy\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"portal\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"proxyImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"registerRollupConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"}],\"name\":\"registerRollupConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"}],\"name\":\"registerRollupConfigByManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"registerRollupConfigByManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"registeredNames\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"rejectCandidateAddOn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"rejectRollupConfig\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"rejectedSeigs\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"removeAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"removeManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"removeRegistrant\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceRegistrant\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit\",\"type\":\"bool\"}],\"name\":\"restoreCandidateAddOn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeManager\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRegistrant\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"rollupInfo\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"rollupType\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"l2TON\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"rejectedSeigs\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"rejectedL2Deposit\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"rollupConfig\",\"type\":\"address\"}],\"name\":\"rollupType\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"rollupType_\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"seigManager\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"seigniorageCommittee\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"selectorImplementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_layer2Manager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_seigManager\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_ton\",\"type\":\"address\"}],\"name\":\"setAddresses\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_seigniorageCommittee\",\"type\":\"address\"}],\"name\":\"setSeigniorageCommittee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ton\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"transferAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// L1BridgeRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use L1BridgeRegistryMetaData.ABI instead.
var L1BridgeRegistryABI = L1BridgeRegistryMetaData.ABI

// L1BridgeRegistry is an auto generated Go binding around an Ethereum contract.
type L1BridgeRegistry struct {
	L1BridgeRegistryCaller     // Read-only binding to the contract
	L1BridgeRegistryTransactor // Write-only binding to the contract
	L1BridgeRegistryFilterer   // Log filterer for contract events
}

// L1BridgeRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BridgeRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BridgeRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BridgeRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BridgeRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1BridgeRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BridgeRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1BridgeRegistrySession struct {
	Contract     *L1BridgeRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1BridgeRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1BridgeRegistryCallerSession struct {
	Contract *L1BridgeRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// L1BridgeRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1BridgeRegistryTransactorSession struct {
	Contract     *L1BridgeRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// L1BridgeRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1BridgeRegistryRaw struct {
	Contract *L1BridgeRegistry // Generic contract binding to access the raw methods on
}

// L1BridgeRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1BridgeRegistryCallerRaw struct {
	Contract *L1BridgeRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// L1BridgeRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1BridgeRegistryTransactorRaw struct {
	Contract *L1BridgeRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1BridgeRegistry creates a new instance of L1BridgeRegistry, bound to a specific deployed contract.
func NewL1BridgeRegistry(address common.Address, backend bind.ContractBackend) (*L1BridgeRegistry, error) {
	contract, err := bindL1BridgeRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistry{L1BridgeRegistryCaller: L1BridgeRegistryCaller{contract: contract}, L1BridgeRegistryTransactor: L1BridgeRegistryTransactor{contract: contract}, L1BridgeRegistryFilterer: L1BridgeRegistryFilterer{contract: contract}}, nil
}

// NewL1BridgeRegistryCaller creates a new read-only instance of L1BridgeRegistry, bound to a specific deployed contract.
func NewL1BridgeRegistryCaller(address common.Address, caller bind.ContractCaller) (*L1BridgeRegistryCaller, error) {
	contract, err := bindL1BridgeRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryCaller{contract: contract}, nil
}

// NewL1BridgeRegistryTransactor creates a new write-only instance of L1BridgeRegistry, bound to a specific deployed contract.
func NewL1BridgeRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BridgeRegistryTransactor, error) {
	contract, err := bindL1BridgeRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryTransactor{contract: contract}, nil
}

// NewL1BridgeRegistryFilterer creates a new log filterer instance of L1BridgeRegistry, bound to a specific deployed contract.
func NewL1BridgeRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*L1BridgeRegistryFilterer, error) {
	contract, err := bindL1BridgeRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryFilterer{contract: contract}, nil
}

// bindL1BridgeRegistry binds a generic wrapper to an already deployed contract.
func bindL1BridgeRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1BridgeRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1BridgeRegistry *L1BridgeRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1BridgeRegistry.Contract.L1BridgeRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1BridgeRegistry *L1BridgeRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.L1BridgeRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1BridgeRegistry *L1BridgeRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.L1BridgeRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1BridgeRegistry *L1BridgeRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1BridgeRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1BridgeRegistry *L1BridgeRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1BridgeRegistry *L1BridgeRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistrySession) DEFAULTADMINROLE() ([32]byte, error) {
	return _L1BridgeRegistry.Contract.DEFAULTADMINROLE(&_L1BridgeRegistry.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _L1BridgeRegistry.Contract.DEFAULTADMINROLE(&_L1BridgeRegistry.CallOpts)
}

// MANAGERROLE is a free data retrieval call binding the contract method 0xec87621c.
//
// Solidity: function MANAGER_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) MANAGERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "MANAGER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MANAGERROLE is a free data retrieval call binding the contract method 0xec87621c.
//
// Solidity: function MANAGER_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistrySession) MANAGERROLE() ([32]byte, error) {
	return _L1BridgeRegistry.Contract.MANAGERROLE(&_L1BridgeRegistry.CallOpts)
}

// MANAGERROLE is a free data retrieval call binding the contract method 0xec87621c.
//
// Solidity: function MANAGER_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) MANAGERROLE() ([32]byte, error) {
	return _L1BridgeRegistry.Contract.MANAGERROLE(&_L1BridgeRegistry.CallOpts)
}

// REGISTRANTROLE is a free data retrieval call binding the contract method 0x6f5b142b.
//
// Solidity: function REGISTRANT_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) REGISTRANTROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "REGISTRANT_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// REGISTRANTROLE is a free data retrieval call binding the contract method 0x6f5b142b.
//
// Solidity: function REGISTRANT_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistrySession) REGISTRANTROLE() ([32]byte, error) {
	return _L1BridgeRegistry.Contract.REGISTRANTROLE(&_L1BridgeRegistry.CallOpts)
}

// REGISTRANTROLE is a free data retrieval call binding the contract method 0x6f5b142b.
//
// Solidity: function REGISTRANT_ROLE() view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) REGISTRANTROLE() ([32]byte, error) {
	return _L1BridgeRegistry.Contract.REGISTRANTROLE(&_L1BridgeRegistry.CallOpts)
}

// AliveImplementation is a free data retrieval call binding the contract method 0x550d01a3.
//
// Solidity: function aliveImplementation(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) AliveImplementation(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "aliveImplementation", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AliveImplementation is a free data retrieval call binding the contract method 0x550d01a3.
//
// Solidity: function aliveImplementation(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) AliveImplementation(arg0 common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.AliveImplementation(&_L1BridgeRegistry.CallOpts, arg0)
}

// AliveImplementation is a free data retrieval call binding the contract method 0x550d01a3.
//
// Solidity: function aliveImplementation(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) AliveImplementation(arg0 common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.AliveImplementation(&_L1BridgeRegistry.CallOpts, arg0)
}

// AvailableForRegistration is a free data retrieval call binding the contract method 0xc87957e7.
//
// Solidity: function availableForRegistration(address rollupConfig, uint8 _type) view returns(bool valid)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) AvailableForRegistration(opts *bind.CallOpts, rollupConfig common.Address, _type uint8) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "availableForRegistration", rollupConfig, _type)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AvailableForRegistration is a free data retrieval call binding the contract method 0xc87957e7.
//
// Solidity: function availableForRegistration(address rollupConfig, uint8 _type) view returns(bool valid)
func (_L1BridgeRegistry *L1BridgeRegistrySession) AvailableForRegistration(rollupConfig common.Address, _type uint8) (bool, error) {
	return _L1BridgeRegistry.Contract.AvailableForRegistration(&_L1BridgeRegistry.CallOpts, rollupConfig, _type)
}

// AvailableForRegistration is a free data retrieval call binding the contract method 0xc87957e7.
//
// Solidity: function availableForRegistration(address rollupConfig, uint8 _type) view returns(bool valid)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) AvailableForRegistration(rollupConfig common.Address, _type uint8) (bool, error) {
	return _L1BridgeRegistry.Contract.AvailableForRegistration(&_L1BridgeRegistry.CallOpts, rollupConfig, _type)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistrySession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _L1BridgeRegistry.Contract.GetRoleAdmin(&_L1BridgeRegistry.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _L1BridgeRegistry.Contract.GetRoleAdmin(&_L1BridgeRegistry.CallOpts, role)
}

// GetRollupInfo is a free data retrieval call binding the contract method 0xf5e9b26b.
//
// Solidity: function getRollupInfo(address rollupConfig) view returns(uint8 type_, address l2TON_, bool rejectedSeigs_, bool rejectedL2Deposit_, string name_)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) GetRollupInfo(opts *bind.CallOpts, rollupConfig common.Address) (struct {
	Type              uint8
	L2TON             common.Address
	RejectedSeigs     bool
	RejectedL2Deposit bool
	Name              string
}, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "getRollupInfo", rollupConfig)

	outstruct := new(struct {
		Type              uint8
		L2TON             common.Address
		RejectedSeigs     bool
		RejectedL2Deposit bool
		Name              string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Type = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.L2TON = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.RejectedSeigs = *abi.ConvertType(out[2], new(bool)).(*bool)
	outstruct.RejectedL2Deposit = *abi.ConvertType(out[3], new(bool)).(*bool)
	outstruct.Name = *abi.ConvertType(out[4], new(string)).(*string)

	return *outstruct, err

}

// GetRollupInfo is a free data retrieval call binding the contract method 0xf5e9b26b.
//
// Solidity: function getRollupInfo(address rollupConfig) view returns(uint8 type_, address l2TON_, bool rejectedSeigs_, bool rejectedL2Deposit_, string name_)
func (_L1BridgeRegistry *L1BridgeRegistrySession) GetRollupInfo(rollupConfig common.Address) (struct {
	Type              uint8
	L2TON             common.Address
	RejectedSeigs     bool
	RejectedL2Deposit bool
	Name              string
}, error) {
	return _L1BridgeRegistry.Contract.GetRollupInfo(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// GetRollupInfo is a free data retrieval call binding the contract method 0xf5e9b26b.
//
// Solidity: function getRollupInfo(address rollupConfig) view returns(uint8 type_, address l2TON_, bool rejectedSeigs_, bool rejectedL2Deposit_, string name_)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) GetRollupInfo(rollupConfig common.Address) (struct {
	Type              uint8
	L2TON             common.Address
	RejectedSeigs     bool
	RejectedL2Deposit bool
	Name              string
}, error) {
	return _L1BridgeRegistry.Contract.GetRollupInfo(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.HasRole(&_L1BridgeRegistry.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.HasRole(&_L1BridgeRegistry.CallOpts, role, account)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) IsAdmin(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "isAdmin", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) IsAdmin(account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsAdmin(&_L1BridgeRegistry.CallOpts, account)
}

// IsAdmin is a free data retrieval call binding the contract method 0x24d7806c.
//
// Solidity: function isAdmin(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) IsAdmin(account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsAdmin(&_L1BridgeRegistry.CallOpts, account)
}

// IsManager is a free data retrieval call binding the contract method 0xf3ae2415.
//
// Solidity: function isManager(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) IsManager(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "isManager", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsManager is a free data retrieval call binding the contract method 0xf3ae2415.
//
// Solidity: function isManager(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) IsManager(account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsManager(&_L1BridgeRegistry.CallOpts, account)
}

// IsManager is a free data retrieval call binding the contract method 0xf3ae2415.
//
// Solidity: function isManager(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) IsManager(account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsManager(&_L1BridgeRegistry.CallOpts, account)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "isOwner")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) IsOwner() (bool, error) {
	return _L1BridgeRegistry.Contract.IsOwner(&_L1BridgeRegistry.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) IsOwner() (bool, error) {
	return _L1BridgeRegistry.Contract.IsOwner(&_L1BridgeRegistry.CallOpts)
}

// IsRegistrant is a free data retrieval call binding the contract method 0x86ad05b6.
//
// Solidity: function isRegistrant(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) IsRegistrant(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "isRegistrant", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRegistrant is a free data retrieval call binding the contract method 0x86ad05b6.
//
// Solidity: function isRegistrant(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) IsRegistrant(account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsRegistrant(&_L1BridgeRegistry.CallOpts, account)
}

// IsRegistrant is a free data retrieval call binding the contract method 0x86ad05b6.
//
// Solidity: function isRegistrant(address account) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) IsRegistrant(account common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsRegistrant(&_L1BridgeRegistry.CallOpts, account)
}

// IsRejectedL2Deposit is a free data retrieval call binding the contract method 0xe65906e0.
//
// Solidity: function isRejectedL2Deposit(address rollupConfig) view returns(bool rejectedL2Deposit)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) IsRejectedL2Deposit(opts *bind.CallOpts, rollupConfig common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "isRejectedL2Deposit", rollupConfig)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRejectedL2Deposit is a free data retrieval call binding the contract method 0xe65906e0.
//
// Solidity: function isRejectedL2Deposit(address rollupConfig) view returns(bool rejectedL2Deposit)
func (_L1BridgeRegistry *L1BridgeRegistrySession) IsRejectedL2Deposit(rollupConfig common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsRejectedL2Deposit(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// IsRejectedL2Deposit is a free data retrieval call binding the contract method 0xe65906e0.
//
// Solidity: function isRejectedL2Deposit(address rollupConfig) view returns(bool rejectedL2Deposit)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) IsRejectedL2Deposit(rollupConfig common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsRejectedL2Deposit(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// IsRejectedSeigs is a free data retrieval call binding the contract method 0x88462d07.
//
// Solidity: function isRejectedSeigs(address rollupConfig) view returns(bool rejectedSeigs)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) IsRejectedSeigs(opts *bind.CallOpts, rollupConfig common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "isRejectedSeigs", rollupConfig)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRejectedSeigs is a free data retrieval call binding the contract method 0x88462d07.
//
// Solidity: function isRejectedSeigs(address rollupConfig) view returns(bool rejectedSeigs)
func (_L1BridgeRegistry *L1BridgeRegistrySession) IsRejectedSeigs(rollupConfig common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsRejectedSeigs(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// IsRejectedSeigs is a free data retrieval call binding the contract method 0x88462d07.
//
// Solidity: function isRejectedSeigs(address rollupConfig) view returns(bool rejectedSeigs)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) IsRejectedSeigs(rollupConfig common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.IsRejectedSeigs(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// L1Bridge is a free data retrieval call binding the contract method 0xb30347c0.
//
// Solidity: function l1Bridge(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) L1Bridge(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "l1Bridge", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// L1Bridge is a free data retrieval call binding the contract method 0xb30347c0.
//
// Solidity: function l1Bridge(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) L1Bridge(arg0 common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.L1Bridge(&_L1BridgeRegistry.CallOpts, arg0)
}

// L1Bridge is a free data retrieval call binding the contract method 0xb30347c0.
//
// Solidity: function l1Bridge(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) L1Bridge(arg0 common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.L1Bridge(&_L1BridgeRegistry.CallOpts, arg0)
}

// L2TON is a free data retrieval call binding the contract method 0x4b7aa5d4.
//
// Solidity: function l2TON(address rollupConfig) view returns(address l2TonAddress)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) L2TON(opts *bind.CallOpts, rollupConfig common.Address) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "l2TON", rollupConfig)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2TON is a free data retrieval call binding the contract method 0x4b7aa5d4.
//
// Solidity: function l2TON(address rollupConfig) view returns(address l2TonAddress)
func (_L1BridgeRegistry *L1BridgeRegistrySession) L2TON(rollupConfig common.Address) (common.Address, error) {
	return _L1BridgeRegistry.Contract.L2TON(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// L2TON is a free data retrieval call binding the contract method 0x4b7aa5d4.
//
// Solidity: function l2TON(address rollupConfig) view returns(address l2TonAddress)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) L2TON(rollupConfig common.Address) (common.Address, error) {
	return _L1BridgeRegistry.Contract.L2TON(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// Layer2Manager is a free data retrieval call binding the contract method 0x16b5d5bd.
//
// Solidity: function layer2Manager() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) Layer2Manager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "layer2Manager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Layer2Manager is a free data retrieval call binding the contract method 0x16b5d5bd.
//
// Solidity: function layer2Manager() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistrySession) Layer2Manager() (common.Address, error) {
	return _L1BridgeRegistry.Contract.Layer2Manager(&_L1BridgeRegistry.CallOpts)
}

// Layer2Manager is a free data retrieval call binding the contract method 0x16b5d5bd.
//
// Solidity: function layer2Manager() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) Layer2Manager() (common.Address, error) {
	return _L1BridgeRegistry.Contract.Layer2Manager(&_L1BridgeRegistry.CallOpts)
}

// Layer2TVL is a free data retrieval call binding the contract method 0xc81ae6db.
//
// Solidity: function layer2TVL(address rollupConfig) view returns(uint256 amount)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) Layer2TVL(opts *bind.CallOpts, rollupConfig common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "layer2TVL", rollupConfig)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Layer2TVL is a free data retrieval call binding the contract method 0xc81ae6db.
//
// Solidity: function layer2TVL(address rollupConfig) view returns(uint256 amount)
func (_L1BridgeRegistry *L1BridgeRegistrySession) Layer2TVL(rollupConfig common.Address) (*big.Int, error) {
	return _L1BridgeRegistry.Contract.Layer2TVL(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// Layer2TVL is a free data retrieval call binding the contract method 0xc81ae6db.
//
// Solidity: function layer2TVL(address rollupConfig) view returns(uint256 amount)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) Layer2TVL(rollupConfig common.Address) (*big.Int, error) {
	return _L1BridgeRegistry.Contract.Layer2TVL(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// PauseProxy is a free data retrieval call binding the contract method 0x63a8fd89.
//
// Solidity: function pauseProxy() view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) PauseProxy(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "pauseProxy")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PauseProxy is a free data retrieval call binding the contract method 0x63a8fd89.
//
// Solidity: function pauseProxy() view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) PauseProxy() (bool, error) {
	return _L1BridgeRegistry.Contract.PauseProxy(&_L1BridgeRegistry.CallOpts)
}

// PauseProxy is a free data retrieval call binding the contract method 0x63a8fd89.
//
// Solidity: function pauseProxy() view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) PauseProxy() (bool, error) {
	return _L1BridgeRegistry.Contract.PauseProxy(&_L1BridgeRegistry.CallOpts)
}

// Portal is a free data retrieval call binding the contract method 0xa2cc0979.
//
// Solidity: function portal(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) Portal(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "portal", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Portal is a free data retrieval call binding the contract method 0xa2cc0979.
//
// Solidity: function portal(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) Portal(arg0 common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.Portal(&_L1BridgeRegistry.CallOpts, arg0)
}

// Portal is a free data retrieval call binding the contract method 0xa2cc0979.
//
// Solidity: function portal(address ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) Portal(arg0 common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.Portal(&_L1BridgeRegistry.CallOpts, arg0)
}

// ProxyImplementation is a free data retrieval call binding the contract method 0xb911135f.
//
// Solidity: function proxyImplementation(uint256 ) view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) ProxyImplementation(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "proxyImplementation", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProxyImplementation is a free data retrieval call binding the contract method 0xb911135f.
//
// Solidity: function proxyImplementation(uint256 ) view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistrySession) ProxyImplementation(arg0 *big.Int) (common.Address, error) {
	return _L1BridgeRegistry.Contract.ProxyImplementation(&_L1BridgeRegistry.CallOpts, arg0)
}

// ProxyImplementation is a free data retrieval call binding the contract method 0xb911135f.
//
// Solidity: function proxyImplementation(uint256 ) view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) ProxyImplementation(arg0 *big.Int) (common.Address, error) {
	return _L1BridgeRegistry.Contract.ProxyImplementation(&_L1BridgeRegistry.CallOpts, arg0)
}

// RegisteredNames is a free data retrieval call binding the contract method 0x25b9535f.
//
// Solidity: function registeredNames(bytes32 ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) RegisteredNames(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "registeredNames", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// RegisteredNames is a free data retrieval call binding the contract method 0x25b9535f.
//
// Solidity: function registeredNames(bytes32 ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) RegisteredNames(arg0 [32]byte) (bool, error) {
	return _L1BridgeRegistry.Contract.RegisteredNames(&_L1BridgeRegistry.CallOpts, arg0)
}

// RegisteredNames is a free data retrieval call binding the contract method 0x25b9535f.
//
// Solidity: function registeredNames(bytes32 ) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) RegisteredNames(arg0 [32]byte) (bool, error) {
	return _L1BridgeRegistry.Contract.RegisteredNames(&_L1BridgeRegistry.CallOpts, arg0)
}

// RejectRollupConfig is a free data retrieval call binding the contract method 0x2977c31f.
//
// Solidity: function rejectRollupConfig(address rollupConfig) view returns(bool rejectedSeigs)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) RejectRollupConfig(opts *bind.CallOpts, rollupConfig common.Address) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "rejectRollupConfig", rollupConfig)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// RejectRollupConfig is a free data retrieval call binding the contract method 0x2977c31f.
//
// Solidity: function rejectRollupConfig(address rollupConfig) view returns(bool rejectedSeigs)
func (_L1BridgeRegistry *L1BridgeRegistrySession) RejectRollupConfig(rollupConfig common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.RejectRollupConfig(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// RejectRollupConfig is a free data retrieval call binding the contract method 0x2977c31f.
//
// Solidity: function rejectRollupConfig(address rollupConfig) view returns(bool rejectedSeigs)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) RejectRollupConfig(rollupConfig common.Address) (bool, error) {
	return _L1BridgeRegistry.Contract.RejectRollupConfig(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// RollupInfo is a free data retrieval call binding the contract method 0xd3c215d8.
//
// Solidity: function rollupInfo(address ) view returns(uint8 rollupType, address l2TON, bool rejectedSeigs, bool rejectedL2Deposit, string name)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) RollupInfo(opts *bind.CallOpts, arg0 common.Address) (struct {
	RollupType        uint8
	L2TON             common.Address
	RejectedSeigs     bool
	RejectedL2Deposit bool
	Name              string
}, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "rollupInfo", arg0)

	outstruct := new(struct {
		RollupType        uint8
		L2TON             common.Address
		RejectedSeigs     bool
		RejectedL2Deposit bool
		Name              string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.RollupType = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.L2TON = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.RejectedSeigs = *abi.ConvertType(out[2], new(bool)).(*bool)
	outstruct.RejectedL2Deposit = *abi.ConvertType(out[3], new(bool)).(*bool)
	outstruct.Name = *abi.ConvertType(out[4], new(string)).(*string)

	return *outstruct, err

}

// RollupInfo is a free data retrieval call binding the contract method 0xd3c215d8.
//
// Solidity: function rollupInfo(address ) view returns(uint8 rollupType, address l2TON, bool rejectedSeigs, bool rejectedL2Deposit, string name)
func (_L1BridgeRegistry *L1BridgeRegistrySession) RollupInfo(arg0 common.Address) (struct {
	RollupType        uint8
	L2TON             common.Address
	RejectedSeigs     bool
	RejectedL2Deposit bool
	Name              string
}, error) {
	return _L1BridgeRegistry.Contract.RollupInfo(&_L1BridgeRegistry.CallOpts, arg0)
}

// RollupInfo is a free data retrieval call binding the contract method 0xd3c215d8.
//
// Solidity: function rollupInfo(address ) view returns(uint8 rollupType, address l2TON, bool rejectedSeigs, bool rejectedL2Deposit, string name)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) RollupInfo(arg0 common.Address) (struct {
	RollupType        uint8
	L2TON             common.Address
	RejectedSeigs     bool
	RejectedL2Deposit bool
	Name              string
}, error) {
	return _L1BridgeRegistry.Contract.RollupInfo(&_L1BridgeRegistry.CallOpts, arg0)
}

// RollupType is a free data retrieval call binding the contract method 0x391e3b37.
//
// Solidity: function rollupType(address rollupConfig) view returns(uint8 rollupType_)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) RollupType(opts *bind.CallOpts, rollupConfig common.Address) (uint8, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "rollupType", rollupConfig)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// RollupType is a free data retrieval call binding the contract method 0x391e3b37.
//
// Solidity: function rollupType(address rollupConfig) view returns(uint8 rollupType_)
func (_L1BridgeRegistry *L1BridgeRegistrySession) RollupType(rollupConfig common.Address) (uint8, error) {
	return _L1BridgeRegistry.Contract.RollupType(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// RollupType is a free data retrieval call binding the contract method 0x391e3b37.
//
// Solidity: function rollupType(address rollupConfig) view returns(uint8 rollupType_)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) RollupType(rollupConfig common.Address) (uint8, error) {
	return _L1BridgeRegistry.Contract.RollupType(&_L1BridgeRegistry.CallOpts, rollupConfig)
}

// SeigManager is a free data retrieval call binding the contract method 0x6fb7f558.
//
// Solidity: function seigManager() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) SeigManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "seigManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SeigManager is a free data retrieval call binding the contract method 0x6fb7f558.
//
// Solidity: function seigManager() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistrySession) SeigManager() (common.Address, error) {
	return _L1BridgeRegistry.Contract.SeigManager(&_L1BridgeRegistry.CallOpts)
}

// SeigManager is a free data retrieval call binding the contract method 0x6fb7f558.
//
// Solidity: function seigManager() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) SeigManager() (common.Address, error) {
	return _L1BridgeRegistry.Contract.SeigManager(&_L1BridgeRegistry.CallOpts)
}

// SeigniorageCommittee is a free data retrieval call binding the contract method 0x35583e6c.
//
// Solidity: function seigniorageCommittee() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) SeigniorageCommittee(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "seigniorageCommittee")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SeigniorageCommittee is a free data retrieval call binding the contract method 0x35583e6c.
//
// Solidity: function seigniorageCommittee() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistrySession) SeigniorageCommittee() (common.Address, error) {
	return _L1BridgeRegistry.Contract.SeigniorageCommittee(&_L1BridgeRegistry.CallOpts)
}

// SeigniorageCommittee is a free data retrieval call binding the contract method 0x35583e6c.
//
// Solidity: function seigniorageCommittee() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) SeigniorageCommittee() (common.Address, error) {
	return _L1BridgeRegistry.Contract.SeigniorageCommittee(&_L1BridgeRegistry.CallOpts)
}

// SelectorImplementation is a free data retrieval call binding the contract method 0x50d2a276.
//
// Solidity: function selectorImplementation(bytes4 ) view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) SelectorImplementation(opts *bind.CallOpts, arg0 [4]byte) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "selectorImplementation", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SelectorImplementation is a free data retrieval call binding the contract method 0x50d2a276.
//
// Solidity: function selectorImplementation(bytes4 ) view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistrySession) SelectorImplementation(arg0 [4]byte) (common.Address, error) {
	return _L1BridgeRegistry.Contract.SelectorImplementation(&_L1BridgeRegistry.CallOpts, arg0)
}

// SelectorImplementation is a free data retrieval call binding the contract method 0x50d2a276.
//
// Solidity: function selectorImplementation(bytes4 ) view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) SelectorImplementation(arg0 [4]byte) (common.Address, error) {
	return _L1BridgeRegistry.Contract.SelectorImplementation(&_L1BridgeRegistry.CallOpts, arg0)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistrySession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _L1BridgeRegistry.Contract.SupportsInterface(&_L1BridgeRegistry.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _L1BridgeRegistry.Contract.SupportsInterface(&_L1BridgeRegistry.CallOpts, interfaceId)
}

// Ton is a free data retrieval call binding the contract method 0xcc48b947.
//
// Solidity: function ton() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCaller) Ton(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1BridgeRegistry.contract.Call(opts, &out, "ton")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Ton is a free data retrieval call binding the contract method 0xcc48b947.
//
// Solidity: function ton() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistrySession) Ton() (common.Address, error) {
	return _L1BridgeRegistry.Contract.Ton(&_L1BridgeRegistry.CallOpts)
}

// Ton is a free data retrieval call binding the contract method 0xcc48b947.
//
// Solidity: function ton() view returns(address)
func (_L1BridgeRegistry *L1BridgeRegistryCallerSession) Ton() (common.Address, error) {
	return _L1BridgeRegistry.Contract.Ton(&_L1BridgeRegistry.CallOpts)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) AddAdmin(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "addAdmin", account)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) AddAdmin(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.AddAdmin(&_L1BridgeRegistry.TransactOpts, account)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) AddAdmin(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.AddAdmin(&_L1BridgeRegistry.TransactOpts, account)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) AddManager(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "addManager", account)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) AddManager(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.AddManager(&_L1BridgeRegistry.TransactOpts, account)
}

// AddManager is a paid mutator transaction binding the contract method 0x2d06177a.
//
// Solidity: function addManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) AddManager(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.AddManager(&_L1BridgeRegistry.TransactOpts, account)
}

// AddRegistrant is a paid mutator transaction binding the contract method 0xbbfcfc1e.
//
// Solidity: function addRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) AddRegistrant(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "addRegistrant", account)
}

// AddRegistrant is a paid mutator transaction binding the contract method 0xbbfcfc1e.
//
// Solidity: function addRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) AddRegistrant(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.AddRegistrant(&_L1BridgeRegistry.TransactOpts, account)
}

// AddRegistrant is a paid mutator transaction binding the contract method 0xbbfcfc1e.
//
// Solidity: function addRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) AddRegistrant(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.AddRegistrant(&_L1BridgeRegistry.TransactOpts, account)
}

// ChangeType is a paid mutator transaction binding the contract method 0x6af685c1.
//
// Solidity: function changeType(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) ChangeType(opts *bind.TransactOpts, rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "changeType", rollupConfig, _type, _l2TON, _name)
}

// ChangeType is a paid mutator transaction binding the contract method 0x6af685c1.
//
// Solidity: function changeType(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) ChangeType(rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.ChangeType(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON, _name)
}

// ChangeType is a paid mutator transaction binding the contract method 0x6af685c1.
//
// Solidity: function changeType(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) ChangeType(rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.ChangeType(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON, _name)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.GrantRole(&_L1BridgeRegistry.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.GrantRole(&_L1BridgeRegistry.TransactOpts, role, account)
}

// RegisterRollupConfig is a paid mutator transaction binding the contract method 0x1d4c9c88.
//
// Solidity: function registerRollupConfig(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RegisterRollupConfig(opts *bind.TransactOpts, rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "registerRollupConfig", rollupConfig, _type, _l2TON, _name)
}

// RegisterRollupConfig is a paid mutator transaction binding the contract method 0x1d4c9c88.
//
// Solidity: function registerRollupConfig(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RegisterRollupConfig(rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfig(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON, _name)
}

// RegisterRollupConfig is a paid mutator transaction binding the contract method 0x1d4c9c88.
//
// Solidity: function registerRollupConfig(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RegisterRollupConfig(rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfig(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON, _name)
}

// RegisterRollupConfig0 is a paid mutator transaction binding the contract method 0x6e91948d.
//
// Solidity: function registerRollupConfig(address rollupConfig, uint8 _type, address _l2TON) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RegisterRollupConfig0(opts *bind.TransactOpts, rollupConfig common.Address, _type uint8, _l2TON common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "registerRollupConfig0", rollupConfig, _type, _l2TON)
}

// RegisterRollupConfig0 is a paid mutator transaction binding the contract method 0x6e91948d.
//
// Solidity: function registerRollupConfig(address rollupConfig, uint8 _type, address _l2TON) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RegisterRollupConfig0(rollupConfig common.Address, _type uint8, _l2TON common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfig0(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON)
}

// RegisterRollupConfig0 is a paid mutator transaction binding the contract method 0x6e91948d.
//
// Solidity: function registerRollupConfig(address rollupConfig, uint8 _type, address _l2TON) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RegisterRollupConfig0(rollupConfig common.Address, _type uint8, _l2TON common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfig0(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON)
}

// RegisterRollupConfigByManager is a paid mutator transaction binding the contract method 0x2f02c3e0.
//
// Solidity: function registerRollupConfigByManager(address rollupConfig, uint8 _type, address _l2TON) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RegisterRollupConfigByManager(opts *bind.TransactOpts, rollupConfig common.Address, _type uint8, _l2TON common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "registerRollupConfigByManager", rollupConfig, _type, _l2TON)
}

// RegisterRollupConfigByManager is a paid mutator transaction binding the contract method 0x2f02c3e0.
//
// Solidity: function registerRollupConfigByManager(address rollupConfig, uint8 _type, address _l2TON) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RegisterRollupConfigByManager(rollupConfig common.Address, _type uint8, _l2TON common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfigByManager(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON)
}

// RegisterRollupConfigByManager is a paid mutator transaction binding the contract method 0x2f02c3e0.
//
// Solidity: function registerRollupConfigByManager(address rollupConfig, uint8 _type, address _l2TON) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RegisterRollupConfigByManager(rollupConfig common.Address, _type uint8, _l2TON common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfigByManager(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON)
}

// RegisterRollupConfigByManager0 is a paid mutator transaction binding the contract method 0x3163b4b5.
//
// Solidity: function registerRollupConfigByManager(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RegisterRollupConfigByManager0(opts *bind.TransactOpts, rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "registerRollupConfigByManager0", rollupConfig, _type, _l2TON, _name)
}

// RegisterRollupConfigByManager0 is a paid mutator transaction binding the contract method 0x3163b4b5.
//
// Solidity: function registerRollupConfigByManager(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RegisterRollupConfigByManager0(rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfigByManager0(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON, _name)
}

// RegisterRollupConfigByManager0 is a paid mutator transaction binding the contract method 0x3163b4b5.
//
// Solidity: function registerRollupConfigByManager(address rollupConfig, uint8 _type, address _l2TON, string _name) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RegisterRollupConfigByManager0(rollupConfig common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RegisterRollupConfigByManager0(&_L1BridgeRegistry.TransactOpts, rollupConfig, _type, _l2TON, _name)
}

// RejectCandidateAddOn is a paid mutator transaction binding the contract method 0x4b49264c.
//
// Solidity: function rejectCandidateAddOn(address rollupConfig) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RejectCandidateAddOn(opts *bind.TransactOpts, rollupConfig common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "rejectCandidateAddOn", rollupConfig)
}

// RejectCandidateAddOn is a paid mutator transaction binding the contract method 0x4b49264c.
//
// Solidity: function rejectCandidateAddOn(address rollupConfig) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RejectCandidateAddOn(rollupConfig common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RejectCandidateAddOn(&_L1BridgeRegistry.TransactOpts, rollupConfig)
}

// RejectCandidateAddOn is a paid mutator transaction binding the contract method 0x4b49264c.
//
// Solidity: function rejectCandidateAddOn(address rollupConfig) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RejectCandidateAddOn(rollupConfig common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RejectCandidateAddOn(&_L1BridgeRegistry.TransactOpts, rollupConfig)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RemoveAdmin(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "removeAdmin", account)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RemoveAdmin(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RemoveAdmin(&_L1BridgeRegistry.TransactOpts, account)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RemoveAdmin(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RemoveAdmin(&_L1BridgeRegistry.TransactOpts, account)
}

// RemoveManager is a paid mutator transaction binding the contract method 0xac18de43.
//
// Solidity: function removeManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RemoveManager(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "removeManager", account)
}

// RemoveManager is a paid mutator transaction binding the contract method 0xac18de43.
//
// Solidity: function removeManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RemoveManager(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RemoveManager(&_L1BridgeRegistry.TransactOpts, account)
}

// RemoveManager is a paid mutator transaction binding the contract method 0xac18de43.
//
// Solidity: function removeManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RemoveManager(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RemoveManager(&_L1BridgeRegistry.TransactOpts, account)
}

// RemoveRegistrant is a paid mutator transaction binding the contract method 0x8cfc9347.
//
// Solidity: function removeRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RemoveRegistrant(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "removeRegistrant", account)
}

// RemoveRegistrant is a paid mutator transaction binding the contract method 0x8cfc9347.
//
// Solidity: function removeRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RemoveRegistrant(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RemoveRegistrant(&_L1BridgeRegistry.TransactOpts, account)
}

// RemoveRegistrant is a paid mutator transaction binding the contract method 0x8cfc9347.
//
// Solidity: function removeRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RemoveRegistrant(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RemoveRegistrant(&_L1BridgeRegistry.TransactOpts, account)
}

// RenounceManager is a paid mutator transaction binding the contract method 0xf8b91abe.
//
// Solidity: function renounceManager() returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RenounceManager(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "renounceManager")
}

// RenounceManager is a paid mutator transaction binding the contract method 0xf8b91abe.
//
// Solidity: function renounceManager() returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RenounceManager() (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceManager(&_L1BridgeRegistry.TransactOpts)
}

// RenounceManager is a paid mutator transaction binding the contract method 0xf8b91abe.
//
// Solidity: function renounceManager() returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RenounceManager() (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceManager(&_L1BridgeRegistry.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RenounceOwnership() (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceOwnership(&_L1BridgeRegistry.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceOwnership(&_L1BridgeRegistry.TransactOpts)
}

// RenounceRegistrant is a paid mutator transaction binding the contract method 0x35556fe3.
//
// Solidity: function renounceRegistrant() returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RenounceRegistrant(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "renounceRegistrant")
}

// RenounceRegistrant is a paid mutator transaction binding the contract method 0x35556fe3.
//
// Solidity: function renounceRegistrant() returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RenounceRegistrant() (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceRegistrant(&_L1BridgeRegistry.TransactOpts)
}

// RenounceRegistrant is a paid mutator transaction binding the contract method 0x35556fe3.
//
// Solidity: function renounceRegistrant() returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RenounceRegistrant() (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceRegistrant(&_L1BridgeRegistry.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceRole(&_L1BridgeRegistry.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RenounceRole(&_L1BridgeRegistry.TransactOpts, role, account)
}

// RestoreCandidateAddOn is a paid mutator transaction binding the contract method 0x4fb1aee6.
//
// Solidity: function restoreCandidateAddOn(address rollupConfig, bool rejectedL2Deposit) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RestoreCandidateAddOn(opts *bind.TransactOpts, rollupConfig common.Address, rejectedL2Deposit bool) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "restoreCandidateAddOn", rollupConfig, rejectedL2Deposit)
}

// RestoreCandidateAddOn is a paid mutator transaction binding the contract method 0x4fb1aee6.
//
// Solidity: function restoreCandidateAddOn(address rollupConfig, bool rejectedL2Deposit) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RestoreCandidateAddOn(rollupConfig common.Address, rejectedL2Deposit bool) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RestoreCandidateAddOn(&_L1BridgeRegistry.TransactOpts, rollupConfig, rejectedL2Deposit)
}

// RestoreCandidateAddOn is a paid mutator transaction binding the contract method 0x4fb1aee6.
//
// Solidity: function restoreCandidateAddOn(address rollupConfig, bool rejectedL2Deposit) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RestoreCandidateAddOn(rollupConfig common.Address, rejectedL2Deposit bool) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RestoreCandidateAddOn(&_L1BridgeRegistry.TransactOpts, rollupConfig, rejectedL2Deposit)
}

// RevokeManager is a paid mutator transaction binding the contract method 0x377e32e6.
//
// Solidity: function revokeManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RevokeManager(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "revokeManager", account)
}

// RevokeManager is a paid mutator transaction binding the contract method 0x377e32e6.
//
// Solidity: function revokeManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RevokeManager(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RevokeManager(&_L1BridgeRegistry.TransactOpts, account)
}

// RevokeManager is a paid mutator transaction binding the contract method 0x377e32e6.
//
// Solidity: function revokeManager(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RevokeManager(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RevokeManager(&_L1BridgeRegistry.TransactOpts, account)
}

// RevokeRegistrant is a paid mutator transaction binding the contract method 0xd00491c6.
//
// Solidity: function revokeRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RevokeRegistrant(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "revokeRegistrant", account)
}

// RevokeRegistrant is a paid mutator transaction binding the contract method 0xd00491c6.
//
// Solidity: function revokeRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RevokeRegistrant(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RevokeRegistrant(&_L1BridgeRegistry.TransactOpts, account)
}

// RevokeRegistrant is a paid mutator transaction binding the contract method 0xd00491c6.
//
// Solidity: function revokeRegistrant(address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RevokeRegistrant(account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RevokeRegistrant(&_L1BridgeRegistry.TransactOpts, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RevokeRole(&_L1BridgeRegistry.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.RevokeRole(&_L1BridgeRegistry.TransactOpts, role, account)
}

// SetAddresses is a paid mutator transaction binding the contract method 0x363bf964.
//
// Solidity: function setAddresses(address _layer2Manager, address _seigManager, address _ton) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) SetAddresses(opts *bind.TransactOpts, _layer2Manager common.Address, _seigManager common.Address, _ton common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "setAddresses", _layer2Manager, _seigManager, _ton)
}

// SetAddresses is a paid mutator transaction binding the contract method 0x363bf964.
//
// Solidity: function setAddresses(address _layer2Manager, address _seigManager, address _ton) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) SetAddresses(_layer2Manager common.Address, _seigManager common.Address, _ton common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.SetAddresses(&_L1BridgeRegistry.TransactOpts, _layer2Manager, _seigManager, _ton)
}

// SetAddresses is a paid mutator transaction binding the contract method 0x363bf964.
//
// Solidity: function setAddresses(address _layer2Manager, address _seigManager, address _ton) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) SetAddresses(_layer2Manager common.Address, _seigManager common.Address, _ton common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.SetAddresses(&_L1BridgeRegistry.TransactOpts, _layer2Manager, _seigManager, _ton)
}

// SetSeigniorageCommittee is a paid mutator transaction binding the contract method 0x1c564985.
//
// Solidity: function setSeigniorageCommittee(address _seigniorageCommittee) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) SetSeigniorageCommittee(opts *bind.TransactOpts, _seigniorageCommittee common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "setSeigniorageCommittee", _seigniorageCommittee)
}

// SetSeigniorageCommittee is a paid mutator transaction binding the contract method 0x1c564985.
//
// Solidity: function setSeigniorageCommittee(address _seigniorageCommittee) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) SetSeigniorageCommittee(_seigniorageCommittee common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.SetSeigniorageCommittee(&_L1BridgeRegistry.TransactOpts, _seigniorageCommittee)
}

// SetSeigniorageCommittee is a paid mutator transaction binding the contract method 0x1c564985.
//
// Solidity: function setSeigniorageCommittee(address _seigniorageCommittee) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) SetSeigniorageCommittee(_seigniorageCommittee common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.SetSeigniorageCommittee(&_L1BridgeRegistry.TransactOpts, _seigniorageCommittee)
}

// TransferAdmin is a paid mutator transaction binding the contract method 0x75829def.
//
// Solidity: function transferAdmin(address newAdmin) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) TransferAdmin(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "transferAdmin", newAdmin)
}

// TransferAdmin is a paid mutator transaction binding the contract method 0x75829def.
//
// Solidity: function transferAdmin(address newAdmin) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) TransferAdmin(newAdmin common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.TransferAdmin(&_L1BridgeRegistry.TransactOpts, newAdmin)
}

// TransferAdmin is a paid mutator transaction binding the contract method 0x75829def.
//
// Solidity: function transferAdmin(address newAdmin) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) TransferAdmin(newAdmin common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.TransferAdmin(&_L1BridgeRegistry.TransactOpts, newAdmin)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newAdmin) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactor) TransferOwnership(opts *bind.TransactOpts, newAdmin common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.contract.Transact(opts, "transferOwnership", newAdmin)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newAdmin) returns()
func (_L1BridgeRegistry *L1BridgeRegistrySession) TransferOwnership(newAdmin common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.TransferOwnership(&_L1BridgeRegistry.TransactOpts, newAdmin)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newAdmin) returns()
func (_L1BridgeRegistry *L1BridgeRegistryTransactorSession) TransferOwnership(newAdmin common.Address) (*types.Transaction, error) {
	return _L1BridgeRegistry.Contract.TransferOwnership(&_L1BridgeRegistry.TransactOpts, newAdmin)
}

// L1BridgeRegistryAddedBridgeIterator is returned from FilterAddedBridge and is used to iterate over the raw logs and unpacked data for AddedBridge events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryAddedBridgeIterator struct {
	Event *L1BridgeRegistryAddedBridge // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryAddedBridgeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryAddedBridge)
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
		it.Event = new(L1BridgeRegistryAddedBridge)
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
func (it *L1BridgeRegistryAddedBridgeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryAddedBridgeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryAddedBridge represents a AddedBridge event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryAddedBridge struct {
	RollupConfig common.Address
	Bridge       common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterAddedBridge is a free log retrieval operation binding the contract event 0x43251870f9292d3c0a61a78baa68c9f29d70ea868008a66d8afdb4d574f2c231.
//
// Solidity: event AddedBridge(address rollupConfig, address bridge)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterAddedBridge(opts *bind.FilterOpts) (*L1BridgeRegistryAddedBridgeIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "AddedBridge")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryAddedBridgeIterator{contract: _L1BridgeRegistry.contract, event: "AddedBridge", logs: logs, sub: sub}, nil
}

// WatchAddedBridge is a free log subscription operation binding the contract event 0x43251870f9292d3c0a61a78baa68c9f29d70ea868008a66d8afdb4d574f2c231.
//
// Solidity: event AddedBridge(address rollupConfig, address bridge)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchAddedBridge(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryAddedBridge) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "AddedBridge")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryAddedBridge)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "AddedBridge", log); err != nil {
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

// ParseAddedBridge is a log parse operation binding the contract event 0x43251870f9292d3c0a61a78baa68c9f29d70ea868008a66d8afdb4d574f2c231.
//
// Solidity: event AddedBridge(address rollupConfig, address bridge)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseAddedBridge(log types.Log) (*L1BridgeRegistryAddedBridge, error) {
	event := new(L1BridgeRegistryAddedBridge)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "AddedBridge", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryAddedPortalIterator is returned from FilterAddedPortal and is used to iterate over the raw logs and unpacked data for AddedPortal events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryAddedPortalIterator struct {
	Event *L1BridgeRegistryAddedPortal // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryAddedPortalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryAddedPortal)
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
		it.Event = new(L1BridgeRegistryAddedPortal)
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
func (it *L1BridgeRegistryAddedPortalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryAddedPortalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryAddedPortal represents a AddedPortal event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryAddedPortal struct {
	RollupConfig common.Address
	Portal       common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterAddedPortal is a free log retrieval operation binding the contract event 0xf8bdba0336133f3634bf70846cd60e94e3b9dd7ce3a59c5831262cfdb636d006.
//
// Solidity: event AddedPortal(address rollupConfig, address portal)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterAddedPortal(opts *bind.FilterOpts) (*L1BridgeRegistryAddedPortalIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "AddedPortal")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryAddedPortalIterator{contract: _L1BridgeRegistry.contract, event: "AddedPortal", logs: logs, sub: sub}, nil
}

// WatchAddedPortal is a free log subscription operation binding the contract event 0xf8bdba0336133f3634bf70846cd60e94e3b9dd7ce3a59c5831262cfdb636d006.
//
// Solidity: event AddedPortal(address rollupConfig, address portal)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchAddedPortal(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryAddedPortal) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "AddedPortal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryAddedPortal)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "AddedPortal", log); err != nil {
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

// ParseAddedPortal is a log parse operation binding the contract event 0xf8bdba0336133f3634bf70846cd60e94e3b9dd7ce3a59c5831262cfdb636d006.
//
// Solidity: event AddedPortal(address rollupConfig, address portal)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseAddedPortal(log types.Log) (*L1BridgeRegistryAddedPortal, error) {
	event := new(L1BridgeRegistryAddedPortal)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "AddedPortal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryChangedTypeIterator is returned from FilterChangedType and is used to iterate over the raw logs and unpacked data for ChangedType events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryChangedTypeIterator struct {
	Event *L1BridgeRegistryChangedType // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryChangedTypeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryChangedType)
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
		it.Event = new(L1BridgeRegistryChangedType)
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
func (it *L1BridgeRegistryChangedTypeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryChangedTypeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryChangedType represents a ChangedType event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryChangedType struct {
	RollupConfig common.Address
	Type         uint8
	L2TON        common.Address
	Name         string
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterChangedType is a free log retrieval operation binding the contract event 0x5260076e8db4ab859cb61bfb44f01112e492e853d3e7af965d698c8ef6e76d1b.
//
// Solidity: event ChangedType(address rollupConfig, uint8 type_, address l2TON, string name)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterChangedType(opts *bind.FilterOpts) (*L1BridgeRegistryChangedTypeIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "ChangedType")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryChangedTypeIterator{contract: _L1BridgeRegistry.contract, event: "ChangedType", logs: logs, sub: sub}, nil
}

// WatchChangedType is a free log subscription operation binding the contract event 0x5260076e8db4ab859cb61bfb44f01112e492e853d3e7af965d698c8ef6e76d1b.
//
// Solidity: event ChangedType(address rollupConfig, uint8 type_, address l2TON, string name)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchChangedType(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryChangedType) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "ChangedType")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryChangedType)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "ChangedType", log); err != nil {
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

// ParseChangedType is a log parse operation binding the contract event 0x5260076e8db4ab859cb61bfb44f01112e492e853d3e7af965d698c8ef6e76d1b.
//
// Solidity: event ChangedType(address rollupConfig, uint8 type_, address l2TON, string name)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseChangedType(log types.Log) (*L1BridgeRegistryChangedType, error) {
	event := new(L1BridgeRegistryChangedType)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "ChangedType", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryRegisteredRollupConfigIterator is returned from FilterRegisteredRollupConfig and is used to iterate over the raw logs and unpacked data for RegisteredRollupConfig events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRegisteredRollupConfigIterator struct {
	Event *L1BridgeRegistryRegisteredRollupConfig // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryRegisteredRollupConfigIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryRegisteredRollupConfig)
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
		it.Event = new(L1BridgeRegistryRegisteredRollupConfig)
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
func (it *L1BridgeRegistryRegisteredRollupConfigIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryRegisteredRollupConfigIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryRegisteredRollupConfig represents a RegisteredRollupConfig event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRegisteredRollupConfig struct {
	RollupConfig common.Address
	Type         uint8
	L2TON        common.Address
	Name         string
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRegisteredRollupConfig is a free log retrieval operation binding the contract event 0x79025bb0223ed3a500c1098338b784302bcf993c87487625036460907a45939e.
//
// Solidity: event RegisteredRollupConfig(address rollupConfig, uint8 type_, address l2TON, string name)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterRegisteredRollupConfig(opts *bind.FilterOpts) (*L1BridgeRegistryRegisteredRollupConfigIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "RegisteredRollupConfig")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryRegisteredRollupConfigIterator{contract: _L1BridgeRegistry.contract, event: "RegisteredRollupConfig", logs: logs, sub: sub}, nil
}

// WatchRegisteredRollupConfig is a free log subscription operation binding the contract event 0x79025bb0223ed3a500c1098338b784302bcf993c87487625036460907a45939e.
//
// Solidity: event RegisteredRollupConfig(address rollupConfig, uint8 type_, address l2TON, string name)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchRegisteredRollupConfig(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryRegisteredRollupConfig) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "RegisteredRollupConfig")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryRegisteredRollupConfig)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "RegisteredRollupConfig", log); err != nil {
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

// ParseRegisteredRollupConfig is a log parse operation binding the contract event 0x79025bb0223ed3a500c1098338b784302bcf993c87487625036460907a45939e.
//
// Solidity: event RegisteredRollupConfig(address rollupConfig, uint8 type_, address l2TON, string name)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseRegisteredRollupConfig(log types.Log) (*L1BridgeRegistryRegisteredRollupConfig, error) {
	event := new(L1BridgeRegistryRegisteredRollupConfig)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "RegisteredRollupConfig", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryRejectedCandidateAddOnIterator is returned from FilterRejectedCandidateAddOn and is used to iterate over the raw logs and unpacked data for RejectedCandidateAddOn events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRejectedCandidateAddOnIterator struct {
	Event *L1BridgeRegistryRejectedCandidateAddOn // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryRejectedCandidateAddOnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryRejectedCandidateAddOn)
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
		it.Event = new(L1BridgeRegistryRejectedCandidateAddOn)
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
func (it *L1BridgeRegistryRejectedCandidateAddOnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryRejectedCandidateAddOnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryRejectedCandidateAddOn represents a RejectedCandidateAddOn event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRejectedCandidateAddOn struct {
	RollupConfig common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRejectedCandidateAddOn is a free log retrieval operation binding the contract event 0x0d294c5651970c5fab6bc6f2c023263af93bc2c381b1dc76215156cc95ea675d.
//
// Solidity: event RejectedCandidateAddOn(address rollupConfig)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterRejectedCandidateAddOn(opts *bind.FilterOpts) (*L1BridgeRegistryRejectedCandidateAddOnIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "RejectedCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryRejectedCandidateAddOnIterator{contract: _L1BridgeRegistry.contract, event: "RejectedCandidateAddOn", logs: logs, sub: sub}, nil
}

// WatchRejectedCandidateAddOn is a free log subscription operation binding the contract event 0x0d294c5651970c5fab6bc6f2c023263af93bc2c381b1dc76215156cc95ea675d.
//
// Solidity: event RejectedCandidateAddOn(address rollupConfig)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchRejectedCandidateAddOn(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryRejectedCandidateAddOn) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "RejectedCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryRejectedCandidateAddOn)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "RejectedCandidateAddOn", log); err != nil {
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

// ParseRejectedCandidateAddOn is a log parse operation binding the contract event 0x0d294c5651970c5fab6bc6f2c023263af93bc2c381b1dc76215156cc95ea675d.
//
// Solidity: event RejectedCandidateAddOn(address rollupConfig)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseRejectedCandidateAddOn(log types.Log) (*L1BridgeRegistryRejectedCandidateAddOn, error) {
	event := new(L1BridgeRegistryRejectedCandidateAddOn)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "RejectedCandidateAddOn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryRestoredCandidateAddOnIterator is returned from FilterRestoredCandidateAddOn and is used to iterate over the raw logs and unpacked data for RestoredCandidateAddOn events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRestoredCandidateAddOnIterator struct {
	Event *L1BridgeRegistryRestoredCandidateAddOn // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryRestoredCandidateAddOnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryRestoredCandidateAddOn)
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
		it.Event = new(L1BridgeRegistryRestoredCandidateAddOn)
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
func (it *L1BridgeRegistryRestoredCandidateAddOnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryRestoredCandidateAddOnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryRestoredCandidateAddOn represents a RestoredCandidateAddOn event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRestoredCandidateAddOn struct {
	RollupConfig common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRestoredCandidateAddOn is a free log retrieval operation binding the contract event 0xbc89b42e2b372c8e9e1d48bb5dc897d26927cbe07da9bff7d167d0dbef68e9b3.
//
// Solidity: event RestoredCandidateAddOn(address rollupConfig)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterRestoredCandidateAddOn(opts *bind.FilterOpts) (*L1BridgeRegistryRestoredCandidateAddOnIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "RestoredCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryRestoredCandidateAddOnIterator{contract: _L1BridgeRegistry.contract, event: "RestoredCandidateAddOn", logs: logs, sub: sub}, nil
}

// WatchRestoredCandidateAddOn is a free log subscription operation binding the contract event 0xbc89b42e2b372c8e9e1d48bb5dc897d26927cbe07da9bff7d167d0dbef68e9b3.
//
// Solidity: event RestoredCandidateAddOn(address rollupConfig)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchRestoredCandidateAddOn(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryRestoredCandidateAddOn) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "RestoredCandidateAddOn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryRestoredCandidateAddOn)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "RestoredCandidateAddOn", log); err != nil {
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

// ParseRestoredCandidateAddOn is a log parse operation binding the contract event 0xbc89b42e2b372c8e9e1d48bb5dc897d26927cbe07da9bff7d167d0dbef68e9b3.
//
// Solidity: event RestoredCandidateAddOn(address rollupConfig)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseRestoredCandidateAddOn(log types.Log) (*L1BridgeRegistryRestoredCandidateAddOn, error) {
	event := new(L1BridgeRegistryRestoredCandidateAddOn)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "RestoredCandidateAddOn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRoleAdminChangedIterator struct {
	Event *L1BridgeRegistryRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryRoleAdminChanged)
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
		it.Event = new(L1BridgeRegistryRoleAdminChanged)
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
func (it *L1BridgeRegistryRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryRoleAdminChanged represents a RoleAdminChanged event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*L1BridgeRegistryRoleAdminChangedIterator, error) {

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

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryRoleAdminChangedIterator{contract: _L1BridgeRegistry.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryRoleAdminChanged)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseRoleAdminChanged(log types.Log) (*L1BridgeRegistryRoleAdminChanged, error) {
	event := new(L1BridgeRegistryRoleAdminChanged)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRoleGrantedIterator struct {
	Event *L1BridgeRegistryRoleGranted // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryRoleGranted)
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
		it.Event = new(L1BridgeRegistryRoleGranted)
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
func (it *L1BridgeRegistryRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryRoleGranted represents a RoleGranted event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*L1BridgeRegistryRoleGrantedIterator, error) {

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

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryRoleGrantedIterator{contract: _L1BridgeRegistry.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryRoleGranted)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseRoleGranted(log types.Log) (*L1BridgeRegistryRoleGranted, error) {
	event := new(L1BridgeRegistryRoleGranted)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistryRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRoleRevokedIterator struct {
	Event *L1BridgeRegistryRoleRevoked // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistryRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistryRoleRevoked)
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
		it.Event = new(L1BridgeRegistryRoleRevoked)
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
func (it *L1BridgeRegistryRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistryRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistryRoleRevoked represents a RoleRevoked event raised by the L1BridgeRegistry contract.
type L1BridgeRegistryRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*L1BridgeRegistryRoleRevokedIterator, error) {

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

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistryRoleRevokedIterator{contract: _L1BridgeRegistry.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistryRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistryRoleRevoked)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseRoleRevoked(log types.Log) (*L1BridgeRegistryRoleRevoked, error) {
	event := new(L1BridgeRegistryRoleRevoked)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistrySetAddressesIterator is returned from FilterSetAddresses and is used to iterate over the raw logs and unpacked data for SetAddresses events raised by the L1BridgeRegistry contract.
type L1BridgeRegistrySetAddressesIterator struct {
	Event *L1BridgeRegistrySetAddresses // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistrySetAddressesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistrySetAddresses)
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
		it.Event = new(L1BridgeRegistrySetAddresses)
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
func (it *L1BridgeRegistrySetAddressesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistrySetAddressesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistrySetAddresses represents a SetAddresses event raised by the L1BridgeRegistry contract.
type L1BridgeRegistrySetAddresses struct {
	Layer2Manager common.Address
	SeigManager   common.Address
	Ton           common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSetAddresses is a free log retrieval operation binding the contract event 0xbbfb274df95bebde0669697bf0d15986b4ad73e11c495ae4e2d08d1bc5c90bad.
//
// Solidity: event SetAddresses(address _layer2Manager, address _seigManager, address _ton)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterSetAddresses(opts *bind.FilterOpts) (*L1BridgeRegistrySetAddressesIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "SetAddresses")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistrySetAddressesIterator{contract: _L1BridgeRegistry.contract, event: "SetAddresses", logs: logs, sub: sub}, nil
}

// WatchSetAddresses is a free log subscription operation binding the contract event 0xbbfb274df95bebde0669697bf0d15986b4ad73e11c495ae4e2d08d1bc5c90bad.
//
// Solidity: event SetAddresses(address _layer2Manager, address _seigManager, address _ton)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchSetAddresses(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistrySetAddresses) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "SetAddresses")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistrySetAddresses)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "SetAddresses", log); err != nil {
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

// ParseSetAddresses is a log parse operation binding the contract event 0xbbfb274df95bebde0669697bf0d15986b4ad73e11c495ae4e2d08d1bc5c90bad.
//
// Solidity: event SetAddresses(address _layer2Manager, address _seigManager, address _ton)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseSetAddresses(log types.Log) (*L1BridgeRegistrySetAddresses, error) {
	event := new(L1BridgeRegistrySetAddresses)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "SetAddresses", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistrySetBlockingL2DepositIterator is returned from FilterSetBlockingL2Deposit and is used to iterate over the raw logs and unpacked data for SetBlockingL2Deposit events raised by the L1BridgeRegistry contract.
type L1BridgeRegistrySetBlockingL2DepositIterator struct {
	Event *L1BridgeRegistrySetBlockingL2Deposit // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistrySetBlockingL2DepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistrySetBlockingL2Deposit)
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
		it.Event = new(L1BridgeRegistrySetBlockingL2Deposit)
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
func (it *L1BridgeRegistrySetBlockingL2DepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistrySetBlockingL2DepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistrySetBlockingL2Deposit represents a SetBlockingL2Deposit event raised by the L1BridgeRegistry contract.
type L1BridgeRegistrySetBlockingL2Deposit struct {
	RollupConfig      common.Address
	RejectedL2Deposit bool
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSetBlockingL2Deposit is a free log retrieval operation binding the contract event 0x86092b164c4e590c975cbda3041ef6cb41f3b7b98257bd51fc1858a57d7d3486.
//
// Solidity: event SetBlockingL2Deposit(address rollupConfig, bool rejectedL2Deposit)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterSetBlockingL2Deposit(opts *bind.FilterOpts) (*L1BridgeRegistrySetBlockingL2DepositIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "SetBlockingL2Deposit")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistrySetBlockingL2DepositIterator{contract: _L1BridgeRegistry.contract, event: "SetBlockingL2Deposit", logs: logs, sub: sub}, nil
}

// WatchSetBlockingL2Deposit is a free log subscription operation binding the contract event 0x86092b164c4e590c975cbda3041ef6cb41f3b7b98257bd51fc1858a57d7d3486.
//
// Solidity: event SetBlockingL2Deposit(address rollupConfig, bool rejectedL2Deposit)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchSetBlockingL2Deposit(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistrySetBlockingL2Deposit) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "SetBlockingL2Deposit")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistrySetBlockingL2Deposit)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "SetBlockingL2Deposit", log); err != nil {
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

// ParseSetBlockingL2Deposit is a log parse operation binding the contract event 0x86092b164c4e590c975cbda3041ef6cb41f3b7b98257bd51fc1858a57d7d3486.
//
// Solidity: event SetBlockingL2Deposit(address rollupConfig, bool rejectedL2Deposit)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseSetBlockingL2Deposit(log types.Log) (*L1BridgeRegistrySetBlockingL2Deposit, error) {
	event := new(L1BridgeRegistrySetBlockingL2Deposit)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "SetBlockingL2Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1BridgeRegistrySetSeigniorageCommitteeIterator is returned from FilterSetSeigniorageCommittee and is used to iterate over the raw logs and unpacked data for SetSeigniorageCommittee events raised by the L1BridgeRegistry contract.
type L1BridgeRegistrySetSeigniorageCommitteeIterator struct {
	Event *L1BridgeRegistrySetSeigniorageCommittee // Event containing the contract specifics and raw log

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
func (it *L1BridgeRegistrySetSeigniorageCommitteeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1BridgeRegistrySetSeigniorageCommittee)
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
		it.Event = new(L1BridgeRegistrySetSeigniorageCommittee)
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
func (it *L1BridgeRegistrySetSeigniorageCommitteeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1BridgeRegistrySetSeigniorageCommitteeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1BridgeRegistrySetSeigniorageCommittee represents a SetSeigniorageCommittee event raised by the L1BridgeRegistry contract.
type L1BridgeRegistrySetSeigniorageCommittee struct {
	SeigniorageCommittee common.Address
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterSetSeigniorageCommittee is a free log retrieval operation binding the contract event 0x1c7847585b331d5fb688c03d1dc8e73e13231744377f3b1e2f5c26e548c621c1.
//
// Solidity: event SetSeigniorageCommittee(address _seigniorageCommittee)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) FilterSetSeigniorageCommittee(opts *bind.FilterOpts) (*L1BridgeRegistrySetSeigniorageCommitteeIterator, error) {

	logs, sub, err := _L1BridgeRegistry.contract.FilterLogs(opts, "SetSeigniorageCommittee")
	if err != nil {
		return nil, err
	}
	return &L1BridgeRegistrySetSeigniorageCommitteeIterator{contract: _L1BridgeRegistry.contract, event: "SetSeigniorageCommittee", logs: logs, sub: sub}, nil
}

// WatchSetSeigniorageCommittee is a free log subscription operation binding the contract event 0x1c7847585b331d5fb688c03d1dc8e73e13231744377f3b1e2f5c26e548c621c1.
//
// Solidity: event SetSeigniorageCommittee(address _seigniorageCommittee)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) WatchSetSeigniorageCommittee(opts *bind.WatchOpts, sink chan<- *L1BridgeRegistrySetSeigniorageCommittee) (event.Subscription, error) {

	logs, sub, err := _L1BridgeRegistry.contract.WatchLogs(opts, "SetSeigniorageCommittee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1BridgeRegistrySetSeigniorageCommittee)
				if err := _L1BridgeRegistry.contract.UnpackLog(event, "SetSeigniorageCommittee", log); err != nil {
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

// ParseSetSeigniorageCommittee is a log parse operation binding the contract event 0x1c7847585b331d5fb688c03d1dc8e73e13231744377f3b1e2f5c26e548c621c1.
//
// Solidity: event SetSeigniorageCommittee(address _seigniorageCommittee)
func (_L1BridgeRegistry *L1BridgeRegistryFilterer) ParseSetSeigniorageCommittee(log types.Log) (*L1BridgeRegistrySetSeigniorageCommittee, error) {
	event := new(L1BridgeRegistrySetSeigniorageCommittee)
	if err := _L1BridgeRegistry.contract.UnpackLog(event, "SetSeigniorageCommittee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
