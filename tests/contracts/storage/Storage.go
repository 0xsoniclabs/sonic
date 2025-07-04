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

package storage

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

// StorageMetaData contains all meta data concerning the Storage contract.
var StorageMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"a\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"b\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"c\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getA\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getB\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getC\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sumABC\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405260015f5560026001556003600255348015601c575f5ffd5b506102738061002a5f395ff3fe608060405234801561000f575f5ffd5b506004361061007b575f3560e01c8063a2375d1e11610059578063a2375d1e146100d9578063bdd60569146100f7578063c3da42b814610115578063d46300fd146101335761007b565b80630dbe671f1461007f5780634df7e3d01461009d578063a1c51915146100bb575b5f5ffd5b610087610151565b60405161009491906101b6565b60405180910390f35b6100a5610156565b6040516100b291906101b6565b60405180910390f35b6100c361015c565b6040516100d091906101b6565b60405180910390f35b6100e1610165565b6040516100ee91906101b6565b60405180910390f35b6100ff61016e565b60405161010c91906101b6565b60405180910390f35b61011d610190565b60405161012a91906101b6565b60405180910390f35b61013b610196565b60405161014891906101b6565b60405180910390f35b5f5481565b60015481565b5f600154905090565b5f600254905090565b5f6002546001545f5461018191906101fc565b61018b91906101fc565b905090565b60025481565b5f5f54905090565b5f819050919050565b6101b08161019e565b82525050565b5f6020820190506101c95f8301846101a7565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6102068261019e565b91506102118361019e565b92508282019050828112155f8312168382125f841215161715610237576102366101cf565b5b9291505056fea26469706673582212208922ca1baee0869a32acbeb2099b0115fa6fac84bab50e66706e21f0fd036b0964736f6c634300081c0033",
}

// StorageABI is the input ABI used to generate the binding from.
// Deprecated: Use StorageMetaData.ABI instead.
var StorageABI = StorageMetaData.ABI

// StorageBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StorageMetaData.Bin instead.
var StorageBin = StorageMetaData.Bin

// DeployStorage deploys a new Ethereum contract, binding an instance of Storage to it.
func DeployStorage(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Storage, error) {
	parsed, err := StorageMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StorageBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Storage{StorageCaller: StorageCaller{contract: contract}, StorageTransactor: StorageTransactor{contract: contract}, StorageFilterer: StorageFilterer{contract: contract}}, nil
}

// Storage is an auto generated Go binding around an Ethereum contract.
type Storage struct {
	StorageCaller     // Read-only binding to the contract
	StorageTransactor // Write-only binding to the contract
	StorageFilterer   // Log filterer for contract events
}

// StorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type StorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StorageSession struct {
	Contract     *Storage          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StorageCallerSession struct {
	Contract *StorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// StorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StorageTransactorSession struct {
	Contract     *StorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// StorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type StorageRaw struct {
	Contract *Storage // Generic contract binding to access the raw methods on
}

// StorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StorageCallerRaw struct {
	Contract *StorageCaller // Generic read-only contract binding to access the raw methods on
}

// StorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StorageTransactorRaw struct {
	Contract *StorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStorage creates a new instance of Storage, bound to a specific deployed contract.
func NewStorage(address common.Address, backend bind.ContractBackend) (*Storage, error) {
	contract, err := bindStorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Storage{StorageCaller: StorageCaller{contract: contract}, StorageTransactor: StorageTransactor{contract: contract}, StorageFilterer: StorageFilterer{contract: contract}}, nil
}

// NewStorageCaller creates a new read-only instance of Storage, bound to a specific deployed contract.
func NewStorageCaller(address common.Address, caller bind.ContractCaller) (*StorageCaller, error) {
	contract, err := bindStorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StorageCaller{contract: contract}, nil
}

// NewStorageTransactor creates a new write-only instance of Storage, bound to a specific deployed contract.
func NewStorageTransactor(address common.Address, transactor bind.ContractTransactor) (*StorageTransactor, error) {
	contract, err := bindStorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StorageTransactor{contract: contract}, nil
}

// NewStorageFilterer creates a new log filterer instance of Storage, bound to a specific deployed contract.
func NewStorageFilterer(address common.Address, filterer bind.ContractFilterer) (*StorageFilterer, error) {
	contract, err := bindStorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StorageFilterer{contract: contract}, nil
}

// bindStorage binds a generic wrapper to an already deployed contract.
func bindStorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StorageMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Storage *StorageRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Storage.Contract.StorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Storage *StorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Storage.Contract.StorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Storage *StorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Storage.Contract.StorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Storage *StorageCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Storage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Storage *StorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Storage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Storage *StorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Storage.Contract.contract.Transact(opts, method, params...)
}

// A is a free data retrieval call binding the contract method 0x0dbe671f.
//
// Solidity: function a() view returns(int256)
func (_Storage *StorageCaller) A(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "a")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// A is a free data retrieval call binding the contract method 0x0dbe671f.
//
// Solidity: function a() view returns(int256)
func (_Storage *StorageSession) A() (*big.Int, error) {
	return _Storage.Contract.A(&_Storage.CallOpts)
}

// A is a free data retrieval call binding the contract method 0x0dbe671f.
//
// Solidity: function a() view returns(int256)
func (_Storage *StorageCallerSession) A() (*big.Int, error) {
	return _Storage.Contract.A(&_Storage.CallOpts)
}

// B is a free data retrieval call binding the contract method 0x4df7e3d0.
//
// Solidity: function b() view returns(int256)
func (_Storage *StorageCaller) B(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "b")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// B is a free data retrieval call binding the contract method 0x4df7e3d0.
//
// Solidity: function b() view returns(int256)
func (_Storage *StorageSession) B() (*big.Int, error) {
	return _Storage.Contract.B(&_Storage.CallOpts)
}

// B is a free data retrieval call binding the contract method 0x4df7e3d0.
//
// Solidity: function b() view returns(int256)
func (_Storage *StorageCallerSession) B() (*big.Int, error) {
	return _Storage.Contract.B(&_Storage.CallOpts)
}

// C is a free data retrieval call binding the contract method 0xc3da42b8.
//
// Solidity: function c() view returns(int256)
func (_Storage *StorageCaller) C(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "c")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// C is a free data retrieval call binding the contract method 0xc3da42b8.
//
// Solidity: function c() view returns(int256)
func (_Storage *StorageSession) C() (*big.Int, error) {
	return _Storage.Contract.C(&_Storage.CallOpts)
}

// C is a free data retrieval call binding the contract method 0xc3da42b8.
//
// Solidity: function c() view returns(int256)
func (_Storage *StorageCallerSession) C() (*big.Int, error) {
	return _Storage.Contract.C(&_Storage.CallOpts)
}

// GetA is a free data retrieval call binding the contract method 0xd46300fd.
//
// Solidity: function getA() view returns(int256)
func (_Storage *StorageCaller) GetA(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "getA")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetA is a free data retrieval call binding the contract method 0xd46300fd.
//
// Solidity: function getA() view returns(int256)
func (_Storage *StorageSession) GetA() (*big.Int, error) {
	return _Storage.Contract.GetA(&_Storage.CallOpts)
}

// GetA is a free data retrieval call binding the contract method 0xd46300fd.
//
// Solidity: function getA() view returns(int256)
func (_Storage *StorageCallerSession) GetA() (*big.Int, error) {
	return _Storage.Contract.GetA(&_Storage.CallOpts)
}

// GetB is a free data retrieval call binding the contract method 0xa1c51915.
//
// Solidity: function getB() view returns(int256)
func (_Storage *StorageCaller) GetB(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "getB")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetB is a free data retrieval call binding the contract method 0xa1c51915.
//
// Solidity: function getB() view returns(int256)
func (_Storage *StorageSession) GetB() (*big.Int, error) {
	return _Storage.Contract.GetB(&_Storage.CallOpts)
}

// GetB is a free data retrieval call binding the contract method 0xa1c51915.
//
// Solidity: function getB() view returns(int256)
func (_Storage *StorageCallerSession) GetB() (*big.Int, error) {
	return _Storage.Contract.GetB(&_Storage.CallOpts)
}

// GetC is a free data retrieval call binding the contract method 0xa2375d1e.
//
// Solidity: function getC() view returns(int256)
func (_Storage *StorageCaller) GetC(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "getC")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetC is a free data retrieval call binding the contract method 0xa2375d1e.
//
// Solidity: function getC() view returns(int256)
func (_Storage *StorageSession) GetC() (*big.Int, error) {
	return _Storage.Contract.GetC(&_Storage.CallOpts)
}

// GetC is a free data retrieval call binding the contract method 0xa2375d1e.
//
// Solidity: function getC() view returns(int256)
func (_Storage *StorageCallerSession) GetC() (*big.Int, error) {
	return _Storage.Contract.GetC(&_Storage.CallOpts)
}

// SumABC is a free data retrieval call binding the contract method 0xbdd60569.
//
// Solidity: function sumABC() view returns(int256)
func (_Storage *StorageCaller) SumABC(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Storage.contract.Call(opts, &out, "sumABC")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SumABC is a free data retrieval call binding the contract method 0xbdd60569.
//
// Solidity: function sumABC() view returns(int256)
func (_Storage *StorageSession) SumABC() (*big.Int, error) {
	return _Storage.Contract.SumABC(&_Storage.CallOpts)
}

// SumABC is a free data retrieval call binding the contract method 0xbdd60569.
//
// Solidity: function sumABC() view returns(int256)
func (_Storage *StorageCallerSession) SumABC() (*big.Int, error) {
	return _Storage.Contract.SumABC(&_Storage.CallOpts)
}
