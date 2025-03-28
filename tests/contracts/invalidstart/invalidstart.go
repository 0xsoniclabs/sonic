// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package invalidstart

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

// InvalidstartMetaData contains all meta data concerning the Invalidstart contract.
var InvalidstartMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"create2ContractWithInvalidCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"create2ContractWithValidCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createContractWithInvalidCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createContractWithValidCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"invalidBytecode\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"validBytecode\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040526040518060400160405280600a81526020017f60ef60005360016000f3000000000000000000000000000000000000000000008152505f908161004791906102d8565b506040518060400160405280600a81526020017f60fe60005360016000f3000000000000000000000000000000000000000000008152506001908161008c91906102d8565b50348015610098575f80fd5b506103a7565b5f81519050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061011957607f821691505b60208210810361012c5761012b6100d5565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f6008830261018e7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82610153565b6101988683610153565b95508019841693508086168417925050509392505050565b5f819050919050565b5f819050919050565b5f6101dc6101d76101d2846101b0565b6101b9565b6101b0565b9050919050565b5f819050919050565b6101f5836101c2565b610209610201826101e3565b84845461015f565b825550505050565b5f90565b61021d610211565b6102288184846101ec565b505050565b5b8181101561024b576102405f82610215565b60018101905061022e565b5050565b601f8211156102905761026181610132565b61026a84610144565b81016020851015610279578190505b61028d61028585610144565b83018261022d565b50505b505050565b5f82821c905092915050565b5f6102b05f1984600802610295565b1980831691505092915050565b5f6102c883836102a1565b9150826002028217905092915050565b6102e18261009e565b67ffffffffffffffff8111156102fa576102f96100a8565b5b6103048254610102565b61030f82828561024f565b5f60209050601f831160018114610340575f841561032e578287015190505b61033885826102bd565b86555061039f565b601f19841661034e86610132565b5f5b8281101561037557848901518255600182019150602085019450602081019050610350565b86831015610392578489015161038e601f8916826102a1565b8355505b6001600288020188555050505b505050505050565b6105b7806103b45f395ff3fe608060405234801561000f575f80fd5b5060043610610060575f3560e01c806302fc6a88146100645780630da11c4b1461006e5780634652ff071461008c57806389c0ea30146100aa578063a2990d42146100b4578063f3245b9a146100be575b5f80fd5b61006c6100c8565b005b61007661015b565b60405161008391906104de565b60405180910390f35b6100946101e6565b6040516100a191906104de565b60405180910390f35b6100b2610272565b005b6100bc610305565b005b6100c6610397565b005b610159600180546100d89061052b565b80601f01602080910402602001604051908101604052809291908181526020018280546101049061052b565b801561014f5780601f106101265761010080835404028352916020019161014f565b820191905f5260205f20905b81548152906001019060200180831161013257829003601f168201915b5050505050610429565b565b5f80546101679061052b565b80601f01602080910402602001604051908101604052809291908181526020018280546101939061052b565b80156101de5780601f106101b5576101008083540402835291602001916101de565b820191905f5260205f20905b8154815290600101906020018083116101c157829003601f168201915b505050505081565b600180546101f39061052b565b80601f016020809104026020016040519081016040528092919081815260200182805461021f9061052b565b801561026a5780601f106102415761010080835404028352916020019161026a565b820191905f5260205f20905b81548152906001019060200180831161024d57829003601f168201915b505050505081565b610303600180546102829061052b565b80601f01602080910402602001604051908101604052809291908181526020018280546102ae9061052b565b80156102f95780601f106102d0576101008083540402835291602001916102f9565b820191905f5260205f20905b8154815290600101906020018083116102dc57829003601f168201915b505050505061043e565b565b6103955f80546103149061052b565b80601f01602080910402602001604051908101604052809291908181526020018280546103409061052b565b801561038b5780601f106103625761010080835404028352916020019161038b565b820191905f5260205f20905b81548152906001019060200180831161036e57829003601f168201915b5050505050610429565b565b6104275f80546103a69061052b565b80601f01602080910402602001604051908101604052809291908181526020018280546103d29061052b565b801561041d5780601f106103f45761010080835404028352916020019161041d565b820191905f5260205f20905b81548152906001019060200180831161040057829003601f168201915b505050505061043e565b565b8051602082015ff08061043a575f80fd5b5050565b5f8151602083015ff580610450575f80fd5b5050565b5f81519050919050565b5f82825260208201905092915050565b5f5b8381101561048b578082015181840152602081019050610470565b5f8484015250505050565b5f601f19601f8301169050919050565b5f6104b082610454565b6104ba818561045e565b93506104ca81856020860161046e565b6104d381610496565b840191505092915050565b5f6020820190508181035f8301526104f681846104a6565b905092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061054257607f821691505b602082108103610555576105546104fe565b5b5091905056fea26469706673582212200547fde2d710837b8d1adee02981a17b45d7191cb3b2b5a32aceedad8ded930064736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
}

// InvalidstartABI is the input ABI used to generate the binding from.
// Deprecated: Use InvalidstartMetaData.ABI instead.
var InvalidstartABI = InvalidstartMetaData.ABI

// InvalidstartBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InvalidstartMetaData.Bin instead.
var InvalidstartBin = InvalidstartMetaData.Bin

// DeployInvalidstart deploys a new Ethereum contract, binding an instance of Invalidstart to it.
func DeployInvalidstart(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Invalidstart, error) {
	parsed, err := InvalidstartMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InvalidstartBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Invalidstart{InvalidstartCaller: InvalidstartCaller{contract: contract}, InvalidstartTransactor: InvalidstartTransactor{contract: contract}, InvalidstartFilterer: InvalidstartFilterer{contract: contract}}, nil
}

// Invalidstart is an auto generated Go binding around an Ethereum contract.
type Invalidstart struct {
	InvalidstartCaller     // Read-only binding to the contract
	InvalidstartTransactor // Write-only binding to the contract
	InvalidstartFilterer   // Log filterer for contract events
}

// InvalidstartCaller is an auto generated read-only Go binding around an Ethereum contract.
type InvalidstartCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvalidstartTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InvalidstartTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvalidstartFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InvalidstartFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvalidstartSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InvalidstartSession struct {
	Contract     *Invalidstart     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InvalidstartCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InvalidstartCallerSession struct {
	Contract *InvalidstartCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// InvalidstartTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InvalidstartTransactorSession struct {
	Contract     *InvalidstartTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// InvalidstartRaw is an auto generated low-level Go binding around an Ethereum contract.
type InvalidstartRaw struct {
	Contract *Invalidstart // Generic contract binding to access the raw methods on
}

// InvalidstartCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InvalidstartCallerRaw struct {
	Contract *InvalidstartCaller // Generic read-only contract binding to access the raw methods on
}

// InvalidstartTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InvalidstartTransactorRaw struct {
	Contract *InvalidstartTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInvalidstart creates a new instance of Invalidstart, bound to a specific deployed contract.
func NewInvalidstart(address common.Address, backend bind.ContractBackend) (*Invalidstart, error) {
	contract, err := bindInvalidstart(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Invalidstart{InvalidstartCaller: InvalidstartCaller{contract: contract}, InvalidstartTransactor: InvalidstartTransactor{contract: contract}, InvalidstartFilterer: InvalidstartFilterer{contract: contract}}, nil
}

// NewInvalidstartCaller creates a new read-only instance of Invalidstart, bound to a specific deployed contract.
func NewInvalidstartCaller(address common.Address, caller bind.ContractCaller) (*InvalidstartCaller, error) {
	contract, err := bindInvalidstart(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InvalidstartCaller{contract: contract}, nil
}

// NewInvalidstartTransactor creates a new write-only instance of Invalidstart, bound to a specific deployed contract.
func NewInvalidstartTransactor(address common.Address, transactor bind.ContractTransactor) (*InvalidstartTransactor, error) {
	contract, err := bindInvalidstart(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InvalidstartTransactor{contract: contract}, nil
}

// NewInvalidstartFilterer creates a new log filterer instance of Invalidstart, bound to a specific deployed contract.
func NewInvalidstartFilterer(address common.Address, filterer bind.ContractFilterer) (*InvalidstartFilterer, error) {
	contract, err := bindInvalidstart(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InvalidstartFilterer{contract: contract}, nil
}

// bindInvalidstart binds a generic wrapper to an already deployed contract.
func bindInvalidstart(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InvalidstartMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Invalidstart *InvalidstartRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Invalidstart.Contract.InvalidstartCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Invalidstart *InvalidstartRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invalidstart.Contract.InvalidstartTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Invalidstart *InvalidstartRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Invalidstart.Contract.InvalidstartTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Invalidstart *InvalidstartCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Invalidstart.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Invalidstart *InvalidstartTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invalidstart.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Invalidstart *InvalidstartTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Invalidstart.Contract.contract.Transact(opts, method, params...)
}

// InvalidBytecode is a free data retrieval call binding the contract method 0x0da11c4b.
//
// Solidity: function invalidBytecode() view returns(bytes)
func (_Invalidstart *InvalidstartCaller) InvalidBytecode(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _Invalidstart.contract.Call(opts, &out, "invalidBytecode")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// InvalidBytecode is a free data retrieval call binding the contract method 0x0da11c4b.
//
// Solidity: function invalidBytecode() view returns(bytes)
func (_Invalidstart *InvalidstartSession) InvalidBytecode() ([]byte, error) {
	return _Invalidstart.Contract.InvalidBytecode(&_Invalidstart.CallOpts)
}

// InvalidBytecode is a free data retrieval call binding the contract method 0x0da11c4b.
//
// Solidity: function invalidBytecode() view returns(bytes)
func (_Invalidstart *InvalidstartCallerSession) InvalidBytecode() ([]byte, error) {
	return _Invalidstart.Contract.InvalidBytecode(&_Invalidstart.CallOpts)
}

// ValidBytecode is a free data retrieval call binding the contract method 0x4652ff07.
//
// Solidity: function validBytecode() view returns(bytes)
func (_Invalidstart *InvalidstartCaller) ValidBytecode(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _Invalidstart.contract.Call(opts, &out, "validBytecode")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// ValidBytecode is a free data retrieval call binding the contract method 0x4652ff07.
//
// Solidity: function validBytecode() view returns(bytes)
func (_Invalidstart *InvalidstartSession) ValidBytecode() ([]byte, error) {
	return _Invalidstart.Contract.ValidBytecode(&_Invalidstart.CallOpts)
}

// ValidBytecode is a free data retrieval call binding the contract method 0x4652ff07.
//
// Solidity: function validBytecode() view returns(bytes)
func (_Invalidstart *InvalidstartCallerSession) ValidBytecode() ([]byte, error) {
	return _Invalidstart.Contract.ValidBytecode(&_Invalidstart.CallOpts)
}

// Create2ContractWithInvalidCode is a paid mutator transaction binding the contract method 0xf3245b9a.
//
// Solidity: function create2ContractWithInvalidCode() returns()
func (_Invalidstart *InvalidstartTransactor) Create2ContractWithInvalidCode(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invalidstart.contract.Transact(opts, "create2ContractWithInvalidCode")
}

// Create2ContractWithInvalidCode is a paid mutator transaction binding the contract method 0xf3245b9a.
//
// Solidity: function create2ContractWithInvalidCode() returns()
func (_Invalidstart *InvalidstartSession) Create2ContractWithInvalidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.Create2ContractWithInvalidCode(&_Invalidstart.TransactOpts)
}

// Create2ContractWithInvalidCode is a paid mutator transaction binding the contract method 0xf3245b9a.
//
// Solidity: function create2ContractWithInvalidCode() returns()
func (_Invalidstart *InvalidstartTransactorSession) Create2ContractWithInvalidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.Create2ContractWithInvalidCode(&_Invalidstart.TransactOpts)
}

// Create2ContractWithValidCode is a paid mutator transaction binding the contract method 0x89c0ea30.
//
// Solidity: function create2ContractWithValidCode() returns()
func (_Invalidstart *InvalidstartTransactor) Create2ContractWithValidCode(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invalidstart.contract.Transact(opts, "create2ContractWithValidCode")
}

// Create2ContractWithValidCode is a paid mutator transaction binding the contract method 0x89c0ea30.
//
// Solidity: function create2ContractWithValidCode() returns()
func (_Invalidstart *InvalidstartSession) Create2ContractWithValidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.Create2ContractWithValidCode(&_Invalidstart.TransactOpts)
}

// Create2ContractWithValidCode is a paid mutator transaction binding the contract method 0x89c0ea30.
//
// Solidity: function create2ContractWithValidCode() returns()
func (_Invalidstart *InvalidstartTransactorSession) Create2ContractWithValidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.Create2ContractWithValidCode(&_Invalidstart.TransactOpts)
}

// CreateContractWithInvalidCode is a paid mutator transaction binding the contract method 0xa2990d42.
//
// Solidity: function createContractWithInvalidCode() returns()
func (_Invalidstart *InvalidstartTransactor) CreateContractWithInvalidCode(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invalidstart.contract.Transact(opts, "createContractWithInvalidCode")
}

// CreateContractWithInvalidCode is a paid mutator transaction binding the contract method 0xa2990d42.
//
// Solidity: function createContractWithInvalidCode() returns()
func (_Invalidstart *InvalidstartSession) CreateContractWithInvalidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.CreateContractWithInvalidCode(&_Invalidstart.TransactOpts)
}

// CreateContractWithInvalidCode is a paid mutator transaction binding the contract method 0xa2990d42.
//
// Solidity: function createContractWithInvalidCode() returns()
func (_Invalidstart *InvalidstartTransactorSession) CreateContractWithInvalidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.CreateContractWithInvalidCode(&_Invalidstart.TransactOpts)
}

// CreateContractWithValidCode is a paid mutator transaction binding the contract method 0x02fc6a88.
//
// Solidity: function createContractWithValidCode() returns()
func (_Invalidstart *InvalidstartTransactor) CreateContractWithValidCode(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invalidstart.contract.Transact(opts, "createContractWithValidCode")
}

// CreateContractWithValidCode is a paid mutator transaction binding the contract method 0x02fc6a88.
//
// Solidity: function createContractWithValidCode() returns()
func (_Invalidstart *InvalidstartSession) CreateContractWithValidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.CreateContractWithValidCode(&_Invalidstart.TransactOpts)
}

// CreateContractWithValidCode is a paid mutator transaction binding the contract method 0x02fc6a88.
//
// Solidity: function createContractWithValidCode() returns()
func (_Invalidstart *InvalidstartTransactorSession) CreateContractWithValidCode() (*types.Transaction, error) {
	return _Invalidstart.Contract.CreateContractWithValidCode(&_Invalidstart.TransactOpts)
}
