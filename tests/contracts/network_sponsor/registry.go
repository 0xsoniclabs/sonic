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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimitForChooseFund\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForDeductFees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForTrack\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForFundBackedSponsorships\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForNetworkSponsorshipsWithTracking\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b5061064b8061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061004a575f3560e01c8063399f59ca1461004e5780634b5c54c01461007f578063b9ed9f26146100a1578063bf70eb15146100bd575b5f80fd5b610068600480360381019061006391906102b2565b6100d9565b604051610076929190610383565b60405180910390f35b6100876100f0565b6040516100989594939291906103aa565b60405180910390f35b6100bb60048036038101906100b69190610425565b610146565b005b6100d760048036038101906100d29190610425565b610181565b005b5f8060025f801b9150915097509795505050505050565b5f805f805f8061c3509050620186a0955061ea6094506201388093508085876101199190610490565b6101239190610490565b92508084876101329190610490565b61013c9190610490565b9150509091929394565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161017890610543565b60405180910390fd5b6040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101b3906105d1565b60405180910390fd5b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6101ed826101c4565b9050919050565b6101fd816101e3565b8114610207575f80fd5b50565b5f81359050610218816101f4565b92915050565b5f819050919050565b6102308161021e565b811461023a575f80fd5b50565b5f8135905061024b81610227565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f84011261027257610271610251565b5b8235905067ffffffffffffffff81111561028f5761028e610255565b5b6020830191508360018202830111156102ab576102aa610259565b5b9250929050565b5f805f805f805f60c0888a0312156102cd576102cc6101bc565b5b5f6102da8a828b0161020a565b97505060206102eb8a828b0161020a565b96505060406102fc8a828b0161023d565b955050606061030d8a828b0161023d565b945050608088013567ffffffffffffffff81111561032e5761032d6101c0565b5b61033a8a828b0161025d565b935093505060a061034d8a828b0161023d565b91505092959891949750929550565b6103658161021e565b82525050565b5f819050919050565b61037d8161036b565b82525050565b5f6040820190506103965f83018561035c565b6103a36020830184610374565b9392505050565b5f60a0820190506103bd5f83018861035c565b6103ca602083018761035c565b6103d7604083018661035c565b6103e4606083018561035c565b6103f1608083018461035c565b9695505050505050565b6104048161036b565b811461040e575f80fd5b50565b5f8135905061041f816103fb565b92915050565b5f806040838503121561043b5761043a6101bc565b5b5f61044885828601610411565b92505060206104598582860161023d565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61049a8261021e565b91506104a58361021e565b92508282019050808211156104bd576104bc610463565b5b92915050565b5f82825260208201905092915050565b7f646564756374466565732073686f756c64206e6f742062652063616c6c6564205f8201527f666f72206d6f6465203200000000000000000000000000000000000000000000602082015250565b5f61052d602a836104c3565b9150610538826104d3565b604082019050919050565b5f6020820190508181035f83015261055a81610521565b9050919050565b7f747261636b2073686f756c64206e6f742062652063616c6c656420666f72206d5f8201527f6f64652032000000000000000000000000000000000000000000000000000000602082015250565b5f6105bb6025836104c3565b91506105c682610561565b604082019050919050565b5f6020820190508181035f8301526105e8816105af565b905091905056fea264697066735822122000d152e98983b115de97f2715ca0f9d0fe4878b6bddb5babf0268babb25f962b64736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
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
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_NetworkSponsor *NetworkSponsorCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	var out []interface{}
	err := _NetworkSponsor.contract.Call(opts, &out, "getGasConfig")

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
func (_NetworkSponsor *NetworkSponsorSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _NetworkSponsor.Contract.GetGasConfig(&_NetworkSponsor.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_NetworkSponsor *NetworkSponsorCallerSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
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
