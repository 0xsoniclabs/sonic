// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package blsContracts

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

// BigNumber is an auto generated low-level Go binding around an user-defined struct.
type BigNumber struct {
	Val    []byte
	Neg    bool
	Bitlen *big.Int
}

// ElementsElement is an auto generated low-level Go binding around an user-defined struct.
type ElementsElement struct {
	Val [6]uint64
}

// BLSMetaData contains all meta data concerning the BLS contract.
var BLSMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubKeys\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"CheckAggregatedSignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubKey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"CheckAndUpdate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubKeys\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"CheckAndUpdateAggregatedSignature\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubKey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"CheckSignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"EncodeToG2\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Signature\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610f538061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610060575f3560e01c806317103ecb1461006457806331041a5a146100945780636ad25191146100b0578063c835350b146100e0578063cb70dcf5146100fc578063f94238571461012c575b5f5ffd5b61007e60048036038101906100799190610723565b61014a565b60405161008b91906107ca565b60405180910390f35b6100ae60048036038101906100a991906107ea565b6101cc565b005b6100ca60048036038101906100c591906107ea565b6101fc565b6040516100d791906108a8565b60405180910390f35b6100fa60048036038101906100f591906107ea565b610391565b005b610116600480360381019061011191906107ea565b6103c1565b60405161012391906108a8565b60405180910390f35b61013461054b565b60405161014191906107ca565b60405180910390f35b606073__$eac59e696449672e66084f1d7861c0ae8b$__6317103ecb836040518263ffffffff1660e01b81526004016101839190610909565b5f60405180830381865af415801561019d573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906101c59190610997565b9050919050565b5f6101d88484846101fc565b905060011515811515036101f657825f90816101f49190610be4565b505b50505050565b5f5f6080855161020c9190610ce0565b1461024c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161024390610d6a565b60405180910390fd5b610100835114610291576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161028890610dd2565b60405180910390fd5b5f73__$eac59e696449672e66084f1d7861c0ae8b$__6317103ecb846040518263ffffffff1660e01b81526004016102c99190610909565b5f60405180830381865af41580156102e3573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061030b9190610997565b905073__$eac59e696449672e66084f1d7861c0ae8b$__636ad251918686846040518463ffffffff1660e01b815260040161034893929190610df0565b602060405180830381865af4158015610363573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103879190610e64565b9150509392505050565b5f61039d8484846103c1565b905060011515811515036103bb57825f90816103b99190610be4565b505b50505050565b5f6080845114610406576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103fd90610ed9565b60405180910390fd5b61010083511461044b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161044290610dd2565b60405180910390fd5b5f73__$eac59e696449672e66084f1d7861c0ae8b$__6317103ecb846040518263ffffffff1660e01b81526004016104839190610909565b5f60405180830381865af415801561049d573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906104c59190610997565b905073__$eac59e696449672e66084f1d7861c0ae8b$__63cb70dcf58686846040518463ffffffff1660e01b815260040161050293929190610df0565b602060405180830381865af415801561051d573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906105419190610e64565b9150509392505050565b5f805461055790610a0b565b80601f016020809104026020016040519081016040528092919081815260200182805461058390610a0b565b80156105ce5780601f106105a5576101008083540402835291602001916105ce565b820191905f5260205f20905b8154815290600101906020018083116105b157829003601f168201915b505050505081565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610635826105ef565b810181811067ffffffffffffffff82111715610654576106536105ff565b5b80604052505050565b5f6106666105d6565b9050610672828261062c565b919050565b5f67ffffffffffffffff821115610691576106906105ff565b5b61069a826105ef565b9050602081019050919050565b828183375f83830152505050565b5f6106c76106c284610677565b61065d565b9050828152602081018484840111156106e3576106e26105eb565b5b6106ee8482856106a7565b509392505050565b5f82601f83011261070a576107096105e7565b5b813561071a8482602086016106b5565b91505092915050565b5f60208284031215610738576107376105df565b5b5f82013567ffffffffffffffff811115610755576107546105e3565b5b610761848285016106f6565b91505092915050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f61079c8261076a565b6107a68185610774565b93506107b6818560208601610784565b6107bf816105ef565b840191505092915050565b5f6020820190508181035f8301526107e28184610792565b905092915050565b5f5f5f60608486031215610801576108006105df565b5b5f84013567ffffffffffffffff81111561081e5761081d6105e3565b5b61082a868287016106f6565b935050602084013567ffffffffffffffff81111561084b5761084a6105e3565b5b610857868287016106f6565b925050604084013567ffffffffffffffff811115610878576108776105e3565b5b610884868287016106f6565b9150509250925092565b5f8115159050919050565b6108a28161088e565b82525050565b5f6020820190506108bb5f830184610899565b92915050565b5f82825260208201905092915050565b5f6108db8261076a565b6108e581856108c1565b93506108f5818560208601610784565b6108fe816105ef565b840191505092915050565b5f6020820190508181035f83015261092181846108d1565b905092915050565b5f61093b61093684610677565b61065d565b905082815260208101848484011115610957576109566105eb565b5b610962848285610784565b509392505050565b5f82601f83011261097e5761097d6105e7565b5b815161098e848260208601610929565b91505092915050565b5f602082840312156109ac576109ab6105df565b5b5f82015167ffffffffffffffff8111156109c9576109c86105e3565b5b6109d58482850161096a565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f6002820490506001821680610a2257607f821691505b602082108103610a3557610a346109de565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f60088302610a977fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82610a5c565b610aa18683610a5c565b95508019841693508086168417925050509392505050565b5f819050919050565b5f819050919050565b5f610ae5610ae0610adb84610ab9565b610ac2565b610ab9565b9050919050565b5f819050919050565b610afe83610acb565b610b12610b0a82610aec565b848454610a68565b825550505050565b5f5f905090565b610b29610b1a565b610b34818484610af5565b505050565b5b81811015610b5757610b4c5f82610b21565b600181019050610b3a565b5050565b601f821115610b9c57610b6d81610a3b565b610b7684610a4d565b81016020851015610b85578190505b610b99610b9185610a4d565b830182610b39565b50505b505050565b5f82821c905092915050565b5f610bbc5f1984600802610ba1565b1980831691505092915050565b5f610bd48383610bad565b9150826002028217905092915050565b610bed8261076a565b67ffffffffffffffff811115610c0657610c056105ff565b5b610c108254610a0b565b610c1b828285610b5b565b5f60209050601f831160018114610c4c575f8415610c3a578287015190505b610c448582610bc9565b865550610cab565b601f198416610c5a86610a3b565b5f5b82811015610c8157848901518255600182019150602085019450602081019050610c5c565b86831015610c9e5784890151610c9a601f891682610bad565b8355505b6001600288020188555050505b505050505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601260045260245ffd5b5f610cea82610ab9565b9150610cf583610ab9565b925082610d0557610d04610cb3565b5b828206905092915050565b5f82825260208201905092915050565b7f496e76616c6964207075626c6963206b657973206c656e6774680000000000005f82015250565b5f610d54601a83610d10565b9150610d5f82610d20565b602082019050919050565b5f6020820190508181035f830152610d8181610d48565b9050919050565b7f496e76616c6964207369676e6174757265206c656e67746800000000000000005f82015250565b5f610dbc601883610d10565b9150610dc782610d88565b602082019050919050565b5f6020820190508181035f830152610de981610db0565b9050919050565b5f6060820190508181035f830152610e0881866108d1565b90508181036020830152610e1c81856108d1565b90508181036040830152610e3081846108d1565b9050949350505050565b610e438161088e565b8114610e4d575f5ffd5b50565b5f81519050610e5e81610e3a565b92915050565b5f60208284031215610e7957610e786105df565b5b5f610e8684828501610e50565b91505092915050565b7f496e76616c6964207075626c6963206b6579206c656e677468000000000000005f82015250565b5f610ec3601983610d10565b9150610ece82610e8f565b602082019050919050565b5f6020820190508181035f830152610ef081610eb7565b905091905056fea2646970667358221220c02e5c017bcd551f54ce4f4c70991f2c0139ed5ae1b8bf40657dc51ce216da6164736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
}

// BLSABI is the input ABI used to generate the binding from.
// Deprecated: Use BLSMetaData.ABI instead.
var BLSABI = BLSMetaData.ABI

// BLSBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BLSMetaData.Bin instead.
var BLSBin = BLSMetaData.Bin

// DeployBLS deploys a new Ethereum contract, binding an instance of BLS to it.
func DeployBLS(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BLS, error) {
	parsed, err := BLSMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	bLSLibraryAddr, _, _, _ := DeployBLSLibrary(auth, backend)
	BLSBin = strings.ReplaceAll(BLSBin, "__$eac59e696449672e66084f1d7861c0ae8b$__", bLSLibraryAddr.String()[2:])

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BLSBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BLS{BLSCaller: BLSCaller{contract: contract}, BLSTransactor: BLSTransactor{contract: contract}, BLSFilterer: BLSFilterer{contract: contract}}, nil
}

// BLS is an auto generated Go binding around an Ethereum contract.
type BLS struct {
	BLSCaller     // Read-only binding to the contract
	BLSTransactor // Write-only binding to the contract
	BLSFilterer   // Log filterer for contract events
}

// BLSCaller is an auto generated read-only Go binding around an Ethereum contract.
type BLSCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BLSTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BLSTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BLSFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BLSFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BLSSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BLSSession struct {
	Contract     *BLS              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BLSCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BLSCallerSession struct {
	Contract *BLSCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BLSTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BLSTransactorSession struct {
	Contract     *BLSTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BLSRaw is an auto generated low-level Go binding around an Ethereum contract.
type BLSRaw struct {
	Contract *BLS // Generic contract binding to access the raw methods on
}

// BLSCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BLSCallerRaw struct {
	Contract *BLSCaller // Generic read-only contract binding to access the raw methods on
}

// BLSTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BLSTransactorRaw struct {
	Contract *BLSTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBLS creates a new instance of BLS, bound to a specific deployed contract.
func NewBLS(address common.Address, backend bind.ContractBackend) (*BLS, error) {
	contract, err := bindBLS(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BLS{BLSCaller: BLSCaller{contract: contract}, BLSTransactor: BLSTransactor{contract: contract}, BLSFilterer: BLSFilterer{contract: contract}}, nil
}

// NewBLSCaller creates a new read-only instance of BLS, bound to a specific deployed contract.
func NewBLSCaller(address common.Address, caller bind.ContractCaller) (*BLSCaller, error) {
	contract, err := bindBLS(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BLSCaller{contract: contract}, nil
}

// NewBLSTransactor creates a new write-only instance of BLS, bound to a specific deployed contract.
func NewBLSTransactor(address common.Address, transactor bind.ContractTransactor) (*BLSTransactor, error) {
	contract, err := bindBLS(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BLSTransactor{contract: contract}, nil
}

// NewBLSFilterer creates a new log filterer instance of BLS, bound to a specific deployed contract.
func NewBLSFilterer(address common.Address, filterer bind.ContractFilterer) (*BLSFilterer, error) {
	contract, err := bindBLS(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BLSFilterer{contract: contract}, nil
}

// bindBLS binds a generic wrapper to an already deployed contract.
func bindBLS(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BLSMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BLS *BLSRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BLS.Contract.BLSCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BLS *BLSRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BLS.Contract.BLSTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BLS *BLSRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BLS.Contract.BLSTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BLS *BLSCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BLS.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BLS *BLSTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BLS.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BLS *BLSTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BLS.Contract.contract.Transact(opts, method, params...)
}

// CheckAggregatedSignature is a free data retrieval call binding the contract method 0x6ad25191.
//
// Solidity: function CheckAggregatedSignature(bytes pubKeys, bytes signature, bytes message) view returns(bool)
func (_BLS *BLSCaller) CheckAggregatedSignature(opts *bind.CallOpts, pubKeys []byte, signature []byte, message []byte) (bool, error) {
	var out []interface{}
	err := _BLS.contract.Call(opts, &out, "CheckAggregatedSignature", pubKeys, signature, message)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CheckAggregatedSignature is a free data retrieval call binding the contract method 0x6ad25191.
//
// Solidity: function CheckAggregatedSignature(bytes pubKeys, bytes signature, bytes message) view returns(bool)
func (_BLS *BLSSession) CheckAggregatedSignature(pubKeys []byte, signature []byte, message []byte) (bool, error) {
	return _BLS.Contract.CheckAggregatedSignature(&_BLS.CallOpts, pubKeys, signature, message)
}

// CheckAggregatedSignature is a free data retrieval call binding the contract method 0x6ad25191.
//
// Solidity: function CheckAggregatedSignature(bytes pubKeys, bytes signature, bytes message) view returns(bool)
func (_BLS *BLSCallerSession) CheckAggregatedSignature(pubKeys []byte, signature []byte, message []byte) (bool, error) {
	return _BLS.Contract.CheckAggregatedSignature(&_BLS.CallOpts, pubKeys, signature, message)
}

// CheckSignature is a free data retrieval call binding the contract method 0xcb70dcf5.
//
// Solidity: function CheckSignature(bytes pubKey, bytes signature, bytes message) view returns(bool)
func (_BLS *BLSCaller) CheckSignature(opts *bind.CallOpts, pubKey []byte, signature []byte, message []byte) (bool, error) {
	var out []interface{}
	err := _BLS.contract.Call(opts, &out, "CheckSignature", pubKey, signature, message)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CheckSignature is a free data retrieval call binding the contract method 0xcb70dcf5.
//
// Solidity: function CheckSignature(bytes pubKey, bytes signature, bytes message) view returns(bool)
func (_BLS *BLSSession) CheckSignature(pubKey []byte, signature []byte, message []byte) (bool, error) {
	return _BLS.Contract.CheckSignature(&_BLS.CallOpts, pubKey, signature, message)
}

// CheckSignature is a free data retrieval call binding the contract method 0xcb70dcf5.
//
// Solidity: function CheckSignature(bytes pubKey, bytes signature, bytes message) view returns(bool)
func (_BLS *BLSCallerSession) CheckSignature(pubKey []byte, signature []byte, message []byte) (bool, error) {
	return _BLS.Contract.CheckSignature(&_BLS.CallOpts, pubKey, signature, message)
}

// EncodeToG2 is a free data retrieval call binding the contract method 0x17103ecb.
//
// Solidity: function EncodeToG2(bytes message) view returns(bytes)
func (_BLS *BLSCaller) EncodeToG2(opts *bind.CallOpts, message []byte) ([]byte, error) {
	var out []interface{}
	err := _BLS.contract.Call(opts, &out, "EncodeToG2", message)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// EncodeToG2 is a free data retrieval call binding the contract method 0x17103ecb.
//
// Solidity: function EncodeToG2(bytes message) view returns(bytes)
func (_BLS *BLSSession) EncodeToG2(message []byte) ([]byte, error) {
	return _BLS.Contract.EncodeToG2(&_BLS.CallOpts, message)
}

// EncodeToG2 is a free data retrieval call binding the contract method 0x17103ecb.
//
// Solidity: function EncodeToG2(bytes message) view returns(bytes)
func (_BLS *BLSCallerSession) EncodeToG2(message []byte) ([]byte, error) {
	return _BLS.Contract.EncodeToG2(&_BLS.CallOpts, message)
}

// Signature is a free data retrieval call binding the contract method 0xf9423857.
//
// Solidity: function Signature() view returns(bytes)
func (_BLS *BLSCaller) Signature(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _BLS.contract.Call(opts, &out, "Signature")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Signature is a free data retrieval call binding the contract method 0xf9423857.
//
// Solidity: function Signature() view returns(bytes)
func (_BLS *BLSSession) Signature() ([]byte, error) {
	return _BLS.Contract.Signature(&_BLS.CallOpts)
}

// Signature is a free data retrieval call binding the contract method 0xf9423857.
//
// Solidity: function Signature() view returns(bytes)
func (_BLS *BLSCallerSession) Signature() ([]byte, error) {
	return _BLS.Contract.Signature(&_BLS.CallOpts)
}

// CheckAndUpdate is a paid mutator transaction binding the contract method 0xc835350b.
//
// Solidity: function CheckAndUpdate(bytes pubKey, bytes signature, bytes message) returns()
func (_BLS *BLSTransactor) CheckAndUpdate(opts *bind.TransactOpts, pubKey []byte, signature []byte, message []byte) (*types.Transaction, error) {
	return _BLS.contract.Transact(opts, "CheckAndUpdate", pubKey, signature, message)
}

// CheckAndUpdate is a paid mutator transaction binding the contract method 0xc835350b.
//
// Solidity: function CheckAndUpdate(bytes pubKey, bytes signature, bytes message) returns()
func (_BLS *BLSSession) CheckAndUpdate(pubKey []byte, signature []byte, message []byte) (*types.Transaction, error) {
	return _BLS.Contract.CheckAndUpdate(&_BLS.TransactOpts, pubKey, signature, message)
}

// CheckAndUpdate is a paid mutator transaction binding the contract method 0xc835350b.
//
// Solidity: function CheckAndUpdate(bytes pubKey, bytes signature, bytes message) returns()
func (_BLS *BLSTransactorSession) CheckAndUpdate(pubKey []byte, signature []byte, message []byte) (*types.Transaction, error) {
	return _BLS.Contract.CheckAndUpdate(&_BLS.TransactOpts, pubKey, signature, message)
}

// CheckAndUpdateAggregatedSignature is a paid mutator transaction binding the contract method 0x31041a5a.
//
// Solidity: function CheckAndUpdateAggregatedSignature(bytes pubKeys, bytes signature, bytes message) returns()
func (_BLS *BLSTransactor) CheckAndUpdateAggregatedSignature(opts *bind.TransactOpts, pubKeys []byte, signature []byte, message []byte) (*types.Transaction, error) {
	return _BLS.contract.Transact(opts, "CheckAndUpdateAggregatedSignature", pubKeys, signature, message)
}

// CheckAndUpdateAggregatedSignature is a paid mutator transaction binding the contract method 0x31041a5a.
//
// Solidity: function CheckAndUpdateAggregatedSignature(bytes pubKeys, bytes signature, bytes message) returns()
func (_BLS *BLSSession) CheckAndUpdateAggregatedSignature(pubKeys []byte, signature []byte, message []byte) (*types.Transaction, error) {
	return _BLS.Contract.CheckAndUpdateAggregatedSignature(&_BLS.TransactOpts, pubKeys, signature, message)
}

// CheckAndUpdateAggregatedSignature is a paid mutator transaction binding the contract method 0x31041a5a.
//
// Solidity: function CheckAndUpdateAggregatedSignature(bytes pubKeys, bytes signature, bytes message) returns()
func (_BLS *BLSTransactorSession) CheckAndUpdateAggregatedSignature(pubKeys []byte, signature []byte, message []byte) (*types.Transaction, error) {
	return _BLS.Contract.CheckAndUpdateAggregatedSignature(&_BLS.TransactOpts, pubKeys, signature, message)
}

// BLSLibraryMetaData contains all meta data concerning the BLSLibrary contract.
var BLSLibraryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubKeys\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"messageHash\",\"type\":\"bytes\"}],\"name\":\"CheckAggregatedSignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"pubKey\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"messageHash\",\"type\":\"bytes\"}],\"name\":\"CheckSignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"EncodeToG2\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6121bb61004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe730000000000000000000000000000000000000000301460806040526004361061004a575f3560e01c806317103ecb1461004e5780636ad251911461007e578063cb70dcf5146100ae575b5f5ffd5b610068600480360381019061006391906111fe565b6100de565b60405161007591906112a5565b60405180910390f35b610098600480360381019061009391906112c5565b6102a5565b6040516100a59190611383565b60405180910390f35b6100c860048036038101906100c391906112c5565b6104f8565b6040516100d59190611383565b60405180910390f35b60605f6100ea836105d7565b905060605f5f5b83518160ff16101561029157828283868460ff16815181106101165761011561139c565b5b60200260200101515f01516005600681106101345761013361139c565b5b6020020151878560ff168151811061014f5761014e61139c565b5b60200260200101515f015160046006811061016d5761016c61139c565b5b6020020151888660ff16815181106101885761018761139c565b5b60200260200101515f01516003600681106101a6576101a561139c565b5b6020020151898760ff16815181106101c1576101c061139c565b5b60200260200101515f01516002600681106101df576101de61139c565b5b60200201518a8860ff16815181106101fa576101f961139c565b5b60200260200101515f01516001600681106102185761021761139c565b5b60200201518b8960ff16815181106102335761023261139c565b5b60200260200101515f01515f600681106102505761024f61139c565b5b602002015160405160200161026d9998979695949392919061144a565b6040516020818303038152906040529250808061028990611521565b9150506100f1565b5061029b826108bf565b9350505050919050565b5f5f608085516102b5919061157f565b146102f5576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102ec90611609565b60405180910390fd5b61010083511461033a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161033190611671565b60405180910390fd5b5f6040518060a00160405280608081526020016120b0608091398460405160200161036692919061168f565b60405160208183030381529060405290505f5f90505b6080865161038a91906116b2565b811015610455575f608067ffffffffffffffff8111156103ad576103ac6110da565b5b6040519080825280601f01601f1916602001820160405280156103df5781602001600182028036833780820191505090505b5090505f6080836103f091906116e2565b9050602082018160208a010180518252602081015160208301526040810151604083015260608101516060830152505083828760405160200161043593929190611723565b60405160208183030381529060405293505050808060010191505061037c565b505f61046082610975565b905060208151146104a6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161049d9061179d565b60405180910390fd5b5f60f81b81601f815181106104be576104bd61139c565b5b602001015160f81c60f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19161415925050509392505050565b5f5f6105406040518060a00160405280608081526020016120b06080913985878660405160200161052c94939291906117bb565b604051602081830303815290604052610975565b90506020815114610586576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161057d9061179d565b60405180910390fd5b5f60f81b81601f8151811061059e5761059d61139c565b5b602001015160f81c60f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff191614159150509392505050565b60605f610632836040516020016105ee91906117f8565b60405160208183030381529060405260405180602001604052805f81525060405160200161061c91906117f8565b6040516020818303038152906040526002610a2b565b90505f815103610677576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161066e90611858565b60405180910390fd5b5f600261ffff1667ffffffffffffffff811115610697576106966110da565b5b6040519080825280602002602001820160405280156106d057816020015b6106bd611055565b8152602001906001900390816106b55790505b5090506106db61106e565b73__$49e1895db7bc958afca1a7fe6c92641d94$__63f33d5856604051806060016040528060308152602001612130603091395f6040518363ffffffff1660e01b815260040161072c929190611876565b5f60405180830381865af4158015610746573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061076e91906119eb565b90505f5f90505b600261ffff168110156108b3575f604067ffffffffffffffff81111561079e5761079d6110da565b5b6040519080825280601f01601f1916602001820160405280156107d05781602001600182028036833780820191505090505b5090505f6040836107e191906116e2565b905060208201816020880101805182526020810151602083015250505f73__$49e1895db7bc958afca1a7fe6c92641d94$__63f33d5856845f6040518363ffffffff1660e01b8152600401610837929190611876565b5f60405180830381865af4158015610851573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061087991906119eb565b90506108858186610d0d565b8685815181106108985761089761139c565b5b60200260200101819052505050508080600101915050610775565b50819350505050919050565b60605f5f601173ffffffffffffffffffffffffffffffffffffffff16846040516108e991906117f8565b5f60405180830381855afa9150503d805f8114610921576040519150601f19603f3d011682016040523d82523d5f602084013e610926565b606091505b50915091508161096b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161096290611a7c565b60405180910390fd5b8092505050919050565b60605f5f600f73ffffffffffffffffffffffffffffffffffffffff168460405161099f91906117f8565b5f60405180830381855afa9150503d805f81146109d7576040519150601f19603f3d011682016040523d82523d5f602084013e6109dc565b606091505b509150915081610a21576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a1890611ae4565b60405180910390fd5b8092505050919050565b60605f835114610a70576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a6790611b4c565b60405180910390fd5b5f826040610a7e9190611b77565b90505f6020601f83610a909190611bb3565b610a9a9190611be8565b61ffff1690505f5f5f600190505f600284858c898788604051602001610ac596959493929190611ca9565b604051602081830303815290604052604051610ae191906117f8565b602060405180830381855afa158015610afc573d5f5f3e3d5ffd5b5050506040513d601f19601f82011682018060405250810190610b1f9190611d3e565b90505f6002828486604051602001610b3993929190611d69565b604051602081830303815290604052604051610b5591906117f8565b602060405180830381855afa158015610b70573d5f5f3e3d5ffd5b5050506040513d601f19601f82011682018060405250810190610b939190611d3e565b905060028b604051602001610ba891906117f8565b604051602081830303815290604052604051610bc491906117f8565b602060405180830381855afa158015610bdf573d5f5f3e3d5ffd5b5050506040513d601f19601f82011682018060405250810190610c029190611d3e565b8103610c0c575f5ffd5b5f81604051602001610c1e9190611da5565b60405160208183030381529060405290505f5f600290505b8867ffffffffffffffff168160ff1611610cf95783851891506002828289604051602001610c6693929190611d69565b604051602081830303815290604052604051610c8291906117f8565b602060405180830381855afa158015610c9d573d5f5f3e3d5ffd5b5050506040513d601f19601f82011682018060405250810190610cc09190611d3e565b93508284604051602001610cd5929190611dbf565b60405160208183030381529060405292508080610cf190611521565b915050610c36565b508199505050505050505050509392505050565b610d15611055565b5f73__$49e1895db7bc958afca1a7fe6c92641d94$__63bc1b392d6040518163ffffffff1660e01b81526004015f60405180830381865af4158015610d5c573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f82011682018060405250810190610d8491906119eb565b90505f8473__$49e1895db7bc958afca1a7fe6c92641d94$__63bc39804e9091865f6040518463ffffffff1660e01b8152600401610dc493929190611e99565b602060405180830381865af4158015610ddf573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610e039190611f0f565b90505f8103610e1f57610e14611055565b80935050505061104f565b60018114158015610ecb57507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8573__$49e1895db7bc958afca1a7fe6c92641d94$__63bc39804e9091855f6040518463ffffffff1660e01b8152600401610e8993929190611e99565b602060405180830381865af4158015610ea4573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610ec89190611f0f565b14155b15610f525773__$9f9e0af0b1d8aaef18d08a09a93a4cbce4$__634350ee1f865f01516040518263ffffffff1660e01b8152600401610f0a91906112a5565b60c060405180830381865af4158015610f25573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610f49919061204f565b9250505061104f565b8473__$49e1895db7bc958afca1a7fe6c92641d94$__637afa85539091866040518363ffffffff1660e01b8152600401610f8d92919061207a565b5f60405180830381865af4158015610fa7573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f82011682018060405250810190610fcf91906119eb565b945073__$9f9e0af0b1d8aaef18d08a09a93a4cbce4$__634350ee1f865f01516040518263ffffffff1660e01b815260040161100b91906112a5565b60c060405180830381865af4158015611026573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061104a919061204f565b925050505b92915050565b604051806020016040528061106861108f565b81525090565b6040518060600160405280606081526020015f151581526020015f81525090565b6040518060c00160405280600690602082028036833780820191505090505090565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b611110826110ca565b810181811067ffffffffffffffff8211171561112f5761112e6110da565b5b80604052505050565b5f6111416110b1565b905061114d8282611107565b919050565b5f67ffffffffffffffff82111561116c5761116b6110da565b5b611175826110ca565b9050602081019050919050565b828183375f83830152505050565b5f6111a261119d84611152565b611138565b9050828152602081018484840111156111be576111bd6110c6565b5b6111c9848285611182565b509392505050565b5f82601f8301126111e5576111e46110c2565b5b81356111f5848260208601611190565b91505092915050565b5f60208284031215611213576112126110ba565b5b5f82013567ffffffffffffffff8111156112305761122f6110be565b5b61123c848285016111d1565b91505092915050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f61127782611245565b611281818561124f565b935061129181856020860161125f565b61129a816110ca565b840191505092915050565b5f6020820190508181035f8301526112bd818461126d565b905092915050565b5f5f5f606084860312156112dc576112db6110ba565b5b5f84013567ffffffffffffffff8111156112f9576112f86110be565b5b611305868287016111d1565b935050602084013567ffffffffffffffff811115611326576113256110be565b5b611332868287016111d1565b925050604084013567ffffffffffffffff811115611353576113526110be565b5b61135f868287016111d1565b9150509250925092565b5f8115159050919050565b61137d81611369565b82525050565b5f6020820190506113965f830184611374565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f81905092915050565b5f6113dd82611245565b6113e781856113c9565b93506113f781856020860161125f565b80840191505092915050565b5f67ffffffffffffffff82169050919050565b5f8160c01b9050919050565b5f61142c82611416565b9050919050565b61144461143f82611403565b611422565b82525050565b5f611455828c6113d3565b9150611461828b611433565b600882019150611471828a611433565b6008820191506114818289611433565b6008820191506114918288611433565b6008820191506114a18287611433565b6008820191506114b18286611433565b6008820191506114c18285611433565b6008820191506114d18284611433565b6008820191508190509a9950505050505050505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f60ff82169050919050565b5f61152b82611515565b915060ff820361153e5761153d6114e8565b5b600182019050919050565b5f819050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601260045260245ffd5b5f61158982611549565b915061159483611549565b9250826115a4576115a3611552565b5b828206905092915050565b5f82825260208201905092915050565b7f496e76616c6964207075626c6963206b657973206c656e6774680000000000005f82015250565b5f6115f3601a836115af565b91506115fe826115bf565b602082019050919050565b5f6020820190508181035f830152611620816115e7565b9050919050565b7f496e76616c6964207369676e6174757265206c656e67746800000000000000005f82015250565b5f61165b6018836115af565b915061166682611627565b602082019050919050565b5f6020820190508181035f8301526116888161164f565b9050919050565b5f61169a82856113d3565b91506116a682846113d3565b91508190509392505050565b5f6116bc82611549565b91506116c783611549565b9250826116d7576116d6611552565b5b828204905092915050565b5f6116ec82611549565b91506116f783611549565b925082820261170581611549565b9150828204841483151761171c5761171b6114e8565b5b5092915050565b5f61172e82866113d3565b915061173a82856113d3565b915061174682846113d3565b9150819050949350505050565b7f496e76616c696420726573756c74206c656e67746800000000000000000000005f82015250565b5f6117876015836115af565b915061179282611753565b602082019050919050565b5f6020820190508181035f8301526117b48161177b565b9050919050565b5f6117c682876113d3565b91506117d282866113d3565b91506117de82856113d3565b91506117ea82846113d3565b915081905095945050505050565b5f61180382846113d3565b915081905092915050565b7f4e6f2064617461000000000000000000000000000000000000000000000000005f82015250565b5f6118426007836115af565b915061184d8261180e565b602082019050919050565b5f6020820190508181035f83015261186f81611836565b9050919050565b5f6040820190508181035f83015261188e818561126d565b905061189d6020830184611374565b9392505050565b5f5ffd5b5f5ffd5b5f6118be6118b984611152565b611138565b9050828152602081018484840111156118da576118d96110c6565b5b6118e584828561125f565b509392505050565b5f82601f830112611901576119006110c2565b5b81516119118482602086016118ac565b91505092915050565b61192381611369565b811461192d575f5ffd5b50565b5f8151905061193e8161191a565b92915050565b61194d81611549565b8114611957575f5ffd5b50565b5f8151905061196881611944565b92915050565b5f60608284031215611983576119826118a4565b5b61198d6060611138565b90505f82015167ffffffffffffffff8111156119ac576119ab6118a8565b5b6119b8848285016118ed565b5f8301525060206119cb84828501611930565b60208301525060406119df8482850161195a565b60408301525092915050565b5f60208284031215611a00576119ff6110ba565b5b5f82015167ffffffffffffffff811115611a1d57611a1c6110be565b5b611a298482850161196e565b91505092915050565b7f424c53206d6170546f47322063616c6c206661696c65640000000000000000005f82015250565b5f611a666017836115af565b9150611a7182611a32565b602082019050919050565b5f6020820190508181035f830152611a9381611a5a565b9050919050565b7f424c5320706169722063616c6c206661696c65640000000000000000000000005f82015250565b5f611ace6014836115af565b9150611ad982611a9a565b602082019050919050565b5f6020820190508181035f830152611afb81611ac2565b9050919050565b7f445354206e6f7420696d706c656d656e746564207965740000000000000000005f82015250565b5f611b366017836115af565b9150611b4182611b02565b602082019050919050565b5f6020820190508181035f830152611b6381611b2a565b9050919050565b5f61ffff82169050919050565b5f611b8182611b6a565b9150611b8c83611b6a565b9250828202611b9a81611b6a565b9150808214611bac57611bab6114e8565b5b5092915050565b5f611bbd82611b6a565b9150611bc883611b6a565b9250828201905061ffff811115611be257611be16114e8565b5b92915050565b5f611bf282611b6a565b9150611bfd83611b6a565b925082611c0d57611c0c611552565b5b828204905092915050565b5f819050919050565b5f819050919050565b611c3b611c3682611c18565b611c21565b82525050565b5f8160f01b9050919050565b5f611c5782611c41565b9050919050565b611c6f611c6a82611b6a565b611c4d565b82525050565b5f8160f81b9050919050565b5f611c8b82611c75565b9050919050565b611ca3611c9e82611515565b611c81565b82525050565b5f611cb48289611c2a565b602082019150611cc48288611c2a565b602082019150611cd482876113d3565b9150611ce08286611c5e565b600282019150611cf08285611c92565b600182019150611d008284611c92565b600182019150819050979650505050505050565b611d1d81611c18565b8114611d27575f5ffd5b50565b5f81519050611d3881611d14565b92915050565b5f60208284031215611d5357611d526110ba565b5b5f611d6084828501611d2a565b91505092915050565b5f611d748286611c2a565b602082019150611d848285611c92565b600182019150611d948284611c92565b600182019150819050949350505050565b5f611db08284611c2a565b60208201915081905092915050565b5f611dca82856113d3565b9150611dd68284611c2a565b6020820191508190509392505050565b5f82825260208201905092915050565b5f611e0082611245565b611e0a8185611de6565b9350611e1a81856020860161125f565b611e23816110ca565b840191505092915050565b611e3781611369565b82525050565b611e4681611549565b82525050565b5f606083015f8301518482035f860152611e668282611df6565b9150506020830151611e7b6020860182611e2e565b506040830151611e8e6040860182611e3d565b508091505092915050565b5f6060820190508181035f830152611eb18186611e4c565b90508181036020830152611ec58185611e4c565b9050611ed46040830184611374565b949350505050565b5f819050919050565b611eee81611edc565b8114611ef8575f5ffd5b50565b5f81519050611f0981611ee5565b92915050565b5f60208284031215611f2457611f236110ba565b5b5f611f3184828501611efb565b91505092915050565b5f67ffffffffffffffff821115611f5457611f536110da565b5b602082029050919050565b5f5ffd5b611f6c81611403565b8114611f76575f5ffd5b50565b5f81519050611f8781611f63565b92915050565b5f611f9f611f9a84611f3a565b611138565b90508060208402830185811115611fb957611fb8611f5f565b5b835b81811015611fe25780611fce8882611f79565b845260208401935050602081019050611fbb565b5050509392505050565b5f82601f83011261200057611fff6110c2565b5b600661200d848285611f8d565b91505092915050565b5f60c0828403121561202b5761202a6118a4565b5b6120356020611138565b90505f61204484828501611fec565b5f8301525092915050565b5f60c08284031215612064576120636110ba565b5b5f61207184828501612016565b91505092915050565b5f6040820190508181035f8301526120928185611e4c565b905081810360208301526120a68184611e4c565b9050939250505056fe0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb00000000000000000000000000000000114d1d6855d545a8aa7d76c8cf2e21f267816aef1db507c96655b9d5caac42364e6f38ba0ecb751bad54dcd6b939c2ca1a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaaba2646970667358221220edbb47ed207486d7dae2856651c3d81241ea51093f465b9ac72ba55e6086205264736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
}

// BLSLibraryABI is the input ABI used to generate the binding from.
// Deprecated: Use BLSLibraryMetaData.ABI instead.
var BLSLibraryABI = BLSLibraryMetaData.ABI

// BLSLibraryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BLSLibraryMetaData.Bin instead.
var BLSLibraryBin = BLSLibraryMetaData.Bin

// DeployBLSLibrary deploys a new Ethereum contract, binding an instance of BLSLibrary to it.
func DeployBLSLibrary(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BLSLibrary, error) {
	parsed, err := BLSLibraryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	bigNumbersAddr, _, _, _ := DeployBigNumbers(auth, backend)
	BLSLibraryBin = strings.ReplaceAll(BLSLibraryBin, "__$49e1895db7bc958afca1a7fe6c92641d94$__", bigNumbersAddr.String()[2:])

	elementsAddr, _, _, _ := DeployElements(auth, backend)
	BLSLibraryBin = strings.ReplaceAll(BLSLibraryBin, "__$9f9e0af0b1d8aaef18d08a09a93a4cbce4$__", elementsAddr.String()[2:])

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BLSLibraryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BLSLibrary{BLSLibraryCaller: BLSLibraryCaller{contract: contract}, BLSLibraryTransactor: BLSLibraryTransactor{contract: contract}, BLSLibraryFilterer: BLSLibraryFilterer{contract: contract}}, nil
}

// BLSLibrary is an auto generated Go binding around an Ethereum contract.
type BLSLibrary struct {
	BLSLibraryCaller     // Read-only binding to the contract
	BLSLibraryTransactor // Write-only binding to the contract
	BLSLibraryFilterer   // Log filterer for contract events
}

// BLSLibraryCaller is an auto generated read-only Go binding around an Ethereum contract.
type BLSLibraryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BLSLibraryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BLSLibraryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BLSLibraryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BLSLibraryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BLSLibrarySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BLSLibrarySession struct {
	Contract     *BLSLibrary       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BLSLibraryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BLSLibraryCallerSession struct {
	Contract *BLSLibraryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// BLSLibraryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BLSLibraryTransactorSession struct {
	Contract     *BLSLibraryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// BLSLibraryRaw is an auto generated low-level Go binding around an Ethereum contract.
type BLSLibraryRaw struct {
	Contract *BLSLibrary // Generic contract binding to access the raw methods on
}

// BLSLibraryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BLSLibraryCallerRaw struct {
	Contract *BLSLibraryCaller // Generic read-only contract binding to access the raw methods on
}

// BLSLibraryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BLSLibraryTransactorRaw struct {
	Contract *BLSLibraryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBLSLibrary creates a new instance of BLSLibrary, bound to a specific deployed contract.
func NewBLSLibrary(address common.Address, backend bind.ContractBackend) (*BLSLibrary, error) {
	contract, err := bindBLSLibrary(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BLSLibrary{BLSLibraryCaller: BLSLibraryCaller{contract: contract}, BLSLibraryTransactor: BLSLibraryTransactor{contract: contract}, BLSLibraryFilterer: BLSLibraryFilterer{contract: contract}}, nil
}

// NewBLSLibraryCaller creates a new read-only instance of BLSLibrary, bound to a specific deployed contract.
func NewBLSLibraryCaller(address common.Address, caller bind.ContractCaller) (*BLSLibraryCaller, error) {
	contract, err := bindBLSLibrary(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BLSLibraryCaller{contract: contract}, nil
}

// NewBLSLibraryTransactor creates a new write-only instance of BLSLibrary, bound to a specific deployed contract.
func NewBLSLibraryTransactor(address common.Address, transactor bind.ContractTransactor) (*BLSLibraryTransactor, error) {
	contract, err := bindBLSLibrary(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BLSLibraryTransactor{contract: contract}, nil
}

// NewBLSLibraryFilterer creates a new log filterer instance of BLSLibrary, bound to a specific deployed contract.
func NewBLSLibraryFilterer(address common.Address, filterer bind.ContractFilterer) (*BLSLibraryFilterer, error) {
	contract, err := bindBLSLibrary(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BLSLibraryFilterer{contract: contract}, nil
}

// bindBLSLibrary binds a generic wrapper to an already deployed contract.
func bindBLSLibrary(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BLSLibraryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BLSLibrary *BLSLibraryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BLSLibrary.Contract.BLSLibraryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BLSLibrary *BLSLibraryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BLSLibrary.Contract.BLSLibraryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BLSLibrary *BLSLibraryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BLSLibrary.Contract.BLSLibraryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BLSLibrary *BLSLibraryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BLSLibrary.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BLSLibrary *BLSLibraryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BLSLibrary.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BLSLibrary *BLSLibraryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BLSLibrary.Contract.contract.Transact(opts, method, params...)
}

// CheckAggregatedSignature is a free data retrieval call binding the contract method 0x6ad25191.
//
// Solidity: function CheckAggregatedSignature(bytes pubKeys, bytes signature, bytes messageHash) view returns(bool)
func (_BLSLibrary *BLSLibraryCaller) CheckAggregatedSignature(opts *bind.CallOpts, pubKeys []byte, signature []byte, messageHash []byte) (bool, error) {
	var out []interface{}
	err := _BLSLibrary.contract.Call(opts, &out, "CheckAggregatedSignature", pubKeys, signature, messageHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CheckAggregatedSignature is a free data retrieval call binding the contract method 0x6ad25191.
//
// Solidity: function CheckAggregatedSignature(bytes pubKeys, bytes signature, bytes messageHash) view returns(bool)
func (_BLSLibrary *BLSLibrarySession) CheckAggregatedSignature(pubKeys []byte, signature []byte, messageHash []byte) (bool, error) {
	return _BLSLibrary.Contract.CheckAggregatedSignature(&_BLSLibrary.CallOpts, pubKeys, signature, messageHash)
}

// CheckAggregatedSignature is a free data retrieval call binding the contract method 0x6ad25191.
//
// Solidity: function CheckAggregatedSignature(bytes pubKeys, bytes signature, bytes messageHash) view returns(bool)
func (_BLSLibrary *BLSLibraryCallerSession) CheckAggregatedSignature(pubKeys []byte, signature []byte, messageHash []byte) (bool, error) {
	return _BLSLibrary.Contract.CheckAggregatedSignature(&_BLSLibrary.CallOpts, pubKeys, signature, messageHash)
}

// CheckSignature is a free data retrieval call binding the contract method 0xcb70dcf5.
//
// Solidity: function CheckSignature(bytes pubKey, bytes signature, bytes messageHash) view returns(bool)
func (_BLSLibrary *BLSLibraryCaller) CheckSignature(opts *bind.CallOpts, pubKey []byte, signature []byte, messageHash []byte) (bool, error) {
	var out []interface{}
	err := _BLSLibrary.contract.Call(opts, &out, "CheckSignature", pubKey, signature, messageHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CheckSignature is a free data retrieval call binding the contract method 0xcb70dcf5.
//
// Solidity: function CheckSignature(bytes pubKey, bytes signature, bytes messageHash) view returns(bool)
func (_BLSLibrary *BLSLibrarySession) CheckSignature(pubKey []byte, signature []byte, messageHash []byte) (bool, error) {
	return _BLSLibrary.Contract.CheckSignature(&_BLSLibrary.CallOpts, pubKey, signature, messageHash)
}

// CheckSignature is a free data retrieval call binding the contract method 0xcb70dcf5.
//
// Solidity: function CheckSignature(bytes pubKey, bytes signature, bytes messageHash) view returns(bool)
func (_BLSLibrary *BLSLibraryCallerSession) CheckSignature(pubKey []byte, signature []byte, messageHash []byte) (bool, error) {
	return _BLSLibrary.Contract.CheckSignature(&_BLSLibrary.CallOpts, pubKey, signature, messageHash)
}

// EncodeToG2 is a free data retrieval call binding the contract method 0x17103ecb.
//
// Solidity: function EncodeToG2(bytes message) view returns(bytes)
func (_BLSLibrary *BLSLibraryCaller) EncodeToG2(opts *bind.CallOpts, message []byte) ([]byte, error) {
	var out []interface{}
	err := _BLSLibrary.contract.Call(opts, &out, "EncodeToG2", message)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// EncodeToG2 is a free data retrieval call binding the contract method 0x17103ecb.
//
// Solidity: function EncodeToG2(bytes message) view returns(bytes)
func (_BLSLibrary *BLSLibrarySession) EncodeToG2(message []byte) ([]byte, error) {
	return _BLSLibrary.Contract.EncodeToG2(&_BLSLibrary.CallOpts, message)
}

// EncodeToG2 is a free data retrieval call binding the contract method 0x17103ecb.
//
// Solidity: function EncodeToG2(bytes message) view returns(bytes)
func (_BLSLibrary *BLSLibraryCallerSession) EncodeToG2(message []byte) ([]byte, error) {
	return _BLSLibrary.Contract.EncodeToG2(&_BLSLibrary.CallOpts, message)
}

// BigNumbersMetaData contains all meta data concerning the BigNumbers contract.
var BigNumbersMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"add\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"r\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"}],\"name\":\"bitLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"bitLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"r\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"a\",\"type\":\"uint256\"}],\"name\":\"bitLength\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"r\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"},{\"internalType\":\"bool\",\"name\":\"signed\",\"type\":\"bool\"}],\"name\":\"cmp\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"r\",\"type\":\"tuple\"}],\"name\":\"divVerify\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"eq\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"gt\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"gte\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"}],\"name\":\"hash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"h\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"}],\"name\":\"init\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"name\":\"init\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"}],\"name\":\"init\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"}],\"name\":\"isOdd\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"r\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"}],\"name\":\"isZero\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"a\",\"type\":\"bytes\"}],\"name\":\"isZero\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"lt\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"lte\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"n\",\"type\":\"tuple\"}],\"name\":\"mod\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"e\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"n\",\"type\":\"tuple\"}],\"name\":\"modexp\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"ai\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"e\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"n\",\"type\":\"tuple\"}],\"name\":\"modexp\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"n\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"r\",\"type\":\"tuple\"}],\"name\":\"modinvVerify\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"n\",\"type\":\"tuple\"}],\"name\":\"modmul\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"mul\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"r\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"one\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"e\",\"type\":\"uint256\"}],\"name\":\"pow\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"bits\",\"type\":\"uint256\"}],\"name\":\"shl\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"bits\",\"type\":\"uint256\"}],\"name\":\"shr\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"bn\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"bits\",\"type\":\"uint256\"}],\"name\":\"shrPrivate\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"a\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"b\",\"type\":\"tuple\"}],\"name\":\"sub\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"r\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"max\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"min\",\"type\":\"bytes\"}],\"name\":\"subPrivate\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"two\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"bn\",\"type\":\"tuple\"}],\"name\":\"verify\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"zero\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"val\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"neg\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"bitlen\",\"type\":\"uint256\"}],\"internalType\":\"structBigNumber\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x612eb261004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe730000000000000000000000000000000000000000301460806040526004361061020f575f3560e01c8063901717d111610123578063bc39804e116100b6578063eb11d32e11610085578063eb11d32e1461073a578063ecdafa1f1461076a578063ee4a4e4e1461079a578063f33d5856146107ca578063fad242a8146107fa5761020f565b8063bc39804e1461067a578063cb5c2316146106aa578063cd8e0ef7146106da578063e41bbcf81461070a5761020f565b8063a5d4ca5b116100f2578063a5d4ca5b146105cc578063aa7312f6146105fc578063b1d74ee91461062c578063bc1b392d1461065c5761020f565b8063901717d11461051d578063904a94991461053b578063969ecb171461056b57806398cdcfb21461059c5761020f565b80636efac6c7116101a65780637afa8553116101755780637afa85531461042d5780637b19cc9a1461045d57806380147a8e1461048d57806385056b16146104bd5780638b78e386146104ed5761020f565b80636efac6c71461036d57806371f839911461039d57806372864ff3146103cd57806375a01da5146103fd5761020f565b8063418ce38f116101e2578063418ce38f146102d357806344b41c37146103035780635bb65d1a146103335780635fdf05d71461034f5761020f565b806308d78767146102135780630a4638fd146102435780630e7671f21461027357806340957f72146102a3575b5f5ffd5b61022d60048036038101906102289190612502565b61082a565b60405161023a9190612558565b60405180910390f35b61025d60048036038101906102589190612571565b61086b565b60405161026a91906126b2565b60405180910390f35b61028d600480360381019061028891906126d2565b610aa5565b60405161029a91906126b2565b60405180910390f35b6102bd60048036038101906102b89190612776565b610b97565b6040516102ca91906126b2565b60405180910390f35b6102ed60048036038101906102e891906127b4565b610bd1565b6040516102fa91906126b2565b60405180910390f35b61031d60048036038101906103189190612502565b610d29565b60405161032a919061281d565b60405180910390f35b61034d60048036038101906103489190612502565b610d3d565b005b610357610db6565b60405161036491906126b2565b60405180910390f35b610387600480360381019061038291906127b4565b610df7565b60405161039491906126b2565b60405180910390f35b6103b760048036038101906103b29190612571565b610e1f565b6040516103c49190612558565b60405180910390f35b6103e760048036038101906103e29190612836565b610e72565b6040516103f491906126b2565b60405180910390f35b610417600480360381019061041291906127b4565b610f7b565b60405161042491906126b2565b60405180910390f35b61044760048036038101906104429190612571565b610fa8565b60405161045491906126b2565b60405180910390f35b6104776004803603810190610472919061290a565b610fca565b604051610484919061281d565b60405180910390f35b6104a760048036038101906104a29190612571565b611007565b6040516104b49190612558565b60405180910390f35b6104d760048036038101906104d29190612951565b61103b565b6040516104e491906126b2565b60405180910390f35b610507600480360381019061050291906126d2565b611057565b6040516105149190612558565b60405180910390f35b610525611265565b60405161053291906126b2565b60405180910390f35b61055560048036038101906105509190612502565b6112a6565b60405161056291906129d5565b60405180910390f35b610585600480360381019061058091906129ee565b6112b9565b604051610593929190612aac565b60405180910390f35b6105b660048036038101906105b19190612571565b6113df565b6040516105c391906126b2565b60405180910390f35b6105e660048036038101906105e191906126d2565b6115f1565b6040516105f391906126b2565b60405180910390f35b61061660048036038101906106119190612571565b611615565b6040516106239190612558565b60405180910390f35b61064660048036038101906106419190612571565b61163f565b60405161065391906126b2565b60405180910390f35b6106646116e0565b60405161067191906126b2565b60405180910390f35b610694600480360381019061068f9190612ada565b61171f565b6040516106a19190612b7a565b60405180910390f35b6106c460048036038101906106bf91906126d2565b6118f6565b6040516106d19190612558565b60405180910390f35b6106f460048036038101906106ef9190612571565b611945565b6040516107019190612558565b60405180910390f35b610724600480360381019061071f9190612b93565b61196e565b604051610731919061281d565b60405180910390f35b610754600480360381019061074f91906127b4565b611bc1565b60405161076191906126b2565b60405180910390f35b610784600480360381019061077f9190612571565b611be9565b6040516107919190612558565b60405180910390f35b6107b460048036038101906107af9190612502565b611c32565b6040516107c19190612558565b60405180910390f35b6107e460048036038101906107df9190612bbe565b611c46565b6040516107f191906126b2565b60405180910390f35b610814600480360381019061080f919061290a565b611c61565b6040516108219190612558565b60405180910390f35b5f610837825f0151611c61565b801561084757506020825f015151145b801561085557508160200151155b801561086457505f8260400151145b9050919050565b6108736122a7565b5f836040015114801561088957505f8260400151145b1561089d576108966116e0565b9050610a9f565b60605f5f6108ac86865f61171f565b91508560200151806108bf575084602001515b156109eb57856020015180156108d6575084602001515b1561097d5760018203610911576108f3865f0151865f01516112b9565b80925081945050506001846020019015159081151581525050610978565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361096557610948855f0151875f01516112b9565b80925081945050505f846020019015159081151581525050610977565b61096d6116e0565b9350505050610a9f565b5b6109e6565b5f82126109a657610999865f0151865f01518860400151611cb6565b80925081945050506109c4565b6109bb855f0151875f01518760400151611cb6565b80925081945050505b85602001516109d3575f6109d6565b60015b8460200190151590811515815250505b610a89565b60018203610a2057610a03865f0151865f01516112b9565b80925081945050505f846020019015159081151581525050610a88565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610a7557610a57855f0151875f01516112b9565b80925081945050506001846020019015159081151581525050610a87565b610a7d6116e0565b9350505050610a9f565b5b5b82845f0181905250808460400181815250505050505b92915050565b610aad6122a7565b5f151583602001511515148015610acb57505f151582602001511515145b8015610ae05750610ade825f0151611c61565b155b610ae8575f5ffd5b5f610afd855f0151855f0151855f0151611de3565b90505f610b0982610fca565b90505f8103610b2357610b1a6116e0565b92505050610b90565b85602001518015610b395750610b3885611c32565b5b15610b6e57610b65604051806060016040528084815260200160011515815260200183815250856113df565b92505050610b90565b60405180606001604052808381526020015f1515815260200182815250925050505b9392505050565b610b9f6122a7565b610bc983604051602001610bb39190612c38565b604051602081830303815290604052835f611eb9565b905092915050565b610bd96122a7565b5f835151905083604001518310610c245760405180606001604052806040518060400160405280602081526020015f81525081526020015f151581526020015f815250915050610d23565b828460400151610c349190612c7f565b846040018181525050610100831461010084111715610c655760206101008404028103905080845152610100830692505b83515f600885061460018114610cd2575f5f8661010003858501865b5f5f821403610cc8576020811460018114610ca25760208303519550610ca6565b5f95505b5081518a1c935084831b94508484178252602082039150602081039050610c81565b5050505050610cff565b6008850486518051602081018383018482038282848760046101c2fa5f601f840153848652505050505050505b50602081015f81511460018103610d1b57602083510382528187525b505050839150505b92915050565b5f610d36825f0151610fca565b9050919050565b5f5f825f01519050602081015191505f8203610d6957610d5c8361082a565b610d64575f5ffd5b610db1565b5f6020845f015151610d7b9190612cdf565b148015610da85750600161010060018560400151610d999190612c7f565b610da39190612cdf565b83901c145b610db0575f5ffd5b5b505050565b610dbe6122a7565b6040518060600160405280604051806040016040528060208152602001600281525081526020015f151581526020016002815250905090565b610dff6122a7565b826020015115610e0d575f5ffd5b610e178383610bd1565b905092915050565b5f5f610e2d8484600161171f565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff811480610e5c57505f81145b610e66575f610e69565b60015b91505092915050565b610e7a6122a7565b8460200151158015610e8d575082602001515b610e95575f5ffd5b8160200151158015610eb05750610eae825f0151611c61565b155b610eb8575f5ffd5b610ec38583866118f6565b610ecb575f5ffd5b5f610ee0855f0151855f0151855f0151611de3565b90505f610eec82610fca565b90505f8103610f0657610efd6116e0565b92505050610f73565b85602001518015610f1c5750610f1b85611c32565b5b15610f5157610f48604051806060016040528084815260200160011515815260200183815250856113df565b92505050610f73565b60405180606001604052808381526020015f1515815260200182815250925050505b949350505050565b610f836122a7565b610fa083610f91845f610b97565b610f9b8686611f80565b610aa5565b905092915050565b610fb06122a7565b610fc283610fbc611265565b84610aa5565b905092915050565b5f610fd482611c61565b15610fe1575f9050611002565b5f60208301519050610ff28161196e565b9150600860208451030282019150505b919050565b5f5f6110158484600161171f565b9050600181148061102557505f81145b61102f575f611032565b60015b91505092915050565b6110436122a7565b61104e848484611eb9565b90509392505050565b5f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff61108485855f61171f565b036110ae575f61109c6110956116e0565b845f61171f565b146110a5575f5ffd5b6001905061125e565b5f846020015180156110c1575083602001515b806110dc575084602001511580156110db57508360200151155b5b9050806110ed5782602001516110f4565b8260200151155b6110fc575f5ffd5b5f611110856111096116e0565b600161171f565b03611119575f5ffd5b5f60405180606001604052808760200151151515158152602001866020015115151515815260200185602001511515151581525090505f8660200190151590811515815250505f8560200190151590811515815250505f8460200190151590811515815250505f61118a868661163f565b90505f6111998289600161171f565b036111aa576001935050505061125e565b5f6111bd886111b7611265565b84610aa5565b90505f6111d56111cd84846113df565b8a600161171f565b146111de575f5ffd5b825f600381106111f1576111f0612d0f565b5b60200201518860200190151590811515815250508260016003811061121957611218612d0f565b5b60200201518760200190151590811515815250508260026003811061124157611240612d0f565b5b602002015186602001901515908115158152505060019450505050505b9392505050565b61126d6122a7565b6040518060600160405280604051806040016040528060208152602001600181525081526020015f151581526020016001815250905090565b5f60608251510160208301209050919050565b60605f60605f5f90505f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90505987518751808203828b01828b0184860160208101865b5f5f8214036113895784518682116001811461133b578c8203855260018d145f83141660018114611330575f9d50611335565b60019d505b50611370565b85518d81840303865260018e148d8214168e820184101760018114611362575f9e50611367565b60019e505b50602087039650505b50602084039350602086039550506020810390506112fd565b506020820191505b5f8251036113b057602088019750602087039650602082019150611391565b879a50868b528060405250505050505050505f6113cc84610fca565b9050838195509550505050509250929050565b6113e76122a7565b5f83604001511480156113fd57505f8260400151145b156114115761140a6116e0565b90506115eb565b5f836040015103611424578190506115eb565b5f826040015103611437578290506115eb565b60605f5f61144686865f61171f565b9050856020015180611459575084602001515b1561157d5785602001518015611470575084602001515b156114d2575f811261149e57611491865f0151865f01518860400151611cb6565b80935081945050506114bc565b6114b3855f0151875f01518760400151611cb6565b80935081945050505b6001846020019015159081151581525050611578565b6001810361150b576114ea865f0151865f01516112b9565b80935081945050508560200151846020019015159081151581525050611577565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff810361156457611542855f0151875f01516112b9565b8093508194505050856020015115846020019015159081151581525050611576565b61156c6116e0565b93505050506115eb565b5b5b6115d5565b5f81126115a657611599865f0151865f01518860400151611cb6565b80935081945050506115c4565b6115bb855f0151875f01518760400151611cb6565b80935081945050505b5f8460200190151590811515815250505b82845f0181905250818460400181815250505050505b92915050565b6115f96122a7565b61160c611606858561163f565b83610fa8565b90509392505050565b5f5f6116238484600161171f565b905060018114611633575f611636565b60015b91505092915050565b6116476122a7565b5f61165284846113df565b90505f61167182611661610db6565b61166c856002611f80565b610aa5565b905061167d8585611945565b6116ca575f61168c868661086b565b90505f6116ab8261169b610db6565b6116a6856002611f80565b610aa5565b90506116c16116ba848361086b565b6002610bd1565b945050506116d8565b6116d5816002610bd1565b92505b505092915050565b6116e86122a7565b60405180606001604052806040518060400160405280602081526020015f81525081526020015f151581526020015f815250905090565b5f5f6001905082156117e7578460200151801561173d575083602001515b1561176a577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90506117e6565b5f15158560200151151514801561178957506001151584602001511515145b156117985760019150506118ef565b60011515856020015115151480156117b757505f151584602001511515145b156117e5577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9150506118ef565b5b5b83604001518560400151111561180057809150506118ef565b84604001518460400151111561184457807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff61183c9190612d3c565b9150506118ef565b5f5f5f5f5f895f015151905060208a51019450602089510193505f5f90505b818110156118e4578086015193508085015192508284111561188e57869750505050505050506118ef565b838311156118d057867fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff6118c29190612d3c565b9750505050505050506118ef565b6020816118dd9190612db2565b9050611863565b505f96505050505050505b9392505050565b5f836020015115801561190b57508260200151155b611913575f5ffd5b5f6119316119228685876115f1565b61192a611265565b600161171f565b1461193a575f5ffd5b600190509392505050565b5f5f6119538484600161171f565b90505f8114611962575f611965565b60015b91505092915050565b5f5f821460018114611bb7578260018403935060028404841793506004840484179350601084048417935061010084048417935062010000840484179350640100000000840484179350680100000000000000008404841793507001000000000000000000000000000000008404841793506001840193506040517ff8f9cbfae6cc78fbefe7cdc3a1793dfcf4f0e8bbd8cec470b6a28a7a5a3e1efd81527ff5ecf1b3e9debc68e1d9cfabc5997135bfb7a7a3938b7b606b5b4b3f2f1f0ffe60208201527ff6e4ed9ff2d6b458eadcdf97bd91692de2d4da8fd2d0ac50c6ae9a827252361660408201527fc8c0b887b0a8a4489c948c7f847c6125746c645c544c444038302820181008ff60608201527ff7cae577eec2a03cf3bad76fb589591debb2dd67e0aa9834bea6925f6a4a2e0e60808201527fe39ed557db96902cd38ed14fad815115c786af479b7e8324736353433727170760a08201527fc976c13bb96e881cb166a933a55e490d9d56952b8d4e801485467d236242260660c08201527f753a6d1b65325d0c552a4d1345224105391a310b29122104190a11030902010060e082015261010081016040527e818283848586878898a8b8c8d8e8f929395969799a9b9d9e9faaeb6bedeeff7f01000000000000000000000000000000000000000000000000000000000000008082880204818160ff038501510496507f8000000000000000000000000000000000000000000000000000000000000000851161010002870196505f60018603861603611bad576001870196505b5050505050611bbb565b5f91505b50919050565b611bc96122a7565b826020015115611bd7575f5ffd5b611be18383611ff8565b905092915050565b5f5f611bf78484600161171f565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114611c26575f611c29565b60015b91505092915050565b5f8151518251016002815106915050919050565b611c4e6122a7565b611c5983835f611eb9565b905092915050565b5f5f5f6020840190505f5f90505b8451811015611ca957815192505f831115611c8f575f9350505050611cb1565b602082019150602081611ca29190612db2565b9050611c6f565b506001925050505b919050565b60605f6060595f60015f0388518901885189018a5160208601018b515b5f5f821403611d7a5783518c518e5103821160018114611d15578782018452600188148288141660018114611d0a575f9850611d0f565b600198505b50611d61565b8451888184010185528881038803831160018114611d53575f82115f8b11178985141660018114611d48575f9a50611d4d565b60019a505b50611d58565b600199505b50602086039550505b5060208303925060208503945050602081039050611cd3565b505f851460018114611d8f5760018252611d96565b6020870196505b50859650846020028c510187526020875101870160405260208701516001816101008d061c14600187141715611dcd5760018b019a505b5050505050505080849250925050935093915050565b606083518351835160405183815282602082015281604082015283606082018560208b0160046101c2fa84606001848184018660208c0160046101c2fa91508481019050838184018560208b0160046101c2fa9150815f8103611e4257fe5b5083810190508360608401828560056105465a03fa9150815f8103611e6357fe5b5083606084015b5f6020831403611e98575f81511460018114611e865750611e98565b60208201915050602082039150611e6a565b60208103985081895285856060010160405250505050505050509392505050565b611ec16122a7565b6020840184515f595f602084061460018114611ef157602084066020036020818401019350808501835250611efb565b6020820192508382525b508282848660046101c2fa506020815101810160405280515b5f6020821403611f45575f60208301511460018114611f335750611f45565b60208301925050602081039050611f14565b80825281865287602087015250505050505f8214611f635781611f70565b611f6f815f0151610fca565b5b8160400181815250509392505050565b611f886122a7565b5f6040518060400160405280602081526020015f81525090505f836040860151029050600161010082061b60206001610100840401028352806020840152602083510183016040525060405180606001604052808381526020015f15158152602001828152509250505092915050565b6120006122a7565b5f82148061201157505f8360400151145b1561201e578290506122a1565b5f835f01515190505f60016101006001876040015161203d9190612c7f565b6120479190612cdf565b6120519190612db2565b90505f816101006120629190612c7f565b610100866120709190612cdf565b101561207c575f61207f565b60015b60ff16610100866120909190612de5565b61209a9190612db2565b90505f6020826120aa9190612e15565b846120b59190612db2565b90508587604001516120c79190612db2565b8560400181815250508660200151856020019015159081151581525050610100866120f29190612cdf565b955060605f6040518381528381015f81528193506020840192506020810160405250505f6008896121239190612cdf565b036121ed575f600860016101008b60018e604001516121429190612c7f565b61214c9190612db2565b6121569190612cdf565b6101006121639190612c7f565b61216d9190612c7f565b6121779190612de5565b90505f5f60088c6040015161218c9190612cdf565b03612197575f61219a565b60015b60ff1660088c604001516121ae9190612de5565b6121b89190612db2565b9050600887610100030460208c5101018284018981848460046101c2fa50505083895f018190525050505050505050506122a1565b5f5f896101006121fd9190612c7f565b90505f5f60208d510190506101008c8a6122179190612db2565b111561222d578051915081831c85526020850194505b5f8d5f01515190505b5f811461228d57815192506020811460018114612259576020830151955061225d565b5f95505b50828d1b925084841c945084831786526020820191506020860195506020816122869190612c7f565b9050612236565b50858b5f0181905250505050505050505050505b92915050565b6040518060600160405280606081526020015f151581526020015f81525090565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b612323826122dd565b810181811067ffffffffffffffff82111715612342576123416122ed565b5b80604052505050565b5f6123546122c8565b9050612360828261231a565b919050565b5f5ffd5b5f5ffd5b5f5ffd5b5f67ffffffffffffffff82111561238b5761238a6122ed565b5b612394826122dd565b9050602081019050919050565b828183375f83830152505050565b5f6123c16123bc84612371565b61234b565b9050828152602081018484840111156123dd576123dc61236d565b5b6123e88482856123a1565b509392505050565b5f82601f83011261240457612403612369565b5b81356124148482602086016123af565b91505092915050565b5f8115159050919050565b6124318161241d565b811461243b575f5ffd5b50565b5f8135905061244c81612428565b92915050565b5f819050919050565b61246481612452565b811461246e575f5ffd5b50565b5f8135905061247f8161245b565b92915050565b5f6060828403121561249a576124996122d9565b5b6124a4606061234b565b90505f82013567ffffffffffffffff8111156124c3576124c2612365565b5b6124cf848285016123f0565b5f8301525060206124e28482850161243e565b60208301525060406124f684828501612471565b60408301525092915050565b5f60208284031215612517576125166122d1565b5b5f82013567ffffffffffffffff811115612534576125336122d5565b5b61254084828501612485565b91505092915050565b6125528161241d565b82525050565b5f60208201905061256b5f830184612549565b92915050565b5f5f60408385031215612587576125866122d1565b5b5f83013567ffffffffffffffff8111156125a4576125a36122d5565b5b6125b085828601612485565b925050602083013567ffffffffffffffff8111156125d1576125d06122d5565b5b6125dd85828601612485565b9150509250929050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f612619826125e7565b61262381856125f1565b9350612633818560208601612601565b61263c816122dd565b840191505092915050565b6126508161241d565b82525050565b61265f81612452565b82525050565b5f606083015f8301518482035f86015261267f828261260f565b91505060208301516126946020860182612647565b5060408301516126a76040860182612656565b508091505092915050565b5f6020820190508181035f8301526126ca8184612665565b905092915050565b5f5f5f606084860312156126e9576126e86122d1565b5b5f84013567ffffffffffffffff811115612706576127056122d5565b5b61271286828701612485565b935050602084013567ffffffffffffffff811115612733576127326122d5565b5b61273f86828701612485565b925050604084013567ffffffffffffffff8111156127605761275f6122d5565b5b61276c86828701612485565b9150509250925092565b5f5f6040838503121561278c5761278b6122d1565b5b5f61279985828601612471565b92505060206127aa8582860161243e565b9150509250929050565b5f5f604083850312156127ca576127c96122d1565b5b5f83013567ffffffffffffffff8111156127e7576127e66122d5565b5b6127f385828601612485565b925050602061280485828601612471565b9150509250929050565b61281781612452565b82525050565b5f6020820190506128305f83018461280e565b92915050565b5f5f5f5f6080858703121561284e5761284d6122d1565b5b5f85013567ffffffffffffffff81111561286b5761286a6122d5565b5b61287787828801612485565b945050602085013567ffffffffffffffff811115612898576128976122d5565b5b6128a487828801612485565b935050604085013567ffffffffffffffff8111156128c5576128c46122d5565b5b6128d187828801612485565b925050606085013567ffffffffffffffff8111156128f2576128f16122d5565b5b6128fe87828801612485565b91505092959194509250565b5f6020828403121561291f5761291e6122d1565b5b5f82013567ffffffffffffffff81111561293c5761293b6122d5565b5b612948848285016123f0565b91505092915050565b5f5f5f60608486031215612968576129676122d1565b5b5f84013567ffffffffffffffff811115612985576129846122d5565b5b612991868287016123f0565b93505060206129a28682870161243e565b92505060406129b386828701612471565b9150509250925092565b5f819050919050565b6129cf816129bd565b82525050565b5f6020820190506129e85f8301846129c6565b92915050565b5f5f60408385031215612a0457612a036122d1565b5b5f83013567ffffffffffffffff811115612a2157612a206122d5565b5b612a2d858286016123f0565b925050602083013567ffffffffffffffff811115612a4e57612a4d6122d5565b5b612a5a858286016123f0565b9150509250929050565b5f82825260208201905092915050565b5f612a7e826125e7565b612a888185612a64565b9350612a98818560208601612601565b612aa1816122dd565b840191505092915050565b5f6040820190508181035f830152612ac48185612a74565b9050612ad3602083018461280e565b9392505050565b5f5f5f60608486031215612af157612af06122d1565b5b5f84013567ffffffffffffffff811115612b0e57612b0d6122d5565b5b612b1a86828701612485565b935050602084013567ffffffffffffffff811115612b3b57612b3a6122d5565b5b612b4786828701612485565b9250506040612b588682870161243e565b9150509250925092565b5f819050919050565b612b7481612b62565b82525050565b5f602082019050612b8d5f830184612b6b565b92915050565b5f60208284031215612ba857612ba76122d1565b5b5f612bb584828501612471565b91505092915050565b5f5f60408385031215612bd457612bd36122d1565b5b5f83013567ffffffffffffffff811115612bf157612bf06122d5565b5b612bfd858286016123f0565b9250506020612c0e8582860161243e565b9150509250929050565b5f819050919050565b612c32612c2d82612452565b612c18565b82525050565b5f612c438284612c21565b60208201915081905092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f612c8982612452565b9150612c9483612452565b9250828203905081811115612cac57612cab612c52565b5b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601260045260245ffd5b5f612ce982612452565b9150612cf483612452565b925082612d0457612d03612cb2565b5b828206905092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f612d4682612b62565b9150612d5183612b62565b9250828202612d5f81612b62565b91507f800000000000000000000000000000000000000000000000000000000000000084145f84121615612d9657612d95612c52565b5b8282058414831517612dab57612daa612c52565b5b5092915050565b5f612dbc82612452565b9150612dc783612452565b9250828201905080821115612ddf57612dde612c52565b5b92915050565b5f612def82612452565b9150612dfa83612452565b925082612e0a57612e09612cb2565b5b828204905092915050565b5f612e1f82612452565b9150612e2a83612452565b9250828202612e3881612452565b91508282048414831517612e4f57612e4e612c52565b5b509291505056fea2646970667358221220995ea0fe0f42a1a4bdb87b057bcd61c4b9ca3cce6d0e15ce969ab866b1fbe4cf64736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
}

// BigNumbersABI is the input ABI used to generate the binding from.
// Deprecated: Use BigNumbersMetaData.ABI instead.
var BigNumbersABI = BigNumbersMetaData.ABI

// BigNumbersBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BigNumbersMetaData.Bin instead.
var BigNumbersBin = BigNumbersMetaData.Bin

// DeployBigNumbers deploys a new Ethereum contract, binding an instance of BigNumbers to it.
func DeployBigNumbers(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BigNumbers, error) {
	parsed, err := BigNumbersMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BigNumbersBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BigNumbers{BigNumbersCaller: BigNumbersCaller{contract: contract}, BigNumbersTransactor: BigNumbersTransactor{contract: contract}, BigNumbersFilterer: BigNumbersFilterer{contract: contract}}, nil
}

// BigNumbers is an auto generated Go binding around an Ethereum contract.
type BigNumbers struct {
	BigNumbersCaller     // Read-only binding to the contract
	BigNumbersTransactor // Write-only binding to the contract
	BigNumbersFilterer   // Log filterer for contract events
}

// BigNumbersCaller is an auto generated read-only Go binding around an Ethereum contract.
type BigNumbersCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BigNumbersTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BigNumbersTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BigNumbersFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BigNumbersFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BigNumbersSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BigNumbersSession struct {
	Contract     *BigNumbers       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BigNumbersCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BigNumbersCallerSession struct {
	Contract *BigNumbersCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// BigNumbersTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BigNumbersTransactorSession struct {
	Contract     *BigNumbersTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// BigNumbersRaw is an auto generated low-level Go binding around an Ethereum contract.
type BigNumbersRaw struct {
	Contract *BigNumbers // Generic contract binding to access the raw methods on
}

// BigNumbersCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BigNumbersCallerRaw struct {
	Contract *BigNumbersCaller // Generic read-only contract binding to access the raw methods on
}

// BigNumbersTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BigNumbersTransactorRaw struct {
	Contract *BigNumbersTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBigNumbers creates a new instance of BigNumbers, bound to a specific deployed contract.
func NewBigNumbers(address common.Address, backend bind.ContractBackend) (*BigNumbers, error) {
	contract, err := bindBigNumbers(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BigNumbers{BigNumbersCaller: BigNumbersCaller{contract: contract}, BigNumbersTransactor: BigNumbersTransactor{contract: contract}, BigNumbersFilterer: BigNumbersFilterer{contract: contract}}, nil
}

// NewBigNumbersCaller creates a new read-only instance of BigNumbers, bound to a specific deployed contract.
func NewBigNumbersCaller(address common.Address, caller bind.ContractCaller) (*BigNumbersCaller, error) {
	contract, err := bindBigNumbers(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BigNumbersCaller{contract: contract}, nil
}

// NewBigNumbersTransactor creates a new write-only instance of BigNumbers, bound to a specific deployed contract.
func NewBigNumbersTransactor(address common.Address, transactor bind.ContractTransactor) (*BigNumbersTransactor, error) {
	contract, err := bindBigNumbers(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BigNumbersTransactor{contract: contract}, nil
}

// NewBigNumbersFilterer creates a new log filterer instance of BigNumbers, bound to a specific deployed contract.
func NewBigNumbersFilterer(address common.Address, filterer bind.ContractFilterer) (*BigNumbersFilterer, error) {
	contract, err := bindBigNumbers(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BigNumbersFilterer{contract: contract}, nil
}

// bindBigNumbers binds a generic wrapper to an already deployed contract.
func bindBigNumbers(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BigNumbersMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BigNumbers *BigNumbersRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BigNumbers.Contract.BigNumbersCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BigNumbers *BigNumbersRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BigNumbers.Contract.BigNumbersTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BigNumbers *BigNumbersRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BigNumbers.Contract.BigNumbersTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BigNumbers *BigNumbersCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BigNumbers.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BigNumbers *BigNumbersTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BigNumbers.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BigNumbers *BigNumbersTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BigNumbers.Contract.contract.Transact(opts, method, params...)
}

// Add is a free data retrieval call binding the contract method 0xdab6c95e.
//
// Solidity: function add((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersCaller) Add(opts *bind.CallOpts, a BigNumber, b BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "add", a, b)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Add is a free data retrieval call binding the contract method 0xdab6c95e.
//
// Solidity: function add((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersSession) Add(a BigNumber, b BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Add(&_BigNumbers.CallOpts, a, b)
}

// Add is a free data retrieval call binding the contract method 0xdab6c95e.
//
// Solidity: function add((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersCallerSession) Add(a BigNumber, b BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Add(&_BigNumbers.CallOpts, a, b)
}

// BitLength is a free data retrieval call binding the contract method 0x7215b378.
//
// Solidity: function bitLength((bytes,bool,uint256) a) pure returns(uint256)
func (_BigNumbers *BigNumbersCaller) BitLength(opts *bind.CallOpts, a BigNumber) (*big.Int, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "bitLength", a)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BitLength is a free data retrieval call binding the contract method 0x7215b378.
//
// Solidity: function bitLength((bytes,bool,uint256) a) pure returns(uint256)
func (_BigNumbers *BigNumbersSession) BitLength(a BigNumber) (*big.Int, error) {
	return _BigNumbers.Contract.BitLength(&_BigNumbers.CallOpts, a)
}

// BitLength is a free data retrieval call binding the contract method 0x7215b378.
//
// Solidity: function bitLength((bytes,bool,uint256) a) pure returns(uint256)
func (_BigNumbers *BigNumbersCallerSession) BitLength(a BigNumber) (*big.Int, error) {
	return _BigNumbers.Contract.BitLength(&_BigNumbers.CallOpts, a)
}

// BitLength0 is a free data retrieval call binding the contract method 0x7b19cc9a.
//
// Solidity: function bitLength(bytes a) pure returns(uint256 r)
func (_BigNumbers *BigNumbersCaller) BitLength0(opts *bind.CallOpts, a []byte) (*big.Int, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "bitLength0", a)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BitLength0 is a free data retrieval call binding the contract method 0x7b19cc9a.
//
// Solidity: function bitLength(bytes a) pure returns(uint256 r)
func (_BigNumbers *BigNumbersSession) BitLength0(a []byte) (*big.Int, error) {
	return _BigNumbers.Contract.BitLength0(&_BigNumbers.CallOpts, a)
}

// BitLength0 is a free data retrieval call binding the contract method 0x7b19cc9a.
//
// Solidity: function bitLength(bytes a) pure returns(uint256 r)
func (_BigNumbers *BigNumbersCallerSession) BitLength0(a []byte) (*big.Int, error) {
	return _BigNumbers.Contract.BitLength0(&_BigNumbers.CallOpts, a)
}

// BitLength1 is a free data retrieval call binding the contract method 0xe41bbcf8.
//
// Solidity: function bitLength(uint256 a) pure returns(uint256 r)
func (_BigNumbers *BigNumbersCaller) BitLength1(opts *bind.CallOpts, a *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "bitLength1", a)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BitLength1 is a free data retrieval call binding the contract method 0xe41bbcf8.
//
// Solidity: function bitLength(uint256 a) pure returns(uint256 r)
func (_BigNumbers *BigNumbersSession) BitLength1(a *big.Int) (*big.Int, error) {
	return _BigNumbers.Contract.BitLength1(&_BigNumbers.CallOpts, a)
}

// BitLength1 is a free data retrieval call binding the contract method 0xe41bbcf8.
//
// Solidity: function bitLength(uint256 a) pure returns(uint256 r)
func (_BigNumbers *BigNumbersCallerSession) BitLength1(a *big.Int) (*big.Int, error) {
	return _BigNumbers.Contract.BitLength1(&_BigNumbers.CallOpts, a)
}

// Cmp is a free data retrieval call binding the contract method 0xa1bbe28f.
//
// Solidity: function cmp((bytes,bool,uint256) a, (bytes,bool,uint256) b, bool signed) pure returns(int256)
func (_BigNumbers *BigNumbersCaller) Cmp(opts *bind.CallOpts, a BigNumber, b BigNumber, signed bool) (*big.Int, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "cmp", a, b, signed)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Cmp is a free data retrieval call binding the contract method 0xa1bbe28f.
//
// Solidity: function cmp((bytes,bool,uint256) a, (bytes,bool,uint256) b, bool signed) pure returns(int256)
func (_BigNumbers *BigNumbersSession) Cmp(a BigNumber, b BigNumber, signed bool) (*big.Int, error) {
	return _BigNumbers.Contract.Cmp(&_BigNumbers.CallOpts, a, b, signed)
}

// Cmp is a free data retrieval call binding the contract method 0xa1bbe28f.
//
// Solidity: function cmp((bytes,bool,uint256) a, (bytes,bool,uint256) b, bool signed) pure returns(int256)
func (_BigNumbers *BigNumbersCallerSession) Cmp(a BigNumber, b BigNumber, signed bool) (*big.Int, error) {
	return _BigNumbers.Contract.Cmp(&_BigNumbers.CallOpts, a, b, signed)
}

// DivVerify is a free data retrieval call binding the contract method 0x14ea0aa4.
//
// Solidity: function divVerify((bytes,bool,uint256) a, (bytes,bool,uint256) b, (bytes,bool,uint256) r) view returns(bool)
func (_BigNumbers *BigNumbersCaller) DivVerify(opts *bind.CallOpts, a BigNumber, b BigNumber, r BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "divVerify", a, b, r)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// DivVerify is a free data retrieval call binding the contract method 0x14ea0aa4.
//
// Solidity: function divVerify((bytes,bool,uint256) a, (bytes,bool,uint256) b, (bytes,bool,uint256) r) view returns(bool)
func (_BigNumbers *BigNumbersSession) DivVerify(a BigNumber, b BigNumber, r BigNumber) (bool, error) {
	return _BigNumbers.Contract.DivVerify(&_BigNumbers.CallOpts, a, b, r)
}

// DivVerify is a free data retrieval call binding the contract method 0x14ea0aa4.
//
// Solidity: function divVerify((bytes,bool,uint256) a, (bytes,bool,uint256) b, (bytes,bool,uint256) r) view returns(bool)
func (_BigNumbers *BigNumbersCallerSession) DivVerify(a BigNumber, b BigNumber, r BigNumber) (bool, error) {
	return _BigNumbers.Contract.DivVerify(&_BigNumbers.CallOpts, a, b, r)
}

// Eq is a free data retrieval call binding the contract method 0xcf808b5b.
//
// Solidity: function eq((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) Eq(opts *bind.CallOpts, a BigNumber, b BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "eq", a, b)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Eq is a free data retrieval call binding the contract method 0xcf808b5b.
//
// Solidity: function eq((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersSession) Eq(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Eq(&_BigNumbers.CallOpts, a, b)
}

// Eq is a free data retrieval call binding the contract method 0xcf808b5b.
//
// Solidity: function eq((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) Eq(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Eq(&_BigNumbers.CallOpts, a, b)
}

// Gt is a free data retrieval call binding the contract method 0x6a39e86f.
//
// Solidity: function gt((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) Gt(opts *bind.CallOpts, a BigNumber, b BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "gt", a, b)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Gt is a free data retrieval call binding the contract method 0x6a39e86f.
//
// Solidity: function gt((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersSession) Gt(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Gt(&_BigNumbers.CallOpts, a, b)
}

// Gt is a free data retrieval call binding the contract method 0x6a39e86f.
//
// Solidity: function gt((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) Gt(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Gt(&_BigNumbers.CallOpts, a, b)
}

// Gte is a free data retrieval call binding the contract method 0x49417f81.
//
// Solidity: function gte((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) Gte(opts *bind.CallOpts, a BigNumber, b BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "gte", a, b)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Gte is a free data retrieval call binding the contract method 0x49417f81.
//
// Solidity: function gte((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersSession) Gte(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Gte(&_BigNumbers.CallOpts, a, b)
}

// Gte is a free data retrieval call binding the contract method 0x49417f81.
//
// Solidity: function gte((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) Gte(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Gte(&_BigNumbers.CallOpts, a, b)
}

// Hash is a free data retrieval call binding the contract method 0x0bf3782a.
//
// Solidity: function hash((bytes,bool,uint256) a) pure returns(bytes32 h)
func (_BigNumbers *BigNumbersCaller) Hash(opts *bind.CallOpts, a BigNumber) ([32]byte, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "hash", a)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Hash is a free data retrieval call binding the contract method 0x0bf3782a.
//
// Solidity: function hash((bytes,bool,uint256) a) pure returns(bytes32 h)
func (_BigNumbers *BigNumbersSession) Hash(a BigNumber) ([32]byte, error) {
	return _BigNumbers.Contract.Hash(&_BigNumbers.CallOpts, a)
}

// Hash is a free data retrieval call binding the contract method 0x0bf3782a.
//
// Solidity: function hash((bytes,bool,uint256) a) pure returns(bytes32 h)
func (_BigNumbers *BigNumbersCallerSession) Hash(a BigNumber) ([32]byte, error) {
	return _BigNumbers.Contract.Hash(&_BigNumbers.CallOpts, a)
}

// Init is a free data retrieval call binding the contract method 0x40957f72.
//
// Solidity: function init(uint256 val, bool neg) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Init(opts *bind.CallOpts, val *big.Int, neg bool) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "init", val, neg)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Init is a free data retrieval call binding the contract method 0x40957f72.
//
// Solidity: function init(uint256 val, bool neg) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Init(val *big.Int, neg bool) (BigNumber, error) {
	return _BigNumbers.Contract.Init(&_BigNumbers.CallOpts, val, neg)
}

// Init is a free data retrieval call binding the contract method 0x40957f72.
//
// Solidity: function init(uint256 val, bool neg) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Init(val *big.Int, neg bool) (BigNumber, error) {
	return _BigNumbers.Contract.Init(&_BigNumbers.CallOpts, val, neg)
}

// Init0 is a free data retrieval call binding the contract method 0x85056b16.
//
// Solidity: function init(bytes val, bool neg, uint256 bitlen) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Init0(opts *bind.CallOpts, val []byte, neg bool, bitlen *big.Int) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "init0", val, neg, bitlen)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Init0 is a free data retrieval call binding the contract method 0x85056b16.
//
// Solidity: function init(bytes val, bool neg, uint256 bitlen) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Init0(val []byte, neg bool, bitlen *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Init0(&_BigNumbers.CallOpts, val, neg, bitlen)
}

// Init0 is a free data retrieval call binding the contract method 0x85056b16.
//
// Solidity: function init(bytes val, bool neg, uint256 bitlen) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Init0(val []byte, neg bool, bitlen *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Init0(&_BigNumbers.CallOpts, val, neg, bitlen)
}

// Init1 is a free data retrieval call binding the contract method 0xf33d5856.
//
// Solidity: function init(bytes val, bool neg) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Init1(opts *bind.CallOpts, val []byte, neg bool) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "init1", val, neg)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Init1 is a free data retrieval call binding the contract method 0xf33d5856.
//
// Solidity: function init(bytes val, bool neg) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Init1(val []byte, neg bool) (BigNumber, error) {
	return _BigNumbers.Contract.Init1(&_BigNumbers.CallOpts, val, neg)
}

// Init1 is a free data retrieval call binding the contract method 0xf33d5856.
//
// Solidity: function init(bytes val, bool neg) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Init1(val []byte, neg bool) (BigNumber, error) {
	return _BigNumbers.Contract.Init1(&_BigNumbers.CallOpts, val, neg)
}

// IsOdd is a free data retrieval call binding the contract method 0x0e3e15d2.
//
// Solidity: function isOdd((bytes,bool,uint256) a) pure returns(bool r)
func (_BigNumbers *BigNumbersCaller) IsOdd(opts *bind.CallOpts, a BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "isOdd", a)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOdd is a free data retrieval call binding the contract method 0x0e3e15d2.
//
// Solidity: function isOdd((bytes,bool,uint256) a) pure returns(bool r)
func (_BigNumbers *BigNumbersSession) IsOdd(a BigNumber) (bool, error) {
	return _BigNumbers.Contract.IsOdd(&_BigNumbers.CallOpts, a)
}

// IsOdd is a free data retrieval call binding the contract method 0x0e3e15d2.
//
// Solidity: function isOdd((bytes,bool,uint256) a) pure returns(bool r)
func (_BigNumbers *BigNumbersCallerSession) IsOdd(a BigNumber) (bool, error) {
	return _BigNumbers.Contract.IsOdd(&_BigNumbers.CallOpts, a)
}

// IsZero is a free data retrieval call binding the contract method 0x0ee8b649.
//
// Solidity: function isZero((bytes,bool,uint256) a) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) IsZero(opts *bind.CallOpts, a BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "isZero", a)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsZero is a free data retrieval call binding the contract method 0x0ee8b649.
//
// Solidity: function isZero((bytes,bool,uint256) a) pure returns(bool)
func (_BigNumbers *BigNumbersSession) IsZero(a BigNumber) (bool, error) {
	return _BigNumbers.Contract.IsZero(&_BigNumbers.CallOpts, a)
}

// IsZero is a free data retrieval call binding the contract method 0x0ee8b649.
//
// Solidity: function isZero((bytes,bool,uint256) a) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) IsZero(a BigNumber) (bool, error) {
	return _BigNumbers.Contract.IsZero(&_BigNumbers.CallOpts, a)
}

// IsZero0 is a free data retrieval call binding the contract method 0xfad242a8.
//
// Solidity: function isZero(bytes a) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) IsZero0(opts *bind.CallOpts, a []byte) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "isZero0", a)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsZero0 is a free data retrieval call binding the contract method 0xfad242a8.
//
// Solidity: function isZero(bytes a) pure returns(bool)
func (_BigNumbers *BigNumbersSession) IsZero0(a []byte) (bool, error) {
	return _BigNumbers.Contract.IsZero0(&_BigNumbers.CallOpts, a)
}

// IsZero0 is a free data retrieval call binding the contract method 0xfad242a8.
//
// Solidity: function isZero(bytes a) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) IsZero0(a []byte) (bool, error) {
	return _BigNumbers.Contract.IsZero0(&_BigNumbers.CallOpts, a)
}

// Lt is a free data retrieval call binding the contract method 0x55bb315d.
//
// Solidity: function lt((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) Lt(opts *bind.CallOpts, a BigNumber, b BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "lt", a, b)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Lt is a free data retrieval call binding the contract method 0x55bb315d.
//
// Solidity: function lt((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersSession) Lt(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Lt(&_BigNumbers.CallOpts, a, b)
}

// Lt is a free data retrieval call binding the contract method 0x55bb315d.
//
// Solidity: function lt((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) Lt(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Lt(&_BigNumbers.CallOpts, a, b)
}

// Lte is a free data retrieval call binding the contract method 0x01adfbac.
//
// Solidity: function lte((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCaller) Lte(opts *bind.CallOpts, a BigNumber, b BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "lte", a, b)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Lte is a free data retrieval call binding the contract method 0x01adfbac.
//
// Solidity: function lte((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersSession) Lte(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Lte(&_BigNumbers.CallOpts, a, b)
}

// Lte is a free data retrieval call binding the contract method 0x01adfbac.
//
// Solidity: function lte((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns(bool)
func (_BigNumbers *BigNumbersCallerSession) Lte(a BigNumber, b BigNumber) (bool, error) {
	return _BigNumbers.Contract.Lte(&_BigNumbers.CallOpts, a, b)
}

// Mod is a free data retrieval call binding the contract method 0x84316280.
//
// Solidity: function mod((bytes,bool,uint256) a, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Mod(opts *bind.CallOpts, a BigNumber, n BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "mod", a, n)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Mod is a free data retrieval call binding the contract method 0x84316280.
//
// Solidity: function mod((bytes,bool,uint256) a, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Mod(a BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Mod(&_BigNumbers.CallOpts, a, n)
}

// Mod is a free data retrieval call binding the contract method 0x84316280.
//
// Solidity: function mod((bytes,bool,uint256) a, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Mod(a BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Mod(&_BigNumbers.CallOpts, a, n)
}

// Modexp is a free data retrieval call binding the contract method 0xe7853f76.
//
// Solidity: function modexp((bytes,bool,uint256) a, (bytes,bool,uint256) e, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Modexp(opts *bind.CallOpts, a BigNumber, e BigNumber, n BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "modexp", a, e, n)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Modexp is a free data retrieval call binding the contract method 0xe7853f76.
//
// Solidity: function modexp((bytes,bool,uint256) a, (bytes,bool,uint256) e, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Modexp(a BigNumber, e BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Modexp(&_BigNumbers.CallOpts, a, e, n)
}

// Modexp is a free data retrieval call binding the contract method 0xe7853f76.
//
// Solidity: function modexp((bytes,bool,uint256) a, (bytes,bool,uint256) e, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Modexp(a BigNumber, e BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Modexp(&_BigNumbers.CallOpts, a, e, n)
}

// Modexp0 is a free data retrieval call binding the contract method 0xe6c11652.
//
// Solidity: function modexp((bytes,bool,uint256) a, (bytes,bool,uint256) ai, (bytes,bool,uint256) e, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Modexp0(opts *bind.CallOpts, a BigNumber, ai BigNumber, e BigNumber, n BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "modexp0", a, ai, e, n)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Modexp0 is a free data retrieval call binding the contract method 0xe6c11652.
//
// Solidity: function modexp((bytes,bool,uint256) a, (bytes,bool,uint256) ai, (bytes,bool,uint256) e, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Modexp0(a BigNumber, ai BigNumber, e BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Modexp0(&_BigNumbers.CallOpts, a, ai, e, n)
}

// Modexp0 is a free data retrieval call binding the contract method 0xe6c11652.
//
// Solidity: function modexp((bytes,bool,uint256) a, (bytes,bool,uint256) ai, (bytes,bool,uint256) e, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Modexp0(a BigNumber, ai BigNumber, e BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Modexp0(&_BigNumbers.CallOpts, a, ai, e, n)
}

// ModinvVerify is a free data retrieval call binding the contract method 0x88a006f2.
//
// Solidity: function modinvVerify((bytes,bool,uint256) a, (bytes,bool,uint256) n, (bytes,bool,uint256) r) view returns(bool)
func (_BigNumbers *BigNumbersCaller) ModinvVerify(opts *bind.CallOpts, a BigNumber, n BigNumber, r BigNumber) (bool, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "modinvVerify", a, n, r)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ModinvVerify is a free data retrieval call binding the contract method 0x88a006f2.
//
// Solidity: function modinvVerify((bytes,bool,uint256) a, (bytes,bool,uint256) n, (bytes,bool,uint256) r) view returns(bool)
func (_BigNumbers *BigNumbersSession) ModinvVerify(a BigNumber, n BigNumber, r BigNumber) (bool, error) {
	return _BigNumbers.Contract.ModinvVerify(&_BigNumbers.CallOpts, a, n, r)
}

// ModinvVerify is a free data retrieval call binding the contract method 0x88a006f2.
//
// Solidity: function modinvVerify((bytes,bool,uint256) a, (bytes,bool,uint256) n, (bytes,bool,uint256) r) view returns(bool)
func (_BigNumbers *BigNumbersCallerSession) ModinvVerify(a BigNumber, n BigNumber, r BigNumber) (bool, error) {
	return _BigNumbers.Contract.ModinvVerify(&_BigNumbers.CallOpts, a, n, r)
}

// Modmul is a free data retrieval call binding the contract method 0x8e4027af.
//
// Solidity: function modmul((bytes,bool,uint256) a, (bytes,bool,uint256) b, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Modmul(opts *bind.CallOpts, a BigNumber, b BigNumber, n BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "modmul", a, b, n)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Modmul is a free data retrieval call binding the contract method 0x8e4027af.
//
// Solidity: function modmul((bytes,bool,uint256) a, (bytes,bool,uint256) b, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Modmul(a BigNumber, b BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Modmul(&_BigNumbers.CallOpts, a, b, n)
}

// Modmul is a free data retrieval call binding the contract method 0x8e4027af.
//
// Solidity: function modmul((bytes,bool,uint256) a, (bytes,bool,uint256) b, (bytes,bool,uint256) n) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Modmul(a BigNumber, b BigNumber, n BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Modmul(&_BigNumbers.CallOpts, a, b, n)
}

// Mul is a free data retrieval call binding the contract method 0x25b68140.
//
// Solidity: function mul((bytes,bool,uint256) a, (bytes,bool,uint256) b) view returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersCaller) Mul(opts *bind.CallOpts, a BigNumber, b BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "mul", a, b)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Mul is a free data retrieval call binding the contract method 0x25b68140.
//
// Solidity: function mul((bytes,bool,uint256) a, (bytes,bool,uint256) b) view returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersSession) Mul(a BigNumber, b BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Mul(&_BigNumbers.CallOpts, a, b)
}

// Mul is a free data retrieval call binding the contract method 0x25b68140.
//
// Solidity: function mul((bytes,bool,uint256) a, (bytes,bool,uint256) b) view returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersCallerSession) Mul(a BigNumber, b BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Mul(&_BigNumbers.CallOpts, a, b)
}

// One is a free data retrieval call binding the contract method 0x901717d1.
//
// Solidity: function one() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) One(opts *bind.CallOpts) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "one")

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// One is a free data retrieval call binding the contract method 0x901717d1.
//
// Solidity: function one() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) One() (BigNumber, error) {
	return _BigNumbers.Contract.One(&_BigNumbers.CallOpts)
}

// One is a free data retrieval call binding the contract method 0x901717d1.
//
// Solidity: function one() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) One() (BigNumber, error) {
	return _BigNumbers.Contract.One(&_BigNumbers.CallOpts)
}

// Pow is a free data retrieval call binding the contract method 0xe2f534b1.
//
// Solidity: function pow((bytes,bool,uint256) a, uint256 e) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Pow(opts *bind.CallOpts, a BigNumber, e *big.Int) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "pow", a, e)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Pow is a free data retrieval call binding the contract method 0xe2f534b1.
//
// Solidity: function pow((bytes,bool,uint256) a, uint256 e) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Pow(a BigNumber, e *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Pow(&_BigNumbers.CallOpts, a, e)
}

// Pow is a free data retrieval call binding the contract method 0xe2f534b1.
//
// Solidity: function pow((bytes,bool,uint256) a, uint256 e) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Pow(a BigNumber, e *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Pow(&_BigNumbers.CallOpts, a, e)
}

// Shl is a free data retrieval call binding the contract method 0x95c9734d.
//
// Solidity: function shl((bytes,bool,uint256) a, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Shl(opts *bind.CallOpts, a BigNumber, bits *big.Int) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "shl", a, bits)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Shl is a free data retrieval call binding the contract method 0x95c9734d.
//
// Solidity: function shl((bytes,bool,uint256) a, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Shl(a BigNumber, bits *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Shl(&_BigNumbers.CallOpts, a, bits)
}

// Shl is a free data retrieval call binding the contract method 0x95c9734d.
//
// Solidity: function shl((bytes,bool,uint256) a, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Shl(a BigNumber, bits *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Shl(&_BigNumbers.CallOpts, a, bits)
}

// Shr is a free data retrieval call binding the contract method 0x7e8cde22.
//
// Solidity: function shr((bytes,bool,uint256) a, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Shr(opts *bind.CallOpts, a BigNumber, bits *big.Int) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "shr", a, bits)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Shr is a free data retrieval call binding the contract method 0x7e8cde22.
//
// Solidity: function shr((bytes,bool,uint256) a, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Shr(a BigNumber, bits *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Shr(&_BigNumbers.CallOpts, a, bits)
}

// Shr is a free data retrieval call binding the contract method 0x7e8cde22.
//
// Solidity: function shr((bytes,bool,uint256) a, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Shr(a BigNumber, bits *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.Shr(&_BigNumbers.CallOpts, a, bits)
}

// ShrPrivate is a free data retrieval call binding the contract method 0x72ff9c92.
//
// Solidity: function shrPrivate((bytes,bool,uint256) bn, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) ShrPrivate(opts *bind.CallOpts, bn BigNumber, bits *big.Int) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "shrPrivate", bn, bits)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// ShrPrivate is a free data retrieval call binding the contract method 0x72ff9c92.
//
// Solidity: function shrPrivate((bytes,bool,uint256) bn, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) ShrPrivate(bn BigNumber, bits *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.ShrPrivate(&_BigNumbers.CallOpts, bn, bits)
}

// ShrPrivate is a free data retrieval call binding the contract method 0x72ff9c92.
//
// Solidity: function shrPrivate((bytes,bool,uint256) bn, uint256 bits) view returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) ShrPrivate(bn BigNumber, bits *big.Int) (BigNumber, error) {
	return _BigNumbers.Contract.ShrPrivate(&_BigNumbers.CallOpts, bn, bits)
}

// Sub is a free data retrieval call binding the contract method 0x7da5919e.
//
// Solidity: function sub((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersCaller) Sub(opts *bind.CallOpts, a BigNumber, b BigNumber) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "sub", a, b)

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Sub is a free data retrieval call binding the contract method 0x7da5919e.
//
// Solidity: function sub((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersSession) Sub(a BigNumber, b BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Sub(&_BigNumbers.CallOpts, a, b)
}

// Sub is a free data retrieval call binding the contract method 0x7da5919e.
//
// Solidity: function sub((bytes,bool,uint256) a, (bytes,bool,uint256) b) pure returns((bytes,bool,uint256) r)
func (_BigNumbers *BigNumbersCallerSession) Sub(a BigNumber, b BigNumber) (BigNumber, error) {
	return _BigNumbers.Contract.Sub(&_BigNumbers.CallOpts, a, b)
}

// SubPrivate is a free data retrieval call binding the contract method 0x969ecb17.
//
// Solidity: function subPrivate(bytes max, bytes min) pure returns(bytes, uint256)
func (_BigNumbers *BigNumbersCaller) SubPrivate(opts *bind.CallOpts, max []byte, min []byte) ([]byte, *big.Int, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "subPrivate", max, min)

	if err != nil {
		return *new([]byte), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// SubPrivate is a free data retrieval call binding the contract method 0x969ecb17.
//
// Solidity: function subPrivate(bytes max, bytes min) pure returns(bytes, uint256)
func (_BigNumbers *BigNumbersSession) SubPrivate(max []byte, min []byte) ([]byte, *big.Int, error) {
	return _BigNumbers.Contract.SubPrivate(&_BigNumbers.CallOpts, max, min)
}

// SubPrivate is a free data retrieval call binding the contract method 0x969ecb17.
//
// Solidity: function subPrivate(bytes max, bytes min) pure returns(bytes, uint256)
func (_BigNumbers *BigNumbersCallerSession) SubPrivate(max []byte, min []byte) ([]byte, *big.Int, error) {
	return _BigNumbers.Contract.SubPrivate(&_BigNumbers.CallOpts, max, min)
}

// Two is a free data retrieval call binding the contract method 0x5fdf05d7.
//
// Solidity: function two() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Two(opts *bind.CallOpts) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "two")

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Two is a free data retrieval call binding the contract method 0x5fdf05d7.
//
// Solidity: function two() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Two() (BigNumber, error) {
	return _BigNumbers.Contract.Two(&_BigNumbers.CallOpts)
}

// Two is a free data retrieval call binding the contract method 0x5fdf05d7.
//
// Solidity: function two() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Two() (BigNumber, error) {
	return _BigNumbers.Contract.Two(&_BigNumbers.CallOpts)
}

// Verify is a free data retrieval call binding the contract method 0xdf8e3603.
//
// Solidity: function verify((bytes,bool,uint256) bn) pure returns()
func (_BigNumbers *BigNumbersCaller) Verify(opts *bind.CallOpts, bn BigNumber) error {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "verify", bn)

	if err != nil {
		return err
	}

	return err

}

// Verify is a free data retrieval call binding the contract method 0xdf8e3603.
//
// Solidity: function verify((bytes,bool,uint256) bn) pure returns()
func (_BigNumbers *BigNumbersSession) Verify(bn BigNumber) error {
	return _BigNumbers.Contract.Verify(&_BigNumbers.CallOpts, bn)
}

// Verify is a free data retrieval call binding the contract method 0xdf8e3603.
//
// Solidity: function verify((bytes,bool,uint256) bn) pure returns()
func (_BigNumbers *BigNumbersCallerSession) Verify(bn BigNumber) error {
	return _BigNumbers.Contract.Verify(&_BigNumbers.CallOpts, bn)
}

// Zero is a free data retrieval call binding the contract method 0xbc1b392d.
//
// Solidity: function zero() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCaller) Zero(opts *bind.CallOpts) (BigNumber, error) {
	var out []interface{}
	err := _BigNumbers.contract.Call(opts, &out, "zero")

	if err != nil {
		return *new(BigNumber), err
	}

	out0 := *abi.ConvertType(out[0], new(BigNumber)).(*BigNumber)

	return out0, err

}

// Zero is a free data retrieval call binding the contract method 0xbc1b392d.
//
// Solidity: function zero() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersSession) Zero() (BigNumber, error) {
	return _BigNumbers.Contract.Zero(&_BigNumbers.CallOpts)
}

// Zero is a free data retrieval call binding the contract method 0xbc1b392d.
//
// Solidity: function zero() pure returns((bytes,bool,uint256))
func (_BigNumbers *BigNumbersCallerSession) Zero() (BigNumber, error) {
	return _BigNumbers.Contract.Zero(&_BigNumbers.CallOpts)
}

// ElementsMetaData contains all meta data concerning the Elements contract.
var ElementsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"a\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"b\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"carryIn\",\"type\":\"uint64\"}],\"name\":\"Add64\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"sum\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"carryOut\",\"type\":\"uint64\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ElementFromBytes\",\"outputs\":[{\"components\":[{\"internalType\":\"uint64[6]\",\"name\":\"val\",\"type\":\"uint64[6]\"}],\"internalType\":\"structElements.Element\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"a\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"b\",\"type\":\"uint64\"}],\"name\":\"Mul64\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"high\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"low\",\"type\":\"uint64\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64[6]\",\"name\":\"val\",\"type\":\"uint64[6]\"}],\"internalType\":\"structElements.Element\",\"name\":\"x\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint64[6]\",\"name\":\"val\",\"type\":\"uint64[6]\"}],\"internalType\":\"structElements.Element\",\"name\":\"y\",\"type\":\"tuple\"}],\"name\":\"mul\",\"outputs\":[{\"components\":[{\"internalType\":\"uint64[6]\",\"name\":\"val\",\"type\":\"uint64[6]\"}],\"internalType\":\"structElements.Element\",\"name\":\"z\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint64[6]\",\"name\":\"val\",\"type\":\"uint64[6]\"}],\"internalType\":\"structElements.Element\",\"name\":\"z\",\"type\":\"tuple\"}],\"name\":\"toMont\",\"outputs\":[{\"components\":[{\"internalType\":\"uint64[6]\",\"name\":\"val\",\"type\":\"uint64[6]\"}],\"internalType\":\"structElements.Element\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x61410a61004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe7300000000000000000000000000000000000000003014608060405260043610610060575f3560e01c806327866f79146100645780634350ee1f1461009557806345d00df2146100c5578063e178e41a146100f5578063e430513b14610126575b5f5ffd5b61007e60048036038101906100799190613ad5565b610156565b60405161008c929190613b22565b60405180910390f35b6100af60048036038101906100aa9190613c85565b61019f565b6040516100bc9190613d8b565b60405180910390f35b6100df60048036038101906100da9190613e93565b6102ba565b6040516100ec9190613d8b565b60405180910390f35b61010f600480360381019061010a9190613ed2565b613769565b60405161011d929190613b22565b60405180910390f35b610140600480360381019061013b9190613f22565b6137e4565b60405161014d9190613d8b565b60405180910390f35b5f5f5f8367ffffffffffffffff168567ffffffffffffffff166101799190613f95565b90508091506040816fffffffffffffffffffffffffffffffff16901c9250509250929050565b6101a761397f565b5f600890506101b461397f565b5f600290505b828110156102af575f5f90505f5f90505b600881101561025e575f816008856101e39190613fda565b6101ed919061401b565b9050875181101561025057816007610205919061404e565b60086102119190613fda565b67ffffffffffffffff1688828151811061022e5761022d614081565b5b602001015160f81c60f81b60f81c60ff1667ffffffffffffffff16901b831792505b5080806001019150506101cb565b5080835f0151836007610271919061404e565b6006811061028257610281614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250505080806001019150506101ba565b508092505050919050565b6102c261397f565b6102ca613998565b5f5f5f5f61030c885f01515f600681106102e7576102e6614081565b5b6020020151885f01515f6006811061030257610301614081565b5b6020020151610156565b8660c001875f018267ffffffffffffffff1667ffffffffffffffff168152508267ffffffffffffffff1667ffffffffffffffff168152505050610384885f01515f6006811061035e5761035d614081565b5b6020020151885f015160016006811061037a57610379614081565b5b6020020151610156565b8660e001876020018267ffffffffffffffff1667ffffffffffffffff168152508267ffffffffffffffff1667ffffffffffffffff1681525050506103fd885f01515f600681106103d7576103d6614081565b5b6020020151885f01516002600681106103f3576103f2614081565b5b6020020151610156565b8661010001876040018267ffffffffffffffff1667ffffffffffffffff168152508267ffffffffffffffff1667ffffffffffffffff168152505050610477885f01515f6006811061045157610450614081565b5b6020020151885f015160036006811061046d5761046c614081565b5b6020020151610156565b8661012001876060018267ffffffffffffffff1667ffffffffffffffff168152508267ffffffffffffffff1667ffffffffffffffff1681525050506104f1885f01515f600681106104cb576104ca614081565b5b6020020151885f01516004600681106104e7576104e6614081565b5b6020020151610156565b8661014001876080018267ffffffffffffffff1667ffffffffffffffff168152508267ffffffffffffffff1667ffffffffffffffff16815250505061056b885f01515f6006811061054557610544614081565b5b6020020151885f015160056006811061056157610560614081565b5b6020020151610156565b86610160018760a0018267ffffffffffffffff1667ffffffffffffffff168152508267ffffffffffffffff1667ffffffffffffffff1681525050506105b98560c0015186602001515f613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff1681525050506105ed8560e00151866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050610622856101000151866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff168152505050610657856101200151866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff16815250505061068c8561014001518660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506106bd8561016001515f85613769565b80955081925050505f6106db6789f3fffcfffcfffd875f0151610156565b80925081965050506106f58167b9feffffffffaaab610156565b8760c0018195508267ffffffffffffffff1667ffffffffffffffff168152505050610724865f0151845f613769565b809550819650505061073e81671eabfffeb153ffff610156565b8760e0018195508267ffffffffffffffff1667ffffffffffffffff16815250505061076e86602001518486613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff1681525050506107a081676730d2a0f6b0f624610156565b87610100018195508267ffffffffffffffff1667ffffffffffffffff1681525050506107d186604001518486613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff168152505050610804816764774b84f38512bf610156565b87610120018195508267ffffffffffffffff1667ffffffffffffffff16815250505061083586606001518486613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff16815250505061086881674b1ba7b6434bacd7610156565b87610140018195508267ffffffffffffffff1667ffffffffffffffff16815250505061089986608001518486613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff1681525050506108cc81671a0111ea397fe69a610156565b87610160018195508267ffffffffffffffff1667ffffffffffffffff1681525050506108f95f8486613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff16815250505061092a8661016001515f86613769565b87610160018197508267ffffffffffffffff1667ffffffffffffffff16815250505061095e8660c00151875f01515f613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff1681525050506109918660e00151876020015186613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff1681525050506109c6866101000151876040015186613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff1681525050506109fb866101200151876060015186613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff168152505050610a30866101400151876080015186613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050610a5c825f86613769565b8096508193505050610a778660a0015187608001515f613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050610aa88661016001518386613769565b8760a0018197508267ffffffffffffffff1667ffffffffffffffff168152505050505050505f5f5f610b0f885f0151600160068110610aea57610ae9614081565b5b6020020151885f01515f60068110610b0557610b04614081565b5b6020020151610156565b8660c0018194508267ffffffffffffffff1667ffffffffffffffff168152505050610b3e82865f01515f613769565b865f018195508267ffffffffffffffff1667ffffffffffffffff168152505050610b9e885f0151600160068110610b7857610b77614081565b5b6020020151885f0151600160068110610b9457610b93614081565b5b6020020151610156565b8660e0018194508267ffffffffffffffff1667ffffffffffffffff168152505050610bce82866020015185613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff168152505050610c2f885f0151600160068110610c0957610c08614081565b5b6020020151885f0151600260068110610c2557610c24614081565b5b6020020151610156565b86610100018194508267ffffffffffffffff1667ffffffffffffffff168152505050610c6082866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050610cc1885f0151600160068110610c9b57610c9a614081565b5b6020020151885f0151600360068110610cb757610cb6614081565b5b6020020151610156565b86610120018194508267ffffffffffffffff1667ffffffffffffffff168152505050610cf282866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff168152505050610d53885f0151600160068110610d2d57610d2c614081565b5b6020020151885f0151600460068110610d4957610d48614081565b5b6020020151610156565b86610140018194508267ffffffffffffffff1667ffffffffffffffff168152505050610d8482866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff168152505050610de5885f0151600160068110610dbf57610dbe614081565b5b6020020151885f0151600560068110610ddb57610dda614081565b5b6020020151610156565b86610160018194508267ffffffffffffffff1667ffffffffffffffff168152505050610e16828660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff168152505050610e425f5f85613769565b8095508192505050610e5d8560c0015186602001515f613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff168152505050610e918560e00151866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050610ec6856101000151866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff168152505050610efb856101200151866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff168152505050610f308561014001518660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff168152505050610f618561016001518285613769565b80955081925050505f610f7f6789f3fffcfffcfffd875f0151610156565b8092508196505050610f998167b9feffffffffaaab610156565b8760c0018195508267ffffffffffffffff1667ffffffffffffffff168152505050610fc8865f0151845f613769565b8095508196505050610fe281671eabfffeb153ffff610156565b8760e0018195508267ffffffffffffffff1667ffffffffffffffff16815250505061101286602001518486613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff16815250505061104481676730d2a0f6b0f624610156565b87610100018195508267ffffffffffffffff1667ffffffffffffffff16815250505061107586604001518486613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff1681525050506110a8816764774b84f38512bf610156565b87610120018195508267ffffffffffffffff1667ffffffffffffffff1681525050506110d986606001518486613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff16815250505061110c81674b1ba7b6434bacd7610156565b87610140018195508267ffffffffffffffff1667ffffffffffffffff16815250505061113d86608001518486613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff16815250505061117081671a0111ea397fe69a610156565b87610160018195508267ffffffffffffffff1667ffffffffffffffff16815250505061119d5f8486613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff1681525050506111ce8661016001515f86613769565b87610160018197508267ffffffffffffffff1667ffffffffffffffff1681525050506112028660c00151875f01515f613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff1681525050506112358660e00151876020015186613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff16815250505061126a866101000151876040015186613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff16815250505061129f866101200151876060015186613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff1681525050506112d4866101400151876080015186613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050611300825f86613769565b809650819350505061131b8660a0015187608001515f613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff16815250505061134c8661016001518386613769565b8760a0018197508267ffffffffffffffff1667ffffffffffffffff168152505050505050505f5f5f6113b3885f015160026006811061138e5761138d614081565b5b6020020151885f01515f600681106113a9576113a8614081565b5b6020020151610156565b8660c0018194508267ffffffffffffffff1667ffffffffffffffff1681525050506113e282865f01515f613769565b865f018195508267ffffffffffffffff1667ffffffffffffffff168152505050611442885f015160026006811061141c5761141b614081565b5b6020020151885f015160016006811061143857611437614081565b5b6020020151610156565b8660e0018194508267ffffffffffffffff1667ffffffffffffffff16815250505061147282866020015185613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff1681525050506114d3885f01516002600681106114ad576114ac614081565b5b6020020151885f01516002600681106114c9576114c8614081565b5b6020020151610156565b86610100018194508267ffffffffffffffff1667ffffffffffffffff16815250505061150482866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050611565885f015160026006811061153f5761153e614081565b5b6020020151885f015160036006811061155b5761155a614081565b5b6020020151610156565b86610120018194508267ffffffffffffffff1667ffffffffffffffff16815250505061159682866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff1681525050506115f7885f01516002600681106115d1576115d0614081565b5b6020020151885f01516004600681106115ed576115ec614081565b5b6020020151610156565b86610140018194508267ffffffffffffffff1667ffffffffffffffff16815250505061162882866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff168152505050611689885f015160026006811061166357611662614081565b5b6020020151885f015160056006811061167f5761167e614081565b5b6020020151610156565b86610160018194508267ffffffffffffffff1667ffffffffffffffff1681525050506116ba828660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506116e65f5f85613769565b80955081925050506117018560c0015186602001515f613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff1681525050506117358560e00151866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff16815250505061176a856101000151866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff16815250505061179f856101200151866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff1681525050506117d48561014001518660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506118058561016001518285613769565b80955081925050505f6118236789f3fffcfffcfffd875f0151610156565b809250819650505061183d8167b9feffffffffaaab610156565b8760c0018195508267ffffffffffffffff1667ffffffffffffffff16815250505061186c865f0151845f613769565b809550819650505061188681671eabfffeb153ffff610156565b8760e0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506118b686602001518486613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff1681525050506118e881676730d2a0f6b0f624610156565b87610100018195508267ffffffffffffffff1667ffffffffffffffff16815250505061191986604001518486613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff16815250505061194c816764774b84f38512bf610156565b87610120018195508267ffffffffffffffff1667ffffffffffffffff16815250505061197d86606001518486613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff1681525050506119b081674b1ba7b6434bacd7610156565b87610140018195508267ffffffffffffffff1667ffffffffffffffff1681525050506119e186608001518486613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff168152505050611a1481671a0111ea397fe69a610156565b87610160018195508267ffffffffffffffff1667ffffffffffffffff168152505050611a415f8486613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050611a728661016001515f86613769565b87610160018197508267ffffffffffffffff1667ffffffffffffffff168152505050611aa68660c00151875f01515f613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff168152505050611ad98660e00151876020015186613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff168152505050611b0e866101000151876040015186613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff168152505050611b43866101200151876060015186613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff168152505050611b78866101400151876080015186613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050611ba4825f86613769565b8096508193505050611bbf8660a0015187608001515f613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050611bf08661016001518386613769565b8760a0018197508267ffffffffffffffff1667ffffffffffffffff168152505050505050505f5f5f611c57885f0151600360068110611c3257611c31614081565b5b6020020151885f01515f60068110611c4d57611c4c614081565b5b6020020151610156565b8660c0018194508267ffffffffffffffff1667ffffffffffffffff168152505050611c8682865f01515f613769565b865f018195508267ffffffffffffffff1667ffffffffffffffff168152505050611ce6885f0151600360068110611cc057611cbf614081565b5b6020020151885f0151600160068110611cdc57611cdb614081565b5b6020020151610156565b8660e0018194508267ffffffffffffffff1667ffffffffffffffff168152505050611d1682866020015185613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff168152505050611d77885f0151600360068110611d5157611d50614081565b5b6020020151885f0151600260068110611d6d57611d6c614081565b5b6020020151610156565b86610100018194508267ffffffffffffffff1667ffffffffffffffff168152505050611da882866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050611e09885f0151600360068110611de357611de2614081565b5b6020020151885f0151600360068110611dff57611dfe614081565b5b6020020151610156565b86610120018194508267ffffffffffffffff1667ffffffffffffffff168152505050611e3a82866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff168152505050611e9b885f0151600360068110611e7557611e74614081565b5b6020020151885f0151600460068110611e9157611e90614081565b5b6020020151610156565b86610140018194508267ffffffffffffffff1667ffffffffffffffff168152505050611ecc82866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff168152505050611f2d885f0151600360068110611f0757611f06614081565b5b6020020151885f0151600560068110611f2357611f22614081565b5b6020020151610156565b86610160018194508267ffffffffffffffff1667ffffffffffffffff168152505050611f5e828660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff168152505050611f8a5f5f85613769565b8095508192505050611fa58560c0015186602001515f613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff168152505050611fd98560e00151866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff16815250505061200e856101000151866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff168152505050612043856101200151866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff1681525050506120788561014001518660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506120a98561016001518285613769565b80955081925050505f6120c76789f3fffcfffcfffd875f0151610156565b80925081965050506120e18167b9feffffffffaaab610156565b8760c0018195508267ffffffffffffffff1667ffffffffffffffff168152505050612110865f0151845f613769565b809550819650505061212a81671eabfffeb153ffff610156565b8760e0018195508267ffffffffffffffff1667ffffffffffffffff16815250505061215a86602001518486613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff16815250505061218c81676730d2a0f6b0f624610156565b87610100018195508267ffffffffffffffff1667ffffffffffffffff1681525050506121bd86604001518486613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff1681525050506121f0816764774b84f38512bf610156565b87610120018195508267ffffffffffffffff1667ffffffffffffffff16815250505061222186606001518486613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff16815250505061225481674b1ba7b6434bacd7610156565b87610140018195508267ffffffffffffffff1667ffffffffffffffff16815250505061228586608001518486613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff1681525050506122b881671a0111ea397fe69a610156565b87610160018195508267ffffffffffffffff1667ffffffffffffffff1681525050506122e55f8486613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff1681525050506123168661016001515f86613769565b87610160018197508267ffffffffffffffff1667ffffffffffffffff16815250505061234a8660c00151875f01515f613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff16815250505061237d8660e00151876020015186613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff1681525050506123b2866101000151876040015186613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff1681525050506123e7866101200151876060015186613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff16815250505061241c866101400151876080015186613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050612448825f86613769565b80965081935050506124638660a0015187608001515f613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff1681525050506124948661016001518386613769565b8760a0018197508267ffffffffffffffff1667ffffffffffffffff168152505050505050505f5f5f6124fb885f01516004600681106124d6576124d5614081565b5b6020020151885f01515f600681106124f1576124f0614081565b5b6020020151610156565b8660c0018194508267ffffffffffffffff1667ffffffffffffffff16815250505061252a82865f01515f613769565b865f018195508267ffffffffffffffff1667ffffffffffffffff16815250505061258a885f015160046006811061256457612563614081565b5b6020020151885f01516001600681106125805761257f614081565b5b6020020151610156565b8660e0018194508267ffffffffffffffff1667ffffffffffffffff1681525050506125ba82866020015185613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff16815250505061261b885f01516004600681106125f5576125f4614081565b5b6020020151885f015160026006811061261157612610614081565b5b6020020151610156565b86610100018194508267ffffffffffffffff1667ffffffffffffffff16815250505061264c82866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff1681525050506126ad885f015160046006811061268757612686614081565b5b6020020151885f01516003600681106126a3576126a2614081565b5b6020020151610156565b86610120018194508267ffffffffffffffff1667ffffffffffffffff1681525050506126de82866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff16815250505061273f885f015160046006811061271957612718614081565b5b6020020151885f015160046006811061273557612734614081565b5b6020020151610156565b86610140018194508267ffffffffffffffff1667ffffffffffffffff16815250505061277082866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff1681525050506127d1885f01516004600681106127ab576127aa614081565b5b6020020151885f01516005600681106127c7576127c6614081565b5b6020020151610156565b86610160018194508267ffffffffffffffff1667ffffffffffffffff168152505050612802828660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff16815250505061282e5f5f85613769565b80955081925050506128498560c0015186602001515f613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff16815250505061287d8560e00151866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff1681525050506128b2856101000151866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff1681525050506128e7856101200151866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff16815250505061291c8561014001518660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff16815250505061294d8561016001518285613769565b80955081925050505f61296b6789f3fffcfffcfffd875f0151610156565b80925081965050506129858167b9feffffffffaaab610156565b8760c0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506129b4865f0151845f613769565b80955081965050506129ce81671eabfffeb153ffff610156565b8760e0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506129fe86602001518486613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff168152505050612a3081676730d2a0f6b0f624610156565b87610100018195508267ffffffffffffffff1667ffffffffffffffff168152505050612a6186604001518486613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff168152505050612a94816764774b84f38512bf610156565b87610120018195508267ffffffffffffffff1667ffffffffffffffff168152505050612ac586606001518486613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff168152505050612af881674b1ba7b6434bacd7610156565b87610140018195508267ffffffffffffffff1667ffffffffffffffff168152505050612b2986608001518486613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff168152505050612b5c81671a0111ea397fe69a610156565b87610160018195508267ffffffffffffffff1667ffffffffffffffff168152505050612b895f8486613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050612bba8661016001515f86613769565b87610160018197508267ffffffffffffffff1667ffffffffffffffff168152505050612bee8660c00151875f01515f613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff168152505050612c218660e00151876020015186613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff168152505050612c56866101000151876040015186613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff168152505050612c8b866101200151876060015186613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff168152505050612cc0866101400151876080015186613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050612cec825f86613769565b8096508193505050612d078660a0015187608001515f613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050612d388661016001518386613769565b8760a0018197508267ffffffffffffffff1667ffffffffffffffff168152505050505050505f5f5f612d9f885f0151600560068110612d7a57612d79614081565b5b6020020151885f01515f60068110612d9557612d94614081565b5b6020020151610156565b8660c0018194508267ffffffffffffffff1667ffffffffffffffff168152505050612dce82865f01515f613769565b865f018195508267ffffffffffffffff1667ffffffffffffffff168152505050612e2e885f0151600560068110612e0857612e07614081565b5b6020020151885f0151600160068110612e2457612e23614081565b5b6020020151610156565b8660e0018194508267ffffffffffffffff1667ffffffffffffffff168152505050612e5e82866020015185613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff168152505050612ebf885f0151600560068110612e9957612e98614081565b5b6020020151885f0151600260068110612eb557612eb4614081565b5b6020020151610156565b86610100018194508267ffffffffffffffff1667ffffffffffffffff168152505050612ef082866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050612f51885f0151600560068110612f2b57612f2a614081565b5b6020020151885f0151600360068110612f4757612f46614081565b5b6020020151610156565b86610120018194508267ffffffffffffffff1667ffffffffffffffff168152505050612f8282866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff168152505050612fe3885f0151600560068110612fbd57612fbc614081565b5b6020020151885f0151600460068110612fd957612fd8614081565b5b6020020151610156565b86610140018194508267ffffffffffffffff1667ffffffffffffffff16815250505061301482866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff168152505050613075885f015160056006811061304f5761304e614081565b5b6020020151885f015160056006811061306b5761306a614081565b5b6020020151610156565b86610160018194508267ffffffffffffffff1667ffffffffffffffff1681525050506130a6828660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506130d25f5f85613769565b80955081925050506130ed8560c0015186602001515f613769565b866020018195508267ffffffffffffffff1667ffffffffffffffff1681525050506131218560e00151866040015185613769565b866040018195508267ffffffffffffffff1667ffffffffffffffff168152505050613156856101000151866060015185613769565b866060018195508267ffffffffffffffff1667ffffffffffffffff16815250505061318b856101200151866080015185613769565b866080018195508267ffffffffffffffff1667ffffffffffffffff1681525050506131c08561014001518660a0015185613769565b8660a0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506131f18561016001518285613769565b80955081925050505f61320f6789f3fffcfffcfffd875f0151610156565b80925081965050506132298167b9feffffffffaaab610156565b8760c0018195508267ffffffffffffffff1667ffffffffffffffff168152505050613258865f0151845f613769565b809550819650505061327281671eabfffeb153ffff610156565b8760e0018195508267ffffffffffffffff1667ffffffffffffffff1681525050506132a286602001518486613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff1681525050506132d481676730d2a0f6b0f624610156565b87610100018195508267ffffffffffffffff1667ffffffffffffffff16815250505061330586604001518486613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff168152505050613338816764774b84f38512bf610156565b87610120018195508267ffffffffffffffff1667ffffffffffffffff16815250505061336986606001518486613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff16815250505061339c81674b1ba7b6434bacd7610156565b87610140018195508267ffffffffffffffff1667ffffffffffffffff1681525050506133cd86608001518486613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff16815250505061340081671a0111ea397fe69a610156565b87610160018195508267ffffffffffffffff1667ffffffffffffffff16815250505061342d5f8486613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff16815250505061345e8661016001515f86613769565b87610160018197508267ffffffffffffffff1667ffffffffffffffff1681525050506134928660c00151875f01515f613769565b875f018196508267ffffffffffffffff1667ffffffffffffffff1681525050506134c58660e00151876020015186613769565b876020018196508267ffffffffffffffff1667ffffffffffffffff1681525050506134fa866101000151876040015186613769565b876040018196508267ffffffffffffffff1667ffffffffffffffff16815250505061352f866101200151876060015186613769565b876060018196508267ffffffffffffffff1667ffffffffffffffff168152505050613564866101400151876080015186613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff168152505050613590825f86613769565b80965081935050506135ab8660a0015187608001515f613769565b876080018196508267ffffffffffffffff1667ffffffffffffffff1681525050506135dc8661016001518386613769565b8760a0018197508267ffffffffffffffff1667ffffffffffffffff16815250505050505050815f0151835f01515f6006811061361b5761361a614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250508160200151835f015160016006811061365657613655614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250508160400151835f015160026006811061369157613690614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250508160600151835f01516003600681106136cc576136cb614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250508160800151835f015160046006811061370757613706614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250508160a00151835f015160056006811061374257613741614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff1681525050505092915050565b5f5f5f838587010190508092508567ffffffffffffffff168167ffffffffffffffff1610806137ab57508467ffffffffffffffff168167ffffffffffffffff16105b806137c957508367ffffffffffffffff168167ffffffffffffffff16105b6137d3575f6137d6565b60015b60ff16915050935093915050565b6137ec61397f565b6137f461397f565b67f4df1f341c341746815f01515f6006811061381357613812614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff1681525050670a76e6a609d104f1815f015160016006811061385257613851614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff1681525050678de5476c4c95b6d5815f015160026006811061389157613890614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250506767eb88a9939d83c0815f01516003600681106138d0576138cf614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff1681525050679a793e85b519952d815f015160046006811061390f5761390e614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff16815250506711988fe592cae3aa815f015160056006811061394e5761394d614081565b5b602002019067ffffffffffffffff16908167ffffffffffffffff168152505061397783826102ba565b915050919050565b6040518060200160405280613992613a65565b81525090565b6040518061018001604052805f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681526020015f67ffffffffffffffff1681525090565b6040518060c00160405280600690602082028036833780820191505090505090565b5f604051905090565b5f5ffd5b5f5ffd5b5f67ffffffffffffffff82169050919050565b613ab481613a98565b8114613abe575f5ffd5b50565b5f81359050613acf81613aab565b92915050565b5f5f60408385031215613aeb57613aea613a90565b5b5f613af885828601613ac1565b9250506020613b0985828601613ac1565b9150509250929050565b613b1c81613a98565b82525050565b5f604082019050613b355f830185613b13565b613b426020830184613b13565b9392505050565b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b613b9782613b51565b810181811067ffffffffffffffff82111715613bb657613bb5613b61565b5b80604052505050565b5f613bc8613a87565b9050613bd48282613b8e565b919050565b5f67ffffffffffffffff821115613bf357613bf2613b61565b5b613bfc82613b51565b9050602081019050919050565b828183375f83830152505050565b5f613c29613c2484613bd9565b613bbf565b905082815260208101848484011115613c4557613c44613b4d565b5b613c50848285613c09565b509392505050565b5f82601f830112613c6c57613c6b613b49565b5b8135613c7c848260208601613c17565b91505092915050565b5f60208284031215613c9a57613c99613a90565b5b5f82013567ffffffffffffffff811115613cb757613cb6613a94565b5b613cc384828501613c58565b91505092915050565b5f60069050919050565b5f81905092915050565b5f819050919050565b613cf281613a98565b82525050565b5f613d038383613ce9565b60208301905092915050565b5f602082019050919050565b613d2481613ccc565b613d2e8184613cd6565b9250613d3982613ce0565b805f5b83811015613d69578151613d508782613cf8565b9650613d5b83613d0f565b925050600181019050613d3c565b505050505050565b60c082015f820151613d855f850182613d1b565b50505050565b5f60c082019050613d9e5f830184613d71565b92915050565b5f5ffd5b5f67ffffffffffffffff821115613dc257613dc1613b61565b5b602082029050919050565b5f5ffd5b5f613de3613dde84613da8565b613bbf565b90508060208402830185811115613dfd57613dfc613dcd565b5b835b81811015613e265780613e128882613ac1565b845260208401935050602081019050613dff565b5050509392505050565b5f82601f830112613e4457613e43613b49565b5b6006613e51848285613dd1565b91505092915050565b5f60c08284031215613e6f57613e6e613da4565b5b613e796020613bbf565b90505f613e8884828501613e30565b5f8301525092915050565b5f5f6101808385031215613eaa57613ea9613a90565b5b5f613eb785828601613e5a565b92505060c0613ec885828601613e5a565b9150509250929050565b5f5f5f60608486031215613ee957613ee8613a90565b5b5f613ef686828701613ac1565b9350506020613f0786828701613ac1565b9250506040613f1886828701613ac1565b9150509250925092565b5f60c08284031215613f3757613f36613a90565b5b5f613f4484828501613e5a565b91505092915050565b5f6fffffffffffffffffffffffffffffffff82169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f613f9f82613f4d565b9150613faa83613f4d565b9250828202613fb881613f4d565b9150808214613fca57613fc9613f68565b5b5092915050565b5f819050919050565b5f613fe482613fd1565b9150613fef83613fd1565b9250828202613ffd81613fd1565b9150828204841483151761401457614013613f68565b5b5092915050565b5f61402582613fd1565b915061403083613fd1565b925082820190508082111561404857614047613f68565b5b92915050565b5f61405882613fd1565b915061406383613fd1565b925082820390508181111561407b5761407a613f68565b5b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffdfea26469706673582212207cbc15f1fae5d59fba518ee43031149d330ba0ae2858e0e34324f693de03d11064736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
}

// ElementsABI is the input ABI used to generate the binding from.
// Deprecated: Use ElementsMetaData.ABI instead.
var ElementsABI = ElementsMetaData.ABI

// ElementsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ElementsMetaData.Bin instead.
var ElementsBin = ElementsMetaData.Bin

// DeployElements deploys a new Ethereum contract, binding an instance of Elements to it.
func DeployElements(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Elements, error) {
	parsed, err := ElementsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ElementsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Elements{ElementsCaller: ElementsCaller{contract: contract}, ElementsTransactor: ElementsTransactor{contract: contract}, ElementsFilterer: ElementsFilterer{contract: contract}}, nil
}

// Elements is an auto generated Go binding around an Ethereum contract.
type Elements struct {
	ElementsCaller     // Read-only binding to the contract
	ElementsTransactor // Write-only binding to the contract
	ElementsFilterer   // Log filterer for contract events
}

// ElementsCaller is an auto generated read-only Go binding around an Ethereum contract.
type ElementsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ElementsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ElementsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ElementsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ElementsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ElementsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ElementsSession struct {
	Contract     *Elements         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ElementsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ElementsCallerSession struct {
	Contract *ElementsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ElementsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ElementsTransactorSession struct {
	Contract     *ElementsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ElementsRaw is an auto generated low-level Go binding around an Ethereum contract.
type ElementsRaw struct {
	Contract *Elements // Generic contract binding to access the raw methods on
}

// ElementsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ElementsCallerRaw struct {
	Contract *ElementsCaller // Generic read-only contract binding to access the raw methods on
}

// ElementsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ElementsTransactorRaw struct {
	Contract *ElementsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewElements creates a new instance of Elements, bound to a specific deployed contract.
func NewElements(address common.Address, backend bind.ContractBackend) (*Elements, error) {
	contract, err := bindElements(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Elements{ElementsCaller: ElementsCaller{contract: contract}, ElementsTransactor: ElementsTransactor{contract: contract}, ElementsFilterer: ElementsFilterer{contract: contract}}, nil
}

// NewElementsCaller creates a new read-only instance of Elements, bound to a specific deployed contract.
func NewElementsCaller(address common.Address, caller bind.ContractCaller) (*ElementsCaller, error) {
	contract, err := bindElements(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ElementsCaller{contract: contract}, nil
}

// NewElementsTransactor creates a new write-only instance of Elements, bound to a specific deployed contract.
func NewElementsTransactor(address common.Address, transactor bind.ContractTransactor) (*ElementsTransactor, error) {
	contract, err := bindElements(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ElementsTransactor{contract: contract}, nil
}

// NewElementsFilterer creates a new log filterer instance of Elements, bound to a specific deployed contract.
func NewElementsFilterer(address common.Address, filterer bind.ContractFilterer) (*ElementsFilterer, error) {
	contract, err := bindElements(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ElementsFilterer{contract: contract}, nil
}

// bindElements binds a generic wrapper to an already deployed contract.
func bindElements(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ElementsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Elements *ElementsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Elements.Contract.ElementsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Elements *ElementsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Elements.Contract.ElementsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Elements *ElementsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Elements.Contract.ElementsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Elements *ElementsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Elements.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Elements *ElementsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Elements.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Elements *ElementsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Elements.Contract.contract.Transact(opts, method, params...)
}

// Add64 is a free data retrieval call binding the contract method 0xe178e41a.
//
// Solidity: function Add64(uint64 a, uint64 b, uint64 carryIn) pure returns(uint64 sum, uint64 carryOut)
func (_Elements *ElementsCaller) Add64(opts *bind.CallOpts, a uint64, b uint64, carryIn uint64) (struct {
	Sum      uint64
	CarryOut uint64
}, error) {
	var out []interface{}
	err := _Elements.contract.Call(opts, &out, "Add64", a, b, carryIn)

	outstruct := new(struct {
		Sum      uint64
		CarryOut uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Sum = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.CarryOut = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// Add64 is a free data retrieval call binding the contract method 0xe178e41a.
//
// Solidity: function Add64(uint64 a, uint64 b, uint64 carryIn) pure returns(uint64 sum, uint64 carryOut)
func (_Elements *ElementsSession) Add64(a uint64, b uint64, carryIn uint64) (struct {
	Sum      uint64
	CarryOut uint64
}, error) {
	return _Elements.Contract.Add64(&_Elements.CallOpts, a, b, carryIn)
}

// Add64 is a free data retrieval call binding the contract method 0xe178e41a.
//
// Solidity: function Add64(uint64 a, uint64 b, uint64 carryIn) pure returns(uint64 sum, uint64 carryOut)
func (_Elements *ElementsCallerSession) Add64(a uint64, b uint64, carryIn uint64) (struct {
	Sum      uint64
	CarryOut uint64
}, error) {
	return _Elements.Contract.Add64(&_Elements.CallOpts, a, b, carryIn)
}

// ElementFromBytes is a free data retrieval call binding the contract method 0x4350ee1f.
//
// Solidity: function ElementFromBytes(bytes data) pure returns((uint64[6]))
func (_Elements *ElementsCaller) ElementFromBytes(opts *bind.CallOpts, data []byte) (ElementsElement, error) {
	var out []interface{}
	err := _Elements.contract.Call(opts, &out, "ElementFromBytes", data)

	if err != nil {
		return *new(ElementsElement), err
	}

	out0 := *abi.ConvertType(out[0], new(ElementsElement)).(*ElementsElement)

	return out0, err

}

// ElementFromBytes is a free data retrieval call binding the contract method 0x4350ee1f.
//
// Solidity: function ElementFromBytes(bytes data) pure returns((uint64[6]))
func (_Elements *ElementsSession) ElementFromBytes(data []byte) (ElementsElement, error) {
	return _Elements.Contract.ElementFromBytes(&_Elements.CallOpts, data)
}

// ElementFromBytes is a free data retrieval call binding the contract method 0x4350ee1f.
//
// Solidity: function ElementFromBytes(bytes data) pure returns((uint64[6]))
func (_Elements *ElementsCallerSession) ElementFromBytes(data []byte) (ElementsElement, error) {
	return _Elements.Contract.ElementFromBytes(&_Elements.CallOpts, data)
}

// Mul64 is a free data retrieval call binding the contract method 0x27866f79.
//
// Solidity: function Mul64(uint64 a, uint64 b) pure returns(uint64 high, uint64 low)
func (_Elements *ElementsCaller) Mul64(opts *bind.CallOpts, a uint64, b uint64) (struct {
	High uint64
	Low  uint64
}, error) {
	var out []interface{}
	err := _Elements.contract.Call(opts, &out, "Mul64", a, b)

	outstruct := new(struct {
		High uint64
		Low  uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.High = *abi.ConvertType(out[0], new(uint64)).(*uint64)
	outstruct.Low = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// Mul64 is a free data retrieval call binding the contract method 0x27866f79.
//
// Solidity: function Mul64(uint64 a, uint64 b) pure returns(uint64 high, uint64 low)
func (_Elements *ElementsSession) Mul64(a uint64, b uint64) (struct {
	High uint64
	Low  uint64
}, error) {
	return _Elements.Contract.Mul64(&_Elements.CallOpts, a, b)
}

// Mul64 is a free data retrieval call binding the contract method 0x27866f79.
//
// Solidity: function Mul64(uint64 a, uint64 b) pure returns(uint64 high, uint64 low)
func (_Elements *ElementsCallerSession) Mul64(a uint64, b uint64) (struct {
	High uint64
	Low  uint64
}, error) {
	return _Elements.Contract.Mul64(&_Elements.CallOpts, a, b)
}

// Mul is a free data retrieval call binding the contract method 0x49f26826.
//
// Solidity: function mul((uint64[6]) x, (uint64[6]) y) pure returns((uint64[6]) z)
func (_Elements *ElementsCaller) Mul(opts *bind.CallOpts, x ElementsElement, y ElementsElement) (ElementsElement, error) {
	var out []interface{}
	err := _Elements.contract.Call(opts, &out, "mul", x, y)

	if err != nil {
		return *new(ElementsElement), err
	}

	out0 := *abi.ConvertType(out[0], new(ElementsElement)).(*ElementsElement)

	return out0, err

}

// Mul is a free data retrieval call binding the contract method 0x49f26826.
//
// Solidity: function mul((uint64[6]) x, (uint64[6]) y) pure returns((uint64[6]) z)
func (_Elements *ElementsSession) Mul(x ElementsElement, y ElementsElement) (ElementsElement, error) {
	return _Elements.Contract.Mul(&_Elements.CallOpts, x, y)
}

// Mul is a free data retrieval call binding the contract method 0x49f26826.
//
// Solidity: function mul((uint64[6]) x, (uint64[6]) y) pure returns((uint64[6]) z)
func (_Elements *ElementsCallerSession) Mul(x ElementsElement, y ElementsElement) (ElementsElement, error) {
	return _Elements.Contract.Mul(&_Elements.CallOpts, x, y)
}

// ToMont is a free data retrieval call binding the contract method 0x27e0925c.
//
// Solidity: function toMont((uint64[6]) z) pure returns((uint64[6]))
func (_Elements *ElementsCaller) ToMont(opts *bind.CallOpts, z ElementsElement) (ElementsElement, error) {
	var out []interface{}
	err := _Elements.contract.Call(opts, &out, "toMont", z)

	if err != nil {
		return *new(ElementsElement), err
	}

	out0 := *abi.ConvertType(out[0], new(ElementsElement)).(*ElementsElement)

	return out0, err

}

// ToMont is a free data retrieval call binding the contract method 0x27e0925c.
//
// Solidity: function toMont((uint64[6]) z) pure returns((uint64[6]))
func (_Elements *ElementsSession) ToMont(z ElementsElement) (ElementsElement, error) {
	return _Elements.Contract.ToMont(&_Elements.CallOpts, z)
}

// ToMont is a free data retrieval call binding the contract method 0x27e0925c.
//
// Solidity: function toMont((uint64[6]) z) pure returns((uint64[6]))
func (_Elements *ElementsCallerSession) ToMont(z ElementsElement) (ElementsElement, error) {
	return _Elements.Contract.ToMont(&_Elements.CallOpts, z)
}
