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
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"sendData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506102168061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c8063093165d31461002d575b5f5ffd5b61004760048036038101906100429190610199565b610049565b005b50565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6100ab82610065565b810181811067ffffffffffffffff821117156100ca576100c9610075565b5b80604052505050565b5f6100dc61004c565b90506100e882826100a2565b919050565b5f67ffffffffffffffff82111561010757610106610075565b5b61011082610065565b9050602081019050919050565b828183375f83830152505050565b5f61013d610138846100ed565b6100d3565b90508281526020810184848401111561015957610158610061565b5b61016484828561011d565b509392505050565b5f82601f8301126101805761017f61005d565b5b813561019084826020860161012b565b91505092915050565b5f602082840312156101ae576101ad610055565b5b5f82013567ffffffffffffffff8111156101cb576101ca610059565b5b6101d78482850161016c565b9150509291505056fea2646970667358221220355b45a3bee70fa03cd5d4ba1201bcf83fa70bd3375c34075f01c7cf8fcce99e64736f6c634300081e0033",
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
