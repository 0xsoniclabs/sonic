// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package block_hash

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

// BlockHashMetaData contains all meta data concerning the BlockHash contract.
var BlockHashMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"currentBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"observedBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"name\":\"Seen\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nr\",\"type\":\"uint256\"}],\"name\":\"getBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"observe\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506102f78061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c806314fc78fc14610038578063ee82ac5e14610042575b5f5ffd5b610040610072565b005b61005c60048036038101906100579190610147565b610106565b604051610069919061018a565b60405180910390f35b5f5f90505f60054361008491906101d0565b90506101048111156100a15761010e8161009e9190610203565b91505b5f8290505b818111610101575f814090507f2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db746964383836040516100e593929190610245565b60405180910390a15080806100f99061027a565b9150506100a6565b505050565b5f81409050919050565b5f5ffd5b5f819050919050565b61012681610114565b8114610130575f5ffd5b50565b5f813590506101418161011d565b92915050565b5f6020828403121561015c5761015b610110565b5b5f61016984828501610133565b91505092915050565b5f819050919050565b61018481610172565b82525050565b5f60208201905061019d5f83018461017b565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6101da82610114565b91506101e583610114565b92508282019050808211156101fd576101fc6101a3565b5b92915050565b5f61020d82610114565b915061021883610114565b92508282039050818111156102305761022f6101a3565b5b92915050565b61023f81610114565b82525050565b5f6060820190506102585f830186610236565b6102656020830185610236565b610272604083018461017b565b949350505050565b5f61028482610114565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036102b6576102b56101a3565b5b60018201905091905056fea2646970667358221220baed7e08b24bf54ce79facd650b4ce1bfa3f33a18e1448fab3a92213bf31d4bb64736f6c634300081c0033",
}

// BlockHashABI is the input ABI used to generate the binding from.
// Deprecated: Use BlockHashMetaData.ABI instead.
var BlockHashABI = BlockHashMetaData.ABI

// BlockHashBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BlockHashMetaData.Bin instead.
var BlockHashBin = BlockHashMetaData.Bin

// DeployBlockHash deploys a new Ethereum contract, binding an instance of BlockHash to it.
func DeployBlockHash(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BlockHash, error) {
	parsed, err := BlockHashMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BlockHashBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BlockHash{BlockHashCaller: BlockHashCaller{contract: contract}, BlockHashTransactor: BlockHashTransactor{contract: contract}, BlockHashFilterer: BlockHashFilterer{contract: contract}}, nil
}

// BlockHash is an auto generated Go binding around an Ethereum contract.
type BlockHash struct {
	BlockHashCaller     // Read-only binding to the contract
	BlockHashTransactor // Write-only binding to the contract
	BlockHashFilterer   // Log filterer for contract events
}

// BlockHashCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlockHashCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockHashTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlockHashTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockHashFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlockHashFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockHashSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlockHashSession struct {
	Contract     *BlockHash        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlockHashCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlockHashCallerSession struct {
	Contract *BlockHashCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// BlockHashTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlockHashTransactorSession struct {
	Contract     *BlockHashTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// BlockHashRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlockHashRaw struct {
	Contract *BlockHash // Generic contract binding to access the raw methods on
}

// BlockHashCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlockHashCallerRaw struct {
	Contract *BlockHashCaller // Generic read-only contract binding to access the raw methods on
}

// BlockHashTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlockHashTransactorRaw struct {
	Contract *BlockHashTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlockHash creates a new instance of BlockHash, bound to a specific deployed contract.
func NewBlockHash(address common.Address, backend bind.ContractBackend) (*BlockHash, error) {
	contract, err := bindBlockHash(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BlockHash{BlockHashCaller: BlockHashCaller{contract: contract}, BlockHashTransactor: BlockHashTransactor{contract: contract}, BlockHashFilterer: BlockHashFilterer{contract: contract}}, nil
}

// NewBlockHashCaller creates a new read-only instance of BlockHash, bound to a specific deployed contract.
func NewBlockHashCaller(address common.Address, caller bind.ContractCaller) (*BlockHashCaller, error) {
	contract, err := bindBlockHash(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlockHashCaller{contract: contract}, nil
}

// NewBlockHashTransactor creates a new write-only instance of BlockHash, bound to a specific deployed contract.
func NewBlockHashTransactor(address common.Address, transactor bind.ContractTransactor) (*BlockHashTransactor, error) {
	contract, err := bindBlockHash(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlockHashTransactor{contract: contract}, nil
}

// NewBlockHashFilterer creates a new log filterer instance of BlockHash, bound to a specific deployed contract.
func NewBlockHashFilterer(address common.Address, filterer bind.ContractFilterer) (*BlockHashFilterer, error) {
	contract, err := bindBlockHash(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlockHashFilterer{contract: contract}, nil
}

// bindBlockHash binds a generic wrapper to an already deployed contract.
func bindBlockHash(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BlockHashMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockHash *BlockHashRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockHash.Contract.BlockHashCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockHash *BlockHashRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockHash.Contract.BlockHashTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockHash *BlockHashRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockHash.Contract.BlockHashTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockHash *BlockHashCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockHash.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockHash *BlockHashTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockHash.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockHash *BlockHashTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockHash.Contract.contract.Transact(opts, method, params...)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 nr) view returns(bytes32)
func (_BlockHash *BlockHashCaller) GetBlockHash(opts *bind.CallOpts, nr *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BlockHash.contract.Call(opts, &out, "getBlockHash", nr)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 nr) view returns(bytes32)
func (_BlockHash *BlockHashSession) GetBlockHash(nr *big.Int) ([32]byte, error) {
	return _BlockHash.Contract.GetBlockHash(&_BlockHash.CallOpts, nr)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 nr) view returns(bytes32)
func (_BlockHash *BlockHashCallerSession) GetBlockHash(nr *big.Int) ([32]byte, error) {
	return _BlockHash.Contract.GetBlockHash(&_BlockHash.CallOpts, nr)
}

// Observe is a paid mutator transaction binding the contract method 0x14fc78fc.
//
// Solidity: function observe() returns()
func (_BlockHash *BlockHashTransactor) Observe(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockHash.contract.Transact(opts, "observe")
}

// Observe is a paid mutator transaction binding the contract method 0x14fc78fc.
//
// Solidity: function observe() returns()
func (_BlockHash *BlockHashSession) Observe() (*types.Transaction, error) {
	return _BlockHash.Contract.Observe(&_BlockHash.TransactOpts)
}

// Observe is a paid mutator transaction binding the contract method 0x14fc78fc.
//
// Solidity: function observe() returns()
func (_BlockHash *BlockHashTransactorSession) Observe() (*types.Transaction, error) {
	return _BlockHash.Contract.Observe(&_BlockHash.TransactOpts)
}

// BlockHashSeenIterator is returned from FilterSeen and is used to iterate over the raw logs and unpacked data for Seen events raised by the BlockHash contract.
type BlockHashSeenIterator struct {
	Event *BlockHashSeen // Event containing the contract specifics and raw log

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
func (it *BlockHashSeenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlockHashSeen)
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
		it.Event = new(BlockHashSeen)
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
func (it *BlockHashSeenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlockHashSeenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlockHashSeen represents a Seen event raised by the BlockHash contract.
type BlockHashSeen struct {
	CurrentBlock  *big.Int
	ObservedBlock *big.Int
	BlockHash     [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSeen is a free log retrieval operation binding the contract event 0x2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db74696.
//
// Solidity: event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash)
func (_BlockHash *BlockHashFilterer) FilterSeen(opts *bind.FilterOpts) (*BlockHashSeenIterator, error) {

	logs, sub, err := _BlockHash.contract.FilterLogs(opts, "Seen")
	if err != nil {
		return nil, err
	}
	return &BlockHashSeenIterator{contract: _BlockHash.contract, event: "Seen", logs: logs, sub: sub}, nil
}

// WatchSeen is a free log subscription operation binding the contract event 0x2e2db0da10eef8180d8a58ccf88e981740e8a677554b25fe1e1f973a8db74696.
//
// Solidity: event Seen(uint256 currentBlock, uint256 observedBlock, bytes32 blockHash)
func (_BlockHash *BlockHashFilterer) WatchSeen(opts *bind.WatchOpts, sink chan<- *BlockHashSeen) (event.Subscription, error) {

	logs, sub, err := _BlockHash.contract.WatchLogs(opts, "Seen")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlockHashSeen)
				if err := _BlockHash.contract.UnpackLog(event, "Seen", log); err != nil {
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
func (_BlockHash *BlockHashFilterer) ParseSeen(log types.Log) (*BlockHashSeen, error) {
	event := new(BlockHashSeen)
	if err := _BlockHash.contract.UnpackLog(event, "Seen", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
