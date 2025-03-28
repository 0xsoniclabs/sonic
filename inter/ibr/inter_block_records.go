package ibr

import (
	"math/big"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/core/types"
)

type LlrFullBlockRecord struct {
	BlockHash  consensus.Hash
	ParentHash consensus.Hash
	StateRoot  consensus.Hash
	Time       inter.Timestamp
	Duration   uint64
	Difficulty uint64
	GasLimit   uint64
	GasUsed    uint64
	BaseFee    *big.Int
	PrevRandao consensus.Hash
	Epoch      consensus.Epoch
	Txs        types.Transactions
	Receipts   []*types.ReceiptForStorage
}

type LlrIdxFullBlockRecord struct {
	LlrFullBlockRecord
	Idx consensus.BlockID
}

// FullBlockRecordFor returns the full block record used in Genesis processing
// for the given block, list of transactions, and list of transaction receipts.
func FullBlockRecordFor(block *inter.Block, txs types.Transactions,
	rawReceipts []*types.ReceiptForStorage) *LlrFullBlockRecord {
	return &LlrFullBlockRecord{
		BlockHash:  consensus.Hash(block.Hash()),
		ParentHash: consensus.Hash(block.ParentHash),
		StateRoot:  consensus.Hash(block.StateRoot),
		Time:       block.Time,
		Duration:   block.Duration,
		Difficulty: block.Difficulty,
		GasLimit:   block.GasLimit,
		GasUsed:    block.GasUsed,
		BaseFee:    block.BaseFee,
		PrevRandao: consensus.Hash(block.PrevRandao),
		Epoch:      block.Epoch,
		Txs:        txs,
		Receipts:   rawReceipts,
	}
}
