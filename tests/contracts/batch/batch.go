// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package batch

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

// BatchCallDelegationCall is an auto generated low-level Go binding around an user-defined struct.
type BatchCallDelegationCall struct {
	To    common.Address
	Value *big.Int
}

// BatchMetaData contains all meta data concerning the Batch contract.
var BatchMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"addresspayable\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"internalType\":\"structBatchCallDelegation.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"execute\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103868061001c5f395ff3fe60806040526004361061001d575f3560e01c806313426fdf14610021575b5f5ffd5b61003b6004803603810190610036919061019a565b61003d565b005b5f5f90505b8282905081101561012c575f838383818110610061576100606101e5565b5b9050604002015f016020810190610078919061026c565b73ffffffffffffffffffffffffffffffffffffffff163460405161009b906102c4565b5f6040518083038185875af1925050503d805f81146100d5576040519150601f19603f3d011682016040523d82523d5f602084013e6100da565b606091505b505090508061011e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161011590610332565b60405180910390fd5b508080600101915050610042565b505050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261015a57610159610139565b5b8235905067ffffffffffffffff8111156101775761017661013d565b5b60208301915083604082028301111561019357610192610141565b5b9250929050565b5f5f602083850312156101b0576101af610131565b5b5f83013567ffffffffffffffff8111156101cd576101cc610135565b5b6101d985828601610145565b92509250509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61023b82610212565b9050919050565b61024b81610231565b8114610255575f5ffd5b50565b5f8135905061026681610242565b92915050565b5f6020828403121561028157610280610131565b5b5f61028e84828501610258565b91505092915050565b5f81905092915050565b50565b5f6102af5f83610297565b91506102ba826102a1565b5f82019050919050565b5f6102ce826102a4565b9150819050919050565b5f82825260208201905092915050565b7f63616c6c207265766572746564000000000000000000000000000000000000005f82015250565b5f61031c600d836102d8565b9150610327826102e8565b602082019050919050565b5f6020820190508181035f83015261034981610310565b905091905056fea2646970667358221220b34ceede61316055f20247122cde1f139e873a4006e59db67a2b26f0ad17de8864736f6c634300081c0033",
}

// BatchABI is the input ABI used to generate the binding from.
// Deprecated: Use BatchMetaData.ABI instead.
var BatchABI = BatchMetaData.ABI

// BatchBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BatchMetaData.Bin instead.
var BatchBin = BatchMetaData.Bin

// DeployBatch deploys a new Ethereum contract, binding an instance of Batch to it.
func DeployBatch(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Batch, error) {
	parsed, err := BatchMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BatchBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Batch{BatchCaller: BatchCaller{contract: contract}, BatchTransactor: BatchTransactor{contract: contract}, BatchFilterer: BatchFilterer{contract: contract}}, nil
}

// Batch is an auto generated Go binding around an Ethereum contract.
type Batch struct {
	BatchCaller     // Read-only binding to the contract
	BatchTransactor // Write-only binding to the contract
	BatchFilterer   // Log filterer for contract events
}

// BatchCaller is an auto generated read-only Go binding around an Ethereum contract.
type BatchCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BatchTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BatchFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BatchSession struct {
	Contract     *Batch            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BatchCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BatchCallerSession struct {
	Contract *BatchCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BatchTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BatchTransactorSession struct {
	Contract     *BatchTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BatchRaw is an auto generated low-level Go binding around an Ethereum contract.
type BatchRaw struct {
	Contract *Batch // Generic contract binding to access the raw methods on
}

// BatchCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BatchCallerRaw struct {
	Contract *BatchCaller // Generic read-only contract binding to access the raw methods on
}

// BatchTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BatchTransactorRaw struct {
	Contract *BatchTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBatch creates a new instance of Batch, bound to a specific deployed contract.
func NewBatch(address common.Address, backend bind.ContractBackend) (*Batch, error) {
	contract, err := bindBatch(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Batch{BatchCaller: BatchCaller{contract: contract}, BatchTransactor: BatchTransactor{contract: contract}, BatchFilterer: BatchFilterer{contract: contract}}, nil
}

// NewBatchCaller creates a new read-only instance of Batch, bound to a specific deployed contract.
func NewBatchCaller(address common.Address, caller bind.ContractCaller) (*BatchCaller, error) {
	contract, err := bindBatch(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BatchCaller{contract: contract}, nil
}

// NewBatchTransactor creates a new write-only instance of Batch, bound to a specific deployed contract.
func NewBatchTransactor(address common.Address, transactor bind.ContractTransactor) (*BatchTransactor, error) {
	contract, err := bindBatch(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BatchTransactor{contract: contract}, nil
}

// NewBatchFilterer creates a new log filterer instance of Batch, bound to a specific deployed contract.
func NewBatchFilterer(address common.Address, filterer bind.ContractFilterer) (*BatchFilterer, error) {
	contract, err := bindBatch(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BatchFilterer{contract: contract}, nil
}

// bindBatch binds a generic wrapper to an already deployed contract.
func bindBatch(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BatchMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Batch *BatchRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Batch.Contract.BatchCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Batch *BatchRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Batch.Contract.BatchTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Batch *BatchRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Batch.Contract.BatchTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Batch *BatchCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Batch.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Batch *BatchTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Batch.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Batch *BatchTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Batch.Contract.contract.Transact(opts, method, params...)
}

// Execute is a paid mutator transaction binding the contract method 0x13426fdf.
//
// Solidity: function execute((address,uint256)[] calls) payable returns()
func (_Batch *BatchTransactor) Execute(opts *bind.TransactOpts, calls []BatchCallDelegationCall) (*types.Transaction, error) {
	return _Batch.contract.Transact(opts, "execute", calls)
}

// Execute is a paid mutator transaction binding the contract method 0x13426fdf.
//
// Solidity: function execute((address,uint256)[] calls) payable returns()
func (_Batch *BatchSession) Execute(calls []BatchCallDelegationCall) (*types.Transaction, error) {
	return _Batch.Contract.Execute(&_Batch.TransactOpts, calls)
}

// Execute is a paid mutator transaction binding the contract method 0x13426fdf.
//
// Solidity: function execute((address,uint256)[] calls) payable returns()
func (_Batch *BatchTransactorSession) Execute(calls []BatchCallDelegationCall) (*types.Transaction, error) {
	return _Batch.Contract.Execute(&_Batch.TransactOpts, calls)
}
