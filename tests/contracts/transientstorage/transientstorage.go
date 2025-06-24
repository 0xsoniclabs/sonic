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

package transientstorage

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

// TransientstorageMetaData contains all meta data concerning the Transientstorage contract.
var TransientstorageMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"StoredValue\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"storeValue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506040516019906074565b604051809103905ff0801580156031573d5f5f3e3d5ffd5b505f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506081565b6101458061036383390190565b6102d58061008e5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063209652551461003857806381bff17014610056575b5f5ffd5b610040610060565b60405161004d91906101d2565b60405180910390f35b61005e6100f3565b005b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a4b407786040518163ffffffff1660e01b8152600401602060405180830381865afa1580156100ca573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100ee9190610219565b905090565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166386fa8f14602a6040518263ffffffff1660e01b815260040161014d9190610286565b5f604051808303815f87803b158015610164575f5ffd5b505af1158015610176573d5f5f3e3d5ffd5b505050507f4df99912c217e455d257a5b4b2624508c42e58f5ebdd62d7cf2a6e18c0c327d26101a3610060565b6040516101b091906101d2565b60405180910390a1565b5f819050919050565b6101cc816101ba565b82525050565b5f6020820190506101e55f8301846101c3565b92915050565b5f5ffd5b6101f8816101ba565b8114610202575f5ffd5b50565b5f81519050610213816101ef565b92915050565b5f6020828403121561022e5761022d6101eb565b5b5f61023b84828501610205565b91505092915050565b5f819050919050565b5f819050919050565b5f61027061026b61026684610244565b61024d565b6101ba565b9050919050565b61028081610256565b82525050565b5f6020820190506102995f830184610277565b9291505056fea2646970667358221220aa81cdf4c34a9e324b01aa5782deafb488d504fea7d4ec4f29dc4bed4f27a52264736f6c634300081c00336080604052348015600e575f5ffd5b506101298061001c5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c806386fa8f14146034578063a4b4077814604c575b5f5ffd5b604a60048036038101906046919060a9565b6066565b005b6052606f565b604051605d919060dc565b60405180910390f35b805f81905d5050565b5f5f5c905090565b5f5ffd5b5f819050919050565b608b81607b565b81146094575f5ffd5b50565b5f8135905060a3816084565b92915050565b5f6020828403121560bb5760ba6077565b5b5f60c6848285016097565b91505092915050565b60d681607b565b82525050565b5f60208201905060ed5f83018460cf565b9291505056fea2646970667358221220064c09f76e3e80e6de159107b2a15f7642e5df3f5781aa00348b8c44bbfcca0864736f6c634300081c0033",
}

// TransientstorageABI is the input ABI used to generate the binding from.
// Deprecated: Use TransientstorageMetaData.ABI instead.
var TransientstorageABI = TransientstorageMetaData.ABI

// TransientstorageBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TransientstorageMetaData.Bin instead.
var TransientstorageBin = TransientstorageMetaData.Bin

// DeployTransientstorage deploys a new Ethereum contract, binding an instance of Transientstorage to it.
func DeployTransientstorage(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Transientstorage, error) {
	parsed, err := TransientstorageMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TransientstorageBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Transientstorage{TransientstorageCaller: TransientstorageCaller{contract: contract}, TransientstorageTransactor: TransientstorageTransactor{contract: contract}, TransientstorageFilterer: TransientstorageFilterer{contract: contract}}, nil
}

// Transientstorage is an auto generated Go binding around an Ethereum contract.
type Transientstorage struct {
	TransientstorageCaller     // Read-only binding to the contract
	TransientstorageTransactor // Write-only binding to the contract
	TransientstorageFilterer   // Log filterer for contract events
}

// TransientstorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type TransientstorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransientstorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TransientstorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransientstorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TransientstorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransientstorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TransientstorageSession struct {
	Contract     *Transientstorage // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TransientstorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TransientstorageCallerSession struct {
	Contract *TransientstorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// TransientstorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TransientstorageTransactorSession struct {
	Contract     *TransientstorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// TransientstorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type TransientstorageRaw struct {
	Contract *Transientstorage // Generic contract binding to access the raw methods on
}

// TransientstorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TransientstorageCallerRaw struct {
	Contract *TransientstorageCaller // Generic read-only contract binding to access the raw methods on
}

// TransientstorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TransientstorageTransactorRaw struct {
	Contract *TransientstorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTransientstorage creates a new instance of Transientstorage, bound to a specific deployed contract.
func NewTransientstorage(address common.Address, backend bind.ContractBackend) (*Transientstorage, error) {
	contract, err := bindTransientstorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Transientstorage{TransientstorageCaller: TransientstorageCaller{contract: contract}, TransientstorageTransactor: TransientstorageTransactor{contract: contract}, TransientstorageFilterer: TransientstorageFilterer{contract: contract}}, nil
}

// NewTransientstorageCaller creates a new read-only instance of Transientstorage, bound to a specific deployed contract.
func NewTransientstorageCaller(address common.Address, caller bind.ContractCaller) (*TransientstorageCaller, error) {
	contract, err := bindTransientstorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TransientstorageCaller{contract: contract}, nil
}

// NewTransientstorageTransactor creates a new write-only instance of Transientstorage, bound to a specific deployed contract.
func NewTransientstorageTransactor(address common.Address, transactor bind.ContractTransactor) (*TransientstorageTransactor, error) {
	contract, err := bindTransientstorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TransientstorageTransactor{contract: contract}, nil
}

// NewTransientstorageFilterer creates a new log filterer instance of Transientstorage, bound to a specific deployed contract.
func NewTransientstorageFilterer(address common.Address, filterer bind.ContractFilterer) (*TransientstorageFilterer, error) {
	contract, err := bindTransientstorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TransientstorageFilterer{contract: contract}, nil
}

// bindTransientstorage binds a generic wrapper to an already deployed contract.
func bindTransientstorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TransientstorageMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Transientstorage *TransientstorageRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Transientstorage.Contract.TransientstorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Transientstorage *TransientstorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Transientstorage.Contract.TransientstorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Transientstorage *TransientstorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Transientstorage.Contract.TransientstorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Transientstorage *TransientstorageCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Transientstorage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Transientstorage *TransientstorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Transientstorage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Transientstorage *TransientstorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Transientstorage.Contract.contract.Transact(opts, method, params...)
}

// GetValue is a free data retrieval call binding the contract method 0x20965255.
//
// Solidity: function getValue() view returns(uint256)
func (_Transientstorage *TransientstorageCaller) GetValue(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Transientstorage.contract.Call(opts, &out, "getValue")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetValue is a free data retrieval call binding the contract method 0x20965255.
//
// Solidity: function getValue() view returns(uint256)
func (_Transientstorage *TransientstorageSession) GetValue() (*big.Int, error) {
	return _Transientstorage.Contract.GetValue(&_Transientstorage.CallOpts)
}

// GetValue is a free data retrieval call binding the contract method 0x20965255.
//
// Solidity: function getValue() view returns(uint256)
func (_Transientstorage *TransientstorageCallerSession) GetValue() (*big.Int, error) {
	return _Transientstorage.Contract.GetValue(&_Transientstorage.CallOpts)
}

// StoreValue is a paid mutator transaction binding the contract method 0x81bff170.
//
// Solidity: function storeValue() returns()
func (_Transientstorage *TransientstorageTransactor) StoreValue(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Transientstorage.contract.Transact(opts, "storeValue")
}

// StoreValue is a paid mutator transaction binding the contract method 0x81bff170.
//
// Solidity: function storeValue() returns()
func (_Transientstorage *TransientstorageSession) StoreValue() (*types.Transaction, error) {
	return _Transientstorage.Contract.StoreValue(&_Transientstorage.TransactOpts)
}

// StoreValue is a paid mutator transaction binding the contract method 0x81bff170.
//
// Solidity: function storeValue() returns()
func (_Transientstorage *TransientstorageTransactorSession) StoreValue() (*types.Transaction, error) {
	return _Transientstorage.Contract.StoreValue(&_Transientstorage.TransactOpts)
}

// TransientstorageStoredValueIterator is returned from FilterStoredValue and is used to iterate over the raw logs and unpacked data for StoredValue events raised by the Transientstorage contract.
type TransientstorageStoredValueIterator struct {
	Event *TransientstorageStoredValue // Event containing the contract specifics and raw log

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
func (it *TransientstorageStoredValueIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TransientstorageStoredValue)
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
		it.Event = new(TransientstorageStoredValue)
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
func (it *TransientstorageStoredValueIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TransientstorageStoredValueIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TransientstorageStoredValue represents a StoredValue event raised by the Transientstorage contract.
type TransientstorageStoredValue struct {
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterStoredValue is a free log retrieval operation binding the contract event 0x4df99912c217e455d257a5b4b2624508c42e58f5ebdd62d7cf2a6e18c0c327d2.
//
// Solidity: event StoredValue(uint256 value)
func (_Transientstorage *TransientstorageFilterer) FilterStoredValue(opts *bind.FilterOpts) (*TransientstorageStoredValueIterator, error) {

	logs, sub, err := _Transientstorage.contract.FilterLogs(opts, "StoredValue")
	if err != nil {
		return nil, err
	}
	return &TransientstorageStoredValueIterator{contract: _Transientstorage.contract, event: "StoredValue", logs: logs, sub: sub}, nil
}

// WatchStoredValue is a free log subscription operation binding the contract event 0x4df99912c217e455d257a5b4b2624508c42e58f5ebdd62d7cf2a6e18c0c327d2.
//
// Solidity: event StoredValue(uint256 value)
func (_Transientstorage *TransientstorageFilterer) WatchStoredValue(opts *bind.WatchOpts, sink chan<- *TransientstorageStoredValue) (event.Subscription, error) {

	logs, sub, err := _Transientstorage.contract.WatchLogs(opts, "StoredValue")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TransientstorageStoredValue)
				if err := _Transientstorage.contract.UnpackLog(event, "StoredValue", log); err != nil {
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

// ParseStoredValue is a log parse operation binding the contract event 0x4df99912c217e455d257a5b4b2624508c42e58f5ebdd62d7cf2a6e18c0c327d2.
//
// Solidity: event StoredValue(uint256 value)
func (_Transientstorage *TransientstorageFilterer) ParseStoredValue(log types.Log) (*TransientstorageStoredValue, error) {
	event := new(TransientstorageStoredValue)
	if err := _Transientstorage.contract.UnpackLog(event, "StoredValue", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
