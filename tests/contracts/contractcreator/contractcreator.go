// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contractcreator

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

// ContractcreatorMetaData contains all meta data concerning the Contractcreator contract.
var ContractcreatorMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"cost\",\"type\":\"uint256\"}],\"name\":\"LogCost\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"codeSize\",\"type\":\"uint256\"}],\"name\":\"Create2With\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"codeSize\",\"type\":\"uint256\"}],\"name\":\"CreatetWith\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"someValue\",\"type\":\"uint256\"}],\"name\":\"GetOverheadCost\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b506102d18061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061003f575f3560e01c80632870387414610043578063aa547d881461005f578063cdcfe0cd1461007b575b5f80fd5b61005d600480360381019061005891906101c2565b610097565b005b610079600480360381019061007491906101c2565b6100e6565b005b610095600480360381019061009091906101c2565b610139565b005b5f5a90505f8290507fc3263769c8cc487b9d31817141d5ff9ce159867184d644cc2b238d6c54619fdf5a836100cc919061021a565b6040516100d9919061025c565b60405180910390a1505050565b5f5a90505f80835f80f590507fc3263769c8cc487b9d31817141d5ff9ce159867184d644cc2b238d6c54619fdf5a8361011f919061021a565b60405161012c919061025c565b60405180910390a1505050565b5f5a90505f825f80f090507fc3263769c8cc487b9d31817141d5ff9ce159867184d644cc2b238d6c54619fdf5a83610171919061021a565b60405161017e919061025c565b60405180910390a1505050565b5f80fd5b5f819050919050565b6101a18161018f565b81146101ab575f80fd5b50565b5f813590506101bc81610198565b92915050565b5f602082840312156101d7576101d661018b565b5b5f6101e4848285016101ae565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6102248261018f565b915061022f8361018f565b9250828203905081811115610247576102466101ed565b5b92915050565b6102568161018f565b82525050565b5f60208201905061026f5f83018461024d565b9291505056fea26469706673582212201536169bd6f69ddc7403da1ca5080b790a40e2c111a8ff6a18aae1af6b0eccc864736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
}

// ContractcreatorABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractcreatorMetaData.ABI instead.
var ContractcreatorABI = ContractcreatorMetaData.ABI

// ContractcreatorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ContractcreatorMetaData.Bin instead.
var ContractcreatorBin = ContractcreatorMetaData.Bin

// DeployContractcreator deploys a new Ethereum contract, binding an instance of Contractcreator to it.
func DeployContractcreator(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Contractcreator, error) {
	parsed, err := ContractcreatorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ContractcreatorBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Contractcreator{ContractcreatorCaller: ContractcreatorCaller{contract: contract}, ContractcreatorTransactor: ContractcreatorTransactor{contract: contract}, ContractcreatorFilterer: ContractcreatorFilterer{contract: contract}}, nil
}

// Contractcreator is an auto generated Go binding around an Ethereum contract.
type Contractcreator struct {
	ContractcreatorCaller     // Read-only binding to the contract
	ContractcreatorTransactor // Write-only binding to the contract
	ContractcreatorFilterer   // Log filterer for contract events
}

// ContractcreatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractcreatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractcreatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractcreatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractcreatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractcreatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractcreatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractcreatorSession struct {
	Contract     *Contractcreator  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractcreatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractcreatorCallerSession struct {
	Contract *ContractcreatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ContractcreatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractcreatorTransactorSession struct {
	Contract     *ContractcreatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ContractcreatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractcreatorRaw struct {
	Contract *Contractcreator // Generic contract binding to access the raw methods on
}

// ContractcreatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractcreatorCallerRaw struct {
	Contract *ContractcreatorCaller // Generic read-only contract binding to access the raw methods on
}

// ContractcreatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractcreatorTransactorRaw struct {
	Contract *ContractcreatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContractcreator creates a new instance of Contractcreator, bound to a specific deployed contract.
func NewContractcreator(address common.Address, backend bind.ContractBackend) (*Contractcreator, error) {
	contract, err := bindContractcreator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contractcreator{ContractcreatorCaller: ContractcreatorCaller{contract: contract}, ContractcreatorTransactor: ContractcreatorTransactor{contract: contract}, ContractcreatorFilterer: ContractcreatorFilterer{contract: contract}}, nil
}

// NewContractcreatorCaller creates a new read-only instance of Contractcreator, bound to a specific deployed contract.
func NewContractcreatorCaller(address common.Address, caller bind.ContractCaller) (*ContractcreatorCaller, error) {
	contract, err := bindContractcreator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractcreatorCaller{contract: contract}, nil
}

// NewContractcreatorTransactor creates a new write-only instance of Contractcreator, bound to a specific deployed contract.
func NewContractcreatorTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractcreatorTransactor, error) {
	contract, err := bindContractcreator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractcreatorTransactor{contract: contract}, nil
}

// NewContractcreatorFilterer creates a new log filterer instance of Contractcreator, bound to a specific deployed contract.
func NewContractcreatorFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractcreatorFilterer, error) {
	contract, err := bindContractcreator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractcreatorFilterer{contract: contract}, nil
}

// bindContractcreator binds a generic wrapper to an already deployed contract.
func bindContractcreator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ContractcreatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contractcreator *ContractcreatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contractcreator.Contract.ContractcreatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contractcreator *ContractcreatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contractcreator.Contract.ContractcreatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contractcreator *ContractcreatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contractcreator.Contract.ContractcreatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contractcreator *ContractcreatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contractcreator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contractcreator *ContractcreatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contractcreator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contractcreator *ContractcreatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contractcreator.Contract.contract.Transact(opts, method, params...)
}

// Create2With is a paid mutator transaction binding the contract method 0xaa547d88.
//
// Solidity: function Create2With(uint256 codeSize) returns()
func (_Contractcreator *ContractcreatorTransactor) Create2With(opts *bind.TransactOpts, codeSize *big.Int) (*types.Transaction, error) {
	return _Contractcreator.contract.Transact(opts, "Create2With", codeSize)
}

// Create2With is a paid mutator transaction binding the contract method 0xaa547d88.
//
// Solidity: function Create2With(uint256 codeSize) returns()
func (_Contractcreator *ContractcreatorSession) Create2With(codeSize *big.Int) (*types.Transaction, error) {
	return _Contractcreator.Contract.Create2With(&_Contractcreator.TransactOpts, codeSize)
}

// Create2With is a paid mutator transaction binding the contract method 0xaa547d88.
//
// Solidity: function Create2With(uint256 codeSize) returns()
func (_Contractcreator *ContractcreatorTransactorSession) Create2With(codeSize *big.Int) (*types.Transaction, error) {
	return _Contractcreator.Contract.Create2With(&_Contractcreator.TransactOpts, codeSize)
}

// CreatetWith is a paid mutator transaction binding the contract method 0xcdcfe0cd.
//
// Solidity: function CreatetWith(uint256 codeSize) returns()
func (_Contractcreator *ContractcreatorTransactor) CreatetWith(opts *bind.TransactOpts, codeSize *big.Int) (*types.Transaction, error) {
	return _Contractcreator.contract.Transact(opts, "CreatetWith", codeSize)
}

// CreatetWith is a paid mutator transaction binding the contract method 0xcdcfe0cd.
//
// Solidity: function CreatetWith(uint256 codeSize) returns()
func (_Contractcreator *ContractcreatorSession) CreatetWith(codeSize *big.Int) (*types.Transaction, error) {
	return _Contractcreator.Contract.CreatetWith(&_Contractcreator.TransactOpts, codeSize)
}

// CreatetWith is a paid mutator transaction binding the contract method 0xcdcfe0cd.
//
// Solidity: function CreatetWith(uint256 codeSize) returns()
func (_Contractcreator *ContractcreatorTransactorSession) CreatetWith(codeSize *big.Int) (*types.Transaction, error) {
	return _Contractcreator.Contract.CreatetWith(&_Contractcreator.TransactOpts, codeSize)
}

// GetOverheadCost is a paid mutator transaction binding the contract method 0x28703874.
//
// Solidity: function GetOverheadCost(uint256 someValue) returns()
func (_Contractcreator *ContractcreatorTransactor) GetOverheadCost(opts *bind.TransactOpts, someValue *big.Int) (*types.Transaction, error) {
	return _Contractcreator.contract.Transact(opts, "GetOverheadCost", someValue)
}

// GetOverheadCost is a paid mutator transaction binding the contract method 0x28703874.
//
// Solidity: function GetOverheadCost(uint256 someValue) returns()
func (_Contractcreator *ContractcreatorSession) GetOverheadCost(someValue *big.Int) (*types.Transaction, error) {
	return _Contractcreator.Contract.GetOverheadCost(&_Contractcreator.TransactOpts, someValue)
}

// GetOverheadCost is a paid mutator transaction binding the contract method 0x28703874.
//
// Solidity: function GetOverheadCost(uint256 someValue) returns()
func (_Contractcreator *ContractcreatorTransactorSession) GetOverheadCost(someValue *big.Int) (*types.Transaction, error) {
	return _Contractcreator.Contract.GetOverheadCost(&_Contractcreator.TransactOpts, someValue)
}

// ContractcreatorLogCostIterator is returned from FilterLogCost and is used to iterate over the raw logs and unpacked data for LogCost events raised by the Contractcreator contract.
type ContractcreatorLogCostIterator struct {
	Event *ContractcreatorLogCost // Event containing the contract specifics and raw log

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
func (it *ContractcreatorLogCostIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractcreatorLogCost)
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
		it.Event = new(ContractcreatorLogCost)
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
func (it *ContractcreatorLogCostIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractcreatorLogCostIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractcreatorLogCost represents a LogCost event raised by the Contractcreator contract.
type ContractcreatorLogCost struct {
	Cost *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterLogCost is a free log retrieval operation binding the contract event 0xc3263769c8cc487b9d31817141d5ff9ce159867184d644cc2b238d6c54619fdf.
//
// Solidity: event LogCost(uint256 cost)
func (_Contractcreator *ContractcreatorFilterer) FilterLogCost(opts *bind.FilterOpts) (*ContractcreatorLogCostIterator, error) {

	logs, sub, err := _Contractcreator.contract.FilterLogs(opts, "LogCost")
	if err != nil {
		return nil, err
	}
	return &ContractcreatorLogCostIterator{contract: _Contractcreator.contract, event: "LogCost", logs: logs, sub: sub}, nil
}

// WatchLogCost is a free log subscription operation binding the contract event 0xc3263769c8cc487b9d31817141d5ff9ce159867184d644cc2b238d6c54619fdf.
//
// Solidity: event LogCost(uint256 cost)
func (_Contractcreator *ContractcreatorFilterer) WatchLogCost(opts *bind.WatchOpts, sink chan<- *ContractcreatorLogCost) (event.Subscription, error) {

	logs, sub, err := _Contractcreator.contract.WatchLogs(opts, "LogCost")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractcreatorLogCost)
				if err := _Contractcreator.contract.UnpackLog(event, "LogCost", log); err != nil {
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

// ParseLogCost is a log parse operation binding the contract event 0xc3263769c8cc487b9d31817141d5ff9ce159867184d644cc2b238d6c54619fdf.
//
// Solidity: event LogCost(uint256 cost)
func (_Contractcreator *ContractcreatorFilterer) ParseLogCost(log types.Log) (*ContractcreatorLogCost, error) {
	event := new(ContractcreatorLogCost)
	if err := _Contractcreator.contract.UnpackLog(event, "LogCost", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
