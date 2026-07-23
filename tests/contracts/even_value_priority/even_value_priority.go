// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package even_value_priority

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

// EvenValuePriorityMetaData contains all meta data concerning the EvenValuePriority contract.
var EvenValuePriorityMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103c18061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063928461bd14610038578063d9dceeb814610057575b5f5ffd5b610040610089565b60405161004e9291906100ef565b60405180910390f35b610071600480360381019061006c9190610203565b61009b565b604051610080939291906102f9565b60405180910390f35b5f5f633b9aca006103e8915091509091565b5f5f5f5f6002896100ac919061035b565b036100c05760015f5f9250925092506100ca565b5f5f5f9250925092505b9750975097945050505050565b5f819050919050565b6100e9816100d7565b82525050565b5f6040820190506101025f8301856100e0565b61010f60208301846100e0565b9392505050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6101478261011e565b9050919050565b6101578161013d565b8114610161575f5ffd5b50565b5f813590506101728161014e565b92915050565b610181816100d7565b811461018b575f5ffd5b50565b5f8135905061019c81610178565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f8401126101c3576101c26101a2565b5b8235905067ffffffffffffffff8111156101e0576101df6101a6565b5b6020830191508360018202830111156101fc576101fb6101aa565b5b9250929050565b5f5f5f5f5f5f5f60c0888a03121561021e5761021d610116565b5b5f61022b8a828b01610164565b975050602061023c8a828b01610164565b965050604061024d8a828b0161018e565b955050606061025e8a828b0161018e565b945050608088013567ffffffffffffffff81111561027f5761027e61011a565b5b61028b8a828b016101ae565b935093505060a061029e8a828b0161018e565b91505092959891949750929550565b5f67ffffffffffffffff82169050919050565b6102c9816102ad565b82525050565b5f6fffffffffffffffffffffffffffffffff82169050919050565b6102f3816102cf565b82525050565b5f60608201905061030c5f8301866102c0565b61031960208301856102c0565b61032660408301846102ea565b949350505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601260045260245ffd5b5f610365826100d7565b9150610370836100d7565b9250826103805761037f61032e565b5b82820690509291505056fea2646970667358221220f009951b1aa335b727642ded8fd333d3fdca32ab2964db2bb392a419ba3314eb64736f6c634300081e0033",
}

// EvenValuePriorityABI is the input ABI used to generate the binding from.
// Deprecated: Use EvenValuePriorityMetaData.ABI instead.
var EvenValuePriorityABI = EvenValuePriorityMetaData.ABI

// EvenValuePriorityBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use EvenValuePriorityMetaData.Bin instead.
var EvenValuePriorityBin = EvenValuePriorityMetaData.Bin

// DeployEvenValuePriority deploys a new Ethereum contract, binding an instance of EvenValuePriority to it.
func DeployEvenValuePriority(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *EvenValuePriority, error) {
	parsed, err := EvenValuePriorityMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(EvenValuePriorityBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &EvenValuePriority{EvenValuePriorityCaller: EvenValuePriorityCaller{contract: contract}, EvenValuePriorityTransactor: EvenValuePriorityTransactor{contract: contract}, EvenValuePriorityFilterer: EvenValuePriorityFilterer{contract: contract}}, nil
}

// EvenValuePriority is an auto generated Go binding around an Ethereum contract.
type EvenValuePriority struct {
	EvenValuePriorityCaller     // Read-only binding to the contract
	EvenValuePriorityTransactor // Write-only binding to the contract
	EvenValuePriorityFilterer   // Log filterer for contract events
}

// EvenValuePriorityCaller is an auto generated read-only Go binding around an Ethereum contract.
type EvenValuePriorityCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EvenValuePriorityTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EvenValuePriorityTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EvenValuePriorityFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EvenValuePriorityFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EvenValuePrioritySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EvenValuePrioritySession struct {
	Contract     *EvenValuePriority // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// EvenValuePriorityCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EvenValuePriorityCallerSession struct {
	Contract *EvenValuePriorityCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// EvenValuePriorityTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EvenValuePriorityTransactorSession struct {
	Contract     *EvenValuePriorityTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// EvenValuePriorityRaw is an auto generated low-level Go binding around an Ethereum contract.
type EvenValuePriorityRaw struct {
	Contract *EvenValuePriority // Generic contract binding to access the raw methods on
}

// EvenValuePriorityCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EvenValuePriorityCallerRaw struct {
	Contract *EvenValuePriorityCaller // Generic read-only contract binding to access the raw methods on
}

// EvenValuePriorityTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EvenValuePriorityTransactorRaw struct {
	Contract *EvenValuePriorityTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEvenValuePriority creates a new instance of EvenValuePriority, bound to a specific deployed contract.
func NewEvenValuePriority(address common.Address, backend bind.ContractBackend) (*EvenValuePriority, error) {
	contract, err := bindEvenValuePriority(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &EvenValuePriority{EvenValuePriorityCaller: EvenValuePriorityCaller{contract: contract}, EvenValuePriorityTransactor: EvenValuePriorityTransactor{contract: contract}, EvenValuePriorityFilterer: EvenValuePriorityFilterer{contract: contract}}, nil
}

// NewEvenValuePriorityCaller creates a new read-only instance of EvenValuePriority, bound to a specific deployed contract.
func NewEvenValuePriorityCaller(address common.Address, caller bind.ContractCaller) (*EvenValuePriorityCaller, error) {
	contract, err := bindEvenValuePriority(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EvenValuePriorityCaller{contract: contract}, nil
}

// NewEvenValuePriorityTransactor creates a new write-only instance of EvenValuePriority, bound to a specific deployed contract.
func NewEvenValuePriorityTransactor(address common.Address, transactor bind.ContractTransactor) (*EvenValuePriorityTransactor, error) {
	contract, err := bindEvenValuePriority(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EvenValuePriorityTransactor{contract: contract}, nil
}

// NewEvenValuePriorityFilterer creates a new log filterer instance of EvenValuePriority, bound to a specific deployed contract.
func NewEvenValuePriorityFilterer(address common.Address, filterer bind.ContractFilterer) (*EvenValuePriorityFilterer, error) {
	contract, err := bindEvenValuePriority(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EvenValuePriorityFilterer{contract: contract}, nil
}

// bindEvenValuePriority binds a generic wrapper to an already deployed contract.
func bindEvenValuePriority(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := EvenValuePriorityMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EvenValuePriority *EvenValuePriorityRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EvenValuePriority.Contract.EvenValuePriorityCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EvenValuePriority *EvenValuePriorityRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EvenValuePriority.Contract.EvenValuePriorityTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EvenValuePriority *EvenValuePriorityRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EvenValuePriority.Contract.EvenValuePriorityTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_EvenValuePriority *EvenValuePriorityCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _EvenValuePriority.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_EvenValuePriority *EvenValuePriorityTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _EvenValuePriority.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_EvenValuePriority *EvenValuePriorityTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _EvenValuePriority.Contract.contract.Transact(opts, method, params...)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) pure returns(uint64 level, uint64 weight, uint128 id)
func (_EvenValuePriority *EvenValuePriorityCaller) GetPriority(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	var out []interface{}
	err := _EvenValuePriority.contract.Call(opts, &out, "getPriority", arg0, arg1, value, arg3, arg4, arg5)

	outstruct := new(struct {
		Level  uint64
		Weight uint64
		Id     *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Level = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.Weight = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Id = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) pure returns(uint64 level, uint64 weight, uint128 id)
func (_EvenValuePriority *EvenValuePrioritySession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _EvenValuePriority.Contract.GetPriority(&_EvenValuePriority.CallOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) pure returns(uint64 level, uint64 weight, uint128 id)
func (_EvenValuePriority *EvenValuePriorityCallerSession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _EvenValuePriority.Contract.GetPriority(&_EvenValuePriority.CallOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_EvenValuePriority *EvenValuePriorityCaller) GetPriorityConfig(opts *bind.CallOpts) (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	var out []interface{}
	err := _EvenValuePriority.contract.Call(opts, &out, "getPriorityConfig")

	outstruct := new(struct {
		MaxGasPerEntityPerBlock          *big.Int
		MaxPiggybackTxsPerEntityPerEvent *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.MaxGasPerEntityPerBlock = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.MaxPiggybackTxsPerEntityPerEvent = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_EvenValuePriority *EvenValuePrioritySession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _EvenValuePriority.Contract.GetPriorityConfig(&_EvenValuePriority.CallOpts)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_EvenValuePriority *EvenValuePriorityCallerSession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _EvenValuePriority.Contract.GetPriorityConfig(&_EvenValuePriority.CallOpts)
}
