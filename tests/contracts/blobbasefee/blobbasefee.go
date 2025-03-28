// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package blobbasefee

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

// BlobbasefeeMetaData contains all meta data concerning the Blobbasefee contract.
var BlobbasefeeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"CurrentBlobBaseFee\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getBlobBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"logCurrentBlobBaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5060f78061001b5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c80631f6d6ef7146034578063f7505d4014604e575b5f5ffd5b603a6056565b6040516045919060aa565b60405180910390f35b6054605d565b005b5f4a905090565b7ffb393adc5f0cf6fbe93314b17d3f3345d5d076fa952b76c3b3ef84768f2df5424a604051608a919060aa565b60405180910390a1565b5f819050919050565b60a4816094565b82525050565b5f60208201905060bb5f830184609d565b9291505056fea26469706673582212200e7832ad20e251a217456d6839aba478780bf836f8b4a6c1a793d76d4a4df51364736f6c634300081c0033",
}

// BlobbasefeeABI is the input ABI used to generate the binding from.
// Deprecated: Use BlobbasefeeMetaData.ABI instead.
var BlobbasefeeABI = BlobbasefeeMetaData.ABI

// BlobbasefeeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BlobbasefeeMetaData.Bin instead.
var BlobbasefeeBin = BlobbasefeeMetaData.Bin

// DeployBlobbasefee deploys a new Ethereum contract, binding an instance of Blobbasefee to it.
func DeployBlobbasefee(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Blobbasefee, error) {
	parsed, err := BlobbasefeeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BlobbasefeeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Blobbasefee{BlobbasefeeCaller: BlobbasefeeCaller{contract: contract}, BlobbasefeeTransactor: BlobbasefeeTransactor{contract: contract}, BlobbasefeeFilterer: BlobbasefeeFilterer{contract: contract}}, nil
}

// Blobbasefee is an auto generated Go binding around an Ethereum contract.
type Blobbasefee struct {
	BlobbasefeeCaller     // Read-only binding to the contract
	BlobbasefeeTransactor // Write-only binding to the contract
	BlobbasefeeFilterer   // Log filterer for contract events
}

// BlobbasefeeCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlobbasefeeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlobbasefeeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlobbasefeeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlobbasefeeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlobbasefeeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlobbasefeeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlobbasefeeSession struct {
	Contract     *Blobbasefee      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlobbasefeeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlobbasefeeCallerSession struct {
	Contract *BlobbasefeeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// BlobbasefeeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlobbasefeeTransactorSession struct {
	Contract     *BlobbasefeeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// BlobbasefeeRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlobbasefeeRaw struct {
	Contract *Blobbasefee // Generic contract binding to access the raw methods on
}

// BlobbasefeeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlobbasefeeCallerRaw struct {
	Contract *BlobbasefeeCaller // Generic read-only contract binding to access the raw methods on
}

// BlobbasefeeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlobbasefeeTransactorRaw struct {
	Contract *BlobbasefeeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlobbasefee creates a new instance of Blobbasefee, bound to a specific deployed contract.
func NewBlobbasefee(address common.Address, backend bind.ContractBackend) (*Blobbasefee, error) {
	contract, err := bindBlobbasefee(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Blobbasefee{BlobbasefeeCaller: BlobbasefeeCaller{contract: contract}, BlobbasefeeTransactor: BlobbasefeeTransactor{contract: contract}, BlobbasefeeFilterer: BlobbasefeeFilterer{contract: contract}}, nil
}

// NewBlobbasefeeCaller creates a new read-only instance of Blobbasefee, bound to a specific deployed contract.
func NewBlobbasefeeCaller(address common.Address, caller bind.ContractCaller) (*BlobbasefeeCaller, error) {
	contract, err := bindBlobbasefee(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlobbasefeeCaller{contract: contract}, nil
}

// NewBlobbasefeeTransactor creates a new write-only instance of Blobbasefee, bound to a specific deployed contract.
func NewBlobbasefeeTransactor(address common.Address, transactor bind.ContractTransactor) (*BlobbasefeeTransactor, error) {
	contract, err := bindBlobbasefee(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlobbasefeeTransactor{contract: contract}, nil
}

// NewBlobbasefeeFilterer creates a new log filterer instance of Blobbasefee, bound to a specific deployed contract.
func NewBlobbasefeeFilterer(address common.Address, filterer bind.ContractFilterer) (*BlobbasefeeFilterer, error) {
	contract, err := bindBlobbasefee(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlobbasefeeFilterer{contract: contract}, nil
}

// bindBlobbasefee binds a generic wrapper to an already deployed contract.
func bindBlobbasefee(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BlobbasefeeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Blobbasefee *BlobbasefeeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Blobbasefee.Contract.BlobbasefeeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Blobbasefee *BlobbasefeeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Blobbasefee.Contract.BlobbasefeeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Blobbasefee *BlobbasefeeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Blobbasefee.Contract.BlobbasefeeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Blobbasefee *BlobbasefeeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Blobbasefee.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Blobbasefee *BlobbasefeeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Blobbasefee.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Blobbasefee *BlobbasefeeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Blobbasefee.Contract.contract.Transact(opts, method, params...)
}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x1f6d6ef7.
//
// Solidity: function getBlobBaseFee() view returns(uint256)
func (_Blobbasefee *BlobbasefeeCaller) GetBlobBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Blobbasefee.contract.Call(opts, &out, "getBlobBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x1f6d6ef7.
//
// Solidity: function getBlobBaseFee() view returns(uint256)
func (_Blobbasefee *BlobbasefeeSession) GetBlobBaseFee() (*big.Int, error) {
	return _Blobbasefee.Contract.GetBlobBaseFee(&_Blobbasefee.CallOpts)
}

// GetBlobBaseFee is a free data retrieval call binding the contract method 0x1f6d6ef7.
//
// Solidity: function getBlobBaseFee() view returns(uint256)
func (_Blobbasefee *BlobbasefeeCallerSession) GetBlobBaseFee() (*big.Int, error) {
	return _Blobbasefee.Contract.GetBlobBaseFee(&_Blobbasefee.CallOpts)
}

// LogCurrentBlobBaseFee is a paid mutator transaction binding the contract method 0xf7505d40.
//
// Solidity: function logCurrentBlobBaseFee() returns()
func (_Blobbasefee *BlobbasefeeTransactor) LogCurrentBlobBaseFee(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Blobbasefee.contract.Transact(opts, "logCurrentBlobBaseFee")
}

// LogCurrentBlobBaseFee is a paid mutator transaction binding the contract method 0xf7505d40.
//
// Solidity: function logCurrentBlobBaseFee() returns()
func (_Blobbasefee *BlobbasefeeSession) LogCurrentBlobBaseFee() (*types.Transaction, error) {
	return _Blobbasefee.Contract.LogCurrentBlobBaseFee(&_Blobbasefee.TransactOpts)
}

// LogCurrentBlobBaseFee is a paid mutator transaction binding the contract method 0xf7505d40.
//
// Solidity: function logCurrentBlobBaseFee() returns()
func (_Blobbasefee *BlobbasefeeTransactorSession) LogCurrentBlobBaseFee() (*types.Transaction, error) {
	return _Blobbasefee.Contract.LogCurrentBlobBaseFee(&_Blobbasefee.TransactOpts)
}

// BlobbasefeeCurrentBlobBaseFeeIterator is returned from FilterCurrentBlobBaseFee and is used to iterate over the raw logs and unpacked data for CurrentBlobBaseFee events raised by the Blobbasefee contract.
type BlobbasefeeCurrentBlobBaseFeeIterator struct {
	Event *BlobbasefeeCurrentBlobBaseFee // Event containing the contract specifics and raw log

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
func (it *BlobbasefeeCurrentBlobBaseFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlobbasefeeCurrentBlobBaseFee)
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
		it.Event = new(BlobbasefeeCurrentBlobBaseFee)
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
func (it *BlobbasefeeCurrentBlobBaseFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlobbasefeeCurrentBlobBaseFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlobbasefeeCurrentBlobBaseFee represents a CurrentBlobBaseFee event raised by the Blobbasefee contract.
type BlobbasefeeCurrentBlobBaseFee struct {
	Fee *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterCurrentBlobBaseFee is a free log retrieval operation binding the contract event 0xfb393adc5f0cf6fbe93314b17d3f3345d5d076fa952b76c3b3ef84768f2df542.
//
// Solidity: event CurrentBlobBaseFee(uint256 fee)
func (_Blobbasefee *BlobbasefeeFilterer) FilterCurrentBlobBaseFee(opts *bind.FilterOpts) (*BlobbasefeeCurrentBlobBaseFeeIterator, error) {

	logs, sub, err := _Blobbasefee.contract.FilterLogs(opts, "CurrentBlobBaseFee")
	if err != nil {
		return nil, err
	}
	return &BlobbasefeeCurrentBlobBaseFeeIterator{contract: _Blobbasefee.contract, event: "CurrentBlobBaseFee", logs: logs, sub: sub}, nil
}

// WatchCurrentBlobBaseFee is a free log subscription operation binding the contract event 0xfb393adc5f0cf6fbe93314b17d3f3345d5d076fa952b76c3b3ef84768f2df542.
//
// Solidity: event CurrentBlobBaseFee(uint256 fee)
func (_Blobbasefee *BlobbasefeeFilterer) WatchCurrentBlobBaseFee(opts *bind.WatchOpts, sink chan<- *BlobbasefeeCurrentBlobBaseFee) (event.Subscription, error) {

	logs, sub, err := _Blobbasefee.contract.WatchLogs(opts, "CurrentBlobBaseFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlobbasefeeCurrentBlobBaseFee)
				if err := _Blobbasefee.contract.UnpackLog(event, "CurrentBlobBaseFee", log); err != nil {
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

// ParseCurrentBlobBaseFee is a log parse operation binding the contract event 0xfb393adc5f0cf6fbe93314b17d3f3345d5d076fa952b76c3b3ef84768f2df542.
//
// Solidity: event CurrentBlobBaseFee(uint256 fee)
func (_Blobbasefee *BlobbasefeeFilterer) ParseCurrentBlobBaseFee(log types.Log) (*BlobbasefeeCurrentBlobBaseFee, error) {
	event := new(BlobbasefeeCurrentBlobBaseFee)
	if err := _Blobbasefee.contract.UnpackLog(event, "CurrentBlobBaseFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
