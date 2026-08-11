// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

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

// RegistryMetaData contains all meta data concerning the Registry contract.
var RegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"senderPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"perBlockGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"perEvent\",\"type\":\"uint256\"}],\"name\":\"setConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"name\":\"setSenderPriority\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103e68061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610055575f3560e01c80631e34c58514610059578063392ebdff14610074578063928461bd14610110578063d9dceeb814610132578063e3c1859d146101bc575b5f5ffd5b610072610067366004610237565b600191909155600255565b005b610072610082366004610289565b6040805160608101825267ffffffffffffffff948516815292841660208085019182526001600160801b039384168584019081526001600160a01b039097165f908152908190529190912092518354915195518316600160801b02958516600160401b026fffffffffffffffffffffffffffffffff1990921694169390931792909217909116919091179055565b610118610203565b604080519283526020830191909152015b60405180910390f35b61018c6101403660046102e7565b505050506001600160a01b03929092165f9081526020819052604090205467ffffffffffffffff80821694600160401b83049091169350600160801b9091046001600160801b03169150565b6040805167ffffffffffffffff94851681529390921660208401526001600160801b031690820152606001610129565b61018c6101ca366004610390565b5f6020819052908152604090205467ffffffffffffffff80821691600160401b810490911690600160801b90046001600160801b031683565b5f5f6001545f146102165760015461021b565b629896805b91506002545f1461022e57600254610231565b60045b90509091565b5f5f60408385031215610248575f5ffd5b50508035926020909101359150565b80356001600160a01b038116811461026d575f5ffd5b919050565b803567ffffffffffffffff8116811461026d575f5ffd5b5f5f5f5f6080858703121561029c575f5ffd5b6102a585610257565b93506102b360208601610272565b92506102c160408601610272565b915060608501356001600160801b03811681146102dc575f5ffd5b939692955090935050565b5f5f5f5f5f5f5f60c0888a0312156102fd575f5ffd5b61030688610257565b965061031460208901610257565b95506040880135945060608801359350608088013567ffffffffffffffff81111561033d575f5ffd5b8801601f81018a1361034d575f5ffd5b803567ffffffffffffffff811115610363575f5ffd5b8a6020828401011115610374575f5ffd5b979a9699509497939660209095019560a0909401359392505050565b5f602082840312156103a0575f5ffd5b6103a982610257565b939250505056fea264697066735822122088608aa4558fb657bca2a1c650eb0463b8dd47c833f7f7a4c6ac7e92d4adbea464736f6c634300081e0033",
}

// RegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryMetaData.ABI instead.
var RegistryABI = RegistryMetaData.ABI

// RegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RegistryMetaData.Bin instead.
var RegistryBin = RegistryMetaData.Bin

// DeployRegistry deploys a new Ethereum contract, binding an instance of Registry to it.
func DeployRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Registry, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// Registry is an auto generated Go binding around an Ethereum contract.
type Registry struct {
	RegistryCaller     // Read-only binding to the contract
	RegistryTransactor // Write-only binding to the contract
	RegistryFilterer   // Log filterer for contract events
}

// RegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistrySession struct {
	Contract     *Registry         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryCallerSession struct {
	Contract *RegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryTransactorSession struct {
	Contract     *RegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryRaw struct {
	Contract *Registry // Generic contract binding to access the raw methods on
}

// RegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryCallerRaw struct {
	Contract *RegistryCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryTransactorRaw struct {
	Contract *RegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistry creates a new instance of Registry, bound to a specific deployed contract.
func NewRegistry(address common.Address, backend bind.ContractBackend) (*Registry, error) {
	contract, err := bindRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// NewRegistryCaller creates a new read-only instance of Registry, bound to a specific deployed contract.
func NewRegistryCaller(address common.Address, caller bind.ContractCaller) (*RegistryCaller, error) {
	contract, err := bindRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryCaller{contract: contract}, nil
}

// NewRegistryTransactor creates a new write-only instance of Registry, bound to a specific deployed contract.
func NewRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryTransactor, error) {
	contract, err := bindRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryTransactor{contract: contract}, nil
}

// NewRegistryFilterer creates a new log filterer instance of Registry, bound to a specific deployed contract.
func NewRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryFilterer, error) {
	contract, err := bindRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryFilterer{contract: contract}, nil
}

// bindRegistry binds a generic wrapper to an already deployed contract.
func bindRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.RegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transact(opts, method, params...)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address from, address , uint256 , uint256 , bytes , uint256 ) view returns(uint64 level, uint64 weight, uint128 id)
func (_Registry *RegistryCaller) GetPriority(opts *bind.CallOpts, from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getPriority", from, arg1, arg2, arg3, arg4, arg5)

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
// Solidity: function getPriority(address from, address , uint256 , uint256 , bytes , uint256 ) view returns(uint64 level, uint64 weight, uint128 id)
func (_Registry *RegistrySession) GetPriority(from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _Registry.Contract.GetPriority(&_Registry.CallOpts, from, arg1, arg2, arg3, arg4, arg5)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address from, address , uint256 , uint256 , bytes , uint256 ) view returns(uint64 level, uint64 weight, uint128 id)
func (_Registry *RegistryCallerSession) GetPriority(from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _Registry.Contract.GetPriority(&_Registry.CallOpts, from, arg1, arg2, arg3, arg4, arg5)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() view returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_Registry *RegistryCaller) GetPriorityConfig(opts *bind.CallOpts) (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getPriorityConfig")

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
// Solidity: function getPriorityConfig() view returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_Registry *RegistrySession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _Registry.Contract.GetPriorityConfig(&_Registry.CallOpts)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() view returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_Registry *RegistryCallerSession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _Registry.Contract.GetPriorityConfig(&_Registry.CallOpts)
}

// SenderPriority is a free data retrieval call binding the contract method 0xe3c1859d.
//
// Solidity: function senderPriority(address ) view returns(uint64 level, uint64 weight, uint128 id)
func (_Registry *RegistryCaller) SenderPriority(opts *bind.CallOpts, arg0 common.Address) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "senderPriority", arg0)

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

// SenderPriority is a free data retrieval call binding the contract method 0xe3c1859d.
//
// Solidity: function senderPriority(address ) view returns(uint64 level, uint64 weight, uint128 id)
func (_Registry *RegistrySession) SenderPriority(arg0 common.Address) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _Registry.Contract.SenderPriority(&_Registry.CallOpts, arg0)
}

// SenderPriority is a free data retrieval call binding the contract method 0xe3c1859d.
//
// Solidity: function senderPriority(address ) view returns(uint64 level, uint64 weight, uint128 id)
func (_Registry *RegistryCallerSession) SenderPriority(arg0 common.Address) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _Registry.Contract.SenderPriority(&_Registry.CallOpts, arg0)
}

// SetConfig is a paid mutator transaction binding the contract method 0x1e34c585.
//
// Solidity: function setConfig(uint256 perBlockGas, uint256 perEvent) returns()
func (_Registry *RegistryTransactor) SetConfig(opts *bind.TransactOpts, perBlockGas *big.Int, perEvent *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setConfig", perBlockGas, perEvent)
}

// SetConfig is a paid mutator transaction binding the contract method 0x1e34c585.
//
// Solidity: function setConfig(uint256 perBlockGas, uint256 perEvent) returns()
func (_Registry *RegistrySession) SetConfig(perBlockGas *big.Int, perEvent *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SetConfig(&_Registry.TransactOpts, perBlockGas, perEvent)
}

// SetConfig is a paid mutator transaction binding the contract method 0x1e34c585.
//
// Solidity: function setConfig(uint256 perBlockGas, uint256 perEvent) returns()
func (_Registry *RegistryTransactorSession) SetConfig(perBlockGas *big.Int, perEvent *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SetConfig(&_Registry.TransactOpts, perBlockGas, perEvent)
}

// SetSenderPriority is a paid mutator transaction binding the contract method 0x392ebdff.
//
// Solidity: function setSenderPriority(address from, uint64 level, uint64 weight, uint128 id) returns()
func (_Registry *RegistryTransactor) SetSenderPriority(opts *bind.TransactOpts, from common.Address, level uint64, weight uint64, id *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setSenderPriority", from, level, weight, id)
}

// SetSenderPriority is a paid mutator transaction binding the contract method 0x392ebdff.
//
// Solidity: function setSenderPriority(address from, uint64 level, uint64 weight, uint128 id) returns()
func (_Registry *RegistrySession) SetSenderPriority(from common.Address, level uint64, weight uint64, id *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SetSenderPriority(&_Registry.TransactOpts, from, level, weight, id)
}

// SetSenderPriority is a paid mutator transaction binding the contract method 0x392ebdff.
//
// Solidity: function setSenderPriority(address from, uint64 level, uint64 weight, uint128 id) returns()
func (_Registry *RegistryTransactorSession) SetSenderPriority(from common.Address, level uint64, weight uint64, id *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SetSenderPriority(&_Registry.TransactOpts, from, level, weight, id)
}
