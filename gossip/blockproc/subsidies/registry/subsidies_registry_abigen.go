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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"approvalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"bootstrapSponsorshipFund\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"callSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"contractSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimitForChooseFund\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForDeductFees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForTrack\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForFundBackedSponsorships\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForNetworkSponsorshipsWithTracking\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b50610d6f8061001c5f395ff3fe6080604052600436106100bf575f3560e01c8063779a43ac1161007c578063b9ed9f2611610057578063b9ed9f2614610213578063bf70eb1514610232578063e327d1ac14610251578063fecb2bc314610270575f80fd5b8063779a43ac146101cd5780639ec88e99146101e1578063a5dc4518146101f4575f80fd5b8063040cf020146100c35780630ad1fcfc146100e4578063399f59ca1461011f5780634b5c54c01461015357806351ee41a01461018f57806363f2cdca146101ae575b5f80fd5b3480156100ce575f80fd5b506100e26100dd366004610a87565b6102a2565b005b3480156100ef575f80fd5b506101036100fe366004610b03565b6104af565b6040805192151583526020830191909152015b60405180910390f35b34801561012a575f80fd5b5061013e610139366004610b64565b6105ae565b60408051928352602083019190915201610116565b34801561015e575f80fd5b50610167610728565b604080519586526020860194909452928401919091526060830152608082015260a001610116565b34801561019a575f80fd5b506101036101a9366004610be3565b61076e565b3480156101b9575f80fd5b506101036101c8366004610c05565b6107b9565b3480156101d8575f80fd5b506101036107e6565b6100e26101ef366004610c05565b61081f565b3480156101ff575f80fd5b5061010361020e366004610b03565b610888565b34801561021e575f80fd5b506100e261022d366004610a87565b61092e565b34801561023d575f80fd5b506100e261024c366004610a87565b610a49565b34801561025c575f80fd5b5061010361026b366004610be3565b610a57565b34801561027b575f80fd5b5061013e61028a366004610c05565b5f602081905290815260409020805460019091015482565b5f3a1161031c5760405162461bcd60e51b815260206004820152603c60248201527f5769746864726177616c7320617265206e6f7420737570706f7274656420746860448201527f726f7567682073706f6e736f726564207472616e73616374696f6e730000000060648201526084015b60405180910390fd5b5f8281526020818152604080832033808552600282019093529220548311156103935760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b6064820152608401610313565b600182015482545f91906103a79086610c30565b6103b19190610c4d565b83549091508111156104145760405162461bcd60e51b815260206004820152602660248201527f4e6f7420656e6f75676820617661696c61626c652066756e647320746f20776960448201526574686472617760d01b6064820152608401610313565b6001600160a01b0382165f9081526002840160205260408120805486929061043d908490610c6c565b9250508190555083836001015f8282546104579190610c6c565b90915550508254819084905f9061046f908490610c6c565b90915550506040516001600160a01b0383169082156108fc029083905f818181858888f193505050501580156104a7573d5f803e3d5ffd5b505050505050565b5f806001600160a01b03851615806104c8575060448314155b156104d757505f9050806105a5565b5f6104e56004828688610c7f565b6104ee91610ca6565b905063095ea7b360e01b6001600160e01b031982161461051457505f91508190506105a5565b5f80610523866004818a610c7f565b8101906105309190610cd6565b91509150600181101561054c57505f93508392506105a5915050565b604051606160f81b60208201526001600160601b031960608b811b821660218401528a811b8216603584015284901b166049820152600190605d0160405160208183030381529060405280519060200120945094505050505b94509492505050565b5f805f806105be8b8b89896104af565b90925090508180156105dd57505f818152602081905260409020548511155b156105ef5760019350915061071c9050565b6105fb8b8b8989610888565b909250905081801561061a57505f818152602081905260409020548511155b1561062c5760019350915061071c9050565b6106358b61076e565b909250905081801561065457505f818152602081905260409020548511155b156106665760019350915061071c9050565b61066f8a610a57565b909250905081801561068e57505f818152602081905260409020548511155b156106a05760019350915061071c9050565b6106a9886107b9565b90925090508180156106c857505f818152602081905260409020548511155b156106da5760019350915061071c9050565b6106e26107e6565b909250905081801561070157505f818152602081905260409020548511155b156107135760019350915061071c9050565b505f9250829150505b97509795505050505050565b620186a061ea60620138805f8061c350806107438688610d00565b61074d9190610d00565b92508061075a8588610d00565b6107649190610d00565b9150509091929394565b604051606160f81b60208201526001600160601b0319606083901b1660218201525f9081906001906035015b6040516020818303038152906040528051906020012091509150915091565b5f8060038310156107dc57604051603160f91b602082015260019060210161079a565b505f928392509050565b5f80600160405160200161080190606760f81b815260010190565b60405160208183030381529060405280519060200120915091509091565b5f818152602081905260408120805490913491839190610840908490610d00565b9091555050335f90815260028201602052604081208054349290610865908490610d00565b9250508190555034816001015f82825461087f9190610d00565b90915550505050565b5f806001600160a01b03851615806108a05750600483105b156108af57505f9050806105a5565b5f6108bd6004828688610c7f565b6108c691610ca6565b604051606360f81b60208201526001600160601b031960608a811b8216602184015289901b1660358201526001600160e01b031982166049820152909150600190604d0160405160208183030381529060405280519060200120925092505094509492505050565b3315610938575f80fd5b816109855760405162461bcd60e51b815260206004820152601a60248201527f4e6f2073706f6e736f72736869702066756e642063686f73656e0000000000006044820152606401610313565b5f82815260208190526040902080548211156109d65760405162461bcd60e51b815260206004820152601060248201526f4e6f7420656e6f7567682066756e647360801b6044820152606401610313565b637e007d6760811b6001600160a01b031663850a10c0836040518263ffffffff1660e01b81526004015f604051808303818588803b158015610a16575f80fd5b505af1158015610a28573d5f803e3d5ffd5b505050505081815f015f828254610a3f9190610c6c565b9091555050505050565b3315610a53575f80fd5b5050565b604051606360f81b60208201526001600160601b0319606083901b1660218201525f90819060019060350161079a565b5f8060408385031215610a98575f80fd5b50508035926020909101359150565b6001600160a01b0381168114610abb575f80fd5b50565b5f8083601f840112610ace575f80fd5b50813567ffffffffffffffff811115610ae5575f80fd5b602083019150836020828501011115610afc575f80fd5b9250929050565b5f805f8060608587031215610b16575f80fd5b8435610b2181610aa7565b93506020850135610b3181610aa7565b9250604085013567ffffffffffffffff811115610b4c575f80fd5b610b5887828801610abe565b95989497509550505050565b5f805f805f805f60c0888a031215610b7a575f80fd5b8735610b8581610aa7565b96506020880135610b9581610aa7565b95506040880135945060608801359350608088013567ffffffffffffffff811115610bbe575f80fd5b610bca8a828b01610abe565b989b979a5095989497959660a090950135949350505050565b5f60208284031215610bf3575f80fd5b8135610bfe81610aa7565b9392505050565b5f60208284031215610c15575f80fd5b5035919050565b634e487b7160e01b5f52601160045260245ffd5b8082028115828204841417610c4757610c47610c1c565b92915050565b5f82610c6757634e487b7160e01b5f52601260045260245ffd5b500490565b81810381811115610c4757610c47610c1c565b5f8085851115610c8d575f80fd5b83861115610c99575f80fd5b5050820193919092039150565b6001600160e01b03198135818116916004851015610cce5780818660040360031b1b83161692505b505092915050565b5f8060408385031215610ce7575f80fd5b8235610cf281610aa7565b946020939093013593505050565b80820180821115610c4757610c47610c1c56fea26469706673582212203ba1354181a41b7e6c649ef4bc0d1071fdaf3a4abd14a88ebac9f2e825ca3ab964736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
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
// Solidity: function chooseFund(address from, address to, uint256 , uint256 nonce, bytes callData, uint256 fee) view returns(uint256 mode, bytes32 payload)
func (_Registry *RegistryCaller) ChooseFund(opts *bind.CallOpts, from common.Address, to common.Address, arg2 *big.Int, nonce *big.Int, callData []byte, fee *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "chooseFund", from, to, arg2, nonce, callData, fee)

	outstruct := new(struct {
		Mode    *big.Int
		Payload [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Mode = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Payload = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address to, uint256 , uint256 nonce, bytes callData, uint256 fee) view returns(uint256 mode, bytes32 payload)
func (_Registry *RegistrySession) ChooseFund(from common.Address, to common.Address, arg2 *big.Int, nonce *big.Int, callData []byte, fee *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _Registry.Contract.ChooseFund(&_Registry.CallOpts, from, to, arg2, nonce, callData, fee)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address to, uint256 , uint256 nonce, bytes callData, uint256 fee) view returns(uint256 mode, bytes32 payload)
func (_Registry *RegistryCallerSession) ChooseFund(from common.Address, to common.Address, arg2 *big.Int, nonce *big.Int, callData []byte, fee *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
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

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_Registry *RegistryCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getGasConfig")

	outstruct := new(struct {
		GasLimitForChooseFund                            *big.Int
		GasLimitForDeductFees                            *big.Int
		GasLimitForTrack                                 *big.Int
		OverheadChargeForFundBackedSponsorships          *big.Int
		OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.GasLimitForChooseFund = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.GasLimitForDeductFees = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.GasLimitForTrack = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.OverheadChargeForFundBackedSponsorships = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.OverheadChargeForNetworkSponsorshipsWithTracking = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_Registry *RegistrySession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _Registry.Contract.GetGasConfig(&_Registry.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_Registry *RegistryCallerSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _Registry.Contract.GetGasConfig(&_Registry.CallOpts)
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

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) view returns()
func (_Registry *RegistryCaller) Track(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "track", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) view returns()
func (_Registry *RegistrySession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _Registry.Contract.Track(&_Registry.CallOpts, arg0, arg1)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) view returns()
func (_Registry *RegistryCallerSession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _Registry.Contract.Track(&_Registry.CallOpts, arg0, arg1)
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
