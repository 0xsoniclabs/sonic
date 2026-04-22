// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package store

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

// StoreMetaData contains all meta data concerning the Store contract.
var StoreMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"int256\",\"name\":\"from\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"to\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"value\",\"type\":\"int256\"}],\"name\":\"fill\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int256\",\"name\":\"key\",\"type\":\"int256\"}],\"name\":\"get\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCount\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int256\",\"name\":\"key\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"value\",\"type\":\"int256\"}],\"name\":\"put\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040525f5f553480156011575f5ffd5b506103e58061001f5f395ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80638110c48f1461004e578063846719e01461006a578063a87d942c1461009a578063e37f1e80146100b8575b5f5ffd5b6100686004803603810190610063919061025a565b6100d4565b005b610084600480360381019061007f9190610298565b61013f565b60405161009191906102d2565b60405180910390f35b6100a2610194565b6040516100af91906102d2565b60405180910390f35b6100d260048036038101906100cd91906102eb565b61019c565b005b8060015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8481526020019081526020015f20819055505f5f81548092919061013690610368565b91905055505050565b5f60015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8381526020019081526020015f20549050919050565b5f5f54905090565b5f8390505b82811215610207578160015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8381526020019081526020015f208190555080806001019150506101a1565b505f5f81548092919061021990610368565b9190505550505050565b5f5ffd5b5f819050919050565b61023981610227565b8114610243575f5ffd5b50565b5f8135905061025481610230565b92915050565b5f5f604083850312156102705761026f610223565b5b5f61027d85828601610246565b925050602061028e85828601610246565b9150509250929050565b5f602082840312156102ad576102ac610223565b5b5f6102ba84828501610246565b91505092915050565b6102cc81610227565b82525050565b5f6020820190506102e55f8301846102c3565b92915050565b5f5f5f6060848603121561030257610301610223565b5b5f61030f86828701610246565b935050602061032086828701610246565b925050604061033186828701610246565b9150509250925092565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61037282610227565b91507f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036103a4576103a361033b565b5b60018201905091905056fea26469706673582212201fa625b744e8fa6cdcb445fb731d2cf8d7d6f1381308c7fd5a0ea7466979b20164736f6c634300081e0033",
}

// StoreABI is the input ABI used to generate the binding from.
// Deprecated: Use StoreMetaData.ABI instead.
var StoreABI = StoreMetaData.ABI

// StoreBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StoreMetaData.Bin instead.
var StoreBin = StoreMetaData.Bin

// DeployStore deploys a new Ethereum contract, binding an instance of Store to it.
func DeployStore(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Store, error) {
	parsed, err := StoreMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StoreBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Store{StoreCaller: StoreCaller{contract: contract}, StoreTransactor: StoreTransactor{contract: contract}, StoreFilterer: StoreFilterer{contract: contract}}, nil
}

// Store is an auto generated Go binding around an Ethereum contract.
type Store struct {
	StoreCaller     // Read-only binding to the contract
	StoreTransactor // Write-only binding to the contract
	StoreFilterer   // Log filterer for contract events
}

// StoreCaller is an auto generated read-only Go binding around an Ethereum contract.
type StoreCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StoreTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StoreTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StoreFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StoreFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StoreSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StoreSession struct {
	Contract     *Store            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StoreCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StoreCallerSession struct {
	Contract *StoreCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// StoreTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StoreTransactorSession struct {
	Contract     *StoreTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StoreRaw is an auto generated low-level Go binding around an Ethereum contract.
type StoreRaw struct {
	Contract *Store // Generic contract binding to access the raw methods on
}

// StoreCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StoreCallerRaw struct {
	Contract *StoreCaller // Generic read-only contract binding to access the raw methods on
}

// StoreTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StoreTransactorRaw struct {
	Contract *StoreTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStore creates a new instance of Store, bound to a specific deployed contract.
func NewStore(address common.Address, backend bind.ContractBackend) (*Store, error) {
	contract, err := bindStore(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Store{StoreCaller: StoreCaller{contract: contract}, StoreTransactor: StoreTransactor{contract: contract}, StoreFilterer: StoreFilterer{contract: contract}}, nil
}

// NewStoreCaller creates a new read-only instance of Store, bound to a specific deployed contract.
func NewStoreCaller(address common.Address, caller bind.ContractCaller) (*StoreCaller, error) {
	contract, err := bindStore(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StoreCaller{contract: contract}, nil
}

// NewStoreTransactor creates a new write-only instance of Store, bound to a specific deployed contract.
func NewStoreTransactor(address common.Address, transactor bind.ContractTransactor) (*StoreTransactor, error) {
	contract, err := bindStore(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StoreTransactor{contract: contract}, nil
}

// NewStoreFilterer creates a new log filterer instance of Store, bound to a specific deployed contract.
func NewStoreFilterer(address common.Address, filterer bind.ContractFilterer) (*StoreFilterer, error) {
	contract, err := bindStore(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StoreFilterer{contract: contract}, nil
}

// bindStore binds a generic wrapper to an already deployed contract.
func bindStore(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StoreMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Store *StoreRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Store.Contract.StoreCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Store *StoreRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Store.Contract.StoreTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Store *StoreRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Store.Contract.StoreTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Store *StoreCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Store.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Store *StoreTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Store.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Store *StoreTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Store.Contract.contract.Transact(opts, method, params...)
}

// Get is a free data retrieval call binding the contract method 0x846719e0.
//
// Solidity: function get(int256 key) view returns(int256)
func (_Store *StoreCaller) Get(opts *bind.CallOpts, key *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Store.contract.Call(opts, &out, "get", key)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Get is a free data retrieval call binding the contract method 0x846719e0.
//
// Solidity: function get(int256 key) view returns(int256)
func (_Store *StoreSession) Get(key *big.Int) (*big.Int, error) {
	return _Store.Contract.Get(&_Store.CallOpts, key)
}

// Get is a free data retrieval call binding the contract method 0x846719e0.
//
// Solidity: function get(int256 key) view returns(int256)
func (_Store *StoreCallerSession) Get(key *big.Int) (*big.Int, error) {
	return _Store.Contract.Get(&_Store.CallOpts, key)
}

// GetCount is a free data retrieval call binding the contract method 0xa87d942c.
//
// Solidity: function getCount() view returns(int256)
func (_Store *StoreCaller) GetCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Store.contract.Call(opts, &out, "getCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCount is a free data retrieval call binding the contract method 0xa87d942c.
//
// Solidity: function getCount() view returns(int256)
func (_Store *StoreSession) GetCount() (*big.Int, error) {
	return _Store.Contract.GetCount(&_Store.CallOpts)
}

// GetCount is a free data retrieval call binding the contract method 0xa87d942c.
//
// Solidity: function getCount() view returns(int256)
func (_Store *StoreCallerSession) GetCount() (*big.Int, error) {
	return _Store.Contract.GetCount(&_Store.CallOpts)
}

// Fill is a paid mutator transaction binding the contract method 0xe37f1e80.
//
// Solidity: function fill(int256 from, int256 to, int256 value) returns()
func (_Store *StoreTransactor) Fill(opts *bind.TransactOpts, from *big.Int, to *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Store.contract.Transact(opts, "fill", from, to, value)
}

// Fill is a paid mutator transaction binding the contract method 0xe37f1e80.
//
// Solidity: function fill(int256 from, int256 to, int256 value) returns()
func (_Store *StoreSession) Fill(from *big.Int, to *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Store.Contract.Fill(&_Store.TransactOpts, from, to, value)
}

// Fill is a paid mutator transaction binding the contract method 0xe37f1e80.
//
// Solidity: function fill(int256 from, int256 to, int256 value) returns()
func (_Store *StoreTransactorSession) Fill(from *big.Int, to *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Store.Contract.Fill(&_Store.TransactOpts, from, to, value)
}

// Put is a paid mutator transaction binding the contract method 0x8110c48f.
//
// Solidity: function put(int256 key, int256 value) returns()
func (_Store *StoreTransactor) Put(opts *bind.TransactOpts, key *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Store.contract.Transact(opts, "put", key, value)
}

// Put is a paid mutator transaction binding the contract method 0x8110c48f.
//
// Solidity: function put(int256 key, int256 value) returns()
func (_Store *StoreSession) Put(key *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Store.Contract.Put(&_Store.TransactOpts, key, value)
}

// Put is a paid mutator transaction binding the contract method 0x8110c48f.
//
// Solidity: function put(int256 key, int256 value) returns()
func (_Store *StoreTransactorSession) Put(key *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Store.Contract.Put(&_Store.TransactOpts, key, value)
}
