// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package read_history_storage

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

// ReadHistoryStorageMetaData contains all meta data concerning the ReadHistoryStorage contract.
var ReadHistoryStorageMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"queriedBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"BlockHash\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"readHistoryStorage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103b38061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c806341a64b4c1461002d575b5f5ffd5b610047600480360381019061004291906101b6565b610049565b005b5f71f90827f1c53a10cb7a02335b17532000293590505f5f8273ffffffffffffffffffffffffffffffffffffffff168460405160200161008991906101f0565b6040516020818303038152906040526040516100a5919061025b565b5f604051808303815f865af19150503d805f81146100de576040519150601f19603f3d011682016040523d82523d5f602084013e6100e3565b606091505b509150915081610128576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161011f906102cb565b60405180910390fd5b5f8180602001905181019061013d919061031c565b90507f1599ad63580ca2bbd26e39c5584488358d60e2a67869cfe401936098ac9841768582604051610170929190610356565b60405180910390a15050505050565b5f5ffd5b5f819050919050565b61019581610183565b811461019f575f5ffd5b50565b5f813590506101b08161018c565b92915050565b5f602082840312156101cb576101ca61017f565b5b5f6101d8848285016101a2565b91505092915050565b6101ea81610183565b82525050565b5f6020820190506102035f8301846101e1565b92915050565b5f81519050919050565b5f81905092915050565b8281835e5f83830152505050565b5f61023582610209565b61023f8185610213565b935061024f81856020860161021d565b80840191505092915050565b5f610266828461022b565b915081905092915050565b5f82825260208201905092915050565b7f63616c6c206661696c65640000000000000000000000000000000000000000005f82015250565b5f6102b5600b83610271565b91506102c082610281565b602082019050919050565b5f6020820190508181035f8301526102e2816102a9565b9050919050565b5f819050919050565b6102fb816102e9565b8114610305575f5ffd5b50565b5f81519050610316816102f2565b92915050565b5f602082840312156103315761033061017f565b5b5f61033e84828501610308565b91505092915050565b610350816102e9565b82525050565b5f6040820190506103695f8301856101e1565b6103766020830184610347565b939250505056fea2646970667358221220b184d47d287a25e70a42a95c688def12410c474de0341b82ba7c8c30cae84d2664736f6c634300081d0033",
}

// ReadHistoryStorageABI is the input ABI used to generate the binding from.
// Deprecated: Use ReadHistoryStorageMetaData.ABI instead.
var ReadHistoryStorageABI = ReadHistoryStorageMetaData.ABI

// ReadHistoryStorageBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ReadHistoryStorageMetaData.Bin instead.
var ReadHistoryStorageBin = ReadHistoryStorageMetaData.Bin

// DeployReadHistoryStorage deploys a new Ethereum contract, binding an instance of ReadHistoryStorage to it.
func DeployReadHistoryStorage(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ReadHistoryStorage, error) {
	parsed, err := ReadHistoryStorageMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ReadHistoryStorageBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ReadHistoryStorage{ReadHistoryStorageCaller: ReadHistoryStorageCaller{contract: contract}, ReadHistoryStorageTransactor: ReadHistoryStorageTransactor{contract: contract}, ReadHistoryStorageFilterer: ReadHistoryStorageFilterer{contract: contract}}, nil
}

// ReadHistoryStorage is an auto generated Go binding around an Ethereum contract.
type ReadHistoryStorage struct {
	ReadHistoryStorageCaller     // Read-only binding to the contract
	ReadHistoryStorageTransactor // Write-only binding to the contract
	ReadHistoryStorageFilterer   // Log filterer for contract events
}

// ReadHistoryStorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type ReadHistoryStorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReadHistoryStorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ReadHistoryStorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReadHistoryStorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ReadHistoryStorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ReadHistoryStorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ReadHistoryStorageSession struct {
	Contract     *ReadHistoryStorage // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ReadHistoryStorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ReadHistoryStorageCallerSession struct {
	Contract *ReadHistoryStorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// ReadHistoryStorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ReadHistoryStorageTransactorSession struct {
	Contract     *ReadHistoryStorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// ReadHistoryStorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type ReadHistoryStorageRaw struct {
	Contract *ReadHistoryStorage // Generic contract binding to access the raw methods on
}

// ReadHistoryStorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ReadHistoryStorageCallerRaw struct {
	Contract *ReadHistoryStorageCaller // Generic read-only contract binding to access the raw methods on
}

// ReadHistoryStorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ReadHistoryStorageTransactorRaw struct {
	Contract *ReadHistoryStorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewReadHistoryStorage creates a new instance of ReadHistoryStorage, bound to a specific deployed contract.
func NewReadHistoryStorage(address common.Address, backend bind.ContractBackend) (*ReadHistoryStorage, error) {
	contract, err := bindReadHistoryStorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ReadHistoryStorage{ReadHistoryStorageCaller: ReadHistoryStorageCaller{contract: contract}, ReadHistoryStorageTransactor: ReadHistoryStorageTransactor{contract: contract}, ReadHistoryStorageFilterer: ReadHistoryStorageFilterer{contract: contract}}, nil
}

// NewReadHistoryStorageCaller creates a new read-only instance of ReadHistoryStorage, bound to a specific deployed contract.
func NewReadHistoryStorageCaller(address common.Address, caller bind.ContractCaller) (*ReadHistoryStorageCaller, error) {
	contract, err := bindReadHistoryStorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ReadHistoryStorageCaller{contract: contract}, nil
}

// NewReadHistoryStorageTransactor creates a new write-only instance of ReadHistoryStorage, bound to a specific deployed contract.
func NewReadHistoryStorageTransactor(address common.Address, transactor bind.ContractTransactor) (*ReadHistoryStorageTransactor, error) {
	contract, err := bindReadHistoryStorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ReadHistoryStorageTransactor{contract: contract}, nil
}

// NewReadHistoryStorageFilterer creates a new log filterer instance of ReadHistoryStorage, bound to a specific deployed contract.
func NewReadHistoryStorageFilterer(address common.Address, filterer bind.ContractFilterer) (*ReadHistoryStorageFilterer, error) {
	contract, err := bindReadHistoryStorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ReadHistoryStorageFilterer{contract: contract}, nil
}

// bindReadHistoryStorage binds a generic wrapper to an already deployed contract.
func bindReadHistoryStorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ReadHistoryStorageMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ReadHistoryStorage *ReadHistoryStorageRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ReadHistoryStorage.Contract.ReadHistoryStorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ReadHistoryStorage *ReadHistoryStorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ReadHistoryStorage.Contract.ReadHistoryStorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ReadHistoryStorage *ReadHistoryStorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ReadHistoryStorage.Contract.ReadHistoryStorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ReadHistoryStorage *ReadHistoryStorageCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ReadHistoryStorage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ReadHistoryStorage *ReadHistoryStorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ReadHistoryStorage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ReadHistoryStorage *ReadHistoryStorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ReadHistoryStorage.Contract.contract.Transact(opts, method, params...)
}

// ReadHistoryStorage is a paid mutator transaction binding the contract method 0x41a64b4c.
//
// Solidity: function readHistoryStorage(uint256 blockNumber) returns()
func (_ReadHistoryStorage *ReadHistoryStorageTransactor) ReadHistoryStorage(opts *bind.TransactOpts, blockNumber *big.Int) (*types.Transaction, error) {
	return _ReadHistoryStorage.contract.Transact(opts, "readHistoryStorage", blockNumber)
}

// ReadHistoryStorage is a paid mutator transaction binding the contract method 0x41a64b4c.
//
// Solidity: function readHistoryStorage(uint256 blockNumber) returns()
func (_ReadHistoryStorage *ReadHistoryStorageSession) ReadHistoryStorage(blockNumber *big.Int) (*types.Transaction, error) {
	return _ReadHistoryStorage.Contract.ReadHistoryStorage(&_ReadHistoryStorage.TransactOpts, blockNumber)
}

// ReadHistoryStorage is a paid mutator transaction binding the contract method 0x41a64b4c.
//
// Solidity: function readHistoryStorage(uint256 blockNumber) returns()
func (_ReadHistoryStorage *ReadHistoryStorageTransactorSession) ReadHistoryStorage(blockNumber *big.Int) (*types.Transaction, error) {
	return _ReadHistoryStorage.Contract.ReadHistoryStorage(&_ReadHistoryStorage.TransactOpts, blockNumber)
}

// ReadHistoryStorageBlockHashIterator is returned from FilterBlockHash and is used to iterate over the raw logs and unpacked data for BlockHash events raised by the ReadHistoryStorage contract.
type ReadHistoryStorageBlockHashIterator struct {
	Event *ReadHistoryStorageBlockHash // Event containing the contract specifics and raw log

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
func (it *ReadHistoryStorageBlockHashIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ReadHistoryStorageBlockHash)
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
		it.Event = new(ReadHistoryStorageBlockHash)
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
func (it *ReadHistoryStorageBlockHashIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ReadHistoryStorageBlockHashIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ReadHistoryStorageBlockHash represents a BlockHash event raised by the ReadHistoryStorage contract.
type ReadHistoryStorageBlockHash struct {
	QueriedBlock *big.Int
	BlockHash    [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBlockHash is a free log retrieval operation binding the contract event 0x1599ad63580ca2bbd26e39c5584488358d60e2a67869cfe401936098ac984176.
//
// Solidity: event BlockHash(uint256 queriedBlock, bytes32 blockHash)
func (_ReadHistoryStorage *ReadHistoryStorageFilterer) FilterBlockHash(opts *bind.FilterOpts) (*ReadHistoryStorageBlockHashIterator, error) {

	logs, sub, err := _ReadHistoryStorage.contract.FilterLogs(opts, "BlockHash")
	if err != nil {
		return nil, err
	}
	return &ReadHistoryStorageBlockHashIterator{contract: _ReadHistoryStorage.contract, event: "BlockHash", logs: logs, sub: sub}, nil
}

// WatchBlockHash is a free log subscription operation binding the contract event 0x1599ad63580ca2bbd26e39c5584488358d60e2a67869cfe401936098ac984176.
//
// Solidity: event BlockHash(uint256 queriedBlock, bytes32 blockHash)
func (_ReadHistoryStorage *ReadHistoryStorageFilterer) WatchBlockHash(opts *bind.WatchOpts, sink chan<- *ReadHistoryStorageBlockHash) (event.Subscription, error) {

	logs, sub, err := _ReadHistoryStorage.contract.WatchLogs(opts, "BlockHash")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ReadHistoryStorageBlockHash)
				if err := _ReadHistoryStorage.contract.UnpackLog(event, "BlockHash", log); err != nil {
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

// ParseBlockHash is a log parse operation binding the contract event 0x1599ad63580ca2bbd26e39c5584488358d60e2a67869cfe401936098ac984176.
//
// Solidity: event BlockHash(uint256 queriedBlock, bytes32 blockHash)
func (_ReadHistoryStorage *ReadHistoryStorageFilterer) ParseBlockHash(log types.Log) (*ReadHistoryStorageBlockHash, error) {
	event := new(ReadHistoryStorageBlockHash)
	if err := _ReadHistoryStorage.contract.UnpackLog(event, "BlockHash", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
