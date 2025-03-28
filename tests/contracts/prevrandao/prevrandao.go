// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package prevrandao

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

// PrevrandaoMetaData contains all meta data concerning the Prevrandao contract.
var PrevrandaoMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"prevrandao\",\"type\":\"uint256\"}],\"name\":\"CurrentPrevRandao\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getPrevRandao\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"logCurrentPrevRandao\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5060f78061001b5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c806380b802a9146034578063f4c3a9b814603c575b5f5ffd5b603a6056565b005b6042608d565b604051604d919060aa565b60405180910390f35b7f414363e7c6a88d9affe1d1c4b2a6f3423f30f1a2cf70c9da900aa53ee333290f446040516083919060aa565b60405180910390a1565b5f44905090565b5f819050919050565b60a4816094565b82525050565b5f60208201905060bb5f830184609d565b9291505056fea26469706673582212207689705309519928e7d845fe9955973ad5d4c5f3ba68f90f847c8cb20c6a5ade64736f6c634300081c0033",
}

// PrevrandaoABI is the input ABI used to generate the binding from.
// Deprecated: Use PrevrandaoMetaData.ABI instead.
var PrevrandaoABI = PrevrandaoMetaData.ABI

// PrevrandaoBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PrevrandaoMetaData.Bin instead.
var PrevrandaoBin = PrevrandaoMetaData.Bin

// DeployPrevrandao deploys a new Ethereum contract, binding an instance of Prevrandao to it.
func DeployPrevrandao(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Prevrandao, error) {
	parsed, err := PrevrandaoMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PrevrandaoBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Prevrandao{PrevrandaoCaller: PrevrandaoCaller{contract: contract}, PrevrandaoTransactor: PrevrandaoTransactor{contract: contract}, PrevrandaoFilterer: PrevrandaoFilterer{contract: contract}}, nil
}

// Prevrandao is an auto generated Go binding around an Ethereum contract.
type Prevrandao struct {
	PrevrandaoCaller     // Read-only binding to the contract
	PrevrandaoTransactor // Write-only binding to the contract
	PrevrandaoFilterer   // Log filterer for contract events
}

// PrevrandaoCaller is an auto generated read-only Go binding around an Ethereum contract.
type PrevrandaoCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PrevrandaoTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PrevrandaoTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PrevrandaoFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PrevrandaoFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PrevrandaoSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PrevrandaoSession struct {
	Contract     *Prevrandao       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PrevrandaoCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PrevrandaoCallerSession struct {
	Contract *PrevrandaoCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// PrevrandaoTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PrevrandaoTransactorSession struct {
	Contract     *PrevrandaoTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// PrevrandaoRaw is an auto generated low-level Go binding around an Ethereum contract.
type PrevrandaoRaw struct {
	Contract *Prevrandao // Generic contract binding to access the raw methods on
}

// PrevrandaoCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PrevrandaoCallerRaw struct {
	Contract *PrevrandaoCaller // Generic read-only contract binding to access the raw methods on
}

// PrevrandaoTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PrevrandaoTransactorRaw struct {
	Contract *PrevrandaoTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPrevrandao creates a new instance of Prevrandao, bound to a specific deployed contract.
func NewPrevrandao(address common.Address, backend bind.ContractBackend) (*Prevrandao, error) {
	contract, err := bindPrevrandao(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Prevrandao{PrevrandaoCaller: PrevrandaoCaller{contract: contract}, PrevrandaoTransactor: PrevrandaoTransactor{contract: contract}, PrevrandaoFilterer: PrevrandaoFilterer{contract: contract}}, nil
}

// NewPrevrandaoCaller creates a new read-only instance of Prevrandao, bound to a specific deployed contract.
func NewPrevrandaoCaller(address common.Address, caller bind.ContractCaller) (*PrevrandaoCaller, error) {
	contract, err := bindPrevrandao(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PrevrandaoCaller{contract: contract}, nil
}

// NewPrevrandaoTransactor creates a new write-only instance of Prevrandao, bound to a specific deployed contract.
func NewPrevrandaoTransactor(address common.Address, transactor bind.ContractTransactor) (*PrevrandaoTransactor, error) {
	contract, err := bindPrevrandao(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PrevrandaoTransactor{contract: contract}, nil
}

// NewPrevrandaoFilterer creates a new log filterer instance of Prevrandao, bound to a specific deployed contract.
func NewPrevrandaoFilterer(address common.Address, filterer bind.ContractFilterer) (*PrevrandaoFilterer, error) {
	contract, err := bindPrevrandao(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PrevrandaoFilterer{contract: contract}, nil
}

// bindPrevrandao binds a generic wrapper to an already deployed contract.
func bindPrevrandao(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PrevrandaoMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Prevrandao *PrevrandaoRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Prevrandao.Contract.PrevrandaoCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Prevrandao *PrevrandaoRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Prevrandao.Contract.PrevrandaoTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Prevrandao *PrevrandaoRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Prevrandao.Contract.PrevrandaoTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Prevrandao *PrevrandaoCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Prevrandao.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Prevrandao *PrevrandaoTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Prevrandao.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Prevrandao *PrevrandaoTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Prevrandao.Contract.contract.Transact(opts, method, params...)
}

// GetPrevRandao is a free data retrieval call binding the contract method 0xf4c3a9b8.
//
// Solidity: function getPrevRandao() view returns(uint256)
func (_Prevrandao *PrevrandaoCaller) GetPrevRandao(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Prevrandao.contract.Call(opts, &out, "getPrevRandao")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPrevRandao is a free data retrieval call binding the contract method 0xf4c3a9b8.
//
// Solidity: function getPrevRandao() view returns(uint256)
func (_Prevrandao *PrevrandaoSession) GetPrevRandao() (*big.Int, error) {
	return _Prevrandao.Contract.GetPrevRandao(&_Prevrandao.CallOpts)
}

// GetPrevRandao is a free data retrieval call binding the contract method 0xf4c3a9b8.
//
// Solidity: function getPrevRandao() view returns(uint256)
func (_Prevrandao *PrevrandaoCallerSession) GetPrevRandao() (*big.Int, error) {
	return _Prevrandao.Contract.GetPrevRandao(&_Prevrandao.CallOpts)
}

// LogCurrentPrevRandao is a paid mutator transaction binding the contract method 0x80b802a9.
//
// Solidity: function logCurrentPrevRandao() returns()
func (_Prevrandao *PrevrandaoTransactor) LogCurrentPrevRandao(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Prevrandao.contract.Transact(opts, "logCurrentPrevRandao")
}

// LogCurrentPrevRandao is a paid mutator transaction binding the contract method 0x80b802a9.
//
// Solidity: function logCurrentPrevRandao() returns()
func (_Prevrandao *PrevrandaoSession) LogCurrentPrevRandao() (*types.Transaction, error) {
	return _Prevrandao.Contract.LogCurrentPrevRandao(&_Prevrandao.TransactOpts)
}

// LogCurrentPrevRandao is a paid mutator transaction binding the contract method 0x80b802a9.
//
// Solidity: function logCurrentPrevRandao() returns()
func (_Prevrandao *PrevrandaoTransactorSession) LogCurrentPrevRandao() (*types.Transaction, error) {
	return _Prevrandao.Contract.LogCurrentPrevRandao(&_Prevrandao.TransactOpts)
}

// PrevrandaoCurrentPrevRandaoIterator is returned from FilterCurrentPrevRandao and is used to iterate over the raw logs and unpacked data for CurrentPrevRandao events raised by the Prevrandao contract.
type PrevrandaoCurrentPrevRandaoIterator struct {
	Event *PrevrandaoCurrentPrevRandao // Event containing the contract specifics and raw log

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
func (it *PrevrandaoCurrentPrevRandaoIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PrevrandaoCurrentPrevRandao)
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
		it.Event = new(PrevrandaoCurrentPrevRandao)
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
func (it *PrevrandaoCurrentPrevRandaoIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PrevrandaoCurrentPrevRandaoIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PrevrandaoCurrentPrevRandao represents a CurrentPrevRandao event raised by the Prevrandao contract.
type PrevrandaoCurrentPrevRandao struct {
	Prevrandao *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterCurrentPrevRandao is a free log retrieval operation binding the contract event 0x414363e7c6a88d9affe1d1c4b2a6f3423f30f1a2cf70c9da900aa53ee333290f.
//
// Solidity: event CurrentPrevRandao(uint256 prevrandao)
func (_Prevrandao *PrevrandaoFilterer) FilterCurrentPrevRandao(opts *bind.FilterOpts) (*PrevrandaoCurrentPrevRandaoIterator, error) {

	logs, sub, err := _Prevrandao.contract.FilterLogs(opts, "CurrentPrevRandao")
	if err != nil {
		return nil, err
	}
	return &PrevrandaoCurrentPrevRandaoIterator{contract: _Prevrandao.contract, event: "CurrentPrevRandao", logs: logs, sub: sub}, nil
}

// WatchCurrentPrevRandao is a free log subscription operation binding the contract event 0x414363e7c6a88d9affe1d1c4b2a6f3423f30f1a2cf70c9da900aa53ee333290f.
//
// Solidity: event CurrentPrevRandao(uint256 prevrandao)
func (_Prevrandao *PrevrandaoFilterer) WatchCurrentPrevRandao(opts *bind.WatchOpts, sink chan<- *PrevrandaoCurrentPrevRandao) (event.Subscription, error) {

	logs, sub, err := _Prevrandao.contract.WatchLogs(opts, "CurrentPrevRandao")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PrevrandaoCurrentPrevRandao)
				if err := _Prevrandao.contract.UnpackLog(event, "CurrentPrevRandao", log); err != nil {
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

// ParseCurrentPrevRandao is a log parse operation binding the contract event 0x414363e7c6a88d9affe1d1c4b2a6f3423f30f1a2cf70c9da900aa53ee333290f.
//
// Solidity: event CurrentPrevRandao(uint256 prevrandao)
func (_Prevrandao *PrevrandaoFilterer) ParseCurrentPrevRandao(log types.Log) (*PrevrandaoCurrentPrevRandao, error) {
	event := new(PrevrandaoCurrentPrevRandao)
	if err := _Prevrandao.contract.UnpackLog(event, "CurrentPrevRandao", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
