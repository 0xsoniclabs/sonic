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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"traceLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fundBackedOverheadCharge\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"networkTrackedOverheadCharge\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"track\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b506109848061001c5f395ff3fe60806040526004361061006f575f3560e01c80639ec88e991161004d5780639ec88e991461011b578063b9ed9f2614610137578063bf70eb151461015f578063fecb2bc3146101875761006f565b8063399f59ca146100735780634b5c54c0146100b057806351ee41a0146100de575b5f80fd5b34801561007e575f80fd5b5061009960048036038101906100949190610579565b6101c4565b6040516100a792919061064a565b60405180910390f35b3480156100bb575f80fd5b506100c46101f0565b6040516100d5959493929190610671565b60405180910390f35b3480156100e9575f80fd5b5061010460048036038101906100ff91906106c2565b610247565b604051610112929190610707565b60405180910390f35b61013560048036038101906101309190610758565b610257565b005b348015610142575f80fd5b5061015d60048036038101906101589190610783565b6102f6565b005b34801561016a575f80fd5b5061018560048036038101906101809190610783565b610429565b005b348015610192575f80fd5b506101ad60048036038101906101a89190610758565b610464565b6040516101bb9291906107c1565b60405180910390f35b5f808247106101db576001805f1b915091506101e4565b5f805f1b915091505b97509795505050505050565b5f805f805f8061c35090506212d68795506209fbf19450620acc7b935080858761021a9190610815565b6102249190610815565b92508084876102339190610815565b61023d9190610815565b9150509091929394565b5f806001805f1b91509150915091565b5f805f8381526020019081526020015f20905034815f015f82825461027c9190610815565b9250508190555034816002015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546102d19190610815565b9250508190555034816001015f8282546102eb9190610815565b925050819055505050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461032d575f80fd5b5f801b8203610371576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610368906108a2565b60405180910390fd5b804710156103b4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103ab9061090a565b60405180910390fd5b73fc00face0000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663850a10c0826040518263ffffffff1660e01b81526004015f604051808303818588803b15801561040e575f80fd5b505af1158015610420573d5f803e3d5ffd5b50505050505050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610460575f80fd5b5050565b5f602052805f5260405f205f91509050805f0154908060010154905082565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6104b48261048b565b9050919050565b6104c4816104aa565b81146104ce575f80fd5b50565b5f813590506104df816104bb565b92915050565b5f819050919050565b6104f7816104e5565b8114610501575f80fd5b50565b5f81359050610512816104ee565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f84011261053957610538610518565b5b8235905067ffffffffffffffff8111156105565761055561051c565b5b60208301915083600182028301111561057257610571610520565b5b9250929050565b5f805f805f805f60c0888a03121561059457610593610483565b5b5f6105a18a828b016104d1565b97505060206105b28a828b016104d1565b96505060406105c38a828b01610504565b95505060606105d48a828b01610504565b945050608088013567ffffffffffffffff8111156105f5576105f4610487565b5b6106018a828b01610524565b935093505060a06106148a828b01610504565b91505092959891949750929550565b61062c816104e5565b82525050565b5f819050919050565b61064481610632565b82525050565b5f60408201905061065d5f830185610623565b61066a602083018461063b565b9392505050565b5f60a0820190506106845f830188610623565b6106916020830187610623565b61069e6040830186610623565b6106ab6060830185610623565b6106b86080830184610623565b9695505050505050565b5f602082840312156106d7576106d6610483565b5b5f6106e4848285016104d1565b91505092915050565b5f8115159050919050565b610701816106ed565b82525050565b5f60408201905061071a5f8301856106f8565b610727602083018461063b565b9392505050565b61073781610632565b8114610741575f80fd5b50565b5f813590506107528161072e565b92915050565b5f6020828403121561076d5761076c610483565b5b5f61077a84828501610744565b91505092915050565b5f806040838503121561079957610798610483565b5b5f6107a685828601610744565b92505060206107b785828601610504565b9150509250929050565b5f6040820190506107d45f830185610623565b6107e16020830184610623565b9392505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61081f826104e5565b915061082a836104e5565b9250828201905080821115610842576108416107e8565b5b92915050565b5f82825260208201905092915050565b7f4e6f2073706f6e736f72736869702066756e642063686f73656e0000000000005f82015250565b5f61088c601a83610848565b915061089782610858565b602082019050919050565b5f6020820190508181035f8301526108b981610880565b9050919050565b7f4e6f7420656e6f7567682066756e6473000000000000000000000000000000005f82015250565b5f6108f4601083610848565b91506108ff826108c0565b602082019050919050565b5f6020820190508181035f830152610921816108e8565b905091905056fea2646970667358221220213e887d11c41d2eaec6ab89f268ff195f026a1e2b693b9ae080b0646b24acd664736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
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
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(uint256 mode, bytes32 fundId)
func (_SponsorEverything *SponsorEverythingCaller) ChooseFund(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) (struct {
	Mode   *big.Int
	FundId [32]byte
}, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "chooseFund", arg0, arg1, arg2, arg3, arg4, fee)

	outstruct := new(struct {
		Mode   *big.Int
		FundId [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Mode = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.FundId = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(uint256 mode, bytes32 fundId)
func (_SponsorEverything *SponsorEverythingSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) (struct {
	Mode   *big.Int
	FundId [32]byte
}, error) {
	return _SponsorEverything.Contract.ChooseFund(&_SponsorEverything.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// ChooseFund is a free data retrieval call binding the contract method 0x399f59ca.
//
// Solidity: function chooseFund(address , address , uint256 , uint256 , bytes , uint256 fee) view returns(uint256 mode, bytes32 fundId)
func (_SponsorEverything *SponsorEverythingCallerSession) ChooseFund(arg0 common.Address, arg1 common.Address, arg2 *big.Int, arg3 *big.Int, arg4 []byte, fee *big.Int) (struct {
	Mode   *big.Int
	FundId [32]byte
}, error) {
	return _SponsorEverything.Contract.ChooseFund(&_SponsorEverything.CallOpts, arg0, arg1, arg2, arg3, arg4, fee)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 traceLimit, uint256 fundBackedOverheadCharge, uint256 networkTrackedOverheadCharge)
func (_SponsorEverything *SponsorEverythingCaller) GetGasConfig(opts *bind.CallOpts) (struct {
	ChooseFundLimit              *big.Int
	DeductFeesLimit              *big.Int
	TraceLimit                   *big.Int
	FundBackedOverheadCharge     *big.Int
	NetworkTrackedOverheadCharge *big.Int
}, error) {
	var out []interface{}
	err := _SponsorEverything.contract.Call(opts, &out, "getGasConfig")

	outstruct := new(struct {
		ChooseFundLimit              *big.Int
		DeductFeesLimit              *big.Int
		TraceLimit                   *big.Int
		FundBackedOverheadCharge     *big.Int
		NetworkTrackedOverheadCharge *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ChooseFundLimit = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.DeductFeesLimit = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TraceLimit = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.FundBackedOverheadCharge = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.NetworkTrackedOverheadCharge = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 traceLimit, uint256 fundBackedOverheadCharge, uint256 networkTrackedOverheadCharge)
func (_SponsorEverything *SponsorEverythingSession) GetGasConfig() (struct {
	ChooseFundLimit              *big.Int
	DeductFeesLimit              *big.Int
	TraceLimit                   *big.Int
	FundBackedOverheadCharge     *big.Int
	NetworkTrackedOverheadCharge *big.Int
}, error) {
	return _SponsorEverything.Contract.GetGasConfig(&_SponsorEverything.CallOpts)
}

// GetGasConfig is a free data retrieval call binding the contract method 0x4b5c54c0.
//
// Solidity: function getGasConfig() pure returns(uint256 chooseFundLimit, uint256 deductFeesLimit, uint256 traceLimit, uint256 fundBackedOverheadCharge, uint256 networkTrackedOverheadCharge)
func (_SponsorEverything *SponsorEverythingCallerSession) GetGasConfig() (struct {
	ChooseFundLimit              *big.Int
	DeductFeesLimit              *big.Int
	TraceLimit                   *big.Int
	FundBackedOverheadCharge     *big.Int
	NetworkTrackedOverheadCharge *big.Int
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
