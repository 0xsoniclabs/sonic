// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bundle

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

// BundleMetaData contains all meta data concerning the Bundle contract.
var BundleMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x6080604052348015600e575f5ffd5b50603e80601a5f395ff3fe60806040525f5ffdfea2646970667358221220c40213d17a21b0ae740980829b1f67d25558e849db0dad29f7f5d9377c6e19ed64736f6c634300081e0033",
}

// BundleABI is the input ABI used to generate the binding from.
// Deprecated: Use BundleMetaData.ABI instead.
var BundleABI = BundleMetaData.ABI

// BundleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BundleMetaData.Bin instead.
var BundleBin = BundleMetaData.Bin

// DeployBundle deploys a new Ethereum contract, binding an instance of Bundle to it.
func DeployBundle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Bundle, error) {
	parsed, err := BundleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BundleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Bundle{BundleCaller: BundleCaller{contract: contract}, BundleTransactor: BundleTransactor{contract: contract}, BundleFilterer: BundleFilterer{contract: contract}}, nil
}

// Bundle is an auto generated Go binding around an Ethereum contract.
type Bundle struct {
	BundleCaller     // Read-only binding to the contract
	BundleTransactor // Write-only binding to the contract
	BundleFilterer   // Log filterer for contract events
}

// BundleCaller is an auto generated read-only Go binding around an Ethereum contract.
type BundleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BundleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BundleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BundleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BundleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BundleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BundleSession struct {
	Contract     *Bundle           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BundleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BundleCallerSession struct {
	Contract *BundleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BundleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BundleTransactorSession struct {
	Contract     *BundleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BundleRaw is an auto generated low-level Go binding around an Ethereum contract.
type BundleRaw struct {
	Contract *Bundle // Generic contract binding to access the raw methods on
}

// BundleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BundleCallerRaw struct {
	Contract *BundleCaller // Generic read-only contract binding to access the raw methods on
}

// BundleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BundleTransactorRaw struct {
	Contract *BundleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBundle creates a new instance of Bundle, bound to a specific deployed contract.
func NewBundle(address common.Address, backend bind.ContractBackend) (*Bundle, error) {
	contract, err := bindBundle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Bundle{BundleCaller: BundleCaller{contract: contract}, BundleTransactor: BundleTransactor{contract: contract}, BundleFilterer: BundleFilterer{contract: contract}}, nil
}

// NewBundleCaller creates a new read-only instance of Bundle, bound to a specific deployed contract.
func NewBundleCaller(address common.Address, caller bind.ContractCaller) (*BundleCaller, error) {
	contract, err := bindBundle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BundleCaller{contract: contract}, nil
}

// NewBundleTransactor creates a new write-only instance of Bundle, bound to a specific deployed contract.
func NewBundleTransactor(address common.Address, transactor bind.ContractTransactor) (*BundleTransactor, error) {
	contract, err := bindBundle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BundleTransactor{contract: contract}, nil
}

// NewBundleFilterer creates a new log filterer instance of Bundle, bound to a specific deployed contract.
func NewBundleFilterer(address common.Address, filterer bind.ContractFilterer) (*BundleFilterer, error) {
	contract, err := bindBundle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BundleFilterer{contract: contract}, nil
}

// bindBundle binds a generic wrapper to an already deployed contract.
func bindBundle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BundleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bundle *BundleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Bundle.Contract.BundleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bundle *BundleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bundle.Contract.BundleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bundle *BundleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bundle.Contract.BundleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bundle *BundleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Bundle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bundle *BundleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bundle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bundle *BundleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bundle.Contract.contract.Transact(opts, method, params...)
}
