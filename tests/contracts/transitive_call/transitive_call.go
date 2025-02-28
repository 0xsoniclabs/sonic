// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package transitive_call

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

// TransitiveCallMetaData contains all meta data concerning the TransitiveCall contract.
var TransitiveCallMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"getCount\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"leafCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"call_chain\",\"type\":\"address[]\"}],\"name\":\"transitiveCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60806040525f5f553480156011575f5ffd5b506107278061001f5f395ff3fe608060405260043610610033575f3560e01c806347d1699e14610037578063a87d942c14610053578063e6fbb2f71461007d575b5f5ffd5b610051600480360381019061004c919061039d565b610087565b005b34801561005e575f5ffd5b50610067610312565b6040516100749190610400565b60405180910390f35b61008561031a565b005b5f82829050116100cc576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016100c390610473565b60405180910390fd5b60015f5f8282546100dd91906104be565b925050819055506001828290500361017d575f82825f818110610103576101026104ff565b5b90506020020160208101906101189190610586565b90508073ffffffffffffffffffffffffffffffffffffffff1663e6fbb2f7346040518263ffffffff1660e01b81526004015f604051808303818588803b158015610160575f5ffd5b505af1158015610172573d5f5f3e3d5ffd5b50505050505061030e565b5f82825f818110610191576101906104ff565b5b90506020020160208101906101a69190610586565b90505f6001848490506101b991906105ba565b67ffffffffffffffff8111156101d2576101d16105ed565b5b6040519080825280602002602001820160405280156102005781602001602082028036833780820191505090505b5090505f600190505b848490508110156102a257848482818110610227576102266104ff565b5b905060200201602081019061023c9190610586565b8260018361024a91906105ba565b8151811061025b5761025a6104ff565b5b602002602001019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250508080600101915050610209565b508173ffffffffffffffffffffffffffffffffffffffff166347d1699e34836040518363ffffffff1660e01b81526004016102dd91906106d1565b5f604051808303818588803b1580156102f4575f5ffd5b505af1158015610306573d5f5f3e3d5ffd5b505050505050505b5050565b5f5f54905090565b60015f5f82825461032b91906104be565b92505081905550565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261035d5761035c61033c565b5b8235905067ffffffffffffffff81111561037a57610379610340565b5b60208301915083602082028301111561039657610395610344565b5b9250929050565b5f5f602083850312156103b3576103b2610334565b5b5f83013567ffffffffffffffff8111156103d0576103cf610338565b5b6103dc85828601610348565b92509250509250929050565b5f819050919050565b6103fa816103e8565b82525050565b5f6020820190506104135f8301846103f1565b92915050565b5f82825260208201905092915050565b7f63616c6c5f636861696e20697320656d707479000000000000000000000000005f82015250565b5f61045d601383610419565b915061046882610429565b602082019050919050565b5f6020820190508181035f83015261048a81610451565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6104c8826103e8565b91506104d3836103e8565b92508282019050828112155f8312168382125f8412151617156104f9576104f8610491565b5b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6105558261052c565b9050919050565b6105658161054b565b811461056f575f5ffd5b50565b5f813590506105808161055c565b92915050565b5f6020828403121561059b5761059a610334565b5b5f6105a884828501610572565b91505092915050565b5f819050919050565b5f6105c4826105b1565b91506105cf836105b1565b92508282039050818111156105e7576105e6610491565b5b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b61064c8161054b565b82525050565b5f61065d8383610643565b60208301905092915050565b5f602082019050919050565b5f61067f8261061a565b6106898185610624565b935061069483610634565b805f5b838110156106c45781516106ab8882610652565b97506106b683610669565b925050600181019050610697565b5085935050505092915050565b5f6020820190508181035f8301526106e98184610675565b90509291505056fea2646970667358221220ce9b77bf2ee22f160af75f8c00fc4bd74dbab7fd47e918fc0bcf8a6fa269696164736f6c634300081c0033",
}

// TransitiveCallABI is the input ABI used to generate the binding from.
// Deprecated: Use TransitiveCallMetaData.ABI instead.
var TransitiveCallABI = TransitiveCallMetaData.ABI

// TransitiveCallBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TransitiveCallMetaData.Bin instead.
var TransitiveCallBin = TransitiveCallMetaData.Bin

// DeployTransitiveCall deploys a new Ethereum contract, binding an instance of TransitiveCall to it.
func DeployTransitiveCall(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TransitiveCall, error) {
	parsed, err := TransitiveCallMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TransitiveCallBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TransitiveCall{TransitiveCallCaller: TransitiveCallCaller{contract: contract}, TransitiveCallTransactor: TransitiveCallTransactor{contract: contract}, TransitiveCallFilterer: TransitiveCallFilterer{contract: contract}}, nil
}

// TransitiveCall is an auto generated Go binding around an Ethereum contract.
type TransitiveCall struct {
	TransitiveCallCaller     // Read-only binding to the contract
	TransitiveCallTransactor // Write-only binding to the contract
	TransitiveCallFilterer   // Log filterer for contract events
}

// TransitiveCallCaller is an auto generated read-only Go binding around an Ethereum contract.
type TransitiveCallCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransitiveCallTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TransitiveCallTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransitiveCallFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TransitiveCallFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransitiveCallSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TransitiveCallSession struct {
	Contract     *TransitiveCall   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TransitiveCallCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TransitiveCallCallerSession struct {
	Contract *TransitiveCallCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// TransitiveCallTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TransitiveCallTransactorSession struct {
	Contract     *TransitiveCallTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// TransitiveCallRaw is an auto generated low-level Go binding around an Ethereum contract.
type TransitiveCallRaw struct {
	Contract *TransitiveCall // Generic contract binding to access the raw methods on
}

// TransitiveCallCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TransitiveCallCallerRaw struct {
	Contract *TransitiveCallCaller // Generic read-only contract binding to access the raw methods on
}

// TransitiveCallTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TransitiveCallTransactorRaw struct {
	Contract *TransitiveCallTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTransitiveCall creates a new instance of TransitiveCall, bound to a specific deployed contract.
func NewTransitiveCall(address common.Address, backend bind.ContractBackend) (*TransitiveCall, error) {
	contract, err := bindTransitiveCall(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TransitiveCall{TransitiveCallCaller: TransitiveCallCaller{contract: contract}, TransitiveCallTransactor: TransitiveCallTransactor{contract: contract}, TransitiveCallFilterer: TransitiveCallFilterer{contract: contract}}, nil
}

// NewTransitiveCallCaller creates a new read-only instance of TransitiveCall, bound to a specific deployed contract.
func NewTransitiveCallCaller(address common.Address, caller bind.ContractCaller) (*TransitiveCallCaller, error) {
	contract, err := bindTransitiveCall(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TransitiveCallCaller{contract: contract}, nil
}

// NewTransitiveCallTransactor creates a new write-only instance of TransitiveCall, bound to a specific deployed contract.
func NewTransitiveCallTransactor(address common.Address, transactor bind.ContractTransactor) (*TransitiveCallTransactor, error) {
	contract, err := bindTransitiveCall(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TransitiveCallTransactor{contract: contract}, nil
}

// NewTransitiveCallFilterer creates a new log filterer instance of TransitiveCall, bound to a specific deployed contract.
func NewTransitiveCallFilterer(address common.Address, filterer bind.ContractFilterer) (*TransitiveCallFilterer, error) {
	contract, err := bindTransitiveCall(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TransitiveCallFilterer{contract: contract}, nil
}

// bindTransitiveCall binds a generic wrapper to an already deployed contract.
func bindTransitiveCall(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TransitiveCallMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TransitiveCall *TransitiveCallRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TransitiveCall.Contract.TransitiveCallCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TransitiveCall *TransitiveCallRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TransitiveCall.Contract.TransitiveCallTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TransitiveCall *TransitiveCallRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TransitiveCall.Contract.TransitiveCallTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TransitiveCall *TransitiveCallCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TransitiveCall.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TransitiveCall *TransitiveCallTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TransitiveCall.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TransitiveCall *TransitiveCallTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TransitiveCall.Contract.contract.Transact(opts, method, params...)
}

// GetCount is a free data retrieval call binding the contract method 0xa87d942c.
//
// Solidity: function getCount() view returns(int256)
func (_TransitiveCall *TransitiveCallCaller) GetCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TransitiveCall.contract.Call(opts, &out, "getCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCount is a free data retrieval call binding the contract method 0xa87d942c.
//
// Solidity: function getCount() view returns(int256)
func (_TransitiveCall *TransitiveCallSession) GetCount() (*big.Int, error) {
	return _TransitiveCall.Contract.GetCount(&_TransitiveCall.CallOpts)
}

// GetCount is a free data retrieval call binding the contract method 0xa87d942c.
//
// Solidity: function getCount() view returns(int256)
func (_TransitiveCall *TransitiveCallCallerSession) GetCount() (*big.Int, error) {
	return _TransitiveCall.Contract.GetCount(&_TransitiveCall.CallOpts)
}

// LeafCall is a paid mutator transaction binding the contract method 0xe6fbb2f7.
//
// Solidity: function leafCall() payable returns()
func (_TransitiveCall *TransitiveCallTransactor) LeafCall(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TransitiveCall.contract.Transact(opts, "leafCall")
}

// LeafCall is a paid mutator transaction binding the contract method 0xe6fbb2f7.
//
// Solidity: function leafCall() payable returns()
func (_TransitiveCall *TransitiveCallSession) LeafCall() (*types.Transaction, error) {
	return _TransitiveCall.Contract.LeafCall(&_TransitiveCall.TransactOpts)
}

// LeafCall is a paid mutator transaction binding the contract method 0xe6fbb2f7.
//
// Solidity: function leafCall() payable returns()
func (_TransitiveCall *TransitiveCallTransactorSession) LeafCall() (*types.Transaction, error) {
	return _TransitiveCall.Contract.LeafCall(&_TransitiveCall.TransactOpts)
}

// TransitiveCall is a paid mutator transaction binding the contract method 0x47d1699e.
//
// Solidity: function transitiveCall(address[] call_chain) payable returns()
func (_TransitiveCall *TransitiveCallTransactor) TransitiveCall(opts *bind.TransactOpts, call_chain []common.Address) (*types.Transaction, error) {
	return _TransitiveCall.contract.Transact(opts, "transitiveCall", call_chain)
}

// TransitiveCall is a paid mutator transaction binding the contract method 0x47d1699e.
//
// Solidity: function transitiveCall(address[] call_chain) payable returns()
func (_TransitiveCall *TransitiveCallSession) TransitiveCall(call_chain []common.Address) (*types.Transaction, error) {
	return _TransitiveCall.Contract.TransitiveCall(&_TransitiveCall.TransactOpts, call_chain)
}

// TransitiveCall is a paid mutator transaction binding the contract method 0x47d1699e.
//
// Solidity: function transitiveCall(address[] call_chain) payable returns()
func (_TransitiveCall *TransitiveCallTransactorSession) TransitiveCall(call_chain []common.Address) (*types.Transaction, error) {
	return _TransitiveCall.Contract.TransitiveCall(&_TransitiveCall.TransactOpts, call_chain)
}
