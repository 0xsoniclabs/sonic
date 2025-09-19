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
	ABI: "[{\"inputs\":[{\"internalType\":\"contractFeeBurner\",\"name\":\"feeBurner_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"locked\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"callSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"locked\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"contractSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"locked\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorship\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"locked\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"isCovered\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"serviceSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"locked\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"sponsorAccount\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorContract\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sponsorGlobal\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"}],\"name\":\"sponsorService\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"sponsorUser\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userSponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"locked\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawAccountSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawCallSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawContractSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawGlobalSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"functionSelector\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawServiceSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdrawUserSponsorship\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50604051610eb2380380610eb2833981016040819052602b91604e565b5f80546001600160a01b0319166001600160a01b03929092169190911790556079565b5f60208284031215605d575f5ffd5b81516001600160a01b03811681146072575f5ffd5b9392505050565b610e2c806100865f395ff3fe60806040526004361061011b575f3560e01c8063944557d61161009d578063cc77aec811610062578063cc77aec814610379578063daf21aa3146103b3578063e32213bb146103c6578063f1cdef06146103e5578063f8117aa814610404575f5ffd5b8063944557d61461026657806399bc8bee146102855780639c31691e146102a4578063aae83110146102ef578063b5dfce0714610334575f5ffd5b80633f49695a116100e35780633f49695a146101df5780633ff8b209146101fe578063533c23c6146102385780636c2f07861461024b5780637d2e55641461025e575f5ffd5b806302f8297c1461011f5780630c617f961461015d578063174299631461017e5780632cc051571461019d57806336a656a7146101b0575b5f5ffd5b34801561012a575f5ffd5b5060015460025460035461013d92919083565b604080519384526020840192909252908201526060015b60405180910390f35b348015610168575f5ffd5b5061017c610177366004610bab565b610417565b005b348015610189575f5ffd5b5061017c610198366004610bdd565b610426565b61017c6101ab366004610c05565b61044c565b3480156101bb575f5ffd5b506101cf6101ca366004610c4d565b61047b565b6040519015158152602001610154565b3480156101ea575f5ffd5b5061017c6101f9366004610c95565b610598565b348015610209575f5ffd5b5061013d610218366004610ccf565b60056020525f908152604090208054600182015460029092015490919083565b61017c610246366004610cef565b6105cc565b61017c610259366004610d17565b610603565b61017c610644565b348015610271575f5ffd5b5061017c610280366004610c4d565b610652565b348015610290575f5ffd5b5061017c61029f366004610c4d565b6108d7565b3480156102af575f5ffd5b5061013d6102be366004610d17565b600760209081525f938452604080852082529284528284209052825290208054600182015460029092015490919083565b3480156102fa575f5ffd5b5061013d610309366004610c05565b600660209081525f928352604080842090915290825290208054600182015460029092015490919083565b34801561033f575f5ffd5b5061013d61034e366004610cef565b600860209081525f928352604080842090915290825290208054600182015460029092015490919083565b348015610384575f5ffd5b5061013d610393366004610ccf565b60096020525f908152604090208054600182015460029092015490919083565b61017c6103c1366004610ccf565b610918565b3480156103d1575f5ffd5b5061017c6103e0366004610d57565b61093a565b3480156103f0575f5ffd5b5061017c6103ff366004610bdd565b610971565b61017c610412366004610ccf565b610993565b610423600133836109b5565b50565b6001600160a01b0382165f9081526005602052604090206104489033836109b5565b5050565b6001600160a01b038083165f908152600660209081526040808320938516835292905220610448903334610b47565b6001600160a01b038085165f90815260076020908152604080832093871683529281528282206001600160e01b03198616835290529081205482116104c257506001610590565b6001600160a01b038086165f9081526006602090815260408083209388168352929052205482116104f557506001610590565b6001600160a01b0385165f90815260056020526040902054821161051b57506001610590565b6001600160a01b0384165f9081526008602090815260408083206001600160e01b031987168452909152902054821161055657506001610590565b6001600160a01b0384165f90815260096020526040902054821161057c57506001610590565b600154821161058d57506001610590565b505f5b949350505050565b6001600160a01b038084165f9081526006602090815260408083209386168352929052206105c79033836109b5565b505050565b6001600160a01b0382165f9081526008602090815260408083206001600160e01b0319851684529091529020610448903334610b47565b6001600160a01b038084165f90815260076020908152604080832093861683529281528282206001600160e01b0319851683529052206105c7903334610b47565b61065060013334610b47565b565b331561065c575f5ffd5b6106688484848461047b565b610670575f5ffd5b5f805460408051630214284360e61b815290516001600160a01b039092169263850a10c0928592600480820193929182900301818588803b1580156106b3575f5ffd5b505af11580156106c5573d5f5f3e3d5ffd5b505050506001600160a01b038581165f90815260076020908152604080832093881683529281528282206001600160e01b031987168352905220548211905061075b576001600160a01b038085165f90815260076020908152604080832093871683529281528282206001600160e01b03198616835290529081208054839290610750908490610d94565b909155506108d19050565b6001600160a01b038085165f9081526006602090815260408083209387168352929052205481116107bc576001600160a01b038085165f90815260066020908152604080832093871683529290529081208054839290610750908490610d94565b6001600160a01b0384165f908152600560205260409020548111610801576001600160a01b0384165f9081526005602052604081208054839290610750908490610d94565b6001600160a01b0383165f9081526008602090815260408083206001600160e01b0319861684529091529020548111610870576001600160a01b0383165f9081526008602090815260408083206001600160e01b03198616845290915281208054839290610750908490610d94565b6001600160a01b0383165f9081526009602052604090205481116108b5576001600160a01b0383165f9081526009602052604081208054839290610750908490610d94565b60015481116108d1578060015f015f8282546107509190610d94565b50505050565b6001600160a01b038085165f90815260076020908152604080832093871683529281528282206001600160e01b0319861683529052206108d19033836109b5565b6001600160a01b0381165f908152600960205260409020610423903334610b47565b6001600160a01b0383165f9081526008602090815260408083206001600160e01b03198616845290915290206105c79033836109b5565b6001600160a01b0382165f9081526009602052604090206104489033836109b5565b6001600160a01b0381165f908152600560205260409020610423903334610b47565b6001600160a01b0382165f908152600384016020526040902054811115610a2f5760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b60648201526084015b60405180910390fd5b600283015483545f9190610a439084610dad565b610a4d9190610dc4565b90505f836001600160a01b0316826040515f6040518083038185875af1925050503d805f8114610a98576040519150601f19603f3d011682016040523d82523d5f602084013e610a9d565b606091505b5050905080610ae05760405162461bcd60e51b815260206004820152600f60248201526e151c985b9cd9995c8819985a5b1959608a1b6044820152606401610a26565b6001600160a01b0384165f90815260038601602052604081208054859290610b09908490610d94565b9250508190555082856002015f828254610b239190610d94565b90915550508454829086905f90610b3b908490610d94565b90915550505050505050565b80835f015f828254610b599190610de3565b90915550506001600160a01b0382165f90815260038401602052604081208054839290610b87908490610de3565b9250508190555080836002015f828254610ba19190610de3565b9091555050505050565b5f60208284031215610bbb575f5ffd5b5035919050565b80356001600160a01b0381168114610bd8575f5ffd5b919050565b5f5f60408385031215610bee575f5ffd5b610bf783610bc2565b946020939093013593505050565b5f5f60408385031215610c16575f5ffd5b610c1f83610bc2565b9150610c2d60208401610bc2565b90509250929050565b80356001600160e01b031981168114610bd8575f5ffd5b5f5f5f5f60808587031215610c60575f5ffd5b610c6985610bc2565b9350610c7760208601610bc2565b9250610c8560408601610c36565b9396929550929360600135925050565b5f5f5f60608486031215610ca7575f5ffd5b610cb084610bc2565b9250610cbe60208501610bc2565b929592945050506040919091013590565b5f60208284031215610cdf575f5ffd5b610ce882610bc2565b9392505050565b5f5f60408385031215610d00575f5ffd5b610d0983610bc2565b9150610c2d60208401610c36565b5f5f5f60608486031215610d29575f5ffd5b610d3284610bc2565b9250610d4060208501610bc2565b9150610d4e60408501610c36565b90509250925092565b5f5f5f60608486031215610d69575f5ffd5b610d7284610bc2565b9250610cbe60208501610c36565b634e487b7160e01b5f52601160045260245ffd5b81810381811115610da757610da7610d80565b92915050565b8082028115828204841417610da757610da7610d80565b5f82610dde57634e487b7160e01b5f52601260045260245ffd5b500490565b80820180821115610da757610da7610d8056fea26469706673582212204414c63da8d831fc70634421eeec4776926efac81a2054bf072a865de0f05c8364736f6c634300081b0033",
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
// Solidity: function accountSponsorships(address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCaller) AccountSponsorships(opts *bind.CallOpts, arg0 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "accountSponsorships", arg0)

	outstruct := new(struct {
		Funds              *big.Int
		Locked             *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Locked = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// AccountSponsorships is a free data retrieval call binding the contract method 0x3ff8b209.
//
// Solidity: function accountSponsorships(address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistrySession) AccountSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.AccountSponsorships(&_Registry.CallOpts, arg0)
}

// AccountSponsorships is a free data retrieval call binding the contract method 0x3ff8b209.
//
// Solidity: function accountSponsorships(address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCallerSession) AccountSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.AccountSponsorships(&_Registry.CallOpts, arg0)
}

// CallSponsorships is a free data retrieval call binding the contract method 0x9c31691e.
//
// Solidity: function callSponsorships(address , address , bytes4 ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCaller) CallSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 [4]byte) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "callSponsorships", arg0, arg1, arg2)

	outstruct := new(struct {
		Funds              *big.Int
		Locked             *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Locked = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// CallSponsorships is a free data retrieval call binding the contract method 0x9c31691e.
//
// Solidity: function callSponsorships(address , address , bytes4 ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistrySession) CallSponsorships(arg0 common.Address, arg1 common.Address, arg2 [4]byte) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.CallSponsorships(&_Registry.CallOpts, arg0, arg1, arg2)
}

// CallSponsorships is a free data retrieval call binding the contract method 0x9c31691e.
//
// Solidity: function callSponsorships(address , address , bytes4 ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCallerSession) CallSponsorships(arg0 common.Address, arg1 common.Address, arg2 [4]byte) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.CallSponsorships(&_Registry.CallOpts, arg0, arg1, arg2)
}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCaller) ContractSponsorships(opts *bind.CallOpts, arg0 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "contractSponsorships", arg0)

	outstruct := new(struct {
		Funds              *big.Int
		Locked             *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Locked = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistrySession) ContractSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ContractSponsorships(&_Registry.CallOpts, arg0)
}

// ContractSponsorships is a free data retrieval call binding the contract method 0xcc77aec8.
//
// Solidity: function contractSponsorships(address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCallerSession) ContractSponsorships(arg0 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ContractSponsorships(&_Registry.CallOpts, arg0)
}

// GlobalSponsorship is a free data retrieval call binding the contract method 0x02f8297c.
//
// Solidity: function globalSponsorship() view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCaller) GlobalSponsorship(opts *bind.CallOpts) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "globalSponsorship")

	outstruct := new(struct {
		Funds              *big.Int
		Locked             *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Locked = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GlobalSponsorship is a free data retrieval call binding the contract method 0x02f8297c.
//
// Solidity: function globalSponsorship() view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistrySession) GlobalSponsorship() (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.GlobalSponsorship(&_Registry.CallOpts)
}

// GlobalSponsorship is a free data retrieval call binding the contract method 0x02f8297c.
//
// Solidity: function globalSponsorship() view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCallerSession) GlobalSponsorship() (struct {
	Funds              *big.Int
	Locked             *big.Int
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
// Solidity: function serviceSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCaller) ServiceSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "serviceSponsorships", arg0, arg1)

	outstruct := new(struct {
		Funds              *big.Int
		Locked             *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Locked = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ServiceSponsorships is a free data retrieval call binding the contract method 0xb5dfce07.
//
// Solidity: function serviceSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistrySession) ServiceSponsorships(arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ServiceSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// ServiceSponsorships is a free data retrieval call binding the contract method 0xb5dfce07.
//
// Solidity: function serviceSponsorships(address , bytes4 ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCallerSession) ServiceSponsorships(arg0 common.Address, arg1 [4]byte) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.ServiceSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCaller) UserSponsorships(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "userSponsorships", arg0, arg1)

	outstruct := new(struct {
		Funds              *big.Int
		Locked             *big.Int
		TotalContributions *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Funds = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Locked = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalContributions = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistrySession) UserSponsorships(arg0 common.Address, arg1 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
	TotalContributions *big.Int
}, error) {
	return _Registry.Contract.UserSponsorships(&_Registry.CallOpts, arg0, arg1)
}

// UserSponsorships is a free data retrieval call binding the contract method 0xaae83110.
//
// Solidity: function userSponsorships(address , address ) view returns(uint256 funds, uint256 locked, uint256 totalContributions)
func (_Registry *RegistryCallerSession) UserSponsorships(arg0 common.Address, arg1 common.Address) (struct {
	Funds              *big.Int
	Locked             *big.Int
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
