// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package add

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

// AddMetaData contains all meta data concerning the Add contract.
var AddMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"iter\",\"type\":\"uint256\"}],\"name\":\"add\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506101c88061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c80631003e2d21461002d575b5f5ffd5b610047600480360381019061004291906100cb565b61005d565b6040516100549190610105565b60405180910390f35b5f5f5f90505f5f90505b8381101561008a57818061007a9061014b565b9250508080600101915050610067565b5080915050919050565b5f5ffd5b5f819050919050565b6100aa81610098565b81146100b4575f5ffd5b50565b5f813590506100c5816100a1565b92915050565b5f602082840312156100e0576100df610094565b5b5f6100ed848285016100b7565b91505092915050565b6100ff81610098565b82525050565b5f6020820190506101185f8301846100f6565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61015582610098565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036101875761018661011e565b5b60018201905091905056fea26469706673582212209f6d6acc143c6d9476ecacf6739a417ed550f2876e34ea53b597c0fe75fc3e4864736f6c634300081e0033",
}

// AddABI is the input ABI used to generate the binding from.
// Deprecated: Use AddMetaData.ABI instead.
var AddABI = AddMetaData.ABI

// AddBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AddMetaData.Bin instead.
var AddBin = AddMetaData.Bin

// DeployAdd deploys a new Ethereum contract, binding an instance of Add to it.
func DeployAdd(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Add, error) {
	parsed, err := AddMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AddBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Add{AddCaller: AddCaller{contract: contract}, AddTransactor: AddTransactor{contract: contract}, AddFilterer: AddFilterer{contract: contract}}, nil
}

// Add is an auto generated Go binding around an Ethereum contract.
type Add struct {
	AddCaller     // Read-only binding to the contract
	AddTransactor // Write-only binding to the contract
	AddFilterer   // Log filterer for contract events
}

// AddCaller is an auto generated read-only Go binding around an Ethereum contract.
type AddCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AddTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AddFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AddSession struct {
	Contract     *Add              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AddCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AddCallerSession struct {
	Contract *AddCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// AddTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AddTransactorSession struct {
	Contract     *AddTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AddRaw is an auto generated low-level Go binding around an Ethereum contract.
type AddRaw struct {
	Contract *Add // Generic contract binding to access the raw methods on
}

// AddCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AddCallerRaw struct {
	Contract *AddCaller // Generic read-only contract binding to access the raw methods on
}

// AddTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AddTransactorRaw struct {
	Contract *AddTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAdd creates a new instance of Add, bound to a specific deployed contract.
func NewAdd(address common.Address, backend bind.ContractBackend) (*Add, error) {
	contract, err := bindAdd(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Add{AddCaller: AddCaller{contract: contract}, AddTransactor: AddTransactor{contract: contract}, AddFilterer: AddFilterer{contract: contract}}, nil
}

// NewAddCaller creates a new read-only instance of Add, bound to a specific deployed contract.
func NewAddCaller(address common.Address, caller bind.ContractCaller) (*AddCaller, error) {
	contract, err := bindAdd(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AddCaller{contract: contract}, nil
}

// NewAddTransactor creates a new write-only instance of Add, bound to a specific deployed contract.
func NewAddTransactor(address common.Address, transactor bind.ContractTransactor) (*AddTransactor, error) {
	contract, err := bindAdd(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AddTransactor{contract: contract}, nil
}

// NewAddFilterer creates a new log filterer instance of Add, bound to a specific deployed contract.
func NewAddFilterer(address common.Address, filterer bind.ContractFilterer) (*AddFilterer, error) {
	contract, err := bindAdd(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AddFilterer{contract: contract}, nil
}

// bindAdd binds a generic wrapper to an already deployed contract.
func bindAdd(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AddMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Add *AddRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Add.Contract.AddCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Add *AddRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Add.Contract.AddTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Add *AddRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Add.Contract.AddTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Add *AddCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Add.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Add *AddTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Add.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Add *AddTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Add.Contract.contract.Transact(opts, method, params...)
}

// Add is a free data retrieval call binding the contract method 0x1003e2d2.
//
// Solidity: function add(uint256 iter) pure returns(uint256)
func (_Add *AddCaller) Add(opts *bind.CallOpts, iter *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Add.contract.Call(opts, &out, "add", iter)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Add is a free data retrieval call binding the contract method 0x1003e2d2.
//
// Solidity: function add(uint256 iter) pure returns(uint256)
func (_Add *AddSession) Add(iter *big.Int) (*big.Int, error) {
	return _Add.Contract.Add(&_Add.CallOpts, iter)
}

// Add is a free data retrieval call binding the contract method 0x1003e2d2.
//
// Solidity: function add(uint256 iter) pure returns(uint256)
func (_Add *AddCallerSession) Add(iter *big.Int) (*big.Int, error) {
	return _Add.Contract.Add(&_Add.CallOpts, iter)
}
