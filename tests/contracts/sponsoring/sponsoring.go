// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package sponsoring

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

// SponsoringMetaData contains all meta data concerning the Sponsoring contract.
var SponsoringMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"execute\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103598061001c5f395ff3fe60806040526004361061001d575f3560e01c8063b61d27f614610021575b5f5ffd5b61003b600480360381019061003691906101e6565b61003d565b005b5f8473ffffffffffffffffffffffffffffffffffffffff16848484604051610066929190610293565b5f6040518083038185875af1925050503d805f81146100a0576040519150601f19603f3d011682016040523d82523d5f602084013e6100a5565b606091505b50509050806100e9576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016100e090610305565b60405180910390fd5b5050505050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610121826100f8565b9050919050565b61013181610117565b811461013b575f5ffd5b50565b5f8135905061014c81610128565b92915050565b5f819050919050565b61016481610152565b811461016e575f5ffd5b50565b5f8135905061017f8161015b565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f8401126101a6576101a5610185565b5b8235905067ffffffffffffffff8111156101c3576101c2610189565b5b6020830191508360018202830111156101df576101de61018d565b5b9250929050565b5f5f5f5f606085870312156101fe576101fd6100f0565b5b5f61020b8782880161013e565b945050602061021c87828801610171565b935050604085013567ffffffffffffffff81111561023d5761023c6100f4565b5b61024987828801610191565b925092505092959194509250565b5f81905092915050565b828183375f83830152505050565b5f61027a8385610257565b9350610287838584610261565b82840190509392505050565b5f61029f82848661026f565b91508190509392505050565b5f82825260208201905092915050565b7f63616c6c207265766572746564000000000000000000000000000000000000005f82015250565b5f6102ef600d836102ab565b91506102fa826102bb565b602082019050919050565b5f6020820190508181035f83015261031c816102e3565b905091905056fea2646970667358221220f0bf04f7f75064acb620b48b0935443698013efad4ab9d3a7524321e62811c4d64736f6c634300081d0033",
}

// SponsoringABI is the input ABI used to generate the binding from.
// Deprecated: Use SponsoringMetaData.ABI instead.
var SponsoringABI = SponsoringMetaData.ABI

// SponsoringBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SponsoringMetaData.Bin instead.
var SponsoringBin = SponsoringMetaData.Bin

// DeploySponsoring deploys a new Ethereum contract, binding an instance of Sponsoring to it.
func DeploySponsoring(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Sponsoring, error) {
	parsed, err := SponsoringMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SponsoringBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Sponsoring{SponsoringCaller: SponsoringCaller{contract: contract}, SponsoringTransactor: SponsoringTransactor{contract: contract}, SponsoringFilterer: SponsoringFilterer{contract: contract}}, nil
}

// Sponsoring is an auto generated Go binding around an Ethereum contract.
type Sponsoring struct {
	SponsoringCaller     // Read-only binding to the contract
	SponsoringTransactor // Write-only binding to the contract
	SponsoringFilterer   // Log filterer for contract events
}

// SponsoringCaller is an auto generated read-only Go binding around an Ethereum contract.
type SponsoringCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SponsoringTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SponsoringTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SponsoringFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SponsoringFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SponsoringSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SponsoringSession struct {
	Contract     *Sponsoring       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SponsoringCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SponsoringCallerSession struct {
	Contract *SponsoringCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// SponsoringTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SponsoringTransactorSession struct {
	Contract     *SponsoringTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// SponsoringRaw is an auto generated low-level Go binding around an Ethereum contract.
type SponsoringRaw struct {
	Contract *Sponsoring // Generic contract binding to access the raw methods on
}

// SponsoringCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SponsoringCallerRaw struct {
	Contract *SponsoringCaller // Generic read-only contract binding to access the raw methods on
}

// SponsoringTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SponsoringTransactorRaw struct {
	Contract *SponsoringTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSponsoring creates a new instance of Sponsoring, bound to a specific deployed contract.
func NewSponsoring(address common.Address, backend bind.ContractBackend) (*Sponsoring, error) {
	contract, err := bindSponsoring(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Sponsoring{SponsoringCaller: SponsoringCaller{contract: contract}, SponsoringTransactor: SponsoringTransactor{contract: contract}, SponsoringFilterer: SponsoringFilterer{contract: contract}}, nil
}

// NewSponsoringCaller creates a new read-only instance of Sponsoring, bound to a specific deployed contract.
func NewSponsoringCaller(address common.Address, caller bind.ContractCaller) (*SponsoringCaller, error) {
	contract, err := bindSponsoring(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SponsoringCaller{contract: contract}, nil
}

// NewSponsoringTransactor creates a new write-only instance of Sponsoring, bound to a specific deployed contract.
func NewSponsoringTransactor(address common.Address, transactor bind.ContractTransactor) (*SponsoringTransactor, error) {
	contract, err := bindSponsoring(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SponsoringTransactor{contract: contract}, nil
}

// NewSponsoringFilterer creates a new log filterer instance of Sponsoring, bound to a specific deployed contract.
func NewSponsoringFilterer(address common.Address, filterer bind.ContractFilterer) (*SponsoringFilterer, error) {
	contract, err := bindSponsoring(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SponsoringFilterer{contract: contract}, nil
}

// bindSponsoring binds a generic wrapper to an already deployed contract.
func bindSponsoring(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SponsoringMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Sponsoring *SponsoringRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Sponsoring.Contract.SponsoringCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Sponsoring *SponsoringRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Sponsoring.Contract.SponsoringTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Sponsoring *SponsoringRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Sponsoring.Contract.SponsoringTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Sponsoring *SponsoringCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Sponsoring.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Sponsoring *SponsoringTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Sponsoring.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Sponsoring *SponsoringTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Sponsoring.Contract.contract.Transact(opts, method, params...)
}

// Execute is a paid mutator transaction binding the contract method 0xb61d27f6.
//
// Solidity: function execute(address to, uint256 value, bytes data) payable returns()
func (_Sponsoring *SponsoringTransactor) Execute(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Sponsoring.contract.Transact(opts, "execute", to, value, data)
}

// Execute is a paid mutator transaction binding the contract method 0xb61d27f6.
//
// Solidity: function execute(address to, uint256 value, bytes data) payable returns()
func (_Sponsoring *SponsoringSession) Execute(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Sponsoring.Contract.Execute(&_Sponsoring.TransactOpts, to, value, data)
}

// Execute is a paid mutator transaction binding the contract method 0xb61d27f6.
//
// Solidity: function execute(address to, uint256 value, bytes data) payable returns()
func (_Sponsoring *SponsoringTransactorSession) Execute(to common.Address, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Sponsoring.Contract.Execute(&_Sponsoring.TransactOpts, to, value, data)
}
