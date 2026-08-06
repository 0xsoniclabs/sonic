// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package magic_value_priority

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

// MagicValuePriorityMetaData contains all meta data concerning the MagicValuePriority contract.
var MagicValuePriorityMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"MAGIC_VALUE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103a68061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c80632067319614610043578063928461bd14610061578063d9dceeb814610080575b5f5ffd5b61004b6100b2565b6040516100589190610118565b60405180910390f35b6100696100ba565b604051610077929190610131565b60405180910390f35b61009a60048036038101906100959190610245565b6100cc565b6040516100a99392919061033b565b60405180910390f35b63075bcd1581565b5f5f633b9aca006103e8915091509091565b5f5f5f63075bcd1588036100e95760015f5f9250925092506100f3565b5f5f5f9250925092505b9750975097945050505050565b5f819050919050565b61011281610100565b82525050565b5f60208201905061012b5f830184610109565b92915050565b5f6040820190506101445f830185610109565b6101516020830184610109565b9392505050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61018982610160565b9050919050565b6101998161017f565b81146101a3575f5ffd5b50565b5f813590506101b481610190565b92915050565b6101c381610100565b81146101cd575f5ffd5b50565b5f813590506101de816101ba565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f840112610205576102046101e4565b5b8235905067ffffffffffffffff811115610222576102216101e8565b5b60208301915083600182028301111561023e5761023d6101ec565b5b9250929050565b5f5f5f5f5f5f5f60c0888a0312156102605761025f610158565b5b5f61026d8a828b016101a6565b975050602061027e8a828b016101a6565b965050604061028f8a828b016101d0565b95505060606102a08a828b016101d0565b945050608088013567ffffffffffffffff8111156102c1576102c061015c565b5b6102cd8a828b016101f0565b935093505060a06102e08a828b016101d0565b91505092959891949750929550565b5f67ffffffffffffffff82169050919050565b61030b816102ef565b82525050565b5f6fffffffffffffffffffffffffffffffff82169050919050565b61033581610311565b82525050565b5f60608201905061034e5f830186610302565b61035b6020830185610302565b610368604083018461032c565b94935050505056fea2646970667358221220362237f8a8af3ea5b63675f225bd8693e1bdfd9daf3ffca46d166fc0cc47ca3c64736f6c634300081e0033",
}

// MagicValuePriorityABI is the input ABI used to generate the binding from.
// Deprecated: Use MagicValuePriorityMetaData.ABI instead.
var MagicValuePriorityABI = MagicValuePriorityMetaData.ABI

// MagicValuePriorityBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MagicValuePriorityMetaData.Bin instead.
var MagicValuePriorityBin = MagicValuePriorityMetaData.Bin

// DeployMagicValuePriority deploys a new Ethereum contract, binding an instance of MagicValuePriority to it.
func DeployMagicValuePriority(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MagicValuePriority, error) {
	parsed, err := MagicValuePriorityMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MagicValuePriorityBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MagicValuePriority{MagicValuePriorityCaller: MagicValuePriorityCaller{contract: contract}, MagicValuePriorityTransactor: MagicValuePriorityTransactor{contract: contract}, MagicValuePriorityFilterer: MagicValuePriorityFilterer{contract: contract}}, nil
}

// MagicValuePriority is an auto generated Go binding around an Ethereum contract.
type MagicValuePriority struct {
	MagicValuePriorityCaller     // Read-only binding to the contract
	MagicValuePriorityTransactor // Write-only binding to the contract
	MagicValuePriorityFilterer   // Log filterer for contract events
}

// MagicValuePriorityCaller is an auto generated read-only Go binding around an Ethereum contract.
type MagicValuePriorityCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MagicValuePriorityTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MagicValuePriorityTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MagicValuePriorityFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MagicValuePriorityFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MagicValuePrioritySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MagicValuePrioritySession struct {
	Contract     *MagicValuePriority // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// MagicValuePriorityCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MagicValuePriorityCallerSession struct {
	Contract *MagicValuePriorityCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// MagicValuePriorityTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MagicValuePriorityTransactorSession struct {
	Contract     *MagicValuePriorityTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// MagicValuePriorityRaw is an auto generated low-level Go binding around an Ethereum contract.
type MagicValuePriorityRaw struct {
	Contract *MagicValuePriority // Generic contract binding to access the raw methods on
}

// MagicValuePriorityCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MagicValuePriorityCallerRaw struct {
	Contract *MagicValuePriorityCaller // Generic read-only contract binding to access the raw methods on
}

// MagicValuePriorityTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MagicValuePriorityTransactorRaw struct {
	Contract *MagicValuePriorityTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMagicValuePriority creates a new instance of MagicValuePriority, bound to a specific deployed contract.
func NewMagicValuePriority(address common.Address, backend bind.ContractBackend) (*MagicValuePriority, error) {
	contract, err := bindMagicValuePriority(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MagicValuePriority{MagicValuePriorityCaller: MagicValuePriorityCaller{contract: contract}, MagicValuePriorityTransactor: MagicValuePriorityTransactor{contract: contract}, MagicValuePriorityFilterer: MagicValuePriorityFilterer{contract: contract}}, nil
}

// NewMagicValuePriorityCaller creates a new read-only instance of MagicValuePriority, bound to a specific deployed contract.
func NewMagicValuePriorityCaller(address common.Address, caller bind.ContractCaller) (*MagicValuePriorityCaller, error) {
	contract, err := bindMagicValuePriority(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MagicValuePriorityCaller{contract: contract}, nil
}

// NewMagicValuePriorityTransactor creates a new write-only instance of MagicValuePriority, bound to a specific deployed contract.
func NewMagicValuePriorityTransactor(address common.Address, transactor bind.ContractTransactor) (*MagicValuePriorityTransactor, error) {
	contract, err := bindMagicValuePriority(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MagicValuePriorityTransactor{contract: contract}, nil
}

// NewMagicValuePriorityFilterer creates a new log filterer instance of MagicValuePriority, bound to a specific deployed contract.
func NewMagicValuePriorityFilterer(address common.Address, filterer bind.ContractFilterer) (*MagicValuePriorityFilterer, error) {
	contract, err := bindMagicValuePriority(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MagicValuePriorityFilterer{contract: contract}, nil
}

// bindMagicValuePriority binds a generic wrapper to an already deployed contract.
func bindMagicValuePriority(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MagicValuePriorityMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MagicValuePriority *MagicValuePriorityRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MagicValuePriority.Contract.MagicValuePriorityCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MagicValuePriority *MagicValuePriorityRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MagicValuePriority.Contract.MagicValuePriorityTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MagicValuePriority *MagicValuePriorityRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MagicValuePriority.Contract.MagicValuePriorityTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MagicValuePriority *MagicValuePriorityCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MagicValuePriority.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MagicValuePriority *MagicValuePriorityTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MagicValuePriority.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MagicValuePriority *MagicValuePriorityTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MagicValuePriority.Contract.contract.Transact(opts, method, params...)
}

// MAGICVALUE is a free data retrieval call binding the contract method 0x20673196.
//
// Solidity: function MAGIC_VALUE() view returns(uint256)
func (_MagicValuePriority *MagicValuePriorityCaller) MAGICVALUE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MagicValuePriority.contract.Call(opts, &out, "MAGIC_VALUE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAGICVALUE is a free data retrieval call binding the contract method 0x20673196.
//
// Solidity: function MAGIC_VALUE() view returns(uint256)
func (_MagicValuePriority *MagicValuePrioritySession) MAGICVALUE() (*big.Int, error) {
	return _MagicValuePriority.Contract.MAGICVALUE(&_MagicValuePriority.CallOpts)
}

// MAGICVALUE is a free data retrieval call binding the contract method 0x20673196.
//
// Solidity: function MAGIC_VALUE() view returns(uint256)
func (_MagicValuePriority *MagicValuePriorityCallerSession) MAGICVALUE() (*big.Int, error) {
	return _MagicValuePriority.Contract.MAGICVALUE(&_MagicValuePriority.CallOpts)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) pure returns(uint64 level, uint64 weight, uint128 id)
func (_MagicValuePriority *MagicValuePriorityCaller) GetPriority(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	var out []interface{}
	err := _MagicValuePriority.contract.Call(opts, &out, "getPriority", arg0, arg1, value, arg3, arg4, arg5)

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
func (_MagicValuePriority *MagicValuePrioritySession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _MagicValuePriority.Contract.GetPriority(&_MagicValuePriority.CallOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) pure returns(uint64 level, uint64 weight, uint128 id)
func (_MagicValuePriority *MagicValuePriorityCallerSession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _MagicValuePriority.Contract.GetPriority(&_MagicValuePriority.CallOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MagicValuePriority *MagicValuePriorityCaller) GetPriorityConfig(opts *bind.CallOpts) (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	var out []interface{}
	err := _MagicValuePriority.contract.Call(opts, &out, "getPriorityConfig")

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
func (_MagicValuePriority *MagicValuePrioritySession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _MagicValuePriority.Contract.GetPriorityConfig(&_MagicValuePriority.CallOpts)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MagicValuePriority *MagicValuePriorityCallerSession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _MagicValuePriority.Contract.GetPriorityConfig(&_MagicValuePriority.CallOpts)
}
