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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"approvalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"bootstrapSponsorshipFund\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"name\":\"callSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"contractSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadCharge\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"globalSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b50610cd48061001c5f395ff3fe6080604052600436106100a5575f3560e01c8063779a43ac11610062578063779a43ac1461019f5780639ec88e99146101b3578063a5dc4518146101c6578063b9ed9f26146101e5578063e327d1ac14610204578063fecb2bc314610223575f80fd5b8063040cf020146100a95780630ad1fcfc146100ca578063399f59ca146101055780634b5c54c01461013257806351ee41a01461016157806363f2cdca14610180575b5f80fd5b3480156100b4575f80fd5b506100c86100c33660046109ec565b61026a565b005b3480156100d5575f80fd5b506100e96100e4366004610a68565b610477565b6040805192151583526020830191909152015b60405180910390f35b348015610110575f80fd5b5061012461011f366004610ac9565b610576565b6040519081526020016100fc565b34801561013d575f80fd5b506101466106b9565b604080519384526020840192909252908201526060016100fc565b34801561016c575f80fd5b506100e961017b366004610b48565b6106e1565b34801561018b575f80fd5b506100e961019a366004610b6a565b61072c565b3480156101aa575f80fd5b506100e9610759565b6100c86101c1366004610b6a565b610792565b3480156101d1575f80fd5b506100e96101e0366004610a68565b6107fb565b3480156101f0575f80fd5b506100c86101ff3660046109ec565b6108a1565b34801561020f575f80fd5b506100e961021e366004610b48565b6109bc565b34801561022e575f80fd5b5061025561023d366004610b6a565b5f602081905290815260409020805460019091015482565b604080519283526020830191909152016100fc565b5f3a116102e45760405162461bcd60e51b815260206004820152603c60248201527f5769746864726177616c7320617265206e6f7420737570706f7274656420746860448201527f726f7567682073706f6e736f726564207472616e73616374696f6e730000000060648201526084015b60405180910390fd5b5f82815260208181526040808320338085526002820190935292205483111561035b5760405162461bcd60e51b8152602060048201526024808201527f4e6f7420656e6f75676820636f6e747269627574696f6e7320746f20776974686044820152636472617760e01b60648201526084016102db565b600182015482545f919061036f9086610b95565b6103799190610bb2565b83549091508111156103dc5760405162461bcd60e51b815260206004820152602660248201527f4e6f7420656e6f75676820617661696c61626c652066756e647320746f20776960448201526574686472617760d01b60648201526084016102db565b6001600160a01b0382165f90815260028401602052604081208054869290610405908490610bd1565b9250508190555083836001015f82825461041f9190610bd1565b90915550508254819084905f90610437908490610bd1565b90915550506040516001600160a01b0383169082156108fc029083905f818181858888f1935050505015801561046f573d5f803e3d5ffd5b505050505050565b5f806001600160a01b0385161580610490575060448314155b1561049f57505f90508061056d565b5f6104ad6004828688610be4565b6104b691610c0b565b905063095ea7b360e01b6001600160e01b03198216146104dc57505f915081905061056d565b5f806104eb866004818a610be4565b8101906104f89190610c3b565b91509150600181101561051457505f935083925061056d915050565b604051606160f81b60208201526001600160601b031960608b811b821660218401528a811b8216603584015284901b166049820152600190605d0160405160208183030381529060405280519060200120945094505050505b94509492505050565b5f8061058489898787610477565b925090508080156105a257505f828152602081905260409020548311155b156105ad57506106ae565b6105b9898987876107fb565b925090508080156105d757505f828152602081905260409020548311155b156105e257506106ae565b6105eb896106e1565b9250905080801561060957505f828152602081905260409020548311155b1561061457506106ae565b61061d886109bc565b9250905080801561063b57505f828152602081905260409020548311155b1561064657506106ae565b61064f8661072c565b9250905080801561066d57505f828152602081905260409020548311155b1561067857506106ae565b610680610759565b9250905080801561069e57505f828152602081905260409020548311155b156106a957506106ae565b505f90505b979650505050505050565b620186a061ea605f61c350806106cf8486610c65565b6106d99190610c65565b915050909192565b604051606160f81b60208201526001600160601b0319606083901b1660218201525f9081906001906035015b6040516020818303038152906040528051906020012091509150915091565b5f80600383101561074f57604051603160f91b602082015260019060210161070d565b505f928392509050565b5f80600160405160200161077490606760f81b815260010190565b60405160208183030381529060405280519060200120915091509091565b5f8181526020819052604081208054909134918391906107b3908490610c65565b9091555050335f908152600282016020526040812080543492906107d8908490610c65565b9250508190555034816001015f8282546107f29190610c65565b90915550505050565b5f806001600160a01b03851615806108135750600483105b1561082257505f90508061056d565b5f6108306004828688610be4565b61083991610c0b565b604051606360f81b60208201526001600160601b031960608a811b8216602184015289901b1660358201526001600160e01b031982166049820152909150600190604d0160405160208183030381529060405280519060200120925092505094509492505050565b33156108ab575f80fd5b816108f85760405162461bcd60e51b815260206004820152601a60248201527f4e6f2073706f6e736f72736869702066756e642063686f73656e00000000000060448201526064016102db565b5f82815260208190526040902080548211156109495760405162461bcd60e51b815260206004820152601060248201526f4e6f7420656e6f7567682066756e647360801b60448201526064016102db565b637e007d6760811b6001600160a01b031663850a10c0836040518263ffffffff1660e01b81526004015f604051808303818588803b158015610989575f80fd5b505af115801561099b573d5f803e3d5ffd5b505050505081815f015f8282546109b29190610bd1565b9091555050505050565b604051606360f81b60208201526001600160601b0319606083901b1660218201525f90819060019060350161070d565b5f80604083850312156109fd575f80fd5b50508035926020909101359150565b6001600160a01b0381168114610a20575f80fd5b50565b5f8083601f840112610a33575f80fd5b50813567ffffffffffffffff811115610a4a575f80fd5b602083019150836020828501011115610a61575f80fd5b9250929050565b5f805f8060608587031215610a7b575f80fd5b8435610a8681610a0c565b93506020850135610a9681610a0c565b9250604085013567ffffffffffffffff811115610ab1575f80fd5b610abd87828801610a23565b95989497509550505050565b5f805f805f805f60c0888a031215610adf575f80fd5b8735610aea81610a0c565b96506020880135610afa81610a0c565b95506040880135945060608801359350608088013567ffffffffffffffff811115610b23575f80fd5b610b2f8a828b01610a23565b989b979a5095989497959660a090950135949350505050565b5f60208284031215610b58575f80fd5b8135610b6381610a0c565b9392505050565b5f60208284031215610b7a575f80fd5b5035919050565b634e487b7160e01b5f52601160045260245ffd5b8082028115828204841417610bac57610bac610b81565b92915050565b5f82610bcc57634e487b7160e01b5f52601260045260245ffd5b500490565b81810381811115610bac57610bac610b81565b5f8085851115610bf2575f80fd5b83861115610bfe575f80fd5b5050820193919092039150565b6001600160e01b03198135818116916004851015610c335780818660040360031b1b83161692505b505092915050565b5f8060408385031215610c4c575f80fd5b8235610c5781610a0c565b946020939093013593505050565b80820180821115610bac57610bac610b8156fea2646970667358221220e35d1aab0674e81a57b70b67b8bb12aa870db3180e805fb970993f00dd02b3e764736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
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

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_Registry *RegistryCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	var out []interface{}
	err := _Registry.contract.Call(opts, &out, "getGasConfig")

	outstruct := new(struct {
		ChooseFundLimit *big.Int
		DeductFeesLimit *big.Int
		OverheadCharge  *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ChooseFundLimit = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.DeductFeesLimit = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.OverheadCharge = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_Registry *RegistrySession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	return _Registry.Contract.GetGasConfig(&_Registry.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_Registry *RegistryCallerSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
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
