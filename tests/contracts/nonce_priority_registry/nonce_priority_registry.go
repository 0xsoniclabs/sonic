// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package nonce_priority_registry

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

// NoncePriorityRegistryPriority is an auto generated low-level Go binding around an user-defined struct.
type NoncePriorityRegistryPriority struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}

// NoncePriorityRegistryMetaData contains all meta data concerning the NoncePriorityRegistry contract.
var NoncePriorityRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"getPriority\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getPriorityConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"maxGasPerEntityPerBlock\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maxPiggybackTxsPerEntityPerEvent\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"perBlockGas\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"perEventTxs\",\"type\":\"uint256\"}],\"name\":\"setConfig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"firstNonce\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint64\",\"name\":\"level\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"weight\",\"type\":\"uint64\"},{\"internalType\":\"uint128\",\"name\":\"id\",\"type\":\"uint128\"}],\"internalType\":\"structNoncePriorityRegistry.Priority[]\",\"name\":\"window\",\"type\":\"tuple[]\"}],\"name\":\"setPriorities\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506108b88061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80631e34c5851461004e578063928461bd1461006a578063c4dd168d14610089578063d9dceeb8146100a5575b5f5ffd5b6100686004803603810190610063919061028a565b6100d7565b005b6100726100e9565b6040516100809291906102d7565b60405180910390f35b6100a3600480360381019061009e91906103b9565b6100f9565b005b6100bf60048036038101906100ba919061047f565b61019b565b6040516100ce93929190610575565b60405180910390f35b81600181905550806002819055505050565b5f5f600154915060025490509091565b5f5f90505b828290508110156101945782828281811061011c5761011b6105aa565b5b9050606002015f5f8773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f838761016b9190610604565b81526020019081526020015f2081816101849190610874565b90505080806001019150506100fe565b5050505050565b5f5f5f5f5f5f8c73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8981526020019081526020015f209050805f015f9054906101000a900467ffffffffffffffff16815f0160089054906101000a900467ffffffffffffffff16825f0160109054906101000a90046fffffffffffffffffffffffffffffffff16935093509350509750975097945050505050565b5f5ffd5b5f5ffd5b5f819050919050565b61026981610257565b8114610273575f5ffd5b50565b5f8135905061028481610260565b92915050565b5f5f604083850312156102a05761029f61024f565b5b5f6102ad85828601610276565b92505060206102be85828601610276565b9150509250929050565b6102d181610257565b82525050565b5f6040820190506102ea5f8301856102c8565b6102f760208301846102c8565b9392505050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610327826102fe565b9050919050565b6103378161031d565b8114610341575f5ffd5b50565b5f813590506103528161032e565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261037957610378610358565b5b8235905067ffffffffffffffff8111156103965761039561035c565b5b6020830191508360608202830111156103b2576103b1610360565b5b9250929050565b5f5f5f5f606085870312156103d1576103d061024f565b5b5f6103de87828801610344565b94505060206103ef87828801610276565b935050604085013567ffffffffffffffff8111156104105761040f610253565b5b61041c87828801610364565b925092505092959194509250565b5f5f83601f84011261043f5761043e610358565b5b8235905067ffffffffffffffff81111561045c5761045b61035c565b5b60208301915083600182028301111561047857610477610360565b5b9250929050565b5f5f5f5f5f5f5f60c0888a03121561049a5761049961024f565b5b5f6104a78a828b01610344565b97505060206104b88a828b01610344565b96505060406104c98a828b01610276565b95505060606104da8a828b01610276565b945050608088013567ffffffffffffffff8111156104fb576104fa610253565b5b6105078a828b0161042a565b935093505060a061051a8a828b01610276565b91505092959891949750929550565b5f67ffffffffffffffff82169050919050565b61054581610529565b82525050565b5f6fffffffffffffffffffffffffffffffff82169050919050565b61056f8161054b565b82525050565b5f6060820190506105885f83018661053c565b610595602083018561053c565b6105a26040830184610566565b949350505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61060e82610257565b915061061983610257565b9250828201905080821115610631576106306105d7565b5b92915050565b61064081610529565b811461064a575f5ffd5b50565b5f813561065981610637565b80915050919050565b5f815f1b9050919050565b5f67ffffffffffffffff61068084610662565b9350801983169250808416831791505092915050565b5f819050919050565b5f6106b96106b46106af84610529565b610696565b610529565b9050919050565b5f819050919050565b6106d28261069f565b6106e56106de826106c0565b835461066d565b8255505050565b5f8160401b9050919050565b5f6fffffffffffffffff0000000000000000610713846106ec565b9350801983169250808416831791505092915050565b6107328261069f565b61074561073e826106c0565b83546106f8565b8255505050565b6107558161054b565b811461075f575f5ffd5b50565b5f813561076e8161074c565b80915050919050565b5f8160801b9050919050565b5f7fffffffffffffffffffffffffffffffff000000000000000000000000000000006107ae84610777565b9350801983169250808416831791505092915050565b5f6107de6107d96107d48461054b565b610696565b61054b565b9050919050565b5f819050919050565b6107f7826107c4565b61080a610803826107e5565b8354610783565b8255505050565b5f81015f8301806108218161064d565b905061082d81846106c9565b5050505f810160208301806108418161064d565b905061084d8184610729565b5050505f8101604083018061086181610762565b905061086d81846107ee565b5050505050565b61087e8282610811565b505056fea2646970667358221220c6875fbccfb8485bfdb2a10f204f74f1db16127da5ca1ead071b188331fd21f464736f6c634300081e0033",
}

// NoncePriorityRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use NoncePriorityRegistryMetaData.ABI instead.
var NoncePriorityRegistryABI = NoncePriorityRegistryMetaData.ABI

// NoncePriorityRegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use NoncePriorityRegistryMetaData.Bin instead.
var NoncePriorityRegistryBin = NoncePriorityRegistryMetaData.Bin

// DeployNoncePriorityRegistry deploys a new Ethereum contract, binding an instance of NoncePriorityRegistry to it.
func DeployNoncePriorityRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *NoncePriorityRegistry, error) {
	parsed, err := NoncePriorityRegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(NoncePriorityRegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NoncePriorityRegistry{NoncePriorityRegistryCaller: NoncePriorityRegistryCaller{contract: contract}, NoncePriorityRegistryTransactor: NoncePriorityRegistryTransactor{contract: contract}, NoncePriorityRegistryFilterer: NoncePriorityRegistryFilterer{contract: contract}}, nil
}

// NoncePriorityRegistry is an auto generated Go binding around an Ethereum contract.
type NoncePriorityRegistry struct {
	NoncePriorityRegistryCaller     // Read-only binding to the contract
	NoncePriorityRegistryTransactor // Write-only binding to the contract
	NoncePriorityRegistryFilterer   // Log filterer for contract events
}

// NoncePriorityRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type NoncePriorityRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NoncePriorityRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NoncePriorityRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NoncePriorityRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NoncePriorityRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NoncePriorityRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NoncePriorityRegistrySession struct {
	Contract     *NoncePriorityRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// NoncePriorityRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NoncePriorityRegistryCallerSession struct {
	Contract *NoncePriorityRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// NoncePriorityRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NoncePriorityRegistryTransactorSession struct {
	Contract     *NoncePriorityRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// NoncePriorityRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type NoncePriorityRegistryRaw struct {
	Contract *NoncePriorityRegistry // Generic contract binding to access the raw methods on
}

// NoncePriorityRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NoncePriorityRegistryCallerRaw struct {
	Contract *NoncePriorityRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// NoncePriorityRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NoncePriorityRegistryTransactorRaw struct {
	Contract *NoncePriorityRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNoncePriorityRegistry creates a new instance of NoncePriorityRegistry, bound to a specific deployed contract.
func NewNoncePriorityRegistry(address common.Address, backend bind.ContractBackend) (*NoncePriorityRegistry, error) {
	contract, err := bindNoncePriorityRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NoncePriorityRegistry{NoncePriorityRegistryCaller: NoncePriorityRegistryCaller{contract: contract}, NoncePriorityRegistryTransactor: NoncePriorityRegistryTransactor{contract: contract}, NoncePriorityRegistryFilterer: NoncePriorityRegistryFilterer{contract: contract}}, nil
}

// NewNoncePriorityRegistryCaller creates a new read-only instance of NoncePriorityRegistry, bound to a specific deployed contract.
func NewNoncePriorityRegistryCaller(address common.Address, caller bind.ContractCaller) (*NoncePriorityRegistryCaller, error) {
	contract, err := bindNoncePriorityRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NoncePriorityRegistryCaller{contract: contract}, nil
}

// NewNoncePriorityRegistryTransactor creates a new write-only instance of NoncePriorityRegistry, bound to a specific deployed contract.
func NewNoncePriorityRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*NoncePriorityRegistryTransactor, error) {
	contract, err := bindNoncePriorityRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NoncePriorityRegistryTransactor{contract: contract}, nil
}

// NewNoncePriorityRegistryFilterer creates a new log filterer instance of NoncePriorityRegistry, bound to a specific deployed contract.
func NewNoncePriorityRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*NoncePriorityRegistryFilterer, error) {
	contract, err := bindNoncePriorityRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NoncePriorityRegistryFilterer{contract: contract}, nil
}

// bindNoncePriorityRegistry binds a generic wrapper to an already deployed contract.
func bindNoncePriorityRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NoncePriorityRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NoncePriorityRegistry *NoncePriorityRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NoncePriorityRegistry.Contract.NoncePriorityRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NoncePriorityRegistry *NoncePriorityRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.NoncePriorityRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NoncePriorityRegistry *NoncePriorityRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.NoncePriorityRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NoncePriorityRegistry *NoncePriorityRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NoncePriorityRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NoncePriorityRegistry *NoncePriorityRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NoncePriorityRegistry *NoncePriorityRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.contract.Transact(opts, method, params...)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address from, address , uint256 , uint256 nonce, bytes , uint256 ) view returns(uint64 level, uint64 weight, uint128 id)
func (_NoncePriorityRegistry *NoncePriorityRegistryCaller) GetPriority(opts *bind.CallOpts, from common.Address, arg1 common.Address, arg2 *big.Int, nonce *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	var out []interface{}
	err := _NoncePriorityRegistry.contract.Call(opts, &out, "getPriority", from, arg1, arg2, nonce, arg4, arg5)

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
// Solidity: function getPriority(address from, address , uint256 , uint256 nonce, bytes , uint256 ) view returns(uint64 level, uint64 weight, uint128 id)
func (_NoncePriorityRegistry *NoncePriorityRegistrySession) GetPriority(from common.Address, arg1 common.Address, arg2 *big.Int, nonce *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _NoncePriorityRegistry.Contract.GetPriority(&_NoncePriorityRegistry.CallOpts, from, arg1, arg2, nonce, arg4, arg5)
}

// GetPriority is a free data retrieval call binding the contract method 0xd9dceeb8.
//
// Solidity: function getPriority(address from, address , uint256 , uint256 nonce, bytes , uint256 ) view returns(uint64 level, uint64 weight, uint128 id)
func (_NoncePriorityRegistry *NoncePriorityRegistryCallerSession) GetPriority(from common.Address, arg1 common.Address, arg2 *big.Int, nonce *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Level  uint64
	Weight uint64
	Id     *big.Int
}, error) {
	return _NoncePriorityRegistry.Contract.GetPriority(&_NoncePriorityRegistry.CallOpts, from, arg1, arg2, nonce, arg4, arg5)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() view returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_NoncePriorityRegistry *NoncePriorityRegistryCaller) GetPriorityConfig(opts *bind.CallOpts) (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	var out []interface{}
	err := _NoncePriorityRegistry.contract.Call(opts, &out, "getPriorityConfig")

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
func (_NoncePriorityRegistry *NoncePriorityRegistrySession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _NoncePriorityRegistry.Contract.GetPriorityConfig(&_NoncePriorityRegistry.CallOpts)
}

// GetPriorityConfig is a free data retrieval call binding the contract method 0x928461bd.
//
// Solidity: function getPriorityConfig() view returns(uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent)
func (_NoncePriorityRegistry *NoncePriorityRegistryCallerSession) GetPriorityConfig() (struct {
	MaxGasPerEntityPerBlock          *big.Int
	MaxPiggybackTxsPerEntityPerEvent *big.Int
}, error) {
	return _NoncePriorityRegistry.Contract.GetPriorityConfig(&_NoncePriorityRegistry.CallOpts)
}

// SetConfig is a paid mutator transaction binding the contract method 0x1e34c585.
//
// Solidity: function setConfig(uint256 perBlockGas, uint256 perEventTxs) returns()
func (_NoncePriorityRegistry *NoncePriorityRegistryTransactor) SetConfig(opts *bind.TransactOpts, perBlockGas *big.Int, perEventTxs *big.Int) (*types.Transaction, error) {
	return _NoncePriorityRegistry.contract.Transact(opts, "setConfig", perBlockGas, perEventTxs)
}

// SetConfig is a paid mutator transaction binding the contract method 0x1e34c585.
//
// Solidity: function setConfig(uint256 perBlockGas, uint256 perEventTxs) returns()
func (_NoncePriorityRegistry *NoncePriorityRegistrySession) SetConfig(perBlockGas *big.Int, perEventTxs *big.Int) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.SetConfig(&_NoncePriorityRegistry.TransactOpts, perBlockGas, perEventTxs)
}

// SetConfig is a paid mutator transaction binding the contract method 0x1e34c585.
//
// Solidity: function setConfig(uint256 perBlockGas, uint256 perEventTxs) returns()
func (_NoncePriorityRegistry *NoncePriorityRegistryTransactorSession) SetConfig(perBlockGas *big.Int, perEventTxs *big.Int) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.SetConfig(&_NoncePriorityRegistry.TransactOpts, perBlockGas, perEventTxs)
}

// SetPriorities is a paid mutator transaction binding the contract method 0xc4dd168d.
//
// Solidity: function setPriorities(address from, uint256 firstNonce, (uint64,uint64,uint128)[] window) returns()
func (_NoncePriorityRegistry *NoncePriorityRegistryTransactor) SetPriorities(opts *bind.TransactOpts, from common.Address, firstNonce *big.Int, window []NoncePriorityRegistryPriority) (*types.Transaction, error) {
	return _NoncePriorityRegistry.contract.Transact(opts, "setPriorities", from, firstNonce, window)
}

// SetPriorities is a paid mutator transaction binding the contract method 0xc4dd168d.
//
// Solidity: function setPriorities(address from, uint256 firstNonce, (uint64,uint64,uint128)[] window) returns()
func (_NoncePriorityRegistry *NoncePriorityRegistrySession) SetPriorities(from common.Address, firstNonce *big.Int, window []NoncePriorityRegistryPriority) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.SetPriorities(&_NoncePriorityRegistry.TransactOpts, from, firstNonce, window)
}

// SetPriorities is a paid mutator transaction binding the contract method 0xc4dd168d.
//
// Solidity: function setPriorities(address from, uint256 firstNonce, (uint64,uint64,uint128)[] window) returns()
func (_NoncePriorityRegistry *NoncePriorityRegistryTransactorSession) SetPriorities(from common.Address, firstNonce *big.Int, window []NoncePriorityRegistryPriority) (*types.Transaction, error) {
	return _NoncePriorityRegistry.Contract.SetPriorities(&_NoncePriorityRegistry.TransactOpts, from, firstNonce, window)
}
