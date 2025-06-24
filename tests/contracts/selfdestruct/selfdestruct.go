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

// SelfDestructMetaData contains all meta data concerning the SelfDestruct contract.
var SelfDestructMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"destroyNow\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"selfBeneficiary\",\"type\":\"bool\"},{\"internalType\":\"addresspayable\",\"name\":\"recipient\",\"type\":\"address\"}],\"stateMutability\":\"payable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"LogAfterDestruct\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"selfBeneficiary\",\"type\":\"bool\"},{\"internalType\":\"addresspayable\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"destroyContract\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"someData\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052607b5f55604051610353380380610353833981810160405281019061002991906100fe565b82156100405761003f828261004860201b60201c565b5b50505061014e565b8115610052573090505b8073ffffffffffffffffffffffffffffffffffffffff16ff5b5f80fd5b5f8115159050919050565b6100838161006f565b811461008d575f80fd5b50565b5f8151905061009e8161007a565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6100cd826100a4565b9050919050565b6100dd816100c3565b81146100e7575f80fd5b50565b5f815190506100f8816100d4565b92915050565b5f805f606084860312156101155761011461006b565b5b5f61012286828701610090565b935050602061013386828701610090565b9250506040610144868287016100ea565b9150509250925092565b6101f88061015b5f395ff3fe608060405234801561000f575f80fd5b5060043610610034575f3560e01c8063c73b717514610038578063ca29aee914610056575b5f80fd5b610040610072565b60405161004d91906100b2565b60405180910390f35b610070600480360381019061006b919061015e565b610077565b005b5f5481565b8115610081573090505b8073ffffffffffffffffffffffffffffffffffffffff16ff5b5f819050919050565b6100ac8161009a565b82525050565b5f6020820190506100c55f8301846100a3565b92915050565b5f80fd5b5f8115159050919050565b6100e3816100cf565b81146100ed575f80fd5b50565b5f813590506100fe816100da565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61012d82610104565b9050919050565b61013d81610123565b8114610147575f80fd5b50565b5f8135905061015881610134565b92915050565b5f8060408385031215610174576101736100cb565b5b5f610181858286016100f0565b92505060206101928582860161014a565b915050925092905056fea2646970667358221220f55239a0719f2123be397b9a4d1f00b2ec9cba1f516fac3e6c7379a5f4dd007064736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
}

// SelfDestructABI is the input ABI used to generate the binding from.
// Deprecated: Use SelfDestructMetaData.ABI instead.
var SelfDestructABI = SelfDestructMetaData.ABI

// SelfDestructBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SelfDestructMetaData.Bin instead.
var SelfDestructBin = SelfDestructMetaData.Bin

// DeploySelfDestruct deploys a new Ethereum contract, binding an instance of SelfDestruct to it.
func DeploySelfDestruct(auth *bind.TransactOpts, backend bind.ContractBackend, destroyNow bool, selfBeneficiary bool, recipient common.Address) (common.Address, *types.Transaction, *SelfDestruct, error) {
	parsed, err := SelfDestructMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SelfDestructBin), backend, destroyNow, selfBeneficiary, recipient)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SelfDestruct{SelfDestructCaller: SelfDestructCaller{contract: contract}, SelfDestructTransactor: SelfDestructTransactor{contract: contract}, SelfDestructFilterer: SelfDestructFilterer{contract: contract}}, nil
}

// SelfDestruct is an auto generated Go binding around an Ethereum contract.
type SelfDestruct struct {
	SelfDestructCaller     // Read-only binding to the contract
	SelfDestructTransactor // Write-only binding to the contract
	SelfDestructFilterer   // Log filterer for contract events
}

// SelfDestructCaller is an auto generated read-only Go binding around an Ethereum contract.
type SelfDestructCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfDestructTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SelfDestructTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfDestructFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SelfDestructFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SelfDestructSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SelfDestructSession struct {
	Contract     *SelfDestruct     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SelfDestructCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SelfDestructCallerSession struct {
	Contract *SelfDestructCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// SelfDestructTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SelfDestructTransactorSession struct {
	Contract     *SelfDestructTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SelfDestructRaw is an auto generated low-level Go binding around an Ethereum contract.
type SelfDestructRaw struct {
	Contract *SelfDestruct // Generic contract binding to access the raw methods on
}

// SelfDestructCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SelfDestructCallerRaw struct {
	Contract *SelfDestructCaller // Generic read-only contract binding to access the raw methods on
}

// SelfDestructTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SelfDestructTransactorRaw struct {
	Contract *SelfDestructTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSelfDestruct creates a new instance of SelfDestruct, bound to a specific deployed contract.
func NewSelfDestruct(address common.Address, backend bind.ContractBackend) (*SelfDestruct, error) {
	contract, err := bindSelfDestruct(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SelfDestruct{SelfDestructCaller: SelfDestructCaller{contract: contract}, SelfDestructTransactor: SelfDestructTransactor{contract: contract}, SelfDestructFilterer: SelfDestructFilterer{contract: contract}}, nil
}

// NewSelfDestructCaller creates a new read-only instance of SelfDestruct, bound to a specific deployed contract.
func NewSelfDestructCaller(address common.Address, caller bind.ContractCaller) (*SelfDestructCaller, error) {
	contract, err := bindSelfDestruct(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SelfDestructCaller{contract: contract}, nil
}

// NewSelfDestructTransactor creates a new write-only instance of SelfDestruct, bound to a specific deployed contract.
func NewSelfDestructTransactor(address common.Address, transactor bind.ContractTransactor) (*SelfDestructTransactor, error) {
	contract, err := bindSelfDestruct(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SelfDestructTransactor{contract: contract}, nil
}

// NewSelfDestructFilterer creates a new log filterer instance of SelfDestruct, bound to a specific deployed contract.
func NewSelfDestructFilterer(address common.Address, filterer bind.ContractFilterer) (*SelfDestructFilterer, error) {
	contract, err := bindSelfDestruct(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SelfDestructFilterer{contract: contract}, nil
}

// bindSelfDestruct binds a generic wrapper to an already deployed contract.
func bindSelfDestruct(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SelfDestructMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SelfDestruct *SelfDestructRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SelfDestruct.Contract.SelfDestructCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SelfDestruct *SelfDestructRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestruct.Contract.SelfDestructTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SelfDestruct *SelfDestructRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SelfDestruct.Contract.SelfDestructTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SelfDestruct *SelfDestructCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SelfDestruct.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SelfDestruct *SelfDestructTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SelfDestruct.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SelfDestruct *SelfDestructTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SelfDestruct.Contract.contract.Transact(opts, method, params...)
}

// SomeData is a free data retrieval call binding the contract method 0xc73b7175.
//
// Solidity: function someData() view returns(uint256)
func (_SelfDestruct *SelfDestructCaller) SomeData(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SelfDestruct.contract.Call(opts, &out, "someData")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SomeData is a free data retrieval call binding the contract method 0xc73b7175.
//
// Solidity: function someData() view returns(uint256)
func (_SelfDestruct *SelfDestructSession) SomeData() (*big.Int, error) {
	return _SelfDestruct.Contract.SomeData(&_SelfDestruct.CallOpts)
}

// SomeData is a free data retrieval call binding the contract method 0xc73b7175.
//
// Solidity: function someData() view returns(uint256)
func (_SelfDestruct *SelfDestructCallerSession) SomeData() (*big.Int, error) {
	return _SelfDestruct.Contract.SomeData(&_SelfDestruct.CallOpts)
}

// DestroyContract is a paid mutator transaction binding the contract method 0xca29aee9.
//
// Solidity: function destroyContract(bool selfBeneficiary, address recipient) returns()
func (_SelfDestruct *SelfDestructTransactor) DestroyContract(opts *bind.TransactOpts, selfBeneficiary bool, recipient common.Address) (*types.Transaction, error) {
	return _SelfDestruct.contract.Transact(opts, "destroyContract", selfBeneficiary, recipient)
}

// DestroyContract is a paid mutator transaction binding the contract method 0xca29aee9.
//
// Solidity: function destroyContract(bool selfBeneficiary, address recipient) returns()
func (_SelfDestruct *SelfDestructSession) DestroyContract(selfBeneficiary bool, recipient common.Address) (*types.Transaction, error) {
	return _SelfDestruct.Contract.DestroyContract(&_SelfDestruct.TransactOpts, selfBeneficiary, recipient)
}

// DestroyContract is a paid mutator transaction binding the contract method 0xca29aee9.
//
// Solidity: function destroyContract(bool selfBeneficiary, address recipient) returns()
func (_SelfDestruct *SelfDestructTransactorSession) DestroyContract(selfBeneficiary bool, recipient common.Address) (*types.Transaction, error) {
	return _SelfDestruct.Contract.DestroyContract(&_SelfDestruct.TransactOpts, selfBeneficiary, recipient)
}

// SelfDestructLogAfterDestructIterator is returned from FilterLogAfterDestruct and is used to iterate over the raw logs and unpacked data for LogAfterDestruct events raised by the SelfDestruct contract.
type SelfDestructLogAfterDestructIterator struct {
	Event *SelfDestructLogAfterDestruct // Event containing the contract specifics and raw log

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
func (it *SelfDestructLogAfterDestructIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SelfDestructLogAfterDestruct)
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
		it.Event = new(SelfDestructLogAfterDestruct)
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
func (it *SelfDestructLogAfterDestructIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SelfDestructLogAfterDestructIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SelfDestructLogAfterDestruct represents a LogAfterDestruct event raised by the SelfDestruct contract.
type SelfDestructLogAfterDestruct struct {
	Balance *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterLogAfterDestruct is a free log retrieval operation binding the contract event 0xf91cb366864fbc31224338e7a7493dfb408ce511551c30da5895d780d726a9b5.
//
// Solidity: event LogAfterDestruct(uint256 balance)
func (_SelfDestruct *SelfDestructFilterer) FilterLogAfterDestruct(opts *bind.FilterOpts) (*SelfDestructLogAfterDestructIterator, error) {

	logs, sub, err := _SelfDestruct.contract.FilterLogs(opts, "LogAfterDestruct")
	if err != nil {
		return nil, err
	}
	return &SelfDestructLogAfterDestructIterator{contract: _SelfDestruct.contract, event: "LogAfterDestruct", logs: logs, sub: sub}, nil
}

// WatchLogAfterDestruct is a free log subscription operation binding the contract event 0xf91cb366864fbc31224338e7a7493dfb408ce511551c30da5895d780d726a9b5.
//
// Solidity: event LogAfterDestruct(uint256 balance)
func (_SelfDestruct *SelfDestructFilterer) WatchLogAfterDestruct(opts *bind.WatchOpts, sink chan<- *SelfDestructLogAfterDestruct) (event.Subscription, error) {

	logs, sub, err := _SelfDestruct.contract.WatchLogs(opts, "LogAfterDestruct")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SelfDestructLogAfterDestruct)
				if err := _SelfDestruct.contract.UnpackLog(event, "LogAfterDestruct", log); err != nil {
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

// ParseLogAfterDestruct is a log parse operation binding the contract event 0xf91cb366864fbc31224338e7a7493dfb408ce511551c30da5895d780d726a9b5.
//
// Solidity: event LogAfterDestruct(uint256 balance)
func (_SelfDestruct *SelfDestructFilterer) ParseLogAfterDestruct(log types.Log) (*SelfDestructLogAfterDestruct, error) {
	event := new(SelfDestructLogAfterDestruct)
	if err := _SelfDestruct.contract.UnpackLog(event, "LogAfterDestruct", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
