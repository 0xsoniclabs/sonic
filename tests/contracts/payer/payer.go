// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package payer

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

// PayerMetaData contains all meta data concerning the Payer contract.
var PayerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"forward\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"pay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"payIgnoringResult\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506105ee8061001c5f395ff3fe608060405260043610610037575f3560e01c806360f33168146100425780636fadcf721461006a578063c4076876146100925761003e565b3661003e57005b5f5ffd5b34801561004d575f5ffd5b5061006860048036038101906100639190610316565b6100ba565b005b348015610075575f5ffd5b50610090600480360381019061008b91906103f0565b610124565b005b34801561009d575f5ffd5b506100b860048036038101906100b39190610316565b6101d4565b005b8173ffffffffffffffffffffffffffffffffffffffff16816040516100de9061047a565b5f6040518083038185875af1925050503d805f8114610118576040519150601f19603f3d011682016040523d82523d5f602084013e61011d565b606091505b5050505050565b5f8373ffffffffffffffffffffffffffffffffffffffff16838360405161014c9291906104c0565b5f604051808303815f865af19150503d805f8114610185576040519150601f19603f3d011682016040523d82523d5f602084013e61018a565b606091505b50509050806101ce576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101c590610532565b60405180910390fd5b50505050565b5f8273ffffffffffffffffffffffffffffffffffffffff16826040516101f99061047a565b5f6040518083038185875af1925050503d805f8114610233576040519150601f19603f3d011682016040523d82523d5f602084013e610238565b606091505b505090508061027c576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102739061059a565b60405180910390fd5b505050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6102b282610289565b9050919050565b6102c2816102a8565b81146102cc575f5ffd5b50565b5f813590506102dd816102b9565b92915050565b5f819050919050565b6102f5816102e3565b81146102ff575f5ffd5b50565b5f81359050610310816102ec565b92915050565b5f5f6040838503121561032c5761032b610281565b5b5f610339858286016102cf565b925050602061034a85828601610302565b9150509250929050565b5f61035e82610289565b9050919050565b61036e81610354565b8114610378575f5ffd5b50565b5f8135905061038981610365565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f8401126103b0576103af61038f565b5b8235905067ffffffffffffffff8111156103cd576103cc610393565b5b6020830191508360018202830111156103e9576103e8610397565b5b9250929050565b5f5f5f6040848603121561040757610406610281565b5b5f6104148682870161037b565b935050602084013567ffffffffffffffff81111561043557610434610285565b5b6104418682870161039b565b92509250509250925092565b5f81905092915050565b50565b5f6104655f8361044d565b915061047082610457565b5f82019050919050565b5f6104848261045a565b9150819050919050565b828183375f83830152505050565b5f6104a7838561044d565b93506104b483858461048e565b82840190509392505050565b5f6104cc82848661049c565b91508190509392505050565b5f82825260208201905092915050565b7f666f7277617264206661696c65640000000000000000000000000000000000005f82015250565b5f61051c600e836104d8565b9150610527826104e8565b602082019050919050565b5f6020820190508181035f83015261054981610510565b9050919050565b7f7061796d656e74206661696c65640000000000000000000000000000000000005f82015250565b5f610584600e836104d8565b915061058f82610550565b602082019050919050565b5f6020820190508181035f8301526105b181610578565b905091905056fea26469706673582212203883dff900460d21d357ab04d5cb635e892a8de06a236eaf744e52ee55952f0364736f6c63430008260033",
}

// PayerABI is the input ABI used to generate the binding from.
// Deprecated: Use PayerMetaData.ABI instead.
var PayerABI = PayerMetaData.ABI

// PayerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PayerMetaData.Bin instead.
var PayerBin = PayerMetaData.Bin

// DeployPayer deploys a new Ethereum contract, binding an instance of Payer to it.
func DeployPayer(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Payer, error) {
	parsed, err := PayerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PayerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Payer{PayerCaller: PayerCaller{contract: contract}, PayerTransactor: PayerTransactor{contract: contract}, PayerFilterer: PayerFilterer{contract: contract}}, nil
}

// Payer is an auto generated Go binding around an Ethereum contract.
type Payer struct {
	PayerCaller     // Read-only binding to the contract
	PayerTransactor // Write-only binding to the contract
	PayerFilterer   // Log filterer for contract events
}

// PayerCaller is an auto generated read-only Go binding around an Ethereum contract.
type PayerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PayerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PayerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PayerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PayerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PayerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PayerSession struct {
	Contract     *Payer            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PayerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PayerCallerSession struct {
	Contract *PayerCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// PayerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PayerTransactorSession struct {
	Contract     *PayerTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PayerRaw is an auto generated low-level Go binding around an Ethereum contract.
type PayerRaw struct {
	Contract *Payer // Generic contract binding to access the raw methods on
}

// PayerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PayerCallerRaw struct {
	Contract *PayerCaller // Generic read-only contract binding to access the raw methods on
}

// PayerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PayerTransactorRaw struct {
	Contract *PayerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPayer creates a new instance of Payer, bound to a specific deployed contract.
func NewPayer(address common.Address, backend bind.ContractBackend) (*Payer, error) {
	contract, err := bindPayer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Payer{PayerCaller: PayerCaller{contract: contract}, PayerTransactor: PayerTransactor{contract: contract}, PayerFilterer: PayerFilterer{contract: contract}}, nil
}

// NewPayerCaller creates a new read-only instance of Payer, bound to a specific deployed contract.
func NewPayerCaller(address common.Address, caller bind.ContractCaller) (*PayerCaller, error) {
	contract, err := bindPayer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PayerCaller{contract: contract}, nil
}

// NewPayerTransactor creates a new write-only instance of Payer, bound to a specific deployed contract.
func NewPayerTransactor(address common.Address, transactor bind.ContractTransactor) (*PayerTransactor, error) {
	contract, err := bindPayer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PayerTransactor{contract: contract}, nil
}

// NewPayerFilterer creates a new log filterer instance of Payer, bound to a specific deployed contract.
func NewPayerFilterer(address common.Address, filterer bind.ContractFilterer) (*PayerFilterer, error) {
	contract, err := bindPayer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PayerFilterer{contract: contract}, nil
}

// bindPayer binds a generic wrapper to an already deployed contract.
func bindPayer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PayerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Payer *PayerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Payer.Contract.PayerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Payer *PayerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Payer.Contract.PayerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Payer *PayerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Payer.Contract.PayerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Payer *PayerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Payer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Payer *PayerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Payer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Payer *PayerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Payer.Contract.contract.Transact(opts, method, params...)
}

// Forward is a paid mutator transaction binding the contract method 0x6fadcf72.
//
// Solidity: function forward(address to, bytes data) returns()
func (_Payer *PayerTransactor) Forward(opts *bind.TransactOpts, to common.Address, data []byte) (*types.Transaction, error) {
	return _Payer.contract.Transact(opts, "forward", to, data)
}

// Forward is a paid mutator transaction binding the contract method 0x6fadcf72.
//
// Solidity: function forward(address to, bytes data) returns()
func (_Payer *PayerSession) Forward(to common.Address, data []byte) (*types.Transaction, error) {
	return _Payer.Contract.Forward(&_Payer.TransactOpts, to, data)
}

// Forward is a paid mutator transaction binding the contract method 0x6fadcf72.
//
// Solidity: function forward(address to, bytes data) returns()
func (_Payer *PayerTransactorSession) Forward(to common.Address, data []byte) (*types.Transaction, error) {
	return _Payer.Contract.Forward(&_Payer.TransactOpts, to, data)
}

// Pay is a paid mutator transaction binding the contract method 0xc4076876.
//
// Solidity: function pay(address to, uint256 amount) returns()
func (_Payer *PayerTransactor) Pay(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Payer.contract.Transact(opts, "pay", to, amount)
}

// Pay is a paid mutator transaction binding the contract method 0xc4076876.
//
// Solidity: function pay(address to, uint256 amount) returns()
func (_Payer *PayerSession) Pay(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Payer.Contract.Pay(&_Payer.TransactOpts, to, amount)
}

// Pay is a paid mutator transaction binding the contract method 0xc4076876.
//
// Solidity: function pay(address to, uint256 amount) returns()
func (_Payer *PayerTransactorSession) Pay(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Payer.Contract.Pay(&_Payer.TransactOpts, to, amount)
}

// PayIgnoringResult is a paid mutator transaction binding the contract method 0x60f33168.
//
// Solidity: function payIgnoringResult(address to, uint256 amount) returns()
func (_Payer *PayerTransactor) PayIgnoringResult(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Payer.contract.Transact(opts, "payIgnoringResult", to, amount)
}

// PayIgnoringResult is a paid mutator transaction binding the contract method 0x60f33168.
//
// Solidity: function payIgnoringResult(address to, uint256 amount) returns()
func (_Payer *PayerSession) PayIgnoringResult(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Payer.Contract.PayIgnoringResult(&_Payer.TransactOpts, to, amount)
}

// PayIgnoringResult is a paid mutator transaction binding the contract method 0x60f33168.
//
// Solidity: function payIgnoringResult(address to, uint256 amount) returns()
func (_Payer *PayerTransactorSession) PayIgnoringResult(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Payer.Contract.PayIgnoringResult(&_Payer.TransactOpts, to, amount)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Payer *PayerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Payer.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Payer *PayerSession) Receive() (*types.Transaction, error) {
	return _Payer.Contract.Receive(&_Payer.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Payer *PayerTransactorSession) Receive() (*types.Transaction, error) {
	return _Payer.Contract.Receive(&_Payer.TransactOpts)
}
