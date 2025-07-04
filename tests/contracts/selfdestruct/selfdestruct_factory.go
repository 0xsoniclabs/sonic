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

package selfdestruct

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

// SelfDestructFactoryMetaData contains all meta data concerning the SelfDestructFactory contract.
var SelfDestructFactoryMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"LogContractStorage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"LogDeployed\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"create\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"beneficiary\",\"type\":\"address\"}],\"name\":\"createAndDestroy\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createAndDestroyWithoutBeneficiary\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"beneficiary\",\"type\":\"address\"}],\"name\":\"destroy\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"destroyWithoutBeneficiary\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506109a98061001c5f395ff3fe608060405260043610610048575f3560e01c8062f55d9d1461004c5780635244e834146100745780637d3cf16a14610090578063aaff233b146100a6578063efc81a8c146100b0575b5f5ffd5b348015610057575f5ffd5b50610072600480360381019061006d91906104d7565b6100ba565b005b61008e600480360381019061008991906104d7565b610145565b005b34801561009b575f5ffd5b506100a461021c565b005b6100ae6102a7565b005b6100b861039d565b005b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663ca29aee95f836040518363ffffffff1660e01b815260040161011592919061052b565b5f604051808303815f87803b15801561012c575f5ffd5b505af115801561013e573d5f5f3e3d5ffd5b5050505050565b61014d61039d565b610156816100ba565b7f3be3676a60e58335e9af2a05f4c6208663920cd9e2cd3900745377049fa721f05f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663c73b71756040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101e0573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906102049190610585565b60405161021191906105bf565b60405180910390a150565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663ca29aee960015f6040518363ffffffff1660e01b815260040161027892919061052b565b5f604051808303815f87803b15801561028f575f5ffd5b505af11580156102a1573d5f5f3e3d5ffd5b50505050565b6102af61039d565b6102d85f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff166100ba565b7f3be3676a60e58335e9af2a05f4c6208663920cd9e2cd3900745377049fa721f05f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663c73b71756040518163ffffffff1660e01b8152600401602060405180830381865afa158015610362573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103869190610585565b60405161039391906105bf565b60405180910390a1565b345f5f5f6040516103ad9061046c565b6103b9939291906105d8565b6040518091039082f09050801580156103d4573d5f5f3e3d5ffd5b505f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055507f4a0f3be106bcb1b76a2cf127e86da8ceb3dba6b2d1bb19ec8ffbfa90c4647b515f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff16604051610462919061062d565b60405180910390a1565b61032d8061064783390190565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6104a68261047d565b9050919050565b6104b68161049c565b81146104c0575f5ffd5b50565b5f813590506104d1816104ad565b92915050565b5f602082840312156104ec576104eb610479565b5b5f6104f9848285016104c3565b91505092915050565b5f8115159050919050565b61051681610502565b82525050565b6105258161049c565b82525050565b5f60408201905061053e5f83018561050d565b61054b602083018461051c565b9392505050565b5f819050919050565b61056481610552565b811461056e575f5ffd5b50565b5f8151905061057f8161055b565b92915050565b5f6020828403121561059a57610599610479565b5b5f6105a784828501610571565b91505092915050565b6105b981610552565b82525050565b5f6020820190506105d25f8301846105b0565b92915050565b5f6060820190506105eb5f83018661050d565b6105f8602083018561050d565b610605604083018461051c565b949350505050565b5f6106178261047d565b9050919050565b6106278161060d565b82525050565b5f6020820190506106405f83018461061e565b9291505056fe6080604052607b5f5560405161032d38038061032d833981810160405281019061002991906100fe565b82156100405761003f828261004860201b60201c565b5b50505061014e565b8115610052573090505b8073ffffffffffffffffffffffffffffffffffffffff16ff5b5f5ffd5b5f8115159050919050565b6100838161006f565b811461008d575f5ffd5b50565b5f8151905061009e8161007a565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6100cd826100a4565b9050919050565b6100dd816100c3565b81146100e7575f5ffd5b50565b5f815190506100f8816100d4565b92915050565b5f5f5f606084860312156101155761011461006b565b5b5f61012286828701610090565b935050602061013386828701610090565b9250506040610144868287016100ea565b9150509250925092565b6101d28061015b5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063c73b717514610038578063ca29aee914610056575b5f5ffd5b610040610072565b60405161004d91906100b2565b60405180910390f35b610070600480360381019061006b919061015e565b610077565b005b5f5481565b8115610081573090505b8073ffffffffffffffffffffffffffffffffffffffff16ff5b5f819050919050565b6100ac8161009a565b82525050565b5f6020820190506100c55f8301846100a3565b92915050565b5f5ffd5b5f8115159050919050565b6100e3816100cf565b81146100ed575f5ffd5b50565b5f813590506100fe816100da565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61012d82610104565b9050919050565b61013d81610123565b8114610147575f5ffd5b50565b5f8135905061015881610134565b92915050565b5f5f60408385031215610174576101736100cb565b5b5f610181858286016100f0565b92505060206101928582860161014a565b915050925092905056fea264697066735822122081a84691ce3c2c7d21e21307637fdd8eda51775204ab8b5fa5fd60d18c247ff764736f6c634300081c0033a2646970667358221220dfb0f6ca72eff5312774b7ac0a817ae8af925e31da21232c5884ada3039159a464736f6c634300081c0033",
}

// SelfDestructFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use SelfDestructFactoryMetaData.ABI instead.
var SelfDestructFactoryABI = SelfDestructFactoryMetaData.ABI

// SelfDestructFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SelfDestructFactoryMetaData.Bin instead.
var SelfDestructFactoryBin = SelfDestructFactoryMetaData.Bin

// DeploySelfDestructFactory deploys a new Ethereum contract, binding an instance of SelfDestructFactory to it.
func DeploySelfDestructFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SelfDestructFactory, error) {
	parsed, err := SelfDestructFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SelfDestructFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SelfDestructFactory{SelfDestructFactoryCaller: SelfDestructFactoryCaller{contract: contract}, SelfDestructFactoryTransactor: SelfDestructFactoryTransactor{contract: contract}, SelfDestructFactoryFilterer: SelfDestructFactoryFilterer{contract: contract}}, nil
}

// SelfDestructFactory is an auto generated Go binding around an Ethereum contract.
type SelfDestructFactory struct {
	SelfDestructFactoryCaller     // Read-only binding to the contract
	SelfDestructFactoryTransactor // Write-only binding to the contract
	SelfDestructFactoryFilterer   // Log filterer for contract events
}

// SelfDestructFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type SelfDestructFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfDestructFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SelfDestructFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfDestructFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SelfDestructFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfDestructFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SelfDestructFactorySession struct {
	Contract     *SelfDestructFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// SelfDestructFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SelfDestructFactoryCallerSession struct {
	Contract *SelfDestructFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// SelfDestructFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SelfDestructFactoryTransactorSession struct {
	Contract     *SelfDestructFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// SelfDestructFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type SelfDestructFactoryRaw struct {
	Contract *SelfDestructFactory // Generic contract binding to access the raw methods on
}

// SelfDestructFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SelfDestructFactoryCallerRaw struct {
	Contract *SelfDestructFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// SelfDestructFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SelfDestructFactoryTransactorRaw struct {
	Contract *SelfDestructFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSelfDestructFactory creates a new instance of SelfDestructFactory, bound to a specific deployed contract.
func NewSelfDestructFactory(address common.Address, backend bind.ContractBackend) (*SelfDestructFactory, error) {
	contract, err := bindSelfDestructFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SelfDestructFactory{SelfDestructFactoryCaller: SelfDestructFactoryCaller{contract: contract}, SelfDestructFactoryTransactor: SelfDestructFactoryTransactor{contract: contract}, SelfDestructFactoryFilterer: SelfDestructFactoryFilterer{contract: contract}}, nil
}

// NewSelfDestructFactoryCaller creates a new read-only instance of SelfDestructFactory, bound to a specific deployed contract.
func NewSelfDestructFactoryCaller(address common.Address, caller bind.ContractCaller) (*SelfDestructFactoryCaller, error) {
	contract, err := bindSelfDestructFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SelfDestructFactoryCaller{contract: contract}, nil
}

// NewSelfDestructFactoryTransactor creates a new write-only instance of SelfDestructFactory, bound to a specific deployed contract.
func NewSelfDestructFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*SelfDestructFactoryTransactor, error) {
	contract, err := bindSelfDestructFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SelfDestructFactoryTransactor{contract: contract}, nil
}

// NewSelfDestructFactoryFilterer creates a new log filterer instance of SelfDestructFactory, bound to a specific deployed contract.
func NewSelfDestructFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*SelfDestructFactoryFilterer, error) {
	contract, err := bindSelfDestructFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SelfDestructFactoryFilterer{contract: contract}, nil
}

// bindSelfDestructFactory binds a generic wrapper to an already deployed contract.
func bindSelfDestructFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SelfDestructFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SelfDestructFactory *SelfDestructFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SelfDestructFactory.Contract.SelfDestructFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SelfDestructFactory *SelfDestructFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.SelfDestructFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SelfDestructFactory *SelfDestructFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.SelfDestructFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SelfDestructFactory *SelfDestructFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SelfDestructFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SelfDestructFactory *SelfDestructFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SelfDestructFactory *SelfDestructFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.contract.Transact(opts, method, params...)
}

// Create is a paid mutator transaction binding the contract method 0xefc81a8c.
//
// Solidity: function create() payable returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactor) Create(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestructFactory.contract.Transact(opts, "create")
}

// Create is a paid mutator transaction binding the contract method 0xefc81a8c.
//
// Solidity: function create() payable returns()
func (_SelfDestructFactory *SelfDestructFactorySession) Create() (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.Create(&_SelfDestructFactory.TransactOpts)
}

// Create is a paid mutator transaction binding the contract method 0xefc81a8c.
//
// Solidity: function create() payable returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactorSession) Create() (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.Create(&_SelfDestructFactory.TransactOpts)
}

// CreateAndDestroy is a paid mutator transaction binding the contract method 0x5244e834.
//
// Solidity: function createAndDestroy(address beneficiary) payable returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactor) CreateAndDestroy(opts *bind.TransactOpts, beneficiary common.Address) (*types.Transaction, error) {
	return _SelfDestructFactory.contract.Transact(opts, "createAndDestroy", beneficiary)
}

// CreateAndDestroy is a paid mutator transaction binding the contract method 0x5244e834.
//
// Solidity: function createAndDestroy(address beneficiary) payable returns()
func (_SelfDestructFactory *SelfDestructFactorySession) CreateAndDestroy(beneficiary common.Address) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.CreateAndDestroy(&_SelfDestructFactory.TransactOpts, beneficiary)
}

// CreateAndDestroy is a paid mutator transaction binding the contract method 0x5244e834.
//
// Solidity: function createAndDestroy(address beneficiary) payable returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactorSession) CreateAndDestroy(beneficiary common.Address) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.CreateAndDestroy(&_SelfDestructFactory.TransactOpts, beneficiary)
}

// CreateAndDestroyWithoutBeneficiary is a paid mutator transaction binding the contract method 0xaaff233b.
//
// Solidity: function createAndDestroyWithoutBeneficiary() payable returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactor) CreateAndDestroyWithoutBeneficiary(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestructFactory.contract.Transact(opts, "createAndDestroyWithoutBeneficiary")
}

// CreateAndDestroyWithoutBeneficiary is a paid mutator transaction binding the contract method 0xaaff233b.
//
// Solidity: function createAndDestroyWithoutBeneficiary() payable returns()
func (_SelfDestructFactory *SelfDestructFactorySession) CreateAndDestroyWithoutBeneficiary() (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.CreateAndDestroyWithoutBeneficiary(&_SelfDestructFactory.TransactOpts)
}

// CreateAndDestroyWithoutBeneficiary is a paid mutator transaction binding the contract method 0xaaff233b.
//
// Solidity: function createAndDestroyWithoutBeneficiary() payable returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactorSession) CreateAndDestroyWithoutBeneficiary() (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.CreateAndDestroyWithoutBeneficiary(&_SelfDestructFactory.TransactOpts)
}

// Destroy is a paid mutator transaction binding the contract method 0x00f55d9d.
//
// Solidity: function destroy(address beneficiary) returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactor) Destroy(opts *bind.TransactOpts, beneficiary common.Address) (*types.Transaction, error) {
	return _SelfDestructFactory.contract.Transact(opts, "destroy", beneficiary)
}

// Destroy is a paid mutator transaction binding the contract method 0x00f55d9d.
//
// Solidity: function destroy(address beneficiary) returns()
func (_SelfDestructFactory *SelfDestructFactorySession) Destroy(beneficiary common.Address) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.Destroy(&_SelfDestructFactory.TransactOpts, beneficiary)
}

// Destroy is a paid mutator transaction binding the contract method 0x00f55d9d.
//
// Solidity: function destroy(address beneficiary) returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactorSession) Destroy(beneficiary common.Address) (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.Destroy(&_SelfDestructFactory.TransactOpts, beneficiary)
}

// DestroyWithoutBeneficiary is a paid mutator transaction binding the contract method 0x7d3cf16a.
//
// Solidity: function destroyWithoutBeneficiary() returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactor) DestroyWithoutBeneficiary(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestructFactory.contract.Transact(opts, "destroyWithoutBeneficiary")
}

// DestroyWithoutBeneficiary is a paid mutator transaction binding the contract method 0x7d3cf16a.
//
// Solidity: function destroyWithoutBeneficiary() returns()
func (_SelfDestructFactory *SelfDestructFactorySession) DestroyWithoutBeneficiary() (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.DestroyWithoutBeneficiary(&_SelfDestructFactory.TransactOpts)
}

// DestroyWithoutBeneficiary is a paid mutator transaction binding the contract method 0x7d3cf16a.
//
// Solidity: function destroyWithoutBeneficiary() returns()
func (_SelfDestructFactory *SelfDestructFactoryTransactorSession) DestroyWithoutBeneficiary() (*types.Transaction, error) {
	return _SelfDestructFactory.Contract.DestroyWithoutBeneficiary(&_SelfDestructFactory.TransactOpts)
}

// SelfDestructFactoryLogContractStorageIterator is returned from FilterLogContractStorage and is used to iterate over the raw logs and unpacked data for LogContractStorage events raised by the SelfDestructFactory contract.
type SelfDestructFactoryLogContractStorageIterator struct {
	Event *SelfDestructFactoryLogContractStorage // Event containing the contract specifics and raw log

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
func (it *SelfDestructFactoryLogContractStorageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SelfDestructFactoryLogContractStorage)
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
		it.Event = new(SelfDestructFactoryLogContractStorage)
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
func (it *SelfDestructFactoryLogContractStorageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SelfDestructFactoryLogContractStorageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SelfDestructFactoryLogContractStorage represents a LogContractStorage event raised by the SelfDestructFactory contract.
type SelfDestructFactoryLogContractStorage struct {
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterLogContractStorage is a free log retrieval operation binding the contract event 0x3be3676a60e58335e9af2a05f4c6208663920cd9e2cd3900745377049fa721f0.
//
// Solidity: event LogContractStorage(uint256 value)
func (_SelfDestructFactory *SelfDestructFactoryFilterer) FilterLogContractStorage(opts *bind.FilterOpts) (*SelfDestructFactoryLogContractStorageIterator, error) {

	logs, sub, err := _SelfDestructFactory.contract.FilterLogs(opts, "LogContractStorage")
	if err != nil {
		return nil, err
	}
	return &SelfDestructFactoryLogContractStorageIterator{contract: _SelfDestructFactory.contract, event: "LogContractStorage", logs: logs, sub: sub}, nil
}

// WatchLogContractStorage is a free log subscription operation binding the contract event 0x3be3676a60e58335e9af2a05f4c6208663920cd9e2cd3900745377049fa721f0.
//
// Solidity: event LogContractStorage(uint256 value)
func (_SelfDestructFactory *SelfDestructFactoryFilterer) WatchLogContractStorage(opts *bind.WatchOpts, sink chan<- *SelfDestructFactoryLogContractStorage) (event.Subscription, error) {

	logs, sub, err := _SelfDestructFactory.contract.WatchLogs(opts, "LogContractStorage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SelfDestructFactoryLogContractStorage)
				if err := _SelfDestructFactory.contract.UnpackLog(event, "LogContractStorage", log); err != nil {
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

// ParseLogContractStorage is a log parse operation binding the contract event 0x3be3676a60e58335e9af2a05f4c6208663920cd9e2cd3900745377049fa721f0.
//
// Solidity: event LogContractStorage(uint256 value)
func (_SelfDestructFactory *SelfDestructFactoryFilterer) ParseLogContractStorage(log types.Log) (*SelfDestructFactoryLogContractStorage, error) {
	event := new(SelfDestructFactoryLogContractStorage)
	if err := _SelfDestructFactory.contract.UnpackLog(event, "LogContractStorage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SelfDestructFactoryLogDeployedIterator is returned from FilterLogDeployed and is used to iterate over the raw logs and unpacked data for LogDeployed events raised by the SelfDestructFactory contract.
type SelfDestructFactoryLogDeployedIterator struct {
	Event *SelfDestructFactoryLogDeployed // Event containing the contract specifics and raw log

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
func (it *SelfDestructFactoryLogDeployedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SelfDestructFactoryLogDeployed)
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
		it.Event = new(SelfDestructFactoryLogDeployed)
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
func (it *SelfDestructFactoryLogDeployedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SelfDestructFactoryLogDeployedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SelfDestructFactoryLogDeployed represents a LogDeployed event raised by the SelfDestructFactory contract.
type SelfDestructFactoryLogDeployed struct {
	Addr common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterLogDeployed is a free log retrieval operation binding the contract event 0x4a0f3be106bcb1b76a2cf127e86da8ceb3dba6b2d1bb19ec8ffbfa90c4647b51.
//
// Solidity: event LogDeployed(address addr)
func (_SelfDestructFactory *SelfDestructFactoryFilterer) FilterLogDeployed(opts *bind.FilterOpts) (*SelfDestructFactoryLogDeployedIterator, error) {

	logs, sub, err := _SelfDestructFactory.contract.FilterLogs(opts, "LogDeployed")
	if err != nil {
		return nil, err
	}
	return &SelfDestructFactoryLogDeployedIterator{contract: _SelfDestructFactory.contract, event: "LogDeployed", logs: logs, sub: sub}, nil
}

// WatchLogDeployed is a free log subscription operation binding the contract event 0x4a0f3be106bcb1b76a2cf127e86da8ceb3dba6b2d1bb19ec8ffbfa90c4647b51.
//
// Solidity: event LogDeployed(address addr)
func (_SelfDestructFactory *SelfDestructFactoryFilterer) WatchLogDeployed(opts *bind.WatchOpts, sink chan<- *SelfDestructFactoryLogDeployed) (event.Subscription, error) {

	logs, sub, err := _SelfDestructFactory.contract.WatchLogs(opts, "LogDeployed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SelfDestructFactoryLogDeployed)
				if err := _SelfDestructFactory.contract.UnpackLog(event, "LogDeployed", log); err != nil {
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

// ParseLogDeployed is a log parse operation binding the contract event 0x4a0f3be106bcb1b76a2cf127e86da8ceb3dba6b2d1bb19ec8ffbfa90c4647b51.
//
// Solidity: event LogDeployed(address addr)
func (_SelfDestructFactory *SelfDestructFactoryFilterer) ParseLogDeployed(log types.Log) (*SelfDestructFactoryLogDeployed, error) {
	event := new(SelfDestructFactoryLogDeployed)
	if err := _SelfDestructFactory.contract.UnpackLog(event, "LogDeployed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
