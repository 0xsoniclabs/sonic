// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package revert

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

// RevertMetaData contains all meta data concerning the Revert contract.
var RevertMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"message\",\"type\":\"string\"}],\"name\":\"SideEffect\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"doCrash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"doRevert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506102718061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063afc874d214610038578063c0c8994a14610042575b5f5ffd5b61004061004c565b005b61004a6100bc565b005b7f129c09367153bae86b3c5ad9663463604ac73f61db23fb620b2de5f33cebe2506040516100799061014d565b60405180910390a16040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016100b3906101b5565b60405180910390fd5b7f129c09367153bae86b3c5ad9663463604ac73f61db23fb620b2de5f33cebe2506040516100e99061021d565b60405180910390a1fe5b5f82825260208201905092915050565b7f4265666f726520726576657274000000000000000000000000000000000000005f82015250565b5f610137600d836100f3565b915061014282610103565b602082019050919050565b5f6020820190508181035f8301526101648161012b565b9050919050565b7f52657665727465640000000000000000000000000000000000000000000000005f82015250565b5f61019f6008836100f3565b91506101aa8261016b565b602082019050919050565b5f6020820190508181035f8301526101cc81610193565b9050919050565b7f4265666f726520637261736800000000000000000000000000000000000000005f82015250565b5f610207600c836100f3565b9150610212826101d3565b602082019050919050565b5f6020820190508181035f830152610234816101fb565b905091905056fea26469706673582212204eb6db705af87094157448a1e10261e2f732420a41fa277f877e9024b2761c5e64736f6c634300081e0033",
}

// RevertABI is the input ABI used to generate the binding from.
// Deprecated: Use RevertMetaData.ABI instead.
var RevertABI = RevertMetaData.ABI

// RevertBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RevertMetaData.Bin instead.
var RevertBin = RevertMetaData.Bin

// DeployRevert deploys a new Ethereum contract, binding an instance of Revert to it.
func DeployRevert(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Revert, error) {
	parsed, err := RevertMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RevertBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Revert{RevertCaller: RevertCaller{contract: contract}, RevertTransactor: RevertTransactor{contract: contract}, RevertFilterer: RevertFilterer{contract: contract}}, nil
}

// Revert is an auto generated Go binding around an Ethereum contract.
type Revert struct {
	RevertCaller     // Read-only binding to the contract
	RevertTransactor // Write-only binding to the contract
	RevertFilterer   // Log filterer for contract events
}

// RevertCaller is an auto generated read-only Go binding around an Ethereum contract.
type RevertCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RevertTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RevertTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RevertFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RevertFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RevertSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RevertSession struct {
	Contract     *Revert           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RevertCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RevertCallerSession struct {
	Contract *RevertCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// RevertTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RevertTransactorSession struct {
	Contract     *RevertTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RevertRaw is an auto generated low-level Go binding around an Ethereum contract.
type RevertRaw struct {
	Contract *Revert // Generic contract binding to access the raw methods on
}

// RevertCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RevertCallerRaw struct {
	Contract *RevertCaller // Generic read-only contract binding to access the raw methods on
}

// RevertTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RevertTransactorRaw struct {
	Contract *RevertTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRevert creates a new instance of Revert, bound to a specific deployed contract.
func NewRevert(address common.Address, backend bind.ContractBackend) (*Revert, error) {
	contract, err := bindRevert(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Revert{RevertCaller: RevertCaller{contract: contract}, RevertTransactor: RevertTransactor{contract: contract}, RevertFilterer: RevertFilterer{contract: contract}}, nil
}

// NewRevertCaller creates a new read-only instance of Revert, bound to a specific deployed contract.
func NewRevertCaller(address common.Address, caller bind.ContractCaller) (*RevertCaller, error) {
	contract, err := bindRevert(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RevertCaller{contract: contract}, nil
}

// NewRevertTransactor creates a new write-only instance of Revert, bound to a specific deployed contract.
func NewRevertTransactor(address common.Address, transactor bind.ContractTransactor) (*RevertTransactor, error) {
	contract, err := bindRevert(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RevertTransactor{contract: contract}, nil
}

// NewRevertFilterer creates a new log filterer instance of Revert, bound to a specific deployed contract.
func NewRevertFilterer(address common.Address, filterer bind.ContractFilterer) (*RevertFilterer, error) {
	contract, err := bindRevert(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RevertFilterer{contract: contract}, nil
}

// bindRevert binds a generic wrapper to an already deployed contract.
func bindRevert(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RevertMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Revert *RevertRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Revert.Contract.RevertCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Revert *RevertRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Revert.Contract.RevertTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Revert *RevertRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Revert.Contract.RevertTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Revert *RevertCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Revert.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Revert *RevertTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Revert.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Revert *RevertTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Revert.Contract.contract.Transact(opts, method, params...)
}

// DoCrash is a paid mutator transaction binding the contract method 0xc0c8994a.
//
// Solidity: function doCrash() returns()
func (_Revert *RevertTransactor) DoCrash(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Revert.contract.Transact(opts, "doCrash")
}

// DoCrash is a paid mutator transaction binding the contract method 0xc0c8994a.
//
// Solidity: function doCrash() returns()
func (_Revert *RevertSession) DoCrash() (*types.Transaction, error) {
	return _Revert.Contract.DoCrash(&_Revert.TransactOpts)
}

// DoCrash is a paid mutator transaction binding the contract method 0xc0c8994a.
//
// Solidity: function doCrash() returns()
func (_Revert *RevertTransactorSession) DoCrash() (*types.Transaction, error) {
	return _Revert.Contract.DoCrash(&_Revert.TransactOpts)
}

// DoRevert is a paid mutator transaction binding the contract method 0xafc874d2.
//
// Solidity: function doRevert() returns()
func (_Revert *RevertTransactor) DoRevert(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Revert.contract.Transact(opts, "doRevert")
}

// DoRevert is a paid mutator transaction binding the contract method 0xafc874d2.
//
// Solidity: function doRevert() returns()
func (_Revert *RevertSession) DoRevert() (*types.Transaction, error) {
	return _Revert.Contract.DoRevert(&_Revert.TransactOpts)
}

// DoRevert is a paid mutator transaction binding the contract method 0xafc874d2.
//
// Solidity: function doRevert() returns()
func (_Revert *RevertTransactorSession) DoRevert() (*types.Transaction, error) {
	return _Revert.Contract.DoRevert(&_Revert.TransactOpts)
}

// RevertSideEffectIterator is returned from FilterSideEffect and is used to iterate over the raw logs and unpacked data for SideEffect events raised by the Revert contract.
type RevertSideEffectIterator struct {
	Event *RevertSideEffect // Event containing the contract specifics and raw log

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
func (it *RevertSideEffectIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RevertSideEffect)
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
		it.Event = new(RevertSideEffect)
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
func (it *RevertSideEffectIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RevertSideEffectIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RevertSideEffect represents a SideEffect event raised by the Revert contract.
type RevertSideEffect struct {
	Message string
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSideEffect is a free log retrieval operation binding the contract event 0x129c09367153bae86b3c5ad9663463604ac73f61db23fb620b2de5f33cebe250.
//
// Solidity: event SideEffect(string message)
func (_Revert *RevertFilterer) FilterSideEffect(opts *bind.FilterOpts) (*RevertSideEffectIterator, error) {

	logs, sub, err := _Revert.contract.FilterLogs(opts, "SideEffect")
	if err != nil {
		return nil, err
	}
	return &RevertSideEffectIterator{contract: _Revert.contract, event: "SideEffect", logs: logs, sub: sub}, nil
}

// WatchSideEffect is a free log subscription operation binding the contract event 0x129c09367153bae86b3c5ad9663463604ac73f61db23fb620b2de5f33cebe250.
//
// Solidity: event SideEffect(string message)
func (_Revert *RevertFilterer) WatchSideEffect(opts *bind.WatchOpts, sink chan<- *RevertSideEffect) (event.Subscription, error) {

	logs, sub, err := _Revert.contract.WatchLogs(opts, "SideEffect")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RevertSideEffect)
				if err := _Revert.contract.UnpackLog(event, "SideEffect", log); err != nil {
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

// ParseSideEffect is a log parse operation binding the contract event 0x129c09367153bae86b3c5ad9663463604ac73f61db23fb620b2de5f33cebe250.
//
// Solidity: event SideEffect(string message)
func (_Revert *RevertFilterer) ParseSideEffect(log types.Log) (*RevertSideEffect, error) {
	event := new(RevertSideEffect)
	if err := _Revert.contract.UnpackLog(event, "SideEffect", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
