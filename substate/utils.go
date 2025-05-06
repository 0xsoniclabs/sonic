package substate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/0xsoniclabs/substate/substate"
	stypes "github.com/0xsoniclabs/substate/types"
	"github.com/0xsoniclabs/substate/types/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var skippedTxStatesFile = ""
var unprocessedSkippedTxs *skippedTxData

type skippedTxData struct {
	blockNumber uint64 // block number
	data        map[int]map[string]interface{}
}

// Utils to convert Geth types to Substate types

// HashGethToSubstate converts map of geth's common.Hash to Substate hashes map
func HashGethToSubstate(g map[uint64]common.Hash) map[uint64]stypes.Hash {
	res := make(map[uint64]stypes.Hash)
	for k, v := range g {
		res[k] = stypes.Hash(v)
	}
	return res
}

// HashListToSubstate converts list of geth's common.Hash to Substate hashes list
func HashListToSubstate(g []common.Hash) []stypes.Hash {
	res := make([]stypes.Hash, len(g))
	for _, v := range g {
		res = append(res, stypes.Hash(v))
	}
	return res
}

// AccessListGethToSubstate converts geth's types.AccessList to Substate types.AccessList
func AccessListGethToSubstate(al types.AccessList) stypes.AccessList {
	st := stypes.AccessList{}
	for _, tuple := range al {
		var keys []stypes.Hash
		for _, key := range tuple.StorageKeys {
			keys = append(keys, stypes.Hash(key))
		}
		st = append(st, stypes.AccessTuple{Address: stypes.Address(tuple.Address), StorageKeys: keys})
	}
	return st
}

// LogsGethToSubstate converts slice of geth's *types.Log to Substate *types.Log
func LogsGethToSubstate(logs []*types.Log) []*stypes.Log {
	var ls []*stypes.Log
	for _, log := range logs {
		var data = log.Data
		// Log.Data is required, so it cannot be nil
		if log.Data == nil {
			data = []byte{}
		}

		l := new(stypes.Log)
		l.BlockHash = stypes.Hash(log.BlockHash)
		l.Data = data
		l.Address = stypes.Address(log.Address)
		l.Index = log.Index
		l.BlockNumber = log.BlockNumber
		l.Removed = log.Removed
		l.TxHash = stypes.Hash(log.TxHash)
		l.TxIndex = log.TxIndex
		for _, topic := range log.Topics {
			l.Topics = append(l.Topics, stypes.Hash(topic))
		}

		ls = append(ls, l)
	}
	return ls
}

// NewEnv prepares *substate.Env from ether's Block
// func NewEnv(etherBlock *types.Block, statedb state2.StateDB, evmHeader *evmcore.EvmBlock) *substate.Env {
func NewEnv(etherBlock *types.Block, blockHashes map[uint64]stypes.Hash, context vm.BlockContext) *substate.Env {
	return substate.NewEnv(
		stypes.Address(etherBlock.Coinbase()),
		etherBlock.Difficulty(),
		etherBlock.GasLimit(),
		etherBlock.NumberU64(),
		etherBlock.Time(),
		etherBlock.BaseFee(),
		big.NewInt(1),
		blockHashes,
		(*stypes.Hash)(context.Random))
}

// NewMessage prepares *substate.Message from ether's Message
func NewMessage(msg *core.Message, txType uint8) *substate.Message {
	var to *stypes.Address
	// for contract creation, To is nil
	if msg.To != nil {
		a := stypes.Address(msg.To.Bytes())
		to = &a
	}

	dataHash := hash.Keccak256Hash(msg.Data)

	txTypeProtobuf := int32(txType)
	return substate.NewMessage(
		msg.Nonce,
		msg.SkipAccountChecks,
		msg.GasPrice,
		msg.GasLimit,
		stypes.Address(msg.From),
		to,
		msg.Value,
		msg.Data,
		&dataHash,
		&txTypeProtobuf,
		AccessListGethToSubstate(msg.AccessList),
		msg.GasFeeCap,
		msg.GasTipCap,
		msg.BlobGasFeeCap,
		HashListToSubstate(msg.BlobHashes))
}

// NewResult prepares *substate.Result from ether's Receipt
func NewResult(receipt *types.Receipt) *substate.Result {
	b := stypes.Bloom{}
	b.SetBytes(receipt.Bloom.Bytes())
	res := substate.NewResult(
		receipt.Status,
		b,
		LogsGethToSubstate(receipt.Logs),
		stypes.Address(receipt.ContractAddress),
		receipt.GasUsed)
	return res
}

//if substateRecordReplay.RecordReplay {
//	// save tx substate into DBs, merge block hashes to env
//	etherBlock := block.RecordingEthBlock()
//
//	//// TODO determine message attributes
//	////type Message struct {
//	////	To            *common.Address
//	////	From          common.Address
//	////	Nonce         uint64
//	////	Value         *big.Int
//	////	GasLimit      uint64
//	////	GasPrice      *big.Int
//	////	GasFeeCap     *big.Int
//	////	GasTipCap     *big.Int
//	////	Data          []byte
//	////	AccessList    types.AccessList
//	////	BlobGasFeeCap *big.Int
//	////	BlobHashes    []common.Hash
//	////
//	////	// When SkipAccountChecks is true, the message nonce is not checked against the
//	////	// account nonce in state. It also disables checking that the sender is an EOA.
//	////	// This field will be set to true for operations like RPC eth_call.
//	////	SkipAccountChecks bool
//	//
//	//var to = types2.Address{}
//	//to.SetBytes(msg.To.Bytes())
//	//var al = make([]types2.AccessTuple, 0, len(msg.AccessList))
//	//for _, l := range msg.AccessList {
//	//	al = append(al, types2.AccessTuple{
//	//		Address: types2.Address(l.Address),
//	//	})
//	//}
//	//var bh = make([]types2.Hash, 0, len(msg.BlobHashes))
//	//for _, hash := range msg.BlobHashes {
//	//	bh = append(bh, types2.Hash(hash))
//	//}
//	//mg := substate.NewMessage(msg.Nonce, false, msg.GasPrice, msg.GasLimit, types2.Address(msg.From), &to, msg.Value, msg.Data, nil, al, msg.GasFeeCap, msg.GasTipCap, msg.BlobGasFeeCap, bh)
//	//
//	////hash := types2.BytesToHash(etherBlock.Hash().Bytes())
//	//
//	//// TODO env blobBaseFee, blobHashes
//	//env := substate.NewEnv(types2.Address(etherBlock.Coinbase()), etherBlock.Difficulty(), etherBlock.GasLimit(), etherBlock.Number().Uint64(), etherBlock.Time(), etherBlock.BaseFee(), etherBlock.BaseFee(), nil)
//
//	//recording := substate.NewSubstate(
//	//	statedb.GetSubstatePreAlloc(),
//	//	statedb.GetSubstatePostAlloc(),
//	//	env,
//	//	mg,
//	//	substate.NewResult(receipt),
//	//	blockNumber.Uint64(),
//	//	i,
//	//)
//
//
//	substate.PutSubstate(block.NumberU64(), txCounter, recording)
//}

// WriteUnprocessedSkippedTxToFile writes the skipped transaction states to a file
func WriteUnprocessedSkippedTxToFile() error {
	if unprocessedSkippedTxs == nil {
		return nil
	}
	defer func() {
		unprocessedSkippedTxs = nil
	}()

	//	open skippedTxStatesFile for writing append only
	file, err := os.OpenFile(skippedTxStatesFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "")
	err = encoder.Encode(unprocessedSkippedTxs.data)
	if err != nil {
		return err
	}

	_, err = file.WriteString(fmt.Sprintf("%d: %s", unprocessedSkippedTxs.blockNumber, buffer.String()))
	if err != nil {
		return err
	}

	return file.Close()
}

func RegisterSkippedTx(block uint64, txCounter int, pre substate.WorldState, post substate.WorldState) error {
	if unprocessedSkippedTxs == nil {
		unprocessedSkippedTxs = &skippedTxData{
			blockNumber: block,
			data:        make(map[int]map[string]interface{}),
		}
	}

	unprocessedSkippedTxs.data[txCounter] = map[string]interface{}{
		"pre":  pre,
		"post": post,
	}
	return nil
}
