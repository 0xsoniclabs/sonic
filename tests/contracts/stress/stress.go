// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package stress

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

// StressMetaData contains all meta data concerning the Stress contract.
var StressMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"result\",\"type\":\"uint256\"}],\"name\":\"ComputationDone\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"rounds\",\"type\":\"uint256\"}],\"name\":\"computeHeavySum\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506102da8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c80632ab647d41461002d575b5f5ffd5b6100476004803603810190610042919061014f565b61005d565b6040516100549190610189565b60405180910390f35b5f5f5f60405160200161007091906101c2565b6040516020818303038152906040528051906020012090505f600190505b8381116100d35781816040516020016100a8929190610205565b60405160208183030381529060405280519060200120915080806100cb9061025d565b91505061008e565b507f9166147bd6409e460385d56a842ba3e08761c4a3036a523486d400ee6a528881815f1c6040516101059190610189565b60405180910390a1805f1c915050919050565b5f5ffd5b5f819050919050565b61012e8161011c565b8114610138575f5ffd5b50565b5f8135905061014981610125565b92915050565b5f6020828403121561016457610163610118565b5b5f6101718482850161013b565b91505092915050565b6101838161011c565b82525050565b5f60208201905061019c5f83018461017a565b92915050565b5f819050919050565b6101bc6101b78261011c565b6101a2565b82525050565b5f6101cd82846101ab565b60208201915081905092915050565b5f819050919050565b5f819050919050565b6101ff6101fa826101dc565b6101e5565b82525050565b5f61021082856101ee565b60208201915061022082846101ab565b6020820191508190509392505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6102678261011c565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361029957610298610230565b5b60018201905091905056fea264697066735822122067a268a7c73d6557d399b5fd102f6939a9f8d6f2867a9cc1a9d5a22e20c43cb564736f6c634300081e0033",
}

// StressABI is the input ABI used to generate the binding from.
// Deprecated: Use StressMetaData.ABI instead.
var StressABI = StressMetaData.ABI

// StressBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StressMetaData.Bin instead.
var StressBin = StressMetaData.Bin

// DeployStress deploys a new Ethereum contract, binding an instance of Stress to it.
func DeployStress(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Stress, error) {
	parsed, err := StressMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StressBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Stress{StressCaller: StressCaller{contract: contract}, StressTransactor: StressTransactor{contract: contract}, StressFilterer: StressFilterer{contract: contract}}, nil
}

// Stress is an auto generated Go binding around an Ethereum contract.
type Stress struct {
	StressCaller     // Read-only binding to the contract
	StressTransactor // Write-only binding to the contract
	StressFilterer   // Log filterer for contract events
}

// StressCaller is an auto generated read-only Go binding around an Ethereum contract.
type StressCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StressTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StressTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StressFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StressFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StressSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StressSession struct {
	Contract     *Stress           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StressCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StressCallerSession struct {
	Contract *StressCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// StressTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StressTransactorSession struct {
	Contract     *StressTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StressRaw is an auto generated low-level Go binding around an Ethereum contract.
type StressRaw struct {
	Contract *Stress // Generic contract binding to access the raw methods on
}

// StressCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StressCallerRaw struct {
	Contract *StressCaller // Generic read-only contract binding to access the raw methods on
}

// StressTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StressTransactorRaw struct {
	Contract *StressTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStress creates a new instance of Stress, bound to a specific deployed contract.
func NewStress(address common.Address, backend bind.ContractBackend) (*Stress, error) {
	contract, err := bindStress(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Stress{StressCaller: StressCaller{contract: contract}, StressTransactor: StressTransactor{contract: contract}, StressFilterer: StressFilterer{contract: contract}}, nil
}

// NewStressCaller creates a new read-only instance of Stress, bound to a specific deployed contract.
func NewStressCaller(address common.Address, caller bind.ContractCaller) (*StressCaller, error) {
	contract, err := bindStress(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StressCaller{contract: contract}, nil
}

// NewStressTransactor creates a new write-only instance of Stress, bound to a specific deployed contract.
func NewStressTransactor(address common.Address, transactor bind.ContractTransactor) (*StressTransactor, error) {
	contract, err := bindStress(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StressTransactor{contract: contract}, nil
}

// NewStressFilterer creates a new log filterer instance of Stress, bound to a specific deployed contract.
func NewStressFilterer(address common.Address, filterer bind.ContractFilterer) (*StressFilterer, error) {
	contract, err := bindStress(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StressFilterer{contract: contract}, nil
}

// bindStress binds a generic wrapper to an already deployed contract.
func bindStress(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StressMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Stress *StressRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Stress.Contract.StressCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Stress *StressRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Stress.Contract.StressTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Stress *StressRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Stress.Contract.StressTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Stress *StressCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Stress.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Stress *StressTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Stress.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Stress *StressTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Stress.Contract.contract.Transact(opts, method, params...)
}

// ComputeHeavySum is a paid mutator transaction binding the contract method 0x2ab647d4.
//
// Solidity: function computeHeavySum(uint256 rounds) returns(uint256)
func (_Stress *StressTransactor) ComputeHeavySum(opts *bind.TransactOpts, rounds *big.Int) (*types.Transaction, error) {
	return _Stress.contract.Transact(opts, "computeHeavySum", rounds)
}

// ComputeHeavySum is a paid mutator transaction binding the contract method 0x2ab647d4.
//
// Solidity: function computeHeavySum(uint256 rounds) returns(uint256)
func (_Stress *StressSession) ComputeHeavySum(rounds *big.Int) (*types.Transaction, error) {
	return _Stress.Contract.ComputeHeavySum(&_Stress.TransactOpts, rounds)
}

// ComputeHeavySum is a paid mutator transaction binding the contract method 0x2ab647d4.
//
// Solidity: function computeHeavySum(uint256 rounds) returns(uint256)
func (_Stress *StressTransactorSession) ComputeHeavySum(rounds *big.Int) (*types.Transaction, error) {
	return _Stress.Contract.ComputeHeavySum(&_Stress.TransactOpts, rounds)
}

// StressComputationDoneIterator is returned from FilterComputationDone and is used to iterate over the raw logs and unpacked data for ComputationDone events raised by the Stress contract.
type StressComputationDoneIterator struct {
	Event *StressComputationDone // Event containing the contract specifics and raw log

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
func (it *StressComputationDoneIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StressComputationDone)
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
		it.Event = new(StressComputationDone)
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
func (it *StressComputationDoneIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StressComputationDoneIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StressComputationDone represents a ComputationDone event raised by the Stress contract.
type StressComputationDone struct {
	Result *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterComputationDone is a free log retrieval operation binding the contract event 0x9166147bd6409e460385d56a842ba3e08761c4a3036a523486d400ee6a528881.
//
// Solidity: event ComputationDone(uint256 result)
func (_Stress *StressFilterer) FilterComputationDone(opts *bind.FilterOpts) (*StressComputationDoneIterator, error) {

	logs, sub, err := _Stress.contract.FilterLogs(opts, "ComputationDone")
	if err != nil {
		return nil, err
	}
	return &StressComputationDoneIterator{contract: _Stress.contract, event: "ComputationDone", logs: logs, sub: sub}, nil
}

// WatchComputationDone is a free log subscription operation binding the contract event 0x9166147bd6409e460385d56a842ba3e08761c4a3036a523486d400ee6a528881.
//
// Solidity: event ComputationDone(uint256 result)
func (_Stress *StressFilterer) WatchComputationDone(opts *bind.WatchOpts, sink chan<- *StressComputationDone) (event.Subscription, error) {

	logs, sub, err := _Stress.contract.WatchLogs(opts, "ComputationDone")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StressComputationDone)
				if err := _Stress.contract.UnpackLog(event, "ComputationDone", log); err != nil {
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

// ParseComputationDone is a log parse operation binding the contract event 0x9166147bd6409e460385d56a842ba3e08761c4a3036a523486d400ee6a528881.
//
// Solidity: event ComputationDone(uint256 result)
func (_Stress *StressFilterer) ParseComputationDone(log types.Log) (*StressComputationDone, error) {
	event := new(StressComputationDone)
	if err := _Stress.contract.UnpackLog(event, "ComputationDone", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
