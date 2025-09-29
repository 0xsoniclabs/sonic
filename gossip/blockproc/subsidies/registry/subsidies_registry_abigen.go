// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

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

// RegistryMetaData contains all meta data concerning the Registry contract.
var RegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"approvalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"bootstrapSponsorshipFund\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"callSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"contractSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610c548061001c5f395ff3fe60806040526004361061009a575f3560e01c8063779a43ac11610062578063779a43ac146101655780639ec88e9914610179578063a5dc45181461018c578063b9ed9f26146101ab578063e327d1ac146101ca578063fecb2bc3146101e9575f5ffd5b8063040cf0201461009e5780630ad1fcfc146100bf578063399f59ca146100fa57806351ee41a01461012757806363f2cdca14610146575b5f5ffd5b3480156100a9575f5ffd5b506100bd6100b836600461098a565b610230565b005b3480156100ca575f5ffd5b506100de6100d9366004610a06565b61043d565b6040805192151583526020830191909152015b60405180910390f35b348015610105575f5ffd5b50610119610114366004610a67565b61053c565b6040519081526020016100f1565b348015610132575f5ffd5b506100de610141366004610ae6565b61067f565b348015610151575f5ffd5b506100de610160366004610b08565b6106ca565b348015610170575f5ffd5b506100de6106f7565b6100bd610187366004610b08565b610730565b348015610197575f5ffd5b506100de6101a6366004610a06565b610799565b3480156101b6575f5ffd5b506100bd6101c536600461098a565b61083f565b3480156101d5575f5ffd5b506100de6101e4366004610ae6565b61095a565b3480156101f4575f5ffd5b5061021b610203366004610b08565b5f602081905290815260409020805460019091015482565b604080519283526020830191909152016100f1565b5f3a116102aa5760405162461bcd60e51b815260206004820152603c60248201527f5769746864726177616c7320617265206e6f7420737570706f7274656420746860448201527f726f7567682073706f6e736f726564207472616e73616374696f6e730000000060648201526084015b60405180910390fd5b5f8281526020818152604080832033808552600282019093529220548311156103215760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b60648201526084016102a1565b600182015482545f91906103359086610b33565b61033f9190610b50565b83549091508111156103a25760405162461bcd60e51b815260206004820152602660248201527f4e6f7420656e6f75676820617661696c61626c652066756e647320746f20776960448201526574686472617760d01b60648201526084016102a1565b6001600160a01b0382165f908152600284016020526040812080548692906103cb908490610b6f565b9250508190555083836001015f8282546103e59190610b6f565b90915550508254819084905f906103fd908490610b6f565b90915550506040516001600160a01b0383169082156108fc029083905f818181858888f19350505050158015610435573d5f5f3e3d5ffd5b505050505050565b5f806001600160a01b0385161580610456575060448314155b1561046557505f905080610533565b5f6104736004828688610b82565b61047c91610ba9565b905063095ea7b360e01b6001600160e01b03198216146104a257505f9150819050610533565b5f806104b1866004818a610b82565b8101906104be9190610be1565b9150915060018110156104da57505f9350839250610533915050565b604051606160f81b60208201526001600160601b031960608b811b821660218401528a811b8216603584015284901b166049820152600190605d0160405160208183030381529060405280519060200120945094505050505b94509492505050565b5f5f61054a8989878761043d565b9250905080801561056857505f828152602081905260409020548311155b156105735750610674565b61057f89898787610799565b9250905080801561059d57505f828152602081905260409020548311155b156105a85750610674565b6105b18961067f565b925090508080156105cf57505f828152602081905260409020548311155b156105da5750610674565b6105e38861095a565b9250905080801561060157505f828152602081905260409020548311155b1561060c5750610674565b610615866106ca565b9250905080801561063357505f828152602081905260409020548311155b1561063e5750610674565b6106466106f7565b9250905080801561066457505f828152602081905260409020548311155b1561066f5750610674565b505f90505b979650505050505050565b604051606160f81b60208201526001600160601b0319606083901b1660218201525f9081906001906035015b6040516020818303038152906040528051906020012091509150915091565b5f5f60038310156106ed57604051603160f91b60208201526001906021016106ab565b505f928392509050565b5f5f600160405160200161071290606760f81b815260010190565b60405160208183030381529060405280519060200120915091509091565b5f818152602081905260408120805490913491839190610751908490610c0b565b9091555050335f90815260028201602052604081208054349290610776908490610c0b565b9250508190555034816001015f8282546107909190610c0b565b90915550505050565b5f806001600160a01b03851615806107b15750600483105b156107c057505f905080610533565b5f6107ce6004828688610b82565b6107d791610ba9565b604051606360f81b60208201526001600160601b031960608a811b8216602184015289901b1660358201526001600160e01b031982166049820152909150600190604d0160405160208183030381529060405280519060200120925092505094509492505050565b3315610849575f5ffd5b816108965760405162461bcd60e51b815260206004820152601a60248201527f4e6f2073706f6e736f72736869702066756e642063686f73656e00000000000060448201526064016102a1565b5f82815260208190526040902080548211156108e75760405162461bcd60e51b815260206004820152601060248201526f4e6f7420656e6f7567682066756e647360801b60448201526064016102a1565b637e007d6760811b6001600160a01b031663850a10c0836040518263ffffffff1660e01b81526004015f604051808303818588803b158015610927575f5ffd5b505af1158015610939573d5f5f3e3d5ffd5b505050505081815f015f8282546109509190610b6f565b9091555050505050565b604051606360f81b60208201526001600160601b0319606083901b1660218201525f9081906001906035016106ab565b5f5f6040838503121561099b575f5ffd5b50508035926020909101359150565b6001600160a01b03811681146109be575f5ffd5b50565b5f5f83601f8401126109d1575f5ffd5b50813567ffffffffffffffff8111156109e8575f5ffd5b6020830191508360208285010111156109ff575f5ffd5b9250929050565b5f5f5f5f60608587031215610a19575f5ffd5b8435610a24816109aa565b93506020850135610a34816109aa565b9250604085013567ffffffffffffffff811115610a4f575f5ffd5b610a5b878288016109c1565b95989497509550505050565b5f5f5f5f5f5f5f60c0888a031215610a7d575f5ffd5b8735610a88816109aa565b96506020880135610a98816109aa565b95506040880135945060608801359350608088013567ffffffffffffffff811115610ac1575f5ffd5b610acd8a828b016109c1565b989b979a5095989497959660a090950135949350505050565b5f60208284031215610af6575f5ffd5b8135610b01816109aa565b9392505050565b5f60208284031215610b18575f5ffd5b5035919050565b634e487b7160e01b5f52601160045260245ffd5b8082028115828204841417610b4a57610b4a610b1f565b92915050565b5f82610b6a57634e487b7160e01b5f52601260045260245ffd5b500490565b81810381811115610b4a57610b4a610b1f565b5f5f85851115610b90575f5ffd5b83861115610b9c575f5ffd5b5050820193919092039150565b80356001600160e01b03198116906004841015610bda576001600160e01b0319600485900360031b81901b82161691505b5092915050565b5f5f60408385031215610bf2575f5ffd5b8235610bfd816109aa565b946020939093013593505050565b80820180821115610b4a57610b4a610b1f56fea26469706673582212205f83e4d137bac5f7d5f1111b7429308c6b12ffb364e43ebd6d58a45641a274ee64736f6c634300081b0033",
}

// RegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryMetaData.ABI instead.
var RegistryABI = RegistryMetaData.ABI

// RegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RegistryMetaData.Bin instead.
var RegistryBin = RegistryMetaData.Bin

// DeployRegistry deploys a new Ethereum contract, binding an instance of Registry to it.
func DeployRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Registry, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// Registry is an auto generated Go binding around an Ethereum contract.
type Registry struct {
	RegistryCaller     // Read-only binding to the contract
	RegistryTransactor // Write-only binding to the contract
	RegistryFilterer   // Log filterer for contract events
}

// RegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistrySession struct {
	Contract     *Registry         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryCallerSession struct {
	Contract *RegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryTransactorSession struct {
	Contract     *RegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryRaw struct {
	Contract *Registry // Generic contract binding to access the raw methods on
}

// RegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryCallerRaw struct {
	Contract *RegistryCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryTransactorRaw struct {
	Contract *RegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistry creates a new instance of Registry, bound to a specific deployed contract.
func NewRegistry(address common.Address, backend bind.ContractBackend) (*Registry, error) {
	contract, err := bindRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Registry{RegistryCaller: RegistryCaller{contract: contract}, RegistryTransactor: RegistryTransactor{contract: contract}, RegistryFilterer: RegistryFilterer{contract: contract}}, nil
}

// NewRegistryCaller creates a new read-only instance of Registry, bound to a specific deployed contract.
func NewRegistryCaller(address common.Address, caller bind.ContractCaller) (*RegistryCaller, error) {
	contract, err := bindRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryCaller{contract: contract}, nil
}

// NewRegistryTransactor creates a new write-only instance of Registry, bound to a specific deployed contract.
func NewRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryTransactor, error) {
	contract, err := bindRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryTransactor{contract: contract}, nil
}

// NewRegistryFilterer creates a new log filterer instance of Registry, bound to a specific deployed contract.
func NewRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryFilterer, error) {
	contract, err := bindRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryFilterer{contract: contract}, nil
}

// bindRegistry binds a generic wrapper to an already deployed contract.
func bindRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.RegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.RegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Registry *RegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Registry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Registry *RegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Registry *RegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Registry.Contract.contract.Transact(opts, method, params...)
}

// AccountSponsorshipFundId is a free data retrieval call binding the contract method 0x51ee41a0.
//
// Solidity: function accountSponsorshipFundId(address from) pure returns(bool, bytes32)
func (_Registry *RegistryCaller) AccountSponsorshipFundId(opts *bind.CallOpts, from common.Address) (bool, [32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "accountSponsorshipFundId", from)

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// AccountSponsorshipFundId is a free data retrieval call binding the contract method 0x51ee41a0.
//
// Solidity: function accountSponsorshipFundId(address from) pure returns(bool, bytes32)
func (_Registry *RegistrySession) AccountSponsorshipFundId(from common.Address) (bool, [32]byte, error) {
	return _Registry.Contract.AccountSponsorshipFundId(&_Registry.CallOpts, from)
}

// AccountSponsorshipFundId is a free data retrieval call binding the contract method 0x51ee41a0.
//
// Solidity: function accountSponsorshipFundId(address from) pure returns(bool, bytes32)
func (_Registry *RegistryCallerSession) AccountSponsorshipFundId(from common.Address) (bool, [32]byte, error) {
	return _Registry.Contract.AccountSponsorshipFundId(&_Registry.CallOpts, from)
}

// ApprovalSponsorshipFundId is a free data retrieval call binding the contract method 0x0ad1fcfc.
//
// Solidity: function approvalSponsorshipFundId(address from, address to, bytes callData) pure returns(bool, bytes32)
func (_Registry *RegistryCaller) ApprovalSponsorshipFundId(opts *bind.CallOpts, from common.Address, to common.Address, callData []byte) (bool, [32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "approvalSponsorshipFundId", from, to, callData)

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// ApprovalSponsorshipFundId is a free data retrieval call binding the contract method 0x0ad1fcfc.
//
// Solidity: function approvalSponsorshipFundId(address from, address to, bytes callData) pure returns(bool, bytes32)
func (_Registry *RegistrySession) ApprovalSponsorshipFundId(from common.Address, to common.Address, callData []byte) (bool, [32]byte, error) {
	return _Registry.Contract.ApprovalSponsorshipFundId(&_Registry.CallOpts, from, to, callData)
}

// ApprovalSponsorshipFundId is a free data retrieval call binding the contract method 0x0ad1fcfc.
//
// Solidity: function approvalSponsorshipFundId(address from, address to, bytes callData) pure returns(bool, bytes32)
func (_Registry *RegistryCallerSession) ApprovalSponsorshipFundId(from common.Address, to common.Address, callData []byte) (bool, [32]byte, error) {
	return _Registry.Contract.ApprovalSponsorshipFundId(&_Registry.CallOpts, from, to, callData)
}

// BootstrapSponsorshipFund is a free data retrieval call binding the contract method 0x63f2cdca.
//
// Solidity: function bootstrapSponsorshipFund(uint256 nonce) pure returns(bool, bytes32)
func (_Registry *RegistryCaller) BootstrapSponsorshipFund(opts *bind.CallOpts, nonce *big.Int) (bool, [32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "bootstrapSponsorshipFund", nonce)

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// BootstrapSponsorshipFund is a free data retrieval call binding the contract method 0x63f2cdca.
//
// Solidity: function bootstrapSponsorshipFund(uint256 nonce) pure returns(bool, bytes32)
func (_Registry *RegistrySession) BootstrapSponsorshipFund(nonce *big.Int) (bool, [32]byte, error) {
	return _Registry.Contract.BootstrapSponsorshipFund(&_Registry.CallOpts, nonce)
}

// BootstrapSponsorshipFund is a free data retrieval call binding the contract method 0x63f2cdca.
//
// Solidity: function bootstrapSponsorshipFund(uint256 nonce) pure returns(bool, bytes32)
func (_Registry *RegistryCallerSession) BootstrapSponsorshipFund(nonce *big.Int) (bool, [32]byte, error) {
	return _Registry.Contract.BootstrapSponsorshipFund(&_Registry.CallOpts, nonce)
}

// CallSponsorshipFundId is a free data retrieval call binding the contract method 0xa5dc4518.
//
// Solidity: function callSponsorshipFundId(address from, address to, bytes callData) pure returns(bool, bytes32)
func (_Registry *RegistryCaller) CallSponsorshipFundId(opts *bind.CallOpts, from common.Address, to common.Address, callData []byte) (bool, [32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "callSponsorshipFundId", from, to, callData)

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// CallSponsorshipFundId is a free data retrieval call binding the contract method 0xa5dc4518.
//
// Solidity: function callSponsorshipFundId(address from, address to, bytes callData) pure returns(bool, bytes32)
func (_Registry *RegistrySession) CallSponsorshipFundId(from common.Address, to common.Address, callData []byte) (bool, [32]byte, error) {
	return _Registry.Contract.CallSponsorshipFundId(&_Registry.CallOpts, from, to, callData)
}

// CallSponsorshipFundId is a free data retrieval call binding the contract method 0xa5dc4518.
//
// Solidity: function callSponsorshipFundId(address from, address to, bytes callData) pure returns(bool, bytes32)
func (_Registry *RegistryCallerSession) CallSponsorshipFundId(from common.Address, to common.Address, callData []byte) (bool, [32]byte, error) {
	return _Registry.Contract.CallSponsorshipFundId(&_Registry.CallOpts, from, to, callData)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address to, uint256 , uint256 nonce, bytes callData, uint256 fee) view returns(bytes32 fundId)
func (_Registry *RegistryCaller) ChooseFund(opts *bind.CallOpts, from common.Address, to common.Address, arg2 *big.Int, nonce *big.Int, callData []byte, fee *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "chooseFund", from, to, arg2, nonce, callData, fee)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address to, uint256 , uint256 nonce, bytes callData, uint256 fee) view returns(bytes32 fundId)
func (_Registry *RegistrySession) ChooseFund(from common.Address, to common.Address, arg2 *big.Int, nonce *big.Int, callData []byte, fee *big.Int) ([32]byte, error) {
	return _Registry.Contract.ChooseFund(&_Registry.CallOpts, from, to, arg2, nonce, callData, fee)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address to, uint256 , uint256 nonce, bytes callData, uint256 fee) view returns(bytes32 fundId)
func (_Registry *RegistryCallerSession) ChooseFund(from common.Address, to common.Address, arg2 *big.Int, nonce *big.Int, callData []byte, fee *big.Int) ([32]byte, error) {
	return _Registry.Contract.ChooseFund(&_Registry.CallOpts, from, to, arg2, nonce, callData, fee)
}

// ContractSponsorshipFundId is a free data retrieval call binding the contract method 0xe327d1ac.
//
// Solidity: function contractSponsorshipFundId(address to) pure returns(bool, bytes32)
func (_Registry *RegistryCaller) ContractSponsorshipFundId(opts *bind.CallOpts, to common.Address) (bool, [32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "contractSponsorshipFundId", to)

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// ContractSponsorshipFundId is a free data retrieval call binding the contract method 0xe327d1ac.
//
// Solidity: function contractSponsorshipFundId(address to) pure returns(bool, bytes32)
func (_Registry *RegistrySession) ContractSponsorshipFundId(to common.Address) (bool, [32]byte, error) {
	return _Registry.Contract.ContractSponsorshipFundId(&_Registry.CallOpts, to)
}

// ContractSponsorshipFundId is a free data retrieval call binding the contract method 0xe327d1ac.
//
// Solidity: function contractSponsorshipFundId(address to) pure returns(bool, bytes32)
func (_Registry *RegistryCallerSession) ContractSponsorshipFundId(to common.Address) (bool, [32]byte, error) {
	return _Registry.Contract.ContractSponsorshipFundId(&_Registry.CallOpts, to)
}

// GlobalSponsorshipFundId is a free data retrieval call binding the contract method 0x779a43ac.
//
// Solidity: function globalSponsorshipFundId() pure returns(bool, bytes32)
func (_Registry *RegistryCaller) GlobalSponsorshipFundId(opts *bind.CallOpts) (bool, [32]byte, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "globalSponsorshipFundId")

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// GlobalSponsorshipFundId is a free data retrieval call binding the contract method 0x779a43ac.
//
// Solidity: function globalSponsorshipFundId() pure returns(bool, bytes32)
func (_Registry *RegistrySession) GlobalSponsorshipFundId() (bool, [32]byte, error) {
	return _Registry.Contract.GlobalSponsorshipFundId(&_Registry.CallOpts)
}

// GlobalSponsorshipFundId is a free data retrieval call binding the contract method 0x779a43ac.
//
// Solidity: function globalSponsorshipFundId() pure returns(bool, bytes32)
func (_Registry *RegistryCallerSession) GlobalSponsorshipFundId() (bool, [32]byte, error) {
	return _Registry.Contract.GlobalSponsorshipFundId(&_Registry.CallOpts)
}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 id) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) Sponsorships(opts *bind.CallOpts, id [32]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "sponsorships", id)

	outstruct := new(struct {
		Funds              *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 id) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) Sponsorships(id [32]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.Sponsorships(&_Registry.CallOpts, id)
}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 id) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) Sponsorships(id [32]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.Sponsorships(&_Registry.CallOpts, id)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_Registry *RegistryTransactor) DeductFees(opts *bind.TransactOpts, fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "deductFees", fundId, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_Registry *RegistrySession) DeductFees(fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.DeductFees(&_Registry.TransactOpts, fundId, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_Registry *RegistryTransactorSession) DeductFees(fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.DeductFees(&_Registry.TransactOpts, fundId, fee)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_Registry *RegistryTransactor) Sponsor(opts *bind.TransactOpts, fundId [32]byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsor", fundId)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_Registry *RegistrySession) Sponsor(fundId [32]byte) (*types.Transaction, error) {
	return _Registry.Contract.Sponsor(&_Registry.TransactOpts, fundId)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_Registry *RegistryTransactorSession) Sponsor(fundId [32]byte) (*types.Transaction, error) {
	return _Registry.Contract.Sponsor(&_Registry.TransactOpts, fundId)
}

// Withdraw is a paid mutator transaction binding the contract method 0x040cf020.
//
// Solidity: function withdraw(bytes32 fundId, uint256 amount) returns()
func (_Registry *RegistryTransactor) Withdraw(opts *bind.TransactOpts, fundId [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdraw", fundId, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x040cf020.
//
// Solidity: function withdraw(bytes32 fundId, uint256 amount) returns()
func (_Registry *RegistrySession) Withdraw(fundId [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.Withdraw(&_Registry.TransactOpts, fundId, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x040cf020.
//
// Solidity: function withdraw(bytes32 fundId, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) Withdraw(fundId [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.Withdraw(&_Registry.TransactOpts, fundId, amount)
}
