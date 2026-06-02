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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"accountSponsorshipFundId\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"chooseFund\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"mode\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"deductFees\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getGasConfig\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chooseFundLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deductFeesLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"traceLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"fundBackedOverheadCharge\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"networkTrackedOverheadCharge\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"fundId\",\"type\":\"bytes32\"}],\"name\":\"sponsor\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"sponsorships\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"funds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalContributions\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f80fd5b506109068061001c5f395ff3fe608060405260043610610054575f3560e01c8063399f59ca146100585780634b5c54c01461009557806351ee41a0146100c35780639ec88e9914610100578063b9ed9f261461011c578063fecb2bc314610144575b5f80fd5b348015610063575f80fd5b5061007e600480360381019061007991906104fb565b610181565b60405161008c9291906105cc565b60405180910390f35b3480156100a0575f80fd5b506100a96101ad565b6040516100ba9594939291906105f3565b60405180910390f35b3480156100ce575f80fd5b506100e960048036038101906100e49190610644565b610204565b6040516100f7929190610689565b60405180910390f35b61011a600480360381019061011591906106da565b610214565b005b348015610127575f80fd5b50610142600480360381019061013d9190610705565b6102b3565b005b34801561014f575f80fd5b5061016a600480360381019061016591906106da565b6103e6565b604051610178929190610743565b60405180910390f35b5f80824710610198576001805f1b915091506101a1565b5f805f1b915091505b97509795505050505050565b5f805f805f8061c35090506212d68795506209fbf19450620acc7b93508085876101d79190610797565b6101e19190610797565b92508084876101f09190610797565b6101fa9190610797565b9150509091929394565b5f806001805f1b91509150915091565b5f805f8381526020019081526020015f20905034815f015f8282546102399190610797565b9250508190555034816002015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f82825461028e9190610797565b9250508190555034816001015f8282546102a89190610797565b925050819055505050565b5f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146102ea575f80fd5b5f801b820361032e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161032590610824565b60405180910390fd5b80471015610371576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103689061088c565b60405180910390fd5b73fc00face0000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663850a10c0826040518263ffffffff1660e01b81526004015f604051808303818588803b1580156103cb575f80fd5b505af11580156103dd573d5f803e3d5ffd5b50505050505050565b5f602052805f5260405f205f91509050805f0154908060010154905082565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6104368261040d565b9050919050565b6104468161042c565b8114610450575f80fd5b50565b5f813590506104618161043d565b92915050565b5f819050919050565b61047981610467565b8114610483575f80fd5b50565b5f8135905061049481610470565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f8401126104bb576104ba61049a565b5b8235905067ffffffffffffffff8111156104d8576104d761049e565b5b6020830191508360018202830111156104f4576104f36104a2565b5b9250929050565b5f805f805f805f60c0888a03121561051657610515610405565b5b5f6105238a828b01610453565b97505060206105348a828b01610453565b96505060406105458a828b01610486565b95505060606105568a828b01610486565b945050608088013567ffffffffffffffff81111561057757610576610409565b5b6105838a828b016104a6565b935093505060a06105968a828b01610486565b91505092959891949750929550565b6105ae81610467565b82525050565b5f819050919050565b6105c6816105b4565b82525050565b5f6040820190506105df5f8301856105a5565b6105ec60208301846105bd565b9392505050565b5f60a0820190506106065f8301886105a5565b61061360208301876105a5565b61062060408301866105a5565b61062d60608301856105a5565b61063a60808301846105a5565b9695505050505050565b5f6020828403121561065957610658610405565b5b5f61066684828501610453565b91505092915050565b5f8115159050919050565b6106838161066f565b82525050565b5f60408201905061069c5f83018561067a565b6106a960208301846105bd565b9392505050565b6106b9816105b4565b81146106c3575f80fd5b50565b5f813590506106d4816106b0565b92915050565b5f602082840312156106ef576106ee610405565b5b5f6106fc848285016106c6565b91505092915050565b5f806040838503121561071b5761071a610405565b5b5f610728858286016106c6565b925050602061073985828601610486565b9150509250929050565b5f6040820190506107565f8301856105a5565b61076360208301846105a5565b9392505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6107a182610467565b91506107ac83610467565b92508282019050808211156107c4576107c361076a565b5b92915050565b5f82825260208201905092915050565b7f4e6f2073706f6e736f72736869702066756e642063686f73656e0000000000005f82015250565b5f61080e601a836107ca565b9150610819826107da565b602082019050919050565b5f6020820190508181035f83015261083b81610802565b9050919050565b7f4e6f7420656e6f7567682066756e6473000000000000000000000000000000005f82015250565b5f6108766010836107ca565b915061088182610842565b602082019050919050565b5f6020820190508181035f8301526108a38161086a565b905091905056fea2646970667358221220ad124e2ad445550b526fe56511d9e8af06b04cd0e30a9f3f918b000c4744458364736f6c637828302e382e32352d646576656c6f702e323032342e322e32342b636f6d6d69742e64626137353465630059",
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
