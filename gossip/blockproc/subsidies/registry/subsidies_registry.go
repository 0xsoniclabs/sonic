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
	ABI: "[{\"inputs\":[{\"internalType\":\"contractFeeBurner\",\"name\":\"feeBurner_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"contractSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"isCovered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"operationSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorContract\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorMethod\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorUser\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawContractSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawMethodSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawUserSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x6080604052348015600e575f80fd5b50604051610ad1380380610ad1833981016040819052602b91604e565b5f80546001600160a01b0319166001600160a01b03929092169190911790556079565b5f60208284031215605d575f80fd5b81516001600160a01b03811681146072575f80fd5b9392505050565b610a4b806100865f395ff3fe60806040526004361061009d575f3560e01c8063aae8311011610062578063aae8311014610198578063b212ba44146101e7578063cc77aec814610224578063daf21aa314610256578063eb7a3e2e14610269578063f1cdef0614610288575f80fd5b80632cc051571461010057806336a656a7146101135780633f49695a146101475780634f6a93ba14610166578063944557d614610179575f80fd5b366100fc5760405162461bcd60e51b815260206004820152602260248201527f5573652073706f6e736f722066756e6374696f6e7320746f206164642066756e604482015261647360f01b60648201526084015b60405180910390fd5b005b5f80fd5b6100fa61010e366004610817565b6102a7565b34801561011e575f80fd5b5061013261012d36600461085f565b6102da565b60405190151581526020015b60405180910390f35b348015610152575f80fd5b506100fa6101613660046108a7565b61037b565b6100fa6101743660046108e0565b6103af565b348015610184575f80fd5b506100fa61019336600461085f565b6103e6565b3480156101a3575f80fd5b506101d26101b2366004610817565b600160208181525f93845260408085209091529183529120805491015482565b6040805192835260208301919091520161013e565b3480156101f2575f80fd5b506101d26102013660046108e0565b600260209081525f92835260408084209091529082529020805460019091015482565b34801561022f575f80fd5b506101d261023e366004610908565b60036020525f90815260409020805460019091015482565b6100fa610264366004610908565b61058d565b348015610274575f80fd5b506100fa610283366004610928565b6105b2565b348015610293575f80fd5b506100fa6102a2366004610951565b6105e9565b6001600160a01b038083165f9081526001602090815260408083209385168352929052206102d690333461060b565b5050565b6001600160a01b038085165f908152600160209081526040808320938716835292905290812054821161030f57506001610373565b6001600160a01b0384165f9081526002602090815260408083206001600160e01b031987168452909152902054821161034a57506001610373565b6001600160a01b0384165f90815260036020526040902054821161037057506001610373565b505f5b949350505050565b6001600160a01b038084165f9081526001602090815260408083209386168352929052206103aa90338361066f565b505050565b6001600160a01b0382165f9081526002602090815260408083206001600160e01b03198516845290915290206102d690333461060b565b33156103f0575f80fd5b6103fc848484846102da565b610404575f80fd5b5f8054906101000a90046001600160a01b03166001600160a01b031663850a10c0826040518263ffffffff1660e01b81526004015f604051808303818588803b15801561044f575f80fd5b505af1158015610461573d5f803e3d5ffd5b505050506001600160a01b038581165f90815260016020908152604080832093881683529290522054821190506104d3576001600160a01b038085165f908152600160209081526040808320938716835292905290812080548392906104c890849061098d565b909155506105879050565b6001600160a01b0383165f9081526002602090815260408083206001600160e01b0319861684529091529020548111610542576001600160a01b0383165f9081526002602090815260408083206001600160e01b031986168452909152812080548392906104c890849061098d565b6001600160a01b0383165f908152600360205260409020548111610587576001600160a01b0383165f90815260036020526040812080548392906104c890849061098d565b50505050565b6001600160a01b0381165f9081526003602052604090206105af90333461060b565b50565b6001600160a01b0383165f9081526002602090815260408083206001600160e01b03198616845290915290206103aa90338361066f565b6001600160a01b0382165f9081526003602052604090206102d690338361066f565b80835f015f82825461061d91906109a6565b90915550506001600160a01b0382165f9081526002840160205260408120805483929061064b9084906109a6565b9250508190555080836001015f82825461066591906109a6565b9091555050505050565b6001600160a01b0382165f9081526002840160205260409020548111156106e45760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b60648201526084016100f1565b600183015483545f91906106f890846109b9565b61070291906109d0565b90505f836001600160a01b0316826040515f6040518083038185875af1925050503d805f811461074d576040519150601f19603f3d011682016040523d82523d5f602084013e610752565b606091505b50509050806107955760405162461bcd60e51b815260206004820152600f60248201526e151c985b9cd9995c8819985a5b1959608a1b60448201526064016100f1565b6001600160a01b0384165f908152600286016020526040812080548592906107be90849061098d565b9250508190555082856001015f8282546107d8919061098d565b90915550508454829086905f906107f090849061098d565b90915550505050505050565b80356001600160a01b0381168114610812575f80fd5b919050565b5f8060408385031215610828575f80fd5b610831836107fc565b915061083f602084016107fc565b90509250929050565b80356001600160e01b031981168114610812575f80fd5b5f805f8060808587031215610872575f80fd5b61087b856107fc565b9350610889602086016107fc565b925061089760408601610848565b9396929550929360600135925050565b5f805f606084860312156108b9575f80fd5b6108c2846107fc565b92506108d0602085016107fc565b9150604084013590509250925092565b5f80604083850312156108f1575f80fd5b6108fa836107fc565b915061083f60208401610848565b5f60208284031215610918575f80fd5b610921826107fc565b9392505050565b5f805f6060848603121561093a575f80fd5b610943846107fc565b92506108d060208501610848565b5f8060408385031215610962575f80fd5b61096b836107fc565b946020939093013593505050565b634e487b7160e01b5f52601160045260245ffd5b818103818111156109a0576109a0610979565b92915050565b808201808211156109a0576109a0610979565b80820281158282048414176109a0576109a0610979565b5f826109ea57634e487b7160e01b5f52601260045260245ffd5b50049056fea2646970667358221220c5fb123a12261d889cdf1679bf25d39eede15b5fa3320f834102c964679aec5964736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
}

// RegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryMetaData.ABI instead.
var RegistryABI = RegistryMetaData.ABI

// RegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RegistryMetaData.Bin instead.
var RegistryBin = RegistryMetaData.Bin

// DeployRegistry deploys a new Ethereum contract, binding an instance of Registry to it.
func DeployRegistry(auth *bind.TransactOpts, backend bind.ContractBackend, feeBurner_ common.Address) (common.Address, *types.Transaction, *Registry, error) {
	parsed, err := RegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RegistryBin), backend, feeBurner_)
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

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) ContractSponsorships(opts *bind.CallOpts, arg0 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "contractSponsorships", arg0)

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

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) ContractSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ContractSponsorships(&_Registry.CallOpts, arg0)
}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) ContractSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ContractSponsorships(&_Registry.CallOpts, arg0)
}

// IsCovered is a free data retrieval call binding the contract method 0x36a656a7.
//
// Solidity: function isCovered(address from, address to, bytes4 functionSelector, uint256 fee) view returns(bool)
func (_Registry *RegistryCaller) IsCovered(opts *bind.CallOpts, from common.Address, to common.Address, functionSelector [4]byte, fee *big.Int) (bool, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "isCovered", from, to, functionSelector, fee)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCovered is a free data retrieval call binding the contract method 0x36a656a7.
//
// Solidity: function isCovered(address from, address to, bytes4 functionSelector, uint256 fee) view returns(bool)
func (_Registry *RegistrySession) IsCovered(from common.Address, to common.Address, functionSelector [4]byte, fee *big.Int) (bool, error) {
	return _Registry.Contract.IsCovered(&_Registry.CallOpts, from, to, functionSelector, fee)
}

// IsCovered is a free data retrieval call binding the contract method 0x36a656a7.
//
// Solidity: function isCovered(address from, address to, bytes4 functionSelector, uint256 fee) view returns(bool)
func (_Registry *RegistryCallerSession) IsCovered(from common.Address, to common.Address, functionSelector [4]byte, fee *big.Int) (bool, error) {
	return _Registry.Contract.IsCovered(&_Registry.CallOpts, from, to, functionSelector, fee)
}

// OperationSponsorships is a free data retrieval call binding the contract method 0xb212ba44.
//
// Solidity: function operationSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) OperationSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "operationSponsorships", arg0, arg1)

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

// OperationSponsorships is a free data retrieval call binding the contract method 0xb212ba44.
//
// Solidity: function operationSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) OperationSponsorships(arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.OperationSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// OperationSponsorships is a free data retrieval call binding the contract method 0xb212ba44.
//
// Solidity: function operationSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) OperationSponsorships(arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.OperationSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) UserSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "userSponsorships", arg0, arg1)

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

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) UserSponsorships(arg0 common.Address, arg1 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.UserSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) UserSponsorships(arg0 common.Address, arg1 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.UserSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// DeductFees is a paid mutator transaction binding the contract method 0x944557d6.
//
// Solidity: function deductFees(address from, address to, bytes4 functionSelector, uint256 fee) returns()
func (_Registry *RegistryTransactor) DeductFees(opts *bind.TransactOpts, from common.Address, to common.Address, functionSelector [4]byte, fee *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "deductFees", from, to, functionSelector, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0x944557d6.
//
// Solidity: function deductFees(address from, address to, bytes4 functionSelector, uint256 fee) returns()
func (_Registry *RegistrySession) DeductFees(from common.Address, to common.Address, functionSelector [4]byte, fee *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.DeductFees(&_Registry.TransactOpts, from, to, functionSelector, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0x944557d6.
//
// Solidity: function deductFees(address from, address to, bytes4 functionSelector, uint256 fee) returns()
func (_Registry *RegistryTransactorSession) DeductFees(from common.Address, to common.Address, functionSelector [4]byte, fee *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.DeductFees(&_Registry.TransactOpts, from, to, functionSelector, fee)
}

// SponsorContract is a paid mutator transaction binding the contract method 0xdaf21aa3.
//
// Solidity: function sponsorContract(address to) payable returns()
func (_Registry *RegistryTransactor) SponsorContract(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorContract", to)
}

// SponsorContract is a paid mutator transaction binding the contract method 0xdaf21aa3.
//
// Solidity: function sponsorContract(address to) payable returns()
func (_Registry *RegistrySession) SponsorContract(to common.Address) (*types.Transaction, error) {
	return _Registry.Contract.SponsorContract(&_Registry.TransactOpts, to)
}

// SponsorContract is a paid mutator transaction binding the contract method 0xdaf21aa3.
//
// Solidity: function sponsorContract(address to) payable returns()
func (_Registry *RegistryTransactorSession) SponsorContract(to common.Address) (*types.Transaction, error) {
	return _Registry.Contract.SponsorContract(&_Registry.TransactOpts, to)
}

// SponsorMethod is a paid mutator transaction binding the contract method 0x4f6a93ba.
//
// Solidity: function sponsorMethod(address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistryTransactor) SponsorMethod(opts *bind.TransactOpts, to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorMethod", to, functionSelector)
}

// SponsorMethod is a paid mutator transaction binding the contract method 0x4f6a93ba.
//
// Solidity: function sponsorMethod(address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistrySession) SponsorMethod(to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.Contract.SponsorMethod(&_Registry.TransactOpts, to, functionSelector)
}

// SponsorMethod is a paid mutator transaction binding the contract method 0x4f6a93ba.
//
// Solidity: function sponsorMethod(address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistryTransactorSession) SponsorMethod(to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.Contract.SponsorMethod(&_Registry.TransactOpts, to, functionSelector)
}

// SponsorUser is a paid mutator transaction binding the contract method 0x2cc05157.
//
// Solidity: function sponsorUser(address from, address to) payable returns()
func (_Registry *RegistryTransactor) SponsorUser(opts *bind.TransactOpts, from common.Address, to common.Address) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorUser", from, to)
}

// SponsorUser is a paid mutator transaction binding the contract method 0x2cc05157.
//
// Solidity: function sponsorUser(address from, address to) payable returns()
func (_Registry *RegistrySession) SponsorUser(from common.Address, to common.Address) (*types.Transaction, error) {
	return _Registry.Contract.SponsorUser(&_Registry.TransactOpts, from, to)
}

// SponsorUser is a paid mutator transaction binding the contract method 0x2cc05157.
//
// Solidity: function sponsorUser(address from, address to) payable returns()
func (_Registry *RegistryTransactorSession) SponsorUser(from common.Address, to common.Address) (*types.Transaction, error) {
	return _Registry.Contract.SponsorUser(&_Registry.TransactOpts, from, to)
}

// WithdrawContractSponsorship is a paid mutator transaction binding the contract method 0xf1cdef06.
//
// Solidity: function withdrawContractSponsorship(address to, uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawContractSponsorship(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawContractSponsorship", to, amount)
}

// WithdrawContractSponsorship is a paid mutator transaction binding the contract method 0xf1cdef06.
//
// Solidity: function withdrawContractSponsorship(address to, uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawContractSponsorship(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawContractSponsorship(&_Registry.TransactOpts, to, amount)
}

// WithdrawContractSponsorship is a paid mutator transaction binding the contract method 0xf1cdef06.
//
// Solidity: function withdrawContractSponsorship(address to, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawContractSponsorship(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawContractSponsorship(&_Registry.TransactOpts, to, amount)
}

// WithdrawMethodSponsorship is a paid mutator transaction binding the contract method 0xeb7a3e2e.
//
// Solidity: function withdrawMethodSponsorship(address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawMethodSponsorship(opts *bind.TransactOpts, to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawMethodSponsorship", to, functionSelector, amount)
}

// WithdrawMethodSponsorship is a paid mutator transaction binding the contract method 0xeb7a3e2e.
//
// Solidity: function withdrawMethodSponsorship(address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawMethodSponsorship(to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawMethodSponsorship(&_Registry.TransactOpts, to, functionSelector, amount)
}

// WithdrawMethodSponsorship is a paid mutator transaction binding the contract method 0xeb7a3e2e.
//
// Solidity: function withdrawMethodSponsorship(address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawMethodSponsorship(to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawMethodSponsorship(&_Registry.TransactOpts, to, functionSelector, amount)
}

// WithdrawUserSponsorship is a paid mutator transaction binding the contract method 0x3f49695a.
//
// Solidity: function withdrawUserSponsorship(address from, address to, uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawUserSponsorship(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawUserSponsorship", from, to, amount)
}

// WithdrawUserSponsorship is a paid mutator transaction binding the contract method 0x3f49695a.
//
// Solidity: function withdrawUserSponsorship(address from, address to, uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawUserSponsorship(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawUserSponsorship(&_Registry.TransactOpts, from, to, amount)
}

// WithdrawUserSponsorship is a paid mutator transaction binding the contract method 0x3f49695a.
//
// Solidity: function withdrawUserSponsorship(address from, address to, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawUserSponsorship(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawUserSponsorship(&_Registry.TransactOpts, from, to, amount)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Registry *RegistryTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Registry *RegistrySession) Receive() (*types.Transaction, error) {
	return _Registry.Contract.Receive(&_Registry.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Registry *RegistryTransactorSession) Receive() (*types.Transaction, error) {
	return _Registry.Contract.Receive(&_Registry.TransactOpts)
}
