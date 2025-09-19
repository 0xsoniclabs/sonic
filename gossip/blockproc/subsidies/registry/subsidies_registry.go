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
	ABI: "[{\"inputs\":[{\"internalType\":\"contractFeeBurner\",\"name\":\"feeBurner_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"callSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"contractSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorship\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"isCovered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"serviceSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"sponsorAccount\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorContract\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sponsorGlobal\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorService\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorUser\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawAccountSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawCallSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawContractSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawGlobalSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawServiceSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawUserSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50604051610ed9380380610ed9833981016040819052602b91604e565b5f80546001600160a01b0319166001600160a01b03929092169190911790556079565b5f60208284031215605d575f5ffd5b81516001600160a01b03811681146072575f5ffd5b9392505050565b610e53806100865f395ff3fe60806040526004361061011b575f3560e01c8063944557d61161009d578063cc77aec811610062578063cc77aec81461034f578063daf21aa314610381578063e32213bb14610394578063f1cdef06146103b3578063f8117aa8146103d2575f5ffd5b8063944557d61461025457806399bc8bee146102735780639c31691e14610292578063aae83110146102d5578063b5dfce0714610312575f5ffd5b80633f49695a116100e35780633f49695a146101d55780633ff8b209146101f4578063533c23c6146102265780636c2f0786146102395780637d2e55641461024c575f5ffd5b806302f8297c1461011f5780630c617f961461015357806317429963146101745780632cc051571461019357806336a656a7146101a6575b5f5ffd5b34801561012a575f5ffd5b50600154600254610139919082565b604080519283526020830191909152015b60405180910390f35b34801561015e575f5ffd5b5061017261016d366004610bd2565b6103e5565b005b34801561017f575f5ffd5b5061017261018e366004610c04565b6103f4565b6101726101a1366004610c2c565b61041a565b3480156101b1575f5ffd5b506101c56101c0366004610c74565b610449565b604051901515815260200161014a565b3480156101e0575f5ffd5b506101726101ef366004610cbc565b610462565b3480156101ff575f5ffd5b5061013961020e366004610cf6565b60046020525f90815260409020805460019091015482565b610172610234366004610d16565b610496565b610172610247366004610d3e565b6104cd565b61017261050e565b34801561025f575f5ffd5b5061017261026e366004610c74565b61051c565b34801561027e575f5ffd5b5061017261028d366004610c74565b610647565b34801561029d575f5ffd5b506101396102ac366004610d3e565b600660209081525f93845260408085208252928452828420905282529020805460019091015482565b3480156102e0575f5ffd5b506101396102ef366004610c2c565b600560209081525f92835260408084209091529082529020805460019091015482565b34801561031d575f5ffd5b5061013961032c366004610d16565b600760209081525f92835260408084209091529082529020805460019091015482565b34801561035a575f5ffd5b50610139610369366004610cf6565b60086020525f90815260409020805460019091015482565b61017261038f366004610cf6565b61068e565b34801561039f575f5ffd5b506101726103ae366004610d7e565b6106b0565b3480156103be575f5ffd5b506101726103cd366004610c04565b6106e7565b6101726103e0366004610cf6565b610709565b6103f16001338361072b565b50565b6001600160a01b0382165f90815260046020526040902061041690338361072b565b5050565b6001600160a01b038083165f90815260056020908152604080832093851683529290522061041690333461098e565b5f5f610457868686866109f2565b979650505050505050565b6001600160a01b038084165f90815260056020908152604080832093861683529290522061049190338361072b565b505050565b6001600160a01b0382165f9081526007602090815260408083206001600160e01b031985168452909152902061041690333461098e565b6001600160a01b038084165f90815260066020908152604080832093861683529281528282206001600160e01b03198516835290522061049190333461098e565b61051a6001333461098e565b565b3315610526575f5ffd5b5f5f610534868686866109f2565b915091508061058a5760405162461bcd60e51b815260206004820152601c60248201527f4e6f2073706f6e736f727368697020706f7420617661696c61626c650000000060448201526064015b60405180910390fd5b81548311156105ce5760405162461bcd60e51b815260206004820152601060248201526f4e6f7420656e6f7567682066756e647360801b6044820152606401610581565b5f805460408051630214284360e61b815290516001600160a01b039092169263850a10c0928792600480820193929182900301818588803b158015610611575f5ffd5b505af1158015610623573d5f5f3e3d5ffd5b505050505082825f015f82825461063a9190610dbb565b9091555050505050505050565b6001600160a01b038085165f90815260066020908152604080832093871683529281528282206001600160e01b03198616835290522061068890338361072b565b50505050565b6001600160a01b0381165f9081526008602052604090206103f190333461098e565b6001600160a01b0383165f9081526007602090815260408083206001600160e01b031986168452909152902061049190338361072b565b6001600160a01b0382165f90815260086020526040902061041690338361072b565b6001600160a01b0381165f9081526004602052604090206103f190333461098e565b5f3a116107a05760405162461bcd60e51b815260206004820152603c60248201527f5769746864726177616c7320617265206e6f7420737570706f7274656420746860448201527f726f7567682073706f6e736f726564207472616e73616374696f6e73000000006064820152608401610581565b6001600160a01b0382165f9081526002840160205260409020548111156108155760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b6064820152608401610581565b600183015483545f91906108299084610dd4565b6108339190610deb565b84549091508111156108965760405162461bcd60e51b815260206004820152602660248201527f4e6f7420656e6f75676820617661696c61626c652066756e647320746f20776960448201526574686472617760d01b6064820152608401610581565b5f836001600160a01b0316826040515f6040518083038185875af1925050503d805f81146108df576040519150601f19603f3d011682016040523d82523d5f602084013e6108e4565b606091505b50509050806109275760405162461bcd60e51b815260206004820152600f60248201526e151c985b9cd9995c8819985a5b1959608a1b6044820152606401610581565b6001600160a01b0384165f90815260028601602052604081208054859290610950908490610dbb565b9250508190555082856001015f82825461096a9190610dbb565b90915550508454829086905f90610982908490610dbb565b90915550505050505050565b80835f015f8282546109a09190610e0a565b90915550506001600160a01b0382165f908152600284016020526040812080548392906109ce908490610e0a565b9250508190555080836001015f8282546109e89190610e0a565b9091555050505050565b6001600160a01b038085165f90815260066020908152604080832093871683529281528282206001600160e01b03198616835290529081205481908311610a725750506001600160a01b038085165f90815260066020908152604080832093871683529281528282206001600160e01b0319861683529052206001610bc9565b6001600160a01b038087165f908152600560209081526040808320938916835292905220548311610aca5750506001600160a01b038085165f9081526005602090815260408083209387168352929052206001610bc9565b6001600160a01b0386165f908152600460205260409020548311610b085750506001600160a01b0384165f9081526004602052604090206001610bc9565b6001600160a01b0385165f9081526007602090815260408083206001600160e01b0319881684529091529020548311610b705750506001600160a01b0383165f9081526007602090815260408083206001600160e01b03198616845290915290206001610bc9565b6001600160a01b0385165f908152600860205260409020548311610bae5750506001600160a01b0383165f9081526008602052604090206001610bc9565b6001548311610bc257506001905080610bc9565b50600190505f5b94509492505050565b5f60208284031215610be2575f5ffd5b5035919050565b80356001600160a01b0381168114610bff575f5ffd5b919050565b5f5f60408385031215610c15575f5ffd5b610c1e83610be9565b946020939093013593505050565b5f5f60408385031215610c3d575f5ffd5b610c4683610be9565b9150610c5460208401610be9565b90509250929050565b80356001600160e01b031981168114610bff575f5ffd5b5f5f5f5f60808587031215610c87575f5ffd5b610c9085610be9565b9350610c9e60208601610be9565b9250610cac60408601610c5d565b9396929550929360600135925050565b5f5f5f60608486031215610cce575f5ffd5b610cd784610be9565b9250610ce560208501610be9565b929592945050506040919091013590565b5f60208284031215610d06575f5ffd5b610d0f82610be9565b9392505050565b5f5f60408385031215610d27575f5ffd5b610d3083610be9565b9150610c5460208401610c5d565b5f5f5f60608486031215610d50575f5ffd5b610d5984610be9565b9250610d6760208501610be9565b9150610d7560408501610c5d565b90509250925092565b5f5f5f60608486031215610d90575f5ffd5b610d9984610be9565b9250610ce560208501610c5d565b634e487b7160e01b5f52601160045260245ffd5b81810381811115610dce57610dce610da7565b92915050565b8082028115828204841417610dce57610dce610da7565b5f82610e0557634e487b7160e01b5f52601260045260245ffd5b500490565b80820180821115610dce57610dce610da756fea26469706673582212202e85733bb495143f5a3a56b984e0d5b53520c2e8bf3a960422ccfe88a0c75a9d64736f6c634300081b0033",
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
