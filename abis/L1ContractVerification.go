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

// IL1ContractVerificationSafeConfig is an auto generated low-level Go binding around an user-defined struct.
type IL1ContractVerificationSafeConfig struct {
	TokamakDAO        common.Address
	Foundation        common.Address
	ThirdOwner        common.Address
	RequiredThreshold *big.Int
}

// L1ContractVerificationMetaData contains all meta data concerning the L1ContractVerification contract.
var L1ContractVerificationMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_proxyAdminAddress\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bridgeRegistry\",\"type\":\"address\"}],\"name\":\"BridgeRegistryUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"contractId\",\"type\":\"bytes32\"}],\"name\":\"ConfigurationSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"verifier\",\"type\":\"address\"}],\"name\":\"RegistrationSuccess\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"required\",\"type\":\"bool\"}],\"name\":\"SafeVerificationRequiredSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"VerificationFailure\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"verifier\",\"type\":\"address\"}],\"name\":\"VerificationSuccess\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"L1BridgeRegistryV1_1Address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L1_CROSS_DOMAIN_MESSENGER\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L1_STANDARD_BRIDGE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OPTIMISM_PORTAL\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PROXY_ADMIN_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SYSTEM_CONFIG\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chainConfigs\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokamakDAO\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"foundation\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"thirdOwner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"requiredThreshold\",\"type\":\"uint256\"}],\"internalType\":\"structIL1ContractVerification.SafeConfig\",\"name\":\"safeConfig\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"expectedNativeToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"safeVerificationRequired\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"contractId\",\"type\":\"bytes32\"}],\"name\":\"getContractConfig\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"implementationHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"proxyHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"expectedProxyAdmin\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getSafeConfig\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"tokamakDAO\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"foundation\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"thirdOwner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"requiredThreshold\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bridgeRegistry\",\"type\":\"address\"}],\"name\":\"setBridgeRegistryAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"contractId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"implementationHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"proxyHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"expectedProxyAdmin\",\"type\":\"address\"}],\"name\":\"setContractConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"name\":\"setExpectedNativeToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokamakDAO\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"foundation\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"thirdOwner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"}],\"name\":\"setSafeConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"required\",\"type\":\"bool\"}],\"name\":\"setSafeVerificationRequired\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"systemConfigProxy\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_type\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"_l2TON\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"verifyAndRegisterRollupConfig\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"systemConfigProxy\",\"type\":\"address\"}],\"name\":\"verifyL1Contracts\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
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

// L1BridgeRegistryV11Address is a free data retrieval call binding the contract method 0xe67634ec.
//
// Solidity: function L1BridgeRegistryV1_1Address() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCaller) L1BridgeRegistryV11Address(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "L1BridgeRegistryV1_1Address")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeRegistryV11Address is a free data retrieval call binding the contract method 0xe67634ec.
//
// Solidity: function L1BridgeRegistryV1_1Address() view returns(address)
func (_L1ContractVerification *L1ContractVerificationSession) L1BridgeRegistryV11Address() (common.Address, error) {
	return _L1ContractVerification.Contract.L1BridgeRegistryV11Address(&_L1ContractVerification.CallOpts)
}

// L1BridgeRegistryV11Address is a free data retrieval call binding the contract method 0xe67634ec.
//
// Solidity: function L1BridgeRegistryV1_1Address() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCallerSession) L1BridgeRegistryV11Address() (common.Address, error) {
	return _L1ContractVerification.Contract.L1BridgeRegistryV11Address(&_L1ContractVerification.CallOpts)
}

// L1CROSSDOMAINMESSENGER is a free data retrieval call binding the contract method 0xf904facb.
//
// Solidity: function L1_CROSS_DOMAIN_MESSENGER() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) L1CROSSDOMAINMESSENGER(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "L1_CROSS_DOMAIN_MESSENGER")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1CROSSDOMAINMESSENGER is a free data retrieval call binding the contract method 0xf904facb.
//
// Solidity: function L1_CROSS_DOMAIN_MESSENGER() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) L1CROSSDOMAINMESSENGER() ([32]byte, error) {
	return _L1ContractVerification.Contract.L1CROSSDOMAINMESSENGER(&_L1ContractVerification.CallOpts)
}

// L1CROSSDOMAINMESSENGER is a free data retrieval call binding the contract method 0xf904facb.
//
// Solidity: function L1_CROSS_DOMAIN_MESSENGER() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) L1CROSSDOMAINMESSENGER() ([32]byte, error) {
	return _L1ContractVerification.Contract.L1CROSSDOMAINMESSENGER(&_L1ContractVerification.CallOpts)
}

// L1STANDARDBRIDGE is a free data retrieval call binding the contract method 0x35a2db6a.
//
// Solidity: function L1_STANDARD_BRIDGE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) L1STANDARDBRIDGE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "L1_STANDARD_BRIDGE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1STANDARDBRIDGE is a free data retrieval call binding the contract method 0x35a2db6a.
//
// Solidity: function L1_STANDARD_BRIDGE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) L1STANDARDBRIDGE() ([32]byte, error) {
	return _L1ContractVerification.Contract.L1STANDARDBRIDGE(&_L1ContractVerification.CallOpts)
}

// L1STANDARDBRIDGE is a free data retrieval call binding the contract method 0x35a2db6a.
//
// Solidity: function L1_STANDARD_BRIDGE() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) L1STANDARDBRIDGE() ([32]byte, error) {
	return _L1ContractVerification.Contract.L1STANDARDBRIDGE(&_L1ContractVerification.CallOpts)
}

// OPTIMISMPORTAL is a free data retrieval call binding the contract method 0x85734ee1.
//
// Solidity: function OPTIMISM_PORTAL() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) OPTIMISMPORTAL(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "OPTIMISM_PORTAL")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// OPTIMISMPORTAL is a free data retrieval call binding the contract method 0x85734ee1.
//
// Solidity: function OPTIMISM_PORTAL() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) OPTIMISMPORTAL() ([32]byte, error) {
	return _L1ContractVerification.Contract.OPTIMISMPORTAL(&_L1ContractVerification.CallOpts)
}

// OPTIMISMPORTAL is a free data retrieval call binding the contract method 0x85734ee1.
//
// Solidity: function OPTIMISM_PORTAL() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) OPTIMISMPORTAL() ([32]byte, error) {
	return _L1ContractVerification.Contract.OPTIMISMPORTAL(&_L1ContractVerification.CallOpts)
}

// PROXYADMINADDRESS is a free data retrieval call binding the contract method 0x7e1dfea3.
//
// Solidity: function PROXY_ADMIN_ADDRESS() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCaller) PROXYADMINADDRESS(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "PROXY_ADMIN_ADDRESS")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PROXYADMINADDRESS is a free data retrieval call binding the contract method 0x7e1dfea3.
//
// Solidity: function PROXY_ADMIN_ADDRESS() view returns(address)
func (_L1ContractVerification *L1ContractVerificationSession) PROXYADMINADDRESS() (common.Address, error) {
	return _L1ContractVerification.Contract.PROXYADMINADDRESS(&_L1ContractVerification.CallOpts)
}

// PROXYADMINADDRESS is a free data retrieval call binding the contract method 0x7e1dfea3.
//
// Solidity: function PROXY_ADMIN_ADDRESS() view returns(address)
func (_L1ContractVerification *L1ContractVerificationCallerSession) PROXYADMINADDRESS() (common.Address, error) {
	return _L1ContractVerification.Contract.PROXYADMINADDRESS(&_L1ContractVerification.CallOpts)
}

// SYSTEMCONFIG is a free data retrieval call binding the contract method 0xf0498750.
//
// Solidity: function SYSTEM_CONFIG() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCaller) SYSTEMCONFIG(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "SYSTEM_CONFIG")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// SYSTEMCONFIG is a free data retrieval call binding the contract method 0xf0498750.
//
// Solidity: function SYSTEM_CONFIG() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationSession) SYSTEMCONFIG() ([32]byte, error) {
	return _L1ContractVerification.Contract.SYSTEMCONFIG(&_L1ContractVerification.CallOpts)
}

// SYSTEMCONFIG is a free data retrieval call binding the contract method 0xf0498750.
//
// Solidity: function SYSTEM_CONFIG() view returns(bytes32)
func (_L1ContractVerification *L1ContractVerificationCallerSession) SYSTEMCONFIG() ([32]byte, error) {
	return _L1ContractVerification.Contract.SYSTEMCONFIG(&_L1ContractVerification.CallOpts)
}

// ChainConfigs is a free data retrieval call binding the contract method 0x51bfbc32.
//
// Solidity: function chainConfigs(uint256 ) view returns((address,address,address,uint256) safeConfig, address expectedNativeToken, bool safeVerificationRequired)
func (_L1ContractVerification *L1ContractVerificationCaller) ChainConfigs(opts *bind.CallOpts, arg0 *big.Int) (struct {
	SafeConfig               IL1ContractVerificationSafeConfig
	ExpectedNativeToken      common.Address
	SafeVerificationRequired bool
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "chainConfigs", arg0)

	outstruct := new(struct {
		SafeConfig               IL1ContractVerificationSafeConfig
		ExpectedNativeToken      common.Address
		SafeVerificationRequired bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SafeConfig = *abi.ConvertType(out[0], new(IL1ContractVerificationSafeConfig)).(*IL1ContractVerificationSafeConfig)
	outstruct.ExpectedNativeToken = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.SafeVerificationRequired = *abi.ConvertType(out[2], new(bool)).(*bool)

	return *outstruct, err

}

// ChainConfigs is a free data retrieval call binding the contract method 0x51bfbc32.
//
// Solidity: function chainConfigs(uint256 ) view returns((address,address,address,uint256) safeConfig, address expectedNativeToken, bool safeVerificationRequired)
func (_L1ContractVerification *L1ContractVerificationSession) ChainConfigs(arg0 *big.Int) (struct {
	SafeConfig               IL1ContractVerificationSafeConfig
	ExpectedNativeToken      common.Address
	SafeVerificationRequired bool
}, error) {
	return _L1ContractVerification.Contract.ChainConfigs(&_L1ContractVerification.CallOpts, arg0)
}

// ChainConfigs is a free data retrieval call binding the contract method 0x51bfbc32.
//
// Solidity: function chainConfigs(uint256 ) view returns((address,address,address,uint256) safeConfig, address expectedNativeToken, bool safeVerificationRequired)
func (_L1ContractVerification *L1ContractVerificationCallerSession) ChainConfigs(arg0 *big.Int) (struct {
	SafeConfig               IL1ContractVerificationSafeConfig
	ExpectedNativeToken      common.Address
	SafeVerificationRequired bool
}, error) {
	return _L1ContractVerification.Contract.ChainConfigs(&_L1ContractVerification.CallOpts, arg0)
}

// GetContractConfig is a free data retrieval call binding the contract method 0xfe15a04d.
//
// Solidity: function getContractConfig(bytes32 contractId) view returns(bytes32 implementationHash, bytes32 proxyHash, address expectedProxyAdmin)
func (_L1ContractVerification *L1ContractVerificationCaller) GetContractConfig(opts *bind.CallOpts, contractId [32]byte) (struct {
	ImplementationHash [32]byte
	ProxyHash          [32]byte
	ExpectedProxyAdmin common.Address
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "getContractConfig", contractId)

	outstruct := new(struct {
		ImplementationHash [32]byte
		ProxyHash          [32]byte
		ExpectedProxyAdmin common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ImplementationHash = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.ProxyHash = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.ExpectedProxyAdmin = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// GetContractConfig is a free data retrieval call binding the contract method 0xfe15a04d.
//
// Solidity: function getContractConfig(bytes32 contractId) view returns(bytes32 implementationHash, bytes32 proxyHash, address expectedProxyAdmin)
func (_L1ContractVerification *L1ContractVerificationSession) GetContractConfig(contractId [32]byte) (struct {
	ImplementationHash [32]byte
	ProxyHash          [32]byte
	ExpectedProxyAdmin common.Address
}, error) {
	return _L1ContractVerification.Contract.GetContractConfig(&_L1ContractVerification.CallOpts, contractId)
}

// GetContractConfig is a free data retrieval call binding the contract method 0xfe15a04d.
//
// Solidity: function getContractConfig(bytes32 contractId) view returns(bytes32 implementationHash, bytes32 proxyHash, address expectedProxyAdmin)
func (_L1ContractVerification *L1ContractVerificationCallerSession) GetContractConfig(contractId [32]byte) (struct {
	ImplementationHash [32]byte
	ProxyHash          [32]byte
	ExpectedProxyAdmin common.Address
}, error) {
	return _L1ContractVerification.Contract.GetContractConfig(&_L1ContractVerification.CallOpts, contractId)
}

// GetSafeConfig is a free data retrieval call binding the contract method 0x2d5044b0.
//
// Solidity: function getSafeConfig() view returns(address tokamakDAO, address foundation, address thirdOwner, uint256 requiredThreshold)
func (_L1ContractVerification *L1ContractVerificationCaller) GetSafeConfig(opts *bind.CallOpts) (struct {
	TokamakDAO        common.Address
	Foundation        common.Address
	ThirdOwner        common.Address
	RequiredThreshold *big.Int
}, error) {
	var out []interface{}
	err := _L1ContractVerification.contract.Call(opts, &out, "getSafeConfig")

	outstruct := new(struct {
		TokamakDAO        common.Address
		Foundation        common.Address
		ThirdOwner        common.Address
		RequiredThreshold *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TokamakDAO = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Foundation = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.ThirdOwner = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.RequiredThreshold = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetSafeConfig is a free data retrieval call binding the contract method 0x2d5044b0.
//
// Solidity: function getSafeConfig() view returns(address tokamakDAO, address foundation, address thirdOwner, uint256 requiredThreshold)
func (_L1ContractVerification *L1ContractVerificationSession) GetSafeConfig() (struct {
	TokamakDAO        common.Address
	Foundation        common.Address
	ThirdOwner        common.Address
	RequiredThreshold *big.Int
}, error) {
	return _L1ContractVerification.Contract.GetSafeConfig(&_L1ContractVerification.CallOpts)
}

// GetSafeConfig is a free data retrieval call binding the contract method 0x2d5044b0.
//
// Solidity: function getSafeConfig() view returns(address tokamakDAO, address foundation, address thirdOwner, uint256 requiredThreshold)
func (_L1ContractVerification *L1ContractVerificationCallerSession) GetSafeConfig() (struct {
	TokamakDAO        common.Address
	Foundation        common.Address
	ThirdOwner        common.Address
	RequiredThreshold *big.Int
}, error) {
	return _L1ContractVerification.Contract.GetSafeConfig(&_L1ContractVerification.CallOpts)
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

// SetContractConfig is a paid mutator transaction binding the contract method 0xd51cdefa.
//
// Solidity: function setContractConfig(bytes32 contractId, bytes32 implementationHash, bytes32 proxyHash, address expectedProxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetContractConfig(opts *bind.TransactOpts, contractId [32]byte, implementationHash [32]byte, proxyHash [32]byte, expectedProxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setContractConfig", contractId, implementationHash, proxyHash, expectedProxyAdmin)
}

// SetContractConfig is a paid mutator transaction binding the contract method 0xd51cdefa.
//
// Solidity: function setContractConfig(bytes32 contractId, bytes32 implementationHash, bytes32 proxyHash, address expectedProxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetContractConfig(contractId [32]byte, implementationHash [32]byte, proxyHash [32]byte, expectedProxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetContractConfig(&_L1ContractVerification.TransactOpts, contractId, implementationHash, proxyHash, expectedProxyAdmin)
}

// SetContractConfig is a paid mutator transaction binding the contract method 0xd51cdefa.
//
// Solidity: function setContractConfig(bytes32 contractId, bytes32 implementationHash, bytes32 proxyHash, address expectedProxyAdmin) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetContractConfig(contractId [32]byte, implementationHash [32]byte, proxyHash [32]byte, expectedProxyAdmin common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetContractConfig(&_L1ContractVerification.TransactOpts, contractId, implementationHash, proxyHash, expectedProxyAdmin)
}

// SetExpectedNativeToken is a paid mutator transaction binding the contract method 0x32e92dee.
//
// Solidity: function setExpectedNativeToken(address tokenAddress) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetExpectedNativeToken(opts *bind.TransactOpts, tokenAddress common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setExpectedNativeToken", tokenAddress)
}

// SetExpectedNativeToken is a paid mutator transaction binding the contract method 0x32e92dee.
//
// Solidity: function setExpectedNativeToken(address tokenAddress) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetExpectedNativeToken(tokenAddress common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetExpectedNativeToken(&_L1ContractVerification.TransactOpts, tokenAddress)
}

// SetExpectedNativeToken is a paid mutator transaction binding the contract method 0x32e92dee.
//
// Solidity: function setExpectedNativeToken(address tokenAddress) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetExpectedNativeToken(tokenAddress common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetExpectedNativeToken(&_L1ContractVerification.TransactOpts, tokenAddress)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0xf0ff1d95.
//
// Solidity: function setSafeConfig(address tokamakDAO, address foundation, address thirdOwner, uint256 threshold) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetSafeConfig(opts *bind.TransactOpts, tokamakDAO common.Address, foundation common.Address, thirdOwner common.Address, threshold *big.Int) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setSafeConfig", tokamakDAO, foundation, thirdOwner, threshold)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0xf0ff1d95.
//
// Solidity: function setSafeConfig(address tokamakDAO, address foundation, address thirdOwner, uint256 threshold) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetSafeConfig(tokamakDAO common.Address, foundation common.Address, thirdOwner common.Address, threshold *big.Int) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeConfig(&_L1ContractVerification.TransactOpts, tokamakDAO, foundation, thirdOwner, threshold)
}

// SetSafeConfig is a paid mutator transaction binding the contract method 0xf0ff1d95.
//
// Solidity: function setSafeConfig(address tokamakDAO, address foundation, address thirdOwner, uint256 threshold) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetSafeConfig(tokamakDAO common.Address, foundation common.Address, thirdOwner common.Address, threshold *big.Int) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeConfig(&_L1ContractVerification.TransactOpts, tokamakDAO, foundation, thirdOwner, threshold)
}

// SetSafeVerificationRequired is a paid mutator transaction binding the contract method 0xc2d60eb8.
//
// Solidity: function setSafeVerificationRequired(bool required) returns()
func (_L1ContractVerification *L1ContractVerificationTransactor) SetSafeVerificationRequired(opts *bind.TransactOpts, required bool) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "setSafeVerificationRequired", required)
}

// SetSafeVerificationRequired is a paid mutator transaction binding the contract method 0xc2d60eb8.
//
// Solidity: function setSafeVerificationRequired(bool required) returns()
func (_L1ContractVerification *L1ContractVerificationSession) SetSafeVerificationRequired(required bool) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeVerificationRequired(&_L1ContractVerification.TransactOpts, required)
}

// SetSafeVerificationRequired is a paid mutator transaction binding the contract method 0xc2d60eb8.
//
// Solidity: function setSafeVerificationRequired(bool required) returns()
func (_L1ContractVerification *L1ContractVerificationTransactorSession) SetSafeVerificationRequired(required bool) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.SetSafeVerificationRequired(&_L1ContractVerification.TransactOpts, required)
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

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x034490b9.
//
// Solidity: function verifyAndRegisterRollupConfig(address systemConfigProxy, uint8 _type, address _l2TON, string _name) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactor) VerifyAndRegisterRollupConfig(opts *bind.TransactOpts, systemConfigProxy common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "verifyAndRegisterRollupConfig", systemConfigProxy, _type, _l2TON, _name)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x034490b9.
//
// Solidity: function verifyAndRegisterRollupConfig(address systemConfigProxy, uint8 _type, address _l2TON, string _name) returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) VerifyAndRegisterRollupConfig(systemConfigProxy common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyAndRegisterRollupConfig(&_L1ContractVerification.TransactOpts, systemConfigProxy, _type, _l2TON, _name)
}

// VerifyAndRegisterRollupConfig is a paid mutator transaction binding the contract method 0x034490b9.
//
// Solidity: function verifyAndRegisterRollupConfig(address systemConfigProxy, uint8 _type, address _l2TON, string _name) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactorSession) VerifyAndRegisterRollupConfig(systemConfigProxy common.Address, _type uint8, _l2TON common.Address, _name string) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyAndRegisterRollupConfig(&_L1ContractVerification.TransactOpts, systemConfigProxy, _type, _l2TON, _name)
}

// VerifyL1Contracts is a paid mutator transaction binding the contract method 0x89fe2f87.
//
// Solidity: function verifyL1Contracts(address systemConfigProxy) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactor) VerifyL1Contracts(opts *bind.TransactOpts, systemConfigProxy common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.contract.Transact(opts, "verifyL1Contracts", systemConfigProxy)
}

// VerifyL1Contracts is a paid mutator transaction binding the contract method 0x89fe2f87.
//
// Solidity: function verifyL1Contracts(address systemConfigProxy) returns(bool)
func (_L1ContractVerification *L1ContractVerificationSession) VerifyL1Contracts(systemConfigProxy common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyL1Contracts(&_L1ContractVerification.TransactOpts, systemConfigProxy)
}

// VerifyL1Contracts is a paid mutator transaction binding the contract method 0x89fe2f87.
//
// Solidity: function verifyL1Contracts(address systemConfigProxy) returns(bool)
func (_L1ContractVerification *L1ContractVerificationTransactorSession) VerifyL1Contracts(systemConfigProxy common.Address) (*types.Transaction, error) {
	return _L1ContractVerification.Contract.VerifyL1Contracts(&_L1ContractVerification.TransactOpts, systemConfigProxy)
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
	ContractId [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterConfigurationSet is a free log retrieval operation binding the contract event 0x58acddd661fe333e2843b91e39f7746bd581e75c4820870c0cbd781867ea64b6.
//
// Solidity: event ConfigurationSet(bytes32 indexed contractId)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterConfigurationSet(opts *bind.FilterOpts, contractId [][32]byte) (*L1ContractVerificationConfigurationSetIterator, error) {

	var contractIdRule []interface{}
	for _, contractIdItem := range contractId {
		contractIdRule = append(contractIdRule, contractIdItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "ConfigurationSet", contractIdRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationConfigurationSetIterator{contract: _L1ContractVerification.contract, event: "ConfigurationSet", logs: logs, sub: sub}, nil
}

// WatchConfigurationSet is a free log subscription operation binding the contract event 0x58acddd661fe333e2843b91e39f7746bd581e75c4820870c0cbd781867ea64b6.
//
// Solidity: event ConfigurationSet(bytes32 indexed contractId)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchConfigurationSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationConfigurationSet, contractId [][32]byte) (event.Subscription, error) {

	var contractIdRule []interface{}
	for _, contractIdItem := range contractId {
		contractIdRule = append(contractIdRule, contractIdItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "ConfigurationSet", contractIdRule)
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

// ParseConfigurationSet is a log parse operation binding the contract event 0x58acddd661fe333e2843b91e39f7746bd581e75c4820870c0cbd781867ea64b6.
//
// Solidity: event ConfigurationSet(bytes32 indexed contractId)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseConfigurationSet(log types.Log) (*L1ContractVerificationConfigurationSet, error) {
	event := new(L1ContractVerificationConfigurationSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "ConfigurationSet", log); err != nil {
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

// L1ContractVerificationSafeVerificationRequiredSetIterator is returned from FilterSafeVerificationRequiredSet and is used to iterate over the raw logs and unpacked data for SafeVerificationRequiredSet events raised by the L1ContractVerification contract.
type L1ContractVerificationSafeVerificationRequiredSetIterator struct {
	Event *L1ContractVerificationSafeVerificationRequiredSet // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationSafeVerificationRequiredSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationSafeVerificationRequiredSet)
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
		it.Event = new(L1ContractVerificationSafeVerificationRequiredSet)
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
func (it *L1ContractVerificationSafeVerificationRequiredSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationSafeVerificationRequiredSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationSafeVerificationRequiredSet represents a SafeVerificationRequiredSet event raised by the L1ContractVerification contract.
type L1ContractVerificationSafeVerificationRequiredSet struct {
	ChainId  *big.Int
	Required bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterSafeVerificationRequiredSet is a free log retrieval operation binding the contract event 0xd8293dca6523eb19283de4e3ac4d68739adda8465202391fd566ac9ec17430c4.
//
// Solidity: event SafeVerificationRequiredSet(uint256 indexed chainId, bool required)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterSafeVerificationRequiredSet(opts *bind.FilterOpts, chainId []*big.Int) (*L1ContractVerificationSafeVerificationRequiredSetIterator, error) {

	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "SafeVerificationRequiredSet", chainIdRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationSafeVerificationRequiredSetIterator{contract: _L1ContractVerification.contract, event: "SafeVerificationRequiredSet", logs: logs, sub: sub}, nil
}

// WatchSafeVerificationRequiredSet is a free log subscription operation binding the contract event 0xd8293dca6523eb19283de4e3ac4d68739adda8465202391fd566ac9ec17430c4.
//
// Solidity: event SafeVerificationRequiredSet(uint256 indexed chainId, bool required)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchSafeVerificationRequiredSet(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationSafeVerificationRequiredSet, chainId []*big.Int) (event.Subscription, error) {

	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "SafeVerificationRequiredSet", chainIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationSafeVerificationRequiredSet)
				if err := _L1ContractVerification.contract.UnpackLog(event, "SafeVerificationRequiredSet", log); err != nil {
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

// ParseSafeVerificationRequiredSet is a log parse operation binding the contract event 0xd8293dca6523eb19283de4e3ac4d68739adda8465202391fd566ac9ec17430c4.
//
// Solidity: event SafeVerificationRequiredSet(uint256 indexed chainId, bool required)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseSafeVerificationRequiredSet(log types.Log) (*L1ContractVerificationSafeVerificationRequiredSet, error) {
	event := new(L1ContractVerificationSafeVerificationRequiredSet)
	if err := _L1ContractVerification.contract.UnpackLog(event, "SafeVerificationRequiredSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ContractVerificationVerificationFailureIterator is returned from FilterVerificationFailure and is used to iterate over the raw logs and unpacked data for VerificationFailure events raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationFailureIterator struct {
	Event *L1ContractVerificationVerificationFailure // Event containing the contract specifics and raw log

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
func (it *L1ContractVerificationVerificationFailureIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ContractVerificationVerificationFailure)
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
		it.Event = new(L1ContractVerificationVerificationFailure)
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
func (it *L1ContractVerificationVerificationFailureIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ContractVerificationVerificationFailureIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ContractVerificationVerificationFailure represents a VerificationFailure event raised by the L1ContractVerification contract.
type L1ContractVerificationVerificationFailure struct {
	Operator common.Address
	Reason   string
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterVerificationFailure is a free log retrieval operation binding the contract event 0x05a24be642665ac29790d5b2805427718e343b1f5a1050e2b31a584a9987a5a5.
//
// Solidity: event VerificationFailure(address indexed operator, string reason)
func (_L1ContractVerification *L1ContractVerificationFilterer) FilterVerificationFailure(opts *bind.FilterOpts, operator []common.Address) (*L1ContractVerificationVerificationFailureIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _L1ContractVerification.contract.FilterLogs(opts, "VerificationFailure", operatorRule)
	if err != nil {
		return nil, err
	}
	return &L1ContractVerificationVerificationFailureIterator{contract: _L1ContractVerification.contract, event: "VerificationFailure", logs: logs, sub: sub}, nil
}

// WatchVerificationFailure is a free log subscription operation binding the contract event 0x05a24be642665ac29790d5b2805427718e343b1f5a1050e2b31a584a9987a5a5.
//
// Solidity: event VerificationFailure(address indexed operator, string reason)
func (_L1ContractVerification *L1ContractVerificationFilterer) WatchVerificationFailure(opts *bind.WatchOpts, sink chan<- *L1ContractVerificationVerificationFailure, operator []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _L1ContractVerification.contract.WatchLogs(opts, "VerificationFailure", operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ContractVerificationVerificationFailure)
				if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationFailure", log); err != nil {
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

// ParseVerificationFailure is a log parse operation binding the contract event 0x05a24be642665ac29790d5b2805427718e343b1f5a1050e2b31a584a9987a5a5.
//
// Solidity: event VerificationFailure(address indexed operator, string reason)
func (_L1ContractVerification *L1ContractVerificationFilterer) ParseVerificationFailure(log types.Log) (*L1ContractVerificationVerificationFailure, error) {
	event := new(L1ContractVerificationVerificationFailure)
	if err := _L1ContractVerification.contract.UnpackLog(event, "VerificationFailure", log); err != nil {
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
