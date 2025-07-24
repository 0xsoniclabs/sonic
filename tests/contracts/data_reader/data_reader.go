// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package data_reader

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

// DataReaderMetaData contains all meta data concerning the DataReader contract.
var DataReaderMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"size\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"bufferSize\",\"type\":\"uint64\"}],\"name\":\"DataSize\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5061029c8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c8063093165d31461002d575b5f5ffd5b610047600480360381019061004291906101d6565b610049565b005b7f5170360a993a0fe8b36ebeaaa8cd89f47f953ff868ecc2c003de7217f8d7e86f5f369050825160405161007e92919061023f565b60405180910390a150565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6100e8826100a2565b810181811067ffffffffffffffff82111715610107576101066100b2565b5b80604052505050565b5f610119610089565b905061012582826100df565b919050565b5f67ffffffffffffffff821115610144576101436100b2565b5b61014d826100a2565b9050602081019050919050565b828183375f83830152505050565b5f61017a6101758461012a565b610110565b9050828152602081018484840111156101965761019561009e565b5b6101a184828561015a565b509392505050565b5f82601f8301126101bd576101bc61009a565b5b81356101cd848260208601610168565b91505092915050565b5f602082840312156101eb576101ea610092565b5b5f82013567ffffffffffffffff81111561020857610207610096565b5b610214848285016101a9565b91505092915050565b5f67ffffffffffffffff82169050919050565b6102398161021d565b82525050565b5f6040820190506102525f830185610230565b61025f6020830184610230565b939250505056fea2646970667358221220c13625fb1606edc37e9e3c26d91fbba1b85669c84fee0183a9677d2ba7ce44ba64736f6c634300081e0033",
}

// DataReaderABI is the input ABI used to generate the binding from.
// Deprecated: Use DataReaderMetaData.ABI instead.
var DataReaderABI = DataReaderMetaData.ABI

// DataReaderBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DataReaderMetaData.Bin instead.
var DataReaderBin = DataReaderMetaData.Bin

// DeployDataReader deploys a new Ethereum contract, binding an instance of DataReader to it.
func DeployDataReader(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DataReader, error) {
	parsed, err := DataReaderMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DataReaderBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DataReader{DataReaderCaller: DataReaderCaller{contract: contract}, DataReaderTransactor: DataReaderTransactor{contract: contract}, DataReaderFilterer: DataReaderFilterer{contract: contract}}, nil
}

// DataReader is an auto generated Go binding around an Ethereum contract.
type DataReader struct {
	DataReaderCaller     // Read-only binding to the contract
	DataReaderTransactor // Write-only binding to the contract
	DataReaderFilterer   // Log filterer for contract events
}

// DataReaderCaller is an auto generated read-only Go binding around an Ethereum contract.
type DataReaderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DataReaderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DataReaderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DataReaderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DataReaderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DataReaderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DataReaderSession struct {
	Contract     *DataReader       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DataReaderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DataReaderCallerSession struct {
	Contract *DataReaderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// DataReaderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DataReaderTransactorSession struct {
	Contract     *DataReaderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// DataReaderRaw is an auto generated low-level Go binding around an Ethereum contract.
type DataReaderRaw struct {
	Contract *DataReader // Generic contract binding to access the raw methods on
}

// DataReaderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DataReaderCallerRaw struct {
	Contract *DataReaderCaller // Generic read-only contract binding to access the raw methods on
}

// DataReaderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DataReaderTransactorRaw struct {
	Contract *DataReaderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDataReader creates a new instance of DataReader, bound to a specific deployed contract.
func NewDataReader(address common.Address, backend bind.ContractBackend) (*DataReader, error) {
	contract, err := bindDataReader(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DataReader{DataReaderCaller: DataReaderCaller{contract: contract}, DataReaderTransactor: DataReaderTransactor{contract: contract}, DataReaderFilterer: DataReaderFilterer{contract: contract}}, nil
}

// NewDataReaderCaller creates a new read-only instance of DataReader, bound to a specific deployed contract.
func NewDataReaderCaller(address common.Address, caller bind.ContractCaller) (*DataReaderCaller, error) {
	contract, err := bindDataReader(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DataReaderCaller{contract: contract}, nil
}

// NewDataReaderTransactor creates a new write-only instance of DataReader, bound to a specific deployed contract.
func NewDataReaderTransactor(address common.Address, transactor bind.ContractTransactor) (*DataReaderTransactor, error) {
	contract, err := bindDataReader(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DataReaderTransactor{contract: contract}, nil
}

// NewDataReaderFilterer creates a new log filterer instance of DataReader, bound to a specific deployed contract.
func NewDataReaderFilterer(address common.Address, filterer bind.ContractFilterer) (*DataReaderFilterer, error) {
	contract, err := bindDataReader(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DataReaderFilterer{contract: contract}, nil
}

// bindDataReader binds a generic wrapper to an already deployed contract.
func bindDataReader(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DataReaderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DataReader *DataReaderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DataReader.Contract.DataReaderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DataReader *DataReaderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataReader.Contract.DataReaderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DataReader *DataReaderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DataReader.Contract.DataReaderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DataReader *DataReaderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DataReader.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DataReader *DataReaderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataReader.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DataReader *DataReaderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DataReader.Contract.contract.Transact(opts, method, params...)
}

// SendData is a paid mutator transaction binding the contract method 0x093165d3.
//
// Solidity: function sendData(bytes data) returns()
func (_DataReader *DataReaderTransactor) SendData(opts *bind.TransactOpts, data []byte) (*types.Transaction, error) {
	return _DataReader.contract.Transact(opts, "sendData", data)
}

// SendData is a paid mutator transaction binding the contract method 0x093165d3.
//
// Solidity: function sendData(bytes data) returns()
func (_DataReader *DataReaderSession) SendData(data []byte) (*types.Transaction, error) {
	return _DataReader.Contract.SendData(&_DataReader.TransactOpts, data)
}

// SendData is a paid mutator transaction binding the contract method 0x093165d3.
//
// Solidity: function sendData(bytes data) returns()
func (_DataReader *DataReaderTransactorSession) SendData(data []byte) (*types.Transaction, error) {
	return _DataReader.Contract.SendData(&_DataReader.TransactOpts, data)
}

// DataReaderDataSizeIterator is returned from FilterDataSize and is used to iterate over the raw logs and unpacked data for DataSize events raised by the DataReader contract.
type DataReaderDataSizeIterator struct {
	Event *DataReaderDataSize // Event containing the contract specifics and raw log

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
func (it *DataReaderDataSizeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataReaderDataSize)
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
		it.Event = new(DataReaderDataSize)
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
func (it *DataReaderDataSizeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataReaderDataSizeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataReaderDataSize represents a DataSize event raised by the DataReader contract.
type DataReaderDataSize struct {
	Size       uint64
	BufferSize uint64
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterDataSize is a free log retrieval operation binding the contract event 0x5170360a993a0fe8b36ebeaaa8cd89f47f953ff868ecc2c003de7217f8d7e86f.
//
// Solidity: event DataSize(uint64 size, uint64 bufferSize)
func (_DataReader *DataReaderFilterer) FilterDataSize(opts *bind.FilterOpts) (*DataReaderDataSizeIterator, error) {

	logs, sub, err := _DataReader.contract.FilterLogs(opts, "DataSize")
	if err != nil {
		return nil, err
	}
	return &DataReaderDataSizeIterator{contract: _DataReader.contract, event: "DataSize", logs: logs, sub: sub}, nil
}

// WatchDataSize is a free log subscription operation binding the contract event 0x5170360a993a0fe8b36ebeaaa8cd89f47f953ff868ecc2c003de7217f8d7e86f.
//
// Solidity: event DataSize(uint64 size, uint64 bufferSize)
func (_DataReader *DataReaderFilterer) WatchDataSize(opts *bind.WatchOpts, sink chan<- *DataReaderDataSize) (event.Subscription, error) {

	logs, sub, err := _DataReader.contract.WatchLogs(opts, "DataSize")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataReaderDataSize)
				if err := _DataReader.contract.UnpackLog(event, "DataSize", log); err != nil {
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

// ParseDataSize is a log parse operation binding the contract event 0x5170360a993a0fe8b36ebeaaa8cd89f47f953ff868ecc2c003de7217f8d7e86f.
//
// Solidity: event DataSize(uint64 size, uint64 bufferSize)
func (_DataReader *DataReaderFilterer) ParseDataSize(log types.Log) (*DataReaderDataSize, error) {
	event := new(DataReaderDataSize)
	if err := _DataReader.contract.UnpackLog(event, "DataSize", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
