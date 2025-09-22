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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"callSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"contractSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorship\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"isCovered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"serviceSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"sponsorAccount\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorContract\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sponsorGlobal\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorService\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorUser\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawAccountSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawCallSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawContractSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawGlobalSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawServiceSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawUserSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610e4b8061001c5f395ff3fe60806040526004361061011b575f3560e01c8063944557d61161009d578063cc77aec811610062578063cc77aec81461034e578063daf21aa314610380578063e32213bb14610393578063f1cdef06146103b2578063f8117aa8146103d1575f5ffd5b8063944557d61461025357806399bc8bee146102725780639c31691e14610291578063aae83110146102d4578063b5dfce0714610311575f5ffd5b80633f49695a116100e35780633f49695a146101d45780633ff8b209146101f3578063533c23c6146102255780636c2f0786146102385780637d2e55641461024b575f5ffd5b806302f8297c1461011f5780630c617f961461015257806317429963146101735780632cc051571461019257806336a656a7146101a5575b5f5ffd5b34801561012a575f5ffd5b505f54600154610138919082565b604080519283526020830191909152015b60405180910390f35b34801561015d575f5ffd5b5061017161016c366004610bca565b6103e4565b005b34801561017e575f5ffd5b5061017161018d366004610bfc565b6103f2565b6101716101a0366004610c24565b610418565b3480156101b0575f5ffd5b506101c46101bf366004610c6c565b610447565b6040519015158152602001610149565b3480156101df575f5ffd5b506101716101ee366004610cb4565b610460565b3480156101fe575f5ffd5b5061013861020d366004610cee565b60036020525f90815260409020805460019091015482565b610171610233366004610d0e565b610494565b610171610246366004610d36565b6104cb565b61017161050c565b34801561025e575f5ffd5b5061017161026d366004610c6c565b610519565b34801561027d575f5ffd5b5061017161028c366004610c6c565b610641565b34801561029c575f5ffd5b506101386102ab366004610d36565b600560209081525f93845260408085208252928452828420905282529020805460019091015482565b3480156102df575f5ffd5b506101386102ee366004610c24565b600460209081525f92835260408084209091529082529020805460019091015482565b34801561031c575f5ffd5b5061013861032b366004610d0e565b600660209081525f92835260408084209091529082529020805460019091015482565b348015610359575f5ffd5b50610138610368366004610cee565b60076020525f90815260409020805460019091015482565b61017161038e366004610cee565b610688565b34801561039e575f5ffd5b506101716103ad366004610d76565b6106aa565b3480156103bd575f5ffd5b506101716103cc366004610bfc565b6106e1565b6101716103df366004610cee565b610703565b6103ef5f3383610725565b50565b6001600160a01b0382165f908152600360205260409020610414903383610725565b5050565b6001600160a01b038083165f908152600460209081526040808320938516835292905220610414903334610988565b5f5f610455868686866109ec565b979650505050505050565b6001600160a01b038084165f90815260046020908152604080832093861683529290522061048f903383610725565b505050565b6001600160a01b0382165f9081526006602090815260408083206001600160e01b0319851684529091529020610414903334610988565b6001600160a01b038084165f90815260056020908152604080832093861683529281528282206001600160e01b03198516835290522061048f903334610988565b6105175f3334610988565b565b3315610523575f5ffd5b5f5f610531868686866109ec565b91509150806105875760405162461bcd60e51b815260206004820152601c60248201527f4e6f2073706f6e736f727368697020706f7420617661696c61626c650000000060448201526064015b60405180910390fd5b81548311156105cb5760405162461bcd60e51b815260206004820152601060248201526f4e6f7420656e6f7567682066756e647360801b604482015260640161057e565b637e007d6760811b6001600160a01b031663850a10c0846040518263ffffffff1660e01b81526004015f604051808303818588803b15801561060b575f5ffd5b505af115801561061d573d5f5f3e3d5ffd5b505050505082825f015f8282546106349190610db3565b9091555050505050505050565b6001600160a01b038085165f90815260056020908152604080832093871683529281528282206001600160e01b031986168352905220610682903383610725565b50505050565b6001600160a01b0381165f9081526007602052604090206103ef903334610988565b6001600160a01b0383165f9081526006602090815260408083206001600160e01b031986168452909152902061048f903383610725565b6001600160a01b0382165f908152600760205260409020610414903383610725565b6001600160a01b0381165f9081526003602052604090206103ef903334610988565b5f3a1161079a5760405162461bcd60e51b815260206004820152603c60248201527f5769746864726177616c7320617265206e6f7420737570706f7274656420746860448201527f726f7567682073706f6e736f726564207472616e73616374696f6e7300000000606482015260840161057e565b6001600160a01b0382165f90815260028401602052604090205481111561080f5760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b606482015260840161057e565b600183015483545f91906108239084610dcc565b61082d9190610de3565b84549091508111156108905760405162461bcd60e51b815260206004820152602660248201527f4e6f7420656e6f75676820617661696c61626c652066756e647320746f20776960448201526574686472617760d01b606482015260840161057e565b5f836001600160a01b0316826040515f6040518083038185875af1925050503d805f81146108d9576040519150601f19603f3d011682016040523d82523d5f602084013e6108de565b606091505b50509050806109215760405162461bcd60e51b815260206004820152600f60248201526e151c985b9cd9995c8819985a5b1959608a1b604482015260640161057e565b6001600160a01b0384165f9081526002860160205260408120805485929061094a908490610db3565b9250508190555082856001015f8282546109649190610db3565b90915550508454829086905f9061097c908490610db3565b90915550505050505050565b80835f015f82825461099a9190610e02565b90915550506001600160a01b0382165f908152600284016020526040812080548392906109c8908490610e02565b9250508190555080836001015f8282546109e29190610e02565b9091555050505050565b6001600160a01b038085165f90815260056020908152604080832093871683529281528282206001600160e01b03198616835290529081205481908311610a6c5750506001600160a01b038085165f90815260056020908152604080832093871683529281528282206001600160e01b0319861683529052206001610bc1565b6001600160a01b038087165f908152600460209081526040808320938916835292905220548311610ac45750506001600160a01b038085165f9081526004602090815260408083209387168352929052206001610bc1565b6001600160a01b0386165f908152600360205260409020548311610b025750506001600160a01b0384165f9081526003602052604090206001610bc1565b6001600160a01b0385165f9081526006602090815260408083206001600160e01b0319881684529091529020548311610b6a5750506001600160a01b0383165f9081526006602090815260408083206001600160e01b03198616845290915290206001610bc1565b6001600160a01b0385165f908152600760205260409020548311610ba85750506001600160a01b0383165f9081526007602052604090206001610bc1565b5f548311610bbb57505f90506001610bc1565b505f9050805b94509492505050565b5f60208284031215610bda575f5ffd5b5035919050565b80356001600160a01b0381168114610bf7575f5ffd5b919050565b5f5f60408385031215610c0d575f5ffd5b610c1683610be1565b946020939093013593505050565b5f5f60408385031215610c35575f5ffd5b610c3e83610be1565b9150610c4c60208401610be1565b90509250929050565b80356001600160e01b031981168114610bf7575f5ffd5b5f5f5f5f60808587031215610c7f575f5ffd5b610c8885610be1565b9350610c9660208601610be1565b9250610ca460408601610c55565b9396929550929360600135925050565b5f5f5f60608486031215610cc6575f5ffd5b610ccf84610be1565b9250610cdd60208501610be1565b929592945050506040919091013590565b5f60208284031215610cfe575f5ffd5b610d0782610be1565b9392505050565b5f5f60408385031215610d1f575f5ffd5b610d2883610be1565b9150610c4c60208401610c55565b5f5f5f60608486031215610d48575f5ffd5b610d5184610be1565b9250610d5f60208501610be1565b9150610d6d60408501610c55565b90509250925092565b5f5f5f60608486031215610d88575f5ffd5b610d9184610be1565b9250610cdd60208501610c55565b634e487b7160e01b5f52601160045260245ffd5b81810381811115610dc657610dc6610d9f565b92915050565b8082028115828204841417610dc657610dc6610d9f565b5f82610dfd57634e487b7160e01b5f52601260045260245ffd5b500490565b80820180821115610dc657610dc6610d9f56fea2646970667358221220377f4f363871a2cd0b945a706e115889ee4a1398c9305533cec55df708bd105764736f6c634300081b0033",
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

// AccountSponsorships is a free data retrieval call binding the contract method 0x3ff8b209.
//
// Solidity: function accountSponsorships(address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) AccountSponsorships(opts *bind.CallOpts, arg0 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "accountSponsorships", arg0)

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

// AccountSponsorships is a free data retrieval call binding the contract method 0x3ff8b209.
//
// Solidity: function accountSponsorships(address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) AccountSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.AccountSponsorships(&_Registry.CallOpts, arg0)
}

// AccountSponsorships is a free data retrieval call binding the contract method 0x3ff8b209.
//
// Solidity: function accountSponsorships(address ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) AccountSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.AccountSponsorships(&_Registry.CallOpts, arg0)
}

// CallSponsorships is a free data retrieval call binding the contract method 0x9c31691e.
//
// Solidity: function callSponsorships(address , address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) CallSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "callSponsorships", arg0, arg1, arg2)

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

// CallSponsorships is a free data retrieval call binding the contract method 0x9c31691e.
//
// Solidity: function callSponsorships(address , address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) CallSponsorships(arg0 common.Address, arg1 common.Address, arg2 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.CallSponsorships(&_Registry.CallOpts, arg0, arg1, arg2)
}

// CallSponsorships is a free data retrieval call binding the contract method 0x9c31691e.
//
// Solidity: function callSponsorships(address , address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) CallSponsorships(arg0 common.Address, arg1 common.Address, arg2 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.CallSponsorships(&_Registry.CallOpts, arg0, arg1, arg2)
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

// GlobalSponsorship is a free data retrieval call binding the contract method 0x02f8297c.
//
// Solidity: function globalSponsorship() view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) GlobalSponsorship(opts *bind.CallOpts) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "globalSponsorship")

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

// GlobalSponsorship is a free data retrieval call binding the contract method 0x02f8297c.
//
// Solidity: function globalSponsorship() view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) GlobalSponsorship() (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.GlobalSponsorship(&_Registry.CallOpts)
}

// GlobalSponsorship is a free data retrieval call binding the contract method 0x02f8297c.
//
// Solidity: function globalSponsorship() view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) GlobalSponsorship() (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.GlobalSponsorship(&_Registry.CallOpts)
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

// ServiceSponsorships is a free data retrieval call binding the contract method 0xb5dfce07.
//
// Solidity: function serviceSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCaller) ServiceSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "serviceSponsorships", arg0, arg1)

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

// ServiceSponsorships is a free data retrieval call binding the contract method 0xb5dfce07.
//
// Solidity: function serviceSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistrySession) ServiceSponsorships(arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ServiceSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// ServiceSponsorships is a free data retrieval call binding the contract method 0xb5dfce07.
//
// Solidity: function serviceSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 totalContributions)
func (_Registry *RegistryCallerSession) ServiceSponsorships(arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ServiceSponsorships(&_Registry.CallOpts, arg0, arg1)
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

// SponsorAccount is a paid mutator transaction binding the contract method 0xf8117aa8.
//
// Solidity: function sponsorAccount(address from) payable returns()
func (_Registry *RegistryTransactor) SponsorAccount(opts *bind.TransactOpts, from common.Address) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorAccount", from)
}

// SponsorAccount is a paid mutator transaction binding the contract method 0xf8117aa8.
//
// Solidity: function sponsorAccount(address from) payable returns()
func (_Registry *RegistrySession) SponsorAccount(from common.Address) (*types.Transaction, error) {
	return _Registry.Contract.SponsorAccount(&_Registry.TransactOpts, from)
}

// SponsorAccount is a paid mutator transaction binding the contract method 0xf8117aa8.
//
// Solidity: function sponsorAccount(address from) payable returns()
func (_Registry *RegistryTransactorSession) SponsorAccount(from common.Address) (*types.Transaction, error) {
	return _Registry.Contract.SponsorAccount(&_Registry.TransactOpts, from)
}

// SponsorCall is a paid mutator transaction binding the contract method 0x6c2f0786.
//
// Solidity: function sponsorCall(address from, address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistryTransactor) SponsorCall(opts *bind.TransactOpts, from common.Address, to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorCall", from, to, functionSelector)
}

// SponsorCall is a paid mutator transaction binding the contract method 0x6c2f0786.
//
// Solidity: function sponsorCall(address from, address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistrySession) SponsorCall(from common.Address, to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.Contract.SponsorCall(&_Registry.TransactOpts, from, to, functionSelector)
}

// SponsorCall is a paid mutator transaction binding the contract method 0x6c2f0786.
//
// Solidity: function sponsorCall(address from, address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistryTransactorSession) SponsorCall(from common.Address, to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.Contract.SponsorCall(&_Registry.TransactOpts, from, to, functionSelector)
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

// SponsorGlobal is a paid mutator transaction binding the contract method 0x7d2e5564.
//
// Solidity: function sponsorGlobal() payable returns()
func (_Registry *RegistryTransactor) SponsorGlobal(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorGlobal")
}

// SponsorGlobal is a paid mutator transaction binding the contract method 0x7d2e5564.
//
// Solidity: function sponsorGlobal() payable returns()
func (_Registry *RegistrySession) SponsorGlobal() (*types.Transaction, error) {
	return _Registry.Contract.SponsorGlobal(&_Registry.TransactOpts)
}

// SponsorGlobal is a paid mutator transaction binding the contract method 0x7d2e5564.
//
// Solidity: function sponsorGlobal() payable returns()
func (_Registry *RegistryTransactorSession) SponsorGlobal() (*types.Transaction, error) {
	return _Registry.Contract.SponsorGlobal(&_Registry.TransactOpts)
}

// SponsorService is a paid mutator transaction binding the contract method 0x533c23c6.
//
// Solidity: function sponsorService(address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistryTransactor) SponsorService(opts *bind.TransactOpts, to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "sponsorService", to, functionSelector)
}

// SponsorService is a paid mutator transaction binding the contract method 0x533c23c6.
//
// Solidity: function sponsorService(address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistrySession) SponsorService(to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.Contract.SponsorService(&_Registry.TransactOpts, to, functionSelector)
}

// SponsorService is a paid mutator transaction binding the contract method 0x533c23c6.
//
// Solidity: function sponsorService(address to, bytes4 functionSelector) payable returns()
func (_Registry *RegistryTransactorSession) SponsorService(to common.Address, functionSelector [4]byte) (*types.Transaction, error) {
	return _Registry.Contract.SponsorService(&_Registry.TransactOpts, to, functionSelector)
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

// WithdrawAccountSponsorship is a paid mutator transaction binding the contract method 0x17429963.
//
// Solidity: function withdrawAccountSponsorship(address from, uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawAccountSponsorship(opts *bind.TransactOpts, from common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawAccountSponsorship", from, amount)
}

// WithdrawAccountSponsorship is a paid mutator transaction binding the contract method 0x17429963.
//
// Solidity: function withdrawAccountSponsorship(address from, uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawAccountSponsorship(from common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawAccountSponsorship(&_Registry.TransactOpts, from, amount)
}

// WithdrawAccountSponsorship is a paid mutator transaction binding the contract method 0x17429963.
//
// Solidity: function withdrawAccountSponsorship(address from, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawAccountSponsorship(from common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawAccountSponsorship(&_Registry.TransactOpts, from, amount)
}

// WithdrawCallSponsorship is a paid mutator transaction binding the contract method 0x99bc8bee.
//
// Solidity: function withdrawCallSponsorship(address from, address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawCallSponsorship(opts *bind.TransactOpts, from common.Address, to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawCallSponsorship", from, to, functionSelector, amount)
}

// WithdrawCallSponsorship is a paid mutator transaction binding the contract method 0x99bc8bee.
//
// Solidity: function withdrawCallSponsorship(address from, address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawCallSponsorship(from common.Address, to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawCallSponsorship(&_Registry.TransactOpts, from, to, functionSelector, amount)
}

// WithdrawCallSponsorship is a paid mutator transaction binding the contract method 0x99bc8bee.
//
// Solidity: function withdrawCallSponsorship(address from, address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawCallSponsorship(from common.Address, to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawCallSponsorship(&_Registry.TransactOpts, from, to, functionSelector, amount)
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

// WithdrawGlobalSponsorship is a paid mutator transaction binding the contract method 0x0c617f96.
//
// Solidity: function withdrawGlobalSponsorship(uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawGlobalSponsorship(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawGlobalSponsorship", amount)
}

// WithdrawGlobalSponsorship is a paid mutator transaction binding the contract method 0x0c617f96.
//
// Solidity: function withdrawGlobalSponsorship(uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawGlobalSponsorship(amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawGlobalSponsorship(&_Registry.TransactOpts, amount)
}

// WithdrawGlobalSponsorship is a paid mutator transaction binding the contract method 0x0c617f96.
//
// Solidity: function withdrawGlobalSponsorship(uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawGlobalSponsorship(amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawGlobalSponsorship(&_Registry.TransactOpts, amount)
}

// WithdrawServiceSponsorship is a paid mutator transaction binding the contract method 0xe32213bb.
//
// Solidity: function withdrawServiceSponsorship(address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistryTransactor) WithdrawServiceSponsorship(opts *bind.TransactOpts, to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.contract.Transact(opts, "withdrawServiceSponsorship", to, functionSelector, amount)
}

// WithdrawServiceSponsorship is a paid mutator transaction binding the contract method 0xe32213bb.
//
// Solidity: function withdrawServiceSponsorship(address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistrySession) WithdrawServiceSponsorship(to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawServiceSponsorship(&_Registry.TransactOpts, to, functionSelector, amount)
}

// WithdrawServiceSponsorship is a paid mutator transaction binding the contract method 0xe32213bb.
//
// Solidity: function withdrawServiceSponsorship(address to, bytes4 functionSelector, uint256 amount) returns()
func (_Registry *RegistryTransactorSession) WithdrawServiceSponsorship(to common.Address, functionSelector [4]byte, amount *big.Int) (*types.Transaction, error) {
	return _Registry.Contract.WithdrawServiceSponsorship(&_Registry.TransactOpts, to, functionSelector, amount)
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
