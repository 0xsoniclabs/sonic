// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package erc3009

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

// Erc3009MetaData contains all meta data concerning the Erc3009 contract.
var Erc3009MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"supply\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"authorizer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"nonce\",\"type\":\"bytes32\"}],\"name\":\"AuthorizationUsed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RECEIVE_WITH_AUTHORIZATION_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TRANSFER_WITH_AUTHORIZATION_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"authorizationState\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validAfter\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validBefore\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"nonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"receiveWithAuthorization\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"sum\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validAfter\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"validBefore\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"nonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"transferWithAuthorization\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561000f575f5ffd5b506040516117d03803806117d0833981810160405281019061003191906102c2565b806100413361018460201b60201c565b805f5f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20819055503373ffffffffffffffffffffffffffffffffffffffff165f73ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040516100df91906102fc565b60405180910390a3507f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f7fccba07d9e4024cd23c2642d7b6ad45c79a5597ed87d13a47bf3e24a94d780e677fc89efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc6463060405160200161016195949392919061036c565b6040516020818303038152906040528051906020012060808181525050506103bd565b60035f8273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff1661028857600160035f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff021916908315150217905550600281908060018154018082558091505060019003905f5260205f20015f9091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505b50565b5f5ffd5b5f819050919050565b6102a18161028f565b81146102ab575f5ffd5b50565b5f815190506102bc81610298565b92915050565b5f602082840312156102d7576102d661028b565b5b5f6102e4848285016102ae565b91505092915050565b6102f68161028f565b82525050565b5f60208201905061030f5f8301846102ed565b92915050565b5f819050919050565b61032781610315565b82525050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6103568261032d565b9050919050565b6103668161034c565b82525050565b5f60a08201905061037f5f83018861031e565b61038c602083018761031e565b610399604083018661031e565b6103a660608301856102ed565b6103b3608083018461035d565b9695505050505050565b6080516113f46103dc5f395f81816104c301526108c001526113f45ff3fe608060405234801561000f575f5ffd5b50600436106100b2575f3560e01c8063a0cc6a681161006f578063a0cc6a68146101a0578063a9059cbb146101be578063dd62ed3e146101ee578063e3ee160e1461021e578063e94a01021461023a578063ef55bec61461026a576100b2565b8063095ea7b3146100b657806318160ddd146100e657806323b872dd146101045780633644e5151461013457806370a08231146101525780637f2eecc314610182575b5f5ffd5b6100d060048036038101906100cb9190610c89565b610286565b6040516100dd9190610ce1565b60405180910390f35b6100ee610373565b6040516100fb9190610d09565b60405180910390f35b61011e60048036038101906101199190610d22565b61041c565b60405161012b9190610ce1565b60405180910390f35b61013c6104c1565b6040516101499190610d8a565b60405180910390f35b61016c60048036038101906101679190610da3565b6104e5565b6040516101799190610d09565b60405180910390f35b61018a6104f9565b6040516101979190610d8a565b60405180910390f35b6101a861051d565b6040516101b59190610d8a565b60405180910390f35b6101d860048036038101906101d39190610c89565b610541565b6040516101e59190610ce1565b60405180910390f35b61020860048036038101906102039190610dce565b610557565b6040516102159190610d09565b60405180910390f35b61023860048036038101906102339190610e6c565b610577565b005b610254600480360381019061024f9190610f30565b6105b4565b6040516102619190610ce1565b60405180910390f35b610284600480360381019061027f9190610e6c565b6105de565b005b5f8160015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040516103619190610d09565b60405180910390a36001905092915050565b5f5f5f90505b600280549050811015610418575f5f6002838154811061039c5761039b610f6e565b5b905f5260205f20015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054826104099190610fc8565b91508080600101915050610379565b5090565b5f8160015f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546104a49190610ffb565b925050819055506104b6848484610689565b600190509392505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b5f602052805f5260405f205f915090505481565b7fd099cc98ef71107a616c4f0f941f04c322d8e254fe26b3c6668db87aae413de881565b7f7c7c6cdb67a18743f49ec6fa9b35f50d52ed05cbed4cc592e13b44501c1a226781565b5f61054d338484610689565b6001905092915050565b6001602052815f5260405f20602052805f5260405f205f91509150505481565b6105a97f7c7c6cdb67a18743f49ec6fa9b35f50d52ed05cbed4cc592e13b44501c1a22678a8a8a8a8a8a8a8a8a6107a0565b505050505050505050565b6004602052815f5260405f20602052805f5260405f205f915091509054906101000a900460ff1681565b3373ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff161461064c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161064390611088565b60405180910390fd5b61067e7fd099cc98ef71107a616c4f0f941f04c322d8e254fe26b3c6668db87aae413de88a8a8a8a8a8a8a8a8a6107a0565b505050505050505050565b805f5f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546106d49190610ffb565b92505081905550805f5f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546107269190610fc8565b9250508190555061073682610af1565b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040516107939190610d09565b60405180910390a3505050565b8542116107e2576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107d9906110f0565b60405180910390fd5b844210610824576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161081b90611158565b60405180910390fd5b60045f8a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8581526020019081526020015f205f9054906101000a900460ff16156108bd576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016108b4906111c0565b60405180910390fd5b5f7f00000000000000000000000000000000000000000000000000000000000000008b8b8b8b8b8b8b6040516020016108fc97969594939291906111ed565b604051602081830303815290604052805190602001206040516020016109239291906112ce565b6040516020818303038152906040528051906020012090505f6001828686866040515f815260200160405260405161095e9493929190611313565b6020604051602081039080840390855afa15801561097e573d5f5f3e3d5ffd5b5050506020604051035190505f73ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141580156109f157508a73ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16145b610a30576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a27906113a0565b60405180910390fd5b600160045f8d73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8881526020019081526020015f205f6101000a81548160ff021916908315150217905550858b73ffffffffffffffffffffffffffffffffffffffff167f98de503528ee59b575ef0c0a2576a82497bfc029a5685b209e9ec333479b10a560405160405180910390a3610ae38b8b8b610689565b505050505050505050505050565b60035f8273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff16610bf557600160035f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff021916908315150217905550600281908060018154018082558091505060019003905f5260205f20015f9091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505b50565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610c2582610bfc565b9050919050565b610c3581610c1b565b8114610c3f575f5ffd5b50565b5f81359050610c5081610c2c565b92915050565b5f819050919050565b610c6881610c56565b8114610c72575f5ffd5b50565b5f81359050610c8381610c5f565b92915050565b5f5f60408385031215610c9f57610c9e610bf8565b5b5f610cac85828601610c42565b9250506020610cbd85828601610c75565b9150509250929050565b5f8115159050919050565b610cdb81610cc7565b82525050565b5f602082019050610cf45f830184610cd2565b92915050565b610d0381610c56565b82525050565b5f602082019050610d1c5f830184610cfa565b92915050565b5f5f5f60608486031215610d3957610d38610bf8565b5b5f610d4686828701610c42565b9350506020610d5786828701610c42565b9250506040610d6886828701610c75565b9150509250925092565b5f819050919050565b610d8481610d72565b82525050565b5f602082019050610d9d5f830184610d7b565b92915050565b5f60208284031215610db857610db7610bf8565b5b5f610dc584828501610c42565b91505092915050565b5f5f60408385031215610de457610de3610bf8565b5b5f610df185828601610c42565b9250506020610e0285828601610c42565b9150509250929050565b610e1581610d72565b8114610e1f575f5ffd5b50565b5f81359050610e3081610e0c565b92915050565b5f60ff82169050919050565b610e4b81610e36565b8114610e55575f5ffd5b50565b5f81359050610e6681610e42565b92915050565b5f5f5f5f5f5f5f5f5f6101208a8c031215610e8a57610e89610bf8565b5b5f610e978c828d01610c42565b9950506020610ea88c828d01610c42565b9850506040610eb98c828d01610c75565b9750506060610eca8c828d01610c75565b9650506080610edb8c828d01610c75565b95505060a0610eec8c828d01610e22565b94505060c0610efd8c828d01610e58565b93505060e0610f0e8c828d01610e22565b925050610100610f208c828d01610e22565b9150509295985092959850929598565b5f5f60408385031215610f4657610f45610bf8565b5b5f610f5385828601610c42565b9250506020610f6485828601610e22565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610fd282610c56565b9150610fdd83610c56565b9250828201905080821115610ff557610ff4610f9b565b5b92915050565b5f61100582610c56565b915061101083610c56565b925082820390508181111561102857611027610f9b565b5b92915050565b5f82825260208201905092915050565b7f63616c6c6572206d7573742062652074686520706179656500000000000000005f82015250565b5f61107260188361102e565b915061107d8261103e565b602082019050919050565b5f6020820190508181035f83015261109f81611066565b9050919050565b7f617574686f72697a6174696f6e206973206e6f74207965742076616c696400005f82015250565b5f6110da601e8361102e565b91506110e5826110a6565b602082019050919050565b5f6020820190508181035f830152611107816110ce565b9050919050565b7f617574686f72697a6174696f6e206973206578706972656400000000000000005f82015250565b5f61114260188361102e565b915061114d8261110e565b602082019050919050565b5f6020820190508181035f83015261116f81611136565b9050919050565b7f617574686f72697a6174696f6e206973207573656400000000000000000000005f82015250565b5f6111aa60158361102e565b91506111b582611176565b602082019050919050565b5f6020820190508181035f8301526111d78161119e565b9050919050565b6111e781610c1b565b82525050565b5f60e0820190506112005f83018a610d7b565b61120d60208301896111de565b61121a60408301886111de565b6112276060830187610cfa565b6112346080830186610cfa565b61124160a0830185610cfa565b61124e60c0830184610d7b565b98975050505050505050565b5f81905092915050565b7f19010000000000000000000000000000000000000000000000000000000000005f82015250565b5f61129860028361125a565b91506112a382611264565b600282019050919050565b5f819050919050565b6112c86112c382610d72565b6112ae565b82525050565b5f6112d88261128c565b91506112e482856112b7565b6020820191506112f482846112b7565b6020820191508190509392505050565b61130d81610e36565b82525050565b5f6080820190506113265f830187610d7b565b6113336020830186611304565b6113406040830185610d7b565b61134d6060830184610d7b565b95945050505050565b7f696e76616c6964207369676e61747572650000000000000000000000000000005f82015250565b5f61138a60118361102e565b915061139582611356565b602082019050919050565b5f6020820190508181035f8301526113b78161137e565b905091905056fea2646970667358221220e2b85dc6ea12d7fde075bfe215f6cd9d2311d7a41f78a8923b058520ab2ca5e964736f6c63430008260033",
}

// Erc3009ABI is the input ABI used to generate the binding from.
// Deprecated: Use Erc3009MetaData.ABI instead.
var Erc3009ABI = Erc3009MetaData.ABI

// Erc3009Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Erc3009MetaData.Bin instead.
var Erc3009Bin = Erc3009MetaData.Bin

// DeployErc3009 deploys a new Ethereum contract, binding an instance of Erc3009 to it.
func DeployErc3009(auth *bind.TransactOpts, backend bind.ContractBackend, supply *big.Int) (common.Address, *types.Transaction, *Erc3009, error) {
	parsed, err := Erc3009MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Erc3009Bin), backend, supply)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Erc3009{Erc3009Caller: Erc3009Caller{contract: contract}, Erc3009Transactor: Erc3009Transactor{contract: contract}, Erc3009Filterer: Erc3009Filterer{contract: contract}}, nil
}

// Erc3009 is an auto generated Go binding around an Ethereum contract.
type Erc3009 struct {
	Erc3009Caller     // Read-only binding to the contract
	Erc3009Transactor // Write-only binding to the contract
	Erc3009Filterer   // Log filterer for contract events
}

// Erc3009Caller is an auto generated read-only Go binding around an Ethereum contract.
type Erc3009Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc3009Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Erc3009Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc3009Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Erc3009Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc3009Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Erc3009Session struct {
	Contract     *Erc3009          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Erc3009CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Erc3009CallerSession struct {
	Contract *Erc3009Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// Erc3009TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Erc3009TransactorSession struct {
	Contract     *Erc3009Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// Erc3009Raw is an auto generated low-level Go binding around an Ethereum contract.
type Erc3009Raw struct {
	Contract *Erc3009 // Generic contract binding to access the raw methods on
}

// Erc3009CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Erc3009CallerRaw struct {
	Contract *Erc3009Caller // Generic read-only contract binding to access the raw methods on
}

// Erc3009TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Erc3009TransactorRaw struct {
	Contract *Erc3009Transactor // Generic write-only contract binding to access the raw methods on
}

// NewErc3009 creates a new instance of Erc3009, bound to a specific deployed contract.
func NewErc3009(address common.Address, backend bind.ContractBackend) (*Erc3009, error) {
	contract, err := bindErc3009(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Erc3009{Erc3009Caller: Erc3009Caller{contract: contract}, Erc3009Transactor: Erc3009Transactor{contract: contract}, Erc3009Filterer: Erc3009Filterer{contract: contract}}, nil
}

// NewErc3009Caller creates a new read-only instance of Erc3009, bound to a specific deployed contract.
func NewErc3009Caller(address common.Address, caller bind.ContractCaller) (*Erc3009Caller, error) {
	contract, err := bindErc3009(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Erc3009Caller{contract: contract}, nil
}

// NewErc3009Transactor creates a new write-only instance of Erc3009, bound to a specific deployed contract.
func NewErc3009Transactor(address common.Address, transactor bind.ContractTransactor) (*Erc3009Transactor, error) {
	contract, err := bindErc3009(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Erc3009Transactor{contract: contract}, nil
}

// NewErc3009Filterer creates a new log filterer instance of Erc3009, bound to a specific deployed contract.
func NewErc3009Filterer(address common.Address, filterer bind.ContractFilterer) (*Erc3009Filterer, error) {
	contract, err := bindErc3009(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Erc3009Filterer{contract: contract}, nil
}

// bindErc3009 binds a generic wrapper to an already deployed contract.
func bindErc3009(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Erc3009MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc3009 *Erc3009Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc3009.Contract.Erc3009Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc3009 *Erc3009Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc3009.Contract.Erc3009Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc3009 *Erc3009Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc3009.Contract.Erc3009Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc3009 *Erc3009CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc3009.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc3009 *Erc3009TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc3009.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc3009 *Erc3009TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc3009.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Erc3009 *Erc3009Caller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Erc3009 *Erc3009Session) DOMAINSEPARATOR() ([32]byte, error) {
	return _Erc3009.Contract.DOMAINSEPARATOR(&_Erc3009.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Erc3009 *Erc3009CallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _Erc3009.Contract.DOMAINSEPARATOR(&_Erc3009.CallOpts)
}

// RECEIVEWITHAUTHORIZATIONTYPEHASH is a free data retrieval call binding the contract method 0x7f2eecc3.
//
// Solidity: function RECEIVE_WITH_AUTHORIZATION_TYPEHASH() view returns(bytes32)
func (_Erc3009 *Erc3009Caller) RECEIVEWITHAUTHORIZATIONTYPEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "RECEIVE_WITH_AUTHORIZATION_TYPEHASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RECEIVEWITHAUTHORIZATIONTYPEHASH is a free data retrieval call binding the contract method 0x7f2eecc3.
//
// Solidity: function RECEIVE_WITH_AUTHORIZATION_TYPEHASH() view returns(bytes32)
func (_Erc3009 *Erc3009Session) RECEIVEWITHAUTHORIZATIONTYPEHASH() ([32]byte, error) {
	return _Erc3009.Contract.RECEIVEWITHAUTHORIZATIONTYPEHASH(&_Erc3009.CallOpts)
}

// RECEIVEWITHAUTHORIZATIONTYPEHASH is a free data retrieval call binding the contract method 0x7f2eecc3.
//
// Solidity: function RECEIVE_WITH_AUTHORIZATION_TYPEHASH() view returns(bytes32)
func (_Erc3009 *Erc3009CallerSession) RECEIVEWITHAUTHORIZATIONTYPEHASH() ([32]byte, error) {
	return _Erc3009.Contract.RECEIVEWITHAUTHORIZATIONTYPEHASH(&_Erc3009.CallOpts)
}

// TRANSFERWITHAUTHORIZATIONTYPEHASH is a free data retrieval call binding the contract method 0xa0cc6a68.
//
// Solidity: function TRANSFER_WITH_AUTHORIZATION_TYPEHASH() view returns(bytes32)
func (_Erc3009 *Erc3009Caller) TRANSFERWITHAUTHORIZATIONTYPEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "TRANSFER_WITH_AUTHORIZATION_TYPEHASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// TRANSFERWITHAUTHORIZATIONTYPEHASH is a free data retrieval call binding the contract method 0xa0cc6a68.
//
// Solidity: function TRANSFER_WITH_AUTHORIZATION_TYPEHASH() view returns(bytes32)
func (_Erc3009 *Erc3009Session) TRANSFERWITHAUTHORIZATIONTYPEHASH() ([32]byte, error) {
	return _Erc3009.Contract.TRANSFERWITHAUTHORIZATIONTYPEHASH(&_Erc3009.CallOpts)
}

// TRANSFERWITHAUTHORIZATIONTYPEHASH is a free data retrieval call binding the contract method 0xa0cc6a68.
//
// Solidity: function TRANSFER_WITH_AUTHORIZATION_TYPEHASH() view returns(bytes32)
func (_Erc3009 *Erc3009CallerSession) TRANSFERWITHAUTHORIZATIONTYPEHASH() ([32]byte, error) {
	return _Erc3009.Contract.TRANSFERWITHAUTHORIZATIONTYPEHASH(&_Erc3009.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_Erc3009 *Erc3009Caller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "allowance", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_Erc3009 *Erc3009Session) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _Erc3009.Contract.Allowance(&_Erc3009.CallOpts, arg0, arg1)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) view returns(uint256)
func (_Erc3009 *Erc3009CallerSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _Erc3009.Contract.Allowance(&_Erc3009.CallOpts, arg0, arg1)
}

// AuthorizationState is a free data retrieval call binding the contract method 0xe94a0102.
//
// Solidity: function authorizationState(address , bytes32 ) view returns(bool)
func (_Erc3009 *Erc3009Caller) AuthorizationState(opts *bind.CallOpts, arg0 common.Address, arg1 [32]byte) (bool, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "authorizationState", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AuthorizationState is a free data retrieval call binding the contract method 0xe94a0102.
//
// Solidity: function authorizationState(address , bytes32 ) view returns(bool)
func (_Erc3009 *Erc3009Session) AuthorizationState(arg0 common.Address, arg1 [32]byte) (bool, error) {
	return _Erc3009.Contract.AuthorizationState(&_Erc3009.CallOpts, arg0, arg1)
}

// AuthorizationState is a free data retrieval call binding the contract method 0xe94a0102.
//
// Solidity: function authorizationState(address , bytes32 ) view returns(bool)
func (_Erc3009 *Erc3009CallerSession) AuthorizationState(arg0 common.Address, arg1 [32]byte) (bool, error) {
	return _Erc3009.Contract.AuthorizationState(&_Erc3009.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_Erc3009 *Erc3009Caller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "balanceOf", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_Erc3009 *Erc3009Session) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _Erc3009.Contract.BalanceOf(&_Erc3009.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) view returns(uint256)
func (_Erc3009 *Erc3009CallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _Erc3009.Contract.BalanceOf(&_Erc3009.CallOpts, arg0)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256 sum)
func (_Erc3009 *Erc3009Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Erc3009.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256 sum)
func (_Erc3009 *Erc3009Session) TotalSupply() (*big.Int, error) {
	return _Erc3009.Contract.TotalSupply(&_Erc3009.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256 sum)
func (_Erc3009 *Erc3009CallerSession) TotalSupply() (*big.Int, error) {
	return _Erc3009.Contract.TotalSupply(&_Erc3009.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_Erc3009 *Erc3009Transactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.contract.Transact(opts, "approve", spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_Erc3009 *Erc3009Session) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.Contract.Approve(&_Erc3009.TransactOpts, spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_Erc3009 *Erc3009TransactorSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.Contract.Approve(&_Erc3009.TransactOpts, spender, value)
}

// ReceiveWithAuthorization is a paid mutator transaction binding the contract method 0xef55bec6.
//
// Solidity: function receiveWithAuthorization(address from, address to, uint256 value, uint256 validAfter, uint256 validBefore, bytes32 nonce, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc3009 *Erc3009Transactor) ReceiveWithAuthorization(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int, validAfter *big.Int, validBefore *big.Int, nonce [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc3009.contract.Transact(opts, "receiveWithAuthorization", from, to, value, validAfter, validBefore, nonce, v, r, s)
}

// ReceiveWithAuthorization is a paid mutator transaction binding the contract method 0xef55bec6.
//
// Solidity: function receiveWithAuthorization(address from, address to, uint256 value, uint256 validAfter, uint256 validBefore, bytes32 nonce, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc3009 *Erc3009Session) ReceiveWithAuthorization(from common.Address, to common.Address, value *big.Int, validAfter *big.Int, validBefore *big.Int, nonce [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc3009.Contract.ReceiveWithAuthorization(&_Erc3009.TransactOpts, from, to, value, validAfter, validBefore, nonce, v, r, s)
}

// ReceiveWithAuthorization is a paid mutator transaction binding the contract method 0xef55bec6.
//
// Solidity: function receiveWithAuthorization(address from, address to, uint256 value, uint256 validAfter, uint256 validBefore, bytes32 nonce, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc3009 *Erc3009TransactorSession) ReceiveWithAuthorization(from common.Address, to common.Address, value *big.Int, validAfter *big.Int, validBefore *big.Int, nonce [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc3009.Contract.ReceiveWithAuthorization(&_Erc3009.TransactOpts, from, to, value, validAfter, validBefore, nonce, v, r, s)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_Erc3009 *Erc3009Transactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_Erc3009 *Erc3009Session) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.Contract.Transfer(&_Erc3009.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_Erc3009 *Erc3009TransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.Contract.Transfer(&_Erc3009.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_Erc3009 *Erc3009Transactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_Erc3009 *Erc3009Session) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.Contract.TransferFrom(&_Erc3009.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_Erc3009 *Erc3009TransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc3009.Contract.TransferFrom(&_Erc3009.TransactOpts, from, to, value)
}

// TransferWithAuthorization is a paid mutator transaction binding the contract method 0xe3ee160e.
//
// Solidity: function transferWithAuthorization(address from, address to, uint256 value, uint256 validAfter, uint256 validBefore, bytes32 nonce, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc3009 *Erc3009Transactor) TransferWithAuthorization(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int, validAfter *big.Int, validBefore *big.Int, nonce [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc3009.contract.Transact(opts, "transferWithAuthorization", from, to, value, validAfter, validBefore, nonce, v, r, s)
}

// TransferWithAuthorization is a paid mutator transaction binding the contract method 0xe3ee160e.
//
// Solidity: function transferWithAuthorization(address from, address to, uint256 value, uint256 validAfter, uint256 validBefore, bytes32 nonce, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc3009 *Erc3009Session) TransferWithAuthorization(from common.Address, to common.Address, value *big.Int, validAfter *big.Int, validBefore *big.Int, nonce [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc3009.Contract.TransferWithAuthorization(&_Erc3009.TransactOpts, from, to, value, validAfter, validBefore, nonce, v, r, s)
}

// TransferWithAuthorization is a paid mutator transaction binding the contract method 0xe3ee160e.
//
// Solidity: function transferWithAuthorization(address from, address to, uint256 value, uint256 validAfter, uint256 validBefore, bytes32 nonce, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc3009 *Erc3009TransactorSession) TransferWithAuthorization(from common.Address, to common.Address, value *big.Int, validAfter *big.Int, validBefore *big.Int, nonce [32]byte, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc3009.Contract.TransferWithAuthorization(&_Erc3009.TransactOpts, from, to, value, validAfter, validBefore, nonce, v, r, s)
}

// Erc3009ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Erc3009 contract.
type Erc3009ApprovalIterator struct {
	Event *Erc3009Approval // Event containing the contract specifics and raw log

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
func (it *Erc3009ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc3009Approval)
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
		it.Event = new(Erc3009Approval)
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
func (it *Erc3009ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc3009ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc3009Approval represents a Approval event raised by the Erc3009 contract.
type Erc3009Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Erc3009 *Erc3009Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*Erc3009ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Erc3009.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &Erc3009ApprovalIterator{contract: _Erc3009.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Erc3009 *Erc3009Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *Erc3009Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Erc3009.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc3009Approval)
				if err := _Erc3009.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Erc3009 *Erc3009Filterer) ParseApproval(log types.Log) (*Erc3009Approval, error) {
	event := new(Erc3009Approval)
	if err := _Erc3009.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc3009AuthorizationUsedIterator is returned from FilterAuthorizationUsed and is used to iterate over the raw logs and unpacked data for AuthorizationUsed events raised by the Erc3009 contract.
type Erc3009AuthorizationUsedIterator struct {
	Event *Erc3009AuthorizationUsed // Event containing the contract specifics and raw log

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
func (it *Erc3009AuthorizationUsedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc3009AuthorizationUsed)
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
		it.Event = new(Erc3009AuthorizationUsed)
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
func (it *Erc3009AuthorizationUsedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc3009AuthorizationUsedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc3009AuthorizationUsed represents a AuthorizationUsed event raised by the Erc3009 contract.
type Erc3009AuthorizationUsed struct {
	Authorizer common.Address
	Nonce      [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterAuthorizationUsed is a free log retrieval operation binding the contract event 0x98de503528ee59b575ef0c0a2576a82497bfc029a5685b209e9ec333479b10a5.
//
// Solidity: event AuthorizationUsed(address indexed authorizer, bytes32 indexed nonce)
func (_Erc3009 *Erc3009Filterer) FilterAuthorizationUsed(opts *bind.FilterOpts, authorizer []common.Address, nonce [][32]byte) (*Erc3009AuthorizationUsedIterator, error) {

	var authorizerRule []interface{}
	for _, authorizerItem := range authorizer {
		authorizerRule = append(authorizerRule, authorizerItem)
	}
	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}

	logs, sub, err := _Erc3009.contract.FilterLogs(opts, "AuthorizationUsed", authorizerRule, nonceRule)
	if err != nil {
		return nil, err
	}
	return &Erc3009AuthorizationUsedIterator{contract: _Erc3009.contract, event: "AuthorizationUsed", logs: logs, sub: sub}, nil
}

// WatchAuthorizationUsed is a free log subscription operation binding the contract event 0x98de503528ee59b575ef0c0a2576a82497bfc029a5685b209e9ec333479b10a5.
//
// Solidity: event AuthorizationUsed(address indexed authorizer, bytes32 indexed nonce)
func (_Erc3009 *Erc3009Filterer) WatchAuthorizationUsed(opts *bind.WatchOpts, sink chan<- *Erc3009AuthorizationUsed, authorizer []common.Address, nonce [][32]byte) (event.Subscription, error) {

	var authorizerRule []interface{}
	for _, authorizerItem := range authorizer {
		authorizerRule = append(authorizerRule, authorizerItem)
	}
	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}

	logs, sub, err := _Erc3009.contract.WatchLogs(opts, "AuthorizationUsed", authorizerRule, nonceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc3009AuthorizationUsed)
				if err := _Erc3009.contract.UnpackLog(event, "AuthorizationUsed", log); err != nil {
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

// ParseAuthorizationUsed is a log parse operation binding the contract event 0x98de503528ee59b575ef0c0a2576a82497bfc029a5685b209e9ec333479b10a5.
//
// Solidity: event AuthorizationUsed(address indexed authorizer, bytes32 indexed nonce)
func (_Erc3009 *Erc3009Filterer) ParseAuthorizationUsed(log types.Log) (*Erc3009AuthorizationUsed, error) {
	event := new(Erc3009AuthorizationUsed)
	if err := _Erc3009.contract.UnpackLog(event, "AuthorizationUsed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc3009TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Erc3009 contract.
type Erc3009TransferIterator struct {
	Event *Erc3009Transfer // Event containing the contract specifics and raw log

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
func (it *Erc3009TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc3009Transfer)
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
		it.Event = new(Erc3009Transfer)
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
func (it *Erc3009TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc3009TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc3009Transfer represents a Transfer event raised by the Erc3009 contract.
type Erc3009Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Erc3009 *Erc3009Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*Erc3009TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc3009.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &Erc3009TransferIterator{contract: _Erc3009.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Erc3009 *Erc3009Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *Erc3009Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc3009.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc3009Transfer)
				if err := _Erc3009.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Erc3009 *Erc3009Filterer) ParseTransfer(log types.Log) (*Erc3009Transfer, error) {
	event := new(Erc3009Transfer)
	if err := _Erc3009.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
