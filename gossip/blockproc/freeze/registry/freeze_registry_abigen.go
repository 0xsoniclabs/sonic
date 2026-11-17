// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

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

// RegistryMetaData contains all meta data concerning the Registry contract.
var RegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"isFrozen\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"isOptedOut\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"value\",\"type\":\"bool\"}],\"name\":\"setFrozen\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"value\",\"type\":\"bool\"}],\"name\":\"setOptedOut\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506102508061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c8063ac869cd81461004e578063c2b480dd146100a7578063e5839836146100fe578063ff1468f21461015c575b5f5ffd5b6100a561005c3660046101c1565b6001600160a01b03919091165f9081527f79438978416e308cc48db3f2f690ad3d9d4d8848bddb35c7c4dfe2c0165d590060205260409020805460ff1916911515919091179055565b005b6100a56100b53660046101c1565b6001600160a01b03919091165f9081527f79438978416e308cc48db3f2f690ad3d9d4d8848bddb35c7c4dfe2c0165d590160205260409020805460ff1916911515919091179055565b61014861010c3660046101fa565b6001600160a01b03165f9081527f79438978416e308cc48db3f2f690ad3d9d4d8848bddb35c7c4dfe2c0165d5900602052604090205460ff1690565b604051901515815260200160405180910390f35b61014861016a3660046101fa565b6001600160a01b03165f9081527f79438978416e308cc48db3f2f690ad3d9d4d8848bddb35c7c4dfe2c0165d5901602052604090205460ff1690565b80356001600160a01b03811681146101bc575f5ffd5b919050565b5f5f604083850312156101d2575f5ffd5b6101db836101a6565b9150602083013580151581146101ef575f5ffd5b809150509250929050565b5f6020828403121561020a575f5ffd5b610213826101a6565b939250505056fea26469706673582212207cb17b27c74e5e67fb13344dca79b7a2f31511dfc96c6458c01be2bee3f35a9664736f6c63430008260033",
}

// RegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryMetaData.ABI instead.
var RegistryABI = RegistryMetaData.ABI

// RegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RegistryMetaData.Bin instead.
var RegistryBin = RegistryMetaData.Bin

// DeployRegistry deploys a new Ethereum contract, binding an instance of Registry to it.
func DeployRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Registry, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// Registry is an auto generated Go binding around an Ethereum contract.
type Registry struct {
	RegistryCaller     // Read-only binding to the contract
	RegistryTransactor // Write-only binding to the contract
	RegistryFilterer   // Log filterer for contract events
}

// RegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistrySession struct {
	Contract     *Registry         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryCallerSession struct {
	Contract *RegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryTransactorSession struct {
	Contract     *RegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryRaw struct {
	Contract *Registry // Generic contract binding to access the raw methods on
}

// RegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryCallerRaw struct {
	Contract *RegistryCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryTransactorRaw struct {
	Contract *RegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistry creates a new instance of Registry, bound to a specific deployed contract.
func NewRegistry(address common.Address, backend bind.ContractBackend) (*Registry, error) {
	contract, err := bindRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// NewRegistryCaller creates a new read-only instance of Registry, bound to a specific deployed contract.
func NewRegistryCaller(address common.Address, caller bind.ContractCaller) (*RegistryCaller, error) {
	contract, err := bindRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryCaller{contract: contract}, nil
}

// NewRegistryTransactor creates a new write-only instance of Registry, bound to a specific deployed contract.
func NewRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryTransactor, error) {
	contract, err := bindRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryTransactor{contract: contract}, nil
}

// NewRegistryFilterer creates a new log filterer instance of Registry, bound to a specific deployed contract.
func NewRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryFilterer, error) {
	contract, err := bindRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryFilterer{contract: contract}, nil
}

// bindRegistry binds a generic wrapper to an already deployed contract.
func bindRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.RegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transact(opts, method, params...)
}

// IsFrozen is a free data retrieval call binding the contract method 0xe5839836.
//
// Solidity: function isFrozen(address account) view returns(bool)
func (_Registry *RegistryCaller) IsFrozen(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isFrozen", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsFrozen is a free data retrieval call binding the contract method 0xe5839836.
//
// Solidity: function isFrozen(address account) view returns(bool)
func (_Registry *RegistrySession) IsFrozen(account common.Address) (bool, error) {
	return _Registry.Contract.IsFrozen(&_Registry.CallOpts, account)
}

// IsFrozen is a free data retrieval call binding the contract method 0xe5839836.
//
// Solidity: function isFrozen(address account) view returns(bool)
func (_Registry *RegistryCallerSession) IsFrozen(account common.Address) (bool, error) {
	return _Registry.Contract.IsFrozen(&_Registry.CallOpts, account)
}

// IsOptedOut is a free data retrieval call binding the contract method 0xff1468f2.
//
// Solidity: function isOptedOut(address account) view returns(bool)
func (_Registry *RegistryCaller) IsOptedOut(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isOptedOut", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOptedOut is a free data retrieval call binding the contract method 0xff1468f2.
//
// Solidity: function isOptedOut(address account) view returns(bool)
func (_Registry *RegistrySession) IsOptedOut(account common.Address) (bool, error) {
	return _Registry.Contract.IsOptedOut(&_Registry.CallOpts, account)
}

// IsOptedOut is a free data retrieval call binding the contract method 0xff1468f2.
//
// Solidity: function isOptedOut(address account) view returns(bool)
func (_Registry *RegistryCallerSession) IsOptedOut(account common.Address) (bool, error) {
	return _Registry.Contract.IsOptedOut(&_Registry.CallOpts, account)
}

// SetFrozen is a paid mutator transaction binding the contract method 0xac869cd8.
//
// Solidity: function setFrozen(address account, bool value) returns()
func (_Registry *RegistryTransactor) SetFrozen(opts *bind.TransactOpts, account common.Address, value bool) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setFrozen", account, value)
}

// SetFrozen is a paid mutator transaction binding the contract method 0xac869cd8.
//
// Solidity: function setFrozen(address account, bool value) returns()
func (_Registry *RegistrySession) SetFrozen(account common.Address, value bool) (*types.Transaction, error) {
	return _Registry.Contract.SetFrozen(&_Registry.TransactOpts, account, value)
}

// SetFrozen is a paid mutator transaction binding the contract method 0xac869cd8.
//
// Solidity: function setFrozen(address account, bool value) returns()
func (_Registry *RegistryTransactorSession) SetFrozen(account common.Address, value bool) (*types.Transaction, error) {
	return _Registry.Contract.SetFrozen(&_Registry.TransactOpts, account, value)
}

// SetOptedOut is a paid mutator transaction binding the contract method 0xc2b480dd.
//
// Solidity: function setOptedOut(address account, bool value) returns()
func (_Registry *RegistryTransactor) SetOptedOut(opts *bind.TransactOpts, account common.Address, value bool) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setOptedOut", account, value)
}

// SetOptedOut is a paid mutator transaction binding the contract method 0xc2b480dd.
//
// Solidity: function setOptedOut(address account, bool value) returns()
func (_Registry *RegistrySession) SetOptedOut(account common.Address, value bool) (*types.Transaction, error) {
	return _Registry.Contract.SetOptedOut(&_Registry.TransactOpts, account, value)
}

// SetOptedOut is a paid mutator transaction binding the contract method 0xc2b480dd.
//
// Solidity: function setOptedOut(address account, bool value) returns()
func (_Registry *RegistryTransactorSession) SetOptedOut(account common.Address, value bool) (*types.Transaction, error) {
	return _Registry.Contract.SetOptedOut(&_Registry.TransactOpts, account, value)
}
