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

package ballot

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

// BallotMetaData contains all meta data concerning the Ballot contract.
var BallotMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"proposalNames\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"name\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"voteCount\",\"type\":\"uint256\"}],\"name\":\"NewProposal\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"chairperson\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"voter\",\"type\":\"address\"}],\"name\":\"giveRightToVote\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"proposals\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"name\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"voteCount\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"proposal\",\"type\":\"uint256\"}],\"name\":\"vote\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"voters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"voted\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"delegate\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"vote\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"winnerName\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"winnerName_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"winningProposal\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"winningProposal_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f5ffd5b5060405161137f38038061137f8339818101604052810190610031919061035f565b335f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506001805f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f01819055505f5f90505b81518110156101c45760026040518060400160405280848481518110610102576101016103a6565b5b602002602001015181526020015f815250908060018154018082558091505060019003905f5260205f2090600202015f909190919091505f820151815f01556020820151816001015550503373ffffffffffffffffffffffffffffffffffffffff167f4913a1b403184a1c69ab16947e9f4c7a1e48c069dccde91f2bf550ea77becc5b838381518110610198576101976103a6565b5b60200260200101515f6040516101af92919061042d565b60405180910390a280806001019150506100d9565b5050610454565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610226826101e0565b810181811067ffffffffffffffff82111715610245576102446101f0565b5b80604052505050565b5f6102576101cb565b9050610263828261021d565b919050565b5f67ffffffffffffffff821115610282576102816101f0565b5b602082029050602081019050919050565b5f5ffd5b5f819050919050565b6102a981610297565b81146102b3575f5ffd5b50565b5f815190506102c4816102a0565b92915050565b5f6102dc6102d784610268565b61024e565b905080838252602082019050602084028301858111156102ff576102fe610293565b5b835b81811015610328578061031488826102b6565b845260208401935050602081019050610301565b5050509392505050565b5f82601f830112610346576103456101dc565b5b81516103568482602086016102ca565b91505092915050565b5f60208284031215610374576103736101d4565b5b5f82015167ffffffffffffffff811115610391576103906101d8565b5b61039d84828501610332565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b6103dc81610297565b82525050565b5f819050919050565b5f819050919050565b5f819050919050565b5f61041761041261040d846103e2565b6103f4565b6103eb565b9050919050565b610427816103fd565b82525050565b5f6040820190506104405f8301856103d3565b61044d602083018461041e565b9392505050565b610f1e806104615f395ff3fe608060405234801561000f575f5ffd5b5060043610610086575f3560e01c8063609ff1bd11610059578063609ff1bd146101115780639e7b8d611461012f578063a3ec138d1461014b578063e2ba53f01461017e57610086565b80630121b93f1461008a578063013cf08b146100a65780632e4176cf146100d75780635c19a95c146100f5575b5f5ffd5b6100a4600480360381019061009f9190610998565b61019c565b005b6100c060048036038101906100bb9190610998565b6102d7565b6040516100ce9291906109ea565b60405180910390f35b6100df610306565b6040516100ec9190610a50565b60405180910390f35b61010f600480360381019061010a9190610a93565b61032a565b005b6101196106af565b6040516101269190610abe565b60405180910390f35b61014960048036038101906101449190610a93565b61072d565b005b61016560048036038101906101609190610a93565b6108d9565b6040516101759493929190610af1565b60405180910390f35b610186610931565b6040516101939190610b34565b60405180910390f35b5f60015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2090505f815f015403610221576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161021890610ba7565b60405180910390fd5b806001015f9054906101000a900460ff1615610272576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161026990610c0f565b60405180910390fd5b6001816001015f6101000a81548160ff021916908315150217905550818160020181905550805f0154600283815481106102af576102ae610c2d565b5b905f5260205f2090600202016001015f8282546102cc9190610c87565b925050819055505050565b600281815481106102e6575f80fd5b905f5260205f2090600202015f91509050805f0154908060010154905082565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f60015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f209050806001015f9054906101000a900460ff16156103bb576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103b290610d04565b60405180910390fd5b3373ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610429576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161042090610d6c565b60405180910390fd5b5b5f73ffffffffffffffffffffffffffffffffffffffff1660015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16146105935760015f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1691503373ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff160361058e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161058590610dd4565b60405180910390fd5b61042a565b6001816001015f6101000a81548160ff021916908315150217905550818160010160016101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505f60015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f209050806001015f9054906101000a900460ff161561068d57815f0154600282600201548154811061066457610663610c2d565b5b905f5260205f2090600202016001015f8282546106819190610c87565b925050819055506106aa565b815f0154815f015f8282546106a29190610c87565b925050819055505b505050565b5f5f5f90505f5f90505b6002805490508110156107285781600282815481106106db576106da610c2d565b5b905f5260205f20906002020160010154111561071b576002818154811061070557610704610c2d565b5b905f5260205f2090600202016001015491508092505b80806001019150506106b9565b505090565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146107bb576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107b290610e62565b60405180910390fd5b60015f8273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f206001015f9054906101000a900460ff1615610848576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161083f90610eca565b60405180910390fd5b5f60015f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f015414610892575f5ffd5b6001805f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f018190555050565b6001602052805f5260405f205f91509050805f015490806001015f9054906101000a900460ff16908060010160019054906101000a900473ffffffffffffffffffffffffffffffffffffffff16908060020154905084565b5f600261093c6106af565b8154811061094d5761094c610c2d565b5b905f5260205f2090600202015f0154905090565b5f5ffd5b5f819050919050565b61097781610965565b8114610981575f5ffd5b50565b5f813590506109928161096e565b92915050565b5f602082840312156109ad576109ac610961565b5b5f6109ba84828501610984565b91505092915050565b5f819050919050565b6109d5816109c3565b82525050565b6109e481610965565b82525050565b5f6040820190506109fd5f8301856109cc565b610a0a60208301846109db565b9392505050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610a3a82610a11565b9050919050565b610a4a81610a30565b82525050565b5f602082019050610a635f830184610a41565b92915050565b610a7281610a30565b8114610a7c575f5ffd5b50565b5f81359050610a8d81610a69565b92915050565b5f60208284031215610aa857610aa7610961565b5b5f610ab584828501610a7f565b91505092915050565b5f602082019050610ad15f8301846109db565b92915050565b5f8115159050919050565b610aeb81610ad7565b82525050565b5f608082019050610b045f8301876109db565b610b116020830186610ae2565b610b1e6040830185610a41565b610b2b60608301846109db565b95945050505050565b5f602082019050610b475f8301846109cc565b92915050565b5f82825260208201905092915050565b7f486173206e6f20726967687420746f20766f74650000000000000000000000005f82015250565b5f610b91601483610b4d565b9150610b9c82610b5d565b602082019050919050565b5f6020820190508181035f830152610bbe81610b85565b9050919050565b7f416c726561647920766f7465642e0000000000000000000000000000000000005f82015250565b5f610bf9600e83610b4d565b9150610c0482610bc5565b602082019050919050565b5f6020820190508181035f830152610c2681610bed565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610c9182610965565b9150610c9c83610965565b9250828201905080821115610cb457610cb3610c5a565b5b92915050565b7f596f7520616c726561647920766f7465642e00000000000000000000000000005f82015250565b5f610cee601283610b4d565b9150610cf982610cba565b602082019050919050565b5f6020820190508181035f830152610d1b81610ce2565b9050919050565b7f53656c662d64656c65676174696f6e20697320646973616c6c6f7765642e00005f82015250565b5f610d56601e83610b4d565b9150610d6182610d22565b602082019050919050565b5f6020820190508181035f830152610d8381610d4a565b9050919050565b7f466f756e64206c6f6f7020696e2064656c65676174696f6e2e000000000000005f82015250565b5f610dbe601983610b4d565b9150610dc982610d8a565b602082019050919050565b5f6020820190508181035f830152610deb81610db2565b9050919050565b7f4f6e6c79206368616972706572736f6e2063616e2067697665207269676874205f8201527f746f20766f74652e000000000000000000000000000000000000000000000000602082015250565b5f610e4c602883610b4d565b9150610e5782610df2565b604082019050919050565b5f6020820190508181035f830152610e7981610e40565b9050919050565b7f54686520766f74657220616c726561647920766f7465642e00000000000000005f82015250565b5f610eb4601883610b4d565b9150610ebf82610e80565b602082019050919050565b5f6020820190508181035f830152610ee181610ea8565b905091905056fea26469706673582212200e6ad99452546cf9a4a9f650cdcd23bf80a61a5c3ec9bd821390b37664e7073864736f6c634300081e0033",
}

// BallotABI is the input ABI used to generate the binding from.
// Deprecated: Use BallotMetaData.ABI instead.
var BallotABI = BallotMetaData.ABI

// BallotBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BallotMetaData.Bin instead.
var BallotBin = BallotMetaData.Bin

// DeployBallot deploys a new Ethereum contract, binding an instance of Ballot to it.
func DeployBallot(auth *bind.TransactOpts, backend bind.ContractBackend, proposalNames [][32]byte) (common.Address, *types.Transaction, *Ballot, error) {
	parsed, err := BallotMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BallotBin), backend, proposalNames)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Ballot{BallotCaller: BallotCaller{contract: contract}, BallotTransactor: BallotTransactor{contract: contract}, BallotFilterer: BallotFilterer{contract: contract}}, nil
}

// Ballot is an auto generated Go binding around an Ethereum contract.
type Ballot struct {
	BallotCaller     // Read-only binding to the contract
	BallotTransactor // Write-only binding to the contract
	BallotFilterer   // Log filterer for contract events
}

// BallotCaller is an auto generated read-only Go binding around an Ethereum contract.
type BallotCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BallotTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BallotTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BallotFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BallotFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BallotSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BallotSession struct {
	Contract     *Ballot           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BallotCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BallotCallerSession struct {
	Contract *BallotCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BallotTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BallotTransactorSession struct {
	Contract     *BallotTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BallotRaw is an auto generated low-level Go binding around an Ethereum contract.
type BallotRaw struct {
	Contract *Ballot // Generic contract binding to access the raw methods on
}

// BallotCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BallotCallerRaw struct {
	Contract *BallotCaller // Generic read-only contract binding to access the raw methods on
}

// BallotTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BallotTransactorRaw struct {
	Contract *BallotTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBallot creates a new instance of Ballot, bound to a specific deployed contract.
func NewBallot(address common.Address, backend bind.ContractBackend) (*Ballot, error) {
	contract, err := bindBallot(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Ballot{BallotCaller: BallotCaller{contract: contract}, BallotTransactor: BallotTransactor{contract: contract}, BallotFilterer: BallotFilterer{contract: contract}}, nil
}

// NewBallotCaller creates a new read-only instance of Ballot, bound to a specific deployed contract.
func NewBallotCaller(address common.Address, caller bind.ContractCaller) (*BallotCaller, error) {
	contract, err := bindBallot(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BallotCaller{contract: contract}, nil
}

// NewBallotTransactor creates a new write-only instance of Ballot, bound to a specific deployed contract.
func NewBallotTransactor(address common.Address, transactor bind.ContractTransactor) (*BallotTransactor, error) {
	contract, err := bindBallot(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BallotTransactor{contract: contract}, nil
}

// NewBallotFilterer creates a new log filterer instance of Ballot, bound to a specific deployed contract.
func NewBallotFilterer(address common.Address, filterer bind.ContractFilterer) (*BallotFilterer, error) {
	contract, err := bindBallot(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BallotFilterer{contract: contract}, nil
}

// bindBallot binds a generic wrapper to an already deployed contract.
func bindBallot(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BallotMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ballot *BallotRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ballot.Contract.BallotCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ballot *BallotRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ballot.Contract.BallotTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ballot *BallotRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ballot.Contract.BallotTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ballot *BallotCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ballot.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ballot *BallotTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ballot.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ballot *BallotTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ballot.Contract.contract.Transact(opts, method, params...)
}

// Chairperson is a free data retrieval call binding the contract method 0x2e4176cf.
//
// Solidity: function chairperson() view returns(address)
func (_Ballot *BallotCaller) Chairperson(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Ballot.contract.Call(opts, &out, "chairperson")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Chairperson is a free data retrieval call binding the contract method 0x2e4176cf.
//
// Solidity: function chairperson() view returns(address)
func (_Ballot *BallotSession) Chairperson() (common.Address, error) {
	return _Ballot.Contract.Chairperson(&_Ballot.CallOpts)
}

// Chairperson is a free data retrieval call binding the contract method 0x2e4176cf.
//
// Solidity: function chairperson() view returns(address)
func (_Ballot *BallotCallerSession) Chairperson() (common.Address, error) {
	return _Ballot.Contract.Chairperson(&_Ballot.CallOpts)
}

// Proposals is a free data retrieval call binding the contract method 0x013cf08b.
//
// Solidity: function proposals(uint256 ) view returns(bytes32 name, uint256 voteCount)
func (_Ballot *BallotCaller) Proposals(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Name      [32]byte
	VoteCount *big.Int
}, error) {
	var out []interface{}
	err := _Ballot.contract.Call(opts, &out, "proposals", arg0)

	outstruct := new(struct {
		Name      [32]byte
		VoteCount *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Name = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.VoteCount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Proposals is a free data retrieval call binding the contract method 0x013cf08b.
//
// Solidity: function proposals(uint256 ) view returns(bytes32 name, uint256 voteCount)
func (_Ballot *BallotSession) Proposals(arg0 *big.Int) (struct {
	Name      [32]byte
	VoteCount *big.Int
}, error) {
	return _Ballot.Contract.Proposals(&_Ballot.CallOpts, arg0)
}

// Proposals is a free data retrieval call binding the contract method 0x013cf08b.
//
// Solidity: function proposals(uint256 ) view returns(bytes32 name, uint256 voteCount)
func (_Ballot *BallotCallerSession) Proposals(arg0 *big.Int) (struct {
	Name      [32]byte
	VoteCount *big.Int
}, error) {
	return _Ballot.Contract.Proposals(&_Ballot.CallOpts, arg0)
}

// Voters is a free data retrieval call binding the contract method 0xa3ec138d.
//
// Solidity: function voters(address ) view returns(uint256 weight, bool voted, address delegate, uint256 vote)
func (_Ballot *BallotCaller) Voters(opts *bind.CallOpts, arg0 common.Address) (struct {
	Weight   *big.Int
	Voted    bool
	Delegate common.Address
	Vote     *big.Int
}, error) {
	var out []interface{}
	err := _Ballot.contract.Call(opts, &out, "voters", arg0)

	outstruct := new(struct {
		Weight   *big.Int
		Voted    bool
		Delegate common.Address
		Vote     *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Weight = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Voted = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.Delegate = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.Vote = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Voters is a free data retrieval call binding the contract method 0xa3ec138d.
//
// Solidity: function voters(address ) view returns(uint256 weight, bool voted, address delegate, uint256 vote)
func (_Ballot *BallotSession) Voters(arg0 common.Address) (struct {
	Weight   *big.Int
	Voted    bool
	Delegate common.Address
	Vote     *big.Int
}, error) {
	return _Ballot.Contract.Voters(&_Ballot.CallOpts, arg0)
}

// Voters is a free data retrieval call binding the contract method 0xa3ec138d.
//
// Solidity: function voters(address ) view returns(uint256 weight, bool voted, address delegate, uint256 vote)
func (_Ballot *BallotCallerSession) Voters(arg0 common.Address) (struct {
	Weight   *big.Int
	Voted    bool
	Delegate common.Address
	Vote     *big.Int
}, error) {
	return _Ballot.Contract.Voters(&_Ballot.CallOpts, arg0)
}

// WinnerName is a free data retrieval call binding the contract method 0xe2ba53f0.
//
// Solidity: function winnerName() view returns(bytes32 winnerName_)
func (_Ballot *BallotCaller) WinnerName(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Ballot.contract.Call(opts, &out, "winnerName")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// WinnerName is a free data retrieval call binding the contract method 0xe2ba53f0.
//
// Solidity: function winnerName() view returns(bytes32 winnerName_)
func (_Ballot *BallotSession) WinnerName() ([32]byte, error) {
	return _Ballot.Contract.WinnerName(&_Ballot.CallOpts)
}

// WinnerName is a free data retrieval call binding the contract method 0xe2ba53f0.
//
// Solidity: function winnerName() view returns(bytes32 winnerName_)
func (_Ballot *BallotCallerSession) WinnerName() ([32]byte, error) {
	return _Ballot.Contract.WinnerName(&_Ballot.CallOpts)
}

// WinningProposal is a free data retrieval call binding the contract method 0x609ff1bd.
//
// Solidity: function winningProposal() view returns(uint256 winningProposal_)
func (_Ballot *BallotCaller) WinningProposal(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Ballot.contract.Call(opts, &out, "winningProposal")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WinningProposal is a free data retrieval call binding the contract method 0x609ff1bd.
//
// Solidity: function winningProposal() view returns(uint256 winningProposal_)
func (_Ballot *BallotSession) WinningProposal() (*big.Int, error) {
	return _Ballot.Contract.WinningProposal(&_Ballot.CallOpts)
}

// WinningProposal is a free data retrieval call binding the contract method 0x609ff1bd.
//
// Solidity: function winningProposal() view returns(uint256 winningProposal_)
func (_Ballot *BallotCallerSession) WinningProposal() (*big.Int, error) {
	return _Ballot.Contract.WinningProposal(&_Ballot.CallOpts)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address to) returns()
func (_Ballot *BallotTransactor) Delegate(opts *bind.TransactOpts, to common.Address) (*types.Transaction, error) {
	return _Ballot.contract.Transact(opts, "delegate", to)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address to) returns()
func (_Ballot *BallotSession) Delegate(to common.Address) (*types.Transaction, error) {
	return _Ballot.Contract.Delegate(&_Ballot.TransactOpts, to)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address to) returns()
func (_Ballot *BallotTransactorSession) Delegate(to common.Address) (*types.Transaction, error) {
	return _Ballot.Contract.Delegate(&_Ballot.TransactOpts, to)
}

// GiveRightToVote is a paid mutator transaction binding the contract method 0x9e7b8d61.
//
// Solidity: function giveRightToVote(address voter) returns()
func (_Ballot *BallotTransactor) GiveRightToVote(opts *bind.TransactOpts, voter common.Address) (*types.Transaction, error) {
	return _Ballot.contract.Transact(opts, "giveRightToVote", voter)
}

// GiveRightToVote is a paid mutator transaction binding the contract method 0x9e7b8d61.
//
// Solidity: function giveRightToVote(address voter) returns()
func (_Ballot *BallotSession) GiveRightToVote(voter common.Address) (*types.Transaction, error) {
	return _Ballot.Contract.GiveRightToVote(&_Ballot.TransactOpts, voter)
}

// GiveRightToVote is a paid mutator transaction binding the contract method 0x9e7b8d61.
//
// Solidity: function giveRightToVote(address voter) returns()
func (_Ballot *BallotTransactorSession) GiveRightToVote(voter common.Address) (*types.Transaction, error) {
	return _Ballot.Contract.GiveRightToVote(&_Ballot.TransactOpts, voter)
}

// Vote is a paid mutator transaction binding the contract method 0x0121b93f.
//
// Solidity: function vote(uint256 proposal) returns()
func (_Ballot *BallotTransactor) Vote(opts *bind.TransactOpts, proposal *big.Int) (*types.Transaction, error) {
	return _Ballot.contract.Transact(opts, "vote", proposal)
}

// Vote is a paid mutator transaction binding the contract method 0x0121b93f.
//
// Solidity: function vote(uint256 proposal) returns()
func (_Ballot *BallotSession) Vote(proposal *big.Int) (*types.Transaction, error) {
	return _Ballot.Contract.Vote(&_Ballot.TransactOpts, proposal)
}

// Vote is a paid mutator transaction binding the contract method 0x0121b93f.
//
// Solidity: function vote(uint256 proposal) returns()
func (_Ballot *BallotTransactorSession) Vote(proposal *big.Int) (*types.Transaction, error) {
	return _Ballot.Contract.Vote(&_Ballot.TransactOpts, proposal)
}

// BallotNewProposalIterator is returned from FilterNewProposal and is used to iterate over the raw logs and unpacked data for NewProposal events raised by the Ballot contract.
type BallotNewProposalIterator struct {
	Event *BallotNewProposal // Event containing the contract specifics and raw log

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
func (it *BallotNewProposalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BallotNewProposal)
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
		it.Event = new(BallotNewProposal)
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
func (it *BallotNewProposalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BallotNewProposalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BallotNewProposal represents a NewProposal event raised by the Ballot contract.
type BallotNewProposal struct {
	From      common.Address
	Name      [32]byte
	VoteCount *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterNewProposal is a free log retrieval operation binding the contract event 0x4913a1b403184a1c69ab16947e9f4c7a1e48c069dccde91f2bf550ea77becc5b.
//
// Solidity: event NewProposal(address indexed from, bytes32 name, uint256 voteCount)
func (_Ballot *BallotFilterer) FilterNewProposal(opts *bind.FilterOpts, from []common.Address) (*BallotNewProposalIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _Ballot.contract.FilterLogs(opts, "NewProposal", fromRule)
	if err != nil {
		return nil, err
	}
	return &BallotNewProposalIterator{contract: _Ballot.contract, event: "NewProposal", logs: logs, sub: sub}, nil
}

// WatchNewProposal is a free log subscription operation binding the contract event 0x4913a1b403184a1c69ab16947e9f4c7a1e48c069dccde91f2bf550ea77becc5b.
//
// Solidity: event NewProposal(address indexed from, bytes32 name, uint256 voteCount)
func (_Ballot *BallotFilterer) WatchNewProposal(opts *bind.WatchOpts, sink chan<- *BallotNewProposal, from []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _Ballot.contract.WatchLogs(opts, "NewProposal", fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BallotNewProposal)
				if err := _Ballot.contract.UnpackLog(event, "NewProposal", log); err != nil {
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

// ParseNewProposal is a log parse operation binding the contract event 0x4913a1b403184a1c69ab16947e9f4c7a1e48c069dccde91f2bf550ea77becc5b.
//
// Solidity: event NewProposal(address indexed from, bytes32 name, uint256 voteCount)
func (_Ballot *BallotFilterer) ParseNewProposal(log types.Log) (*BallotNewProposal, error) {
	event := new(BallotNewProposal)
	if err := _Ballot.contract.UnpackLog(event, "NewProposal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
