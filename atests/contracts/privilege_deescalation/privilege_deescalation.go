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

package privilege_deescalation

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

// PrivilegeDeescalationMetaData contains all meta data concerning the PrivilegeDeescalation contract.
var PrivilegeDeescalationMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"allow_payment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"do_payment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506105608061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c806351d1ddc014610038578063b006fdc014610054575b5f5ffd5b610052600480360381019061004d91906102ec565b610070565b005b61006e6004803603810190610069919061032a565b6101ab565b005b3373ffffffffffffffffffffffffffffffffffffffff165f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16146100fe576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016100f5906103d5565b60405180910390fd5b5f8273ffffffffffffffffffffffffffffffffffffffff168260405161012390610420565b5f6040518083038185875af1925050503d805f811461015d576040519150601f19603f3d011682016040523d82523d5f602084013e610162565b606091505b50509050806101a6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161019d9061047e565b60405180910390fd5b505050565b3373ffffffffffffffffffffffffffffffffffffffff163073ffffffffffffffffffffffffffffffffffffffff1614610219576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102109061050c565b60405180910390fd5b805f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6102888261025f565b9050919050565b6102988161027e565b81146102a2575f5ffd5b50565b5f813590506102b38161028f565b92915050565b5f819050919050565b6102cb816102b9565b81146102d5575f5ffd5b50565b5f813590506102e6816102c2565b92915050565b5f5f604083850312156103025761030161025b565b5b5f61030f858286016102a5565b9250506020610320858286016102d8565b9150509250929050565b5f6020828403121561033f5761033e61025b565b5b5f61034c848285016102a5565b91505092915050565b5f82825260208201905092915050565b7f6f6e6c7920616c6c6f776564206164647265737365732063616e207472616e735f8201527f66657220666f756e647300000000000000000000000000000000000000000000602082015250565b5f6103bf602a83610355565b91506103ca82610365565b604082019050919050565b5f6020820190508181035f8301526103ec816103b3565b9050919050565b5f81905092915050565b50565b5f61040b5f836103f3565b9150610416826103fd565b5f82019050919050565b5f61042a82610400565b9150819050919050565b7f63616c6c207265766572746564000000000000000000000000000000000000005f82015250565b5f610468600d83610355565b915061047382610434565b602082019050919050565b5f6020820190508181035f8301526104958161045c565b9050919050565b7f6f6e6c7920746865206f776e206163636f756e742063616e206368616e6765205f8201527f616363657373206c697374000000000000000000000000000000000000000000602082015250565b5f6104f6602b83610355565b91506105018261049c565b604082019050919050565b5f6020820190508181035f830152610523816104ea565b905091905056fea2646970667358221220e56c2fa80cb67a4bf342b6f0bd0b9c5745467d9b7f9f0b7ae7c3085354134bb264736f6c634300081c0033",
}

// PrivilegeDeescalationABI is the input ABI used to generate the binding from.
// Deprecated: Use PrivilegeDeescalationMetaData.ABI instead.
var PrivilegeDeescalationABI = PrivilegeDeescalationMetaData.ABI

// PrivilegeDeescalationBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PrivilegeDeescalationMetaData.Bin instead.
var PrivilegeDeescalationBin = PrivilegeDeescalationMetaData.Bin

// DeployPrivilegeDeescalation deploys a new Ethereum contract, binding an instance of PrivilegeDeescalation to it.
func DeployPrivilegeDeescalation(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PrivilegeDeescalation, error) {
	parsed, err := PrivilegeDeescalationMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PrivilegeDeescalationBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PrivilegeDeescalation{PrivilegeDeescalationCaller: PrivilegeDeescalationCaller{contract: contract}, PrivilegeDeescalationTransactor: PrivilegeDeescalationTransactor{contract: contract}, PrivilegeDeescalationFilterer: PrivilegeDeescalationFilterer{contract: contract}}, nil
}

// PrivilegeDeescalation is an auto generated Go binding around an Ethereum contract.
type PrivilegeDeescalation struct {
	PrivilegeDeescalationCaller     // Read-only binding to the contract
	PrivilegeDeescalationTransactor // Write-only binding to the contract
	PrivilegeDeescalationFilterer   // Log filterer for contract events
}

// PrivilegeDeescalationCaller is an auto generated read-only Go binding around an Ethereum contract.
type PrivilegeDeescalationCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PrivilegeDeescalationTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PrivilegeDeescalationTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PrivilegeDeescalationFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PrivilegeDeescalationFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PrivilegeDeescalationSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PrivilegeDeescalationSession struct {
	Contract     *PrivilegeDeescalation // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// PrivilegeDeescalationCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PrivilegeDeescalationCallerSession struct {
	Contract *PrivilegeDeescalationCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// PrivilegeDeescalationTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PrivilegeDeescalationTransactorSession struct {
	Contract     *PrivilegeDeescalationTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// PrivilegeDeescalationRaw is an auto generated low-level Go binding around an Ethereum contract.
type PrivilegeDeescalationRaw struct {
	Contract *PrivilegeDeescalation // Generic contract binding to access the raw methods on
}

// PrivilegeDeescalationCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PrivilegeDeescalationCallerRaw struct {
	Contract *PrivilegeDeescalationCaller // Generic read-only contract binding to access the raw methods on
}

// PrivilegeDeescalationTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PrivilegeDeescalationTransactorRaw struct {
	Contract *PrivilegeDeescalationTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPrivilegeDeescalation creates a new instance of PrivilegeDeescalation, bound to a specific deployed contract.
func NewPrivilegeDeescalation(address common.Address, backend bind.ContractBackend) (*PrivilegeDeescalation, error) {
	contract, err := bindPrivilegeDeescalation(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PrivilegeDeescalation{PrivilegeDeescalationCaller: PrivilegeDeescalationCaller{contract: contract}, PrivilegeDeescalationTransactor: PrivilegeDeescalationTransactor{contract: contract}, PrivilegeDeescalationFilterer: PrivilegeDeescalationFilterer{contract: contract}}, nil
}

// NewPrivilegeDeescalationCaller creates a new read-only instance of PrivilegeDeescalation, bound to a specific deployed contract.
func NewPrivilegeDeescalationCaller(address common.Address, caller bind.ContractCaller) (*PrivilegeDeescalationCaller, error) {
	contract, err := bindPrivilegeDeescalation(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PrivilegeDeescalationCaller{contract: contract}, nil
}

// NewPrivilegeDeescalationTransactor creates a new write-only instance of PrivilegeDeescalation, bound to a specific deployed contract.
func NewPrivilegeDeescalationTransactor(address common.Address, transactor bind.ContractTransactor) (*PrivilegeDeescalationTransactor, error) {
	contract, err := bindPrivilegeDeescalation(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PrivilegeDeescalationTransactor{contract: contract}, nil
}

// NewPrivilegeDeescalationFilterer creates a new log filterer instance of PrivilegeDeescalation, bound to a specific deployed contract.
func NewPrivilegeDeescalationFilterer(address common.Address, filterer bind.ContractFilterer) (*PrivilegeDeescalationFilterer, error) {
	contract, err := bindPrivilegeDeescalation(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PrivilegeDeescalationFilterer{contract: contract}, nil
}

// bindPrivilegeDeescalation binds a generic wrapper to an already deployed contract.
func bindPrivilegeDeescalation(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PrivilegeDeescalationMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PrivilegeDeescalation *PrivilegeDeescalationRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PrivilegeDeescalation.Contract.PrivilegeDeescalationCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PrivilegeDeescalation *PrivilegeDeescalationRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.PrivilegeDeescalationTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PrivilegeDeescalation *PrivilegeDeescalationRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.PrivilegeDeescalationTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PrivilegeDeescalation *PrivilegeDeescalationCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PrivilegeDeescalation.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PrivilegeDeescalation *PrivilegeDeescalationTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PrivilegeDeescalation *PrivilegeDeescalationTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.contract.Transact(opts, method, params...)
}

// AllowPayment is a paid mutator transaction binding the contract method 0xb006fdc0.
//
// Solidity: function allow_payment(address account) returns()
func (_PrivilegeDeescalation *PrivilegeDeescalationTransactor) AllowPayment(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _PrivilegeDeescalation.contract.Transact(opts, "allow_payment", account)
}

// AllowPayment is a paid mutator transaction binding the contract method 0xb006fdc0.
//
// Solidity: function allow_payment(address account) returns()
func (_PrivilegeDeescalation *PrivilegeDeescalationSession) AllowPayment(account common.Address) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.AllowPayment(&_PrivilegeDeescalation.TransactOpts, account)
}

// AllowPayment is a paid mutator transaction binding the contract method 0xb006fdc0.
//
// Solidity: function allow_payment(address account) returns()
func (_PrivilegeDeescalation *PrivilegeDeescalationTransactorSession) AllowPayment(account common.Address) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.AllowPayment(&_PrivilegeDeescalation.TransactOpts, account)
}

// DoPayment is a paid mutator transaction binding the contract method 0x51d1ddc0.
//
// Solidity: function do_payment(address to, uint256 value) returns()
func (_PrivilegeDeescalation *PrivilegeDeescalationTransactor) DoPayment(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _PrivilegeDeescalation.contract.Transact(opts, "do_payment", to, value)
}

// DoPayment is a paid mutator transaction binding the contract method 0x51d1ddc0.
//
// Solidity: function do_payment(address to, uint256 value) returns()
func (_PrivilegeDeescalation *PrivilegeDeescalationSession) DoPayment(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.DoPayment(&_PrivilegeDeescalation.TransactOpts, to, value)
}

// DoPayment is a paid mutator transaction binding the contract method 0x51d1ddc0.
//
// Solidity: function do_payment(address to, uint256 value) returns()
func (_PrivilegeDeescalation *PrivilegeDeescalationTransactorSession) DoPayment(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _PrivilegeDeescalation.Contract.DoPayment(&_PrivilegeDeescalation.TransactOpts, to, value)
}
