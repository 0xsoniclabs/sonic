// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package failing_post_tx_registry

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

// FailingPostTxRegistryMetaData contains all meta data concerning the FailingPostTxRegistry contract.
var FailingPostTxRegistryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gasLimitForChooseFund\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForDeductFees\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimitForTrack\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForFundBackedSponsorships\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadChargeForNetworkSponsorshipsWithTracking\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b506106298061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061004a575f3560e01c8063399f59ca1461004e5780634b5c54c01461007f578063b9ed9f26146100a1578063bf70eb15146100bd575b5f80fd5b610068600480360381019061006391906102b6565b6100d9565b604051610076929190610387565b60405180910390f35b6100876100f4565b6040516100989594939291906103ae565b60405180910390f35b6100bb60048036038101906100b69190610429565b61014a565b005b6100d760048036038101906100d29190610429565b610185565b005b5f80600363deadbeef5f1b9150915097509795505050505050565b5f805f805f8061c3509050620186a0955061ea60945062013880935080858761011d9190610494565b6101279190610494565b92508084876101369190610494565b6101409190610494565b9150509091929394565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161017c90610547565b60405180910390fd5b6040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101b7906105af565b60405180910390fd5b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6101f1826101c8565b9050919050565b610201816101e7565b811461020b575f80fd5b50565b5f8135905061021c816101f8565b92915050565b5f819050919050565b61023481610222565b811461023e575f80fd5b50565b5f8135905061024f8161022b565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f84011261027657610275610255565b5b8235905067ffffffffffffffff81111561029357610292610259565b5b6020830191508360018202830111156102af576102ae61025d565b5b9250929050565b5f805f805f805f60c0888a0312156102d1576102d06101c0565b5b5f6102de8a828b0161020e565b97505060206102ef8a828b0161020e565b96505060406103008a828b01610241565b95505060606103118a828b01610241565b945050608088013567ffffffffffffffff811115610332576103316101c4565b5b61033e8a828b01610261565b935093505060a06103518a828b01610241565b91505092959891949750929550565b61036981610222565b82525050565b5f819050919050565b6103818161036f565b82525050565b5f60408201905061039a5f830185610360565b6103a76020830184610378565b9392505050565b5f60a0820190506103c15f830188610360565b6103ce6020830187610360565b6103db6040830186610360565b6103e86060830185610360565b6103f56080830184610360565b9695505050505050565b6104088161036f565b8114610412575f80fd5b50565b5f81359050610423816103ff565b92915050565b5f806040838503121561043f5761043e6101c0565b5b5f61044c85828601610415565b925050602061045d85828601610241565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61049e82610222565b91506104a983610222565b92508282019050808211156104c1576104c0610467565b5b92915050565b5f82825260208201905092915050565b7f646564756374466565732073686f756c64206e6f742062652063616c6c6564205f8201527f666f72206d6f6465203300000000000000000000000000000000000000000000602082015250565b5f610531602a836104c7565b915061053c826104d7565b604082019050919050565b5f6020820190508181035f83015261055e81610525565b9050919050565b7f747261636b20616c77617973206661696c7300000000000000000000000000005f82015250565b5f6105996012836104c7565b91506105a482610565565b602082019050919050565b5f6020820190508181035f8301526105c68161058d565b905091905056fea2646970667358221220132db73ba2035efdd67e1afff740241737bb449dd339f506894ccb185e3619b864736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
}

// FailingPostTxRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use FailingPostTxRegistryMetaData.ABI instead.
var FailingPostTxRegistryABI = FailingPostTxRegistryMetaData.ABI

// FailingPostTxRegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use FailingPostTxRegistryMetaData.Bin instead.
var FailingPostTxRegistryBin = FailingPostTxRegistryMetaData.Bin

// DeployFailingPostTxRegistry deploys a new Ethereum contract, binding an instance of FailingPostTxRegistry to it.
func DeployFailingPostTxRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *FailingPostTxRegistry, error) {
	parsed, err := FailingPostTxRegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(FailingPostTxRegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &FailingPostTxRegistry{FailingPostTxRegistryCaller: FailingPostTxRegistryCaller{contract: contract}, FailingPostTxRegistryTransactor: FailingPostTxRegistryTransactor{contract: contract}, FailingPostTxRegistryFilterer: FailingPostTxRegistryFilterer{contract: contract}}, nil
}

// FailingPostTxRegistry is an auto generated Go binding around an Ethereum contract.
type FailingPostTxRegistry struct {
	FailingPostTxRegistryCaller     // Read-only binding to the contract
	FailingPostTxRegistryTransactor // Write-only binding to the contract
	FailingPostTxRegistryFilterer   // Log filterer for contract events
}

// FailingPostTxRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type FailingPostTxRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FailingPostTxRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FailingPostTxRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FailingPostTxRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FailingPostTxRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FailingPostTxRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FailingPostTxRegistrySession struct {
	Contract     *FailingPostTxRegistry // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// FailingPostTxRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FailingPostTxRegistryCallerSession struct {
	Contract *FailingPostTxRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// FailingPostTxRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FailingPostTxRegistryTransactorSession struct {
	Contract     *FailingPostTxRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// FailingPostTxRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type FailingPostTxRegistryRaw struct {
	Contract *FailingPostTxRegistry // Generic contract binding to access the raw methods on
}

// FailingPostTxRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FailingPostTxRegistryCallerRaw struct {
	Contract *FailingPostTxRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// FailingPostTxRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FailingPostTxRegistryTransactorRaw struct {
	Contract *FailingPostTxRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFailingPostTxRegistry creates a new instance of FailingPostTxRegistry, bound to a specific deployed contract.
func NewFailingPostTxRegistry(address common.Address, backend bind.ContractBackend) (*FailingPostTxRegistry, error) {
	contract, err := bindFailingPostTxRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FailingPostTxRegistry{FailingPostTxRegistryCaller: FailingPostTxRegistryCaller{contract: contract}, FailingPostTxRegistryTransactor: FailingPostTxRegistryTransactor{contract: contract}, FailingPostTxRegistryFilterer: FailingPostTxRegistryFilterer{contract: contract}}, nil
}

// NewFailingPostTxRegistryCaller creates a new read-only instance of FailingPostTxRegistry, bound to a specific deployed contract.
func NewFailingPostTxRegistryCaller(address common.Address, caller bind.ContractCaller) (*FailingPostTxRegistryCaller, error) {
	contract, err := bindFailingPostTxRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FailingPostTxRegistryCaller{contract: contract}, nil
}

// NewFailingPostTxRegistryTransactor creates a new write-only instance of FailingPostTxRegistry, bound to a specific deployed contract.
func NewFailingPostTxRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*FailingPostTxRegistryTransactor, error) {
	contract, err := bindFailingPostTxRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FailingPostTxRegistryTransactor{contract: contract}, nil
}

// NewFailingPostTxRegistryFilterer creates a new log filterer instance of FailingPostTxRegistry, bound to a specific deployed contract.
func NewFailingPostTxRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*FailingPostTxRegistryFilterer, error) {
	contract, err := bindFailingPostTxRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FailingPostTxRegistryFilterer{contract: contract}, nil
}

// bindFailingPostTxRegistry binds a generic wrapper to an already deployed contract.
func bindFailingPostTxRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FailingPostTxRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FailingPostTxRegistry *FailingPostTxRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FailingPostTxRegistry.Contract.FailingPostTxRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FailingPostTxRegistry *FailingPostTxRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FailingPostTxRegistry.Contract.FailingPostTxRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FailingPostTxRegistry *FailingPostTxRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FailingPostTxRegistry.Contract.FailingPostTxRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FailingPostTxRegistry *FailingPostTxRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FailingPostTxRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FailingPostTxRegistry *FailingPostTxRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FailingPostTxRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FailingPostTxRegistry *FailingPostTxRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FailingPostTxRegistry.Contract.contract.Transact(opts, method, params...)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_FailingPostTxRegistry *FailingPostTxRegistryCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	var out []interface{}
	err := _FailingPostTxRegistry.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, arg5)

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
func (_FailingPostTxRegistry *FailingPostTxRegistrySession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _FailingPostTxRegistry.Contract.ChooseFund(&_FailingPostTxRegistry.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 ) pure returns(uint256 mode, bytes32 payload)
func (_FailingPostTxRegistry *FailingPostTxRegistryCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, arg5 *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _FailingPostTxRegistry.Contract.ChooseFund(&_FailingPostTxRegistry.CallOpts, arg0, arg1, arg2, arg3, arg4, arg5)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_FailingPostTxRegistry *FailingPostTxRegistryCaller) DeductFees(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _FailingPostTxRegistry.contract.Call(opts, &out, "deductFees", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_FailingPostTxRegistry *FailingPostTxRegistrySession) DeductFees(arg0 [32]byte, arg1 *big.Int) error {
	return _FailingPostTxRegistry.Contract.DeductFees(&_FailingPostTxRegistry.CallOpts, arg0, arg1)
}

// DeductFees is a free data retrieval call binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 , uint256 ) pure returns()
func (_FailingPostTxRegistry *FailingPostTxRegistryCallerSession) DeductFees(arg0 [32]byte, arg1 *big.Int) error {
	return _FailingPostTxRegistry.Contract.DeductFees(&_FailingPostTxRegistry.CallOpts, arg0, arg1)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_FailingPostTxRegistry *FailingPostTxRegistryCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	var out []interface{}
	err := _FailingPostTxRegistry.contract.Call(opts, &out, "getGasConfig")

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
func (_FailingPostTxRegistry *FailingPostTxRegistrySession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _FailingPostTxRegistry.Contract.GetGasConfig(&_FailingPostTxRegistry.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 gasLimitForChooseFund, uint256 gasLimitForDeductFees, uint256 gasLimitForTrack, uint256 overheadChargeForFundBackedSponsorships, uint256 overheadChargeForNetworkSponsorshipsWithTracking)
func (_FailingPostTxRegistry *FailingPostTxRegistryCallerSession) GetGasConfig() (struct {
	GasLimitForChooseFund                            *big.Int
	GasLimitForDeductFees                            *big.Int
	GasLimitForTrack                                 *big.Int
	OverheadChargeForFundBackedSponsorships          *big.Int
	OverheadChargeForNetworkSponsorshipsWithTracking *big.Int
}, error) {
	return _FailingPostTxRegistry.Contract.GetGasConfig(&_FailingPostTxRegistry.CallOpts)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) pure returns()
func (_FailingPostTxRegistry *FailingPostTxRegistryCaller) Track(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _FailingPostTxRegistry.contract.Call(opts, &out, "track", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) pure returns()
func (_FailingPostTxRegistry *FailingPostTxRegistrySession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _FailingPostTxRegistry.Contract.Track(&_FailingPostTxRegistry.CallOpts, arg0, arg1)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) pure returns()
func (_FailingPostTxRegistry *FailingPostTxRegistryCallerSession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _FailingPostTxRegistry.Contract.Track(&_FailingPostTxRegistry.CallOpts, arg0, arg1)
}
