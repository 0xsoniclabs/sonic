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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"gas\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"level\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxGas\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"senderPriority\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"level\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"perBlockGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"perEvent\",\"type\":\"uint256\"}],\"name\":\"setConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"g\",\"type\":\"uint256\"}],\"name\":\"setMaxGas\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"level\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"setSenderPriority\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103af8061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061007a575f3560e01c80638e928076116100585780638e92807614610108578063928461bd1461011b578063d9dceeb814610138578063e3c1859d14610166575f5ffd5b80631e34c5851461007e578063501d815c1461009957806381afb106146100b5575b5f5ffd5b61009761008c366004610228565b600291909155600355565b005b6100a260015481565b6040519081526020015b60405180910390f35b6100976100c3366004610263565b6040805160608101825293845260208085019384528482019283526001600160a01b039095165f90815294859052909320915182555160018201559051600290910155565b610097610116366004610299565b600155565b610123610194565b604080519283526020830191909152016100ac565b61014b6101463660046102b0565b6101c8565b604080519384526020840192909252908201526060016100ac565b61014b610174366004610359565b5f6020819052908152604090208054600182015460029092015490919083565b5f5f6002545f146101a7576002546101ac565b629896805b91506003545f146101bf576003546101c2565b60045b90509091565b5f5f5f6001545f141580156101de575060015484115b156101f057505f91508190508061021b565b5050506001600160a01b0387165f908152602081905260409020805460018201546002909201549091905b9750975097945050505050565b5f5f60408385031215610239575f5ffd5b50508035926020909101359150565b80356001600160a01b038116811461025e575f5ffd5b919050565b5f5f5f5f60808587031215610276575f5ffd5b61027f85610248565b966020860135965060408601359560600135945092505050565b5f602082840312156102a9575f5ffd5b5035919050565b5f5f5f5f5f5f5f60c0888a0312156102c6575f5ffd5b6102cf88610248565b96506102dd60208901610248565b95506040880135945060608801359350608088013567ffffffffffffffff811115610306575f5ffd5b8801601f81018a13610316575f5ffd5b803567ffffffffffffffff81111561032c575f5ffd5b8a602082840101111561033d575f5ffd5b979a9699509497939660209095019560a0909401359392505050565b5f60208284031215610369575f5ffd5b61037282610248565b939250505056fea264697066735822122046cfa9646250caef5fc91a459572420c8cef930f382c5750be36d9e9ad37c83364736f6c634300081e0033",
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
// Solidity: function getPriority(address from, address , uint256 , uint256 , bytes , uint256 gas) view returns(uint256 level, uint256 weight, bytes32 id)
func (_Registry *RegistryCaller) GetPriority(opts *bind.CallOpts, from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, gas *big.Int) (struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getPriority", from, arg1, arg2, arg3, arg4, gas)

	outstruct := new(struct {
		Level  *big.Int
		Weight *big.Int
		Id     [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Level = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Weight = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Id = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address from, address , uint256 , uint256 , bytes , uint256 gas) view returns(uint256 level, uint256 weight, bytes32 id)
func (_Registry *RegistrySession) GetPriority(from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, gas *big.Int) (struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
}, error) {
	return _Registry.Contract.GetPriority(&_Registry.CallOpts, from, arg1, arg2, arg3, arg4, gas)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address from, address , uint256 , uint256 , bytes , uint256 gas) view returns(uint256 level, uint256 weight, bytes32 id)
func (_Registry *RegistryCallerSession) GetPriority(from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, gas *big.Int) (struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
}, error) {
	return _Registry.Contract.GetPriority(&_Registry.CallOpts, from, arg1, arg2, arg3, arg4, gas)
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

// MaxGas is a free data retrieval call binding the contract method 0x501d815c.
//
// Solidity: function maxGas() view returns(uint256)
func (_Registry *RegistryCaller) MaxGas(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "maxGas")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxGas is a free data retrieval call binding the contract method 0x501d815c.
//
// Solidity: function maxGas() view returns(uint256)
func (_Registry *RegistrySession) MaxGas() (*big.Int, error) {
	return _Registry.Contract.MaxGas(&_Registry.CallOpts)
}

// MaxGas is a free data retrieval call binding the contract method 0x501d815c.
//
// Solidity: function maxGas() view returns(uint256)
func (_Registry *RegistryCallerSession) MaxGas() (*big.Int, error) {
	return _Registry.Contract.MaxGas(&_Registry.CallOpts)
}

// SenderPriority is a free data retrieval call binding the contract method 0xe3c1859d.
//
// Solidity: function senderPriority(address ) view returns(uint256 level, uint256 weight, bytes32 id)
func (_Registry *RegistryCaller) SenderPriority(opts *bind.CallOpts, arg0 common.Address) (struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "senderPriority", arg0)

	outstruct := new(struct {
		Level  *big.Int
		Weight *big.Int
		Id     [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Level = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Weight = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Id = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// SenderPriority is a free data retrieval call binding the contract method 0xe3c1859d.
//
// Solidity: function senderPriority(address ) view returns(uint256 level, uint256 weight, bytes32 id)
func (_Registry *RegistrySession) SenderPriority(arg0 common.Address) (struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
}, error) {
	return _Registry.Contract.SenderPriority(&_Registry.CallOpts, arg0)
}

// SenderPriority is a free data retrieval call binding the contract method 0xe3c1859d.
//
// Solidity: function senderPriority(address ) view returns(uint256 level, uint256 weight, bytes32 id)
func (_Registry *RegistryCallerSession) SenderPriority(arg0 common.Address) (struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
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

// SetMaxGas is a paid mutator transaction binding the contract method 0x8e928076.
//
// Solidity: function setMaxGas(uint256 g) returns()
func (_Registry *RegistryTransactor) SetMaxGas(opts *bind.TransactOpts, g *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setMaxGas", g)
}

// SetMaxGas is a paid mutator transaction binding the contract method 0x8e928076.
//
// Solidity: function setMaxGas(uint256 g) returns()
func (_Registry *RegistrySession) SetMaxGas(g *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SetMaxGas(&_Registry.TransactOpts, g)
}

// SetMaxGas is a paid mutator transaction binding the contract method 0x8e928076.
//
// Solidity: function setMaxGas(uint256 g) returns()
func (_Registry *RegistryTransactorSession) SetMaxGas(g *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.SetMaxGas(&_Registry.TransactOpts, g)
}

// SetSenderPriority is a paid mutator transaction binding the contract method 0x81afb106.
//
// Solidity: function setSenderPriority(address from, uint256 level, uint256 weight, bytes32 id) returns()
func (_Registry *RegistryTransactor) SetSenderPriority(opts *bind.TransactOpts, from common.Address, level *big.Int, weight *big.Int, id [32]byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "setSenderPriority", from, level, weight, id)
}

// SetSenderPriority is a paid mutator transaction binding the contract method 0x81afb106.
//
// Solidity: function setSenderPriority(address from, uint256 level, uint256 weight, bytes32 id) returns()
func (_Registry *RegistrySession) SetSenderPriority(from common.Address, level *big.Int, weight *big.Int, id [32]byte) (*types.Transaction, error) {
	return _Registry.Contract.SetSenderPriority(&_Registry.TransactOpts, from, level, weight, id)
}

// SetSenderPriority is a paid mutator transaction binding the contract method 0x81afb106.
//
// Solidity: function setSenderPriority(address from, uint256 level, uint256 weight, bytes32 id) returns()
func (_Registry *RegistryTransactorSession) SetSenderPriority(from common.Address, level *big.Int, weight *big.Int, id [32]byte) (*types.Transaction, error) {
	return _Registry.Contract.SetSenderPriority(&_Registry.TransactOpts, from, level, weight, id)
}
