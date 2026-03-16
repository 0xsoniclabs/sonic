// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package increasingly_expensive

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

// IncreasinglyExpensiveMetaData contains all meta data concerning the IncreasinglyExpensive contract.
var IncreasinglyExpensiveMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"incrementAndLoop\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405260015f553480156012575f5ffd5b50610194806100205f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c80630e1cd8f71461003857806361bc221a14610042575b5f5ffd5b610040610060565b005b61004a6100a1565b60405161005791906100be565b60405180910390f35b60015f5f8282546100719190610104565b925050819055505f5f5490505f8190505b5f81111561009d57808061009590610137565b915050610082565b5050565b5f5481565b5f819050919050565b6100b8816100a6565b82525050565b5f6020820190506100d15f8301846100af565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61010e826100a6565b9150610119836100a6565b9250828201905080821115610131576101306100d7565b5b92915050565b5f610141826100a6565b91505f8203610153576101526100d7565b5b60018203905091905056fea26469706673582212200baf971b580e9d3540f999607ba3917475c112cf35c28fa661cfe8a622dac76764736f6c634300081e0033",
}

// IncreasinglyExpensiveABI is the input ABI used to generate the binding from.
// Deprecated: Use IncreasinglyExpensiveMetaData.ABI instead.
var IncreasinglyExpensiveABI = IncreasinglyExpensiveMetaData.ABI

// IncreasinglyExpensiveBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use IncreasinglyExpensiveMetaData.Bin instead.
var IncreasinglyExpensiveBin = IncreasinglyExpensiveMetaData.Bin

// DeployIncreasinglyExpensive deploys a new Ethereum contract, binding an instance of IncreasinglyExpensive to it.
func DeployIncreasinglyExpensive(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *IncreasinglyExpensive, error) {
	parsed, err := IncreasinglyExpensiveMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(IncreasinglyExpensiveBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &IncreasinglyExpensive{IncreasinglyExpensiveCaller: IncreasinglyExpensiveCaller{contract: contract}, IncreasinglyExpensiveTransactor: IncreasinglyExpensiveTransactor{contract: contract}, IncreasinglyExpensiveFilterer: IncreasinglyExpensiveFilterer{contract: contract}}, nil
}

// IncreasinglyExpensive is an auto generated Go binding around an Ethereum contract.
type IncreasinglyExpensive struct {
	IncreasinglyExpensiveCaller     // Read-only binding to the contract
	IncreasinglyExpensiveTransactor // Write-only binding to the contract
	IncreasinglyExpensiveFilterer   // Log filterer for contract events
}

// IncreasinglyExpensiveCaller is an auto generated read-only Go binding around an Ethereum contract.
type IncreasinglyExpensiveCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IncreasinglyExpensiveTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IncreasinglyExpensiveTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IncreasinglyExpensiveFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IncreasinglyExpensiveFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IncreasinglyExpensiveSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IncreasinglyExpensiveSession struct {
	Contract     *IncreasinglyExpensive // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IncreasinglyExpensiveCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IncreasinglyExpensiveCallerSession struct {
	Contract *IncreasinglyExpensiveCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// IncreasinglyExpensiveTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IncreasinglyExpensiveTransactorSession struct {
	Contract     *IncreasinglyExpensiveTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// IncreasinglyExpensiveRaw is an auto generated low-level Go binding around an Ethereum contract.
type IncreasinglyExpensiveRaw struct {
	Contract *IncreasinglyExpensive // Generic contract binding to access the raw methods on
}

// IncreasinglyExpensiveCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IncreasinglyExpensiveCallerRaw struct {
	Contract *IncreasinglyExpensiveCaller // Generic read-only contract binding to access the raw methods on
}

// IncreasinglyExpensiveTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IncreasinglyExpensiveTransactorRaw struct {
	Contract *IncreasinglyExpensiveTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIncreasinglyExpensive creates a new instance of IncreasinglyExpensive, bound to a specific deployed contract.
func NewIncreasinglyExpensive(address common.Address, backend bind.ContractBackend) (*IncreasinglyExpensive, error) {
	contract, err := bindIncreasinglyExpensive(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IncreasinglyExpensive{IncreasinglyExpensiveCaller: IncreasinglyExpensiveCaller{contract: contract}, IncreasinglyExpensiveTransactor: IncreasinglyExpensiveTransactor{contract: contract}, IncreasinglyExpensiveFilterer: IncreasinglyExpensiveFilterer{contract: contract}}, nil
}

// NewIncreasinglyExpensiveCaller creates a new read-only instance of IncreasinglyExpensive, bound to a specific deployed contract.
func NewIncreasinglyExpensiveCaller(address common.Address, caller bind.ContractCaller) (*IncreasinglyExpensiveCaller, error) {
	contract, err := bindIncreasinglyExpensive(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IncreasinglyExpensiveCaller{contract: contract}, nil
}

// NewIncreasinglyExpensiveTransactor creates a new write-only instance of IncreasinglyExpensive, bound to a specific deployed contract.
func NewIncreasinglyExpensiveTransactor(address common.Address, transactor bind.ContractTransactor) (*IncreasinglyExpensiveTransactor, error) {
	contract, err := bindIncreasinglyExpensive(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IncreasinglyExpensiveTransactor{contract: contract}, nil
}

// NewIncreasinglyExpensiveFilterer creates a new log filterer instance of IncreasinglyExpensive, bound to a specific deployed contract.
func NewIncreasinglyExpensiveFilterer(address common.Address, filterer bind.ContractFilterer) (*IncreasinglyExpensiveFilterer, error) {
	contract, err := bindIncreasinglyExpensive(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IncreasinglyExpensiveFilterer{contract: contract}, nil
}

// bindIncreasinglyExpensive binds a generic wrapper to an already deployed contract.
func bindIncreasinglyExpensive(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IncreasinglyExpensiveMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IncreasinglyExpensive *IncreasinglyExpensiveRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IncreasinglyExpensive.Contract.IncreasinglyExpensiveCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IncreasinglyExpensive *IncreasinglyExpensiveRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IncreasinglyExpensive.Contract.IncreasinglyExpensiveTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IncreasinglyExpensive *IncreasinglyExpensiveRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IncreasinglyExpensive.Contract.IncreasinglyExpensiveTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IncreasinglyExpensive *IncreasinglyExpensiveCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IncreasinglyExpensive.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IncreasinglyExpensive *IncreasinglyExpensiveTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IncreasinglyExpensive.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IncreasinglyExpensive *IncreasinglyExpensiveTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IncreasinglyExpensive.Contract.contract.Transact(opts, method, params...)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_IncreasinglyExpensive *IncreasinglyExpensiveCaller) Counter(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _IncreasinglyExpensive.contract.Call(opts, &out, "counter")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_IncreasinglyExpensive *IncreasinglyExpensiveSession) Counter() (*big.Int, error) {
	return _IncreasinglyExpensive.Contract.Counter(&_IncreasinglyExpensive.CallOpts)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_IncreasinglyExpensive *IncreasinglyExpensiveCallerSession) Counter() (*big.Int, error) {
	return _IncreasinglyExpensive.Contract.Counter(&_IncreasinglyExpensive.CallOpts)
}

// IncrementAndLoop is a paid mutator transaction binding the contract method 0x0e1cd8f7.
//
// Solidity: function incrementAndLoop() returns()
func (_IncreasinglyExpensive *IncreasinglyExpensiveTransactor) IncrementAndLoop(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IncreasinglyExpensive.contract.Transact(opts, "incrementAndLoop")
}

// IncrementAndLoop is a paid mutator transaction binding the contract method 0x0e1cd8f7.
//
// Solidity: function incrementAndLoop() returns()
func (_IncreasinglyExpensive *IncreasinglyExpensiveSession) IncrementAndLoop() (*types.Transaction, error) {
	return _IncreasinglyExpensive.Contract.IncrementAndLoop(&_IncreasinglyExpensive.TransactOpts)
}

// IncrementAndLoop is a paid mutator transaction binding the contract method 0x0e1cd8f7.
//
// Solidity: function incrementAndLoop() returns()
func (_IncreasinglyExpensive *IncreasinglyExpensiveTransactorSession) IncrementAndLoop() (*types.Transaction, error) {
	return _IncreasinglyExpensive.Contract.IncrementAndLoop(&_IncreasinglyExpensive.TransactOpts)
}
