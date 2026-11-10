// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package erc1155

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

// Erc1155MetaData contains all meta data concerning the Erc1155 contract.
var Erc1155MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"name\":\"TransferBatch\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"TransferSingle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"value\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"URI\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"owners\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"}],\"name\":\"balanceOfBatch\",\"outputs\":[{\"internalType\":\"uint256[]\",\"name\":\"balances\",\"type\":\"uint256[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeBatchTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f5ffd5b5060405161229b38038061229b8339818101604052810190610031919061048e565b8051825114610075576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161006c9061055e565b60405180910390fd5b5f5f90505b8251811015610125578181815181106100965761009561057c565b5b60200260200101515f5f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8584815181106100ee576100ed61057c565b5b602002602001015181526020019081526020015f205f82825461011191906105d6565b92505081905550808060010191505061007a565b503373ffffffffffffffffffffffffffffffffffffffff165f73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb858560405161019c9291906106c0565b60405180910390a46101c55f33848460405180602001604052805f8152506101cc60201b60201c565b50506108e2565b5f8473ffffffffffffffffffffffffffffffffffffffff163b11156102f35763bc197c8160e01b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168473ffffffffffffffffffffffffffffffffffffffff1663bc197c8133888787876040518663ffffffff1660e01b8152600401610253959493929190610794565b6020604051808303815f875af115801561026f573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610293919061084f565b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916146102f2576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102e9906108c4565b60405180910390fd5b5b5050505050565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6103558261030f565b810181811067ffffffffffffffff821117156103745761037361031f565b5b80604052505050565b5f6103866102fa565b9050610392828261034c565b919050565b5f67ffffffffffffffff8211156103b1576103b061031f565b5b602082029050602081019050919050565b5f5ffd5b5f819050919050565b6103d8816103c6565b81146103e2575f5ffd5b50565b5f815190506103f3816103cf565b92915050565b5f61040b61040684610397565b61037d565b9050808382526020820190506020840283018581111561042e5761042d6103c2565b5b835b81811015610457578061044388826103e5565b845260208401935050602081019050610430565b5050509392505050565b5f82601f8301126104755761047461030b565b5b81516104858482602086016103f9565b91505092915050565b5f5f604083850312156104a4576104a3610303565b5b5f83015167ffffffffffffffff8111156104c1576104c0610307565b5b6104cd85828601610461565b925050602083015167ffffffffffffffff8111156104ee576104ed610307565b5b6104fa85828601610461565b9150509250929050565b5f82825260208201905092915050565b7f6c656e677468206d69736d6174636800000000000000000000000000000000005f82015250565b5f610548600f83610504565b915061055382610514565b602082019050919050565b5f6020820190508181035f8301526105758161053c565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6105e0826103c6565b91506105eb836103c6565b9250828201905080821115610603576106026105a9565b5b92915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b61063b816103c6565b82525050565b5f61064c8383610632565b60208301905092915050565b5f602082019050919050565b5f61066e82610609565b6106788185610613565b935061068383610623565b805f5b838110156106b357815161069a8882610641565b97506106a583610658565b925050600181019050610686565b5085935050505092915050565b5f6040820190508181035f8301526106d88185610664565b905081810360208301526106ec8184610664565b90509392505050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61071e826106f5565b9050919050565b61072e81610714565b82525050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f61076682610734565b610770818561073e565b935061078081856020860161074e565b6107898161030f565b840191505092915050565b5f60a0820190506107a75f830188610725565b6107b46020830187610725565b81810360408301526107c68186610664565b905081810360608301526107da8185610664565b905081810360808301526107ee818461075c565b90509695505050505050565b5f7fffffffff0000000000000000000000000000000000000000000000000000000082169050919050565b61082e816107fa565b8114610838575f5ffd5b50565b5f8151905061084981610825565b92915050565b5f6020828403121561086457610863610303565b5b5f6108718482850161083b565b91505092915050565b7f756e7361666520726563697069656e74000000000000000000000000000000005f82015250565b5f6108ae601083610504565b91506108b98261087a565b602082019050919050565b5f6020820190508181035f8301526108db816108a2565b9050919050565b6119ac806108ef5f395ff3fe608060405234801561000f575f5ffd5b506004361061007a575f3560e01c80634e1273f4116100595780634e1273f4146100fa578063a22cb4651461012a578063e985e9c514610146578063f242432a146101765761007a565b8062fdd58e1461007e57806301ffc9a7146100ae5780632eb2c2d6146100de575b5f5ffd5b61009860048036038101906100939190610ede565b610192565b6040516100a59190610f2b565b60405180910390f35b6100c860048036038101906100c39190610f99565b6101b1565b6040516100d59190610fde565b60405180910390f35b6100f860048036038101906100f391906110ad565b610212565b005b610114600480360381019061010f91906111d9565b610662565b604051610121919061130e565b60405180910390f35b610144600480360381019061013f9190611358565b6107cb565b005b610160600480360381019061015b9190611396565b6108c3565b60405161016d9190610fde565b60405180910390f35b610190600480360381019061018b91906113d4565b6108ed565b005b5f602052815f5260405f20602052805f5260405f205f91509150505481565b5f6301ffc9a760e01b827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916148061020b575063d9b67a2660e01b827bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b9050919050565b5f73ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff1603610280576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610277906114c4565b60405180910390fd5b8383905086869050146102c8576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102bf9061152c565b60405180910390fd5b8773ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161480610383575060015f8973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff165b6103c2576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103b990611594565b60405180910390fd5b5f5f90505b86869050811015610505578484828181106103e5576103e46115b2565b5b905060200201355f5f8b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f89898581811061043c5761043b6115b2565b5b9050602002013581526020019081526020015f205f82825461045e919061160c565b92505081905550848482818110610478576104776115b2565b5b905060200201355f5f8a73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8989858181106104cf576104ce6115b2565b5b9050602002013581526020019081526020015f205f8282546104f1919061163f565b9250508190555080806001019150506103c7565b508673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb8989898960405161058094939291906116da565b60405180910390a461065888888888808060200260200160405190810160405280939291908181526020018383602002808284375f81840152601f19601f820116905080830192505050505050508787808060200260200160405190810160405280939291908181526020018383602002808284375f81840152601f19601f8201169050808301925050505050505086868080601f0160208091040260200160405190810160405280939291908181526020018383808284375f81840152601f19601f82011690508083019250505050505050610bed565b5050505050505050565b60608282905085859050146106ac576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016106a39061152c565b60405180910390fd5b8484905067ffffffffffffffff8111156106c9576106c8611713565b5b6040519080825280602002602001820160405280156106f75781602001602082028036833780820191505090505b5090505f5f90505b858590508110156107c2575f5f87878481811061071f5761071e6115b2565b5b90506020020160208101906107349190611740565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f858584818110610781576107806115b2565b5b9050602002013581526020019081526020015f20548282815181106107a9576107a86115b2565b5b60200260200101818152505080806001019150506106ff565b50949350505050565b8060015f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff0219169083151502179055508173ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31836040516108b79190610fde565b60405180910390a35050565b6001602052815f5260405f20602052805f5260405f205f915091509054906101000a900460ff1681565b5f73ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff160361095b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610952906114c4565b60405180910390fd5b8573ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161480610a16575060015f8773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f3373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900460ff165b610a55576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a4c90611594565b60405180910390fd5b825f5f8873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8681526020019081526020015f205f828254610aaf919061160c565b92505081905550825f5f8773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8681526020019081526020015f205f828254610b10919061163f565b925050819055508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f628787604051610b8d92919061176b565b60405180910390a4610be58686868686868080601f0160208091040260200160405190810160405280939291908181526020018383808284375f81840152601f19601f82011690508083019250505050505050610d1b565b505050505050565b5f8473ffffffffffffffffffffffffffffffffffffffff163b1115610d145763bc197c8160e01b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168473ffffffffffffffffffffffffffffffffffffffff1663bc197c8133888787876040518663ffffffff1660e01b8152600401610c74959493929190611811565b6020604051808303815f875af1158015610c90573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610cb4919061188b565b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff191614610d13576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610d0a90611900565b60405180910390fd5b5b5050505050565b5f8473ffffffffffffffffffffffffffffffffffffffff163b1115610e425763f23a6e6160e01b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168473ffffffffffffffffffffffffffffffffffffffff1663f23a6e6133888787876040518663ffffffff1660e01b8152600401610da295949392919061191e565b6020604051808303815f875af1158015610dbe573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610de2919061188b565b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff191614610e41576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610e3890611900565b60405180910390fd5b5b5050505050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610e7a82610e51565b9050919050565b610e8a81610e70565b8114610e94575f5ffd5b50565b5f81359050610ea581610e81565b92915050565b5f819050919050565b610ebd81610eab565b8114610ec7575f5ffd5b50565b5f81359050610ed881610eb4565b92915050565b5f5f60408385031215610ef457610ef3610e49565b5b5f610f0185828601610e97565b9250506020610f1285828601610eca565b9150509250929050565b610f2581610eab565b82525050565b5f602082019050610f3e5f830184610f1c565b92915050565b5f7fffffffff0000000000000000000000000000000000000000000000000000000082169050919050565b610f7881610f44565b8114610f82575f5ffd5b50565b5f81359050610f9381610f6f565b92915050565b5f60208284031215610fae57610fad610e49565b5b5f610fbb84828501610f85565b91505092915050565b5f8115159050919050565b610fd881610fc4565b82525050565b5f602082019050610ff15f830184610fcf565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261101857611017610ff7565b5b8235905067ffffffffffffffff81111561103557611034610ffb565b5b60208301915083602082028301111561105157611050610fff565b5b9250929050565b5f5f83601f84011261106d5761106c610ff7565b5b8235905067ffffffffffffffff81111561108a57611089610ffb565b5b6020830191508360018202830111156110a6576110a5610fff565b5b9250929050565b5f5f5f5f5f5f5f5f60a0898b0312156110c9576110c8610e49565b5b5f6110d68b828c01610e97565b98505060206110e78b828c01610e97565b975050604089013567ffffffffffffffff81111561110857611107610e4d565b5b6111148b828c01611003565b9650965050606089013567ffffffffffffffff81111561113757611136610e4d565b5b6111438b828c01611003565b9450945050608089013567ffffffffffffffff81111561116657611165610e4d565b5b6111728b828c01611058565b92509250509295985092959890939650565b5f5f83601f84011261119957611198610ff7565b5b8235905067ffffffffffffffff8111156111b6576111b5610ffb565b5b6020830191508360208202830111156111d2576111d1610fff565b5b9250929050565b5f5f5f5f604085870312156111f1576111f0610e49565b5b5f85013567ffffffffffffffff81111561120e5761120d610e4d565b5b61121a87828801611184565b9450945050602085013567ffffffffffffffff81111561123d5761123c610e4d565b5b61124987828801611003565b925092505092959194509250565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b61128981610eab565b82525050565b5f61129a8383611280565b60208301905092915050565b5f602082019050919050565b5f6112bc82611257565b6112c68185611261565b93506112d183611271565b805f5b838110156113015781516112e8888261128f565b97506112f3836112a6565b9250506001810190506112d4565b5085935050505092915050565b5f6020820190508181035f83015261132681846112b2565b905092915050565b61133781610fc4565b8114611341575f5ffd5b50565b5f813590506113528161132e565b92915050565b5f5f6040838503121561136e5761136d610e49565b5b5f61137b85828601610e97565b925050602061138c85828601611344565b9150509250929050565b5f5f604083850312156113ac576113ab610e49565b5b5f6113b985828601610e97565b92505060206113ca85828601610e97565b9150509250929050565b5f5f5f5f5f5f60a087890312156113ee576113ed610e49565b5b5f6113fb89828a01610e97565b965050602061140c89828a01610e97565b955050604061141d89828a01610eca565b945050606061142e89828a01610eca565b935050608087013567ffffffffffffffff81111561144f5761144e610e4d565b5b61145b89828a01611058565b92509250509295509295509295565b5f82825260208201905092915050565b7f7a65726f206164647265737300000000000000000000000000000000000000005f82015250565b5f6114ae600c8361146a565b91506114b98261147a565b602082019050919050565b5f6020820190508181035f8301526114db816114a2565b9050919050565b7f6c656e677468206d69736d6174636800000000000000000000000000000000005f82015250565b5f611516600f8361146a565b9150611521826114e2565b602082019050919050565b5f6020820190508181035f8301526115438161150a565b9050919050565b7f6e6f7420617574686f72697a65640000000000000000000000000000000000005f82015250565b5f61157e600e8361146a565b91506115898261154a565b602082019050919050565b5f6020820190508181035f8301526115ab81611572565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61161682610eab565b915061162183610eab565b9250828203905081811115611639576116386115df565b5b92915050565b5f61164982610eab565b915061165483610eab565b925082820190508082111561166c5761166b6115df565b5b92915050565b5f5ffd5b82818337505050565b5f61168a8385611261565b93507f07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8311156116bd576116bc611672565b5b6020830292506116ce838584611676565b82840190509392505050565b5f6040820190508181035f8301526116f381868861167f565b9050818103602083015261170881848661167f565b905095945050505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b5f6020828403121561175557611754610e49565b5b5f61176284828501610e97565b91505092915050565b5f60408201905061177e5f830185610f1c565b61178b6020830184610f1c565b9392505050565b61179b81610e70565b82525050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6117e3826117a1565b6117ed81856117ab565b93506117fd8185602086016117bb565b611806816117c9565b840191505092915050565b5f60a0820190506118245f830188611792565b6118316020830187611792565b818103604083015261184381866112b2565b9050818103606083015261185781856112b2565b9050818103608083015261186b81846117d9565b90509695505050505050565b5f8151905061188581610f6f565b92915050565b5f602082840312156118a05761189f610e49565b5b5f6118ad84828501611877565b91505092915050565b7f756e7361666520726563697069656e74000000000000000000000000000000005f82015250565b5f6118ea60108361146a565b91506118f5826118b6565b602082019050919050565b5f6020820190508181035f830152611917816118de565b9050919050565b5f60a0820190506119315f830188611792565b61193e6020830187611792565b61194b6040830186610f1c565b6119586060830185610f1c565b818103608083015261196a81846117d9565b9050969550505050505056fea2646970667358221220d02542dcf258121b4027b1ed122ec7dc9f2a98de7b39f37909fa19674605e73064736f6c63430008260033",
}

// Erc1155ABI is the input ABI used to generate the binding from.
// Deprecated: Use Erc1155MetaData.ABI instead.
var Erc1155ABI = Erc1155MetaData.ABI

// Erc1155Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Erc1155MetaData.Bin instead.
var Erc1155Bin = Erc1155MetaData.Bin

// DeployErc1155 deploys a new Ethereum contract, binding an instance of Erc1155 to it.
func DeployErc1155(auth *bind.TransactOpts, backend bind.ContractBackend, ids []*big.Int, values []*big.Int) (common.Address, *types.Transaction, *Erc1155, error) {
	parsed, err := Erc1155MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Erc1155Bin), backend, ids, values)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Erc1155{Erc1155Caller: Erc1155Caller{contract: contract}, Erc1155Transactor: Erc1155Transactor{contract: contract}, Erc1155Filterer: Erc1155Filterer{contract: contract}}, nil
}

// Erc1155 is an auto generated Go binding around an Ethereum contract.
type Erc1155 struct {
	Erc1155Caller     // Read-only binding to the contract
	Erc1155Transactor // Write-only binding to the contract
	Erc1155Filterer   // Log filterer for contract events
}

// Erc1155Caller is an auto generated read-only Go binding around an Ethereum contract.
type Erc1155Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc1155Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Erc1155Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc1155Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Erc1155Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc1155Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Erc1155Session struct {
	Contract     *Erc1155          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Erc1155CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Erc1155CallerSession struct {
	Contract *Erc1155Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// Erc1155TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Erc1155TransactorSession struct {
	Contract     *Erc1155Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// Erc1155Raw is an auto generated low-level Go binding around an Ethereum contract.
type Erc1155Raw struct {
	Contract *Erc1155 // Generic contract binding to access the raw methods on
}

// Erc1155CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Erc1155CallerRaw struct {
	Contract *Erc1155Caller // Generic read-only contract binding to access the raw methods on
}

// Erc1155TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Erc1155TransactorRaw struct {
	Contract *Erc1155Transactor // Generic write-only contract binding to access the raw methods on
}

// NewErc1155 creates a new instance of Erc1155, bound to a specific deployed contract.
func NewErc1155(address common.Address, backend bind.ContractBackend) (*Erc1155, error) {
	contract, err := bindErc1155(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Erc1155{Erc1155Caller: Erc1155Caller{contract: contract}, Erc1155Transactor: Erc1155Transactor{contract: contract}, Erc1155Filterer: Erc1155Filterer{contract: contract}}, nil
}

// NewErc1155Caller creates a new read-only instance of Erc1155, bound to a specific deployed contract.
func NewErc1155Caller(address common.Address, caller bind.ContractCaller) (*Erc1155Caller, error) {
	contract, err := bindErc1155(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Erc1155Caller{contract: contract}, nil
}

// NewErc1155Transactor creates a new write-only instance of Erc1155, bound to a specific deployed contract.
func NewErc1155Transactor(address common.Address, transactor bind.ContractTransactor) (*Erc1155Transactor, error) {
	contract, err := bindErc1155(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Erc1155Transactor{contract: contract}, nil
}

// NewErc1155Filterer creates a new log filterer instance of Erc1155, bound to a specific deployed contract.
func NewErc1155Filterer(address common.Address, filterer bind.ContractFilterer) (*Erc1155Filterer, error) {
	contract, err := bindErc1155(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Erc1155Filterer{contract: contract}, nil
}

// bindErc1155 binds a generic wrapper to an already deployed contract.
func bindErc1155(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Erc1155MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc1155 *Erc1155Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc1155.Contract.Erc1155Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc1155 *Erc1155Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc1155.Contract.Erc1155Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc1155 *Erc1155Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc1155.Contract.Erc1155Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc1155 *Erc1155CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc1155.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc1155 *Erc1155TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc1155.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc1155 *Erc1155TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc1155.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address , uint256 ) view returns(uint256)
func (_Erc1155 *Erc1155Caller) BalanceOf(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Erc1155.contract.Call(opts, &out, "balanceOf", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address , uint256 ) view returns(uint256)
func (_Erc1155 *Erc1155Session) BalanceOf(arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	return _Erc1155.Contract.BalanceOf(&_Erc1155.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x00fdd58e.
//
// Solidity: function balanceOf(address , uint256 ) view returns(uint256)
func (_Erc1155 *Erc1155CallerSession) BalanceOf(arg0 common.Address, arg1 *big.Int) (*big.Int, error) {
	return _Erc1155.Contract.BalanceOf(&_Erc1155.CallOpts, arg0, arg1)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] owners, uint256[] ids) view returns(uint256[] balances)
func (_Erc1155 *Erc1155Caller) BalanceOfBatch(opts *bind.CallOpts, owners []common.Address, ids []*big.Int) ([]*big.Int, error) {
	var out []interface{}
	err := _Erc1155.contract.Call(opts, &out, "balanceOfBatch", owners, ids)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] owners, uint256[] ids) view returns(uint256[] balances)
func (_Erc1155 *Erc1155Session) BalanceOfBatch(owners []common.Address, ids []*big.Int) ([]*big.Int, error) {
	return _Erc1155.Contract.BalanceOfBatch(&_Erc1155.CallOpts, owners, ids)
}

// BalanceOfBatch is a free data retrieval call binding the contract method 0x4e1273f4.
//
// Solidity: function balanceOfBatch(address[] owners, uint256[] ids) view returns(uint256[] balances)
func (_Erc1155 *Erc1155CallerSession) BalanceOfBatch(owners []common.Address, ids []*big.Int) ([]*big.Int, error) {
	return _Erc1155.Contract.BalanceOfBatch(&_Erc1155.CallOpts, owners, ids)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address , address ) view returns(bool)
func (_Erc1155 *Erc1155Caller) IsApprovedForAll(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (bool, error) {
	var out []interface{}
	err := _Erc1155.contract.Call(opts, &out, "isApprovedForAll", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address , address ) view returns(bool)
func (_Erc1155 *Erc1155Session) IsApprovedForAll(arg0 common.Address, arg1 common.Address) (bool, error) {
	return _Erc1155.Contract.IsApprovedForAll(&_Erc1155.CallOpts, arg0, arg1)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address , address ) view returns(bool)
func (_Erc1155 *Erc1155CallerSession) IsApprovedForAll(arg0 common.Address, arg1 common.Address) (bool, error) {
	return _Erc1155.Contract.IsApprovedForAll(&_Erc1155.CallOpts, arg0, arg1)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_Erc1155 *Erc1155Caller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Erc1155.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_Erc1155 *Erc1155Session) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Erc1155.Contract.SupportsInterface(&_Erc1155.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_Erc1155 *Erc1155CallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Erc1155.Contract.SupportsInterface(&_Erc1155.CallOpts, interfaceId)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0x2eb2c2d6.
//
// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] values, bytes data) returns()
func (_Erc1155 *Erc1155Transactor) SafeBatchTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, ids []*big.Int, values []*big.Int, data []byte) (*types.Transaction, error) {
	return _Erc1155.contract.Transact(opts, "safeBatchTransferFrom", from, to, ids, values, data)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0x2eb2c2d6.
//
// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] values, bytes data) returns()
func (_Erc1155 *Erc1155Session) SafeBatchTransferFrom(from common.Address, to common.Address, ids []*big.Int, values []*big.Int, data []byte) (*types.Transaction, error) {
	return _Erc1155.Contract.SafeBatchTransferFrom(&_Erc1155.TransactOpts, from, to, ids, values, data)
}

// SafeBatchTransferFrom is a paid mutator transaction binding the contract method 0x2eb2c2d6.
//
// Solidity: function safeBatchTransferFrom(address from, address to, uint256[] ids, uint256[] values, bytes data) returns()
func (_Erc1155 *Erc1155TransactorSession) SafeBatchTransferFrom(from common.Address, to common.Address, ids []*big.Int, values []*big.Int, data []byte) (*types.Transaction, error) {
	return _Erc1155.Contract.SafeBatchTransferFrom(&_Erc1155.TransactOpts, from, to, ids, values, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes data) returns()
func (_Erc1155 *Erc1155Transactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Erc1155.contract.Transact(opts, "safeTransferFrom", from, to, id, value, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes data) returns()
func (_Erc1155 *Erc1155Session) SafeTransferFrom(from common.Address, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Erc1155.Contract.SafeTransferFrom(&_Erc1155.TransactOpts, from, to, id, value, data)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0xf242432a.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes data) returns()
func (_Erc1155 *Erc1155TransactorSession) SafeTransferFrom(from common.Address, to common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Erc1155.Contract.SafeTransferFrom(&_Erc1155.TransactOpts, from, to, id, value, data)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Erc1155 *Erc1155Transactor) SetApprovalForAll(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _Erc1155.contract.Transact(opts, "setApprovalForAll", operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Erc1155 *Erc1155Session) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Erc1155.Contract.SetApprovalForAll(&_Erc1155.TransactOpts, operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_Erc1155 *Erc1155TransactorSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _Erc1155.Contract.SetApprovalForAll(&_Erc1155.TransactOpts, operator, approved)
}

// Erc1155ApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the Erc1155 contract.
type Erc1155ApprovalForAllIterator struct {
	Event *Erc1155ApprovalForAll // Event containing the contract specifics and raw log

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
func (it *Erc1155ApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc1155ApprovalForAll)
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
		it.Event = new(Erc1155ApprovalForAll)
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
func (it *Erc1155ApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc1155ApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc1155ApprovalForAll represents a ApprovalForAll event raised by the Erc1155 contract.
type Erc1155ApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Erc1155 *Erc1155Filterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*Erc1155ApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Erc1155.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &Erc1155ApprovalForAllIterator{contract: _Erc1155.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Erc1155 *Erc1155Filterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *Erc1155ApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Erc1155.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc1155ApprovalForAll)
				if err := _Erc1155.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_Erc1155 *Erc1155Filterer) ParseApprovalForAll(log types.Log) (*Erc1155ApprovalForAll, error) {
	event := new(Erc1155ApprovalForAll)
	if err := _Erc1155.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc1155TransferBatchIterator is returned from FilterTransferBatch and is used to iterate over the raw logs and unpacked data for TransferBatch events raised by the Erc1155 contract.
type Erc1155TransferBatchIterator struct {
	Event *Erc1155TransferBatch // Event containing the contract specifics and raw log

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
func (it *Erc1155TransferBatchIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc1155TransferBatch)
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
		it.Event = new(Erc1155TransferBatch)
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
func (it *Erc1155TransferBatchIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc1155TransferBatchIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc1155TransferBatch represents a TransferBatch event raised by the Erc1155 contract.
type Erc1155TransferBatch struct {
	Operator common.Address
	From     common.Address
	To       common.Address
	Ids      []*big.Int
	Values   []*big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTransferBatch is a free log retrieval operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
func (_Erc1155 *Erc1155Filterer) FilterTransferBatch(opts *bind.FilterOpts, operator []common.Address, from []common.Address, to []common.Address) (*Erc1155TransferBatchIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc1155.contract.FilterLogs(opts, "TransferBatch", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &Erc1155TransferBatchIterator{contract: _Erc1155.contract, event: "TransferBatch", logs: logs, sub: sub}, nil
}

// WatchTransferBatch is a free log subscription operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
func (_Erc1155 *Erc1155Filterer) WatchTransferBatch(opts *bind.WatchOpts, sink chan<- *Erc1155TransferBatch, operator []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc1155.contract.WatchLogs(opts, "TransferBatch", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc1155TransferBatch)
				if err := _Erc1155.contract.UnpackLog(event, "TransferBatch", log); err != nil {
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

// ParseTransferBatch is a log parse operation binding the contract event 0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb.
//
// Solidity: event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
func (_Erc1155 *Erc1155Filterer) ParseTransferBatch(log types.Log) (*Erc1155TransferBatch, error) {
	event := new(Erc1155TransferBatch)
	if err := _Erc1155.contract.UnpackLog(event, "TransferBatch", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc1155TransferSingleIterator is returned from FilterTransferSingle and is used to iterate over the raw logs and unpacked data for TransferSingle events raised by the Erc1155 contract.
type Erc1155TransferSingleIterator struct {
	Event *Erc1155TransferSingle // Event containing the contract specifics and raw log

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
func (it *Erc1155TransferSingleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc1155TransferSingle)
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
		it.Event = new(Erc1155TransferSingle)
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
func (it *Erc1155TransferSingleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc1155TransferSingleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc1155TransferSingle represents a TransferSingle event raised by the Erc1155 contract.
type Erc1155TransferSingle struct {
	Operator common.Address
	From     common.Address
	To       common.Address
	Id       *big.Int
	Value    *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTransferSingle is a free log retrieval operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
func (_Erc1155 *Erc1155Filterer) FilterTransferSingle(opts *bind.FilterOpts, operator []common.Address, from []common.Address, to []common.Address) (*Erc1155TransferSingleIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc1155.contract.FilterLogs(opts, "TransferSingle", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &Erc1155TransferSingleIterator{contract: _Erc1155.contract, event: "TransferSingle", logs: logs, sub: sub}, nil
}

// WatchTransferSingle is a free log subscription operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
func (_Erc1155 *Erc1155Filterer) WatchTransferSingle(opts *bind.WatchOpts, sink chan<- *Erc1155TransferSingle, operator []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc1155.contract.WatchLogs(opts, "TransferSingle", operatorRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc1155TransferSingle)
				if err := _Erc1155.contract.UnpackLog(event, "TransferSingle", log); err != nil {
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

// ParseTransferSingle is a log parse operation binding the contract event 0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62.
//
// Solidity: event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
func (_Erc1155 *Erc1155Filterer) ParseTransferSingle(log types.Log) (*Erc1155TransferSingle, error) {
	event := new(Erc1155TransferSingle)
	if err := _Erc1155.contract.UnpackLog(event, "TransferSingle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc1155URIIterator is returned from FilterURI and is used to iterate over the raw logs and unpacked data for URI events raised by the Erc1155 contract.
type Erc1155URIIterator struct {
	Event *Erc1155URI // Event containing the contract specifics and raw log

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
func (it *Erc1155URIIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc1155URI)
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
		it.Event = new(Erc1155URI)
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
func (it *Erc1155URIIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc1155URIIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc1155URI represents a URI event raised by the Erc1155 contract.
type Erc1155URI struct {
	Value string
	Id    *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterURI is a free log retrieval operation binding the contract event 0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b.
//
// Solidity: event URI(string value, uint256 indexed id)
func (_Erc1155 *Erc1155Filterer) FilterURI(opts *bind.FilterOpts, id []*big.Int) (*Erc1155URIIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Erc1155.contract.FilterLogs(opts, "URI", idRule)
	if err != nil {
		return nil, err
	}
	return &Erc1155URIIterator{contract: _Erc1155.contract, event: "URI", logs: logs, sub: sub}, nil
}

// WatchURI is a free log subscription operation binding the contract event 0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b.
//
// Solidity: event URI(string value, uint256 indexed id)
func (_Erc1155 *Erc1155Filterer) WatchURI(opts *bind.WatchOpts, sink chan<- *Erc1155URI, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Erc1155.contract.WatchLogs(opts, "URI", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc1155URI)
				if err := _Erc1155.contract.UnpackLog(event, "URI", log); err != nil {
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

// ParseURI is a log parse operation binding the contract event 0x6bb7ff708619ba0610cba295a58592e0451dee2622938c8755667688daf3529b.
//
// Solidity: event URI(string value, uint256 indexed id)
func (_Erc1155 *Erc1155Filterer) ParseURI(log types.Log) (*Erc1155URI, error) {
	event := new(Erc1155URI)
	if err := _Erc1155.contract.UnpackLog(event, "URI", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
