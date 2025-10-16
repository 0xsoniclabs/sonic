package substate

import (
	"math/big"

	"github.com/0xsoniclabs/substate/substate"
	stypes "github.com/0xsoniclabs/substate/types"
	"github.com/0xsoniclabs/substate/types/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var unprocessedException *unprocessedExceptionData

type unprocessedExceptionData struct {
	blockNumber uint64
	data        map[int]substate.WorldState
}

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

	// TODO handle SetCodeAuthorization whenever they are added to sonic client
	setCodeAuthorizations := []stypes.SetCodeAuthorization{}

	txTypeProtobuf := int32(txType)
	return substate.NewMessage(
		msg.Nonce,
		!msg.SkipNonceChecks,
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
		HashListToSubstate(msg.BlobHashes),
		setCodeAuthorizations)
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

// WriteUnprocessedSkippedTxToDatabase writes the skipped transaction states
func WriteUnprocessedSkippedTxToDatabase() error {
	if unprocessedException == nil {
		return nil
	}
	defer func() {
		unprocessedException = nil
	}()

	exc := &substate.Exception{
		Block: OldBlockNumber,
		Data:  substate.ExceptionBlock{},
	}

	if len(unprocessedException.data) == 1 && TxLastIndex == 0 {
		// there was only single skipped transaction and nothing else
		pre := unprocessedException.data[0]
		exc.Data.PreBlock = &pre
		return staticExceptionDB.PutException(exc)
	}

	for txIdx, alloc := range unprocessedException.data {
		if txIdx != TxLastIndex {
			if exc.Data.Transactions == nil {
				exc.Data.Transactions = make(map[int]substate.ExceptionTx)
			}
			// fixing after skipped transaction the preTransaction at index of next valid transaction
			exc.Data.Transactions[txIdx] = substate.ExceptionTx{
				PreTransaction: &alloc,
			}
		} else {
			// last transaction was skipped, fix before state hash
			exc.Data.PostBlock = &alloc
		}
	}

	return staticExceptionDB.PutException(exc)
}

func RegisterSkippedTx(block uint64, txIndex int, alloc substate.WorldState) error {
	if unprocessedException == nil {
		unprocessedException = &unprocessedExceptionData{
			blockNumber: block,
			data:        make(map[int]substate.WorldState),
		}
	}

	if tx, exists := unprocessedException.data[txIndex]; !exists {
		// if this is the first skipped transaction at this txIndex, we can just add it
		unprocessedException.data[txIndex] = alloc
	} else {
		// if there were two or more skipped transactions right after each other, we need to merge them
		tx.Merge(alloc)
		unprocessedException.data[txIndex] = tx
	}
	return nil
}
