// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package block_parameters

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

// BlockParametersParameters is an auto generated low-level Go binding around an user-defined struct.
type BlockParametersParameters struct {
	ChainId     *big.Int
	Number      *big.Int
	Time        *big.Int
	Coinbase    common.Address
	GasLimit    *big.Int
	BaseFee     *big.Int
	BlobBaseFee *big.Int
	PrevRandao  *big.Int
}

// BlockParametersMetaData contains all meta data concerning the BlockParameters contract.
var BlockParametersMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blobBaseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevRandao\",\"type\":\"uint256\"}],\"indexed\":false,\"internalType\":\"structBlockParameters.Parameters\",\"name\":\"parameters\",\"type\":\"tuple\"}],\"name\":\"Log\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"getBlockParameters\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blobBaseFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevRandao\",\"type\":\"uint256\"}],\"internalType\":\"structBlockParameters.Parameters\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"logBlockParameters\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5061029c8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063a3289b7714610038578063d1c6de5c14610056575b5f5ffd5b610040610060565b60405161004d919061024c565b60405180910390f35b61005e6100c2565b005b610068610102565b5f6040518061010001604052804681526020014381526020014281526020014173ffffffffffffffffffffffffffffffffffffffff1681526020014581526020014881526020014a81526020014481525090508091505090565b7fb6af2c79c567b235530ac3048507207ce0d9e387bcf483d12e7923572ba831a06100eb610060565b6040516100f8919061024c565b60405180910390a1565b6040518061010001604052805f81526020015f81526020015f81526020015f73ffffffffffffffffffffffffffffffffffffffff1681526020015f81526020015f81526020015f81526020015f81525090565b5f819050919050565b61016781610155565b82525050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6101968261016d565b9050919050565b6101a68161018c565b82525050565b61010082015f8201516101c15f85018261015e565b5060208201516101d4602085018261015e565b5060408201516101e7604085018261015e565b5060608201516101fa606085018261019d565b50608082015161020d608085018261015e565b5060a082015161022060a085018261015e565b5060c082015161023360c085018261015e565b5060e082015161024660e085018261015e565b50505050565b5f610100820190506102605f8301846101ac565b9291505056fea26469706673582212209177184ab2deea3d360a7a509538b70ecf1c0d41833c695b7a6668baa33bd4c364736f6c634300081e0033",
}

// BlockParametersABI is the input ABI used to generate the binding from.
// Deprecated: Use BlockParametersMetaData.ABI instead.
var BlockParametersABI = BlockParametersMetaData.ABI

// BlockParametersBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BlockParametersMetaData.Bin instead.
var BlockParametersBin = BlockParametersMetaData.Bin

// DeployBlockParameters deploys a new Ethereum contract, binding an instance of BlockParameters to it.
func DeployBlockParameters(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BlockParameters, error) {
	parsed, err := BlockParametersMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BlockParametersBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BlockParameters{BlockParametersCaller: BlockParametersCaller{contract: contract}, BlockParametersTransactor: BlockParametersTransactor{contract: contract}, BlockParametersFilterer: BlockParametersFilterer{contract: contract}}, nil
}

// BlockParameters is an auto generated Go binding around an Ethereum contract.
type BlockParameters struct {
	BlockParametersCaller     // Read-only binding to the contract
	BlockParametersTransactor // Write-only binding to the contract
	BlockParametersFilterer   // Log filterer for contract events
}

// BlockParametersCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlockParametersCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockParametersTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlockParametersTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockParametersFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlockParametersFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockParametersSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlockParametersSession struct {
	Contract     *BlockParameters  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlockParametersCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlockParametersCallerSession struct {
	Contract *BlockParametersCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// BlockParametersTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlockParametersTransactorSession struct {
	Contract     *BlockParametersTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// BlockParametersRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlockParametersRaw struct {
	Contract *BlockParameters // Generic contract binding to access the raw methods on
}

// BlockParametersCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlockParametersCallerRaw struct {
	Contract *BlockParametersCaller // Generic read-only contract binding to access the raw methods on
}

// BlockParametersTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlockParametersTransactorRaw struct {
	Contract *BlockParametersTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlockParameters creates a new instance of BlockParameters, bound to a specific deployed contract.
func NewBlockParameters(address common.Address, backend bind.ContractBackend) (*BlockParameters, error) {
	contract, err := bindBlockParameters(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BlockParameters{BlockParametersCaller: BlockParametersCaller{contract: contract}, BlockParametersTransactor: BlockParametersTransactor{contract: contract}, BlockParametersFilterer: BlockParametersFilterer{contract: contract}}, nil
}

// NewBlockParametersCaller creates a new read-only instance of BlockParameters, bound to a specific deployed contract.
func NewBlockParametersCaller(address common.Address, caller bind.ContractCaller) (*BlockParametersCaller, error) {
	contract, err := bindBlockParameters(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlockParametersCaller{contract: contract}, nil
}

// NewBlockParametersTransactor creates a new write-only instance of BlockParameters, bound to a specific deployed contract.
func NewBlockParametersTransactor(address common.Address, transactor bind.ContractTransactor) (*BlockParametersTransactor, error) {
	contract, err := bindBlockParameters(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlockParametersTransactor{contract: contract}, nil
}

// NewBlockParametersFilterer creates a new log filterer instance of BlockParameters, bound to a specific deployed contract.
func NewBlockParametersFilterer(address common.Address, filterer bind.ContractFilterer) (*BlockParametersFilterer, error) {
	contract, err := bindBlockParameters(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlockParametersFilterer{contract: contract}, nil
}

// bindBlockParameters binds a generic wrapper to an already deployed contract.
func bindBlockParameters(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BlockParametersMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockParameters *BlockParametersRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockParameters.Contract.BlockParametersCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockParameters *BlockParametersRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockParameters.Contract.BlockParametersTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockParameters *BlockParametersRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockParameters.Contract.BlockParametersTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockParameters *BlockParametersCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockParameters.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockParameters *BlockParametersTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockParameters.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockParameters *BlockParametersTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockParameters.Contract.contract.Transact(opts, method, params...)
}

// GetBlockParameters is a free data retrieval call binding the contract method 0xa3289b77.
//
// Solidity: function getBlockParameters() view returns((uint256,uint256,uint256,address,uint256,uint256,uint256,uint256))
func (_BlockParameters *BlockParametersCaller) GetBlockParameters(opts *bind.CallOpts) (BlockParametersParameters, error) {
	var out []interface{}
	err := _BlockParameters.contract.Call(opts, &out, "getBlockParameters")

	if err != nil {
		return *new(BlockParametersParameters), err
	}

	out0 := *abi.ConvertType(out[0], new(BlockParametersParameters)).(*BlockParametersParameters)

	return out0, err

}

// GetBlockParameters is a free data retrieval call binding the contract method 0xa3289b77.
//
// Solidity: function getBlockParameters() view returns((uint256,uint256,uint256,address,uint256,uint256,uint256,uint256))
func (_BlockParameters *BlockParametersSession) GetBlockParameters() (BlockParametersParameters, error) {
	return _BlockParameters.Contract.GetBlockParameters(&_BlockParameters.CallOpts)
}

// GetBlockParameters is a free data retrieval call binding the contract method 0xa3289b77.
//
// Solidity: function getBlockParameters() view returns((uint256,uint256,uint256,address,uint256,uint256,uint256,uint256))
func (_BlockParameters *BlockParametersCallerSession) GetBlockParameters() (BlockParametersParameters, error) {
	return _BlockParameters.Contract.GetBlockParameters(&_BlockParameters.CallOpts)
}

// LogBlockParameters is a paid mutator transaction binding the contract method 0xd1c6de5c.
//
// Solidity: function logBlockParameters() returns()
func (_BlockParameters *BlockParametersTransactor) LogBlockParameters(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockParameters.contract.Transact(opts, "logBlockParameters")
}

// LogBlockParameters is a paid mutator transaction binding the contract method 0xd1c6de5c.
//
// Solidity: function logBlockParameters() returns()
func (_BlockParameters *BlockParametersSession) LogBlockParameters() (*types.Transaction, error) {
	return _BlockParameters.Contract.LogBlockParameters(&_BlockParameters.TransactOpts)
}

// LogBlockParameters is a paid mutator transaction binding the contract method 0xd1c6de5c.
//
// Solidity: function logBlockParameters() returns()
func (_BlockParameters *BlockParametersTransactorSession) LogBlockParameters() (*types.Transaction, error) {
	return _BlockParameters.Contract.LogBlockParameters(&_BlockParameters.TransactOpts)
}

// BlockParametersLogIterator is returned from FilterLog and is used to iterate over the raw logs and unpacked data for Log events raised by the BlockParameters contract.
type BlockParametersLogIterator struct {
	Event *BlockParametersLog // Event containing the contract specifics and raw log

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
func (it *BlockParametersLogIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlockParametersLog)
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
		it.Event = new(BlockParametersLog)
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
func (it *BlockParametersLogIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlockParametersLogIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlockParametersLog represents a Log event raised by the BlockParameters contract.
type BlockParametersLog struct {
	Parameters BlockParametersParameters
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLog is a free log retrieval operation binding the contract event 0xb6af2c79c567b235530ac3048507207ce0d9e387bcf483d12e7923572ba831a0.
//
// Solidity: event Log((uint256,uint256,uint256,address,uint256,uint256,uint256,uint256) parameters)
func (_BlockParameters *BlockParametersFilterer) FilterLog(opts *bind.FilterOpts) (*BlockParametersLogIterator, error) {

	logs, sub, err := _BlockParameters.contract.FilterLogs(opts, "Log")
	if err != nil {
		return nil, err
	}
	return &BlockParametersLogIterator{contract: _BlockParameters.contract, event: "Log", logs: logs, sub: sub}, nil
}

// WatchLog is a free log subscription operation binding the contract event 0xb6af2c79c567b235530ac3048507207ce0d9e387bcf483d12e7923572ba831a0.
//
// Solidity: event Log((uint256,uint256,uint256,address,uint256,uint256,uint256,uint256) parameters)
func (_BlockParameters *BlockParametersFilterer) WatchLog(opts *bind.WatchOpts, sink chan<- *BlockParametersLog) (event.Subscription, error) {

	logs, sub, err := _BlockParameters.contract.WatchLogs(opts, "Log")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlockParametersLog)
				if err := _BlockParameters.contract.UnpackLog(event, "Log", log); err != nil {
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

// ParseLog is a log parse operation binding the contract event 0xb6af2c79c567b235530ac3048507207ce0d9e387bcf483d12e7923572ba831a0.
//
// Solidity: event Log((uint256,uint256,uint256,address,uint256,uint256,uint256,uint256) parameters)
func (_BlockParameters *BlockParametersFilterer) ParseLog(log types.Log) (*BlockParametersLog, error) {
	event := new(BlockParametersLog)
	if err := _BlockParameters.contract.UnpackLog(event, "Log", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
