// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package network_sponsor_tracking

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

// NetworkSponsorTrackingMetaData contains all meta data concerning the NetworkSponsorTracking contract.
var NetworkSponsorTrackingMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"trackingId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"Tracked\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"TRACKING_ID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadCharge\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"trackGasCost\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"trackingId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f80fd5b506106a78061001d5f395ff3fe608060405234801561000f575f80fd5b5060043610610055575f3560e01c8063399f59ca146100595780634b5c54c01461008a578063659ee7bf146100ab578063b9ed9f26146100c9578063bf70eb15146100e5575b5f80fd5b610073600480360381019061006e9190610336565b610101565b604051610081929190610407565b60405180910390f35b61009261011c565b6040516100a2949392919061042e565b60405180910390f35b6100b3610151565b6040516100c09190610471565b60405180910390f35b6100e360048036038101906100de91906104b4565b61015b565b005b6100ff60048036038101906100fa91906104b4565b610196565b005b5f80600363deadbeef5f1b9150915097509795505050505050565b5f805f80620186a0935061ea60925061c350838561013a919061051f565b610144919061051f565b915061ea60905090919293565b63deadbeef5f1b81565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161018d906105d2565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610204576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101fb9061063a565b60405180910390fd5b817f408d8fef8a4d6c626c249452c6039e7f00e5cff889b3ee3c9e237febce8ce668826040516102349190610658565b60405180910390a25050565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61027182610248565b9050919050565b61028181610267565b811461028b575f80fd5b50565b5f8135905061029c81610278565b92915050565b5f819050919050565b6102b4816102a2565b81146102be575f80fd5b50565b5f813590506102cf816102ab565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f8401126102f6576102f56102d5565b5b8235905067ffffffffffffffff811115610313576103126102d9565b5b60208301915083600182028301111561032f5761032e6102dd565b5b9250929050565b5f805f805f805f60c0888a03121561035157610350610240565b5b5f61035e8a828b0161028e565b975050602061036f8a828b0161028e565b96505060406103808a828b016102c1565b95505060606103918a828b016102c1565b945050608088013567ffffffffffffffff8111156103b2576103b1610244565b5b6103be8a828b016102e1565b935093505060a06103d18a828b016102c1565b91505092959891949750929550565b6103e9816102a2565b82525050565b5f819050919050565b610401816103ef565b82525050565b5f60408201905061041a5f8301856103e0565b61042760208301846103f8565b9392505050565b5f6080820190506104415f8301876103e0565b61044e60208301866103e0565b61045b60408301856103e0565b61046860608301846103e0565b95945050505050565b5f6020820190506104845f8301846103f8565b92915050565b610493816103ef565b811461049d575f80fd5b50565b5f813590506104ae8161048a565b92915050565b5f80604083850312156104ca576104c9610240565b5b5f6104d7858286016104a0565b92505060206104e8858286016102c1565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610529826102a2565b9150610534836102a2565b925082820190508082111561054c5761054b6104f2565b5b92915050565b5f82825260208201905092915050565b7f646564756374466565732073686f756c64206e6f742062652063616c6c6564205f8201527f666f72206d6f6465203300000000000000000000000000000000000000000000602082015250565b5f6105bc602a83610552565b91506105c782610562565b604082019050919050565b5f6020820190508181035f8301526105e9816105b0565b9050919050565b7f6f6e6c7920696e7465726e616c207472616e73616374696f6e730000000000005f82015250565b5f610624601a83610552565b915061062f826105f0565b602082019050919050565b5f6020820190508181035f83015261065181610618565b9050919050565b5f60208201905061066b5f8301846103e0565b9291505056fea26469706673582212207e3413b683b13b8fd7e2db3ec0cbcccd0c24b283b466787fc44a088ecb20ae7964736f6c63430008180033",
}

// NetworkSponsorTrackingABI is the input ABI used to generate the binding from.
// Deprecated: Use NetworkSponsorTrackingMetaData.ABI instead.
var NetworkSponsorTrackingABI = NetworkSponsorTrackingMetaData.ABI

// NetworkSponsorTrackingBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use NetworkSponsorTrackingMetaData.Bin instead.
var NetworkSponsorTrackingBin = NetworkSponsorTrackingMetaData.Bin

// DeployNetworkSponsorTracking deploys a new Ethereum contract, binding an instance of NetworkSponsorTracking to it.
func DeployNetworkSponsorTracking(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *NetworkSponsorTracking, error) {
	parsed, err := NetworkSponsorTrackingMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(NetworkSponsorTrackingBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NetworkSponsorTracking{NetworkSponsorTrackingCaller: NetworkSponsorTrackingCaller{contract: contract}, NetworkSponsorTrackingTransactor: NetworkSponsorTrackingTransactor{contract: contract}, NetworkSponsorTrackingFilterer: NetworkSponsorTrackingFilterer{contract: contract}}, nil
}

// NetworkSponsorTracking is an auto generated Go binding around an Ethereum contract.
type NetworkSponsorTracking struct {
	NetworkSponsorTrackingCaller     // Read-only binding to the contract
	NetworkSponsorTrackingTransactor // Write-only binding to the contract
	NetworkSponsorTrackingFilterer   // Log filterer for contract events
}

// NetworkSponsorTrackingCaller is an auto generated read-only Go binding around an Ethereum contract.
type NetworkSponsorTrackingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorTrackingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NetworkSponsorTrackingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorTrackingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NetworkSponsorTrackingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorTrackingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NetworkSponsorTrackingSession struct {
	Contract     *NetworkSponsorTracking // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// NetworkSponsorTrackingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NetworkSponsorTrackingCallerSession struct {
	Contract *NetworkSponsorTrackingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// NetworkSponsorTrackingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NetworkSponsorTrackingTransactorSession struct {
	Contract     *NetworkSponsorTrackingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// NetworkSponsorTrackingRaw is an auto generated low-level Go binding around an Ethereum contract.
type NetworkSponsorTrackingRaw struct {
	Contract *NetworkSponsorTracking // Generic contract binding to access the raw methods on
}

// NetworkSponsorTrackingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NetworkSponsorTrackingCallerRaw struct {
	Contract *NetworkSponsorTrackingCaller // Generic read-only contract binding to access the raw methods on
}

// NetworkSponsorTrackingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NetworkSponsorTrackingTransactorRaw struct {
	Contract *NetworkSponsorTrackingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNetworkSponsorTracking creates a new instance of NetworkSponsorTracking, bound to a specific deployed contract.
func NewNetworkSponsorTracking(address common.Address, backend bind.ContractBackend) (*NetworkSponsorTracking, error) {
	contract, err := bindNetworkSponsorTracking(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorTracking{NetworkSponsorTrackingCaller: NetworkSponsorTrackingCaller{contract: contract}, NetworkSponsorTrackingTransactor: NetworkSponsorTrackingTransactor{contract: contract}, NetworkSponsorTrackingFilterer: NetworkSponsorTrackingFilterer{contract: contract}}, nil
}

// NewNetworkSponsorTrackingCaller creates a new read-only instance of NetworkSponsorTracking, bound to a specific deployed contract.
func NewNetworkSponsorTrackingCaller(address common.Address, caller bind.ContractCaller) (*NetworkSponsorTrackingCaller, error) {
	contract, err := bindNetworkSponsorTracking(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorTrackingCaller{contract: contract}, nil
}

// NewNetworkSponsorTrackingTransactor creates a new write-only instance of NetworkSponsorTracking, bound to a specific deployed contract.
func NewNetworkSponsorTrackingTransactor(address common.Address, transactor bind.ContractTransactor) (*NetworkSponsorTrackingTransactor, error) {
	contract, err := bindNetworkSponsorTracking(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorTrackingTransactor{contract: contract}, nil
}

// NewNetworkSponsorTrackingFilterer creates a new log filterer instance of NetworkSponsorTracking, bound to a specific deployed contract.
func NewNetworkSponsorTrackingFilterer(address common.Address, filterer bind.ContractFilterer) (*NetworkSponsorTrackingFilterer, error) {
	contract, err := bindNetworkSponsorTracking(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorTrackingFilterer{contract: contract}, nil
}

// bindNetworkSponsorTracking binds a generic wrapper to an already deployed contract.
func bindNetworkSponsorTracking(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NetworkSponsorTrackingMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NetworkSponsorTracking *NetworkSponsorTrackingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NetworkSponsorTracking.Contract.NetworkSponsorTrackingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NetworkSponsorTracking *NetworkSponsorTrackingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NetworkSponsorTracking.Contract.NetworkSponsorTrackingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NetworkSponsorTracking *NetworkSponsorTrackingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NetworkSponsorTracking.Contract.NetworkSponsorTrackingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NetworkSponsorTracking *NetworkSponsorTrackingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NetworkSponsorTracking.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NetworkSponsorTracking *NetworkSponsorTrackingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NetworkSponsorTracking.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NetworkSponsorTracking *NetworkSponsorTrackingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NetworkSponsorTracking.Contract.contract.Transact(opts, method, params...)
}

// TRACKINGID is a free data retrieval call binding the contract method 0x659ee7bf.
//
// Solidity: function TRACKING_ID() view returns(bytes32)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCaller) TRACKINGID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _NetworkSponsorTracking.contract.Call(opts, &out, "TRACKING_ID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// TRACKINGID is a free data retrieval call binding the contract method 0x659ee7bf.
//
// Solidity: function TRACKING_ID() view returns(bytes32)
func (_NetworkSponsorTracking *NetworkSponsorTrackingSession) TRACKINGID() ([32]byte, error) {
	return _NetworkSponsorTracking.Contract.TRACKINGID(&_NetworkSponsorTracking.CallOpts)
}

// TRACKINGID is a free data retrieval call binding the contract method 0x659ee7bf.
//
// Solidity: function TRACKING_ID() view returns(bytes32)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCallerSession) TRACKINGID() ([32]byte, error) {
	return _NetworkSponsorTracking.Contract.TRACKINGID(&_NetworkSponsorTracking.CallOpts)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	var out []interface{}
	err := _NetworkSponsorTracking.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, arg5)

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
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_NetworkSponsorTracking *NetworkSponsorTrackingSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _NetworkSponsorTracking.Contract.ChooseFund(&_NetworkSponsorTracking.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _NetworkSponsorTracking.Contract.ChooseFund(&_NetworkSponsorTracking.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_NetworkSponsorTracking *NetworkSponsorTrackingCaller) DeductFees(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _NetworkSponsorTracking.contract.Call(opts, &out, "deductFees", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_NetworkSponsorTracking *NetworkSponsorTrackingSession) DeductFees(arg0 [32]byte, arg1 *big.Int) error {
	return _NetworkSponsorTracking.Contract.DeductFees(&_NetworkSponsorTracking.CallOpts, arg0, arg1)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_NetworkSponsorTracking *NetworkSponsorTrackingCallerSession) DeductFees(arg0 [32]byte, arg1 *big.Int) error {
	return _NetworkSponsorTracking.Contract.DeductFees(&_NetworkSponsorTracking.CallOpts, arg0, arg1)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	var out []interface{}
	err := _NetworkSponsorTracking.contract.Call(opts, &out, "getGasConfig")

	outstruct := new(struct {
		ChooseFundLimit *big.Int
		DeductFeesLimit *big.Int
		OverheadCharge  *big.Int
		TrackGasCost    *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ChooseFundLimit = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.DeductFeesLimit = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.OverheadCharge = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TrackGasCost = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_NetworkSponsorTracking *NetworkSponsorTrackingSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	return _NetworkSponsorTracking.Contract.GetGasConfig(&_NetworkSponsorTracking.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCallerSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	return _NetworkSponsorTracking.Contract.GetGasConfig(&_NetworkSponsorTracking.CallOpts)
}

// Track is a paid mutator transaction binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 trackingId, uint256 fee) returns()
func (_NetworkSponsorTracking *NetworkSponsorTrackingTransactor) Track(opts *bind.TransactOpts, trackingId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _NetworkSponsorTracking.contract.Transact(opts, "track", trackingId, fee)
}

// Track is a paid mutator transaction binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 trackingId, uint256 fee) returns()
func (_NetworkSponsorTracking *NetworkSponsorTrackingSession) Track(trackingId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _NetworkSponsorTracking.Contract.Track(&_NetworkSponsorTracking.TransactOpts, trackingId, fee)
}

// Track is a paid mutator transaction binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 trackingId, uint256 fee) returns()
func (_NetworkSponsorTracking *NetworkSponsorTrackingTransactorSession) Track(trackingId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _NetworkSponsorTracking.Contract.Track(&_NetworkSponsorTracking.TransactOpts, trackingId, fee)
}

// NetworkSponsorTrackingTrackedIterator is returned from FilterTracked and is used to iterate over the raw logs and unpacked data for Tracked events raised by the NetworkSponsorTracking contract.
type NetworkSponsorTrackingTrackedIterator struct {
	Event *NetworkSponsorTrackingTracked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NetworkSponsorTrackingTrackedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NetworkSponsorTrackingTracked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NetworkSponsorTrackingTracked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NetworkSponsorTrackingTrackedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NetworkSponsorTrackingTrackedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NetworkSponsorTrackingTracked represents a Tracked event raised by the NetworkSponsorTracking contract.
type NetworkSponsorTrackingTracked struct {
	TrackingId [32]byte
	Fee        *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTracked is a free log retrieval operation binding the contract event 0x408d8fef8a4d6c626c249452c6039e7f00e5cff889b3ee3c9e237febce8ce668.
//
// Solidity: event Tracked(bytes32 indexed trackingId, uint256 fee)
func (_NetworkSponsorTracking *NetworkSponsorTrackingFilterer) FilterTracked(opts *bind.FilterOpts, trackingId [][32]byte) (*NetworkSponsorTrackingTrackedIterator, error) {

	var trackingIdRule []interface{}
	for _, trackingIdItem := range trackingId {
		trackingIdRule = append(trackingIdRule, trackingIdItem)
	}

	logs, sub, err := _NetworkSponsorTracking.contract.FilterLogs(opts, "Tracked", trackingIdRule)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorTrackingTrackedIterator{contract: _NetworkSponsorTracking.contract, event: "Tracked", logs: logs, sub: sub}, nil
}

// WatchTracked is a free log subscription operation binding the contract event 0x408d8fef8a4d6c626c249452c6039e7f00e5cff889b3ee3c9e237febce8ce668.
//
// Solidity: event Tracked(bytes32 indexed trackingId, uint256 fee)
func (_NetworkSponsorTracking *NetworkSponsorTrackingFilterer) WatchTracked(opts *bind.WatchOpts, sink chan<- *NetworkSponsorTrackingTracked, trackingId [][32]byte) (event.Subscription, error) {

	var trackingIdRule []interface{}
	for _, trackingIdItem := range trackingId {
		trackingIdRule = append(trackingIdRule, trackingIdItem)
	}

	logs, sub, err := _NetworkSponsorTracking.contract.WatchLogs(opts, "Tracked", trackingIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NetworkSponsorTrackingTracked)
				if err := _NetworkSponsorTracking.contract.UnpackLog(event, "Tracked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTracked is a log parse operation binding the contract event 0x408d8fef8a4d6c626c249452c6039e7f00e5cff889b3ee3c9e237febce8ce668.
//
// Solidity: event Tracked(bytes32 indexed trackingId, uint256 fee)
func (_NetworkSponsorTracking *NetworkSponsorTrackingFilterer) ParseTracked(log types.Log) (*NetworkSponsorTrackingTracked, error) {
	event := new(NetworkSponsorTrackingTracked)
	if err := _NetworkSponsorTracking.contract.UnpackLog(event, "Tracked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
