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
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"trackingId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"Tracked\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"TRACKING_ID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimitForChooseFund\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForDeductFees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForTrack\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForFundBackedSponsorships\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForNetworkSponsorshipsWithTracking\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"trackingId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b506106fd8061001c5f395ff3fe608060405234801561000f575f80fd5b5060043610610055575f3560e01c8063399f59ca146100595780634b5c54c01461008a578063659ee7bf146100ac578063b9ed9f26146100ca578063bf70eb15146100e6575b5f80fd5b610073600480360381019061006e9190610358565b610102565b604051610081929190610429565b60405180910390f35b61009261011d565b6040516100a3959493929190610450565b60405180910390f35b6100b4610173565b6040516100c191906104a1565b60405180910390f35b6100e460048036038101906100df91906104e4565b61017d565b005b61010060048036038101906100fb91906104e4565b6101b8565b005b5f80600363deadbeef5f1b9150915097509795505050505050565b5f805f805f8061c3509050620186a0955061ea609450620138809350808587610146919061054f565b610150919061054f565b925080848761015f919061054f565b610169919061054f565b9150509091929394565b63deadbeef5f1b81565b6040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101af90610602565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610226576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161021d9061066a565b60405180910390fd5b817f408d8fef8a4d6c626c249452c6039e7f00e5cff889b3ee3c9e237febce8ce668826040516102569190610688565b60405180910390a25050565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6102938261026a565b9050919050565b6102a381610289565b81146102ad575f80fd5b50565b5f813590506102be8161029a565b92915050565b5f819050919050565b6102d6816102c4565b81146102e0575f80fd5b50565b5f813590506102f1816102cd565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f840112610318576103176102f7565b5b8235905067ffffffffffffffff811115610335576103346102fb565b5b602083019150836001820283011115610351576103506102ff565b5b9250929050565b5f805f805f805f60c0888a03121561037357610372610262565b5b5f6103808a828b016102b0565b97505060206103918a828b016102b0565b96505060406103a28a828b016102e3565b95505060606103b38a828b016102e3565b945050608088013567ffffffffffffffff8111156103d4576103d3610266565b5b6103e08a828b01610303565b935093505060a06103f38a828b016102e3565b91505092959891949750929550565b61040b816102c4565b82525050565b5f819050919050565b61042381610411565b82525050565b5f60408201905061043c5f830185610402565b610449602083018461041a565b9392505050565b5f60a0820190506104635f830188610402565b6104706020830187610402565b61047d6040830186610402565b61048a6060830185610402565b6104976080830184610402565b9695505050505050565b5f6020820190506104b45f83018461041a565b92915050565b6104c381610411565b81146104cd575f80fd5b50565b5f813590506104de816104ba565b92915050565b5f80604083850312156104fa576104f9610262565b5b5f610507858286016104d0565b9250506020610518858286016102e3565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610559826102c4565b9150610564836102c4565b925082820190508082111561057c5761057b610522565b5b92915050565b5f82825260208201905092915050565b7f646564756374466565732073686f756c64206e6f742062652063616c6c6564205f8201527f666f72206d6f6465203300000000000000000000000000000000000000000000602082015250565b5f6105ec602a83610582565b91506105f782610592565b604082019050919050565b5f6020820190508181035f830152610619816105e0565b9050919050565b7f6f6e6c7920696e7465726e616c207472616e73616374696f6e730000000000005f82015250565b5f610654601a83610582565b915061065f82610620565b602082019050919050565b5f6020820190508181035f83015261068181610648565b9050919050565b5f60208201905061069b5f830184610402565b9291505056fea26469706673582212206fb1641d8cbb468437eae3fb3c3876c24d4d46b4ec8549879172981cd754d4bc64736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
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
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	var out []interface{}
	err := _NetworkSponsorTracking.contract.Call(opts, &out, "getGasConfig")

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
func (_NetworkSponsorTracking *NetworkSponsorTrackingSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _NetworkSponsorTracking.Contract.GetGasConfig(&_NetworkSponsorTracking.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_NetworkSponsorTracking *NetworkSponsorTrackingCallerSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
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
