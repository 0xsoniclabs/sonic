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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"approvalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"bootstrapSponsorshipFund\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"callSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"contractSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"isCovered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"covered\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610c098061001c5f395ff3fe60806040526004361061009a575f3560e01c8063779a43ac11610062578063779a43ac146101575780639ec88e991461016b578063a5dc45181461017e578063b9ed9f261461019d578063e327d1ac146101bc578063fecb2bc3146101db575f5ffd5b8063040cf0201461009e5780630ad1fcfc146100bf5780630f19ea1f146100fa57806351ee41a01461011957806363f2cdca14610138575b5f5ffd5b3480156100a9575f5ffd5b506100bd6100b8366004610948565b610222565b005b3480156100ca575f5ffd5b506100de6100d93660046109c4565b61042f565b6040805192151583526020830191909152015b60405180910390f35b348015610105575f5ffd5b506100de610114366004610a25565b61052e565b348015610124575f5ffd5b506100de610133366004610a9b565b61068a565b348015610143575f5ffd5b506100de610152366004610abd565b6106d5565b348015610162575f5ffd5b506100de610702565b6100bd610179366004610abd565b61073b565b348015610189575f5ffd5b506100de6101983660046109c4565b6107a4565b3480156101a8575f5ffd5b506100bd6101b7366004610948565b61084a565b3480156101c7575f5ffd5b506100de6101d6366004610a9b565b610918565b3480156101e6575f5ffd5b5061020d6101f5366004610abd565b5f602081905290815260409020805460019091015482565b604080519283526020830191909152016100f1565b5f3a1161029c5760405162461bcd60e51b815260206004820152603c60248201527f5769746864726177616c7320617265206e6f7420737570706f7274656420746860448201527f726f7567682073706f6e736f726564207472616e73616374696f6e730000000060648201526084015b60405180910390fd5b5f8281526020818152604080832033808552600282019093529220548311156103135760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b6064820152608401610293565b600182015482545f91906103279086610ae8565b6103319190610b05565b83549091508111156103945760405162461bcd60e51b815260206004820152602660248201527f4e6f7420656e6f75676820617661696c61626c652066756e647320746f20776960448201526574686472617760d01b6064820152608401610293565b6001600160a01b0382165f908152600284016020526040812080548692906103bd908490610b24565b9250508190555083836001015f8282546103d79190610b24565b90915550508254819084905f906103ef908490610b24565b90915550506040516001600160a01b0383169082156108fc029083905f818181858888f19350505050158015610427573d5f5f3e3d5ffd5b505050505050565b5f806001600160a01b0385161580610448575060448314155b1561045757505f905080610525565b5f6104656004828688610b37565b61046e91610b5e565b905063095ea7b360e01b6001600160e01b031982161461049457505f9150819050610525565b5f806104a3866004818a610b37565b8101906104b09190610b96565b9150915060018110156104cc57505f9350839250610525915050565b604051606160f81b60208201526001600160601b031960608b811b821660218401528a811b8216603584015284901b166049820152600190605d0160405160208183030381529060405280519060200120945094505050505b94509492505050565b5f5f61053c8888878761042f565b909250905081801561055b57505f818152602081905260409020548311155b15610569576001915061067f565b610575888887876107a4565b909250905081801561059457505f818152602081905260409020548311155b156105a2576001915061067f565b6105ab8861068a565b90925090508180156105ca57505f818152602081905260409020548311155b156105d8576001915061067f565b6105e187610918565b909250905081801561060057505f818152602081905260409020548311155b1561060e576001915061067f565b610617866106d5565b909250905081801561063657505f818152602081905260409020548311155b15610644576001915061067f565b61064c610702565b909250905081801561066b57505f818152602081905260409020548311155b15610679576001915061067f565b505f9050805b965096945050505050565b604051606160f81b60208201526001600160601b0319606083901b1660218201525f9081906001906035015b6040516020818303038152906040528051906020012091509150915091565b5f5f60038310156106f857604051603160f91b60208201526001906021016106b6565b505f928392509050565b5f5f600160405160200161071d90606760f81b815260010190565b60405160208183030381529060405280519060200120915091509091565b5f81815260208190526040812080549091349183919061075c908490610bc0565b9091555050335f90815260028201602052604081208054349290610781908490610bc0565b9250508190555034816001015f82825461079b9190610bc0565b90915550505050565b5f806001600160a01b03851615806107bc5750600483105b156107cb57505f905080610525565b5f6107d96004828688610b37565b6107e291610b5e565b604051606360f81b60208201526001600160601b031960608a811b8216602184015289901b1660358201526001600160e01b031982166049820152909150600190604d0160405160208183030381529060405280519060200120925092505094509492505050565b3315610854575f5ffd5b5f82815260208190526040902080548211156108a55760405162461bcd60e51b815260206004820152601060248201526f4e6f7420656e6f7567682066756e647360801b6044820152606401610293565b637e007d6760811b6001600160a01b031663850a10c0836040518263ffffffff1660e01b81526004015f604051808303818588803b1580156108e5575f5ffd5b505af11580156108f7573d5f5f3e3d5ffd5b505050505081815f015f82825461090e9190610b24565b9091555050505050565b604051606360f81b60208201526001600160601b0319606083901b1660218201525f9081906001906035016106b6565b5f5f60408385031215610959575f5ffd5b50508035926020909101359150565b6001600160a01b038116811461097c575f5ffd5b50565b5f5f83601f84011261098f575f5ffd5b50813567ffffffffffffffff8111156109a6575f5ffd5b6020830191508360208285010111156109bd575f5ffd5b9250929050565b5f5f5f5f606085870312156109d7575f5ffd5b84356109e281610968565b935060208501356109f281610968565b9250604085013567ffffffffffffffff811115610a0d575f5ffd5b610a198782880161097f565b95989497509550505050565b5f5f5f5f5f5f60a08789031215610a3a575f5ffd5b8635610a4581610968565b95506020870135610a5581610968565b945060408701359350606087013567ffffffffffffffff811115610a77575f5ffd5b610a8389828a0161097f565b979a9699509497949695608090950135949350505050565b5f60208284031215610aab575f5ffd5b8135610ab681610968565b9392505050565b5f60208284031215610acd575f5ffd5b5035919050565b634e487b7160e01b5f52601160045260245ffd5b8082028115828204841417610aff57610aff610ad4565b92915050565b5f82610b1f57634e487b7160e01b5f52601260045260245ffd5b500490565b81810381811115610aff57610aff610ad4565b5f5f85851115610b45575f5ffd5b83861115610b51575f5ffd5b5050820193919092039150565b80356001600160e01b03198116906004841015610b8f576001600160e01b0319600485900360031b81901b82161691505b5092915050565b5f5f60408385031215610ba7575f5ffd5b8235610bb281610968565b946020939093013593505050565b80820180821115610aff57610aff610ad456fea2646970667358221220d7664a6a43603996a44a8084e7e05c882308fe204f77c32eb7a46130419bcc9d64736f6c634300081b0033",
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

// IsCovered is a free data retrieval call binding the contract method 0x0f19ea1f.
//
// Solidity: function isCovered(address from, address to, uint256 nonce, bytes callData, uint256 fee) view returns(bool covered, bytes32 fundId)
func (_Registry *RegistryCaller) IsCovered(opts *bind.CallOpts, from common.Address, to common.Address, nonce *big.Int, callData []byte, fee *big.Int) (struct {
	Covered bool
	FundId  [32]byte
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isCovered", from, to, nonce, callData, fee)

	outstruct := new(struct {
		Covered bool
		FundId  [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Covered = *abi.ConvertType(out[0], new(bool)).(*bool)
	outstruct.FundId = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// IsCovered is a free data retrieval call binding the contract method 0x0f19ea1f.
//
// Solidity: function isCovered(address from, address to, uint256 nonce, bytes callData, uint256 fee) view returns(bool covered, bytes32 fundId)
func (_Registry *RegistrySession) IsCovered(from common.Address, to common.Address, nonce *big.Int, callData []byte, fee *big.Int) (struct {
	Covered bool
	FundId  [32]byte
}, error) {
	return _Registry.Contract.IsCovered(&_Registry.CallOpts, from, to, nonce, callData, fee)
}

// IsCovered is a free data retrieval call binding the contract method 0x0f19ea1f.
//
// Solidity: function isCovered(address from, address to, uint256 nonce, bytes callData, uint256 fee) view returns(bool covered, bytes32 fundId)
func (_Registry *RegistryCallerSession) IsCovered(from common.Address, to common.Address, nonce *big.Int, callData []byte, fee *big.Int) (struct {
	Covered bool
	FundId  [32]byte
}, error) {
	return _Registry.Contract.IsCovered(&_Registry.CallOpts, from, to, nonce, callData, fee)
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
