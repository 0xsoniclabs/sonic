// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package tokenreceiver

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

// TokenreceiverMetaData contains all meta data concerning the Tokenreceiver contract.
var TokenreceiverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"accept\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ERC1155BatchReceived\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ERC1155Received\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ERC721Received\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"ids\",\"type\":\"uint256[]\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"onERC1155BatchReceived\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"onERC1155Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"onERC721Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a0604052348015600e575f5ffd5b506040516108a43803806108a48339818101604052810190602e91906073565b801515608081151581525050506099565b5f5ffd5b5f8115159050919050565b6055816043565b8114605e575f5ffd5b50565b5f81519050606d81604e565b92915050565b5f602082840312156085576084603f565b5b5f6090848285016061565b91505092915050565b6080516107f36100b15f395f6101f301526107f35ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c8063150b7a0214610043578063bc197c8114610073578063f23a6e61146100a3575b5f5ffd5b61005d60048036038101906100589190610322565b6100d3565b60405161006a91906103e0565b60405180910390f35b61008d6004803603810190610088919061044e565b61012e565b60405161009a91906103e0565b60405180910390f35b6100bd60048036038101906100b89190610525565b610192565b6040516100ca91906103e0565b60405180910390f35b5f7fa05d90f300156ad1b545bc5d8197024456f21d22a708f5af04dd293e3d605251868686868660405161010b959493929190610633565b60405180910390a161012363150b7a0260e01b6101f0565b905095945050505050565b5f7f2360e6b28d144bcde27ca99baa660f822a59e919f243c01dfaec5a55986df5f5898989898989898960405161016c9897969594939291906106f7565b60405180910390a161018463bc197c8160e01b6101f0565b905098975050505050505050565b5f7f01109c91bca177ae1324bc2fc89be4b2573d2e59ad0cc75b1f3d3aa807814d368787878787876040516101cc96959493929190610763565b60405180910390a16101e463f23a6e6160e01b6101f0565b90509695505050505050565b5f7f00000000000000000000000000000000000000000000000000000000000000006102235763deadbeef60e01b610225565b815b9050919050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61025d82610234565b9050919050565b61026d81610253565b8114610277575f5ffd5b50565b5f8135905061028881610264565b92915050565b5f819050919050565b6102a08161028e565b81146102aa575f5ffd5b50565b5f813590506102bb81610297565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f8401126102e2576102e16102c1565b5b8235905067ffffffffffffffff8111156102ff576102fe6102c5565b5b60208301915083600182028301111561031b5761031a6102c9565b5b9250929050565b5f5f5f5f5f6080868803121561033b5761033a61022c565b5b5f6103488882890161027a565b95505060206103598882890161027a565b945050604061036a888289016102ad565b935050606086013567ffffffffffffffff81111561038b5761038a610230565b5b610397888289016102cd565b92509250509295509295909350565b5f7fffffffff0000000000000000000000000000000000000000000000000000000082169050919050565b6103da816103a6565b82525050565b5f6020820190506103f35f8301846103d1565b92915050565b5f5f83601f84011261040e5761040d6102c1565b5b8235905067ffffffffffffffff81111561042b5761042a6102c5565b5b602083019150836020820283011115610447576104466102c9565b5b9250929050565b5f5f5f5f5f5f5f5f60a0898b03121561046a5761046961022c565b5b5f6104778b828c0161027a565b98505060206104888b828c0161027a565b975050604089013567ffffffffffffffff8111156104a9576104a8610230565b5b6104b58b828c016103f9565b9650965050606089013567ffffffffffffffff8111156104d8576104d7610230565b5b6104e48b828c016103f9565b9450945050608089013567ffffffffffffffff81111561050757610506610230565b5b6105138b828c016102cd565b92509250509295985092959890939650565b5f5f5f5f5f5f60a0878903121561053f5761053e61022c565b5b5f61054c89828a0161027a565b965050602061055d89828a0161027a565b955050604061056e89828a016102ad565b945050606061057f89828a016102ad565b935050608087013567ffffffffffffffff8111156105a05761059f610230565b5b6105ac89828a016102cd565b92509250509295509295509295565b6105c481610253565b82525050565b6105d38161028e565b82525050565b5f82825260208201905092915050565b828183375f83830152505050565b5f601f19601f8301169050919050565b5f61061283856105d9565b935061061f8385846105e9565b610628836105f7565b840190509392505050565b5f6080820190506106465f8301886105bb565b61065360208301876105bb565b61066060408301866105ca565b8181036060830152610673818486610607565b90509695505050505050565b5f82825260208201905092915050565b5f5ffd5b82818337505050565b5f6106a7838561067f565b93507f07ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8311156106da576106d961068f565b5b6020830292506106eb838584610693565b82840190509392505050565b5f60a08201905061070a5f83018b6105bb565b610717602083018a6105bb565b818103604083015261072a81888a61069c565b9050818103606083015261073f81868861069c565b90508181036080830152610754818486610607565b90509998505050505050505050565b5f60a0820190506107765f8301896105bb565b61078360208301886105bb565b61079060408301876105ca565b61079d60608301866105ca565b81810360808301526107b0818486610607565b905097965050505050505056fea2646970667358221220a4d2fd998f51b5a3eff4fc53f36dd0a2ba217fb57f28db0061217f6f32e4205664736f6c63430008260033",
}

// TokenreceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenreceiverMetaData.ABI instead.
var TokenreceiverABI = TokenreceiverMetaData.ABI

// TokenreceiverBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TokenreceiverMetaData.Bin instead.
var TokenreceiverBin = TokenreceiverMetaData.Bin

// DeployTokenreceiver deploys a new Ethereum contract, binding an instance of Tokenreceiver to it.
func DeployTokenreceiver(auth *bind.TransactOpts, backend bind.ContractBackend, accept bool) (common.Address, *types.Transaction, *Tokenreceiver, error) {
	parsed, err := TokenreceiverMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TokenreceiverBin), backend, accept)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Tokenreceiver{TokenreceiverCaller: TokenreceiverCaller{contract: contract}, TokenreceiverTransactor: TokenreceiverTransactor{contract: contract}, TokenreceiverFilterer: TokenreceiverFilterer{contract: contract}}, nil
}

// Tokenreceiver is an auto generated Go binding around an Ethereum contract.
type Tokenreceiver struct {
	TokenreceiverCaller     // Read-only binding to the contract
	TokenreceiverTransactor // Write-only binding to the contract
	TokenreceiverFilterer   // Log filterer for contract events
}

// TokenreceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenreceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenreceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenreceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenreceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenreceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenreceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenreceiverSession struct {
	Contract     *Tokenreceiver    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TokenreceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenreceiverCallerSession struct {
	Contract *TokenreceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// TokenreceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenreceiverTransactorSession struct {
	Contract     *TokenreceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// TokenreceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenreceiverRaw struct {
	Contract *Tokenreceiver // Generic contract binding to access the raw methods on
}

// TokenreceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenreceiverCallerRaw struct {
	Contract *TokenreceiverCaller // Generic read-only contract binding to access the raw methods on
}

// TokenreceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenreceiverTransactorRaw struct {
	Contract *TokenreceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTokenreceiver creates a new instance of Tokenreceiver, bound to a specific deployed contract.
func NewTokenreceiver(address common.Address, backend bind.ContractBackend) (*Tokenreceiver, error) {
	contract, err := bindTokenreceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Tokenreceiver{TokenreceiverCaller: TokenreceiverCaller{contract: contract}, TokenreceiverTransactor: TokenreceiverTransactor{contract: contract}, TokenreceiverFilterer: TokenreceiverFilterer{contract: contract}}, nil
}

// NewTokenreceiverCaller creates a new read-only instance of Tokenreceiver, bound to a specific deployed contract.
func NewTokenreceiverCaller(address common.Address, caller bind.ContractCaller) (*TokenreceiverCaller, error) {
	contract, err := bindTokenreceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenreceiverCaller{contract: contract}, nil
}

// NewTokenreceiverTransactor creates a new write-only instance of Tokenreceiver, bound to a specific deployed contract.
func NewTokenreceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenreceiverTransactor, error) {
	contract, err := bindTokenreceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenreceiverTransactor{contract: contract}, nil
}

// NewTokenreceiverFilterer creates a new log filterer instance of Tokenreceiver, bound to a specific deployed contract.
func NewTokenreceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenreceiverFilterer, error) {
	contract, err := bindTokenreceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenreceiverFilterer{contract: contract}, nil
}

// bindTokenreceiver binds a generic wrapper to an already deployed contract.
func bindTokenreceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TokenreceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Tokenreceiver *TokenreceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Tokenreceiver.Contract.TokenreceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Tokenreceiver *TokenreceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.TokenreceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Tokenreceiver *TokenreceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.TokenreceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Tokenreceiver *TokenreceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Tokenreceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Tokenreceiver *TokenreceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Tokenreceiver *TokenreceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.contract.Transact(opts, method, params...)
}

// OnERC1155BatchReceived is a paid mutator transaction binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverTransactor) OnERC1155BatchReceived(opts *bind.TransactOpts, operator common.Address, from common.Address, ids []*big.Int, values []*big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.contract.Transact(opts, "onERC1155BatchReceived", operator, from, ids, values, data)
}

// OnERC1155BatchReceived is a paid mutator transaction binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverSession) OnERC1155BatchReceived(operator common.Address, from common.Address, ids []*big.Int, values []*big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.OnERC1155BatchReceived(&_Tokenreceiver.TransactOpts, operator, from, ids, values, data)
}

// OnERC1155BatchReceived is a paid mutator transaction binding the contract method 0xbc197c81.
//
// Solidity: function onERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverTransactorSession) OnERC1155BatchReceived(operator common.Address, from common.Address, ids []*big.Int, values []*big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.OnERC1155BatchReceived(&_Tokenreceiver.TransactOpts, operator, from, ids, values, data)
}

// OnERC1155Received is a paid mutator transaction binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverTransactor) OnERC1155Received(opts *bind.TransactOpts, operator common.Address, from common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.contract.Transact(opts, "onERC1155Received", operator, from, id, value, data)
}

// OnERC1155Received is a paid mutator transaction binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverSession) OnERC1155Received(operator common.Address, from common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.OnERC1155Received(&_Tokenreceiver.TransactOpts, operator, from, id, value, data)
}

// OnERC1155Received is a paid mutator transaction binding the contract method 0xf23a6e61.
//
// Solidity: function onERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverTransactorSession) OnERC1155Received(operator common.Address, from common.Address, id *big.Int, value *big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.OnERC1155Received(&_Tokenreceiver.TransactOpts, operator, from, id, value, data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address operator, address from, uint256 tokenId, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverTransactor) OnERC721Received(opts *bind.TransactOpts, operator common.Address, from common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.contract.Transact(opts, "onERC721Received", operator, from, tokenId, data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address operator, address from, uint256 tokenId, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverSession) OnERC721Received(operator common.Address, from common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.OnERC721Received(&_Tokenreceiver.TransactOpts, operator, from, tokenId, data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address operator, address from, uint256 tokenId, bytes data) returns(bytes4)
func (_Tokenreceiver *TokenreceiverTransactorSession) OnERC721Received(operator common.Address, from common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Tokenreceiver.Contract.OnERC721Received(&_Tokenreceiver.TransactOpts, operator, from, tokenId, data)
}

// TokenreceiverERC1155BatchReceivedIterator is returned from FilterERC1155BatchReceived and is used to iterate over the raw logs and unpacked data for ERC1155BatchReceived events raised by the Tokenreceiver contract.
type TokenreceiverERC1155BatchReceivedIterator struct {
	Event *TokenreceiverERC1155BatchReceived // Event containing the contract specifics and raw log

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
func (it *TokenreceiverERC1155BatchReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenreceiverERC1155BatchReceived)
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
		it.Event = new(TokenreceiverERC1155BatchReceived)
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
func (it *TokenreceiverERC1155BatchReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenreceiverERC1155BatchReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenreceiverERC1155BatchReceived represents a ERC1155BatchReceived event raised by the Tokenreceiver contract.
type TokenreceiverERC1155BatchReceived struct {
	Operator common.Address
	From     common.Address
	Ids      []*big.Int
	Values   []*big.Int
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterERC1155BatchReceived is a free log retrieval operation binding the contract event 0x2360e6b28d144bcde27ca99baa660f822a59e919f243c01dfaec5a55986df5f5.
//
// Solidity: event ERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) FilterERC1155BatchReceived(opts *bind.FilterOpts) (*TokenreceiverERC1155BatchReceivedIterator, error) {

	logs, sub, err := _Tokenreceiver.contract.FilterLogs(opts, "ERC1155BatchReceived")
	if err != nil {
		return nil, err
	}
	return &TokenreceiverERC1155BatchReceivedIterator{contract: _Tokenreceiver.contract, event: "ERC1155BatchReceived", logs: logs, sub: sub}, nil
}

// WatchERC1155BatchReceived is a free log subscription operation binding the contract event 0x2360e6b28d144bcde27ca99baa660f822a59e919f243c01dfaec5a55986df5f5.
//
// Solidity: event ERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) WatchERC1155BatchReceived(opts *bind.WatchOpts, sink chan<- *TokenreceiverERC1155BatchReceived) (event.Subscription, error) {

	logs, sub, err := _Tokenreceiver.contract.WatchLogs(opts, "ERC1155BatchReceived")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenreceiverERC1155BatchReceived)
				if err := _Tokenreceiver.contract.UnpackLog(event, "ERC1155BatchReceived", log); err != nil {
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

// ParseERC1155BatchReceived is a log parse operation binding the contract event 0x2360e6b28d144bcde27ca99baa660f822a59e919f243c01dfaec5a55986df5f5.
//
// Solidity: event ERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) ParseERC1155BatchReceived(log types.Log) (*TokenreceiverERC1155BatchReceived, error) {
	event := new(TokenreceiverERC1155BatchReceived)
	if err := _Tokenreceiver.contract.UnpackLog(event, "ERC1155BatchReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenreceiverERC1155ReceivedIterator is returned from FilterERC1155Received and is used to iterate over the raw logs and unpacked data for ERC1155Received events raised by the Tokenreceiver contract.
type TokenreceiverERC1155ReceivedIterator struct {
	Event *TokenreceiverERC1155Received // Event containing the contract specifics and raw log

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
func (it *TokenreceiverERC1155ReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenreceiverERC1155Received)
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
		it.Event = new(TokenreceiverERC1155Received)
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
func (it *TokenreceiverERC1155ReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenreceiverERC1155ReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenreceiverERC1155Received represents a ERC1155Received event raised by the Tokenreceiver contract.
type TokenreceiverERC1155Received struct {
	Operator common.Address
	From     common.Address
	Id       *big.Int
	Value    *big.Int
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterERC1155Received is a free log retrieval operation binding the contract event 0x01109c91bca177ae1324bc2fc89be4b2573d2e59ad0cc75b1f3d3aa807814d36.
//
// Solidity: event ERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) FilterERC1155Received(opts *bind.FilterOpts) (*TokenreceiverERC1155ReceivedIterator, error) {

	logs, sub, err := _Tokenreceiver.contract.FilterLogs(opts, "ERC1155Received")
	if err != nil {
		return nil, err
	}
	return &TokenreceiverERC1155ReceivedIterator{contract: _Tokenreceiver.contract, event: "ERC1155Received", logs: logs, sub: sub}, nil
}

// WatchERC1155Received is a free log subscription operation binding the contract event 0x01109c91bca177ae1324bc2fc89be4b2573d2e59ad0cc75b1f3d3aa807814d36.
//
// Solidity: event ERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) WatchERC1155Received(opts *bind.WatchOpts, sink chan<- *TokenreceiverERC1155Received) (event.Subscription, error) {

	logs, sub, err := _Tokenreceiver.contract.WatchLogs(opts, "ERC1155Received")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenreceiverERC1155Received)
				if err := _Tokenreceiver.contract.UnpackLog(event, "ERC1155Received", log); err != nil {
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

// ParseERC1155Received is a log parse operation binding the contract event 0x01109c91bca177ae1324bc2fc89be4b2573d2e59ad0cc75b1f3d3aa807814d36.
//
// Solidity: event ERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) ParseERC1155Received(log types.Log) (*TokenreceiverERC1155Received, error) {
	event := new(TokenreceiverERC1155Received)
	if err := _Tokenreceiver.contract.UnpackLog(event, "ERC1155Received", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TokenreceiverERC721ReceivedIterator is returned from FilterERC721Received and is used to iterate over the raw logs and unpacked data for ERC721Received events raised by the Tokenreceiver contract.
type TokenreceiverERC721ReceivedIterator struct {
	Event *TokenreceiverERC721Received // Event containing the contract specifics and raw log

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
func (it *TokenreceiverERC721ReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenreceiverERC721Received)
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
		it.Event = new(TokenreceiverERC721Received)
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
func (it *TokenreceiverERC721ReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenreceiverERC721ReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenreceiverERC721Received represents a ERC721Received event raised by the Tokenreceiver contract.
type TokenreceiverERC721Received struct {
	Operator common.Address
	From     common.Address
	TokenId  *big.Int
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterERC721Received is a free log retrieval operation binding the contract event 0xa05d90f300156ad1b545bc5d8197024456f21d22a708f5af04dd293e3d605251.
//
// Solidity: event ERC721Received(address operator, address from, uint256 tokenId, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) FilterERC721Received(opts *bind.FilterOpts) (*TokenreceiverERC721ReceivedIterator, error) {

	logs, sub, err := _Tokenreceiver.contract.FilterLogs(opts, "ERC721Received")
	if err != nil {
		return nil, err
	}
	return &TokenreceiverERC721ReceivedIterator{contract: _Tokenreceiver.contract, event: "ERC721Received", logs: logs, sub: sub}, nil
}

// WatchERC721Received is a free log subscription operation binding the contract event 0xa05d90f300156ad1b545bc5d8197024456f21d22a708f5af04dd293e3d605251.
//
// Solidity: event ERC721Received(address operator, address from, uint256 tokenId, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) WatchERC721Received(opts *bind.WatchOpts, sink chan<- *TokenreceiverERC721Received) (event.Subscription, error) {

	logs, sub, err := _Tokenreceiver.contract.WatchLogs(opts, "ERC721Received")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenreceiverERC721Received)
				if err := _Tokenreceiver.contract.UnpackLog(event, "ERC721Received", log); err != nil {
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

// ParseERC721Received is a log parse operation binding the contract event 0xa05d90f300156ad1b545bc5d8197024456f21d22a708f5af04dd293e3d605251.
//
// Solidity: event ERC721Received(address operator, address from, uint256 tokenId, bytes data)
func (_Tokenreceiver *TokenreceiverFilterer) ParseERC721Received(log types.Log) (*TokenreceiverERC721Received, error) {
	event := new(TokenreceiverERC721Received)
	if err := _Tokenreceiver.contract.UnpackLog(event, "ERC721Received", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
