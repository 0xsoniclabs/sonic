// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package basefee

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

// BasefeeMetaData contains all meta data concerning the Basefee contract.
var BasefeeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"CurrentFee\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getBaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"logCurrentBaseFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b5061011d8061001c5f395ff3fe6080604052348015600e575f80fd5b50600436106030575f3560e01c806315e812ad146034578063414783bf14604e575b5f80fd5b603a6056565b6040516045919060aa565b60405180910390f35b6054605d565b005b5f48905090565b7fd25c41149a2ac969bcb91bf41383d6aa5f5246c7586901cc546e7e2361cfb66748604051608a919060aa565b60405180910390a1565b5f819050919050565b60a4816094565b82525050565b5f60208201905060bb5f830184609d565b9291505056fea2646970667358221220f596a2d577eb1d855fc953c1b93a276a20980d515c5f58c02ca75fc3b435a6d564736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
}

// BasefeeABI is the input ABI used to generate the binding from.
// Deprecated: Use BasefeeMetaData.ABI instead.
var BasefeeABI = BasefeeMetaData.ABI

// BasefeeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BasefeeMetaData.Bin instead.
var BasefeeBin = BasefeeMetaData.Bin

// DeployBasefee deploys a new Ethereum contract, binding an instance of Basefee to it.
func DeployBasefee(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Basefee, error) {
	parsed, err := BasefeeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BasefeeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Basefee{BasefeeCaller: BasefeeCaller{contract: contract}, BasefeeTransactor: BasefeeTransactor{contract: contract}, BasefeeFilterer: BasefeeFilterer{contract: contract}}, nil
}

// Basefee is an auto generated Go binding around an Ethereum contract.
type Basefee struct {
	BasefeeCaller     // Read-only binding to the contract
	BasefeeTransactor // Write-only binding to the contract
	BasefeeFilterer   // Log filterer for contract events
}

// BasefeeCaller is an auto generated read-only Go binding around an Ethereum contract.
type BasefeeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BasefeeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BasefeeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BasefeeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BasefeeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BasefeeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BasefeeSession struct {
	Contract     *Basefee          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BasefeeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BasefeeCallerSession struct {
	Contract *BasefeeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// BasefeeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BasefeeTransactorSession struct {
	Contract     *BasefeeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// BasefeeRaw is an auto generated low-level Go binding around an Ethereum contract.
type BasefeeRaw struct {
	Contract *Basefee // Generic contract binding to access the raw methods on
}

// BasefeeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BasefeeCallerRaw struct {
	Contract *BasefeeCaller // Generic read-only contract binding to access the raw methods on
}

// BasefeeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BasefeeTransactorRaw struct {
	Contract *BasefeeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBasefee creates a new instance of Basefee, bound to a specific deployed contract.
func NewBasefee(address common.Address, backend bind.ContractBackend) (*Basefee, error) {
	contract, err := bindBasefee(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Basefee{BasefeeCaller: BasefeeCaller{contract: contract}, BasefeeTransactor: BasefeeTransactor{contract: contract}, BasefeeFilterer: BasefeeFilterer{contract: contract}}, nil
}

// NewBasefeeCaller creates a new read-only instance of Basefee, bound to a specific deployed contract.
func NewBasefeeCaller(address common.Address, caller bind.ContractCaller) (*BasefeeCaller, error) {
	contract, err := bindBasefee(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BasefeeCaller{contract: contract}, nil
}

// NewBasefeeTransactor creates a new write-only instance of Basefee, bound to a specific deployed contract.
func NewBasefeeTransactor(address common.Address, transactor bind.ContractTransactor) (*BasefeeTransactor, error) {
	contract, err := bindBasefee(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BasefeeTransactor{contract: contract}, nil
}

// NewBasefeeFilterer creates a new log filterer instance of Basefee, bound to a specific deployed contract.
func NewBasefeeFilterer(address common.Address, filterer bind.ContractFilterer) (*BasefeeFilterer, error) {
	contract, err := bindBasefee(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BasefeeFilterer{contract: contract}, nil
}

// bindBasefee binds a generic wrapper to an already deployed contract.
func bindBasefee(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BasefeeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Basefee *BasefeeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Basefee.Contract.BasefeeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Basefee *BasefeeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Basefee.Contract.BasefeeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Basefee *BasefeeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Basefee.Contract.BasefeeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Basefee *BasefeeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Basefee.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Basefee *BasefeeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Basefee.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Basefee *BasefeeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Basefee.Contract.contract.Transact(opts, method, params...)
}

// GetBaseFee is a free data retrieval call binding the contract method 0x15e812ad.
//
// Solidity: function getBaseFee() view returns(uint256)
func (_Basefee *BasefeeCaller) GetBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Basefee.contract.Call(opts, &out, "getBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBaseFee is a free data retrieval call binding the contract method 0x15e812ad.
//
// Solidity: function getBaseFee() view returns(uint256)
func (_Basefee *BasefeeSession) GetBaseFee() (*big.Int, error) {
	return _Basefee.Contract.GetBaseFee(&_Basefee.CallOpts)
}

// GetBaseFee is a free data retrieval call binding the contract method 0x15e812ad.
//
// Solidity: function getBaseFee() view returns(uint256)
func (_Basefee *BasefeeCallerSession) GetBaseFee() (*big.Int, error) {
	return _Basefee.Contract.GetBaseFee(&_Basefee.CallOpts)
}

// LogCurrentBaseFee is a paid mutator transaction binding the contract method 0x414783bf.
//
// Solidity: function logCurrentBaseFee() returns()
func (_Basefee *BasefeeTransactor) LogCurrentBaseFee(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Basefee.contract.Transact(opts, "logCurrentBaseFee")
}

// LogCurrentBaseFee is a paid mutator transaction binding the contract method 0x414783bf.
//
// Solidity: function logCurrentBaseFee() returns()
func (_Basefee *BasefeeSession) LogCurrentBaseFee() (*types.Transaction, error) {
	return _Basefee.Contract.LogCurrentBaseFee(&_Basefee.TransactOpts)
}

// LogCurrentBaseFee is a paid mutator transaction binding the contract method 0x414783bf.
//
// Solidity: function logCurrentBaseFee() returns()
func (_Basefee *BasefeeTransactorSession) LogCurrentBaseFee() (*types.Transaction, error) {
	return _Basefee.Contract.LogCurrentBaseFee(&_Basefee.TransactOpts)
}

// BasefeeCurrentFeeIterator is returned from FilterCurrentFee and is used to iterate over the raw logs and unpacked data for CurrentFee events raised by the Basefee contract.
type BasefeeCurrentFeeIterator struct {
	Event *BasefeeCurrentFee // Event containing the contract specifics and raw log

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
func (it *BasefeeCurrentFeeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BasefeeCurrentFee)
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
		it.Event = new(BasefeeCurrentFee)
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
func (it *BasefeeCurrentFeeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BasefeeCurrentFeeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BasefeeCurrentFee represents a CurrentFee event raised by the Basefee contract.
type BasefeeCurrentFee struct {
	Fee *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterCurrentFee is a free log retrieval operation binding the contract event 0xd25c41149a2ac969bcb91bf41383d6aa5f5246c7586901cc546e7e2361cfb667.
//
// Solidity: event CurrentFee(uint256 fee)
func (_Basefee *BasefeeFilterer) FilterCurrentFee(opts *bind.FilterOpts) (*BasefeeCurrentFeeIterator, error) {

	logs, sub, err := _Basefee.contract.FilterLogs(opts, "CurrentFee")
	if err != nil {
		return nil, err
	}
	return &BasefeeCurrentFeeIterator{contract: _Basefee.contract, event: "CurrentFee", logs: logs, sub: sub}, nil
}

// WatchCurrentFee is a free log subscription operation binding the contract event 0xd25c41149a2ac969bcb91bf41383d6aa5f5246c7586901cc546e7e2361cfb667.
//
// Solidity: event CurrentFee(uint256 fee)
func (_Basefee *BasefeeFilterer) WatchCurrentFee(opts *bind.WatchOpts, sink chan<- *BasefeeCurrentFee) (event.Subscription, error) {

	logs, sub, err := _Basefee.contract.WatchLogs(opts, "CurrentFee")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BasefeeCurrentFee)
				if err := _Basefee.contract.UnpackLog(event, "CurrentFee", log); err != nil {
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

// ParseCurrentFee is a log parse operation binding the contract event 0xd25c41149a2ac969bcb91bf41383d6aa5f5246c7586901cc546e7e2361cfb667.
//
// Solidity: event CurrentFee(uint256 fee)
func (_Basefee *BasefeeFilterer) ParseCurrentFee(log types.Log) (*BasefeeCurrentFee, error) {
	event := new(BasefeeCurrentFee)
	if err := _Basefee.contract.UnpackLog(event, "CurrentFee", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
