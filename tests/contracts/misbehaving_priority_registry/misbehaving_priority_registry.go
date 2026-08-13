// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package misbehaving_priority_registry

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

// MisbehavingPriorityRegistryMetaData contains all meta data concerning the MisbehavingPriorityRegistry contract.
var MisbehavingPriorityRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"MAGIC_VALUE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"configMode\",\"outputs\":[{\"internalType\":\"enumMisbehavingPriorityRegistry.Mode\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"mode\",\"outputs\":[{\"internalType\":\"enumMisbehavingPriorityRegistry.Mode\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"mutated\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumMisbehavingPriorityRegistry.Mode\",\"name\":\"newMode\",\"type\":\"uint8\"}],\"name\":\"setConfigMode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumMisbehavingPriorityRegistry.Mode\",\"name\":\"newMode\",\"type\":\"uint8\"}],\"name\":\"setMode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610abf8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610086575f3560e01c8063928461bd11610059578063928461bd146101005780639a2941031461011f578063d9dceeb81461013d578063ff8fe2601461016f57610086565b8063206731961461008a57806321175b4a146100a8578063295a5212146100c457806391056cdf146100e2575b5f5ffd5b61009261018b565b60405161009f91906105d0565b60405180910390f35b6100c260048036038101906100bd9190610614565b610193565b005b6100cc6101be565b6040516100d991906106b2565b60405180910390f35b6100ea6101cf565b6040516100f791906106b2565b60405180910390f35b6101086101e1565b6040516101169291906106cb565b60405180910390f35b61012761035a565b604051610134919061070c565b60405180910390f35b6101576004803603810190610152919061080a565b61036c565b60405161016693929190610900565b60405180910390f35b61018960048036038101906101849190610614565b610505565b005b63075bcd1581565b805f5f6101000a81548160ff021916908360058111156101b6576101b561063f565b5b021790555050565b5f5f9054906101000a900460ff1681565b5f60019054906101000a900460ff1681565b5f5f5f5f60019054906101000a900460ff169050600460058111156102095761020861063f565b5b81600581111561021c5761021b61063f565b5b0361025c576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102539061098f565b60405180910390fd5b600160058111156102705761026f61063f565b5b8160058111156102835761028261063f565b5b0361029157610290610531565b5b6005808111156102a4576102a361063f565b5b8160058111156102b7576102b661063f565b5b036102d75760015f60026101000a81548160ff0219169083151502179055505b600260058111156102eb576102ea61063f565b5b8160058111156102fe576102fd61063f565b5b03610311575f195f525f1960205260405ff35b600360058111156103255761032461063f565b5b8160058111156103385761033761063f565b5b0361034957633b9aca005f5260205ff35b633b9aca006103e892509250509091565b5f60029054906101000a900460ff1681565b5f5f5f5f5f5f9054906101000a900460ff169050600460058111156103945761039361063f565b5b8160058111156103a7576103a661063f565b5b036103e7576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103de906109f7565b60405180910390fd5b600160058111156103fb576103fa61063f565b5b81600581111561040e5761040d61063f565b5b0361041c5761041b610531565b5b60058081111561042f5761042e61063f565b5b8160058111156104425761044161063f565b5b036104625760015f60026101000a81548160ff0219169083151502179055505b5f5f5f61046e8c61058a565b925092509250600260058111156104885761048761063f565b5b84600581111561049b5761049a61063f565b5b036104b3575f195f525f196020525f1960405260605ff35b600360058111156104c7576104c661063f565b5b8460058111156104da576104d961063f565b5b036104eb57825f528160205260405ff35b828282965096509650505050509750975097945050505050565b805f60016101000a81548160ff021916908360058111156105295761052861063f565b5b021790555050565b5f5f5f90505b612710811015610579578181604051602001610554929190610a5e565b6040516020818303038152906040528051906020012091508080600101915050610537565b505f5f1b8103610587575f5ffd5b50565b5f5f5f63075bcd1584036105a75760015f5f9250925092506105b1565b5f5f5f9250925092505b9193909250565b5f819050919050565b6105ca816105b8565b82525050565b5f6020820190506105e35f8301846105c1565b92915050565b5f5ffd5b5f5ffd5b600681106105fd575f5ffd5b50565b5f8135905061060e816105f1565b92915050565b5f60208284031215610629576106286105e9565b5b5f61063684828501610600565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602160045260245ffd5b6006811061067d5761067c61063f565b5b50565b5f81905061068d8261066c565b919050565b5f61069c82610680565b9050919050565b6106ac81610692565b82525050565b5f6020820190506106c55f8301846106a3565b92915050565b5f6040820190506106de5f8301856105c1565b6106eb60208301846105c1565b9392505050565b5f8115159050919050565b610706816106f2565b82525050565b5f60208201905061071f5f8301846106fd565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61074e82610725565b9050919050565b61075e81610744565b8114610768575f5ffd5b50565b5f8135905061077981610755565b92915050565b610788816105b8565b8114610792575f5ffd5b50565b5f813590506107a38161077f565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f8401126107ca576107c96107a9565b5b8235905067ffffffffffffffff8111156107e7576107e66107ad565b5b602083019150836001820283011115610803576108026107b1565b5b9250929050565b5f5f5f5f5f5f5f60c0888a031215610825576108246105e9565b5b5f6108328a828b0161076b565b97505060206108438a828b0161076b565b96505060406108548a828b01610795565b95505060606108658a828b01610795565b945050608088013567ffffffffffffffff811115610886576108856105ed565b5b6108928a828b016107b5565b935093505060a06108a58a828b01610795565b91505092959891949750929550565b5f67ffffffffffffffff82169050919050565b6108d0816108b4565b82525050565b5f6fffffffffffffffffffffffffffffffff82169050919050565b6108fa816108d6565b82525050565b5f6060820190506109135f8301866108c7565b61092060208301856108c7565b61092d60408301846108f1565b949350505050565b5f82825260208201905092915050565b7f6765745072696f72697479436f6e66696720616c77617973206661696c7300005f82015250565b5f610979601e83610935565b915061098482610945565b602082019050919050565b5f6020820190508181035f8301526109a68161096d565b9050919050565b7f6765745072696f7269747920616c77617973206661696c7300000000000000005f82015250565b5f6109e1601883610935565b91506109ec826109ad565b602082019050919050565b5f6020820190508181035f830152610a0e816109d5565b9050919050565b5f819050919050565b5f819050919050565b610a38610a3382610a15565b610a1e565b82525050565b5f819050919050565b610a58610a53826105b8565b610a3e565b82525050565b5f610a698285610a27565b602082019150610a798284610a47565b602082019150819050939250505056fea2646970667358221220fd2732d2a01e2330c666f398edc6686d934a9a3b6f8fe026842db4394094228a64736f6c634300081e0033",
}

// MisbehavingPriorityRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use MisbehavingPriorityRegistryMetaData.ABI instead.
var MisbehavingPriorityRegistryABI = MisbehavingPriorityRegistryMetaData.ABI

// MisbehavingPriorityRegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MisbehavingPriorityRegistryMetaData.Bin instead.
var MisbehavingPriorityRegistryBin = MisbehavingPriorityRegistryMetaData.Bin

// DeployMisbehavingPriorityRegistry deploys a new Ethereum contract, binding an instance of MisbehavingPriorityRegistry to it.
func DeployMisbehavingPriorityRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MisbehavingPriorityRegistry, error) {
	parsed, err := MisbehavingPriorityRegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MisbehavingPriorityRegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MisbehavingPriorityRegistry{MisbehavingPriorityRegistryCaller: MisbehavingPriorityRegistryCaller{contract: contract}, MisbehavingPriorityRegistryTransactor: MisbehavingPriorityRegistryTransactor{contract: contract}, MisbehavingPriorityRegistryFilterer: MisbehavingPriorityRegistryFilterer{contract: contract}}, nil
}

// MisbehavingPriorityRegistry is an auto generated Go binding around an Ethereum contract.
type MisbehavingPriorityRegistry struct {
	MisbehavingPriorityRegistryCaller     // Read-only binding to the contract
	MisbehavingPriorityRegistryTransactor // Write-only binding to the contract
	MisbehavingPriorityRegistryFilterer   // Log filterer for contract events
}

// MisbehavingPriorityRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type MisbehavingPriorityRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MisbehavingPriorityRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MisbehavingPriorityRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MisbehavingPriorityRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MisbehavingPriorityRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MisbehavingPriorityRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MisbehavingPriorityRegistrySession struct {
	Contract     *MisbehavingPriorityRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                // Call options to use throughout this session
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// MisbehavingPriorityRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MisbehavingPriorityRegistryCallerSession struct {
	Contract *MisbehavingPriorityRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                      // Call options to use throughout this session
}

// MisbehavingPriorityRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MisbehavingPriorityRegistryTransactorSession struct {
	Contract     *MisbehavingPriorityRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                      // Transaction auth options to use throughout this session
}

// MisbehavingPriorityRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type MisbehavingPriorityRegistryRaw struct {
	Contract *MisbehavingPriorityRegistry // Generic contract binding to access the raw methods on
}

// MisbehavingPriorityRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MisbehavingPriorityRegistryCallerRaw struct {
	Contract *MisbehavingPriorityRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// MisbehavingPriorityRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MisbehavingPriorityRegistryTransactorRaw struct {
	Contract *MisbehavingPriorityRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMisbehavingPriorityRegistry creates a new instance of MisbehavingPriorityRegistry, bound to a specific deployed contract.
func NewMisbehavingPriorityRegistry(address common.Address, backend bind.ContractBackend) (*MisbehavingPriorityRegistry, error) {
	contract, err := bindMisbehavingPriorityRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MisbehavingPriorityRegistry{MisbehavingPriorityRegistryCaller: MisbehavingPriorityRegistryCaller{contract: contract}, MisbehavingPriorityRegistryTransactor: MisbehavingPriorityRegistryTransactor{contract: contract}, MisbehavingPriorityRegistryFilterer: MisbehavingPriorityRegistryFilterer{contract: contract}}, nil
}

// NewMisbehavingPriorityRegistryCaller creates a new read-only instance of MisbehavingPriorityRegistry, bound to a specific deployed contract.
func NewMisbehavingPriorityRegistryCaller(address common.Address, caller bind.ContractCaller) (*MisbehavingPriorityRegistryCaller, error) {
	contract, err := bindMisbehavingPriorityRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MisbehavingPriorityRegistryCaller{contract: contract}, nil
}

// NewMisbehavingPriorityRegistryTransactor creates a new write-only instance of MisbehavingPriorityRegistry, bound to a specific deployed contract.
func NewMisbehavingPriorityRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*MisbehavingPriorityRegistryTransactor, error) {
	contract, err := bindMisbehavingPriorityRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MisbehavingPriorityRegistryTransactor{contract: contract}, nil
}

// NewMisbehavingPriorityRegistryFilterer creates a new log filterer instance of MisbehavingPriorityRegistry, bound to a specific deployed contract.
func NewMisbehavingPriorityRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*MisbehavingPriorityRegistryFilterer, error) {
	contract, err := bindMisbehavingPriorityRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MisbehavingPriorityRegistryFilterer{contract: contract}, nil
}

// bindMisbehavingPriorityRegistry binds a generic wrapper to an already deployed contract.
func bindMisbehavingPriorityRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MisbehavingPriorityRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MisbehavingPriorityRegistry.Contract.MisbehavingPriorityRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.MisbehavingPriorityRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.MisbehavingPriorityRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MisbehavingPriorityRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.contract.Transact(opts, method, params...)
}

// MAGICVALUE is a free data retrieval call binding the contract method 0x20673196.
//
// Solidity: function MAGIC_VALUE() view returns(uint256)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCaller) MAGICVALUE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MisbehavingPriorityRegistry.contract.Call(opts, &out, "MAGIC_VALUE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAGICVALUE is a free data retrieval call binding the contract method 0x20673196.
//
// Solidity: function MAGIC_VALUE() view returns(uint256)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) MAGICVALUE() (*big.Int, error) {
	return _MisbehavingPriorityRegistry.Contract.MAGICVALUE(&_MisbehavingPriorityRegistry.CallOpts)
}

// MAGICVALUE is a free data retrieval call binding the contract method 0x20673196.
//
// Solidity: function MAGIC_VALUE() view returns(uint256)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerSession) MAGICVALUE() (*big.Int, error) {
	return _MisbehavingPriorityRegistry.Contract.MAGICVALUE(&_MisbehavingPriorityRegistry.CallOpts)
}

// ConfigMode is a free data retrieval call binding the contract method 0x91056cdf.
//
// Solidity: function configMode() view returns(uint8)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCaller) ConfigMode(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MisbehavingPriorityRegistry.contract.Call(opts, &out, "configMode")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// ConfigMode is a free data retrieval call binding the contract method 0x91056cdf.
//
// Solidity: function configMode() view returns(uint8)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) ConfigMode() (uint8, error) {
	return _MisbehavingPriorityRegistry.Contract.ConfigMode(&_MisbehavingPriorityRegistry.CallOpts)
}

// ConfigMode is a free data retrieval call binding the contract method 0x91056cdf.
//
// Solidity: function configMode() view returns(uint8)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerSession) ConfigMode() (uint8, error) {
	return _MisbehavingPriorityRegistry.Contract.ConfigMode(&_MisbehavingPriorityRegistry.CallOpts)
}

// Mode is a free data retrieval call binding the contract method 0x295a5212.
//
// Solidity: function mode() view returns(uint8)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCaller) Mode(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MisbehavingPriorityRegistry.contract.Call(opts, &out, "mode")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Mode is a free data retrieval call binding the contract method 0x295a5212.
//
// Solidity: function mode() view returns(uint8)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) Mode() (uint8, error) {
	return _MisbehavingPriorityRegistry.Contract.Mode(&_MisbehavingPriorityRegistry.CallOpts)
}

// Mode is a free data retrieval call binding the contract method 0x295a5212.
//
// Solidity: function mode() view returns(uint8)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerSession) Mode() (uint8, error) {
	return _MisbehavingPriorityRegistry.Contract.Mode(&_MisbehavingPriorityRegistry.CallOpts)
}

// Mutated is a free data retrieval call binding the contract method 0x9a294103.
//
// Solidity: function mutated() view returns(bool)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCaller) Mutated(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _MisbehavingPriorityRegistry.contract.Call(opts, &out, "mutated")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Mutated is a free data retrieval call binding the contract method 0x9a294103.
//
// Solidity: function mutated() view returns(bool)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) Mutated() (bool, error) {
	return _MisbehavingPriorityRegistry.Contract.Mutated(&_MisbehavingPriorityRegistry.CallOpts)
}

// Mutated is a free data retrieval call binding the contract method 0x9a294103.
//
// Solidity: function mutated() view returns(bool)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerSession) Mutated() (bool, error) {
	return _MisbehavingPriorityRegistry.Contract.Mutated(&_MisbehavingPriorityRegistry.CallOpts)
}

// GetPriority is a paid mutator transaction binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) returns(uint64, uint64, uint128)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactor) GetPriority(opts *bind.TransactOpts, arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.contract.Transact(opts, "getPriority", arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriority is a paid mutator transaction binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) returns(uint64, uint64, uint128)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriority(&_MisbehavingPriorityRegistry.TransactOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriority is a paid mutator transaction binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) returns(uint64, uint64, uint128)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactorSession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriority(&_MisbehavingPriorityRegistry.TransactOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriorityConfig is a paid mutator transaction binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactor) GetPriorityConfig(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.contract.Transact(opts, "getPriorityConfig")
}

// GetPriorityConfig is a paid mutator transaction binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) GetPriorityConfig() (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriorityConfig(&_MisbehavingPriorityRegistry.TransactOpts)
}

// GetPriorityConfig is a paid mutator transaction binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactorSession) GetPriorityConfig() (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriorityConfig(&_MisbehavingPriorityRegistry.TransactOpts)
}

// SetConfigMode is a paid mutator transaction binding the contract method 0xff8fe260.
//
// Solidity: function setConfigMode(uint8 newMode) returns()
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactor) SetConfigMode(opts *bind.TransactOpts, newMode uint8) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.contract.Transact(opts, "setConfigMode", newMode)
}

// SetConfigMode is a paid mutator transaction binding the contract method 0xff8fe260.
//
// Solidity: function setConfigMode(uint8 newMode) returns()
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) SetConfigMode(newMode uint8) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.SetConfigMode(&_MisbehavingPriorityRegistry.TransactOpts, newMode)
}

// SetConfigMode is a paid mutator transaction binding the contract method 0xff8fe260.
//
// Solidity: function setConfigMode(uint8 newMode) returns()
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactorSession) SetConfigMode(newMode uint8) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.SetConfigMode(&_MisbehavingPriorityRegistry.TransactOpts, newMode)
}

// SetMode is a paid mutator transaction binding the contract method 0x21175b4a.
//
// Solidity: function setMode(uint8 newMode) returns()
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactor) SetMode(opts *bind.TransactOpts, newMode uint8) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.contract.Transact(opts, "setMode", newMode)
}

// SetMode is a paid mutator transaction binding the contract method 0x21175b4a.
//
// Solidity: function setMode(uint8 newMode) returns()
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) SetMode(newMode uint8) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.SetMode(&_MisbehavingPriorityRegistry.TransactOpts, newMode)
}

// SetMode is a paid mutator transaction binding the contract method 0x21175b4a.
//
// Solidity: function setMode(uint8 newMode) returns()
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryTransactorSession) SetMode(newMode uint8) (*types.Transaction, error) {
	return _MisbehavingPriorityRegistry.Contract.SetMode(&_MisbehavingPriorityRegistry.TransactOpts, newMode)
}
