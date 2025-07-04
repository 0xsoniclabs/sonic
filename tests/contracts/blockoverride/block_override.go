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

package block_override

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

// BlockOverrideMetaData contains all meta data concerning the BlockOverride contract.
var BlockOverrideMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"currentBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"observedBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"Seen\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nr\",\"type\":\"uint256\"}],\"name\":\"getBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBlockParameters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"difficulty\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gaslimit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"random\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"basefee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blobbasefee\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"observe\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5061040c8061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c806314fc78fc14610043578063a3289b771461004d578063ee82ac5e14610072575b5f5ffd5b61004b6100a2565b005b610055610136565b6040516100699897969594939291906101c1565b60405180910390f35b61008c6004803603810190610087919061026b565b610160565b60405161009991906102ae565b60405180910390f35b5f5f90505f6005436100b491906102f4565b90506101048111156100d15761010e816100ce9190610327565b91505b5f8290505b818111610131575f814090507f2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db746964383836040516101159392919061035a565b60405180910390a15080806101299061038f565b9150506100d6565b505050565b5f5f5f5f5f5f5f5f4397504496504295504594504193504492504891504a90509091929394959697565b5f81409050919050565b5f819050919050565b61017c8161016a565b82525050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6101ab82610182565b9050919050565b6101bb816101a1565b82525050565b5f610100820190506101d55f83018b610173565b6101e2602083018a610173565b6101ef6040830189610173565b6101fc6060830188610173565b61020960808301876101b2565b61021660a0830186610173565b61022360c0830185610173565b61023060e0830184610173565b9998505050505050505050565b5f5ffd5b61024a8161016a565b8114610254575f5ffd5b50565b5f8135905061026581610241565b92915050565b5f602082840312156102805761027f61023d565b5b5f61028d84828501610257565b91505092915050565b5f819050919050565b6102a881610296565b82525050565b5f6020820190506102c15f83018461029f565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6102fe8261016a565b91506103098361016a565b9250828201905080821115610321576103206102c7565b5b92915050565b5f6103318261016a565b915061033c8361016a565b9250828203905081811115610354576103536102c7565b5b92915050565b5f60608201905061036d5f830186610173565b61037a6020830185610173565b610387604083018461029f565b949350505050565b5f6103998261016a565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036103cb576103ca6102c7565b5b60018201905091905056fea264697066735822122072246b84b78c5095d5c6e5ae7f4dcde5e38b87274061ad8aab6af7a5d2b0d21f64736f6c634300081c0033",
}

// BlockOverrideABI is the input ABI used to generate the binding from.
// Deprecated: Use BlockOverrideMetaData.ABI instead.
var BlockOverrideABI = BlockOverrideMetaData.ABI

// BlockOverrideBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BlockOverrideMetaData.Bin instead.
var BlockOverrideBin = BlockOverrideMetaData.Bin

// DeployBlockOverride deploys a new Ethereum contract, binding an instance of BlockOverride to it.
func DeployBlockOverride(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BlockOverride, error) {
	parsed, err := BlockOverrideMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BlockOverrideBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BlockOverride{BlockOverrideCaller: BlockOverrideCaller{contract: contract}, BlockOverrideTransactor: BlockOverrideTransactor{contract: contract}, BlockOverrideFilterer: BlockOverrideFilterer{contract: contract}}, nil
}

// BlockOverride is an auto generated Go binding around an Ethereum contract.
type BlockOverride struct {
	BlockOverrideCaller     // Read-only binding to the contract
	BlockOverrideTransactor // Write-only binding to the contract
	BlockOverrideFilterer   // Log filterer for contract events
}

// BlockOverrideCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlockOverrideCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockOverrideTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlockOverrideTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockOverrideFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlockOverrideFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockOverrideSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlockOverrideSession struct {
	Contract     *BlockOverride    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlockOverrideCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlockOverrideCallerSession struct {
	Contract *BlockOverrideCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// BlockOverrideTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlockOverrideTransactorSession struct {
	Contract     *BlockOverrideTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// BlockOverrideRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlockOverrideRaw struct {
	Contract *BlockOverride // Generic contract binding to access the raw methods on
}

// BlockOverrideCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlockOverrideCallerRaw struct {
	Contract *BlockOverrideCaller // Generic read-only contract binding to access the raw methods on
}

// BlockOverrideTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlockOverrideTransactorRaw struct {
	Contract *BlockOverrideTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlockOverride creates a new instance of BlockOverride, bound to a specific deployed contract.
func NewBlockOverride(address common.Address, backend bind.ContractBackend) (*BlockOverride, error) {
	contract, err := bindBlockOverride(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BlockOverride{BlockOverrideCaller: BlockOverrideCaller{contract: contract}, BlockOverrideTransactor: BlockOverrideTransactor{contract: contract}, BlockOverrideFilterer: BlockOverrideFilterer{contract: contract}}, nil
}

// NewBlockOverrideCaller creates a new read-only instance of BlockOverride, bound to a specific deployed contract.
func NewBlockOverrideCaller(address common.Address, caller bind.ContractCaller) (*BlockOverrideCaller, error) {
	contract, err := bindBlockOverride(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlockOverrideCaller{contract: contract}, nil
}

// NewBlockOverrideTransactor creates a new write-only instance of BlockOverride, bound to a specific deployed contract.
func NewBlockOverrideTransactor(address common.Address, transactor bind.ContractTransactor) (*BlockOverrideTransactor, error) {
	contract, err := bindBlockOverride(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlockOverrideTransactor{contract: contract}, nil
}

// NewBlockOverrideFilterer creates a new log filterer instance of BlockOverride, bound to a specific deployed contract.
func NewBlockOverrideFilterer(address common.Address, filterer bind.ContractFilterer) (*BlockOverrideFilterer, error) {
	contract, err := bindBlockOverride(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlockOverrideFilterer{contract: contract}, nil
}

// bindBlockOverride binds a generic wrapper to an already deployed contract.
func bindBlockOverride(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BlockOverrideMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockOverride *BlockOverrideRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockOverride.Contract.BlockOverrideCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockOverride *BlockOverrideRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOverride.Contract.BlockOverrideTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockOverride *BlockOverrideRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockOverride.Contract.BlockOverrideTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockOverride *BlockOverrideCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockOverride.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockOverride *BlockOverrideTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOverride.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockOverride *BlockOverrideTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockOverride.Contract.contract.Transact(opts, method, params...)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 nr) view returns(bytes32)
func (_BlockOverride *BlockOverrideCaller) GetBlockHash(opts *bind.CallOpts, nr *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BlockOverride.contract.Call(opts, &out, "getBlockHash", nr)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 nr) view returns(bytes32)
func (_BlockOverride *BlockOverrideSession) GetBlockHash(nr *big.Int) ([32]byte, error) {
	return _BlockOverride.Contract.GetBlockHash(&_BlockOverride.CallOpts, nr)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 nr) view returns(bytes32)
func (_BlockOverride *BlockOverrideCallerSession) GetBlockHash(nr *big.Int) ([32]byte, error) {
	return _BlockOverride.Contract.GetBlockHash(&_BlockOverride.CallOpts, nr)
}

// GetBlockParameters is a free data retrieval call binding the contract method 0xa3289b77.
//
// Solidity: function getBlockParameters() view returns(uint256 number, uint256 difficulty, uint256 time, uint256 gaslimit, address coinbase, uint256 random, uint256 basefee, uint256 blobbasefee)
func (_BlockOverride *BlockOverrideCaller) GetBlockParameters(opts *bind.CallOpts) (struct {
	Number      *big.Int
	Difficulty  *big.Int
	Time        *big.Int
	Gaslimit    *big.Int
	Coinbase    common.Address
	Random      *big.Int
	Basefee     *big.Int
	Blobbasefee *big.Int
}, error) {
	var out []interface{}
	err := _BlockOverride.contract.Call(opts, &out, "getBlockParameters")

	outstruct := new(struct {
		Number      *big.Int
		Difficulty  *big.Int
		Time        *big.Int
		Gaslimit    *big.Int
		Coinbase    common.Address
		Random      *big.Int
		Basefee     *big.Int
		Blobbasefee *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Number = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Difficulty = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Time = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Gaslimit = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Coinbase = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Random = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Basefee = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)
	outstruct.Blobbasefee = *abi.ConvertType(out[7], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetBlockParameters is a free data retrieval call binding the contract method 0xa3289b77.
//
// Solidity: function getBlockParameters() view returns(uint256 number, uint256 difficulty, uint256 time, uint256 gaslimit, address coinbase, uint256 random, uint256 basefee, uint256 blobbasefee)
func (_BlockOverride *BlockOverrideSession) GetBlockParameters() (struct {
	Number      *big.Int
	Difficulty  *big.Int
	Time        *big.Int
	Gaslimit    *big.Int
	Coinbase    common.Address
	Random      *big.Int
	Basefee     *big.Int
	Blobbasefee *big.Int
}, error) {
	return _BlockOverride.Contract.GetBlockParameters(&_BlockOverride.CallOpts)
}

// GetBlockParameters is a free data retrieval call binding the contract method 0xa3289b77.
//
// Solidity: function getBlockParameters() view returns(uint256 number, uint256 difficulty, uint256 time, uint256 gaslimit, address coinbase, uint256 random, uint256 basefee, uint256 blobbasefee)
func (_BlockOverride *BlockOverrideCallerSession) GetBlockParameters() (struct {
	Number      *big.Int
	Difficulty  *big.Int
	Time        *big.Int
	Gaslimit    *big.Int
	Coinbase    common.Address
	Random      *big.Int
	Basefee     *big.Int
	Blobbasefee *big.Int
}, error) {
	return _BlockOverride.Contract.GetBlockParameters(&_BlockOverride.CallOpts)
}

// Observe is a paid mutator transaction binding the contract method 0x14fc78fc.
//
// Solidity: function observe() returns()
func (_BlockOverride *BlockOverrideTransactor) Observe(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOverride.contract.Transact(opts, "observe")
}

// Observe is a paid mutator transaction binding the contract method 0x14fc78fc.
//
// Solidity: function observe() returns()
func (_BlockOverride *BlockOverrideSession) Observe() (*types.Transaction, error) {
	return _BlockOverride.Contract.Observe(&_BlockOverride.TransactOpts)
}

// Observe is a paid mutator transaction binding the contract method 0x14fc78fc.
//
// Solidity: function observe() returns()
func (_BlockOverride *BlockOverrideTransactorSession) Observe() (*types.Transaction, error) {
	return _BlockOverride.Contract.Observe(&_BlockOverride.TransactOpts)
}

// BlockOverrideSeenIterator is returned from FilterSeen and is used to iterate over the raw logs and unpacked data for Seen events raised by the BlockOverride contract.
type BlockOverrideSeenIterator struct {
	Event *BlockOverrideSeen // Event containing the contract specifics and raw log

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
func (it *BlockOverrideSeenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlockOverrideSeen)
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
		it.Event = new(BlockOverrideSeen)
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
func (it *BlockOverrideSeenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlockOverrideSeenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlockOverrideSeen represents a Seen event raised by the BlockOverride contract.
type BlockOverrideSeen struct {
	CurrentBlock  *big.Int
	ObservedBlock *big.Int
	BlockHash     [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSeen is a free log retrieval operation binding the contract event 0x2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db74696.
//
// Solidity: event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash)
func (_BlockOverride *BlockOverrideFilterer) FilterSeen(opts *bind.FilterOpts) (*BlockOverrideSeenIterator, error) {

	logs, sub, err := _BlockOverride.contract.FilterLogs(opts, "Seen")
	if err != nil {
		return nil, err
	}
	return &BlockOverrideSeenIterator{contract: _BlockOverride.contract, event: "Seen", logs: logs, sub: sub}, nil
}

// WatchSeen is a free log subscription operation binding the contract event 0x2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db74696.
//
// Solidity: event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash)
func (_BlockOverride *BlockOverrideFilterer) WatchSeen(opts *bind.WatchOpts, sink chan<- *BlockOverrideSeen) (event.Subscription, error) {

	logs, sub, err := _BlockOverride.contract.WatchLogs(opts, "Seen")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlockOverrideSeen)
				if err := _BlockOverride.contract.UnpackLog(event, "Seen", log); err != nil {
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

// ParseSeen is a log parse operation binding the contract event 0x2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db74696.
//
// Solidity: event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash)
func (_BlockOverride *BlockOverrideFilterer) ParseSeen(log types.Log) (*BlockOverrideSeen, error) {
	event := new(BlockOverrideSeen)
	if err := _BlockOverride.contract.UnpackLog(event, "Seen", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
