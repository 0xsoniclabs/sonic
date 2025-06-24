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

package driver100

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

// ContractMetaData contains all meta data concerning the Contract contract.
var ContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"AddressEmptyCode\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"ERC1967InvalidImplementation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ERC1967NonPayable\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FailedCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotBackend\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotNode\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UUPSUnauthorizedCallContext\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"slot\",\"type\":\"bytes32\"}],\"name\":\"UUPSUnsupportedProxiableUUID\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"AdvanceEpochs\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"diff\",\"type\":\"bytes\"}],\"name\":\"UpdateNetworkRules\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"}],\"name\":\"UpdateNetworkVersion\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"UpdateValidatorPubkey\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"}],\"name\":\"UpdateValidatorWeight\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"UPGRADE_INTERFACE_VERSION\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"num\",\"type\":\"uint256\"}],\"name\":\"advanceEpochs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"copyCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"status\",\"type\":\"uint256\"}],\"name\":\"deactivateValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"diff\",\"type\":\"uint256\"}],\"name\":\"incNonce\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_backend\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_evmWriterAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proxiableUUID\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"offlineTimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"offlineBlocks\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"uptimes\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"originatedTxsFee\",\"type\":\"uint256[]\"}],\"name\":\"sealEpoch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"nextValidatorIDs\",\"type\":\"uint256[]\"}],\"name\":\"sealEpochValidators\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"setBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"toValidatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"stake\",\"type\":\"uint256\"}],\"name\":\"setGenesisDelegation\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"auth\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"createdTime\",\"type\":\"uint256\"}],\"name\":\"setGenesisValidator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"setStorage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"acc\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"where\",\"type\":\"address\"}],\"name\":\"swapCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"diff\",\"type\":\"bytes\"}],\"name\":\"updateNetworkRules\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"}],\"name\":\"updateNetworkVersion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"pubkey\",\"type\":\"bytes\"}],\"name\":\"updateValidatorPubkey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"validatorID\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"updateValidatorWeight\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newImplementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60a0604052306080523480156012575f5ffd5b5060805161174e6100395f395f8181610c5d01528181610c860152610dca015261174e5ff3fe608060405260043610610131575f3560e01c80638da5cb5b116100a8578063c0c53b8b1161006d578063c0c53b8b1461035d578063d6a0c7af1461037c578063e08d7e661461039b578063e30443bc146103ba578063ebdf104c146103d9578063f2fde38b146103f8575f5ffd5b80638da5cb5b1461027d578063a4066fbe146102c3578063a8ab09ba146102e2578063ad3cb1cc14610301578063b9cc6b1c1461033e575f5ffd5b806339e503ab116100f957806339e503ab146101d25780634f1ef286146101f157806352d1902d14610204578063715018a61461022b57806376fed43a1461023f57806379bead381461025e575f5ffd5b806307690b2a146101355780630aeeca00146101565780631e702f8314610175578063242a6e3f14610194578063267ab446146101b3575b5f5ffd5b348015610140575f5ffd5b5061015461014f366004611129565b610417565b005b348015610161575f5ffd5b5061015461017036600461115a565b6104a8565b348015610180575f5ffd5b5061015461018f366004611171565b610509565b34801561019f575f5ffd5b506101546101ae3660046111d5565b61055f565b3480156101be575f5ffd5b506101546101cd36600461115a565b6105c8565b3480156101dd575f5ffd5b506101546101ec36600461121c565b610622565b6101546101ff366004611260565b6106ba565b34801561020f575f5ffd5b506102186106d9565b6040519081526020015b60405180910390f35b348015610236575f5ffd5b506101546106f4565b34801561024a575f5ffd5b50610154610259366004611321565b610707565b348015610269575f5ffd5b5061015461027836600461137d565b610791565b348015610288575f5ffd5b507f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546040516001600160a01b039091168152602001610222565b3480156102ce575f5ffd5b506101546102dd366004611171565b6107f4565b3480156102ed575f5ffd5b506101546102fc36600461121c565b61085c565b34801561030c575f5ffd5b50610331604051806040016040528060058152602001640352e302e360dc1b81525081565b60405161022291906113a5565b348015610349575f5ffd5b506101546103583660046113da565b6108ba565b348015610368575f5ffd5b50610154610377366004611418565b610921565b348015610387575f5ffd5b50610154610396366004611129565b610a66565b3480156103a6575f5ffd5b506101546103b5366004611498565b610aca565b3480156103c5575f5ffd5b506101546103d436600461137d565b610b1a565b3480156103e4575f5ffd5b506101546103f33660046114ca565b610b7d565b348015610403575f5ffd5b50610154610412366004611594565b610c10565b5f546001600160a01b0316331461044157604051630a31c3dd60e41b815260040160405180910390fd5b6001546040516303b4859560e11b81526001600160a01b0384811660048301528381166024830152909116906307690b2a906044015b5f604051808303815f87803b15801561048e575f5ffd5b505af11580156104a0573d5f5f3e3d5ffd5b505050505050565b5f546001600160a01b031633146104d257604051630a31c3dd60e41b815260040160405180910390fd5b6040518181527f0151256d62457b809bbc891b1f81c6dd0b9987552c70ce915b519750cd434dd1906020015b60405180910390a150565b331561052857604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051631e702f8360e01b815260048101849052602481018390526001600160a01b0390911690631e702f8390604401610477565b5f546001600160a01b0316331461058957604051630a31c3dd60e41b815260040160405180910390fd5b827f0f0ef1ab97439def0a9d2c6d9dc166207f1b13b99e62b442b2993d6153c63a6e83836040516105bb9291906115d5565b60405180910390a2505050565b5f546001600160a01b031633146105f257604051630a31c3dd60e41b815260040160405180910390fd5b6040518181527f2ccdfd47cf0c1f1069d949f1789bb79b2f12821f021634fc835af1de66ea2feb906020016104fe565b5f546001600160a01b0316331461064c57604051630a31c3dd60e41b815260040160405180910390fd5b6001546040516339e503ab60e01b81526001600160a01b0385811660048301526024820185905260448201849052909116906339e503ab906064015b5f604051808303815f87803b15801561069f575f5ffd5b505af11580156106b1573d5f5f3e3d5ffd5b50505050505050565b6106c2610c52565b6106cb82610cf6565b6106d58282610cfe565b5050565b5f6106e2610dbf565b505f5160206116f95f395f51905f5290565b6106fc610e08565b6107055f610e63565b565b331561072657604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051633b7f6a1d60e11b81526001600160a01b03909116906376fed43a9061075d90889088908890889088906004016115f0565b5f604051808303815f87803b158015610774575f5ffd5b505af1158015610786573d5f5f3e3d5ffd5b505050505050505050565b5f546001600160a01b031633146107bb57604051630a31c3dd60e41b815260040160405180910390fd5b600154604051630f37d5a760e31b81526001600160a01b03848116600483015260248201849052909116906379bead3890604401610477565b5f546001600160a01b0316331461081e57604051630a31c3dd60e41b815260040160405180910390fd5b817fb975807576e3b1461be7de07ebf7d20e4790ed802d7a0c4fdd0a1a13df72a9358260405161085091815260200190565b60405180910390a25050565b331561087b57604051630b9a4d6d60e31b815260040160405180910390fd5b5f5460405163545584dd60e11b81526001600160a01b03858116600483015260248201859052604482018490529091169063a8ab09ba90606401610688565b5f546001600160a01b031633146108e457604051630a31c3dd60e41b815260040160405180910390fd5b7f47d10eed096a44e3d0abc586c7e3a5d6cb5358cc90e7d437cd0627f7e765fb9982826040516109159291906115d5565b60405180910390a15050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a008054600160401b810460ff1615906001600160401b03165f811580156109655750825b90505f826001600160401b031660011480156109805750303b155b90508115801561098e575080155b156109ac5760405163f92ee8a960e01b815260040160405180910390fd5b845467ffffffffffffffff1916600117855583156109d657845460ff60401b1916600160401b1785555b6109df86610ed3565b6109e7610ee4565b5f80546001600160a01b03808b166001600160a01b03199283161790925560018054928a16929091169190911790558315610a5c57845460ff60401b19168555604051600181527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15b5050505050505050565b5f546001600160a01b03163314610a9057604051630a31c3dd60e41b815260040160405180910390fd5b60015460405163d6a0c7af60e01b81526001600160a01b03848116600483015283811660248301529091169063d6a0c7af90604401610477565b3315610ae957604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051637046bf3360e11b81526001600160a01b039091169063e08d7e66906104779085908590600401611659565b5f546001600160a01b03163314610b4457604051630a31c3dd60e41b815260040160405180910390fd5b6001546040516338c110ef60e21b81526001600160a01b038481166004830152602482018490529091169063e30443bc90604401610477565b3315610b9c57604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051633af7c41360e21b81526001600160a01b039091169063ebdf104c90610bd9908b908b908b908b908b908b908b908b9060040161166c565b5f604051808303815f87803b158015610bf0575f5ffd5b505af1158015610c02573d5f5f3e3d5ffd5b505050505050505050505050565b610c18610e08565b6001600160a01b038116610c4657604051631e4fbdf760e01b81525f60048201526024015b60405180910390fd5b610c4f81610e63565b50565b306001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000161480610cd857507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316610ccc5f5160206116f95f395f51905f52546001600160a01b031690565b6001600160a01b031614155b156107055760405163703e46dd60e11b815260040160405180910390fd5b610c4f610e08565b816001600160a01b03166352d1902d6040518163ffffffff1660e01b8152600401602060405180830381865afa925050508015610d58575060408051601f3d908101601f19168201909252610d55918101906116cb565b60015b610d8057604051634c9c8ce360e01b81526001600160a01b0383166004820152602401610c3d565b5f5160206116f95f395f51905f528114610db057604051632a87526960e21b815260048101829052602401610c3d565b610dba8383610eec565b505050565b306001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016146107055760405163703e46dd60e11b815260040160405180910390fd5b33610e3a7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b031690565b6001600160a01b0316146107055760405163118cdaa760e01b8152336004820152602401610c3d565b7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c19930080546001600160a01b031981166001600160a01b03848116918217845560405192169182907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0905f90a3505050565b610edb610f41565b610c4f81610f8a565b610705610f41565b610ef582610f92565b6040516001600160a01b038316907fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b905f90a2805115610f3957610dba8282610ff5565b6106d5611067565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a0054600160401b900460ff1661070557604051631afcd79f60e31b815260040160405180910390fd5b610c18610f41565b806001600160a01b03163b5f03610fc757604051634c9c8ce360e01b81526001600160a01b0382166004820152602401610c3d565b5f5160206116f95f395f51905f5280546001600160a01b0319166001600160a01b0392909216919091179055565b60605f5f846001600160a01b03168460405161101191906116e2565b5f60405180830381855af49150503d805f8114611049576040519150601f19603f3d011682016040523d82523d5f602084013e61104e565b606091505b509150915061105e858383611086565b95945050505050565b34156107055760405163b398979f60e01b815260040160405180910390fd5b60608261109b57611096826110e5565b6110de565b81511580156110b257506001600160a01b0384163b155b156110db57604051639996b31560e01b81526001600160a01b0385166004820152602401610c3d565b50805b9392505050565b8051156110f55780518082602001fd5b60405163d6bda27560e01b815260040160405180910390fd5b80356001600160a01b0381168114611124575f5ffd5b919050565b5f5f6040838503121561113a575f5ffd5b6111438361110e565b91506111516020840161110e565b90509250929050565b5f6020828403121561116a575f5ffd5b5035919050565b5f5f60408385031215611182575f5ffd5b50508035926020909101359150565b5f5f83601f8401126111a1575f5ffd5b5081356001600160401b038111156111b7575f5ffd5b6020830191508360208285010111156111ce575f5ffd5b9250929050565b5f5f5f604084860312156111e7575f5ffd5b8335925060208401356001600160401b03811115611203575f5ffd5b61120f86828701611191565b9497909650939450505050565b5f5f5f6060848603121561122e575f5ffd5b6112378461110e565b95602085013595506040909401359392505050565b634e487b7160e01b5f52604160045260245ffd5b5f5f60408385031215611271575f5ffd5b61127a8361110e565b915060208301356001600160401b03811115611294575f5ffd5b8301601f810185136112a4575f5ffd5b80356001600160401b038111156112bd576112bd61124c565b604051601f8201601f19908116603f011681016001600160401b03811182821017156112eb576112eb61124c565b604052818152828201602001871015611302575f5ffd5b816020840160208301375f602083830101528093505050509250929050565b5f5f5f5f5f60808688031215611335575f5ffd5b61133e8661110e565b94506020860135935060408601356001600160401b0381111561135f575f5ffd5b61136b88828901611191565b96999598509660600135949350505050565b5f5f6040838503121561138e575f5ffd5b6113978361110e565b946020939093013593505050565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b5f5f602083850312156113eb575f5ffd5b82356001600160401b03811115611400575f5ffd5b61140c85828601611191565b90969095509350505050565b5f5f5f6060848603121561142a575f5ffd5b6114338461110e565b92506114416020850161110e565b915061144f6040850161110e565b90509250925092565b5f5f83601f840112611468575f5ffd5b5081356001600160401b0381111561147e575f5ffd5b6020830191508360208260051b85010111156111ce575f5ffd5b5f5f602083850312156114a9575f5ffd5b82356001600160401b038111156114be575f5ffd5b61140c85828601611458565b5f5f5f5f5f5f5f5f6080898b0312156114e1575f5ffd5b88356001600160401b038111156114f6575f5ffd5b6115028b828c01611458565b90995097505060208901356001600160401b03811115611520575f5ffd5b61152c8b828c01611458565b90975095505060408901356001600160401b0381111561154a575f5ffd5b6115568b828c01611458565b90955093505060608901356001600160401b03811115611574575f5ffd5b6115808b828c01611458565b999c989b5096995094979396929594505050565b5f602082840312156115a4575f5ffd5b6110de8261110e565b81835281816020850137505f828201602090810191909152601f909101601f19169091010190565b602081525f6115e86020830184866115ad565b949350505050565b60018060a01b0386168152846020820152608060408201525f6116176080830185876115ad565b90508260608301529695505050505050565b8183525f6001600160fb1b03831115611640575f5ffd5b8260051b80836020870137939093016020019392505050565b602081525f6115e8602083018486611629565b608081525f61167f608083018a8c611629565b828103602084015261169281898b611629565b905082810360408401526116a7818789611629565b905082810360608401526116bc818587611629565b9b9a5050505050505050505050565b5f602082840312156116db575f5ffd5b5051919050565b5f82518060208501845e5f92019182525091905056fe360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbca2646970667358221220b9797b4b462efaed14ce37067eea63553ca12b3b54e3144a585a0f02847285d764736f6c634300081b0033",
}

// ContractABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractMetaData.ABI instead.
var ContractABI = ContractMetaData.ABI

// ContractBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ContractMetaData.Bin instead.
var ContractBin = ContractMetaData.Bin

// DeployContract deploys a new Ethereum contract, binding an instance of Contract to it.
func DeployContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Contract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ContractBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// Contract is an auto generated Go binding around an Ethereum contract.
type Contract struct {
	ContractCaller     // Read-only binding to the contract
	ContractTransactor // Write-only binding to the contract
	ContractFilterer   // Log filterer for contract events
}

// ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractSession struct {
	Contract     *Contract         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractCallerSession struct {
	Contract *ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractTransactorSession struct {
	Contract     *ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractRaw struct {
	Contract *Contract // Generic contract binding to access the raw methods on
}

// ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractCallerRaw struct {
	Contract *ContractCaller // Generic read-only contract binding to access the raw methods on
}

// ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractTransactorRaw struct {
	Contract *ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address common.Address, backend bind.ContractBackend) (*Contract, error) {
	contract, err := bindContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// NewContractCaller creates a new read-only instance of Contract, bound to a specific deployed contract.
func NewContractCaller(address common.Address, caller bind.ContractCaller) (*ContractCaller, error) {
	contract, err := bindContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractCaller{contract: contract}, nil
}

// NewContractTransactor creates a new write-only instance of Contract, bound to a specific deployed contract.
func NewContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractTransactor, error) {
	contract, err := bindContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{contract: contract}, nil
}

// NewContractFilterer creates a new log filterer instance of Contract, bound to a specific deployed contract.
func NewContractFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractFilterer, error) {
	contract, err := bindContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractFilterer{contract: contract}, nil
}

// bindContract binds a generic wrapper to an already deployed contract.
func bindContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transact(opts, method, params...)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_Contract *ContractCaller) UPGRADEINTERFACEVERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "UPGRADE_INTERFACE_VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_Contract *ContractSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _Contract.Contract.UPGRADEINTERFACEVERSION(&_Contract.CallOpts)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_Contract *ContractCallerSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _Contract.Contract.UPGRADEINTERFACEVERSION(&_Contract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractSession) Owner() (common.Address, error) {
	return _Contract.Contract.Owner(&_Contract.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Contract *ContractCallerSession) Owner() (common.Address, error) {
	return _Contract.Contract.Owner(&_Contract.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_Contract *ContractCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_Contract *ContractSession) ProxiableUUID() ([32]byte, error) {
	return _Contract.Contract.ProxiableUUID(&_Contract.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_Contract *ContractCallerSession) ProxiableUUID() ([32]byte, error) {
	return _Contract.Contract.ProxiableUUID(&_Contract.CallOpts)
}

// AdvanceEpochs is a paid mutator transaction binding the contract method 0x0aeeca00.
//
// Solidity: function advanceEpochs(uint256 num) returns()
func (_Contract *ContractTransactor) AdvanceEpochs(opts *bind.TransactOpts, num *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "advanceEpochs", num)
}

// AdvanceEpochs is a paid mutator transaction binding the contract method 0x0aeeca00.
//
// Solidity: function advanceEpochs(uint256 num) returns()
func (_Contract *ContractSession) AdvanceEpochs(num *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.AdvanceEpochs(&_Contract.TransactOpts, num)
}

// AdvanceEpochs is a paid mutator transaction binding the contract method 0x0aeeca00.
//
// Solidity: function advanceEpochs(uint256 num) returns()
func (_Contract *ContractTransactorSession) AdvanceEpochs(num *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.AdvanceEpochs(&_Contract.TransactOpts, num)
}

// CopyCode is a paid mutator transaction binding the contract method 0xd6a0c7af.
//
// Solidity: function copyCode(address acc, address from) returns()
func (_Contract *ContractTransactor) CopyCode(opts *bind.TransactOpts, acc common.Address, from common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "copyCode", acc, from)
}

// CopyCode is a paid mutator transaction binding the contract method 0xd6a0c7af.
//
// Solidity: function copyCode(address acc, address from) returns()
func (_Contract *ContractSession) CopyCode(acc common.Address, from common.Address) (*types.Transaction, error) {
	return _Contract.Contract.CopyCode(&_Contract.TransactOpts, acc, from)
}

// CopyCode is a paid mutator transaction binding the contract method 0xd6a0c7af.
//
// Solidity: function copyCode(address acc, address from) returns()
func (_Contract *ContractTransactorSession) CopyCode(acc common.Address, from common.Address) (*types.Transaction, error) {
	return _Contract.Contract.CopyCode(&_Contract.TransactOpts, acc, from)
}

// DeactivateValidator is a paid mutator transaction binding the contract method 0x1e702f83.
//
// Solidity: function deactivateValidator(uint256 validatorID, uint256 status) returns()
func (_Contract *ContractTransactor) DeactivateValidator(opts *bind.TransactOpts, validatorID *big.Int, status *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "deactivateValidator", validatorID, status)
}

// DeactivateValidator is a paid mutator transaction binding the contract method 0x1e702f83.
//
// Solidity: function deactivateValidator(uint256 validatorID, uint256 status) returns()
func (_Contract *ContractSession) DeactivateValidator(validatorID *big.Int, status *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.DeactivateValidator(&_Contract.TransactOpts, validatorID, status)
}

// DeactivateValidator is a paid mutator transaction binding the contract method 0x1e702f83.
//
// Solidity: function deactivateValidator(uint256 validatorID, uint256 status) returns()
func (_Contract *ContractTransactorSession) DeactivateValidator(validatorID *big.Int, status *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.DeactivateValidator(&_Contract.TransactOpts, validatorID, status)
}

// IncNonce is a paid mutator transaction binding the contract method 0x79bead38.
//
// Solidity: function incNonce(address acc, uint256 diff) returns()
func (_Contract *ContractTransactor) IncNonce(opts *bind.TransactOpts, acc common.Address, diff *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "incNonce", acc, diff)
}

// IncNonce is a paid mutator transaction binding the contract method 0x79bead38.
//
// Solidity: function incNonce(address acc, uint256 diff) returns()
func (_Contract *ContractSession) IncNonce(acc common.Address, diff *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.IncNonce(&_Contract.TransactOpts, acc, diff)
}

// IncNonce is a paid mutator transaction binding the contract method 0x79bead38.
//
// Solidity: function incNonce(address acc, uint256 diff) returns()
func (_Contract *ContractTransactorSession) IncNonce(acc common.Address, diff *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.IncNonce(&_Contract.TransactOpts, acc, diff)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _backend, address _evmWriterAddress, address _owner) returns()
func (_Contract *ContractTransactor) Initialize(opts *bind.TransactOpts, _backend common.Address, _evmWriterAddress common.Address, _owner common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "initialize", _backend, _evmWriterAddress, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _backend, address _evmWriterAddress, address _owner) returns()
func (_Contract *ContractSession) Initialize(_backend common.Address, _evmWriterAddress common.Address, _owner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, _backend, _evmWriterAddress, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _backend, address _evmWriterAddress, address _owner) returns()
func (_Contract *ContractTransactorSession) Initialize(_backend common.Address, _evmWriterAddress common.Address, _owner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.Initialize(&_Contract.TransactOpts, _backend, _evmWriterAddress, _owner)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractSession) RenounceOwnership() (*types.Transaction, error) {
	return _Contract.Contract.RenounceOwnership(&_Contract.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Contract *ContractTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Contract.Contract.RenounceOwnership(&_Contract.TransactOpts)
}

// SealEpoch is a paid mutator transaction binding the contract method 0xebdf104c.
//
// Solidity: function sealEpoch(uint256[] offlineTimes, uint256[] offlineBlocks, uint256[] uptimes, uint256[] originatedTxsFee) returns()
func (_Contract *ContractTransactor) SealEpoch(opts *bind.TransactOpts, offlineTimes []*big.Int, offlineBlocks []*big.Int, uptimes []*big.Int, originatedTxsFee []*big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "sealEpoch", offlineTimes, offlineBlocks, uptimes, originatedTxsFee)
}

// SealEpoch is a paid mutator transaction binding the contract method 0xebdf104c.
//
// Solidity: function sealEpoch(uint256[] offlineTimes, uint256[] offlineBlocks, uint256[] uptimes, uint256[] originatedTxsFee) returns()
func (_Contract *ContractSession) SealEpoch(offlineTimes []*big.Int, offlineBlocks []*big.Int, uptimes []*big.Int, originatedTxsFee []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpoch(&_Contract.TransactOpts, offlineTimes, offlineBlocks, uptimes, originatedTxsFee)
}

// SealEpoch is a paid mutator transaction binding the contract method 0xebdf104c.
//
// Solidity: function sealEpoch(uint256[] offlineTimes, uint256[] offlineBlocks, uint256[] uptimes, uint256[] originatedTxsFee) returns()
func (_Contract *ContractTransactorSession) SealEpoch(offlineTimes []*big.Int, offlineBlocks []*big.Int, uptimes []*big.Int, originatedTxsFee []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpoch(&_Contract.TransactOpts, offlineTimes, offlineBlocks, uptimes, originatedTxsFee)
}

// SealEpochValidators is a paid mutator transaction binding the contract method 0xe08d7e66.
//
// Solidity: function sealEpochValidators(uint256[] nextValidatorIDs) returns()
func (_Contract *ContractTransactor) SealEpochValidators(opts *bind.TransactOpts, nextValidatorIDs []*big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "sealEpochValidators", nextValidatorIDs)
}

// SealEpochValidators is a paid mutator transaction binding the contract method 0xe08d7e66.
//
// Solidity: function sealEpochValidators(uint256[] nextValidatorIDs) returns()
func (_Contract *ContractSession) SealEpochValidators(nextValidatorIDs []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpochValidators(&_Contract.TransactOpts, nextValidatorIDs)
}

// SealEpochValidators is a paid mutator transaction binding the contract method 0xe08d7e66.
//
// Solidity: function sealEpochValidators(uint256[] nextValidatorIDs) returns()
func (_Contract *ContractTransactorSession) SealEpochValidators(nextValidatorIDs []*big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SealEpochValidators(&_Contract.TransactOpts, nextValidatorIDs)
}

// SetBalance is a paid mutator transaction binding the contract method 0xe30443bc.
//
// Solidity: function setBalance(address acc, uint256 value) returns()
func (_Contract *ContractTransactor) SetBalance(opts *bind.TransactOpts, acc common.Address, value *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setBalance", acc, value)
}

// SetBalance is a paid mutator transaction binding the contract method 0xe30443bc.
//
// Solidity: function setBalance(address acc, uint256 value) returns()
func (_Contract *ContractSession) SetBalance(acc common.Address, value *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetBalance(&_Contract.TransactOpts, acc, value)
}

// SetBalance is a paid mutator transaction binding the contract method 0xe30443bc.
//
// Solidity: function setBalance(address acc, uint256 value) returns()
func (_Contract *ContractTransactorSession) SetBalance(acc common.Address, value *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetBalance(&_Contract.TransactOpts, acc, value)
}

// SetGenesisDelegation is a paid mutator transaction binding the contract method 0xa8ab09ba.
//
// Solidity: function setGenesisDelegation(address delegator, uint256 toValidatorID, uint256 stake) returns()
func (_Contract *ContractTransactor) SetGenesisDelegation(opts *bind.TransactOpts, delegator common.Address, toValidatorID *big.Int, stake *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setGenesisDelegation", delegator, toValidatorID, stake)
}

// SetGenesisDelegation is a paid mutator transaction binding the contract method 0xa8ab09ba.
//
// Solidity: function setGenesisDelegation(address delegator, uint256 toValidatorID, uint256 stake) returns()
func (_Contract *ContractSession) SetGenesisDelegation(delegator common.Address, toValidatorID *big.Int, stake *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisDelegation(&_Contract.TransactOpts, delegator, toValidatorID, stake)
}

// SetGenesisDelegation is a paid mutator transaction binding the contract method 0xa8ab09ba.
//
// Solidity: function setGenesisDelegation(address delegator, uint256 toValidatorID, uint256 stake) returns()
func (_Contract *ContractTransactorSession) SetGenesisDelegation(delegator common.Address, toValidatorID *big.Int, stake *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisDelegation(&_Contract.TransactOpts, delegator, toValidatorID, stake)
}

// SetGenesisValidator is a paid mutator transaction binding the contract method 0x76fed43a.
//
// Solidity: function setGenesisValidator(address auth, uint256 validatorID, bytes pubkey, uint256 createdTime) returns()
func (_Contract *ContractTransactor) SetGenesisValidator(opts *bind.TransactOpts, auth common.Address, validatorID *big.Int, pubkey []byte, createdTime *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setGenesisValidator", auth, validatorID, pubkey, createdTime)
}

// SetGenesisValidator is a paid mutator transaction binding the contract method 0x76fed43a.
//
// Solidity: function setGenesisValidator(address auth, uint256 validatorID, bytes pubkey, uint256 createdTime) returns()
func (_Contract *ContractSession) SetGenesisValidator(auth common.Address, validatorID *big.Int, pubkey []byte, createdTime *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisValidator(&_Contract.TransactOpts, auth, validatorID, pubkey, createdTime)
}

// SetGenesisValidator is a paid mutator transaction binding the contract method 0x76fed43a.
//
// Solidity: function setGenesisValidator(address auth, uint256 validatorID, bytes pubkey, uint256 createdTime) returns()
func (_Contract *ContractTransactorSession) SetGenesisValidator(auth common.Address, validatorID *big.Int, pubkey []byte, createdTime *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.SetGenesisValidator(&_Contract.TransactOpts, auth, validatorID, pubkey, createdTime)
}

// SetStorage is a paid mutator transaction binding the contract method 0x39e503ab.
//
// Solidity: function setStorage(address acc, bytes32 key, bytes32 value) returns()
func (_Contract *ContractTransactor) SetStorage(opts *bind.TransactOpts, acc common.Address, key [32]byte, value [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setStorage", acc, key, value)
}

// SetStorage is a paid mutator transaction binding the contract method 0x39e503ab.
//
// Solidity: function setStorage(address acc, bytes32 key, bytes32 value) returns()
func (_Contract *ContractSession) SetStorage(acc common.Address, key [32]byte, value [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetStorage(&_Contract.TransactOpts, acc, key, value)
}

// SetStorage is a paid mutator transaction binding the contract method 0x39e503ab.
//
// Solidity: function setStorage(address acc, bytes32 key, bytes32 value) returns()
func (_Contract *ContractTransactorSession) SetStorage(acc common.Address, key [32]byte, value [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetStorage(&_Contract.TransactOpts, acc, key, value)
}

// SwapCode is a paid mutator transaction binding the contract method 0x07690b2a.
//
// Solidity: function swapCode(address acc, address where) returns()
func (_Contract *ContractTransactor) SwapCode(opts *bind.TransactOpts, acc common.Address, where common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "swapCode", acc, where)
}

// SwapCode is a paid mutator transaction binding the contract method 0x07690b2a.
//
// Solidity: function swapCode(address acc, address where) returns()
func (_Contract *ContractSession) SwapCode(acc common.Address, where common.Address) (*types.Transaction, error) {
	return _Contract.Contract.SwapCode(&_Contract.TransactOpts, acc, where)
}

// SwapCode is a paid mutator transaction binding the contract method 0x07690b2a.
//
// Solidity: function swapCode(address acc, address where) returns()
func (_Contract *ContractTransactorSession) SwapCode(acc common.Address, where common.Address) (*types.Transaction, error) {
	return _Contract.Contract.SwapCode(&_Contract.TransactOpts, acc, where)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.TransferOwnership(&_Contract.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Contract *ContractTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Contract.Contract.TransferOwnership(&_Contract.TransactOpts, newOwner)
}

// UpdateNetworkRules is a paid mutator transaction binding the contract method 0xb9cc6b1c.
//
// Solidity: function updateNetworkRules(bytes diff) returns()
func (_Contract *ContractTransactor) UpdateNetworkRules(opts *bind.TransactOpts, diff []byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateNetworkRules", diff)
}

// UpdateNetworkRules is a paid mutator transaction binding the contract method 0xb9cc6b1c.
//
// Solidity: function updateNetworkRules(bytes diff) returns()
func (_Contract *ContractSession) UpdateNetworkRules(diff []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateNetworkRules(&_Contract.TransactOpts, diff)
}

// UpdateNetworkRules is a paid mutator transaction binding the contract method 0xb9cc6b1c.
//
// Solidity: function updateNetworkRules(bytes diff) returns()
func (_Contract *ContractTransactorSession) UpdateNetworkRules(diff []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateNetworkRules(&_Contract.TransactOpts, diff)
}

// UpdateNetworkVersion is a paid mutator transaction binding the contract method 0x267ab446.
//
// Solidity: function updateNetworkVersion(uint256 version) returns()
func (_Contract *ContractTransactor) UpdateNetworkVersion(opts *bind.TransactOpts, version *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateNetworkVersion", version)
}

// UpdateNetworkVersion is a paid mutator transaction binding the contract method 0x267ab446.
//
// Solidity: function updateNetworkVersion(uint256 version) returns()
func (_Contract *ContractSession) UpdateNetworkVersion(version *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateNetworkVersion(&_Contract.TransactOpts, version)
}

// UpdateNetworkVersion is a paid mutator transaction binding the contract method 0x267ab446.
//
// Solidity: function updateNetworkVersion(uint256 version) returns()
func (_Contract *ContractTransactorSession) UpdateNetworkVersion(version *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateNetworkVersion(&_Contract.TransactOpts, version)
}

// UpdateValidatorPubkey is a paid mutator transaction binding the contract method 0x242a6e3f.
//
// Solidity: function updateValidatorPubkey(uint256 validatorID, bytes pubkey) returns()
func (_Contract *ContractTransactor) UpdateValidatorPubkey(opts *bind.TransactOpts, validatorID *big.Int, pubkey []byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateValidatorPubkey", validatorID, pubkey)
}

// UpdateValidatorPubkey is a paid mutator transaction binding the contract method 0x242a6e3f.
//
// Solidity: function updateValidatorPubkey(uint256 validatorID, bytes pubkey) returns()
func (_Contract *ContractSession) UpdateValidatorPubkey(validatorID *big.Int, pubkey []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateValidatorPubkey(&_Contract.TransactOpts, validatorID, pubkey)
}

// UpdateValidatorPubkey is a paid mutator transaction binding the contract method 0x242a6e3f.
//
// Solidity: function updateValidatorPubkey(uint256 validatorID, bytes pubkey) returns()
func (_Contract *ContractTransactorSession) UpdateValidatorPubkey(validatorID *big.Int, pubkey []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpdateValidatorPubkey(&_Contract.TransactOpts, validatorID, pubkey)
}

// UpdateValidatorWeight is a paid mutator transaction binding the contract method 0xa4066fbe.
//
// Solidity: function updateValidatorWeight(uint256 validatorID, uint256 value) returns()
func (_Contract *ContractTransactor) UpdateValidatorWeight(opts *bind.TransactOpts, validatorID *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "updateValidatorWeight", validatorID, value)
}

// UpdateValidatorWeight is a paid mutator transaction binding the contract method 0xa4066fbe.
//
// Solidity: function updateValidatorWeight(uint256 validatorID, uint256 value) returns()
func (_Contract *ContractSession) UpdateValidatorWeight(validatorID *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateValidatorWeight(&_Contract.TransactOpts, validatorID, value)
}

// UpdateValidatorWeight is a paid mutator transaction binding the contract method 0xa4066fbe.
//
// Solidity: function updateValidatorWeight(uint256 validatorID, uint256 value) returns()
func (_Contract *ContractTransactorSession) UpdateValidatorWeight(validatorID *big.Int, value *big.Int) (*types.Transaction, error) {
	return _Contract.Contract.UpdateValidatorWeight(&_Contract.TransactOpts, validatorID, value)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_Contract *ContractTransactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "upgradeToAndCall", newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_Contract *ContractSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpgradeToAndCall(&_Contract.TransactOpts, newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_Contract *ContractTransactorSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _Contract.Contract.UpgradeToAndCall(&_Contract.TransactOpts, newImplementation, data)
}

// ContractAdvanceEpochsIterator is returned from FilterAdvanceEpochs and is used to iterate over the raw logs and unpacked data for AdvanceEpochs events raised by the Contract contract.
type ContractAdvanceEpochsIterator struct {
	Event *ContractAdvanceEpochs // Event containing the contract specifics and raw log

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
func (it *ContractAdvanceEpochsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractAdvanceEpochs)
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
		it.Event = new(ContractAdvanceEpochs)
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
func (it *ContractAdvanceEpochsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractAdvanceEpochsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractAdvanceEpochs represents a AdvanceEpochs event raised by the Contract contract.
type ContractAdvanceEpochs struct {
	Num *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterAdvanceEpochs is a free log retrieval operation binding the contract event 0x0151256d62457b809bbc891b1f81c6dd0b9987552c70ce915b519750cd434dd1.
//
// Solidity: event AdvanceEpochs(uint256 num)
func (_Contract *ContractFilterer) FilterAdvanceEpochs(opts *bind.FilterOpts) (*ContractAdvanceEpochsIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "AdvanceEpochs")
	if err != nil {
		return nil, err
	}
	return &ContractAdvanceEpochsIterator{contract: _Contract.contract, event: "AdvanceEpochs", logs: logs, sub: sub}, nil
}

// WatchAdvanceEpochs is a free log subscription operation binding the contract event 0x0151256d62457b809bbc891b1f81c6dd0b9987552c70ce915b519750cd434dd1.
//
// Solidity: event AdvanceEpochs(uint256 num)
func (_Contract *ContractFilterer) WatchAdvanceEpochs(opts *bind.WatchOpts, sink chan<- *ContractAdvanceEpochs) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "AdvanceEpochs")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractAdvanceEpochs)
				if err := _Contract.contract.UnpackLog(event, "AdvanceEpochs", log); err != nil {
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

// ParseAdvanceEpochs is a log parse operation binding the contract event 0x0151256d62457b809bbc891b1f81c6dd0b9987552c70ce915b519750cd434dd1.
//
// Solidity: event AdvanceEpochs(uint256 num)
func (_Contract *ContractFilterer) ParseAdvanceEpochs(log types.Log) (*ContractAdvanceEpochs, error) {
	event := new(ContractAdvanceEpochs)
	if err := _Contract.contract.UnpackLog(event, "AdvanceEpochs", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Contract contract.
type ContractInitializedIterator struct {
	Event *ContractInitialized // Event containing the contract specifics and raw log

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
func (it *ContractInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractInitialized)
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
		it.Event = new(ContractInitialized)
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
func (it *ContractInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractInitialized represents a Initialized event raised by the Contract contract.
type ContractInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Contract *ContractFilterer) FilterInitialized(opts *bind.FilterOpts) (*ContractInitializedIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ContractInitializedIterator{contract: _Contract.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Contract *ContractFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ContractInitialized) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractInitialized)
				if err := _Contract.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Contract *ContractFilterer) ParseInitialized(log types.Log) (*ContractInitialized, error) {
	event := new(ContractInitialized)
	if err := _Contract.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Contract contract.
type ContractOwnershipTransferredIterator struct {
	Event *ContractOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ContractOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractOwnershipTransferred)
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
		it.Event = new(ContractOwnershipTransferred)
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
func (it *ContractOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractOwnershipTransferred represents a OwnershipTransferred event raised by the Contract contract.
type ContractOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ContractOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ContractOwnershipTransferredIterator{contract: _Contract.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ContractOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractOwnershipTransferred)
				if err := _Contract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Contract *ContractFilterer) ParseOwnershipTransferred(log types.Log) (*ContractOwnershipTransferred, error) {
	event := new(ContractOwnershipTransferred)
	if err := _Contract.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateNetworkRulesIterator is returned from FilterUpdateNetworkRules and is used to iterate over the raw logs and unpacked data for UpdateNetworkRules events raised by the Contract contract.
type ContractUpdateNetworkRulesIterator struct {
	Event *ContractUpdateNetworkRules // Event containing the contract specifics and raw log

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
func (it *ContractUpdateNetworkRulesIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateNetworkRules)
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
		it.Event = new(ContractUpdateNetworkRules)
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
func (it *ContractUpdateNetworkRulesIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateNetworkRulesIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateNetworkRules represents a UpdateNetworkRules event raised by the Contract contract.
type ContractUpdateNetworkRules struct {
	Diff []byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterUpdateNetworkRules is a free log retrieval operation binding the contract event 0x47d10eed096a44e3d0abc586c7e3a5d6cb5358cc90e7d437cd0627f7e765fb99.
//
// Solidity: event UpdateNetworkRules(bytes diff)
func (_Contract *ContractFilterer) FilterUpdateNetworkRules(opts *bind.FilterOpts) (*ContractUpdateNetworkRulesIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateNetworkRules")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateNetworkRulesIterator{contract: _Contract.contract, event: "UpdateNetworkRules", logs: logs, sub: sub}, nil
}

// WatchUpdateNetworkRules is a free log subscription operation binding the contract event 0x47d10eed096a44e3d0abc586c7e3a5d6cb5358cc90e7d437cd0627f7e765fb99.
//
// Solidity: event UpdateNetworkRules(bytes diff)
func (_Contract *ContractFilterer) WatchUpdateNetworkRules(opts *bind.WatchOpts, sink chan<- *ContractUpdateNetworkRules) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateNetworkRules")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateNetworkRules)
				if err := _Contract.contract.UnpackLog(event, "UpdateNetworkRules", log); err != nil {
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

// ParseUpdateNetworkRules is a log parse operation binding the contract event 0x47d10eed096a44e3d0abc586c7e3a5d6cb5358cc90e7d437cd0627f7e765fb99.
//
// Solidity: event UpdateNetworkRules(bytes diff)
func (_Contract *ContractFilterer) ParseUpdateNetworkRules(log types.Log) (*ContractUpdateNetworkRules, error) {
	event := new(ContractUpdateNetworkRules)
	if err := _Contract.contract.UnpackLog(event, "UpdateNetworkRules", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateNetworkVersionIterator is returned from FilterUpdateNetworkVersion and is used to iterate over the raw logs and unpacked data for UpdateNetworkVersion events raised by the Contract contract.
type ContractUpdateNetworkVersionIterator struct {
	Event *ContractUpdateNetworkVersion // Event containing the contract specifics and raw log

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
func (it *ContractUpdateNetworkVersionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateNetworkVersion)
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
		it.Event = new(ContractUpdateNetworkVersion)
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
func (it *ContractUpdateNetworkVersionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateNetworkVersionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateNetworkVersion represents a UpdateNetworkVersion event raised by the Contract contract.
type ContractUpdateNetworkVersion struct {
	Version *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUpdateNetworkVersion is a free log retrieval operation binding the contract event 0x2ccdfd47cf0c1f1069d949f1789bb79b2f12821f021634fc835af1de66ea2feb.
//
// Solidity: event UpdateNetworkVersion(uint256 version)
func (_Contract *ContractFilterer) FilterUpdateNetworkVersion(opts *bind.FilterOpts) (*ContractUpdateNetworkVersionIterator, error) {

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateNetworkVersion")
	if err != nil {
		return nil, err
	}
	return &ContractUpdateNetworkVersionIterator{contract: _Contract.contract, event: "UpdateNetworkVersion", logs: logs, sub: sub}, nil
}

// WatchUpdateNetworkVersion is a free log subscription operation binding the contract event 0x2ccdfd47cf0c1f1069d949f1789bb79b2f12821f021634fc835af1de66ea2feb.
//
// Solidity: event UpdateNetworkVersion(uint256 version)
func (_Contract *ContractFilterer) WatchUpdateNetworkVersion(opts *bind.WatchOpts, sink chan<- *ContractUpdateNetworkVersion) (event.Subscription, error) {

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateNetworkVersion")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateNetworkVersion)
				if err := _Contract.contract.UnpackLog(event, "UpdateNetworkVersion", log); err != nil {
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

// ParseUpdateNetworkVersion is a log parse operation binding the contract event 0x2ccdfd47cf0c1f1069d949f1789bb79b2f12821f021634fc835af1de66ea2feb.
//
// Solidity: event UpdateNetworkVersion(uint256 version)
func (_Contract *ContractFilterer) ParseUpdateNetworkVersion(log types.Log) (*ContractUpdateNetworkVersion, error) {
	event := new(ContractUpdateNetworkVersion)
	if err := _Contract.contract.UnpackLog(event, "UpdateNetworkVersion", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateValidatorPubkeyIterator is returned from FilterUpdateValidatorPubkey and is used to iterate over the raw logs and unpacked data for UpdateValidatorPubkey events raised by the Contract contract.
type ContractUpdateValidatorPubkeyIterator struct {
	Event *ContractUpdateValidatorPubkey // Event containing the contract specifics and raw log

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
func (it *ContractUpdateValidatorPubkeyIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateValidatorPubkey)
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
		it.Event = new(ContractUpdateValidatorPubkey)
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
func (it *ContractUpdateValidatorPubkeyIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateValidatorPubkeyIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateValidatorPubkey represents a UpdateValidatorPubkey event raised by the Contract contract.
type ContractUpdateValidatorPubkey struct {
	ValidatorID *big.Int
	Pubkey      []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterUpdateValidatorPubkey is a free log retrieval operation binding the contract event 0x0f0ef1ab97439def0a9d2c6d9dc166207f1b13b99e62b442b2993d6153c63a6e.
//
// Solidity: event UpdateValidatorPubkey(uint256 indexed validatorID, bytes pubkey)
func (_Contract *ContractFilterer) FilterUpdateValidatorPubkey(opts *bind.FilterOpts, validatorID []*big.Int) (*ContractUpdateValidatorPubkeyIterator, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateValidatorPubkey", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractUpdateValidatorPubkeyIterator{contract: _Contract.contract, event: "UpdateValidatorPubkey", logs: logs, sub: sub}, nil
}

// WatchUpdateValidatorPubkey is a free log subscription operation binding the contract event 0x0f0ef1ab97439def0a9d2c6d9dc166207f1b13b99e62b442b2993d6153c63a6e.
//
// Solidity: event UpdateValidatorPubkey(uint256 indexed validatorID, bytes pubkey)
func (_Contract *ContractFilterer) WatchUpdateValidatorPubkey(opts *bind.WatchOpts, sink chan<- *ContractUpdateValidatorPubkey, validatorID []*big.Int) (event.Subscription, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateValidatorPubkey", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateValidatorPubkey)
				if err := _Contract.contract.UnpackLog(event, "UpdateValidatorPubkey", log); err != nil {
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

// ParseUpdateValidatorPubkey is a log parse operation binding the contract event 0x0f0ef1ab97439def0a9d2c6d9dc166207f1b13b99e62b442b2993d6153c63a6e.
//
// Solidity: event UpdateValidatorPubkey(uint256 indexed validatorID, bytes pubkey)
func (_Contract *ContractFilterer) ParseUpdateValidatorPubkey(log types.Log) (*ContractUpdateValidatorPubkey, error) {
	event := new(ContractUpdateValidatorPubkey)
	if err := _Contract.contract.UnpackLog(event, "UpdateValidatorPubkey", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpdateValidatorWeightIterator is returned from FilterUpdateValidatorWeight and is used to iterate over the raw logs and unpacked data for UpdateValidatorWeight events raised by the Contract contract.
type ContractUpdateValidatorWeightIterator struct {
	Event *ContractUpdateValidatorWeight // Event containing the contract specifics and raw log

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
func (it *ContractUpdateValidatorWeightIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpdateValidatorWeight)
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
		it.Event = new(ContractUpdateValidatorWeight)
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
func (it *ContractUpdateValidatorWeightIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpdateValidatorWeightIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpdateValidatorWeight represents a UpdateValidatorWeight event raised by the Contract contract.
type ContractUpdateValidatorWeight struct {
	ValidatorID *big.Int
	Weight      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterUpdateValidatorWeight is a free log retrieval operation binding the contract event 0xb975807576e3b1461be7de07ebf7d20e4790ed802d7a0c4fdd0a1a13df72a935.
//
// Solidity: event UpdateValidatorWeight(uint256 indexed validatorID, uint256 weight)
func (_Contract *ContractFilterer) FilterUpdateValidatorWeight(opts *bind.FilterOpts, validatorID []*big.Int) (*ContractUpdateValidatorWeightIterator, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "UpdateValidatorWeight", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return &ContractUpdateValidatorWeightIterator{contract: _Contract.contract, event: "UpdateValidatorWeight", logs: logs, sub: sub}, nil
}

// WatchUpdateValidatorWeight is a free log subscription operation binding the contract event 0xb975807576e3b1461be7de07ebf7d20e4790ed802d7a0c4fdd0a1a13df72a935.
//
// Solidity: event UpdateValidatorWeight(uint256 indexed validatorID, uint256 weight)
func (_Contract *ContractFilterer) WatchUpdateValidatorWeight(opts *bind.WatchOpts, sink chan<- *ContractUpdateValidatorWeight, validatorID []*big.Int) (event.Subscription, error) {

	var validatorIDRule []interface{}
	for _, validatorIDItem := range validatorID {
		validatorIDRule = append(validatorIDRule, validatorIDItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "UpdateValidatorWeight", validatorIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpdateValidatorWeight)
				if err := _Contract.contract.UnpackLog(event, "UpdateValidatorWeight", log); err != nil {
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

// ParseUpdateValidatorWeight is a log parse operation binding the contract event 0xb975807576e3b1461be7de07ebf7d20e4790ed802d7a0c4fdd0a1a13df72a935.
//
// Solidity: event UpdateValidatorWeight(uint256 indexed validatorID, uint256 weight)
func (_Contract *ContractFilterer) ParseUpdateValidatorWeight(log types.Log) (*ContractUpdateValidatorWeight, error) {
	event := new(ContractUpdateValidatorWeight)
	if err := _Contract.contract.UnpackLog(event, "UpdateValidatorWeight", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ContractUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the Contract contract.
type ContractUpgradedIterator struct {
	Event *ContractUpgraded // Event containing the contract specifics and raw log

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
func (it *ContractUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractUpgraded)
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
		it.Event = new(ContractUpgraded)
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
func (it *ContractUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractUpgraded represents a Upgraded event raised by the Contract contract.
type ContractUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Contract *ContractFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*ContractUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &ContractUpgradedIterator{contract: _Contract.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Contract *ContractFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *ContractUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractUpgraded)
				if err := _Contract.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_Contract *ContractFilterer) ParseUpgraded(log types.Log) (*ContractUpgraded, error) {
	event := new(ContractUpgraded)
	if err := _Contract.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

var ContractBinRuntime = "0x608060405260043610610131575f3560e01c80638da5cb5b116100a8578063c0c53b8b1161006d578063c0c53b8b1461035d578063d6a0c7af1461037c578063e08d7e661461039b578063e30443bc146103ba578063ebdf104c146103d9578063f2fde38b146103f8575f5ffd5b80638da5cb5b1461027d578063a4066fbe146102c3578063a8ab09ba146102e2578063ad3cb1cc14610301578063b9cc6b1c1461033e575f5ffd5b806339e503ab116100f957806339e503ab146101d25780634f1ef286146101f157806352d1902d14610204578063715018a61461022b57806376fed43a1461023f57806379bead381461025e575f5ffd5b806307690b2a146101355780630aeeca00146101565780631e702f8314610175578063242a6e3f14610194578063267ab446146101b3575b5f5ffd5b348015610140575f5ffd5b5061015461014f366004611129565b610417565b005b348015610161575f5ffd5b5061015461017036600461115a565b6104a8565b348015610180575f5ffd5b5061015461018f366004611171565b610509565b34801561019f575f5ffd5b506101546101ae3660046111d5565b61055f565b3480156101be575f5ffd5b506101546101cd36600461115a565b6105c8565b3480156101dd575f5ffd5b506101546101ec36600461121c565b610622565b6101546101ff366004611260565b6106ba565b34801561020f575f5ffd5b506102186106d9565b6040519081526020015b60405180910390f35b348015610236575f5ffd5b506101546106f4565b34801561024a575f5ffd5b50610154610259366004611321565b610707565b348015610269575f5ffd5b5061015461027836600461137d565b610791565b348015610288575f5ffd5b507f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546040516001600160a01b039091168152602001610222565b3480156102ce575f5ffd5b506101546102dd366004611171565b6107f4565b3480156102ed575f5ffd5b506101546102fc36600461121c565b61085c565b34801561030c575f5ffd5b50610331604051806040016040528060058152602001640352e302e360dc1b81525081565b60405161022291906113a5565b348015610349575f5ffd5b506101546103583660046113da565b6108ba565b348015610368575f5ffd5b50610154610377366004611418565b610921565b348015610387575f5ffd5b50610154610396366004611129565b610a66565b3480156103a6575f5ffd5b506101546103b5366004611498565b610aca565b3480156103c5575f5ffd5b506101546103d436600461137d565b610b1a565b3480156103e4575f5ffd5b506101546103f33660046114ca565b610b7d565b348015610403575f5ffd5b50610154610412366004611594565b610c10565b5f546001600160a01b0316331461044157604051630a31c3dd60e41b815260040160405180910390fd5b6001546040516303b4859560e11b81526001600160a01b0384811660048301528381166024830152909116906307690b2a906044015b5f604051808303815f87803b15801561048e575f5ffd5b505af11580156104a0573d5f5f3e3d5ffd5b505050505050565b5f546001600160a01b031633146104d257604051630a31c3dd60e41b815260040160405180910390fd5b6040518181527f0151256d62457b809bbc891b1f81c6dd0b9987552c70ce915b519750cd434dd1906020015b60405180910390a150565b331561052857604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051631e702f8360e01b815260048101849052602481018390526001600160a01b0390911690631e702f8390604401610477565b5f546001600160a01b0316331461058957604051630a31c3dd60e41b815260040160405180910390fd5b827f0f0ef1ab97439def0a9d2c6d9dc166207f1b13b99e62b442b2993d6153c63a6e83836040516105bb9291906115d5565b60405180910390a2505050565b5f546001600160a01b031633146105f257604051630a31c3dd60e41b815260040160405180910390fd5b6040518181527f2ccdfd47cf0c1f1069d949f1789bb79b2f12821f021634fc835af1de66ea2feb906020016104fe565b5f546001600160a01b0316331461064c57604051630a31c3dd60e41b815260040160405180910390fd5b6001546040516339e503ab60e01b81526001600160a01b0385811660048301526024820185905260448201849052909116906339e503ab906064015b5f604051808303815f87803b15801561069f575f5ffd5b505af11580156106b1573d5f5f3e3d5ffd5b50505050505050565b6106c2610c52565b6106cb82610cf6565b6106d58282610cfe565b5050565b5f6106e2610dbf565b505f5160206116f95f395f51905f5290565b6106fc610e08565b6107055f610e63565b565b331561072657604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051633b7f6a1d60e11b81526001600160a01b03909116906376fed43a9061075d90889088908890889088906004016115f0565b5f604051808303815f87803b158015610774575f5ffd5b505af1158015610786573d5f5f3e3d5ffd5b505050505050505050565b5f546001600160a01b031633146107bb57604051630a31c3dd60e41b815260040160405180910390fd5b600154604051630f37d5a760e31b81526001600160a01b03848116600483015260248201849052909116906379bead3890604401610477565b5f546001600160a01b0316331461081e57604051630a31c3dd60e41b815260040160405180910390fd5b817fb975807576e3b1461be7de07ebf7d20e4790ed802d7a0c4fdd0a1a13df72a9358260405161085091815260200190565b60405180910390a25050565b331561087b57604051630b9a4d6d60e31b815260040160405180910390fd5b5f5460405163545584dd60e11b81526001600160a01b03858116600483015260248201859052604482018490529091169063a8ab09ba90606401610688565b5f546001600160a01b031633146108e457604051630a31c3dd60e41b815260040160405180910390fd5b7f47d10eed096a44e3d0abc586c7e3a5d6cb5358cc90e7d437cd0627f7e765fb9982826040516109159291906115d5565b60405180910390a15050565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a008054600160401b810460ff1615906001600160401b03165f811580156109655750825b90505f826001600160401b031660011480156109805750303b155b90508115801561098e575080155b156109ac5760405163f92ee8a960e01b815260040160405180910390fd5b845467ffffffffffffffff1916600117855583156109d657845460ff60401b1916600160401b1785555b6109df86610ed3565b6109e7610ee4565b5f80546001600160a01b03808b166001600160a01b03199283161790925560018054928a16929091169190911790558315610a5c57845460ff60401b19168555604051600181527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15b5050505050505050565b5f546001600160a01b03163314610a9057604051630a31c3dd60e41b815260040160405180910390fd5b60015460405163d6a0c7af60e01b81526001600160a01b03848116600483015283811660248301529091169063d6a0c7af90604401610477565b3315610ae957604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051637046bf3360e11b81526001600160a01b039091169063e08d7e66906104779085908590600401611659565b5f546001600160a01b03163314610b4457604051630a31c3dd60e41b815260040160405180910390fd5b6001546040516338c110ef60e21b81526001600160a01b038481166004830152602482018490529091169063e30443bc90604401610477565b3315610b9c57604051630b9a4d6d60e31b815260040160405180910390fd5b5f54604051633af7c41360e21b81526001600160a01b039091169063ebdf104c90610bd9908b908b908b908b908b908b908b908b9060040161166c565b5f604051808303815f87803b158015610bf0575f5ffd5b505af1158015610c02573d5f5f3e3d5ffd5b505050505050505050505050565b610c18610e08565b6001600160a01b038116610c4657604051631e4fbdf760e01b81525f60048201526024015b60405180910390fd5b610c4f81610e63565b50565b306001600160a01b037f0000000000000000000000000000000000000000000000000000000000000000161480610cd857507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316610ccc5f5160206116f95f395f51905f52546001600160a01b031690565b6001600160a01b031614155b156107055760405163703e46dd60e11b815260040160405180910390fd5b610c4f610e08565b816001600160a01b03166352d1902d6040518163ffffffff1660e01b8152600401602060405180830381865afa925050508015610d58575060408051601f3d908101601f19168201909252610d55918101906116cb565b60015b610d8057604051634c9c8ce360e01b81526001600160a01b0383166004820152602401610c3d565b5f5160206116f95f395f51905f528114610db057604051632a87526960e21b815260048101829052602401610c3d565b610dba8383610eec565b505050565b306001600160a01b037f000000000000000000000000000000000000000000000000000000000000000016146107055760405163703e46dd60e11b815260040160405180910390fd5b33610e3a7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c199300546001600160a01b031690565b6001600160a01b0316146107055760405163118cdaa760e01b8152336004820152602401610c3d565b7f9016d09d72d40fdae2fd8ceac6b6234c7706214fd39c1cd1e609a0528c19930080546001600160a01b031981166001600160a01b03848116918217845560405192169182907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0905f90a3505050565b610edb610f41565b610c4f81610f8a565b610705610f41565b610ef582610f92565b6040516001600160a01b038316907fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b905f90a2805115610f3957610dba8282610ff5565b6106d5611067565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a0054600160401b900460ff1661070557604051631afcd79f60e31b815260040160405180910390fd5b610c18610f41565b806001600160a01b03163b5f03610fc757604051634c9c8ce360e01b81526001600160a01b0382166004820152602401610c3d565b5f5160206116f95f395f51905f5280546001600160a01b0319166001600160a01b0392909216919091179055565b60605f5f846001600160a01b03168460405161101191906116e2565b5f60405180830381855af49150503d805f8114611049576040519150601f19603f3d011682016040523d82523d5f602084013e61104e565b606091505b509150915061105e858383611086565b95945050505050565b34156107055760405163b398979f60e01b815260040160405180910390fd5b60608261109b57611096826110e5565b6110de565b81511580156110b257506001600160a01b0384163b155b156110db57604051639996b31560e01b81526001600160a01b0385166004820152602401610c3d565b50805b9392505050565b8051156110f55780518082602001fd5b60405163d6bda27560e01b815260040160405180910390fd5b80356001600160a01b0381168114611124575f5ffd5b919050565b5f5f6040838503121561113a575f5ffd5b6111438361110e565b91506111516020840161110e565b90509250929050565b5f6020828403121561116a575f5ffd5b5035919050565b5f5f60408385031215611182575f5ffd5b50508035926020909101359150565b5f5f83601f8401126111a1575f5ffd5b5081356001600160401b038111156111b7575f5ffd5b6020830191508360208285010111156111ce575f5ffd5b9250929050565b5f5f5f604084860312156111e7575f5ffd5b8335925060208401356001600160401b03811115611203575f5ffd5b61120f86828701611191565b9497909650939450505050565b5f5f5f6060848603121561122e575f5ffd5b6112378461110e565b95602085013595506040909401359392505050565b634e487b7160e01b5f52604160045260245ffd5b5f5f60408385031215611271575f5ffd5b61127a8361110e565b915060208301356001600160401b03811115611294575f5ffd5b8301601f810185136112a4575f5ffd5b80356001600160401b038111156112bd576112bd61124c565b604051601f8201601f19908116603f011681016001600160401b03811182821017156112eb576112eb61124c565b604052818152828201602001871015611302575f5ffd5b816020840160208301375f602083830101528093505050509250929050565b5f5f5f5f5f60808688031215611335575f5ffd5b61133e8661110e565b94506020860135935060408601356001600160401b0381111561135f575f5ffd5b61136b88828901611191565b96999598509660600135949350505050565b5f5f6040838503121561138e575f5ffd5b6113978361110e565b946020939093013593505050565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b5f5f602083850312156113eb575f5ffd5b82356001600160401b03811115611400575f5ffd5b61140c85828601611191565b90969095509350505050565b5f5f5f6060848603121561142a575f5ffd5b6114338461110e565b92506114416020850161110e565b915061144f6040850161110e565b90509250925092565b5f5f83601f840112611468575f5ffd5b5081356001600160401b0381111561147e575f5ffd5b6020830191508360208260051b85010111156111ce575f5ffd5b5f5f602083850312156114a9575f5ffd5b82356001600160401b038111156114be575f5ffd5b61140c85828601611458565b5f5f5f5f5f5f5f5f6080898b0312156114e1575f5ffd5b88356001600160401b038111156114f6575f5ffd5b6115028b828c01611458565b90995097505060208901356001600160401b03811115611520575f5ffd5b61152c8b828c01611458565b90975095505060408901356001600160401b0381111561154a575f5ffd5b6115568b828c01611458565b90955093505060608901356001600160401b03811115611574575f5ffd5b6115808b828c01611458565b999c989b5096995094979396929594505050565b5f602082840312156115a4575f5ffd5b6110de8261110e565b81835281816020850137505f828201602090810191909152601f909101601f19169091010190565b602081525f6115e86020830184866115ad565b949350505050565b60018060a01b0386168152846020820152608060408201525f6116176080830185876115ad565b90508260608301529695505050505050565b8183525f6001600160fb1b03831115611640575f5ffd5b8260051b80836020870137939093016020019392505050565b602081525f6115e8602083018486611629565b608081525f61167f608083018a8c611629565b828103602084015261169281898b611629565b905082810360408401526116a7818789611629565b905082810360608401526116bc818587611629565b9b9a5050505050505050505050565b5f602082840312156116db575f5ffd5b5051919050565b5f82518060208501845e5f92019182525091905056fe360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbca2646970667358221220b9797b4b462efaed14ce37067eea63553ca12b3b54e3144a585a0f02847285d764736f6c634300081b0033"
