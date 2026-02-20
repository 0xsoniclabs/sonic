// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package indexed_logs

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

// IndexedLogsMetaData contains all meta data concerning the IndexedLogs contract.
var IndexedLogsMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"Event1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"Event2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"text\",\"type\":\"string\"}],\"name\":\"Event3\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Log1\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"a\",\"type\":\"uint256\"}],\"name\":\"Log2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"a\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"b\",\"type\":\"uint256\"}],\"name\":\"Log3\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"a\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"b\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"c\",\"type\":\"uint256\"}],\"name\":\"Log4\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"n\",\"type\":\"uint256\"}],\"name\":\"emitCartesianProduct\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"emitEvents\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"n\",\"type\":\"uint256\"}],\"name\":\"emitLogs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103e68061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c806345f301e8146100435780634ce4ddea1461005f5780636c8893d31461007b575b5f5ffd5b61005d600480360381019061005891906102d7565b610085565b005b610079600480360381019061007491906102d7565b610192565b005b6100836101dd565b005b5f5fa07f1732d0c17008d342618e7f03069177d8d39391d79811bb4e706d7c6c84108c0f60405160405180910390a15f5f90505b8181101561018e57807f624fb00c2ce79f34cb543884c3af64816dce0f4cec3d32661959e49d488a7a9360405160405180910390a25f5f90505b828110156101805780827febe57242c74e694c7ec0f2fe9302812f324576f94a505b0de3f0ecb473d149bb60405160405180910390a35f5f90505b83811015610172578082847f8540fe9d62711b26f5d55a228125ce553737daafbb466fb5c89ffef0b5907d1460405160405180910390a4808060010191505061012e565b5080806001019150506100f3565b5080806001019150506100b9565b5050565b5f5f90505b818110156101d957807f624fb00c2ce79f34cb543884c3af64816dce0f4cec3d32661959e49d488a7a9360405160405180910390a28080600101915050610197565b5050565b5f5f90505b600581101561029d577f04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce38160405161021a9190610311565b60405180910390a17f06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e816040516102519190610311565b60405180910390a17f93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628816040516102889190610384565b60405180910390a180806001019150506101e2565b50565b5f5ffd5b5f819050919050565b6102b6816102a4565b81146102c0575f5ffd5b50565b5f813590506102d1816102ad565b92915050565b5f602082840312156102ec576102eb6102a0565b5b5f6102f9848285016102c3565b91505092915050565b61030b816102a4565b82525050565b5f6020820190506103245f830184610302565b92915050565b5f82825260208201905092915050565b7f7465737420737472696e670000000000000000000000000000000000000000005f82015250565b5f61036e600b8361032a565b91506103798261033a565b602082019050919050565b5f6040820190506103975f830184610302565b81810360208301526103a881610362565b90509291505056fea26469706673582212205c6314ee3a8e7417a08b7d10a650f013759f35274731ed145f4e9e3a3e460f4064736f6c634300081b0033",
}

// IndexedLogsABI is the input ABI used to generate the binding from.
// Deprecated: Use IndexedLogsMetaData.ABI instead.
var IndexedLogsABI = IndexedLogsMetaData.ABI

// IndexedLogsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use IndexedLogsMetaData.Bin instead.
var IndexedLogsBin = IndexedLogsMetaData.Bin

// DeployIndexedLogs deploys a new Ethereum contract, binding an instance of IndexedLogs to it.
func DeployIndexedLogs(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *IndexedLogs, error) {
	parsed, err := IndexedLogsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(IndexedLogsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &IndexedLogs{IndexedLogsCaller: IndexedLogsCaller{contract: contract}, IndexedLogsTransactor: IndexedLogsTransactor{contract: contract}, IndexedLogsFilterer: IndexedLogsFilterer{contract: contract}}, nil
}

// IndexedLogs is an auto generated Go binding around an Ethereum contract.
type IndexedLogs struct {
	IndexedLogsCaller     // Read-only binding to the contract
	IndexedLogsTransactor // Write-only binding to the contract
	IndexedLogsFilterer   // Log filterer for contract events
}

// IndexedLogsCaller is an auto generated read-only Go binding around an Ethereum contract.
type IndexedLogsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IndexedLogsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IndexedLogsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IndexedLogsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IndexedLogsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IndexedLogsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IndexedLogsSession struct {
	Contract     *IndexedLogs      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IndexedLogsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IndexedLogsCallerSession struct {
	Contract *IndexedLogsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// IndexedLogsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IndexedLogsTransactorSession struct {
	Contract     *IndexedLogsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// IndexedLogsRaw is an auto generated low-level Go binding around an Ethereum contract.
type IndexedLogsRaw struct {
	Contract *IndexedLogs // Generic contract binding to access the raw methods on
}

// IndexedLogsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IndexedLogsCallerRaw struct {
	Contract *IndexedLogsCaller // Generic read-only contract binding to access the raw methods on
}

// IndexedLogsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IndexedLogsTransactorRaw struct {
	Contract *IndexedLogsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIndexedLogs creates a new instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogs(address common.Address, backend bind.ContractBackend) (*IndexedLogs, error) {
	contract, err := bindIndexedLogs(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &IndexedLogs{IndexedLogsCaller: IndexedLogsCaller{contract: contract}, IndexedLogsTransactor: IndexedLogsTransactor{contract: contract}, IndexedLogsFilterer: IndexedLogsFilterer{contract: contract}}, nil
}

// NewIndexedLogsCaller creates a new read-only instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogsCaller(address common.Address, caller bind.ContractCaller) (*IndexedLogsCaller, error) {
	contract, err := bindIndexedLogs(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsCaller{contract: contract}, nil
}

// NewIndexedLogsTransactor creates a new write-only instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogsTransactor(address common.Address, transactor bind.ContractTransactor) (*IndexedLogsTransactor, error) {
	contract, err := bindIndexedLogs(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsTransactor{contract: contract}, nil
}

// NewIndexedLogsFilterer creates a new log filterer instance of IndexedLogs, bound to a specific deployed contract.
func NewIndexedLogsFilterer(address common.Address, filterer bind.ContractFilterer) (*IndexedLogsFilterer, error) {
	contract, err := bindIndexedLogs(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsFilterer{contract: contract}, nil
}

// bindIndexedLogs binds a generic wrapper to an already deployed contract.
func bindIndexedLogs(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := IndexedLogsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IndexedLogs *IndexedLogsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IndexedLogs.Contract.IndexedLogsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IndexedLogs *IndexedLogsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IndexedLogs.Contract.IndexedLogsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IndexedLogs *IndexedLogsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IndexedLogs.Contract.IndexedLogsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_IndexedLogs *IndexedLogsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _IndexedLogs.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_IndexedLogs *IndexedLogsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IndexedLogs.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_IndexedLogs *IndexedLogsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _IndexedLogs.Contract.contract.Transact(opts, method, params...)
}

// EmitCartesianProduct is a paid mutator transaction binding the contract method 0x45f301e8.
//
// Solidity: function emitCartesianProduct(uint256 n) returns()
func (_IndexedLogs *IndexedLogsTransactor) EmitCartesianProduct(opts *bind.TransactOpts, n *big.Int) (*types.Transaction, error) {
	return _IndexedLogs.contract.Transact(opts, "emitCartesianProduct", n)
}

// EmitCartesianProduct is a paid mutator transaction binding the contract method 0x45f301e8.
//
// Solidity: function emitCartesianProduct(uint256 n) returns()
func (_IndexedLogs *IndexedLogsSession) EmitCartesianProduct(n *big.Int) (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitCartesianProduct(&_IndexedLogs.TransactOpts, n)
}

// EmitCartesianProduct is a paid mutator transaction binding the contract method 0x45f301e8.
//
// Solidity: function emitCartesianProduct(uint256 n) returns()
func (_IndexedLogs *IndexedLogsTransactorSession) EmitCartesianProduct(n *big.Int) (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitCartesianProduct(&_IndexedLogs.TransactOpts, n)
}

// EmitEvents is a paid mutator transaction binding the contract method 0x6c8893d3.
//
// Solidity: function emitEvents() returns()
func (_IndexedLogs *IndexedLogsTransactor) EmitEvents(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _IndexedLogs.contract.Transact(opts, "emitEvents")
}

// EmitEvents is a paid mutator transaction binding the contract method 0x6c8893d3.
//
// Solidity: function emitEvents() returns()
func (_IndexedLogs *IndexedLogsSession) EmitEvents() (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitEvents(&_IndexedLogs.TransactOpts)
}

// EmitEvents is a paid mutator transaction binding the contract method 0x6c8893d3.
//
// Solidity: function emitEvents() returns()
func (_IndexedLogs *IndexedLogsTransactorSession) EmitEvents() (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitEvents(&_IndexedLogs.TransactOpts)
}

// EmitLogs is a paid mutator transaction binding the contract method 0x4ce4ddea.
//
// Solidity: function emitLogs(uint256 n) returns()
func (_IndexedLogs *IndexedLogsTransactor) EmitLogs(opts *bind.TransactOpts, n *big.Int) (*types.Transaction, error) {
	return _IndexedLogs.contract.Transact(opts, "emitLogs", n)
}

// EmitLogs is a paid mutator transaction binding the contract method 0x4ce4ddea.
//
// Solidity: function emitLogs(uint256 n) returns()
func (_IndexedLogs *IndexedLogsSession) EmitLogs(n *big.Int) (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitLogs(&_IndexedLogs.TransactOpts, n)
}

// EmitLogs is a paid mutator transaction binding the contract method 0x4ce4ddea.
//
// Solidity: function emitLogs(uint256 n) returns()
func (_IndexedLogs *IndexedLogsTransactorSession) EmitLogs(n *big.Int) (*types.Transaction, error) {
	return _IndexedLogs.Contract.EmitLogs(&_IndexedLogs.TransactOpts, n)
}

// IndexedLogsEvent1Iterator is returned from FilterEvent1 and is used to iterate over the raw logs and unpacked data for Event1 events raised by the IndexedLogs contract.
type IndexedLogsEvent1Iterator struct {
	Event *IndexedLogsEvent1 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsEvent1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsEvent1)
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
		it.Event = new(IndexedLogsEvent1)
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
func (it *IndexedLogsEvent1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsEvent1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsEvent1 represents a Event1 event raised by the IndexedLogs contract.
type IndexedLogsEvent1 struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEvent1 is a free log retrieval operation binding the contract event 0x04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3.
//
// Solidity: event Event1(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) FilterEvent1(opts *bind.FilterOpts) (*IndexedLogsEvent1Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Event1")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsEvent1Iterator{contract: _IndexedLogs.contract, event: "Event1", logs: logs, sub: sub}, nil
}

// WatchEvent1 is a free log subscription operation binding the contract event 0x04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3.
//
// Solidity: event Event1(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) WatchEvent1(opts *bind.WatchOpts, sink chan<- *IndexedLogsEvent1) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Event1")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsEvent1)
				if err := _IndexedLogs.contract.UnpackLog(event, "Event1", log); err != nil {
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

// ParseEvent1 is a log parse operation binding the contract event 0x04474795f5b996ff80cb47c148d4c5ccdbe09ef27551820caa9c2f8ed149cce3.
//
// Solidity: event Event1(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) ParseEvent1(log types.Log) (*IndexedLogsEvent1, error) {
	event := new(IndexedLogsEvent1)
	if err := _IndexedLogs.contract.UnpackLog(event, "Event1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsEvent2Iterator is returned from FilterEvent2 and is used to iterate over the raw logs and unpacked data for Event2 events raised by the IndexedLogs contract.
type IndexedLogsEvent2Iterator struct {
	Event *IndexedLogsEvent2 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsEvent2Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsEvent2)
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
		it.Event = new(IndexedLogsEvent2)
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
func (it *IndexedLogsEvent2Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsEvent2Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsEvent2 represents a Event2 event raised by the IndexedLogs contract.
type IndexedLogsEvent2 struct {
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEvent2 is a free log retrieval operation binding the contract event 0x06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e.
//
// Solidity: event Event2(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) FilterEvent2(opts *bind.FilterOpts) (*IndexedLogsEvent2Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Event2")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsEvent2Iterator{contract: _IndexedLogs.contract, event: "Event2", logs: logs, sub: sub}, nil
}

// WatchEvent2 is a free log subscription operation binding the contract event 0x06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e.
//
// Solidity: event Event2(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) WatchEvent2(opts *bind.WatchOpts, sink chan<- *IndexedLogsEvent2) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Event2")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsEvent2)
				if err := _IndexedLogs.contract.UnpackLog(event, "Event2", log); err != nil {
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

// ParseEvent2 is a log parse operation binding the contract event 0x06df6fb2d6d0b17a870decb858cc46bf7b69142ab7b9318f7603ed3fd4ad240e.
//
// Solidity: event Event2(uint256 id)
func (_IndexedLogs *IndexedLogsFilterer) ParseEvent2(log types.Log) (*IndexedLogsEvent2, error) {
	event := new(IndexedLogsEvent2)
	if err := _IndexedLogs.contract.UnpackLog(event, "Event2", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsEvent3Iterator is returned from FilterEvent3 and is used to iterate over the raw logs and unpacked data for Event3 events raised by the IndexedLogs contract.
type IndexedLogsEvent3Iterator struct {
	Event *IndexedLogsEvent3 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsEvent3Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsEvent3)
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
		it.Event = new(IndexedLogsEvent3)
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
func (it *IndexedLogsEvent3Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsEvent3Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsEvent3 represents a Event3 event raised by the IndexedLogs contract.
type IndexedLogsEvent3 struct {
	Id   *big.Int
	Text string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterEvent3 is a free log retrieval operation binding the contract event 0x93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628.
//
// Solidity: event Event3(uint256 id, string text)
func (_IndexedLogs *IndexedLogsFilterer) FilterEvent3(opts *bind.FilterOpts) (*IndexedLogsEvent3Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Event3")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsEvent3Iterator{contract: _IndexedLogs.contract, event: "Event3", logs: logs, sub: sub}, nil
}

// WatchEvent3 is a free log subscription operation binding the contract event 0x93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628.
//
// Solidity: event Event3(uint256 id, string text)
func (_IndexedLogs *IndexedLogsFilterer) WatchEvent3(opts *bind.WatchOpts, sink chan<- *IndexedLogsEvent3) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Event3")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsEvent3)
				if err := _IndexedLogs.contract.UnpackLog(event, "Event3", log); err != nil {
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

// ParseEvent3 is a log parse operation binding the contract event 0x93af88a66c9681ed3b0530b95b3723732fc309c0c3f7dde9cb86168f64495628.
//
// Solidity: event Event3(uint256 id, string text)
func (_IndexedLogs *IndexedLogsFilterer) ParseEvent3(log types.Log) (*IndexedLogsEvent3, error) {
	event := new(IndexedLogsEvent3)
	if err := _IndexedLogs.contract.UnpackLog(event, "Event3", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsLog1Iterator is returned from FilterLog1 and is used to iterate over the raw logs and unpacked data for Log1 events raised by the IndexedLogs contract.
type IndexedLogsLog1Iterator struct {
	Event *IndexedLogsLog1 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsLog1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsLog1)
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
		it.Event = new(IndexedLogsLog1)
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
func (it *IndexedLogsLog1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsLog1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsLog1 represents a Log1 event raised by the IndexedLogs contract.
type IndexedLogsLog1 struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterLog1 is a free log retrieval operation binding the contract event 0x1732d0c17008d342618e7f03069177d8d39391d79811bb4e706d7c6c84108c0f.
//
// Solidity: event Log1()
func (_IndexedLogs *IndexedLogsFilterer) FilterLog1(opts *bind.FilterOpts) (*IndexedLogsLog1Iterator, error) {

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Log1")
	if err != nil {
		return nil, err
	}
	return &IndexedLogsLog1Iterator{contract: _IndexedLogs.contract, event: "Log1", logs: logs, sub: sub}, nil
}

// WatchLog1 is a free log subscription operation binding the contract event 0x1732d0c17008d342618e7f03069177d8d39391d79811bb4e706d7c6c84108c0f.
//
// Solidity: event Log1()
func (_IndexedLogs *IndexedLogsFilterer) WatchLog1(opts *bind.WatchOpts, sink chan<- *IndexedLogsLog1) (event.Subscription, error) {

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Log1")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsLog1)
				if err := _IndexedLogs.contract.UnpackLog(event, "Log1", log); err != nil {
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

// ParseLog1 is a log parse operation binding the contract event 0x1732d0c17008d342618e7f03069177d8d39391d79811bb4e706d7c6c84108c0f.
//
// Solidity: event Log1()
func (_IndexedLogs *IndexedLogsFilterer) ParseLog1(log types.Log) (*IndexedLogsLog1, error) {
	event := new(IndexedLogsLog1)
	if err := _IndexedLogs.contract.UnpackLog(event, "Log1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsLog2Iterator is returned from FilterLog2 and is used to iterate over the raw logs and unpacked data for Log2 events raised by the IndexedLogs contract.
type IndexedLogsLog2Iterator struct {
	Event *IndexedLogsLog2 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsLog2Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsLog2)
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
		it.Event = new(IndexedLogsLog2)
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
func (it *IndexedLogsLog2Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsLog2Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsLog2 represents a Log2 event raised by the IndexedLogs contract.
type IndexedLogsLog2 struct {
	A   *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterLog2 is a free log retrieval operation binding the contract event 0x624fb00c2ce79f34cb543884c3af64816dce0f4cec3d32661959e49d488a7a93.
//
// Solidity: event Log2(uint256 indexed a)
func (_IndexedLogs *IndexedLogsFilterer) FilterLog2(opts *bind.FilterOpts, a []*big.Int) (*IndexedLogsLog2Iterator, error) {

	var aRule []interface{}
	for _, aItem := range a {
		aRule = append(aRule, aItem)
	}

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Log2", aRule)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsLog2Iterator{contract: _IndexedLogs.contract, event: "Log2", logs: logs, sub: sub}, nil
}

// WatchLog2 is a free log subscription operation binding the contract event 0x624fb00c2ce79f34cb543884c3af64816dce0f4cec3d32661959e49d488a7a93.
//
// Solidity: event Log2(uint256 indexed a)
func (_IndexedLogs *IndexedLogsFilterer) WatchLog2(opts *bind.WatchOpts, sink chan<- *IndexedLogsLog2, a []*big.Int) (event.Subscription, error) {

	var aRule []interface{}
	for _, aItem := range a {
		aRule = append(aRule, aItem)
	}

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Log2", aRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsLog2)
				if err := _IndexedLogs.contract.UnpackLog(event, "Log2", log); err != nil {
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

// ParseLog2 is a log parse operation binding the contract event 0x624fb00c2ce79f34cb543884c3af64816dce0f4cec3d32661959e49d488a7a93.
//
// Solidity: event Log2(uint256 indexed a)
func (_IndexedLogs *IndexedLogsFilterer) ParseLog2(log types.Log) (*IndexedLogsLog2, error) {
	event := new(IndexedLogsLog2)
	if err := _IndexedLogs.contract.UnpackLog(event, "Log2", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsLog3Iterator is returned from FilterLog3 and is used to iterate over the raw logs and unpacked data for Log3 events raised by the IndexedLogs contract.
type IndexedLogsLog3Iterator struct {
	Event *IndexedLogsLog3 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsLog3Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsLog3)
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
		it.Event = new(IndexedLogsLog3)
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
func (it *IndexedLogsLog3Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsLog3Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsLog3 represents a Log3 event raised by the IndexedLogs contract.
type IndexedLogsLog3 struct {
	A   *big.Int
	B   *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterLog3 is a free log retrieval operation binding the contract event 0xebe57242c74e694c7ec0f2fe9302812f324576f94a505b0de3f0ecb473d149bb.
//
// Solidity: event Log3(uint256 indexed a, uint256 indexed b)
func (_IndexedLogs *IndexedLogsFilterer) FilterLog3(opts *bind.FilterOpts, a []*big.Int, b []*big.Int) (*IndexedLogsLog3Iterator, error) {

	var aRule []interface{}
	for _, aItem := range a {
		aRule = append(aRule, aItem)
	}
	var bRule []interface{}
	for _, bItem := range b {
		bRule = append(bRule, bItem)
	}

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Log3", aRule, bRule)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsLog3Iterator{contract: _IndexedLogs.contract, event: "Log3", logs: logs, sub: sub}, nil
}

// WatchLog3 is a free log subscription operation binding the contract event 0xebe57242c74e694c7ec0f2fe9302812f324576f94a505b0de3f0ecb473d149bb.
//
// Solidity: event Log3(uint256 indexed a, uint256 indexed b)
func (_IndexedLogs *IndexedLogsFilterer) WatchLog3(opts *bind.WatchOpts, sink chan<- *IndexedLogsLog3, a []*big.Int, b []*big.Int) (event.Subscription, error) {

	var aRule []interface{}
	for _, aItem := range a {
		aRule = append(aRule, aItem)
	}
	var bRule []interface{}
	for _, bItem := range b {
		bRule = append(bRule, bItem)
	}

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Log3", aRule, bRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsLog3)
				if err := _IndexedLogs.contract.UnpackLog(event, "Log3", log); err != nil {
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

// ParseLog3 is a log parse operation binding the contract event 0xebe57242c74e694c7ec0f2fe9302812f324576f94a505b0de3f0ecb473d149bb.
//
// Solidity: event Log3(uint256 indexed a, uint256 indexed b)
func (_IndexedLogs *IndexedLogsFilterer) ParseLog3(log types.Log) (*IndexedLogsLog3, error) {
	event := new(IndexedLogsLog3)
	if err := _IndexedLogs.contract.UnpackLog(event, "Log3", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// IndexedLogsLog4Iterator is returned from FilterLog4 and is used to iterate over the raw logs and unpacked data for Log4 events raised by the IndexedLogs contract.
type IndexedLogsLog4Iterator struct {
	Event *IndexedLogsLog4 // Event containing the contract specifics and raw log

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
func (it *IndexedLogsLog4Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IndexedLogsLog4)
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
		it.Event = new(IndexedLogsLog4)
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
func (it *IndexedLogsLog4Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IndexedLogsLog4Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IndexedLogsLog4 represents a Log4 event raised by the IndexedLogs contract.
type IndexedLogsLog4 struct {
	A   *big.Int
	B   *big.Int
	C   *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterLog4 is a free log retrieval operation binding the contract event 0x8540fe9d62711b26f5d55a228125ce553737daafbb466fb5c89ffef0b5907d14.
//
// Solidity: event Log4(uint256 indexed a, uint256 indexed b, uint256 indexed c)
func (_IndexedLogs *IndexedLogsFilterer) FilterLog4(opts *bind.FilterOpts, a []*big.Int, b []*big.Int, c []*big.Int) (*IndexedLogsLog4Iterator, error) {

	var aRule []interface{}
	for _, aItem := range a {
		aRule = append(aRule, aItem)
	}
	var bRule []interface{}
	for _, bItem := range b {
		bRule = append(bRule, bItem)
	}
	var cRule []interface{}
	for _, cItem := range c {
		cRule = append(cRule, cItem)
	}

	logs, sub, err := _IndexedLogs.contract.FilterLogs(opts, "Log4", aRule, bRule, cRule)
	if err != nil {
		return nil, err
	}
	return &IndexedLogsLog4Iterator{contract: _IndexedLogs.contract, event: "Log4", logs: logs, sub: sub}, nil
}

// WatchLog4 is a free log subscription operation binding the contract event 0x8540fe9d62711b26f5d55a228125ce553737daafbb466fb5c89ffef0b5907d14.
//
// Solidity: event Log4(uint256 indexed a, uint256 indexed b, uint256 indexed c)
func (_IndexedLogs *IndexedLogsFilterer) WatchLog4(opts *bind.WatchOpts, sink chan<- *IndexedLogsLog4, a []*big.Int, b []*big.Int, c []*big.Int) (event.Subscription, error) {

	var aRule []interface{}
	for _, aItem := range a {
		aRule = append(aRule, aItem)
	}
	var bRule []interface{}
	for _, bItem := range b {
		bRule = append(bRule, bItem)
	}
	var cRule []interface{}
	for _, cItem := range c {
		cRule = append(cRule, cItem)
	}

	logs, sub, err := _IndexedLogs.contract.WatchLogs(opts, "Log4", aRule, bRule, cRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IndexedLogsLog4)
				if err := _IndexedLogs.contract.UnpackLog(event, "Log4", log); err != nil {
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

// ParseLog4 is a log parse operation binding the contract event 0x8540fe9d62711b26f5d55a228125ce553737daafbb466fb5c89ffef0b5907d14.
//
// Solidity: event Log4(uint256 indexed a, uint256 indexed b, uint256 indexed c)
func (_IndexedLogs *IndexedLogsFilterer) ParseLog4(log types.Log) (*IndexedLogsLog4, error) {
	event := new(IndexedLogsLog4)
	if err := _IndexedLogs.contract.UnpackLog(event, "Log4", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
