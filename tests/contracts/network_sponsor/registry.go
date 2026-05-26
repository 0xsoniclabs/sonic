// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package network_sponsor

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

// NetworkSponsorMetaData contains all meta data concerning the NetworkSponsor contract.
var NetworkSponsorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadCharge\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"trackGasCost\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f80fd5b506105f38061001d5f395ff3fe608060405234801561000f575f80fd5b506004361061004a575f3560e01c8063399f59ca1461004e5780634b5c54c01461007f578063b9ed9f26146100a0578063bf70eb15146100bc575b5f80fd5b6100686004803603810190610063919061028e565b6100d8565b60405161007692919061035f565b60405180910390f35b6100876100ef565b6040516100979493929190610386565b60405180910390f35b6100ba60048036038101906100b591906103f3565b610122565b005b6100d660048036038101906100d191906103f3565b61015d565b005b5f8060025f801b9150915097509795505050505050565b5f805f80620186a0935061ea60925061c350838561010d919061045e565b610117919061045e565b91505f905090919293565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161015490610511565b60405180910390fd5b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161018f9061059f565b60405180910390fd5b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6101c9826101a0565b9050919050565b6101d9816101bf565b81146101e3575f80fd5b50565b5f813590506101f4816101d0565b92915050565b5f819050919050565b61020c816101fa565b8114610216575f80fd5b50565b5f8135905061022781610203565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f84011261024e5761024d61022d565b5b8235905067ffffffffffffffff81111561026b5761026a610231565b5b60208301915083600182028301111561028757610286610235565b5b9250929050565b5f805f805f805f60c0888a0312156102a9576102a8610198565b5b5f6102b68a828b016101e6565b97505060206102c78a828b016101e6565b96505060406102d88a828b01610219565b95505060606102e98a828b01610219565b945050608088013567ffffffffffffffff81111561030a5761030961019c565b5b6103168a828b01610239565b935093505060a06103298a828b01610219565b91505092959891949750929550565b610341816101fa565b82525050565b5f819050919050565b61035981610347565b82525050565b5f6040820190506103725f830185610338565b61037f6020830184610350565b9392505050565b5f6080820190506103995f830187610338565b6103a66020830186610338565b6103b36040830185610338565b6103c06060830184610338565b95945050505050565b6103d281610347565b81146103dc575f80fd5b50565b5f813590506103ed816103c9565b92915050565b5f806040838503121561040957610408610198565b5b5f610416858286016103df565b925050602061042785828601610219565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610468826101fa565b9150610473836101fa565b925082820190508082111561048b5761048a610431565b5b92915050565b5f82825260208201905092915050565b7f646564756374466565732073686f756c64206e6f742062652063616c6c6564205f8201527f666f72206d6f6465203200000000000000000000000000000000000000000000602082015250565b5f6104fb602a83610491565b9150610506826104a1565b604082019050919050565b5f6020820190508181035f830152610528816104ef565b9050919050565b7f747261636b2073686f756c64206e6f742062652063616c6c656420666f72206d5f8201527f6f64652032000000000000000000000000000000000000000000000000000000602082015250565b5f610589602583610491565b91506105948261052f565b604082019050919050565b5f6020820190508181035f8301526105b68161057d565b905091905056fea264697066735822122086fc96f2b966f1fd84d91a90e52e5f29ebbb7fb4854ebb0ed7ae73089489970464736f6c63430008180033",
}

// NetworkSponsorABI is the input ABI used to generate the binding from.
// Deprecated: Use NetworkSponsorMetaData.ABI instead.
var NetworkSponsorABI = NetworkSponsorMetaData.ABI

// NetworkSponsorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use NetworkSponsorMetaData.Bin instead.
var NetworkSponsorBin = NetworkSponsorMetaData.Bin

// DeployNetworkSponsor deploys a new Ethereum contract, binding an instance of NetworkSponsor to it.
func DeployNetworkSponsor(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *NetworkSponsor, error) {
	parsed, err := NetworkSponsorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(NetworkSponsorBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NetworkSponsor{NetworkSponsorCaller: NetworkSponsorCaller{contract: contract}, NetworkSponsorTransactor: NetworkSponsorTransactor{contract: contract}, NetworkSponsorFilterer: NetworkSponsorFilterer{contract: contract}}, nil
}

// NetworkSponsor is an auto generated Go binding around an Ethereum contract.
type NetworkSponsor struct {
	NetworkSponsorCaller     // Read-only binding to the contract
	NetworkSponsorTransactor // Write-only binding to the contract
	NetworkSponsorFilterer   // Log filterer for contract events
}

// NetworkSponsorCaller is an auto generated read-only Go binding around an Ethereum contract.
type NetworkSponsorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NetworkSponsorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NetworkSponsorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NetworkSponsorSession struct {
	Contract     *NetworkSponsor   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NetworkSponsorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NetworkSponsorCallerSession struct {
	Contract *NetworkSponsorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// NetworkSponsorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NetworkSponsorTransactorSession struct {
	Contract     *NetworkSponsorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// NetworkSponsorRaw is an auto generated low-level Go binding around an Ethereum contract.
type NetworkSponsorRaw struct {
	Contract *NetworkSponsor // Generic contract binding to access the raw methods on
}

// NetworkSponsorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NetworkSponsorCallerRaw struct {
	Contract *NetworkSponsorCaller // Generic read-only contract binding to access the raw methods on
}

// NetworkSponsorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NetworkSponsorTransactorRaw struct {
	Contract *NetworkSponsorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNetworkSponsor creates a new instance of NetworkSponsor, bound to a specific deployed contract.
func NewNetworkSponsor(address common.Address, backend bind.ContractBackend) (*NetworkSponsor, error) {
	contract, err := bindNetworkSponsor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsor{NetworkSponsorCaller: NetworkSponsorCaller{contract: contract}, NetworkSponsorTransactor: NetworkSponsorTransactor{contract: contract}, NetworkSponsorFilterer: NetworkSponsorFilterer{contract: contract}}, nil
}

// NewNetworkSponsorCaller creates a new read-only instance of NetworkSponsor, bound to a specific deployed contract.
func NewNetworkSponsorCaller(address common.Address, caller bind.ContractCaller) (*NetworkSponsorCaller, error) {
	contract, err := bindNetworkSponsor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorCaller{contract: contract}, nil
}

// NewNetworkSponsorTransactor creates a new write-only instance of NetworkSponsor, bound to a specific deployed contract.
func NewNetworkSponsorTransactor(address common.Address, transactor bind.ContractTransactor) (*NetworkSponsorTransactor, error) {
	contract, err := bindNetworkSponsor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorTransactor{contract: contract}, nil
}

// NewNetworkSponsorFilterer creates a new log filterer instance of NetworkSponsor, bound to a specific deployed contract.
func NewNetworkSponsorFilterer(address common.Address, filterer bind.ContractFilterer) (*NetworkSponsorFilterer, error) {
	contract, err := bindNetworkSponsor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorFilterer{contract: contract}, nil
}

// bindNetworkSponsor binds a generic wrapper to an already deployed contract.
func bindNetworkSponsor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NetworkSponsorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NetworkSponsor *NetworkSponsorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NetworkSponsor.Contract.NetworkSponsorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NetworkSponsor *NetworkSponsorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NetworkSponsor.Contract.NetworkSponsorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NetworkSponsor *NetworkSponsorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NetworkSponsor.Contract.NetworkSponsorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NetworkSponsor *NetworkSponsorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NetworkSponsor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NetworkSponsor *NetworkSponsorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NetworkSponsor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NetworkSponsor *NetworkSponsorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NetworkSponsor.Contract.contract.Transact(opts, method, params...)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_NetworkSponsor *NetworkSponsorCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	var out []interface{}
	err := _NetworkSponsor.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, arg5)

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
func (_NetworkSponsor *NetworkSponsorSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _NetworkSponsor.Contract.ChooseFund(&_NetworkSponsor.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_NetworkSponsor *NetworkSponsorCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _NetworkSponsor.Contract.ChooseFund(&_NetworkSponsor.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_NetworkSponsor *NetworkSponsorCaller) DeductFees(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _NetworkSponsor.contract.Call(opts, &out, "deductFees", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_NetworkSponsor *NetworkSponsorSession) DeductFees(arg0 [32]byte, arg1 *big.Int) error {
	return _NetworkSponsor.Contract.DeductFees(&_NetworkSponsor.CallOpts, arg0, arg1)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_NetworkSponsor *NetworkSponsorCallerSession) DeductFees(arg0 [32]byte, arg1 *big.Int) error {
	return _NetworkSponsor.Contract.DeductFees(&_NetworkSponsor.CallOpts, arg0, arg1)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_NetworkSponsor *NetworkSponsorCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	var out []interface{}
	err := _NetworkSponsor.contract.Call(opts, &out, "getGasConfig")

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
func (_NetworkSponsor *NetworkSponsorSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	return _NetworkSponsor.Contract.GetGasConfig(&_NetworkSponsor.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_NetworkSponsor *NetworkSponsorCallerSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	return _NetworkSponsor.Contract.GetGasConfig(&_NetworkSponsor.CallOpts)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) pure returns()
func (_NetworkSponsor *NetworkSponsorCaller) Track(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _NetworkSponsor.contract.Call(opts, &out, "track", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) pure returns()
func (_NetworkSponsor *NetworkSponsorSession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _NetworkSponsor.Contract.Track(&_NetworkSponsor.CallOpts, arg0, arg1)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) pure returns()
func (_NetworkSponsor *NetworkSponsorCallerSession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _NetworkSponsor.Contract.Track(&_NetworkSponsor.CallOpts, arg0, arg1)
}
