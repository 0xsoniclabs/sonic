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
	Bin: "0x6080604052348015600e575f5ffd5b506104a68061001c5f395ff3fe60806040526004361061001d575f3560e01c806313426fdf14610021575b5f5ffd5b61003b600480360381019061003691906101ae565b61003d565b005b5f5f90505b82829050811015610137575f838383818110610061576100606101f9565b5b905060400201803603810190610077919061038c565b90505f815f015173ffffffffffffffffffffffffffffffffffffffff1682602001516040516100a5906103e4565b5f6040518083038185875af1925050503d805f81146100df576040519150601f19603f3d011682016040523d82523d5f602084013e6100e4565b606091505b5050905080610128576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161011f90610452565b60405180910390fd5b50508080600101915050610042565b505050565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261016e5761016d61014d565b5b8235905067ffffffffffffffff81111561018b5761018a610151565b5b6020830191508360408202830111156101a7576101a6610155565b5b9250929050565b5f5f602083850312156101c4576101c3610145565b5b5f83013567ffffffffffffffff8111156101e1576101e0610149565b5b6101ed85828601610159565b92509250509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6102708261022a565b810181811067ffffffffffffffff8211171561028f5761028e61023a565b5b80604052505050565b5f6102a161013c565b90506102ad8282610267565b919050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6102db826102b2565b9050919050565b6102eb816102d1565b81146102f5575f5ffd5b50565b5f81359050610306816102e2565b92915050565b5f819050919050565b61031e8161030c565b8114610328575f5ffd5b50565b5f8135905061033981610315565b92915050565b5f6040828403121561035457610353610226565b5b61035e6040610298565b90505f61036d848285016102f8565b5f8301525060206103808482850161032b565b60208301525092915050565b5f604082840312156103a1576103a0610145565b5b5f6103ae8482850161033f565b91505092915050565b5f81905092915050565b50565b5f6103cf5f836103b7565b91506103da826103c1565b5f82019050919050565b5f6103ee826103c4565b9150819050919050565b5f82825260208201905092915050565b7f63616c6c207265766572746564000000000000000000000000000000000000005f82015250565b5f61043c600d836103f8565b915061044782610408565b602082019050919050565b5f6020820190508181035f83015261046981610430565b905091905056fea264697066735822122027e940d6c5729c95c47d3beddf8305805a065e300adba87b7078b3cbb5e9355564736f6c634300081c0033",
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
