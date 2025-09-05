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
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"currentBlock\",\"type\":\"uint256\"}],\"name\":\"BlockNumber\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getBlockParameters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"difficulty\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gaslimit\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"random\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"basefee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blobbasefee\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"logBlockNumber\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506102128061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c80635978a24514610038578063a3289b7714610042575b5f5ffd5b610040610067565b005b61004a6100a0565b60405161005e989796959493929190610121565b60405180910390f35b7fc04eeb4cfe0799838abac8fa75bca975bff679179886c80c84a7b93229a1a61843604051610096919061019d565b60405180910390a1565b5f5f5f5f5f5f5f5f4397504496504295504594504193504492504891504a90509091929394959697565b5f819050919050565b6100dc816100ca565b82525050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61010b826100e2565b9050919050565b61011b81610101565b82525050565b5f610100820190506101355f83018b6100d3565b610142602083018a6100d3565b61014f60408301896100d3565b61015c60608301886100d3565b6101696080830187610112565b61017660a08301866100d3565b61018360c08301856100d3565b61019060e08301846100d3565b9998505050505050505050565b5f6020820190506101b05f8301846100d3565b9291505056fea26469706673582212200288b441f8f4067a9fc4e0e79d98c59e35464f2d7003b812fe205acd760dae5664736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
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

// LogBlockNumber is a paid mutator transaction binding the contract method 0x5978a245.
//
// Solidity: function logBlockNumber() returns()
func (_BlockOverride *BlockOverrideTransactor) LogBlockNumber(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOverride.contract.Transact(opts, "logBlockNumber")
}

// LogBlockNumber is a paid mutator transaction binding the contract method 0x5978a245.
//
// Solidity: function logBlockNumber() returns()
func (_BlockOverride *BlockOverrideSession) LogBlockNumber() (*types.Transaction, error) {
	return _BlockOverride.Contract.LogBlockNumber(&_BlockOverride.TransactOpts)
}

// LogBlockNumber is a paid mutator transaction binding the contract method 0x5978a245.
//
// Solidity: function logBlockNumber() returns()
func (_BlockOverride *BlockOverrideTransactorSession) LogBlockNumber() (*types.Transaction, error) {
	return _BlockOverride.Contract.LogBlockNumber(&_BlockOverride.TransactOpts)
}

// BlockOverrideBlockNumberIterator is returned from FilterBlockNumber and is used to iterate over the raw logs and unpacked data for BlockNumber events raised by the BlockOverride contract.
type BlockOverrideBlockNumberIterator struct {
	Event *BlockOverrideBlockNumber // Event containing the contract specifics and raw log

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
func (it *BlockOverrideBlockNumberIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlockOverrideBlockNumber)
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
		it.Event = new(BlockOverrideBlockNumber)
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
func (it *BlockOverrideBlockNumberIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlockOverrideBlockNumberIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlockOverrideBlockNumber represents a BlockNumber event raised by the BlockOverride contract.
type BlockOverrideBlockNumber struct {
	CurrentBlock *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBlockNumber is a free log retrieval operation binding the contract event 0xc04eeb4cfe0799838abac8fa75bca975bff679179886c80c84a7b93229a1a618.
//
// Solidity: event BlockNumber(uint256 currentBlock)
func (_BlockOverride *BlockOverrideFilterer) FilterBlockNumber(opts *bind.FilterOpts) (*BlockOverrideBlockNumberIterator, error) {

	logs, sub, err := _BlockOverride.contract.FilterLogs(opts, "BlockNumber")
	if err != nil {
		return nil, err
	}
	return &BlockOverrideBlockNumberIterator{contract: _BlockOverride.contract, event: "BlockNumber", logs: logs, sub: sub}, nil
}

// WatchBlockNumber is a free log subscription operation binding the contract event 0xc04eeb4cfe0799838abac8fa75bca975bff679179886c80c84a7b93229a1a618.
//
// Solidity: event BlockNumber(uint256 currentBlock)
func (_BlockOverride *BlockOverrideFilterer) WatchBlockNumber(opts *bind.WatchOpts, sink chan<- *BlockOverrideBlockNumber) (event.Subscription, error) {

	logs, sub, err := _BlockOverride.contract.WatchLogs(opts, "BlockNumber")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlockOverrideBlockNumber)
				if err := _BlockOverride.contract.UnpackLog(event, "BlockNumber", log); err != nil {
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

// ParseBlockNumber is a log parse operation binding the contract event 0xc04eeb4cfe0799838abac8fa75bca975bff679179886c80c84a7b93229a1a618.
//
// Solidity: event BlockNumber(uint256 currentBlock)
func (_BlockOverride *BlockOverrideFilterer) ParseBlockNumber(log types.Log) (*BlockOverrideBlockNumber, error) {
	event := new(BlockOverrideBlockNumber)
	if err := _BlockOverride.contract.UnpackLog(event, "BlockNumber", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
