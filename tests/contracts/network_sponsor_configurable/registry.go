// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package network_sponsor_configurable

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

// NetworkSponsorConfigurableMetaData contains all meta data concerning the NetworkSponsorConfigurable contract.
var NetworkSponsorConfigurableMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimitForChooseFund\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForDeductFees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForTrack\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForFundBackedSponsorships\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForNetworkSponsorshipsWithTracking\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sponsored\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"value\",\"type\":\"bool\"}],\"name\":\"setFundSponsored\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sponsored\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"value\",\"type\":\"bool\"}],\"name\":\"setNetworkSponsored\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sponsored\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"value\",\"type\":\"bool\"}],\"name\":\"setNetworkSponsoredWithTrackingSponsored\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"trackingId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b5061090d8061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061007b575f3560e01c8063b9ed9f2611610059578063b9ed9f26146100ee578063bf70eb151461010a578063ccf940d214610126578063f152744b146101425761007b565b8063399f59ca1461007f5780633e3cc02d146100b05780634b5c54c0146100cc575b5f5ffd5b61009960048036038101906100949190610573565b61015e565b6040516100a7929190610644565b60405180910390f35b6100ca60048036038101906100c591906106a0565b61028c565b005b6100d46102e4565b6040516100e59594939291906106de565b60405180910390f35b61010860048036038101906101039190610759565b61033a565b005b610124600480360381019061011f9190610759565b610384565b005b610140600480360381019061013b91906106a0565b6103ce565b005b61015c600480360381019061015791906106a0565b610425565b005b5f5f60015f8a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff16156101bd5760025f5f1b91509150610280565b5f5f8a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff1615610219576001805f1b91509150610280565b60025f8a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff161561027757600360025f1b91509150610280565b5f5f5f1b915091505b97509795505050505050565b8060025f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff0219169083151502179055505050565b5f5f5f5f5f5f61c3509050620186a0955061ea60945062013880935080858761030d91906107c4565b61031791906107c4565b925080848761032691906107c4565b61033091906107c4565b9150509091929394565b60015f1b820315610380576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161037790610851565b60405180910390fd5b5050565b60025f1b8203156103ca576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103c1906108b9565b60405180910390fd5b5050565b805f5f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff0219169083151502179055505050565b8060015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff0219169083151502179055505050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6104ae82610485565b9050919050565b6104be816104a4565b81146104c8575f5ffd5b50565b5f813590506104d9816104b5565b92915050565b5f819050919050565b6104f1816104df565b81146104fb575f5ffd5b50565b5f8135905061050c816104e8565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261053357610532610512565b5b8235905067ffffffffffffffff8111156105505761054f610516565b5b60208301915083600182028301111561056c5761056b61051a565b5b9250929050565b5f5f5f5f5f5f5f60c0888a03121561058e5761058d61047d565b5b5f61059b8a828b016104cb565b97505060206105ac8a828b016104cb565b96505060406105bd8a828b016104fe565b95505060606105ce8a828b016104fe565b945050608088013567ffffffffffffffff8111156105ef576105ee610481565b5b6105fb8a828b0161051e565b935093505060a061060e8a828b016104fe565b91505092959891949750929550565b610626816104df565b82525050565b5f819050919050565b61063e8161062c565b82525050565b5f6040820190506106575f83018561061d565b6106646020830184610635565b9392505050565b5f8115159050919050565b61067f8161066b565b8114610689575f5ffd5b50565b5f8135905061069a81610676565b92915050565b5f5f604083850312156106b6576106b561047d565b5b5f6106c3858286016104cb565b92505060206106d48582860161068c565b9150509250929050565b5f60a0820190506106f15f83018861061d565b6106fe602083018761061d565b61070b604083018661061d565b610718606083018561061d565b610725608083018461061d565b9695505050505050565b6107388161062c565b8114610742575f5ffd5b50565b5f813590506107538161072f565b92915050565b5f5f6040838503121561076f5761076e61047d565b5b5f61077c85828601610745565b925050602061078d858286016104fe565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6107ce826104df565b91506107d9836104df565b92508282019050808211156107f1576107f0610797565b5b92915050565b5f82825260208201905092915050565b7f646564756374466565733a20756e6b6e6f776e2066756e6449640000000000005f82015250565b5f61083b601a836107f7565b915061084682610807565b602082019050919050565b5f6020820190508181035f8301526108688161082f565b9050919050565b7f747261636b3a20756e6b6e6f776e20747261636b696e674964000000000000005f82015250565b5f6108a36019836107f7565b91506108ae8261086f565b602082019050919050565b5f6020820190508181035f8301526108d081610897565b905091905056fea26469706673582212201246cb6c5d55e97a4a87f0cdaf0a72c86cdcb9ce2127a88f3dba1d95537810ff64736f6c634300081b0033",
}

// NetworkSponsorConfigurableABI is the input ABI used to generate the binding from.
// Deprecated: Use NetworkSponsorConfigurableMetaData.ABI instead.
var NetworkSponsorConfigurableABI = NetworkSponsorConfigurableMetaData.ABI

// NetworkSponsorConfigurableBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use NetworkSponsorConfigurableMetaData.Bin instead.
var NetworkSponsorConfigurableBin = NetworkSponsorConfigurableMetaData.Bin

// DeployNetworkSponsorConfigurable deploys a new Ethereum contract, binding an instance of NetworkSponsorConfigurable to it.
func DeployNetworkSponsorConfigurable(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *NetworkSponsorConfigurable, error) {
	parsed, err := NetworkSponsorConfigurableMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(NetworkSponsorConfigurableBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NetworkSponsorConfigurable{NetworkSponsorConfigurableCaller: NetworkSponsorConfigurableCaller{contract: contract}, NetworkSponsorConfigurableTransactor: NetworkSponsorConfigurableTransactor{contract: contract}, NetworkSponsorConfigurableFilterer: NetworkSponsorConfigurableFilterer{contract: contract}}, nil
}

// NetworkSponsorConfigurable is an auto generated Go binding around an Ethereum contract.
type NetworkSponsorConfigurable struct {
	NetworkSponsorConfigurableCaller     // Read-only binding to the contract
	NetworkSponsorConfigurableTransactor // Write-only binding to the contract
	NetworkSponsorConfigurableFilterer   // Log filterer for contract events
}

// NetworkSponsorConfigurableCaller is an auto generated read-only Go binding around an Ethereum contract.
type NetworkSponsorConfigurableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorConfigurableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NetworkSponsorConfigurableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorConfigurableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NetworkSponsorConfigurableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NetworkSponsorConfigurableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NetworkSponsorConfigurableSession struct {
	Contract     *NetworkSponsorConfigurable // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// NetworkSponsorConfigurableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NetworkSponsorConfigurableCallerSession struct {
	Contract *NetworkSponsorConfigurableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// NetworkSponsorConfigurableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NetworkSponsorConfigurableTransactorSession struct {
	Contract     *NetworkSponsorConfigurableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// NetworkSponsorConfigurableRaw is an auto generated low-level Go binding around an Ethereum contract.
type NetworkSponsorConfigurableRaw struct {
	Contract *NetworkSponsorConfigurable // Generic contract binding to access the raw methods on
}

// NetworkSponsorConfigurableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NetworkSponsorConfigurableCallerRaw struct {
	Contract *NetworkSponsorConfigurableCaller // Generic read-only contract binding to access the raw methods on
}

// NetworkSponsorConfigurableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NetworkSponsorConfigurableTransactorRaw struct {
	Contract *NetworkSponsorConfigurableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNetworkSponsorConfigurable creates a new instance of NetworkSponsorConfigurable, bound to a specific deployed contract.
func NewNetworkSponsorConfigurable(address common.Address, backend bind.ContractBackend) (*NetworkSponsorConfigurable, error) {
	contract, err := bindNetworkSponsorConfigurable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorConfigurable{NetworkSponsorConfigurableCaller: NetworkSponsorConfigurableCaller{contract: contract}, NetworkSponsorConfigurableTransactor: NetworkSponsorConfigurableTransactor{contract: contract}, NetworkSponsorConfigurableFilterer: NetworkSponsorConfigurableFilterer{contract: contract}}, nil
}

// NewNetworkSponsorConfigurableCaller creates a new read-only instance of NetworkSponsorConfigurable, bound to a specific deployed contract.
func NewNetworkSponsorConfigurableCaller(address common.Address, caller bind.ContractCaller) (*NetworkSponsorConfigurableCaller, error) {
	contract, err := bindNetworkSponsorConfigurable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorConfigurableCaller{contract: contract}, nil
}

// NewNetworkSponsorConfigurableTransactor creates a new write-only instance of NetworkSponsorConfigurable, bound to a specific deployed contract.
func NewNetworkSponsorConfigurableTransactor(address common.Address, transactor bind.ContractTransactor) (*NetworkSponsorConfigurableTransactor, error) {
	contract, err := bindNetworkSponsorConfigurable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorConfigurableTransactor{contract: contract}, nil
}

// NewNetworkSponsorConfigurableFilterer creates a new log filterer instance of NetworkSponsorConfigurable, bound to a specific deployed contract.
func NewNetworkSponsorConfigurableFilterer(address common.Address, filterer bind.ContractFilterer) (*NetworkSponsorConfigurableFilterer, error) {
	contract, err := bindNetworkSponsorConfigurable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NetworkSponsorConfigurableFilterer{contract: contract}, nil
}

// bindNetworkSponsorConfigurable binds a generic wrapper to an already deployed contract.
func bindNetworkSponsorConfigurable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NetworkSponsorConfigurableMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NetworkSponsorConfigurable.Contract.NetworkSponsorConfigurableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.NetworkSponsorConfigurableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.NetworkSponsorConfigurableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NetworkSponsorConfigurable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.contract.Transact(opts, method, params...)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address , uint256 , uint256 , bytes , uint256 ) view returns(uint256 mode, bytes32 payload)
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCaller) ChooseFund(opts *bind.CallOpts, from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	var out []interface{}
	err := _NetworkSponsorConfigurable.contract.Call(opts, &out, "chooseFund", from, arg1, arg2, arg3, arg4, arg5)

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
// Solidity: function chooseFund(address from, address , uint256 , uint256 , bytes , uint256 ) view returns(uint256 mode, bytes32 payload)
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) ChooseFund(from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _NetworkSponsorConfigurable.Contract.ChooseFund(&_NetworkSponsorConfigurable.CallOpts, from, arg1, arg2, arg3, arg4, arg5)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address from, address , uint256 , uint256 , bytes , uint256 ) view returns(uint256 mode, bytes32 payload)
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCallerSession) ChooseFund(from common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _NetworkSponsorConfigurable.Contract.ChooseFund(&_NetworkSponsorConfigurable.CallOpts, from, arg1, arg2, arg3, arg4, arg5)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 ) pure returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCaller) DeductFees(opts *bind.CallOpts, fundId [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _NetworkSponsorConfigurable.contract.Call(opts, &out, "deductFees", fundId, arg1)

	if err != nil {
		return err
	}

	return err

}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 ) pure returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) DeductFees(fundId [32]byte, arg1 *big.Int) error {
	return _NetworkSponsorConfigurable.Contract.DeductFees(&_NetworkSponsorConfigurable.CallOpts, fundId, arg1)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 ) pure returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCallerSession) DeductFees(fundId [32]byte, arg1 *big.Int) error {
	return _NetworkSponsorConfigurable.Contract.DeductFees(&_NetworkSponsorConfigurable.CallOpts, fundId, arg1)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	var out []interface{}
	err := _NetworkSponsorConfigurable.contract.Call(opts, &out, "getGasConfig")

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
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _NetworkSponsorConfigurable.Contract.GetGasConfig(&_NetworkSponsorConfigurable.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCallerSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _NetworkSponsorConfigurable.Contract.GetGasConfig(&_NetworkSponsorConfigurable.CallOpts)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 trackingId, uint256 ) pure returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCaller) Track(opts *bind.CallOpts, trackingId [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _NetworkSponsorConfigurable.contract.Call(opts, &out, "track", trackingId, arg1)

	if err != nil {
		return err
	}

	return err

}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 trackingId, uint256 ) pure returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) Track(trackingId [32]byte, arg1 *big.Int) error {
	return _NetworkSponsorConfigurable.Contract.Track(&_NetworkSponsorConfigurable.CallOpts, trackingId, arg1)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 trackingId, uint256 ) pure returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableCallerSession) Track(trackingId [32]byte, arg1 *big.Int) error {
	return _NetworkSponsorConfigurable.Contract.Track(&_NetworkSponsorConfigurable.CallOpts, trackingId, arg1)
}

// SetFundSponsored is a paid mutator transaction binding the contract method 0xccf940d2.
//
// Solidity: function setFundSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactor) SetFundSponsored(opts *bind.TransactOpts, sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.contract.Transact(opts, "setFundSponsored", sponsored, value)
}

// SetFundSponsored is a paid mutator transaction binding the contract method 0xccf940d2.
//
// Solidity: function setFundSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) SetFundSponsored(sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.SetFundSponsored(&_NetworkSponsorConfigurable.TransactOpts, sponsored, value)
}

// SetFundSponsored is a paid mutator transaction binding the contract method 0xccf940d2.
//
// Solidity: function setFundSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactorSession) SetFundSponsored(sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.SetFundSponsored(&_NetworkSponsorConfigurable.TransactOpts, sponsored, value)
}

// SetNetworkSponsored is a paid mutator transaction binding the contract method 0xf152744b.
//
// Solidity: function setNetworkSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactor) SetNetworkSponsored(opts *bind.TransactOpts, sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.contract.Transact(opts, "setNetworkSponsored", sponsored, value)
}

// SetNetworkSponsored is a paid mutator transaction binding the contract method 0xf152744b.
//
// Solidity: function setNetworkSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) SetNetworkSponsored(sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.SetNetworkSponsored(&_NetworkSponsorConfigurable.TransactOpts, sponsored, value)
}

// SetNetworkSponsored is a paid mutator transaction binding the contract method 0xf152744b.
//
// Solidity: function setNetworkSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactorSession) SetNetworkSponsored(sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.SetNetworkSponsored(&_NetworkSponsorConfigurable.TransactOpts, sponsored, value)
}

// SetNetworkSponsoredWithTrackingSponsored is a paid mutator transaction binding the contract method 0x3e3cc02d.
//
// Solidity: function setNetworkSponsoredWithTrackingSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactor) SetNetworkSponsoredWithTrackingSponsored(opts *bind.TransactOpts, sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.contract.Transact(opts, "setNetworkSponsoredWithTrackingSponsored", sponsored, value)
}

// SetNetworkSponsoredWithTrackingSponsored is a paid mutator transaction binding the contract method 0x3e3cc02d.
//
// Solidity: function setNetworkSponsoredWithTrackingSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableSession) SetNetworkSponsoredWithTrackingSponsored(sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.SetNetworkSponsoredWithTrackingSponsored(&_NetworkSponsorConfigurable.TransactOpts, sponsored, value)
}

// SetNetworkSponsoredWithTrackingSponsored is a paid mutator transaction binding the contract method 0x3e3cc02d.
//
// Solidity: function setNetworkSponsoredWithTrackingSponsored(address sponsored, bool value) returns()
func (_NetworkSponsorConfigurable *NetworkSponsorConfigurableTransactorSession) SetNetworkSponsoredWithTrackingSponsored(sponsored common.Address, value bool) (*types.Transaction, error) {
	return _NetworkSponsorConfigurable.Contract.SetNetworkSponsoredWithTrackingSponsored(&_NetworkSponsorConfigurable.TransactOpts, sponsored, value)
}
