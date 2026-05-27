// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package sponsor_everything

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

// SponsorEverythingMetaData contains all meta data concerning the SponsorEverything contract.
var SponsorEverythingMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"payload\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadCharge\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"trackGasCost\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f80fd5b506109318061001d5f395ff3fe60806040526004361061006f575f3560e01c80639ec88e991161004d5780639ec88e991461011a578063b9ed9f2614610136578063bf70eb151461015e578063fecb2bc3146101865761006f565b8063399f59ca146100735780634b5c54c0146100b057806351ee41a0146100dd575b5f80fd5b34801561007e575f80fd5b506100996004803603810190610094919061055a565b6101c3565b6040516100a792919061062b565b60405180910390f35b3480156100bb575f80fd5b506100c46101ef565b6040516100d49493929190610652565b60405180910390f35b3480156100e8575f80fd5b5061010360048036038101906100fe9190610695565b610228565b6040516101119291906106da565b60405180910390f35b610134600480360381019061012f919061072b565b610238565b005b348015610141575f80fd5b5061015c60048036038101906101579190610756565b6102d7565b005b348015610169575f80fd5b50610184600480360381019061017f9190610756565b61040a565b005b348015610191575f80fd5b506101ac60048036038101906101a7919061072b565b610445565b6040516101ba929190610794565b60405180910390f35b5f808247106101da576001805f1b915091506101e3565b5f805f1b915091505b97509795505050505050565b5f805f805f61c35090506212d68794506209fbf1935080848661021291906107e8565b61021c91906107e8565b92505f91505090919293565b5f806001805f1b91509150915091565b5f805f8381526020019081526020015f20905034815f015f82825461025d91906107e8565b9250508190555034816002015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546102b291906107e8565b9250508190555034816001015f8282546102cc91906107e8565b925050819055505050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461030e575f80fd5b5f801b8203610352576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161034990610875565b60405180910390fd5b80471015610395576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161038c906108dd565b60405180910390fd5b73fc00face0000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663850a10c0826040518263ffffffff1660e01b81526004015f604051808303818588803b1580156103ef575f80fd5b505af1158015610401573d5f803e3d5ffd5b50505050505050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610441575f80fd5b5050565b5f602052805f5260405f205f91509050805f0154908060010154905082565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6104958261046c565b9050919050565b6104a58161048b565b81146104af575f80fd5b50565b5f813590506104c08161049c565b92915050565b5f819050919050565b6104d8816104c6565b81146104e2575f80fd5b50565b5f813590506104f3816104cf565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f84011261051a576105196104f9565b5b8235905067ffffffffffffffff811115610537576105366104fd565b5b60208301915083600182028301111561055357610552610501565b5b9250929050565b5f805f805f805f60c0888a03121561057557610574610464565b5b5f6105828a828b016104b2565b97505060206105938a828b016104b2565b96505060406105a48a828b016104e5565b95505060606105b58a828b016104e5565b945050608088013567ffffffffffffffff8111156105d6576105d5610468565b5b6105e28a828b01610505565b935093505060a06105f58a828b016104e5565b91505092959891949750929550565b61060d816104c6565b82525050565b5f819050919050565b61062581610613565b82525050565b5f60408201905061063e5f830185610604565b61064b602083018461061c565b9392505050565b5f6080820190506106655f830187610604565b6106726020830186610604565b61067f6040830185610604565b61068c6060830184610604565b95945050505050565b5f602082840312156106aa576106a9610464565b5b5f6106b7848285016104b2565b91505092915050565b5f8115159050919050565b6106d4816106c0565b82525050565b5f6040820190506106ed5f8301856106cb565b6106fa602083018461061c565b9392505050565b61070a81610613565b8114610714575f80fd5b50565b5f8135905061072581610701565b92915050565b5f602082840312156107405761073f610464565b5b5f61074d84828501610717565b91505092915050565b5f806040838503121561076c5761076b610464565b5b5f61077985828601610717565b925050602061078a858286016104e5565b9150509250929050565b5f6040820190506107a75f830185610604565b6107b46020830184610604565b9392505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6107f2826104c6565b91506107fd836104c6565b9250828201905080821115610815576108146107bb565b5b92915050565b5f82825260208201905092915050565b7f4e6f2073706f6e736f72736869702066756e642063686f73656e0000000000005f82015250565b5f61085f601a8361081b565b915061086a8261082b565b602082019050919050565b5f6020820190508181035f83015261088c81610853565b9050919050565b7f4e6f7420656e6f7567682066756e6473000000000000000000000000000000005f82015250565b5f6108c760108361081b565b91506108d282610893565b602082019050919050565b5f6020820190508181035f8301526108f4816108bb565b905091905056fea2646970667358221220a1ee4a60d0ad8f97db5332b9e5901630bc5b0058901e58532c75dd35e96a335f64736f6c63430008180033",
}

// SponsorEverythingABI is the input ABI used to generate the binding from.
// Deprecated: Use SponsorEverythingMetaData.ABI instead.
var SponsorEverythingABI = SponsorEverythingMetaData.ABI

// SponsorEverythingBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SponsorEverythingMetaData.Bin instead.
var SponsorEverythingBin = SponsorEverythingMetaData.Bin

// DeploySponsorEverything deploys a new Ethereum contract, binding an instance of SponsorEverything to it.
func DeploySponsorEverything(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SponsorEverything, error) {
	parsed, err := SponsorEverythingMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SponsorEverythingBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SponsorEverything{SponsorEverythingCaller: SponsorEverythingCaller{contract: contract}, SponsorEverythingTransactor: SponsorEverythingTransactor{contract: contract}, SponsorEverythingFilterer: SponsorEverythingFilterer{contract: contract}}, nil
}

// SponsorEverything is an auto generated Go binding around an Ethereum contract.
type SponsorEverything struct {
	SponsorEverythingCaller     // Read-only binding to the contract
	SponsorEverythingTransactor // Write-only binding to the contract
	SponsorEverythingFilterer   // Log filterer for contract events
}

// SponsorEverythingCaller is an auto generated read-only Go binding around an Ethereum contract.
type SponsorEverythingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SponsorEverythingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SponsorEverythingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SponsorEverythingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SponsorEverythingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SponsorEverythingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SponsorEverythingSession struct {
	Contract     *SponsorEverything // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// SponsorEverythingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SponsorEverythingCallerSession struct {
	Contract *SponsorEverythingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// SponsorEverythingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SponsorEverythingTransactorSession struct {
	Contract     *SponsorEverythingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// SponsorEverythingRaw is an auto generated low-level Go binding around an Ethereum contract.
type SponsorEverythingRaw struct {
	Contract *SponsorEverything // Generic contract binding to access the raw methods on
}

// SponsorEverythingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SponsorEverythingCallerRaw struct {
	Contract *SponsorEverythingCaller // Generic read-only contract binding to access the raw methods on
}

// SponsorEverythingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SponsorEverythingTransactorRaw struct {
	Contract *SponsorEverythingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSponsorEverything creates a new instance of SponsorEverything, bound to a specific deployed contract.
func NewSponsorEverything(address common.Address, backend bind.ContractBackend) (*SponsorEverything, error) {
	contract, err := bindSponsorEverything(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SponsorEverything{SponsorEverythingCaller: SponsorEverythingCaller{contract: contract}, SponsorEverythingTransactor: SponsorEverythingTransactor{contract: contract}, SponsorEverythingFilterer: SponsorEverythingFilterer{contract: contract}}, nil
}

// NewSponsorEverythingCaller creates a new read-only instance of SponsorEverything, bound to a specific deployed contract.
func NewSponsorEverythingCaller(address common.Address, caller bind.ContractCaller) (*SponsorEverythingCaller, error) {
	contract, err := bindSponsorEverything(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SponsorEverythingCaller{contract: contract}, nil
}

// NewSponsorEverythingTransactor creates a new write-only instance of SponsorEverything, bound to a specific deployed contract.
func NewSponsorEverythingTransactor(address common.Address, transactor bind.ContractTransactor) (*SponsorEverythingTransactor, error) {
	contract, err := bindSponsorEverything(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SponsorEverythingTransactor{contract: contract}, nil
}

// NewSponsorEverythingFilterer creates a new log filterer instance of SponsorEverything, bound to a specific deployed contract.
func NewSponsorEverythingFilterer(address common.Address, filterer bind.ContractFilterer) (*SponsorEverythingFilterer, error) {
	contract, err := bindSponsorEverything(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SponsorEverythingFilterer{contract: contract}, nil
}

// bindSponsorEverything binds a generic wrapper to an already deployed contract.
func bindSponsorEverything(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SponsorEverythingMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SponsorEverything *SponsorEverythingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SponsorEverything.Contract.SponsorEverythingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SponsorEverything *SponsorEverythingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SponsorEverything.Contract.SponsorEverythingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SponsorEverything *SponsorEverythingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SponsorEverything.Contract.SponsorEverythingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SponsorEverything *SponsorEverythingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SponsorEverything.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SponsorEverything *SponsorEverythingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SponsorEverything.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SponsorEverything *SponsorEverythingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SponsorEverything.Contract.contract.Transact(opts, method, params...)
}

// AccountSponsorshipFundId is a free data retrieval call binding the contract method 0x51ee41a0.
//
// Solidity: function accountSponsorshipFundId(address ) pure returns(bool, bytes32)
func (_SponsorEverything *SponsorEverythingCaller) AccountSponsorshipFundId(opts *bind.CallOpts, arg0 common.Address) (bool, [32]byte, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "accountSponsorshipFundId", arg0)

	if err != nil {
		return *new(bool), *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	out1 := *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return out0, out1, err

}

// AccountSponsorshipFundId is a free data retrieval call binding the contract method 0x51ee41a0.
//
// Solidity: function accountSponsorshipFundId(address ) pure returns(bool, bytes32)
func (_SponsorEverything *SponsorEverythingSession) AccountSponsorshipFundId(arg0 common.Address) (bool, [32]byte, error) {
	return _SponsorEverything.Contract.AccountSponsorshipFundId(&_SponsorEverything.CallOpts, arg0)
}

// AccountSponsorshipFundId is a free data retrieval call binding the contract method 0x51ee41a0.
//
// Solidity: function accountSponsorshipFundId(address ) pure returns(bool, bytes32)
func (_SponsorEverything *SponsorEverythingCallerSession) AccountSponsorshipFundId(arg0 common.Address) (bool, [32]byte, error) {
	return _SponsorEverything.Contract.AccountSponsorshipFundId(&_SponsorEverything.CallOpts, arg0)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(uint256 mode, bytes32 payload)
func (_SponsorEverything *SponsorEverythingCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, fee)

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
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(uint256 mode, bytes32 payload)
func (_SponsorEverything *SponsorEverythingSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _SponsorEverything.Contract.ChooseFund(&_SponsorEverything.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(uint256 mode, bytes32 payload)
func (_SponsorEverything *SponsorEverythingCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) (struct {
	Mode    *big.Int
	Payload [32]byte
}, error) {
	return _SponsorEverything.Contract.ChooseFund(&_SponsorEverything.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_SponsorEverything *SponsorEverythingCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "getGasConfig")

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
func (_SponsorEverything *SponsorEverythingSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	return _SponsorEverything.Contract.GetGasConfig(&_SponsorEverything.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge, uint256 trackGasCost)
func (_SponsorEverything *SponsorEverythingCallerSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
	TrackGasCost    *big.Int
}, error) {
	return _SponsorEverything.Contract.GetGasConfig(&_SponsorEverything.CallOpts)
}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 id) view returns(uint256 funds, uint256 totalContributions)
func (_SponsorEverything *SponsorEverythingCaller) Sponsorships(opts *bind.CallOpts, id [32]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "sponsorships", id)

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
func (_SponsorEverything *SponsorEverythingSession) Sponsorships(id [32]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _SponsorEverything.Contract.Sponsorships(&_SponsorEverything.CallOpts, id)
}

// Sponsorships is a free data retrieval call binding the contract method 0xfecb2bc3.
//
// Solidity: function sponsorships(bytes32 id) view returns(uint256 funds, uint256 totalContributions)
func (_SponsorEverything *SponsorEverythingCallerSession) Sponsorships(id [32]byte) (struct {
	Funds              *big.Int
	TotalContributions *big.Int
}, error) {
	return _SponsorEverything.Contract.Sponsorships(&_SponsorEverything.CallOpts, id)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) view returns()
func (_SponsorEverything *SponsorEverythingCaller) Track(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) error {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "track", arg0, arg1)

	if err != nil {
		return err
	}

	return err

}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) view returns()
func (_SponsorEverything *SponsorEverythingSession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _SponsorEverything.Contract.Track(&_SponsorEverything.CallOpts, arg0, arg1)
}

// Track is a free data retrieval call binding the contract method 0xbf70eb15.
//
// Solidity: function track(bytes32 , uint256 ) view returns()
func (_SponsorEverything *SponsorEverythingCallerSession) Track(arg0 [32]byte, arg1 *big.Int) error {
	return _SponsorEverything.Contract.Track(&_SponsorEverything.CallOpts, arg0, arg1)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_SponsorEverything *SponsorEverythingTransactor) DeductFees(opts *bind.TransactOpts, fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _SponsorEverything.contract.Transact(opts, "deductFees", fundId, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_SponsorEverything *SponsorEverythingSession) DeductFees(fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _SponsorEverything.Contract.DeductFees(&_SponsorEverything.TransactOpts, fundId, fee)
}

// DeductFees is a paid mutator transaction binding the contract method 0xb9ed9f26.
//
// Solidity: function deductFees(bytes32 fundId, uint256 fee) returns()
func (_SponsorEverything *SponsorEverythingTransactorSession) DeductFees(fundId [32]byte, fee *big.Int) (*types.Transaction, error) {
	return _SponsorEverything.Contract.DeductFees(&_SponsorEverything.TransactOpts, fundId, fee)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_SponsorEverything *SponsorEverythingTransactor) Sponsor(opts *bind.TransactOpts, fundId [32]byte) (*types.Transaction, error) {
	return _SponsorEverything.contract.Transact(opts, "sponsor", fundId)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_SponsorEverything *SponsorEverythingSession) Sponsor(fundId [32]byte) (*types.Transaction, error) {
	return _SponsorEverything.Contract.Sponsor(&_SponsorEverything.TransactOpts, fundId)
}

// Sponsor is a paid mutator transaction binding the contract method 0x9ec88e99.
//
// Solidity: function sponsor(bytes32 fundId) payable returns()
func (_SponsorEverything *SponsorEverythingTransactorSession) Sponsor(fundId [32]byte) (*types.Transaction, error) {
	return _SponsorEverything.Contract.Sponsor(&_SponsorEverything.TransactOpts, fundId)
}
