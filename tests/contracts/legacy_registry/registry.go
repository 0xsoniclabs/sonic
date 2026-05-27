// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package legacy_registry

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

// LegacyRegistryMetaData contains all meta data concerning the LegacyRegistry contract.
var LegacyRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadCharge\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f80fd5b5061070e8061001d5f395ff3fe608060405260043610610049575f3560e01c8063399f59ca1461004d5780634b5c54c0146100895780639ec88e99146100b5578063b9ed9f26146100d1578063fecb2bc3146100f9575b5f80fd5b348015610058575f80fd5b50610073600480360381019061006e9190610402565b610135565b60405161008091906104c4565b60405180910390f35b348015610094575f80fd5b5061009d610171565b6040516100ac939291906104ec565b60405180910390f35b6100cf60048036038101906100ca919061054b565b61019f565b005b3480156100dc575f80fd5b506100f760048036038101906100f29190610576565b6101ca565b005b348015610104575f80fd5b5061011f600480360381019061011a919061054b565b6102f3565b60405161012c91906105b4565b60405180910390f35b5f8060015f1b9050825f808381526020019081526020015f205f01541061015f5780915050610166565b5f801b9150505b979650505050505050565b5f805f620186a0925061ea60915061c350828461018e91906105fa565b61019891906105fa565b9050909192565b345f808381526020019081526020015f205f015f8282546101c091906105fa565b9250508190555050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610201575f80fd5b805f808481526020019081526020015f205f01541015610256576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161024d90610687565b60405180910390fd5b73fc00face0000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663850a10c0826040518263ffffffff1660e01b81526004015f604051808303818588803b1580156102b0575f80fd5b505af11580156102c2573d5f803e3d5ffd5b5050505050805f808481526020019081526020015f205f015f8282546102e891906106a5565b925050819055505050565b5f602052805f5260405f205f91509050805f0154905081565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61033d82610314565b9050919050565b61034d81610333565b8114610357575f80fd5b50565b5f8135905061036881610344565b92915050565b5f819050919050565b6103808161036e565b811461038a575f80fd5b50565b5f8135905061039b81610377565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f8401126103c2576103c16103a1565b5b8235905067ffffffffffffffff8111156103df576103de6103a5565b5b6020830191508360018202830111156103fb576103fa6103a9565b5b9250929050565b5f805f805f805f60c0888a03121561041d5761041c61030c565b5b5f61042a8a828b0161035a565b975050602061043b8a828b0161035a565b965050604061044c8a828b0161038d565b955050606061045d8a828b0161038d565b945050608088013567ffffffffffffffff81111561047e5761047d610310565b5b61048a8a828b016103ad565b935093505060a061049d8a828b0161038d565b91505092959891949750929550565b5f819050919050565b6104be816104ac565b82525050565b5f6020820190506104d75f8301846104b5565b92915050565b6104e68161036e565b82525050565b5f6060820190506104ff5f8301866104dd565b61050c60208301856104dd565b61051960408301846104dd565b949350505050565b61052a816104ac565b8114610534575f80fd5b50565b5f8135905061054581610521565b92915050565b5f602082840312156105605761055f61030c565b5b5f61056d84828501610537565b91505092915050565b5f806040838503121561058c5761058b61030c565b5b5f61059985828601610537565b92505060206105aa8582860161038d565b9150509250929050565b5f6020820190506105c75f8301846104dd565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6106048261036e565b915061060f8361036e565b9250828201905080821115610627576106266105cd565b5b92915050565b5f82825260208201905092915050565b7f4e6f7420656e6f7567682066756e6473000000000000000000000000000000005f82015250565b5f61067160108361062d565b915061067c8261063d565b602082019050919050565b5f6020820190508181035f83015261069e81610665565b9050919050565b5f6106af8261036e565b91506106ba8361036e565b92508282039050818111156106d2576106d16105cd565b5b9291505056fea2646970667358221220d325c38a742f71ebfe98fe1958b8f3cb7e85319a783dc99e5259a5c545cbae7164736f6c63430008180033",
}

// LegacyRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use LegacyRegistryMetaData.ABI instead.
var LegacyRegistryABI = LegacyRegistryMetaData.ABI

// LegacyRegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LegacyRegistryMetaData.Bin instead.
var LegacyRegistryBin = LegacyRegistryMetaData.Bin

// DeployLegacyRegistry deploys a new Ethereum contract, binding an instance of LegacyRegistry to it.
func DeployLegacyRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LegacyRegistry, error) {
	parsed, err := LegacyRegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LegacyRegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LegacyRegistry{LegacyRegistryCaller: LegacyRegistryCaller{contract: contract}, LegacyRegistryTransactor: LegacyRegistryTransactor{contract: contract}, LegacyRegistryFilterer: LegacyRegistryFilterer{contract: contract}}, nil
}

// LegacyRegistry is an auto generated Go binding around an Ethereum contract.
type LegacyRegistry struct {
	LegacyRegistryCaller     // Read-only binding to the contract
	LegacyRegistryTransactor // Write-only binding to the contract
	LegacyRegistryFilterer   // Log filterer for contract events
}

// LegacyRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type LegacyRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LegacyRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LegacyRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LegacyRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LegacyRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LegacyRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LegacyRegistrySession struct {
	Contract     *LegacyRegistry   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LegacyRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LegacyRegistryCallerSession struct {
	Contract *LegacyRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// LegacyRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LegacyRegistryTransactorSession struct {
	Contract     *LegacyRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// LegacyRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type LegacyRegistryRaw struct {
	Contract *LegacyRegistry // Generic contract binding to access the raw methods on
}

// LegacyRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LegacyRegistryCallerRaw struct {
	Contract *LegacyRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// LegacyRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LegacyRegistryTransactorRaw struct {
	Contract *LegacyRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLegacyRegistry creates a new instance of LegacyRegistry, bound to a specific deployed contract.
func NewLegacyRegistry(address common.Address, backend bind.ContractBackend) (*LegacyRegistry, error) {
	contract, err := bindLegacyRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LegacyRegistry{LegacyRegistryCaller: LegacyRegistryCaller{contract: contract}, LegacyRegistryTransactor: LegacyRegistryTransactor{contract: contract}, LegacyRegistryFilterer: LegacyRegistryFilterer{contract: contract}}, nil
}

// NewLegacyRegistryCaller creates a new read-only instance of LegacyRegistry, bound to a specific deployed contract.
func NewLegacyRegistryCaller(address common.Address, caller bind.ContractCaller) (*LegacyRegistryCaller, error) {
	contract, err := bindLegacyRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LegacyRegistryCaller{contract: contract}, nil
}

// NewLegacyRegistryTransactor creates a new write-only instance of LegacyRegistry, bound to a specific deployed contract.
func NewLegacyRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*LegacyRegistryTransactor, error) {
	contract, err := bindLegacyRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LegacyRegistryTransactor{contract: contract}, nil
}

// NewLegacyRegistryFilterer creates a new log filterer instance of LegacyRegistry, bound to a specific deployed contract.
func NewLegacyRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*LegacyRegistryFilterer, error) {
	contract, err := bindLegacyRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LegacyRegistryFilterer{contract: contract}, nil
}

// bindLegacyRegistry binds a generic wrapper to an already deployed contract.
func bindLegacyRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LegacyRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LegacyRegistry *LegacyRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LegacyRegistry.Contract.LegacyRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LegacyRegistry *LegacyRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.LegacyRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LegacyRegistry *LegacyRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.LegacyRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LegacyRegistry *LegacyRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LegacyRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LegacyRegistry *LegacyRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LegacyRegistry *LegacyRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.contract.Transact(opts, method, params...)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(bytes32 fundId)
func (_LegacyRegistry *LegacyRegistryCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _LegacyRegistry.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, fee)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(bytes32 fundId)
func (_LegacyRegistry *LegacyRegistrySession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) ([32]byte, error) {
	return _LegacyRegistry.Contract.ChooseFund(&_LegacyRegistry.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(bytes32 fundId)
func (_LegacyRegistry *LegacyRegistryCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) ([32]byte, error) {
	return _LegacyRegistry.Contract.ChooseFund(&_LegacyRegistry.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_LegacyRegistry *LegacyRegistryCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	var out []interface{}
	err := _LegacyRegistry.contract.Call(opts, &out, "getGasConfig")

	outstruct := new(struct {
		ChooseFundLimit *big.Int
		DeductFeesLimit *big.Int
		OverheadCharge  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ChooseFundLimit = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.DeductFeesLimit = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.OverheadCharge = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_LegacyRegistry *LegacyRegistrySession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	return _LegacyRegistry.Contract.GetGasConfig(&_LegacyRegistry.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_LegacyRegistry *LegacyRegistryCallerSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	return _LegacyRegistry.Contract.GetGasConfig(&_LegacyRegistry.CallOpts)
}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 ) view returns(uint256 funds)
func (_LegacyRegistry *LegacyRegistryCaller) Sponsorships(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _LegacyRegistry.contract.Call(opts, &out, "sponsorships", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 ) view returns(uint256 funds)
func (_LegacyRegistry *LegacyRegistrySession) Sponsorships(arg0 [32]byte) (*big.Int, error) {
	return _LegacyRegistry.Contract.Sponsorships(&_LegacyRegistry.CallOpts, arg0)
}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 ) view returns(uint256 funds)
func (_LegacyRegistry *LegacyRegistryCallerSession) Sponsorships(arg0 [32]byte) (*big.Int, error) {
	return _LegacyRegistry.Contract.Sponsorships(&_LegacyRegistry.CallOpts, arg0)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_LegacyRegistry *LegacyRegistryTransactor) DeductFees(opts *bind.TransactOpts, fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _LegacyRegistry.contract.Transact(opts, "deductFees", fundId, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_LegacyRegistry *LegacyRegistrySession) DeductFees(fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.DeductFees(&_LegacyRegistry.TransactOpts, fundId, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_LegacyRegistry *LegacyRegistryTransactorSession) DeductFees(fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.DeductFees(&_LegacyRegistry.TransactOpts, fundId, fee)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_LegacyRegistry *LegacyRegistryTransactor) Sponsor(opts *bind.TransactOpts, fundId [32]byte) (*types.Transaction, error) {
	return _LegacyRegistry.contract.Transact(opts, "sponsor", fundId)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_LegacyRegistry *LegacyRegistrySession) Sponsor(fundId [32]byte) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.Sponsor(&_LegacyRegistry.TransactOpts, fundId)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_LegacyRegistry *LegacyRegistryTransactorSession) Sponsor(fundId [32]byte) (*types.Transaction, error) {
	return _LegacyRegistry.Contract.Sponsor(&_LegacyRegistry.TransactOpts, fundId)
}
