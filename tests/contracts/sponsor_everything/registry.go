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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"overheadCharge\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506108888061001c5f395ff3fe608060405260043610610054575f3560e01c8063399f59ca146100585780634b5c54c01461009457806351ee41a0146100c05780639ec88e99146100fd578063b9ed9f2614610119578063fecb2bc314610141575b5f5ffd5b348015610063575f5ffd5b5061007e600480360381019061007991906104cd565b61017e565b60405161008b919061058f565b60405180910390f35b34801561009f575f5ffd5b506100a86101a2565b6040516100b7939291906105b7565b60405180910390f35b3480156100cb575f5ffd5b506100e660048036038101906100e191906105ec565b6101d6565b6040516100f4929190610631565b60405180910390f35b61011760048036038101906101129190610682565b6101e6565b005b348015610124575f5ffd5b5061013f600480360381019061013a91906106ad565b610285565b005b34801561014c575f5ffd5b5061016760048036038101906101629190610682565b6103b8565b6040516101759291906106eb565b60405180910390f35b5f8147106101915760015f1b9050610197565b5f5f1b90505b979650505050505050565b5f5f5f5f61c35090506212d68793506209fbf192508083856101c4919061073f565b6101ce919061073f565b915050909192565b5f5f6001805f1b91509150915091565b5f5f5f8381526020019081526020015f20905034815f015f82825461020b919061073f565b9250508190555034816002015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f828254610260919061073f565b9250508190555034816001015f82825461027a919061073f565b925050819055505050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146102bc575f5ffd5b5f5f1b8203610300576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102f7906107cc565b60405180910390fd5b80471015610343576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161033a90610834565b60405180910390fd5b73fc00face0000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663850a10c0826040518263ffffffff1660e01b81526004015f604051808303818588803b15801561039d575f5ffd5b505af11580156103af573d5f5f3e3d5ffd5b50505050505050565b5f602052805f5260405f205f91509050805f0154908060010154905082565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610408826103df565b9050919050565b610418816103fe565b8114610422575f5ffd5b50565b5f813590506104338161040f565b92915050565b5f819050919050565b61044b81610439565b8114610455575f5ffd5b50565b5f8135905061046681610442565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261048d5761048c61046c565b5b8235905067ffffffffffffffff8111156104aa576104a9610470565b5b6020830191508360018202830111156104c6576104c5610474565b5b9250929050565b5f5f5f5f5f5f5f60c0888a0312156104e8576104e76103d7565b5b5f6104f58a828b01610425565b97505060206105068a828b01610425565b96505060406105178a828b01610458565b95505060606105288a828b01610458565b945050608088013567ffffffffffffffff811115610549576105486103db565b5b6105558a828b01610478565b935093505060a06105688a828b01610458565b91505092959891949750929550565b5f819050919050565b61058981610577565b82525050565b5f6020820190506105a25f830184610580565b92915050565b6105b181610439565b82525050565b5f6060820190506105ca5f8301866105a8565b6105d760208301856105a8565b6105e460408301846105a8565b949350505050565b5f60208284031215610601576106006103d7565b5b5f61060e84828501610425565b91505092915050565b5f8115159050919050565b61062b81610617565b82525050565b5f6040820190506106445f830185610622565b6106516020830184610580565b9392505050565b61066181610577565b811461066b575f5ffd5b50565b5f8135905061067c81610658565b92915050565b5f60208284031215610697576106966103d7565b5b5f6106a48482850161066e565b91505092915050565b5f5f604083850312156106c3576106c26103d7565b5b5f6106d08582860161066e565b92505060206106e185828601610458565b9150509250929050565b5f6040820190506106fe5f8301856105a8565b61070b60208301846105a8565b9392505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61074982610439565b915061075483610439565b925082820190508082111561076c5761076b610712565b5b92915050565b5f82825260208201905092915050565b7f4e6f2073706f6e736f72736869702066756e642063686f73656e0000000000005f82015250565b5f6107b6601a83610772565b91506107c182610782565b602082019050919050565b5f6020820190508181035f8301526107e3816107aa565b9050919050565b7f4e6f7420656e6f7567682066756e6473000000000000000000000000000000005f82015250565b5f61081e601083610772565b9150610829826107ea565b602082019050919050565b5f6020820190508181035f83015261084b81610812565b905091905056fea2646970667358221220c0b8ceba4026befc5f7d3a75a3015e34ec7b952bf463ce7360d2f605d26fe20264736f6c634300081b0033",
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
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(bytes32 fundId)
func (_SponsorEverything *SponsorEverythingCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, fee)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(bytes32 fundId)
func (_SponsorEverything *SponsorEverythingSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) ([32]byte, error) {
	return _SponsorEverything.Contract.ChooseFund(&_SponsorEverything.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(bytes32 fundId)
func (_SponsorEverything *SponsorEverythingCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) ([32]byte, error) {
	return _SponsorEverything.Contract.ChooseFund(&_SponsorEverything.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_SponsorEverything *SponsorEverythingCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "getGasConfig")

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
func (_SponsorEverything *SponsorEverythingSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
}, error) {
	return _SponsorEverything.Contract.GetGasConfig(&_SponsorEverything.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 overheadCharge)
func (_SponsorEverything *SponsorEverythingCallerSession) GetGasConfig() (struct {
	ChooseFundLimit *big.Int
	DeductFeesLimit *big.Int
	OverheadCharge  *big.Int
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
