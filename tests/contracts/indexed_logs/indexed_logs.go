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

package indexed_logs

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

// IndexedLogsMetaData contains all meta data concerning the IndexedLogs contract.
var IndexedLogsMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"Event1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"Event2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"text\",\"type\":\"string\"}],\"name\":\"Event3\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"emitEvents\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5061020d8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c80636c8893d31461002d575b5f5ffd5b610035610037565b005b5f5f90505b60058110156100f7577f04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3816040516100749190610112565b60405180910390a17f06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e816040516100ab9190610112565b60405180910390a17f93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628816040516100e29190610185565b60405180910390a1808060010191505061003c565b50565b5f819050919050565b61010c816100fa565b82525050565b5f6020820190506101255f830184610103565b92915050565b5f82825260208201905092915050565b7f7465737420737472696e670000000000000000000000000000000000000000005f82015250565b5f61016f600b8361012b565b915061017a8261013b565b602082019050919050565b5f6040820190506101985f830184610103565b81810360208301526101a981610163565b90509291505056fea2646970667358221220a2c9ad1ac0259afe5651748310c49b90766f2632414598a3fd4694f9e183f55f64736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
}

// IndexedLogsABI is the input ABI used to generate the binding from.
// Deprecated: Use IndexedLogsMetaData.ABI instead.
var IndexedLogsABI = IndexedLogsMetaData.ABI

// IndexedLogsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use IndexedLogsMetaData.Bin instead.
var IndexedLogsBin = IndexedLogsMetaData.Bin

// DeployIndexedLogs deploys a new Ethereum contract, binding an instance of IndexedLogs to it.
func DeployIndexedLogs(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *IndexedLogs, error) {
	parsed, err := IndexedLogsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(IndexedLogsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &IndexedLogs{IndexedLogsCaller: IndexedLogsCaller{contract: contract}, IndexedLogsTransactor: IndexedLogsTransactor{contract: contract}, IndexedLogsFilterer: IndexedLogsFilterer{contract: contract}}, nil
}

// IndexedLogs is an auto generated Go binding around an Ethereum contract.
type IndexedLogs struct {
	IndexedLogsCaller     // Read-only binding to the contract
	IndexedLogsTransactor // Write-only binding to the contract
	IndexedLogsFilterer   // Log filterer for contract events
}

// IndexedLogsCaller is an auto generated read-only Go binding around an Ethereum contract.
type IndexedLogsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IndexedLogsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IndexedLogsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IndexedLogsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IndexedLogsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IndexedLogsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IndexedLogsSession struct {
	Contract     *IndexedLogs      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IndexedLogsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IndexedLogsCallerSession struct {
	Contract *IndexedLogsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// IndexedLogsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IndexedLogsTransactorSession struct {
	Contract     *IndexedLogsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IndexedLogsRaw is an auto generated low-level Go binding around an Ethereum contract.
type IndexedLogsRaw struct {
	Contract *IndexedLogs // Generic contract binding to access the raw methods on
}

// IndexedLogsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IndexedLogsCallerRaw struct {
	Contract *IndexedLogsCaller // Generic read-only contract binding to access the raw methods on
}

// IndexedLogsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IndexedLogsTransactorRaw struct {
	Contract *IndexedLogsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIndexedLogs creates a new instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogs(address common.Address, backend bind.ContractBackend) (*IndexedLogs, error) {
	contract, err := bindIndexedLogs(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IndexedLogs{IndexedLogsCaller: IndexedLogsCaller{contract: contract}, IndexedLogsTransactor: IndexedLogsTransactor{contract: contract}, IndexedLogsFilterer: IndexedLogsFilterer{contract: contract}}, nil
}

// NewIndexedLogsCaller creates a new read-only instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogsCaller(address common.Address, caller bind.ContractCaller) (*IndexedLogsCaller, error) {
	contract, err := bindIndexedLogs(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsCaller{contract: contract}, nil
}

// NewIndexedLogsTransactor creates a new write-only instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogsTransactor(address common.Address, transactor bind.ContractTransactor) (*IndexedLogsTransactor, error) {
	contract, err := bindIndexedLogs(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsTransactor{contract: contract}, nil
}

// NewIndexedLogsFilterer creates a new log filterer instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogsFilterer(address common.Address, filterer bind.ContractFilterer) (*IndexedLogsFilterer, error) {
	contract, err := bindIndexedLogs(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsFilterer{contract: contract}, nil
}

// bindIndexedLogs binds a generic wrapper to an already deployed contract.
func bindIndexedLogs(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IndexedLogsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IndexedLogs *IndexedLogsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IndexedLogs.Contract.IndexedLogsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IndexedLogs *IndexedLogsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IndexedLogs.Contract.IndexedLogsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IndexedLogs *IndexedLogsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IndexedLogs.Contract.IndexedLogsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IndexedLogs *IndexedLogsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IndexedLogs.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IndexedLogs *IndexedLogsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IndexedLogs.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IndexedLogs *IndexedLogsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IndexedLogs.Contract.contract.Transact(opts, method, params...)
}

// EmitEvents is a paid mutator transaction binding the contract method 0x6c8893d3.
//
// Solidity: function emitEvents() returns()
func (_IndexedLogs *IndexedLogsTransactor) EmitEvents(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IndexedLogs.contract.Transact(opts, "emitEvents")
}

// EmitEvents is a paid mutator transaction binding the contract method 0x6c8893d3.
//
// Solidity: function emitEvents() returns()
func (_IndexedLogs *IndexedLogsSession) EmitEvents() (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitEvents(&_IndexedLogs.TransactOpts)
}

// EmitEvents is a paid mutator transaction binding the contract method 0x6c8893d3.
//
// Solidity: function emitEvents() returns()
func (_IndexedLogs *IndexedLogsTransactorSession) EmitEvents() (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitEvents(&_IndexedLogs.TransactOpts)
}

// IndexedLogsEvent1Iterator is returned from FilterEvent1 and is used to iterate over the raw logs and unpacked data for Event1 events raised by the IndexedLogs contract.
type IndexedLogsEvent1Iterator struct {
	Event *IndexedLogsEvent1 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsEvent1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsEvent1)
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
		it.Event = new(IndexedLogsEvent1)
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
func (it *IndexedLogsEvent1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsEvent1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsEvent1 represents a Event1 event raised by the IndexedLogs contract.
type IndexedLogsEvent1 struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEvent1 is a free log retrieval operation binding the contract event 0x04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3.
//
// Solidity: event Event1(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) FilterEvent1(opts *bind.FilterOpts) (*IndexedLogsEvent1Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Event1")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsEvent1Iterator{contract: _IndexedLogs.contract, event: "Event1", logs: logs, sub: sub}, nil
}

// WatchEvent1 is a free log subscription operation binding the contract event 0x04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3.
//
// Solidity: event Event1(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) WatchEvent1(opts *bind.WatchOpts, sink chan<- *IndexedLogsEvent1) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Event1")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsEvent1)
				if err := _IndexedLogs.contract.UnpackLog(event, "Event1", log); err != nil {
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

// ParseEvent1 is a log parse operation binding the contract event 0x04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3.
//
// Solidity: event Event1(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) ParseEvent1(log types.Log) (*IndexedLogsEvent1, error) {
	event := new(IndexedLogsEvent1)
	if err := _IndexedLogs.contract.UnpackLog(event, "Event1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsEvent2Iterator is returned from FilterEvent2 and is used to iterate over the raw logs and unpacked data for Event2 events raised by the IndexedLogs contract.
type IndexedLogsEvent2Iterator struct {
	Event *IndexedLogsEvent2 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsEvent2Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsEvent2)
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
		it.Event = new(IndexedLogsEvent2)
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
func (it *IndexedLogsEvent2Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsEvent2Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsEvent2 represents a Event2 event raised by the IndexedLogs contract.
type IndexedLogsEvent2 struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEvent2 is a free log retrieval operation binding the contract event 0x06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e.
//
// Solidity: event Event2(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) FilterEvent2(opts *bind.FilterOpts) (*IndexedLogsEvent2Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Event2")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsEvent2Iterator{contract: _IndexedLogs.contract, event: "Event2", logs: logs, sub: sub}, nil
}

// WatchEvent2 is a free log subscription operation binding the contract event 0x06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e.
//
// Solidity: event Event2(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) WatchEvent2(opts *bind.WatchOpts, sink chan<- *IndexedLogsEvent2) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Event2")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsEvent2)
				if err := _IndexedLogs.contract.UnpackLog(event, "Event2", log); err != nil {
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

// ParseEvent2 is a log parse operation binding the contract event 0x06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e.
//
// Solidity: event Event2(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) ParseEvent2(log types.Log) (*IndexedLogsEvent2, error) {
	event := new(IndexedLogsEvent2)
	if err := _IndexedLogs.contract.UnpackLog(event, "Event2", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsEvent3Iterator is returned from FilterEvent3 and is used to iterate over the raw logs and unpacked data for Event3 events raised by the IndexedLogs contract.
type IndexedLogsEvent3Iterator struct {
	Event *IndexedLogsEvent3 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsEvent3Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsEvent3)
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
		it.Event = new(IndexedLogsEvent3)
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
func (it *IndexedLogsEvent3Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsEvent3Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsEvent3 represents a Event3 event raised by the IndexedLogs contract.
type IndexedLogsEvent3 struct {
	Id   *big.Int
	Text string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterEvent3 is a free log retrieval operation binding the contract event 0x93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628.
//
// Solidity: event Event3(uint256 id, string text)
func (_IndexedLogs *IndexedLogsFilterer) FilterEvent3(opts *bind.FilterOpts) (*IndexedLogsEvent3Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Event3")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsEvent3Iterator{contract: _IndexedLogs.contract, event: "Event3", logs: logs, sub: sub}, nil
}

// WatchEvent3 is a free log subscription operation binding the contract event 0x93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628.
//
// Solidity: event Event3(uint256 id, string text)
func (_IndexedLogs *IndexedLogsFilterer) WatchEvent3(opts *bind.WatchOpts, sink chan<- *IndexedLogsEvent3) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Event3")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsEvent3)
				if err := _IndexedLogs.contract.UnpackLog(event, "Event3", log); err != nil {
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

// ParseEvent3 is a log parse operation binding the contract event 0x93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628.
//
// Solidity: event Event3(uint256 id, string text)
func (_IndexedLogs *IndexedLogsFilterer) ParseEvent3(log types.Log) (*IndexedLogsEvent3, error) {
	event := new(IndexedLogsEvent3)
	if err := _IndexedLogs.contract.UnpackLog(event, "Event3", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
