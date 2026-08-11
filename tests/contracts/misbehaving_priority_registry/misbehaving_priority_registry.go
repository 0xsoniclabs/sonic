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
	ABI: "[{\"inputs\":[],\"name\":\"MAGIC_VALUE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"mode\",\"outputs\":[{\"internalType\":\"enumMisbehavingPriorityRegistry.Mode\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumMisbehavingPriorityRegistry.Mode\",\"name\":\"newMode\",\"type\":\"uint8\"}],\"name\":\"setMode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506107a08061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610055575f3560e01c8063206731961461005957806321175b4a14610077578063295a521214610093578063928461bd146100b1578063d9dceeb8146100d0575b5f5ffd5b610061610102565b60405161006e919061034c565b60405180910390f35b610091600480360381019061008c9190610390565b61010a565b005b61009b610135565b6040516100a8919061042e565b60405180910390f35b6100b9610146565b6040516100c7929190610447565b60405180910390f35b6100ea60048036038101906100e59190610553565b610158565b6040516100f993929190610649565b60405180910390f35b63075bcd1581565b805f5f6101000a81548160ff0219169083600481111561012d5761012c6103bb565b5b021790555050565b5f5f9054906101000a900460ff1681565b5f5f633b9aca006103e8915091509091565b5f5f5f5f5f5f9054906101000a900460ff16905060048081111561017f5761017e6103bb565b5b816004811115610192576101916103bb565b5b036101d2576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101c9906106d8565b60405180910390fd5b600160048111156101e6576101e56103bb565b5b8160048111156101f9576101f86103bb565b5b03610262575f5f5f90505b61271081101561024657818160405160200161022192919061073f565b6040516020818303038152906040528051906020012091508080600101915050610204565b505f5f1b8103610260575f5f5f94509450945050506102f9565b505b5f5f5f61026e8c610306565b92509250925060026004811115610288576102876103bb565b5b84600481111561029b5761029a6103bb565b5b036102b3575f195f525f196020525f1960405260605ff35b600360048111156102c7576102c66103bb565b5b8460048111156102da576102d96103bb565b5b036102eb57825f528160205260405ff35b828282965096509650505050505b9750975097945050505050565b5f5f5f63075bcd1584036103235760015f5f92509250925061032d565b5f5f5f9250925092505b9193909250565b5f819050919050565b61034681610334565b82525050565b5f60208201905061035f5f83018461033d565b92915050565b5f5ffd5b5f5ffd5b60058110610379575f5ffd5b50565b5f8135905061038a8161036d565b92915050565b5f602082840312156103a5576103a4610365565b5b5f6103b28482850161037c565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602160045260245ffd5b600581106103f9576103f86103bb565b5b50565b5f819050610409826103e8565b919050565b5f610418826103fc565b9050919050565b6104288161040e565b82525050565b5f6020820190506104415f83018461041f565b92915050565b5f60408201905061045a5f83018561033d565b610467602083018461033d565b9392505050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6104978261046e565b9050919050565b6104a78161048d565b81146104b1575f5ffd5b50565b5f813590506104c28161049e565b92915050565b6104d181610334565b81146104db575f5ffd5b50565b5f813590506104ec816104c8565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f840112610513576105126104f2565b5b8235905067ffffffffffffffff8111156105305761052f6104f6565b5b60208301915083600182028301111561054c5761054b6104fa565b5b9250929050565b5f5f5f5f5f5f5f60c0888a03121561056e5761056d610365565b5b5f61057b8a828b016104b4565b975050602061058c8a828b016104b4565b965050604061059d8a828b016104de565b95505060606105ae8a828b016104de565b945050608088013567ffffffffffffffff8111156105cf576105ce610369565b5b6105db8a828b016104fe565b935093505060a06105ee8a828b016104de565b91505092959891949750929550565b5f67ffffffffffffffff82169050919050565b610619816105fd565b82525050565b5f6fffffffffffffffffffffffffffffffff82169050919050565b6106438161061f565b82525050565b5f60608201905061065c5f830186610610565b6106696020830185610610565b610676604083018461063a565b949350505050565b5f82825260208201905092915050565b7f6765745072696f7269747920616c77617973206661696c7300000000000000005f82015250565b5f6106c260188361067e565b91506106cd8261068e565b602082019050919050565b5f6020820190508181035f8301526106ef816106b6565b9050919050565b5f819050919050565b5f819050919050565b610719610714826106f6565b6106ff565b82525050565b5f819050919050565b61073961073482610334565b61071f565b82525050565b5f61074a8285610708565b60208201915061075a8284610728565b602082019150819050939250505056fea26469706673582212202c00b87bde81176beb241af51a5df37a8321cf25c63c69d3284aeba9a2d8760064736f6c634300081e0033",
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

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) view returns(uint64, uint64, uint128)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCaller) GetPriority(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (uint64, uint64, *big.Int, error) {
	var out []interface{}
	err := _MisbehavingPriorityRegistry.contract.Call(opts, &out, "getPriority", arg0, arg1, value, arg3, arg4, arg5)

	if err != nil {
		return *new(uint64), *new(uint64), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)
	out1 := *abi.ConvertType(out[1], new(uint64)).(*uint64)
	out2 := *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return out0, out1, out2, err

}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) view returns(uint64, uint64, uint128)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (uint64, uint64, *big.Int, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriority(&_MisbehavingPriorityRegistry.CallOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address , address , uint256 value, uint256 , bytes , uint256 ) view returns(uint64, uint64, uint128)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerSession) GetPriority(arg0 common.Address, arg1 common.Address, value *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (uint64, uint64, *big.Int, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriority(&_MisbehavingPriorityRegistry.CallOpts, arg0, arg1, value, arg3, arg4, arg5)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCaller) GetPriorityConfig(opts *bind.CallOpts) (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	var out []interface{}
	err := _MisbehavingPriorityRegistry.contract.Call(opts, &out, "getPriorityConfig")

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
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistrySession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriorityConfig(&_MisbehavingPriorityRegistry.CallOpts)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() pure returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_MisbehavingPriorityRegistry *MisbehavingPriorityRegistryCallerSession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _MisbehavingPriorityRegistry.Contract.GetPriorityConfig(&_MisbehavingPriorityRegistry.CallOpts)
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
