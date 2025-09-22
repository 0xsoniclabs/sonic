// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gassubsidies_contract

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

// GassubsidiesContractMetaData contains all meta data concerning the GassubsidiesContract contract.
var GassubsidiesContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"contractSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"operationHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"operationHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"isCovered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"operationSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorContract\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"operationHash\",\"type\":\"bytes32\"}],\"name\":\"sponsorMethod\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorUser\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506105718061001c5f395ff3fe608060405260043610610079575f3560e01c8063aae831101161004c578063aae83110146100f8578063cbad49de14610139578063cc77aec81461016f578063daf21aa31461019a575f5ffd5b80630fd7e3751461007d57806327f6583b146100b15780632cc05157146100d25780638dd34a78146100e5575b5f5ffd5b348015610088575f5ffd5b5061009c61009736600461042f565b6101ad565b60405190151581526020015b60405180910390f35b3480156100bc575f5ffd5b506100d06100cb36600461042f565b610242565b005b6100d06100e036600461046e565b610376565b6100d06100f336600461049f565b6103b3565b348015610103575f5ffd5b5061012b61011236600461046e565b5f60208181529281526040808220909352908152205481565b6040519081526020016100a8565b348015610144575f5ffd5b5061012b61015336600461049f565b600160209081525f928352604080842090915290825290205481565b34801561017a575f5ffd5b5061012b6101893660046104c7565b60026020525f908152604090205481565b6100d06101a83660046104c7565b6103e5565b6001600160a01b038085165f9081526020818152604080832093871683529290529081205482116101e05750600161023a565b6001600160a01b0384165f90815260016020908152604080832086845290915290205482116102115750600161023a565b6001600160a01b0384165f9081526002602052604090205482116102375750600161023a565b505f5b949350505050565b3315610250576102506104e7565b61025c848484846101ad565b610268576102686104e7565b6001600160a01b038085165f908152602081815260408083209387168352929052205481116102d0576001600160a01b038085165f90815260208181526040808320938716835292905290812080548392906102c590849061050f565b909155506103709050565b6001600160a01b0383165f908152600160209081526040808320858452909152902054811161032b576001600160a01b0383165f908152600160209081526040808320858452909152812080548392906102c590849061050f565b6001600160a01b0383165f908152600260205260409020548111610370576001600160a01b0383165f90815260026020526040812080548392906102c590849061050f565b50505050565b6001600160a01b038083165f90815260208181526040808320938516835292905290812080543492906103aa908490610528565b90915550505050565b6001600160a01b0382165f908152600160209081526040808320848452909152812080543492906103aa908490610528565b6001600160a01b0381165f908152600260205260408120805434929061040c908490610528565b909155505050565b80356001600160a01b038116811461042a575f5ffd5b919050565b5f5f5f5f60808587031215610442575f5ffd5b61044b85610414565b935061045960208601610414565b93969395505050506040820135916060013590565b5f5f6040838503121561047f575f5ffd5b61048883610414565b915061049660208401610414565b90509250929050565b5f5f604083850312156104b0575f5ffd5b6104b983610414565b946020939093013593505050565b5f602082840312156104d7575f5ffd5b6104e082610414565b9392505050565b634e487b7160e01b5f52600160045260245ffd5b634e487b7160e01b5f52601160045260245ffd5b81810381811115610522576105226104fb565b92915050565b80820180821115610522576105226104fb56fea264697066735822122014a101d7be78eb2b19df248f3bb1dd3835184f06be712a125b00dc5a93f7d34c64736f6c634300081e0033",
}

// GassubsidiesContractABI is the input ABI used to generate the binding from.
// Deprecated: Use GassubsidiesContractMetaData.ABI instead.
var GassubsidiesContractABI = GassubsidiesContractMetaData.ABI

// GassubsidiesContractBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use GassubsidiesContractMetaData.Bin instead.
var GassubsidiesContractBin = GassubsidiesContractMetaData.Bin

// DeployGassubsidiesContract deploys a new Ethereum contract, binding an instance of GassubsidiesContract to it.
func DeployGassubsidiesContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *GassubsidiesContract, error) {
	parsed, err := GassubsidiesContractMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(GassubsidiesContractBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &GassubsidiesContract{GassubsidiesContractCaller: GassubsidiesContractCaller{contract: contract}, GassubsidiesContractTransactor: GassubsidiesContractTransactor{contract: contract}, GassubsidiesContractFilterer: GassubsidiesContractFilterer{contract: contract}}, nil
}

// GassubsidiesContract is an auto generated Go binding around an Ethereum contract.
type GassubsidiesContract struct {
	GassubsidiesContractCaller     // Read-only binding to the contract
	GassubsidiesContractTransactor // Write-only binding to the contract
	GassubsidiesContractFilterer   // Log filterer for contract events
}

// GassubsidiesContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type GassubsidiesContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GassubsidiesContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GassubsidiesContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GassubsidiesContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GassubsidiesContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GassubsidiesContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GassubsidiesContractSession struct {
	Contract     *GassubsidiesContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// GassubsidiesContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GassubsidiesContractCallerSession struct {
	Contract *GassubsidiesContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// GassubsidiesContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GassubsidiesContractTransactorSession struct {
	Contract     *GassubsidiesContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// GassubsidiesContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type GassubsidiesContractRaw struct {
	Contract *GassubsidiesContract // Generic contract binding to access the raw methods on
}

// GassubsidiesContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GassubsidiesContractCallerRaw struct {
	Contract *GassubsidiesContractCaller // Generic read-only contract binding to access the raw methods on
}

// GassubsidiesContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GassubsidiesContractTransactorRaw struct {
	Contract *GassubsidiesContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGassubsidiesContract creates a new instance of GassubsidiesContract, bound to a specific deployed contract.
func NewGassubsidiesContract(address common.Address, backend bind.ContractBackend) (*GassubsidiesContract, error) {
	contract, err := bindGassubsidiesContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GassubsidiesContract{GassubsidiesContractCaller: GassubsidiesContractCaller{contract: contract}, GassubsidiesContractTransactor: GassubsidiesContractTransactor{contract: contract}, GassubsidiesContractFilterer: GassubsidiesContractFilterer{contract: contract}}, nil
}

// NewGassubsidiesContractCaller creates a new read-only instance of GassubsidiesContract, bound to a specific deployed contract.
func NewGassubsidiesContractCaller(address common.Address, caller bind.ContractCaller) (*GassubsidiesContractCaller, error) {
	contract, err := bindGassubsidiesContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GassubsidiesContractCaller{contract: contract}, nil
}

// NewGassubsidiesContractTransactor creates a new write-only instance of GassubsidiesContract, bound to a specific deployed contract.
func NewGassubsidiesContractTransactor(address common.Address, transactor bind.ContractTransactor) (*GassubsidiesContractTransactor, error) {
	contract, err := bindGassubsidiesContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GassubsidiesContractTransactor{contract: contract}, nil
}

// NewGassubsidiesContractFilterer creates a new log filterer instance of GassubsidiesContract, bound to a specific deployed contract.
func NewGassubsidiesContractFilterer(address common.Address, filterer bind.ContractFilterer) (*GassubsidiesContractFilterer, error) {
	contract, err := bindGassubsidiesContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GassubsidiesContractFilterer{contract: contract}, nil
}

// bindGassubsidiesContract binds a generic wrapper to an already deployed contract.
func bindGassubsidiesContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := GassubsidiesContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GassubsidiesContract *GassubsidiesContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GassubsidiesContract.Contract.GassubsidiesContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GassubsidiesContract *GassubsidiesContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.GassubsidiesContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GassubsidiesContract *GassubsidiesContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.GassubsidiesContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GassubsidiesContract *GassubsidiesContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GassubsidiesContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GassubsidiesContract *GassubsidiesContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GassubsidiesContract *GassubsidiesContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.contract.Transact(opts, method, params...)
}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractCaller) ContractSponsorships(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _GassubsidiesContract.contract.Call(opts, &out, "contractSponsorships", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractSession) ContractSponsorships(arg0 common.Address) (*big.Int, error) {
	return _GassubsidiesContract.Contract.ContractSponsorships(&_GassubsidiesContract.CallOpts, arg0)
}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractCallerSession) ContractSponsorships(arg0 common.Address) (*big.Int, error) {
	return _GassubsidiesContract.Contract.ContractSponsorships(&_GassubsidiesContract.CallOpts, arg0)
}

// IsCovered is a free data retrieval call binding the contract method 0x0fd7e375.
//
// Solidity: function isCovered(address from, address to, bytes32 operationHash, uint256 fee) view returns(bool)
func (_GassubsidiesContract *GassubsidiesContractCaller) IsCovered(opts *bind.CallOpts, from common.Address, to common.Address, operationHash [32]byte, fee *big.Int) (bool, error) {
	var out []interface{}
	err := _GassubsidiesContract.contract.Call(opts, &out, "isCovered", from, to, operationHash, fee)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCovered is a free data retrieval call binding the contract method 0x0fd7e375.
//
// Solidity: function isCovered(address from, address to, bytes32 operationHash, uint256 fee) view returns(bool)
func (_GassubsidiesContract *GassubsidiesContractSession) IsCovered(from common.Address, to common.Address, operationHash [32]byte, fee *big.Int) (bool, error) {
	return _GassubsidiesContract.Contract.IsCovered(&_GassubsidiesContract.CallOpts, from, to, operationHash, fee)
}

// IsCovered is a free data retrieval call binding the contract method 0x0fd7e375.
//
// Solidity: function isCovered(address from, address to, bytes32 operationHash, uint256 fee) view returns(bool)
func (_GassubsidiesContract *GassubsidiesContractCallerSession) IsCovered(from common.Address, to common.Address, operationHash [32]byte, fee *big.Int) (bool, error) {
	return _GassubsidiesContract.Contract.IsCovered(&_GassubsidiesContract.CallOpts, from, to, operationHash, fee)
}

// OperationSponsorships is a free data retrieval call binding the contract method 0xcbad49de.
//
// Solidity: function operationSponsorships(address , bytes32 ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractCaller) OperationSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _GassubsidiesContract.contract.Call(opts, &out, "operationSponsorships", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OperationSponsorships is a free data retrieval call binding the contract method 0xcbad49de.
//
// Solidity: function operationSponsorships(address , bytes32 ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractSession) OperationSponsorships(arg0 common.Address, arg1 [32]byte) (*big.Int, error) {
	return _GassubsidiesContract.Contract.OperationSponsorships(&_GassubsidiesContract.CallOpts, arg0, arg1)
}

// OperationSponsorships is a free data retrieval call binding the contract method 0xcbad49de.
//
// Solidity: function operationSponsorships(address , bytes32 ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractCallerSession) OperationSponsorships(arg0 common.Address, arg1 [32]byte) (*big.Int, error) {
	return _GassubsidiesContract.Contract.OperationSponsorships(&_GassubsidiesContract.CallOpts, arg0, arg1)
}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractCaller) UserSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _GassubsidiesContract.contract.Call(opts, &out, "userSponsorships", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractSession) UserSponsorships(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _GassubsidiesContract.Contract.UserSponsorships(&_GassubsidiesContract.CallOpts, arg0, arg1)
}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256)
func (_GassubsidiesContract *GassubsidiesContractCallerSession) UserSponsorships(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _GassubsidiesContract.Contract.UserSponsorships(&_GassubsidiesContract.CallOpts, arg0, arg1)
}

// DeductFees is a paid mutator transaction binding the contract method 0x27f6583b.
//
// Solidity: function deductFees(address from, address to, bytes32 operationHash, uint256 fee) returns()
func (_GassubsidiesContract *GassubsidiesContractTransactor) DeductFees(opts *bind.TransactOpts, from common.Address, to common.Address, operationHash [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _GassubsidiesContract.contract.Transact(opts, "deductFees", from, to, operationHash, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0x27f6583b.
//
// Solidity: function deductFees(address from, address to, bytes32 operationHash, uint256 fee) returns()
func (_GassubsidiesContract *GassubsidiesContractSession) DeductFees(from common.Address, to common.Address, operationHash [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.DeductFees(&_GassubsidiesContract.TransactOpts, from, to, operationHash, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0x27f6583b.
//
// Solidity: function deductFees(address from, address to, bytes32 operationHash, uint256 fee) returns()
func (_GassubsidiesContract *GassubsidiesContractTransactorSession) DeductFees(from common.Address, to common.Address, operationHash [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.DeductFees(&_GassubsidiesContract.TransactOpts, from, to, operationHash, fee)
}

// SponsorContract is a paid mutator transaction binding the contract method 0xdaf21aa3.
//
// Solidity: function sponsorContract(address to) payable returns()
func (_GassubsidiesContract *GassubsidiesContractTransactor) SponsorContract(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _GassubsidiesContract.contract.Transact(opts, "sponsorContract", to)
}

// SponsorContract is a paid mutator transaction binding the contract method 0xdaf21aa3.
//
// Solidity: function sponsorContract(address to) payable returns()
func (_GassubsidiesContract *GassubsidiesContractSession) SponsorContract(to common.Address) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.SponsorContract(&_GassubsidiesContract.TransactOpts, to)
}

// SponsorContract is a paid mutator transaction binding the contract method 0xdaf21aa3.
//
// Solidity: function sponsorContract(address to) payable returns()
func (_GassubsidiesContract *GassubsidiesContractTransactorSession) SponsorContract(to common.Address) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.SponsorContract(&_GassubsidiesContract.TransactOpts, to)
}

// SponsorMethod is a paid mutator transaction binding the contract method 0x8dd34a78.
//
// Solidity: function sponsorMethod(address to, bytes32 operationHash) payable returns()
func (_GassubsidiesContract *GassubsidiesContractTransactor) SponsorMethod(opts *bind.TransactOpts, to common.Address, operationHash [32]byte) (*types.Transaction, error) {
	return _GassubsidiesContract.contract.Transact(opts, "sponsorMethod", to, operationHash)
}

// SponsorMethod is a paid mutator transaction binding the contract method 0x8dd34a78.
//
// Solidity: function sponsorMethod(address to, bytes32 operationHash) payable returns()
func (_GassubsidiesContract *GassubsidiesContractSession) SponsorMethod(to common.Address, operationHash [32]byte) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.SponsorMethod(&_GassubsidiesContract.TransactOpts, to, operationHash)
}

// SponsorMethod is a paid mutator transaction binding the contract method 0x8dd34a78.
//
// Solidity: function sponsorMethod(address to, bytes32 operationHash) payable returns()
func (_GassubsidiesContract *GassubsidiesContractTransactorSession) SponsorMethod(to common.Address, operationHash [32]byte) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.SponsorMethod(&_GassubsidiesContract.TransactOpts, to, operationHash)
}

// SponsorUser is a paid mutator transaction binding the contract method 0x2cc05157.
//
// Solidity: function sponsorUser(address from, address to) payable returns()
func (_GassubsidiesContract *GassubsidiesContractTransactor) SponsorUser(opts *bind.TransactOpts, from common.Address, to common.Address) (*types.Transaction, error) {
	return _GassubsidiesContract.contract.Transact(opts, "sponsorUser", from, to)
}

// SponsorUser is a paid mutator transaction binding the contract method 0x2cc05157.
//
// Solidity: function sponsorUser(address from, address to) payable returns()
func (_GassubsidiesContract *GassubsidiesContractSession) SponsorUser(from common.Address, to common.Address) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.SponsorUser(&_GassubsidiesContract.TransactOpts, from, to)
}

// SponsorUser is a paid mutator transaction binding the contract method 0x2cc05157.
//
// Solidity: function sponsorUser(address from, address to) payable returns()
func (_GassubsidiesContract *GassubsidiesContractTransactorSession) SponsorUser(from common.Address, to common.Address) (*types.Transaction, error) {
	return _GassubsidiesContract.Contract.SponsorUser(&_GassubsidiesContract.TransactOpts, from, to)
}
