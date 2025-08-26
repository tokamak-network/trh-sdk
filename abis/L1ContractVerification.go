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

// L1ContractVerificationMetaData contains all meta data concerning the L1ContractVerification contract.
var L1ContractVerificationMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"ADMIN_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"DEFAULT_ADMIN_ROLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"addAdmin\",\"inputs\":[{\"name\":\"_admin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"expectedNativeToken\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRoleAdmin\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"grantRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"hasRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_tokenAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_initialAdmin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isVerificationPossible\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1BridgeRegistryAddress\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1CrossDomainMessenger\",\"inputs\":[],\"outputs\":[{\"name\":\"logicAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1StandardBridge\",\"inputs\":[],\"outputs\":[{\"name\":\"logicAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"optimismPortal\",\"inputs\":[],\"outputs\":[{\"name\":\"logicAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"proxyAdminCodehash\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"removeAdmin\",\"inputs\":[{\"name\":\"_admin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revokeRole\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"safeWalletConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"tokamakDAO\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"foundation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"implementationCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"requiredThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"ownersCount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setBridgeRegistryAddress\",\"inputs\":[{\"name\":\"_bridgeRegistry\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setLogicContractInfo\",\"inputs\":[{\"name\":\"_systemConfigProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setSafeConfig\",\"inputs\":[{\"name\":\"_tokamakDAO\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_foundation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_threshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_implementationCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_proxyCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setVerificationPossible\",\"inputs\":[{\"name\":\"_isVerificationPossible\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"supportsInterface\",\"inputs\":[{\"name\":\"interfaceId\",\"type\":\"bytes4\",\"internalType\":\"bytes4\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"systemConfig\",\"inputs\":[],\"outputs\":[{\"name\":\"logicAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"verifyAndRegisterRollupConfig\",\"inputs\":[{\"name\":\"_systemConfigProxy\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_proxyAdmin\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"},{\"name\":\"_safeWalletAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_name\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"BridgeRegistryUpdated\",\"inputs\":[{\"name\":\"bridgeRegistry\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigurationSet\",\"inputs\":[{\"name\":\"contractName\",\"type\":\"string\",\"indexed\":true,\"internalType\":\"string\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"LogicContractConfigured\",\"inputs\":[{\"name\":\"contractType\",\"type\":\"string\",\"indexed\":true,\"internalType\":\"string\"},{\"name\":\"proxyAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"implementationAddress\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"NativeTokenSet\",\"inputs\":[{\"name\":\"tokenAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ProxyAdminCodehashSet\",\"inputs\":[{\"name\":\"codehash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RegistrationCompleted\",\"inputs\":[{\"name\":\"safeWalletAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleAdminChanged\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"previousAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"newAdminRole\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleGranted\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RoleRevoked\",\"inputs\":[{\"name\":\"role\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"account\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SafeConfigSet\",\"inputs\":[{\"name\":\"tokamakDAO\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"foundation\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"threshold\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"implementationCodehash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"proxyCodehash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"VerificationCompleted\",\"inputs\":[{\"name\":\"safeWalletAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"systemConfigProxy\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"proxyAdmin\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"VerificationPossibleSet\",\"inputs\":[{\"name\":\"isVerificationPossible\",\"type\":\"bool\",\"indexed\":true,\"internalType\":\"bool\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BridgeRegistryNotConfigured\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"CallerNotDeployer\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"CallerNotOperator\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ContractNotRegistered\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"GetProxyImplementationFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidProxyAdminAddress\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidThreshold\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"L1CrossDomainMessengerVerificationFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"L1StandardBridgeVerificationFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NativeTokenNotTON\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NoSafeWalletConfigured\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OptimismPortalVerificationFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminCodehashZero\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminInvalidCodehash\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminNotL1CrossDomainMessengerAdmin\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminNotL1StandardBridgeAdmin\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminNotOptimismPortalAdmin\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminNotSystemConfigAdmin\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"RollupConfigNotAvailable\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletAddressMismatch\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletInvalidFallbackHandler\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletInvalidImplCodehash\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletInvalidProxyCodehash\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletInvalidThreshold\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletMissingRequiredOwners\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletUnauthorizedModules\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeWalletWrongOwnerCount\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SystemConfigVerificationFailed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ZeroAddress\",\"inputs\":[{\"name\":\"parameter\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
}

// L1ContractVerificationABI is the input ABI used to generate the binding from.
// Deprecated: Use L1ContractVerificationMetaData.ABI instead.
var L1ContractVerificationABI = L1ContractVerificationMetaData.ABI

// L1ContractVerification is an auto generated Go binding around an Ethereum contract.
type L1ContractVerification struct {
	L1ContractVerificationCaller     // Read-only binding to the contract
	L1ContractVerificationTransactor // Write-only binding to the contract
	L1ContractVerificationFilterer   // Log filterer for contract events
}

// L1ContractVerificationCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1ContractVerificationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ContractVerificationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1ContractVerificationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ContractVerificationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1ContractVerificationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ContractVerificationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1ContractVerificationSession struct {
	Contract     *L1ContractVerification // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// L1ContractVerificationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1ContractVerificationCallerSession struct {
	Contract *L1ContractVerificationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// L1ContractVerificationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1ContractVerificationTransactorSession struct {
	Contract     *L1ContractVerificationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// L1ContractVerificationRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1ContractVerificationRaw struct {
	Contract *L1ContractVerification // Generic contract binding to access the raw methods on
}

// L1ContractVerificationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1ContractVerificationCallerRaw struct {
	Contract *L1ContractVerificationCaller // Generic read-only contract binding to access the raw methods on
}

// L1ContractVerificationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1ContractVerificationTransactorRaw struct {
	Contract *L1ContractVerificationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1ContractVerification creates a new instance of L1ContractVerification, bound to a specific deployed contract.
func NewL1ContractVerification(address common.Address, backend bind.ContractBackend) (*L1ContractVerification, error) {
	contract, err := bindL1ContractVerification(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerification{L1ContractVerificationCaller: L1ContractVerificationCaller{contract: contract}, L1ContractVerificationTransactor: L1ContractVerificationTransactor{contract: contract}, L1ContractVerificationFilterer: L1ContractVerificationFilterer{contract: contract}}, nil
}

// NewL1ContractVerificationCaller creates a new read-only instance of L1ContractVerification, bound to a specific deployed contract.
func NewL1ContractVerificationCaller(address common.Address, caller bind.ContractCaller) (*L1ContractVerificationCaller, error) {
	contract, err := bindL1ContractVerification(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationCaller{contract: contract}, nil
}

// NewL1ContractVerificationTransactor creates a new write-only instance of L1ContractVerification, bound to a specific deployed contract.
func NewL1ContractVerificationTransactor(address common.Address, transactor bind.ContractTransactor) (*L1ContractVerificationTransactor, error) {
	contract, err := bindL1ContractVerification(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationTransactor{contract: contract}, nil
}

// NewL1ContractVerificationFilterer creates a new log filterer instance of L1ContractVerification, bound to a specific deployed contract.
func NewL1ContractVerificationFilterer(address common.Address, filterer bind.ContractFilterer) (*L1ContractVerificationFilterer, error) {
	contract, err := bindL1ContractVerification(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationFilterer{contract: contract}, nil
}

// bindL1ContractVerification binds a generic wrapper to an already deployed contract.
func bindL1ContractVerification(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1ContractVerificationMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ContractVerification *L1ContractVerificationRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ContractVerification.Contract.L1ContractVerificationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ContractVerification *L1ContractVerificationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.L1ContractVerificationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ContractVerification *L1ContractVerificationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.L1ContractVerificationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ContractVerification *L1ContractVerificationCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ContractVerification.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ContractVerification *L1ContractVerificationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ContractVerification *L1ContractVerificationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.contract.Transact(opts, method, params...)
}

// ADMINROLE is a free data retrieval call binding the contract method 0x75b238fc.
//
// Solidity: function ADMIN_ROLE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) ADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ADMINROLE is a free data retrieval call binding the contract method 0x75b238fc.
//
// Solidity: function ADMIN_ROLE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) ADMINROLE() ([32]byte, error) {
	return _L1ContractVerification.Contract.ADMINROLE(&_L1ContractVerification.CallOpts)
}

// ADMINROLE is a free data retrieval call binding the contract method 0x75b238fc.
//
// Solidity: function ADMIN_ROLE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) ADMINROLE() ([32]byte, error) {
	return _L1ContractVerification.Contract.ADMINROLE(&_L1ContractVerification.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _L1ContractVerification.Contract.DEFAULTADMINROLE(&_L1ContractVerification.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _L1ContractVerification.Contract.DEFAULTADMINROLE(&_L1ContractVerification.CallOpts)
}

// ExpectedNativeToken is a free data retrieval call binding the contract method 0x4d39c2c8.
//
// Solidity: function expectedNativeToken() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCaller) ExpectedNativeToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "expectedNativeToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ExpectedNativeToken is a free data retrieval call binding the contract method 0x4d39c2c8.
//
// Solidity: function expectedNativeToken() view returns(address)
func (_L1ContractVerification *L1ContractVerificationSession) ExpectedNativeToken() (common.Address, error) {
	return _L1ContractVerification.Contract.ExpectedNativeToken(&_L1ContractVerification.CallOpts)
}

// ExpectedNativeToken is a free data retrieval call binding the contract method 0x4d39c2c8.
//
// Solidity: function expectedNativeToken() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCallerSession) ExpectedNativeToken() (common.Address, error) {
	return _L1ContractVerification.Contract.ExpectedNativeToken(&_L1ContractVerification.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _L1ContractVerification.Contract.GetRoleAdmin(&_L1ContractVerification.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _L1ContractVerification.Contract.GetRoleAdmin(&_L1ContractVerification.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_L1ContractVerification *L1ContractVerificationCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _L1ContractVerification.Contract.HasRole(&_L1ContractVerification.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_L1ContractVerification *L1ContractVerificationCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _L1ContractVerification.Contract.HasRole(&_L1ContractVerification.CallOpts, role, account)
}

// IsVerificationPossible is a free data retrieval call binding the contract method 0x0752ec7b.
//
// Solidity: function isVerificationPossible() view returns(bool)
func (_L1ContractVerification *L1ContractVerificationCaller) IsVerificationPossible(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "isVerificationPossible")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsVerificationPossible is a free data retrieval call binding the contract method 0x0752ec7b.
//
// Solidity: function isVerificationPossible() view returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) IsVerificationPossible() (bool, error) {
	return _L1ContractVerification.Contract.IsVerificationPossible(&_L1ContractVerification.CallOpts)
}

// IsVerificationPossible is a free data retrieval call binding the contract method 0x0752ec7b.
//
// Solidity: function isVerificationPossible() view returns(bool)
func (_L1ContractVerification *L1ContractVerificationCallerSession) IsVerificationPossible() (bool, error) {
	return _L1ContractVerification.Contract.IsVerificationPossible(&_L1ContractVerification.CallOpts)
}

// L1BridgeRegistryAddress is a free data retrieval call binding the contract method 0xa76e54c1.
//
// Solidity: function l1BridgeRegistryAddress() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCaller) L1BridgeRegistryAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "l1BridgeRegistryAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeRegistryAddress is a free data retrieval call binding the contract method 0xa76e54c1.
//
// Solidity: function l1BridgeRegistryAddress() view returns(address)
func (_L1ContractVerification *L1ContractVerificationSession) L1BridgeRegistryAddress() (common.Address, error) {
	return _L1ContractVerification.Contract.L1BridgeRegistryAddress(&_L1ContractVerification.CallOpts)
}

// L1BridgeRegistryAddress is a free data retrieval call binding the contract method 0xa76e54c1.
//
// Solidity: function l1BridgeRegistryAddress() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCallerSession) L1BridgeRegistryAddress() (common.Address, error) {
	return _L1ContractVerification.Contract.L1BridgeRegistryAddress(&_L1ContractVerification.CallOpts)
}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCaller) L1CrossDomainMessenger(opts *bind.CallOpts) (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "l1CrossDomainMessenger")

	outstruct := new(struct {
		LogicAddress  common.Address
		ProxyCodehash [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LogicAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.ProxyCodehash = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationSession) L1CrossDomainMessenger() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.L1CrossDomainMessenger(&_L1ContractVerification.CallOpts)
}

// L1CrossDomainMessenger is a free data retrieval call binding the contract method 0xa7119869.
//
// Solidity: function l1CrossDomainMessenger() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCallerSession) L1CrossDomainMessenger() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.L1CrossDomainMessenger(&_L1ContractVerification.CallOpts)
}

// L1StandardBridge is a free data retrieval call binding the contract method 0x078f29cf.
//
// Solidity: function l1StandardBridge() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCaller) L1StandardBridge(opts *bind.CallOpts) (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "l1StandardBridge")

	outstruct := new(struct {
		LogicAddress  common.Address
		ProxyCodehash [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LogicAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.ProxyCodehash = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// L1StandardBridge is a free data retrieval call binding the contract method 0x078f29cf.
//
// Solidity: function l1StandardBridge() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationSession) L1StandardBridge() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.L1StandardBridge(&_L1ContractVerification.CallOpts)
}

// L1StandardBridge is a free data retrieval call binding the contract method 0x078f29cf.
//
// Solidity: function l1StandardBridge() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCallerSession) L1StandardBridge() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.L1StandardBridge(&_L1ContractVerification.CallOpts)
}

// OptimismPortal is a free data retrieval call binding the contract method 0x0a49cb03.
//
// Solidity: function optimismPortal() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCaller) OptimismPortal(opts *bind.CallOpts) (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "optimismPortal")

	outstruct := new(struct {
		LogicAddress  common.Address
		ProxyCodehash [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LogicAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.ProxyCodehash = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// OptimismPortal is a free data retrieval call binding the contract method 0x0a49cb03.
//
// Solidity: function optimismPortal() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationSession) OptimismPortal() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.OptimismPortal(&_L1ContractVerification.CallOpts)
}

// OptimismPortal is a free data retrieval call binding the contract method 0x0a49cb03.
//
// Solidity: function optimismPortal() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCallerSession) OptimismPortal() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.OptimismPortal(&_L1ContractVerification.CallOpts)
}

// ProxyAdminCodehash is a free data retrieval call binding the contract method 0x6ab72699.
//
// Solidity: function proxyAdminCodehash() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) ProxyAdminCodehash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "proxyAdminCodehash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxyAdminCodehash is a free data retrieval call binding the contract method 0x6ab72699.
//
// Solidity: function proxyAdminCodehash() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) ProxyAdminCodehash() ([32]byte, error) {
	return _L1ContractVerification.Contract.ProxyAdminCodehash(&_L1ContractVerification.CallOpts)
}

// ProxyAdminCodehash is a free data retrieval call binding the contract method 0x6ab72699.
//
// Solidity: function proxyAdminCodehash() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) ProxyAdminCodehash() ([32]byte, error) {
	return _L1ContractVerification.Contract.ProxyAdminCodehash(&_L1ContractVerification.CallOpts)
}

// SafeWalletConfig is a free data retrieval call binding the contract method 0x85b7db5e.
//
// Solidity: function safeWalletConfig() view returns(address tokamakDAO, address foundation, bytes32 implementationCodehash, bytes32 proxyCodehash, uint256 requiredThreshold, uint256 ownersCount)
func (_L1ContractVerification *L1ContractVerificationCaller) SafeWalletConfig(opts *bind.CallOpts) (struct {
	TokamakDAO             common.Address
	Foundation             common.Address
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	RequiredThreshold      *big.Int
	OwnersCount            *big.Int
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "safeWalletConfig")

	outstruct := new(struct {
		TokamakDAO             common.Address
		Foundation             common.Address
		ImplementationCodehash [32]byte
		ProxyCodehash          [32]byte
		RequiredThreshold      *big.Int
		OwnersCount            *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TokamakDAO = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Foundation = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.ImplementationCodehash = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)
	outstruct.ProxyCodehash = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.RequiredThreshold = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.OwnersCount = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// SafeWalletConfig is a free data retrieval call binding the contract method 0x85b7db5e.
//
// Solidity: function safeWalletConfig() view returns(address tokamakDAO, address foundation, bytes32 implementationCodehash, bytes32 proxyCodehash, uint256 requiredThreshold, uint256 ownersCount)
func (_L1ContractVerification *L1ContractVerificationSession) SafeWalletConfig() (struct {
	TokamakDAO             common.Address
	Foundation             common.Address
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	RequiredThreshold      *big.Int
	OwnersCount            *big.Int
}, error) {
	return _L1ContractVerification.Contract.SafeWalletConfig(&_L1ContractVerification.CallOpts)
}

// SafeWalletConfig is a free data retrieval call binding the contract method 0x85b7db5e.
//
// Solidity: function safeWalletConfig() view returns(address tokamakDAO, address foundation, bytes32 implementationCodehash, bytes32 proxyCodehash, uint256 requiredThreshold, uint256 ownersCount)
func (_L1ContractVerification *L1ContractVerificationCallerSession) SafeWalletConfig() (struct {
	TokamakDAO             common.Address
	Foundation             common.Address
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	RequiredThreshold      *big.Int
	OwnersCount            *big.Int
}, error) {
	return _L1ContractVerification.Contract.SafeWalletConfig(&_L1ContractVerification.CallOpts)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_L1ContractVerification *L1ContractVerificationCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _L1ContractVerification.Contract.SupportsInterface(&_L1ContractVerification.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_L1ContractVerification *L1ContractVerificationCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _L1ContractVerification.Contract.SupportsInterface(&_L1ContractVerification.CallOpts, interfaceId)
}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCaller) SystemConfig(opts *bind.CallOpts) (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "systemConfig")

	outstruct := new(struct {
		LogicAddress  common.Address
		ProxyCodehash [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.LogicAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.ProxyCodehash = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationSession) SystemConfig() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.SystemConfig(&_L1ContractVerification.CallOpts)
}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address logicAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationCallerSession) SystemConfig() (struct {
	LogicAddress  common.Address
	ProxyCodehash [32]byte
}, error) {
	return _L1ContractVerification.Contract.SystemConfig(&_L1ContractVerification.CallOpts)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address _admin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) AddAdmin(opts *bind.TransactOpts, _admin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "addAdmin", _admin)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address _admin) returns()
func (_L1ContractVerification *L1ContractVerificationSession) AddAdmin(_admin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.AddAdmin(&_L1ContractVerification.TransactOpts, _admin)
}

// AddAdmin is a paid mutator transaction binding the contract method 0x70480275.
//
// Solidity: function addAdmin(address _admin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) AddAdmin(_admin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.AddAdmin(&_L1ContractVerification.TransactOpts, _admin)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.GrantRole(&_L1ContractVerification.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.GrantRole(&_L1ContractVerification.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _tokenAddress, address _initialAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) Initialize(opts *bind.TransactOpts, _tokenAddress common.Address, _initialAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "initialize", _tokenAddress, _initialAdmin)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _tokenAddress, address _initialAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationSession) Initialize(_tokenAddress common.Address, _initialAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.Initialize(&_L1ContractVerification.TransactOpts, _tokenAddress, _initialAdmin)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _tokenAddress, address _initialAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) Initialize(_tokenAddress common.Address, _initialAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.Initialize(&_L1ContractVerification.TransactOpts, _tokenAddress, _initialAdmin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address _admin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) RemoveAdmin(opts *bind.TransactOpts, _admin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "removeAdmin", _admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address _admin) returns()
func (_L1ContractVerification *L1ContractVerificationSession) RemoveAdmin(_admin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RemoveAdmin(&_L1ContractVerification.TransactOpts, _admin)
}

// RemoveAdmin is a paid mutator transaction binding the contract method 0x1785f53c.
//
// Solidity: function removeAdmin(address _admin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) RemoveAdmin(_admin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RemoveAdmin(&_L1ContractVerification.TransactOpts, _admin)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RenounceRole(&_L1ContractVerification.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RenounceRole(&_L1ContractVerification.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RevokeRole(&_L1ContractVerification.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RevokeRole(&_L1ContractVerification.TransactOpts, role, account)
}

// SetBridgeRegistryAddress is a paid mutator transaction binding the contract method 0x9ef72b7f.
//
// Solidity: function setBridgeRegistryAddress(address _bridgeRegistry) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetBridgeRegistryAddress(opts *bind.TransactOpts, _bridgeRegistry common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setBridgeRegistryAddress", _bridgeRegistry)
}

// SetBridgeRegistryAddress is a paid mutator transaction binding the contract method 0x9ef72b7f.
//
// Solidity: function setBridgeRegistryAddress(address _bridgeRegistry) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetBridgeRegistryAddress(_bridgeRegistry common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetBridgeRegistryAddress(&_L1ContractVerification.TransactOpts, _bridgeRegistry)
}

// SetBridgeRegistryAddress is a paid mutator transaction binding the contract method 0x9ef72b7f.
//
// Solidity: function setBridgeRegistryAddress(address _bridgeRegistry) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetBridgeRegistryAddress(_bridgeRegistry common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetBridgeRegistryAddress(&_L1ContractVerification.TransactOpts, _bridgeRegistry)
}

// SetLogicContractInfo is a paid mutator transaction binding the contract method 0x5f57fa91.
//
// Solidity: function setLogicContractInfo(address _systemConfigProxy, address _proxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetLogicContractInfo(opts *bind.TransactOpts, _systemConfigProxy common.Address, _proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setLogicContractInfo", _systemConfigProxy, _proxyAdmin)
}

// SetLogicContractInfo is a paid mutator transaction binding the contract method 0x5f57fa91.
//
// Solidity: function setLogicContractInfo(address _systemConfigProxy, address _proxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetLogicContractInfo(_systemConfigProxy common.Address, _proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetLogicContractInfo(&_L1ContractVerification.TransactOpts, _systemConfigProxy, _proxyAdmin)
}

// SetLogicContractInfo is a paid mutator transaction binding the contract method 0x5f57fa91.
//
// Solidity: function setLogicContractInfo(address _systemConfigProxy, address _proxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetLogicContractInfo(_systemConfigProxy common.Address, _proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetLogicContractInfo(&_L1ContractVerification.TransactOpts, _systemConfigProxy, _proxyAdmin)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0xabc66ce9.
//
// Solidity: function setSafeConfig(address _tokamakDAO, address _foundation, uint256 _threshold, bytes32 _implementationCodehash, bytes32 _proxyCodehash) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetSafeConfig(opts *bind.TransactOpts, _tokamakDAO common.Address, _foundation common.Address, _threshold *big.Int, _implementationCodehash [32]byte, _proxyCodehash [32]byte) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setSafeConfig", _tokamakDAO, _foundation, _threshold, _implementationCodehash, _proxyCodehash)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0xabc66ce9.
//
// Solidity: function setSafeConfig(address _tokamakDAO, address _foundation, uint256 _threshold, bytes32 _implementationCodehash, bytes32 _proxyCodehash) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetSafeConfig(_tokamakDAO common.Address, _foundation common.Address, _threshold *big.Int, _implementationCodehash [32]byte, _proxyCodehash [32]byte) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeConfig(&_L1ContractVerification.TransactOpts, _tokamakDAO, _foundation, _threshold, _implementationCodehash, _proxyCodehash)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0xabc66ce9.
//
// Solidity: function setSafeConfig(address _tokamakDAO, address _foundation, uint256 _threshold, bytes32 _implementationCodehash, bytes32 _proxyCodehash) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetSafeConfig(_tokamakDAO common.Address, _foundation common.Address, _threshold *big.Int, _implementationCodehash [32]byte, _proxyCodehash [32]byte) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeConfig(&_L1ContractVerification.TransactOpts, _tokamakDAO, _foundation, _threshold, _implementationCodehash, _proxyCodehash)
}

// SetVerificationPossible is a paid mutator transaction binding the contract method 0xa60065ab.
//
// Solidity: function setVerificationPossible(bool _isVerificationPossible) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetVerificationPossible(opts *bind.TransactOpts, _isVerificationPossible bool) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setVerificationPossible", _isVerificationPossible)
}

// SetVerificationPossible is a paid mutator transaction binding the contract method 0xa60065ab.
//
// Solidity: function setVerificationPossible(bool _isVerificationPossible) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetVerificationPossible(_isVerificationPossible bool) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetVerificationPossible(&_L1ContractVerification.TransactOpts, _isVerificationPossible)
}

// SetVerificationPossible is a paid mutator transaction binding the contract method 0xa60065ab.
//
// Solidity: function setVerificationPossible(bool _isVerificationPossible) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetVerificationPossible(_isVerificationPossible bool) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetVerificationPossible(&_L1ContractVerification.TransactOpts, _isVerificationPossible)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x26e7700f.
//
// Solidity: function verifyAndRegisterRollupConfig(address _systemConfigProxy, address _proxyAdmin, address _safeWalletAddress, string _name) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) VerifyAndRegisterRollupConfig(opts *bind.TransactOpts, _systemConfigProxy common.Address, _proxyAdmin common.Address, _safeWalletAddress common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "verifyAndRegisterRollupConfig", _systemConfigProxy, _proxyAdmin, _safeWalletAddress, _name)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x26e7700f.
//
// Solidity: function verifyAndRegisterRollupConfig(address _systemConfigProxy, address _proxyAdmin, address _safeWalletAddress, string _name) returns()
func (_L1ContractVerification *L1ContractVerificationSession) VerifyAndRegisterRollupConfig(_systemConfigProxy common.Address, _proxyAdmin common.Address, _safeWalletAddress common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyAndRegisterRollupConfig(&_L1ContractVerification.TransactOpts, _systemConfigProxy, _proxyAdmin, _safeWalletAddress, _name)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x26e7700f.
//
// Solidity: function verifyAndRegisterRollupConfig(address _systemConfigProxy, address _proxyAdmin, address _safeWalletAddress, string _name) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) VerifyAndRegisterRollupConfig(_systemConfigProxy common.Address, _proxyAdmin common.Address, _safeWalletAddress common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyAndRegisterRollupConfig(&_L1ContractVerification.TransactOpts, _systemConfigProxy, _proxyAdmin, _safeWalletAddress, _name)
}

// L1ContractVerificationBridgeRegistryUpdatedIterator is returned from FilterBridgeRegistryUpdated and is used to iterate over the raw logs and unpacked data for BridgeRegistryUpdated events raised by the L1ContractVerification contract.
type L1ContractVerificationBridgeRegistryUpdatedIterator struct {
	Event *L1ContractVerificationBridgeRegistryUpdated // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationBridgeRegistryUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationBridgeRegistryUpdated)
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
		it.Event = new(L1ContractVerificationBridgeRegistryUpdated)
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
func (it *L1ContractVerificationBridgeRegistryUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationBridgeRegistryUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationBridgeRegistryUpdated represents a BridgeRegistryUpdated event raised by the L1ContractVerification contract.
type L1ContractVerificationBridgeRegistryUpdated struct {
	BridgeRegistry common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterBridgeRegistryUpdated is a free log retrieval operation binding the contract event 0xe55b1ea7e1026d0229a61bf11b8e0319ca1fabcb0e06f96378a7f31a90318982.
//
// Solidity: event BridgeRegistryUpdated(address indexed bridgeRegistry)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterBridgeRegistryUpdated(opts *bind.FilterOpts, bridgeRegistry []common.Address) (*L1ContractVerificationBridgeRegistryUpdatedIterator, error) {

	var bridgeRegistryRule []interface{}
	for _, bridgeRegistryItem := range bridgeRegistry {
		bridgeRegistryRule = append(bridgeRegistryRule, bridgeRegistryItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "BridgeRegistryUpdated", bridgeRegistryRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationBridgeRegistryUpdatedIterator{contract: _L1ContractVerification.contract, event: "BridgeRegistryUpdated", logs: logs, sub: sub}, nil
}

// WatchBridgeRegistryUpdated is a free log subscription operation binding the contract event 0xe55b1ea7e1026d0229a61bf11b8e0319ca1fabcb0e06f96378a7f31a90318982.
//
// Solidity: event BridgeRegistryUpdated(address indexed bridgeRegistry)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchBridgeRegistryUpdated(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationBridgeRegistryUpdated, bridgeRegistry []common.Address) (event.Subscription, error) {

	var bridgeRegistryRule []interface{}
	for _, bridgeRegistryItem := range bridgeRegistry {
		bridgeRegistryRule = append(bridgeRegistryRule, bridgeRegistryItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "BridgeRegistryUpdated", bridgeRegistryRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationBridgeRegistryUpdated)
				if err := _L1ContractVerification.contract.UnpackLog(event, "BridgeRegistryUpdated", log); err != nil {
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

// ParseBridgeRegistryUpdated is a log parse operation binding the contract event 0xe55b1ea7e1026d0229a61bf11b8e0319ca1fabcb0e06f96378a7f31a90318982.
//
// Solidity: event BridgeRegistryUpdated(address indexed bridgeRegistry)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseBridgeRegistryUpdated(log types.Log) (*L1ContractVerificationBridgeRegistryUpdated, error) {
	event := new(L1ContractVerificationBridgeRegistryUpdated)
	if err := _L1ContractVerification.contract.UnpackLog(event, "BridgeRegistryUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationConfigurationSetIterator is returned from FilterConfigurationSet and is used to iterate over the raw logs and unpacked data for ConfigurationSet events raised by the L1ContractVerification contract.
type L1ContractVerificationConfigurationSetIterator struct {
	Event *L1ContractVerificationConfigurationSet // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationConfigurationSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationConfigurationSet)
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
		it.Event = new(L1ContractVerificationConfigurationSet)
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
func (it *L1ContractVerificationConfigurationSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationConfigurationSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationConfigurationSet represents a ConfigurationSet event raised by the L1ContractVerification contract.
type L1ContractVerificationConfigurationSet struct {
	ContractName common.Hash
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterConfigurationSet is a free log retrieval operation binding the contract event 0x9d9a784c1887058f956491a67b266b813e04339d0453c64b3d9f9de58d0c1bcc.
//
// Solidity: event ConfigurationSet(string indexed contractName)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterConfigurationSet(opts *bind.FilterOpts, contractName []string) (*L1ContractVerificationConfigurationSetIterator, error) {

	var contractNameRule []interface{}
	for _, contractNameItem := range contractName {
		contractNameRule = append(contractNameRule, contractNameItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "ConfigurationSet", contractNameRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationConfigurationSetIterator{contract: _L1ContractVerification.contract, event: "ConfigurationSet", logs: logs, sub: sub}, nil
}

// WatchConfigurationSet is a free log subscription operation binding the contract event 0x9d9a784c1887058f956491a67b266b813e04339d0453c64b3d9f9de58d0c1bcc.
//
// Solidity: event ConfigurationSet(string indexed contractName)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchConfigurationSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationConfigurationSet, contractName []string) (event.Subscription, error) {

	var contractNameRule []interface{}
	for _, contractNameItem := range contractName {
		contractNameRule = append(contractNameRule, contractNameItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "ConfigurationSet", contractNameRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationConfigurationSet)
				if err := _L1ContractVerification.contract.UnpackLog(event, "ConfigurationSet", log); err != nil {
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

// ParseConfigurationSet is a log parse operation binding the contract event 0x9d9a784c1887058f956491a67b266b813e04339d0453c64b3d9f9de58d0c1bcc.
//
// Solidity: event ConfigurationSet(string indexed contractName)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseConfigurationSet(log types.Log) (*L1ContractVerificationConfigurationSet, error) {
	event := new(L1ContractVerificationConfigurationSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "ConfigurationSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1ContractVerification contract.
type L1ContractVerificationInitializedIterator struct {
	Event *L1ContractVerificationInitialized // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationInitialized)
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
		it.Event = new(L1ContractVerificationInitialized)
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
func (it *L1ContractVerificationInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationInitialized represents a Initialized event raised by the L1ContractVerification contract.
type L1ContractVerificationInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1ContractVerificationInitializedIterator, error) {

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationInitializedIterator{contract: _L1ContractVerification.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationInitialized) (event.Subscription, error) {

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationInitialized)
				if err := _L1ContractVerification.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseInitialized(log types.Log) (*L1ContractVerificationInitialized, error) {
	event := new(L1ContractVerificationInitialized)
	if err := _L1ContractVerification.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationLogicContractConfiguredIterator is returned from FilterLogicContractConfigured and is used to iterate over the raw logs and unpacked data for LogicContractConfigured events raised by the L1ContractVerification contract.
type L1ContractVerificationLogicContractConfiguredIterator struct {
	Event *L1ContractVerificationLogicContractConfigured // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationLogicContractConfiguredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationLogicContractConfigured)
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
		it.Event = new(L1ContractVerificationLogicContractConfigured)
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
func (it *L1ContractVerificationLogicContractConfiguredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationLogicContractConfiguredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationLogicContractConfigured represents a LogicContractConfigured event raised by the L1ContractVerification contract.
type L1ContractVerificationLogicContractConfigured struct {
	ContractType          common.Hash
	ProxyAddress          common.Address
	ImplementationAddress common.Address
	ProxyCodehash         [32]byte
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterLogicContractConfigured is a free log retrieval operation binding the contract event 0x6b160f0924ad7879d92f4be149b2e79234f7b6782b09d4ca7f520139ff97af51.
//
// Solidity: event LogicContractConfigured(string indexed contractType, address indexed proxyAddress, address implementationAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterLogicContractConfigured(opts *bind.FilterOpts, contractType []string, proxyAddress []common.Address) (*L1ContractVerificationLogicContractConfiguredIterator, error) {

	var contractTypeRule []interface{}
	for _, contractTypeItem := range contractType {
		contractTypeRule = append(contractTypeRule, contractTypeItem)
	}
	var proxyAddressRule []interface{}
	for _, proxyAddressItem := range proxyAddress {
		proxyAddressRule = append(proxyAddressRule, proxyAddressItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "LogicContractConfigured", contractTypeRule, proxyAddressRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationLogicContractConfiguredIterator{contract: _L1ContractVerification.contract, event: "LogicContractConfigured", logs: logs, sub: sub}, nil
}

// WatchLogicContractConfigured is a free log subscription operation binding the contract event 0x6b160f0924ad7879d92f4be149b2e79234f7b6782b09d4ca7f520139ff97af51.
//
// Solidity: event LogicContractConfigured(string indexed contractType, address indexed proxyAddress, address implementationAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchLogicContractConfigured(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationLogicContractConfigured, contractType []string, proxyAddress []common.Address) (event.Subscription, error) {

	var contractTypeRule []interface{}
	for _, contractTypeItem := range contractType {
		contractTypeRule = append(contractTypeRule, contractTypeItem)
	}
	var proxyAddressRule []interface{}
	for _, proxyAddressItem := range proxyAddress {
		proxyAddressRule = append(proxyAddressRule, proxyAddressItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "LogicContractConfigured", contractTypeRule, proxyAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationLogicContractConfigured)
				if err := _L1ContractVerification.contract.UnpackLog(event, "LogicContractConfigured", log); err != nil {
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

// ParseLogicContractConfigured is a log parse operation binding the contract event 0x6b160f0924ad7879d92f4be149b2e79234f7b6782b09d4ca7f520139ff97af51.
//
// Solidity: event LogicContractConfigured(string indexed contractType, address indexed proxyAddress, address implementationAddress, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseLogicContractConfigured(log types.Log) (*L1ContractVerificationLogicContractConfigured, error) {
	event := new(L1ContractVerificationLogicContractConfigured)
	if err := _L1ContractVerification.contract.UnpackLog(event, "LogicContractConfigured", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationNativeTokenSetIterator is returned from FilterNativeTokenSet and is used to iterate over the raw logs and unpacked data for NativeTokenSet events raised by the L1ContractVerification contract.
type L1ContractVerificationNativeTokenSetIterator struct {
	Event *L1ContractVerificationNativeTokenSet // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationNativeTokenSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationNativeTokenSet)
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
		it.Event = new(L1ContractVerificationNativeTokenSet)
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
func (it *L1ContractVerificationNativeTokenSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationNativeTokenSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationNativeTokenSet represents a NativeTokenSet event raised by the L1ContractVerification contract.
type L1ContractVerificationNativeTokenSet struct {
	TokenAddress common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterNativeTokenSet is a free log retrieval operation binding the contract event 0x69126b322a28773e88593d6fb96fb0106c50388db89b52dbbd94f1ae2decc66d.
//
// Solidity: event NativeTokenSet(address indexed tokenAddress)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterNativeTokenSet(opts *bind.FilterOpts, tokenAddress []common.Address) (*L1ContractVerificationNativeTokenSetIterator, error) {

	var tokenAddressRule []interface{}
	for _, tokenAddressItem := range tokenAddress {
		tokenAddressRule = append(tokenAddressRule, tokenAddressItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "NativeTokenSet", tokenAddressRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationNativeTokenSetIterator{contract: _L1ContractVerification.contract, event: "NativeTokenSet", logs: logs, sub: sub}, nil
}

// WatchNativeTokenSet is a free log subscription operation binding the contract event 0x69126b322a28773e88593d6fb96fb0106c50388db89b52dbbd94f1ae2decc66d.
//
// Solidity: event NativeTokenSet(address indexed tokenAddress)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchNativeTokenSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationNativeTokenSet, tokenAddress []common.Address) (event.Subscription, error) {

	var tokenAddressRule []interface{}
	for _, tokenAddressItem := range tokenAddress {
		tokenAddressRule = append(tokenAddressRule, tokenAddressItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "NativeTokenSet", tokenAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationNativeTokenSet)
				if err := _L1ContractVerification.contract.UnpackLog(event, "NativeTokenSet", log); err != nil {
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

// ParseNativeTokenSet is a log parse operation binding the contract event 0x69126b322a28773e88593d6fb96fb0106c50388db89b52dbbd94f1ae2decc66d.
//
// Solidity: event NativeTokenSet(address indexed tokenAddress)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseNativeTokenSet(log types.Log) (*L1ContractVerificationNativeTokenSet, error) {
	event := new(L1ContractVerificationNativeTokenSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "NativeTokenSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationProxyAdminCodehashSetIterator is returned from FilterProxyAdminCodehashSet and is used to iterate over the raw logs and unpacked data for ProxyAdminCodehashSet events raised by the L1ContractVerification contract.
type L1ContractVerificationProxyAdminCodehashSetIterator struct {
	Event *L1ContractVerificationProxyAdminCodehashSet // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationProxyAdminCodehashSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationProxyAdminCodehashSet)
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
		it.Event = new(L1ContractVerificationProxyAdminCodehashSet)
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
func (it *L1ContractVerificationProxyAdminCodehashSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationProxyAdminCodehashSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationProxyAdminCodehashSet represents a ProxyAdminCodehashSet event raised by the L1ContractVerification contract.
type L1ContractVerificationProxyAdminCodehashSet struct {
	Codehash [32]byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterProxyAdminCodehashSet is a free log retrieval operation binding the contract event 0x834dd2873e30c49e8619df33022f0db6fb97e8ba6b523ca3bef537f8f48792bb.
//
// Solidity: event ProxyAdminCodehashSet(bytes32 indexed codehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterProxyAdminCodehashSet(opts *bind.FilterOpts, codehash [][32]byte) (*L1ContractVerificationProxyAdminCodehashSetIterator, error) {

	var codehashRule []interface{}
	for _, codehashItem := range codehash {
		codehashRule = append(codehashRule, codehashItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "ProxyAdminCodehashSet", codehashRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationProxyAdminCodehashSetIterator{contract: _L1ContractVerification.contract, event: "ProxyAdminCodehashSet", logs: logs, sub: sub}, nil
}

// WatchProxyAdminCodehashSet is a free log subscription operation binding the contract event 0x834dd2873e30c49e8619df33022f0db6fb97e8ba6b523ca3bef537f8f48792bb.
//
// Solidity: event ProxyAdminCodehashSet(bytes32 indexed codehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchProxyAdminCodehashSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationProxyAdminCodehashSet, codehash [][32]byte) (event.Subscription, error) {

	var codehashRule []interface{}
	for _, codehashItem := range codehash {
		codehashRule = append(codehashRule, codehashItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "ProxyAdminCodehashSet", codehashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationProxyAdminCodehashSet)
				if err := _L1ContractVerification.contract.UnpackLog(event, "ProxyAdminCodehashSet", log); err != nil {
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

// ParseProxyAdminCodehashSet is a log parse operation binding the contract event 0x834dd2873e30c49e8619df33022f0db6fb97e8ba6b523ca3bef537f8f48792bb.
//
// Solidity: event ProxyAdminCodehashSet(bytes32 indexed codehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseProxyAdminCodehashSet(log types.Log) (*L1ContractVerificationProxyAdminCodehashSet, error) {
	event := new(L1ContractVerificationProxyAdminCodehashSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "ProxyAdminCodehashSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationRegistrationCompletedIterator is returned from FilterRegistrationCompleted and is used to iterate over the raw logs and unpacked data for RegistrationCompleted events raised by the L1ContractVerification contract.
type L1ContractVerificationRegistrationCompletedIterator struct {
	Event *L1ContractVerificationRegistrationCompleted // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationRegistrationCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationRegistrationCompleted)
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
		it.Event = new(L1ContractVerificationRegistrationCompleted)
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
func (it *L1ContractVerificationRegistrationCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationRegistrationCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationRegistrationCompleted represents a RegistrationCompleted event raised by the L1ContractVerification contract.
type L1ContractVerificationRegistrationCompleted struct {
	SafeWalletAddress common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRegistrationCompleted is a free log retrieval operation binding the contract event 0xe72eca5afa8e0547222b0b56eb58ecf5c653633dcb029882da82bc3a03266645.
//
// Solidity: event RegistrationCompleted(address indexed safeWalletAddress)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterRegistrationCompleted(opts *bind.FilterOpts, safeWalletAddress []common.Address) (*L1ContractVerificationRegistrationCompletedIterator, error) {

	var safeWalletAddressRule []interface{}
	for _, safeWalletAddressItem := range safeWalletAddress {
		safeWalletAddressRule = append(safeWalletAddressRule, safeWalletAddressItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "RegistrationCompleted", safeWalletAddressRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationRegistrationCompletedIterator{contract: _L1ContractVerification.contract, event: "RegistrationCompleted", logs: logs, sub: sub}, nil
}

// WatchRegistrationCompleted is a free log subscription operation binding the contract event 0xe72eca5afa8e0547222b0b56eb58ecf5c653633dcb029882da82bc3a03266645.
//
// Solidity: event RegistrationCompleted(address indexed safeWalletAddress)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchRegistrationCompleted(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationRegistrationCompleted, safeWalletAddress []common.Address) (event.Subscription, error) {

	var safeWalletAddressRule []interface{}
	for _, safeWalletAddressItem := range safeWalletAddress {
		safeWalletAddressRule = append(safeWalletAddressRule, safeWalletAddressItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "RegistrationCompleted", safeWalletAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationRegistrationCompleted)
				if err := _L1ContractVerification.contract.UnpackLog(event, "RegistrationCompleted", log); err != nil {
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

// ParseRegistrationCompleted is a log parse operation binding the contract event 0xe72eca5afa8e0547222b0b56eb58ecf5c653633dcb029882da82bc3a03266645.
//
// Solidity: event RegistrationCompleted(address indexed safeWalletAddress)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseRegistrationCompleted(log types.Log) (*L1ContractVerificationRegistrationCompleted, error) {
	event := new(L1ContractVerificationRegistrationCompleted)
	if err := _L1ContractVerification.contract.UnpackLog(event, "RegistrationCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the L1ContractVerification contract.
type L1ContractVerificationRoleAdminChangedIterator struct {
	Event *L1ContractVerificationRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationRoleAdminChanged)
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
		it.Event = new(L1ContractVerificationRoleAdminChanged)
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
func (it *L1ContractVerificationRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationRoleAdminChanged represents a RoleAdminChanged event raised by the L1ContractVerification contract.
type L1ContractVerificationRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*L1ContractVerificationRoleAdminChangedIterator, error) {

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

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationRoleAdminChangedIterator{contract: _L1ContractVerification.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationRoleAdminChanged)
				if err := _L1ContractVerification.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseRoleAdminChanged(log types.Log) (*L1ContractVerificationRoleAdminChanged, error) {
	event := new(L1ContractVerificationRoleAdminChanged)
	if err := _L1ContractVerification.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the L1ContractVerification contract.
type L1ContractVerificationRoleGrantedIterator struct {
	Event *L1ContractVerificationRoleGranted // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationRoleGranted)
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
		it.Event = new(L1ContractVerificationRoleGranted)
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
func (it *L1ContractVerificationRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationRoleGranted represents a RoleGranted event raised by the L1ContractVerification contract.
type L1ContractVerificationRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*L1ContractVerificationRoleGrantedIterator, error) {

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

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationRoleGrantedIterator{contract: _L1ContractVerification.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationRoleGranted)
				if err := _L1ContractVerification.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseRoleGranted(log types.Log) (*L1ContractVerificationRoleGranted, error) {
	event := new(L1ContractVerificationRoleGranted)
	if err := _L1ContractVerification.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the L1ContractVerification contract.
type L1ContractVerificationRoleRevokedIterator struct {
	Event *L1ContractVerificationRoleRevoked // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationRoleRevoked)
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
		it.Event = new(L1ContractVerificationRoleRevoked)
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
func (it *L1ContractVerificationRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationRoleRevoked represents a RoleRevoked event raised by the L1ContractVerification contract.
type L1ContractVerificationRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*L1ContractVerificationRoleRevokedIterator, error) {

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

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationRoleRevokedIterator{contract: _L1ContractVerification.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationRoleRevoked)
				if err := _L1ContractVerification.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseRoleRevoked(log types.Log) (*L1ContractVerificationRoleRevoked, error) {
	event := new(L1ContractVerificationRoleRevoked)
	if err := _L1ContractVerification.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationSafeConfigSetIterator is returned from FilterSafeConfigSet and is used to iterate over the raw logs and unpacked data for SafeConfigSet events raised by the L1ContractVerification contract.
type L1ContractVerificationSafeConfigSetIterator struct {
	Event *L1ContractVerificationSafeConfigSet // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationSafeConfigSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationSafeConfigSet)
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
		it.Event = new(L1ContractVerificationSafeConfigSet)
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
func (it *L1ContractVerificationSafeConfigSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationSafeConfigSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationSafeConfigSet represents a SafeConfigSet event raised by the L1ContractVerification contract.
type L1ContractVerificationSafeConfigSet struct {
	TokamakDAO             common.Address
	Foundation             common.Address
	Threshold              *big.Int
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterSafeConfigSet is a free log retrieval operation binding the contract event 0xe160f8030381b2b11e340d4a4400274d660cc4827fd3bb45abaa7d8583ba15bd.
//
// Solidity: event SafeConfigSet(address indexed tokamakDAO, address indexed foundation, uint256 indexed threshold, bytes32 implementationCodehash, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterSafeConfigSet(opts *bind.FilterOpts, tokamakDAO []common.Address, foundation []common.Address, threshold []*big.Int) (*L1ContractVerificationSafeConfigSetIterator, error) {

	var tokamakDAORule []interface{}
	for _, tokamakDAOItem := range tokamakDAO {
		tokamakDAORule = append(tokamakDAORule, tokamakDAOItem)
	}
	var foundationRule []interface{}
	for _, foundationItem := range foundation {
		foundationRule = append(foundationRule, foundationItem)
	}
	var thresholdRule []interface{}
	for _, thresholdItem := range threshold {
		thresholdRule = append(thresholdRule, thresholdItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "SafeConfigSet", tokamakDAORule, foundationRule, thresholdRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationSafeConfigSetIterator{contract: _L1ContractVerification.contract, event: "SafeConfigSet", logs: logs, sub: sub}, nil
}

// WatchSafeConfigSet is a free log subscription operation binding the contract event 0xe160f8030381b2b11e340d4a4400274d660cc4827fd3bb45abaa7d8583ba15bd.
//
// Solidity: event SafeConfigSet(address indexed tokamakDAO, address indexed foundation, uint256 indexed threshold, bytes32 implementationCodehash, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchSafeConfigSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationSafeConfigSet, tokamakDAO []common.Address, foundation []common.Address, threshold []*big.Int) (event.Subscription, error) {

	var tokamakDAORule []interface{}
	for _, tokamakDAOItem := range tokamakDAO {
		tokamakDAORule = append(tokamakDAORule, tokamakDAOItem)
	}
	var foundationRule []interface{}
	for _, foundationItem := range foundation {
		foundationRule = append(foundationRule, foundationItem)
	}
	var thresholdRule []interface{}
	for _, thresholdItem := range threshold {
		thresholdRule = append(thresholdRule, thresholdItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "SafeConfigSet", tokamakDAORule, foundationRule, thresholdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationSafeConfigSet)
				if err := _L1ContractVerification.contract.UnpackLog(event, "SafeConfigSet", log); err != nil {
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

// ParseSafeConfigSet is a log parse operation binding the contract event 0xe160f8030381b2b11e340d4a4400274d660cc4827fd3bb45abaa7d8583ba15bd.
//
// Solidity: event SafeConfigSet(address indexed tokamakDAO, address indexed foundation, uint256 indexed threshold, bytes32 implementationCodehash, bytes32 proxyCodehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseSafeConfigSet(log types.Log) (*L1ContractVerificationSafeConfigSet, error) {
	event := new(L1ContractVerificationSafeConfigSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "SafeConfigSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationVerificationCompletedIterator is returned from FilterVerificationCompleted and is used to iterate over the raw logs and unpacked data for VerificationCompleted events raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationCompletedIterator struct {
	Event *L1ContractVerificationVerificationCompleted // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationVerificationCompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationVerificationCompleted)
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
		it.Event = new(L1ContractVerificationVerificationCompleted)
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
func (it *L1ContractVerificationVerificationCompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationVerificationCompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationVerificationCompleted represents a VerificationCompleted event raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationCompleted struct {
	SafeWalletAddress common.Address
	SystemConfigProxy common.Address
	ProxyAdmin        common.Address
	Timestamp         *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterVerificationCompleted is a free log retrieval operation binding the contract event 0x16b634718516af0af7235a764f511944fb8fa5b16bf9ddc221c1894ece3e1459.
//
// Solidity: event VerificationCompleted(address indexed safeWalletAddress, address indexed systemConfigProxy, address indexed proxyAdmin, uint256 timestamp)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterVerificationCompleted(opts *bind.FilterOpts, safeWalletAddress []common.Address, systemConfigProxy []common.Address, proxyAdmin []common.Address) (*L1ContractVerificationVerificationCompletedIterator, error) {

	var safeWalletAddressRule []interface{}
	for _, safeWalletAddressItem := range safeWalletAddress {
		safeWalletAddressRule = append(safeWalletAddressRule, safeWalletAddressItem)
	}
	var systemConfigProxyRule []interface{}
	for _, systemConfigProxyItem := range systemConfigProxy {
		systemConfigProxyRule = append(systemConfigProxyRule, systemConfigProxyItem)
	}
	var proxyAdminRule []interface{}
	for _, proxyAdminItem := range proxyAdmin {
		proxyAdminRule = append(proxyAdminRule, proxyAdminItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "VerificationCompleted", safeWalletAddressRule, systemConfigProxyRule, proxyAdminRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationVerificationCompletedIterator{contract: _L1ContractVerification.contract, event: "VerificationCompleted", logs: logs, sub: sub}, nil
}

// WatchVerificationCompleted is a free log subscription operation binding the contract event 0x16b634718516af0af7235a764f511944fb8fa5b16bf9ddc221c1894ece3e1459.
//
// Solidity: event VerificationCompleted(address indexed safeWalletAddress, address indexed systemConfigProxy, address indexed proxyAdmin, uint256 timestamp)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchVerificationCompleted(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationVerificationCompleted, safeWalletAddress []common.Address, systemConfigProxy []common.Address, proxyAdmin []common.Address) (event.Subscription, error) {

	var safeWalletAddressRule []interface{}
	for _, safeWalletAddressItem := range safeWalletAddress {
		safeWalletAddressRule = append(safeWalletAddressRule, safeWalletAddressItem)
	}
	var systemConfigProxyRule []interface{}
	for _, systemConfigProxyItem := range systemConfigProxy {
		systemConfigProxyRule = append(systemConfigProxyRule, systemConfigProxyItem)
	}
	var proxyAdminRule []interface{}
	for _, proxyAdminItem := range proxyAdmin {
		proxyAdminRule = append(proxyAdminRule, proxyAdminItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "VerificationCompleted", safeWalletAddressRule, systemConfigProxyRule, proxyAdminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationVerificationCompleted)
				if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationCompleted", log); err != nil {
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

// ParseVerificationCompleted is a log parse operation binding the contract event 0x16b634718516af0af7235a764f511944fb8fa5b16bf9ddc221c1894ece3e1459.
//
// Solidity: event VerificationCompleted(address indexed safeWalletAddress, address indexed systemConfigProxy, address indexed proxyAdmin, uint256 timestamp)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseVerificationCompleted(log types.Log) (*L1ContractVerificationVerificationCompleted, error) {
	event := new(L1ContractVerificationVerificationCompleted)
	if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationCompleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationVerificationPossibleSetIterator is returned from FilterVerificationPossibleSet and is used to iterate over the raw logs and unpacked data for VerificationPossibleSet events raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationPossibleSetIterator struct {
	Event *L1ContractVerificationVerificationPossibleSet // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationVerificationPossibleSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationVerificationPossibleSet)
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
		it.Event = new(L1ContractVerificationVerificationPossibleSet)
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
func (it *L1ContractVerificationVerificationPossibleSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationVerificationPossibleSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationVerificationPossibleSet represents a VerificationPossibleSet event raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationPossibleSet struct {
	IsVerificationPossible bool
	Raw                    types.Log // Blockchain specific contextual infos
}

// FilterVerificationPossibleSet is a free log retrieval operation binding the contract event 0xbcda4cba7e7c1304909c614c883771c47e4a34768350beec462af4f1fba2dc6e.
//
// Solidity: event VerificationPossibleSet(bool indexed isVerificationPossible)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterVerificationPossibleSet(opts *bind.FilterOpts, isVerificationPossible []bool) (*L1ContractVerificationVerificationPossibleSetIterator, error) {

	var isVerificationPossibleRule []interface{}
	for _, isVerificationPossibleItem := range isVerificationPossible {
		isVerificationPossibleRule = append(isVerificationPossibleRule, isVerificationPossibleItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "VerificationPossibleSet", isVerificationPossibleRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationVerificationPossibleSetIterator{contract: _L1ContractVerification.contract, event: "VerificationPossibleSet", logs: logs, sub: sub}, nil
}

// WatchVerificationPossibleSet is a free log subscription operation binding the contract event 0xbcda4cba7e7c1304909c614c883771c47e4a34768350beec462af4f1fba2dc6e.
//
// Solidity: event VerificationPossibleSet(bool indexed isVerificationPossible)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchVerificationPossibleSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationVerificationPossibleSet, isVerificationPossible []bool) (event.Subscription, error) {

	var isVerificationPossibleRule []interface{}
	for _, isVerificationPossibleItem := range isVerificationPossible {
		isVerificationPossibleRule = append(isVerificationPossibleRule, isVerificationPossibleItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "VerificationPossibleSet", isVerificationPossibleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationVerificationPossibleSet)
				if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationPossibleSet", log); err != nil {
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

// ParseVerificationPossibleSet is a log parse operation binding the contract event 0xbcda4cba7e7c1304909c614c883771c47e4a34768350beec462af4f1fba2dc6e.
//
// Solidity: event VerificationPossibleSet(bool indexed isVerificationPossible)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseVerificationPossibleSet(log types.Log) (*L1ContractVerificationVerificationPossibleSet, error) {
	event := new(L1ContractVerificationVerificationPossibleSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationPossibleSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
