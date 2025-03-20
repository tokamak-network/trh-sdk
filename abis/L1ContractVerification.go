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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_tokenAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bridgeRegistry\",\"type\":\"address\"}],\"name\":\"BridgeRegistryUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"string\",\"name\":\"contractName\",\"type\":\"string\"}],\"name\":\"ConfigurationSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"name\":\"NativeTokenSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"codehash\",\"type\":\"bytes32\"}],\"name\":\"ProxyAdminCodehashSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"verifier\",\"type\":\"address\"}],\"name\":\"RegistrationSuccess\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"tokamakDAO\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"foundation\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"}],\"name\":\"SafeConfigSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"verifier\",\"type\":\"address\"}],\"name\":\"VerificationSuccess\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"expectedNativeToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BridgeRegistryAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1CrossDomainMessenger\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"logicAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"proxyCodehash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1StandardBridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"logicAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"proxyCodehash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"optimismPortal\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"logicAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"proxyCodehash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxyAdminCodehash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"safeWallet\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"safeWalletAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokamakDAO\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"foundation\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"implementationCodehash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"proxyCodehash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"requiredThreshold\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bridgeRegistry\",\"type\":\"address\"}],\"name\":\"setBridgeRegistryAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_proxyAdmin\",\"type\":\"address\"}],\"name\":\"setLogicContractInfo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_proxyAdmin\",\"type\":\"address\"}],\"name\":\"setProxyAdminCodeHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_tokamakDAO\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_foundation\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_threshold\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"_implementationCodehash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_proxyCodehash\",\"type\":\"bytes32\"}],\"name\":\"setSafeConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"systemConfig\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"logicAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"proxyCodehash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"verifyAndRegisterRollupConfig\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"proxyAdmin\",\"type\":\"address\"}],\"name\":\"verifyL1Contracts\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ContractVerification *L1ContractVerificationSession) Owner() (common.Address, error) {
	return _L1ContractVerification.Contract.Owner(&_L1ContractVerification.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCallerSession) Owner() (common.Address, error) {
	return _L1ContractVerification.Contract.Owner(&_L1ContractVerification.CallOpts)
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

// SafeWallet is a free data retrieval call binding the contract method 0x88cfce56.
//
// Solidity: function safeWallet() view returns(address safeWalletAddress, address tokamakDAO, address foundation, bytes32 implementationCodehash, bytes32 proxyCodehash, uint256 requiredThreshold)
func (_L1ContractVerification *L1ContractVerificationCaller) SafeWallet(opts *bind.CallOpts) (struct {
	SafeWalletAddress      common.Address
	TokamakDAO             common.Address
	Foundation             common.Address
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	RequiredThreshold      *big.Int
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "safeWallet")

	outstruct := new(struct {
		SafeWalletAddress      common.Address
		TokamakDAO             common.Address
		Foundation             common.Address
		ImplementationCodehash [32]byte
		ProxyCodehash          [32]byte
		RequiredThreshold      *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SafeWalletAddress = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.TokamakDAO = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Foundation = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.ImplementationCodehash = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)
	outstruct.ProxyCodehash = *abi.ConvertType(out[4], new([32]byte)).(*[32]byte)
	outstruct.RequiredThreshold = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// SafeWallet is a free data retrieval call binding the contract method 0x88cfce56.
//
// Solidity: function safeWallet() view returns(address safeWalletAddress, address tokamakDAO, address foundation, bytes32 implementationCodehash, bytes32 proxyCodehash, uint256 requiredThreshold)
func (_L1ContractVerification *L1ContractVerificationSession) SafeWallet() (struct {
	SafeWalletAddress      common.Address
	TokamakDAO             common.Address
	Foundation             common.Address
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	RequiredThreshold      *big.Int
}, error) {
	return _L1ContractVerification.Contract.SafeWallet(&_L1ContractVerification.CallOpts)
}

// SafeWallet is a free data retrieval call binding the contract method 0x88cfce56.
//
// Solidity: function safeWallet() view returns(address safeWalletAddress, address tokamakDAO, address foundation, bytes32 implementationCodehash, bytes32 proxyCodehash, uint256 requiredThreshold)
func (_L1ContractVerification *L1ContractVerificationCallerSession) SafeWallet() (struct {
	SafeWalletAddress      common.Address
	TokamakDAO             common.Address
	Foundation             common.Address
	ImplementationCodehash [32]byte
	ProxyCodehash          [32]byte
	RequiredThreshold      *big.Int
}, error) {
	return _L1ContractVerification.Contract.SafeWallet(&_L1ContractVerification.CallOpts)
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

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ContractVerification *L1ContractVerificationSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RenounceOwnership(&_L1ContractVerification.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1ContractVerification.Contract.RenounceOwnership(&_L1ContractVerification.TransactOpts)
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

// SetProxyAdminCodeHash is a paid mutator transaction binding the contract method 0xe4a88622.
//
// Solidity: function setProxyAdminCodeHash(address _proxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetProxyAdminCodeHash(opts *bind.TransactOpts, _proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setProxyAdminCodeHash", _proxyAdmin)
}

// SetProxyAdminCodeHash is a paid mutator transaction binding the contract method 0xe4a88622.
//
// Solidity: function setProxyAdminCodeHash(address _proxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetProxyAdminCodeHash(_proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetProxyAdminCodeHash(&_L1ContractVerification.TransactOpts, _proxyAdmin)
}

// SetProxyAdminCodeHash is a paid mutator transaction binding the contract method 0xe4a88622.
//
// Solidity: function setProxyAdminCodeHash(address _proxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetProxyAdminCodeHash(_proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetProxyAdminCodeHash(&_L1ContractVerification.TransactOpts, _proxyAdmin)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0x4d28fbf2.
//
// Solidity: function setSafeConfig(address _tokamakDAO, address _foundation, uint256 _threshold, address _proxyAdmin, bytes32 _implementationCodehash, bytes32 _proxyCodehash) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetSafeConfig(opts *bind.TransactOpts, _tokamakDAO common.Address, _foundation common.Address, _threshold *big.Int, _proxyAdmin common.Address, _implementationCodehash [32]byte, _proxyCodehash [32]byte) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setSafeConfig", _tokamakDAO, _foundation, _threshold, _proxyAdmin, _implementationCodehash, _proxyCodehash)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0x4d28fbf2.
//
// Solidity: function setSafeConfig(address _tokamakDAO, address _foundation, uint256 _threshold, address _proxyAdmin, bytes32 _implementationCodehash, bytes32 _proxyCodehash) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetSafeConfig(_tokamakDAO common.Address, _foundation common.Address, _threshold *big.Int, _proxyAdmin common.Address, _implementationCodehash [32]byte, _proxyCodehash [32]byte) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeConfig(&_L1ContractVerification.TransactOpts, _tokamakDAO, _foundation, _threshold, _proxyAdmin, _implementationCodehash, _proxyCodehash)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0x4d28fbf2.
//
// Solidity: function setSafeConfig(address _tokamakDAO, address _foundation, uint256 _threshold, address _proxyAdmin, bytes32 _implementationCodehash, bytes32 _proxyCodehash) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetSafeConfig(_tokamakDAO common.Address, _foundation common.Address, _threshold *big.Int, _proxyAdmin common.Address, _implementationCodehash [32]byte, _proxyCodehash [32]byte) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeConfig(&_L1ContractVerification.TransactOpts, _tokamakDAO, _foundation, _threshold, _proxyAdmin, _implementationCodehash, _proxyCodehash)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ContractVerification *L1ContractVerificationSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.TransferOwnership(&_L1ContractVerification.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.TransferOwnership(&_L1ContractVerification.TransactOpts, newOwner)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x6b4c4199.
//
// Solidity: function verifyAndRegisterRollupConfig(address _systemConfigProxy, address _proxyAdmin, uint8 _type, address _l2TON, string _name) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactor) VerifyAndRegisterRollupConfig(opts *bind.TransactOpts, _systemConfigProxy common.Address, _proxyAdmin common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "verifyAndRegisterRollupConfig", _systemConfigProxy, _proxyAdmin, _type, _l2TON, _name)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x6b4c4199.
//
// Solidity: function verifyAndRegisterRollupConfig(address _systemConfigProxy, address _proxyAdmin, uint8 _type, address _l2TON, string _name) returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) VerifyAndRegisterRollupConfig(_systemConfigProxy common.Address, _proxyAdmin common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyAndRegisterRollupConfig(&_L1ContractVerification.TransactOpts, _systemConfigProxy, _proxyAdmin, _type, _l2TON, _name)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x6b4c4199.
//
// Solidity: function verifyAndRegisterRollupConfig(address _systemConfigProxy, address _proxyAdmin, uint8 _type, address _l2TON, string _name) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactorSession) VerifyAndRegisterRollupConfig(_systemConfigProxy common.Address, _proxyAdmin common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyAndRegisterRollupConfig(&_L1ContractVerification.TransactOpts, _systemConfigProxy, _proxyAdmin, _type, _l2TON, _name)
}

// VerifyL1Contracts is a paid mutator transaction binding the contract method 0xb4172b3c.
//
// Solidity: function verifyL1Contracts(address systemConfigProxy, address proxyAdmin) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactor) VerifyL1Contracts(opts *bind.TransactOpts, systemConfigProxy common.Address, proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "verifyL1Contracts", systemConfigProxy, proxyAdmin)
}

// VerifyL1Contracts is a paid mutator transaction binding the contract method 0xb4172b3c.
//
// Solidity: function verifyL1Contracts(address systemConfigProxy, address proxyAdmin) returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) VerifyL1Contracts(systemConfigProxy common.Address, proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyL1Contracts(&_L1ContractVerification.TransactOpts, systemConfigProxy, proxyAdmin)
}

// VerifyL1Contracts is a paid mutator transaction binding the contract method 0xb4172b3c.
//
// Solidity: function verifyL1Contracts(address systemConfigProxy, address proxyAdmin) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactorSession) VerifyL1Contracts(systemConfigProxy common.Address, proxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyL1Contracts(&_L1ContractVerification.TransactOpts, systemConfigProxy, proxyAdmin)
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

// L1ContractVerificationOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L1ContractVerification contract.
type L1ContractVerificationOwnershipTransferredIterator struct {
	Event *L1ContractVerificationOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationOwnershipTransferred)
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
		it.Event = new(L1ContractVerificationOwnershipTransferred)
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
func (it *L1ContractVerificationOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationOwnershipTransferred represents a OwnershipTransferred event raised by the L1ContractVerification contract.
type L1ContractVerificationOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L1ContractVerificationOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationOwnershipTransferredIterator{contract: _L1ContractVerification.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationOwnershipTransferred)
				if err := _L1ContractVerification.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseOwnershipTransferred(log types.Log) (*L1ContractVerificationOwnershipTransferred, error) {
	event := new(L1ContractVerificationOwnershipTransferred)
	if err := _L1ContractVerification.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
// Solidity: event ProxyAdminCodehashSet(bytes32 codehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterProxyAdminCodehashSet(opts *bind.FilterOpts) (*L1ContractVerificationProxyAdminCodehashSetIterator, error) {

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "ProxyAdminCodehashSet")
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationProxyAdminCodehashSetIterator{contract: _L1ContractVerification.contract, event: "ProxyAdminCodehashSet", logs: logs, sub: sub}, nil
}

// WatchProxyAdminCodehashSet is a free log subscription operation binding the contract event 0x834dd2873e30c49e8619df33022f0db6fb97e8ba6b523ca3bef537f8f48792bb.
//
// Solidity: event ProxyAdminCodehashSet(bytes32 codehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchProxyAdminCodehashSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationProxyAdminCodehashSet) (event.Subscription, error) {

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "ProxyAdminCodehashSet")
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
// Solidity: event ProxyAdminCodehashSet(bytes32 codehash)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseProxyAdminCodehashSet(log types.Log) (*L1ContractVerificationProxyAdminCodehashSet, error) {
	event := new(L1ContractVerificationProxyAdminCodehashSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "ProxyAdminCodehashSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationRegistrationSuccessIterator is returned from FilterRegistrationSuccess and is used to iterate over the raw logs and unpacked data for RegistrationSuccess events raised by the L1ContractVerification contract.
type L1ContractVerificationRegistrationSuccessIterator struct {
	Event *L1ContractVerificationRegistrationSuccess // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationRegistrationSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationRegistrationSuccess)
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
		it.Event = new(L1ContractVerificationRegistrationSuccess)
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
func (it *L1ContractVerificationRegistrationSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationRegistrationSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationRegistrationSuccess represents a RegistrationSuccess event raised by the L1ContractVerification contract.
type L1ContractVerificationRegistrationSuccess struct {
	Verifier common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterRegistrationSuccess is a free log retrieval operation binding the contract event 0x8dcd7c08b0210742c621c9ac106f92321fed60215a0adcd8d5c377e1e0460a41.
//
// Solidity: event RegistrationSuccess(address indexed verifier)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterRegistrationSuccess(opts *bind.FilterOpts, verifier []common.Address) (*L1ContractVerificationRegistrationSuccessIterator, error) {

	var verifierRule []interface{}
	for _, verifierItem := range verifier {
		verifierRule = append(verifierRule, verifierItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "RegistrationSuccess", verifierRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationRegistrationSuccessIterator{contract: _L1ContractVerification.contract, event: "RegistrationSuccess", logs: logs, sub: sub}, nil
}

// WatchRegistrationSuccess is a free log subscription operation binding the contract event 0x8dcd7c08b0210742c621c9ac106f92321fed60215a0adcd8d5c377e1e0460a41.
//
// Solidity: event RegistrationSuccess(address indexed verifier)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchRegistrationSuccess(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationRegistrationSuccess, verifier []common.Address) (event.Subscription, error) {

	var verifierRule []interface{}
	for _, verifierItem := range verifier {
		verifierRule = append(verifierRule, verifierItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "RegistrationSuccess", verifierRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationRegistrationSuccess)
				if err := _L1ContractVerification.contract.UnpackLog(event, "RegistrationSuccess", log); err != nil {
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

// ParseRegistrationSuccess is a log parse operation binding the contract event 0x8dcd7c08b0210742c621c9ac106f92321fed60215a0adcd8d5c377e1e0460a41.
//
// Solidity: event RegistrationSuccess(address indexed verifier)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseRegistrationSuccess(log types.Log) (*L1ContractVerificationRegistrationSuccess, error) {
	event := new(L1ContractVerificationRegistrationSuccess)
	if err := _L1ContractVerification.contract.UnpackLog(event, "RegistrationSuccess", log); err != nil {
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
	TokamakDAO common.Address
	Foundation common.Address
	Threshold  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSafeConfigSet is a free log retrieval operation binding the contract event 0x8fdfd2ab3f2409d767e0714693cf22c2054f29719d2fd50397eef6e104225840.
//
// Solidity: event SafeConfigSet(address tokamakDAO, address foundation, uint256 threshold)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterSafeConfigSet(opts *bind.FilterOpts) (*L1ContractVerificationSafeConfigSetIterator, error) {

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "SafeConfigSet")
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationSafeConfigSetIterator{contract: _L1ContractVerification.contract, event: "SafeConfigSet", logs: logs, sub: sub}, nil
}

// WatchSafeConfigSet is a free log subscription operation binding the contract event 0x8fdfd2ab3f2409d767e0714693cf22c2054f29719d2fd50397eef6e104225840.
//
// Solidity: event SafeConfigSet(address tokamakDAO, address foundation, uint256 threshold)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchSafeConfigSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationSafeConfigSet) (event.Subscription, error) {

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "SafeConfigSet")
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

// ParseSafeConfigSet is a log parse operation binding the contract event 0x8fdfd2ab3f2409d767e0714693cf22c2054f29719d2fd50397eef6e104225840.
//
// Solidity: event SafeConfigSet(address tokamakDAO, address foundation, uint256 threshold)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseSafeConfigSet(log types.Log) (*L1ContractVerificationSafeConfigSet, error) {
	event := new(L1ContractVerificationSafeConfigSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "SafeConfigSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationVerificationSuccessIterator is returned from FilterVerificationSuccess and is used to iterate over the raw logs and unpacked data for VerificationSuccess events raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationSuccessIterator struct {
	Event *L1ContractVerificationVerificationSuccess // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationVerificationSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationVerificationSuccess)
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
		it.Event = new(L1ContractVerificationVerificationSuccess)
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
func (it *L1ContractVerificationVerificationSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationVerificationSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationVerificationSuccess represents a VerificationSuccess event raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationSuccess struct {
	Verifier common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterVerificationSuccess is a free log retrieval operation binding the contract event 0xe7d3ae9adcd435646a0c1db62e35bd781df450b6197d8a22be132dfb7f736197.
//
// Solidity: event VerificationSuccess(address indexed verifier)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterVerificationSuccess(opts *bind.FilterOpts, verifier []common.Address) (*L1ContractVerificationVerificationSuccessIterator, error) {

	var verifierRule []interface{}
	for _, verifierItem := range verifier {
		verifierRule = append(verifierRule, verifierItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "VerificationSuccess", verifierRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationVerificationSuccessIterator{contract: _L1ContractVerification.contract, event: "VerificationSuccess", logs: logs, sub: sub}, nil
}

// WatchVerificationSuccess is a free log subscription operation binding the contract event 0xe7d3ae9adcd435646a0c1db62e35bd781df450b6197d8a22be132dfb7f736197.
//
// Solidity: event VerificationSuccess(address indexed verifier)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchVerificationSuccess(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationVerificationSuccess, verifier []common.Address) (event.Subscription, error) {

	var verifierRule []interface{}
	for _, verifierItem := range verifier {
		verifierRule = append(verifierRule, verifierItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "VerificationSuccess", verifierRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationVerificationSuccess)
				if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationSuccess", log); err != nil {
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

// ParseVerificationSuccess is a log parse operation binding the contract event 0xe7d3ae9adcd435646a0c1db62e35bd781df450b6197d8a22be132dfb7f736197.
//
// Solidity: event VerificationSuccess(address indexed verifier)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseVerificationSuccess(log types.Log) (*L1ContractVerificationVerificationSuccess, error) {
	event := new(L1ContractVerificationVerificationSuccess)
	if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationSuccess", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
