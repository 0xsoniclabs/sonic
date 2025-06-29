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

package inter

import (
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"slices"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Block represents the on-disk storage format of a block. It contains all
// fields required to reconstruct the block header, as well as a list of
// hashes of the transactions being executed as part of the represented block.
//
// This struct should be considered immutable. No fields should be modified,
// directly or indirectly. Ideally, all fields should be private, but that
// would invalidate support for RLP encoding as it is used to store instances
// on the disk. However, future updates may make fields inaccessible.
//
// To create a new block, use the BlockBuilder, handling the computation of
// key properties implicitly.
type Block struct {
	blockData

	// The hash of this block, cached on first access.
	hash atomic.Pointer[common.Hash]
}

// blockData is a helper type to retain the fields of a block. It is a trivially
// copyable struct, which allows for a reliable copy of the block data. Note that
// due to the use of atomic pointers, the Block struct is not trivially copyable.
type blockData struct {
	// Fields required for the block header.
	Number               uint64
	ParentHash           common.Hash
	StateRoot            common.Hash
	Time                 Timestamp
	Difficulty           uint64
	GasLimit             uint64
	GasUsed              uint64
	BaseFee              *big.Int
	PrevRandao           common.Hash
	TransactionsHashRoot common.Hash
	ReceiptsHashRoot     common.Hash
	LogBloom             types.Bloom

	// Fields required for linking blocks to contained transactions.
	TransactionHashes []common.Hash

	// Fields required for linking the block internally to a lachesis epoch.
	Epoch idx.Epoch

	// The duration of this block, being the difference between the predecessor
	// block's timestamp and this block's timestamp, in nanoseconds.
	Duration uint64
}

// Hash computes the hash of this block, committing all its fields.
func (b *Block) Hash() common.Hash {
	if ptr := b.hash.Load(); ptr != nil {
		return *ptr
	}
	hash := b.GetEthereumHeader().Hash()
	b.hash.Store(&hash)
	return hash
}

// EncodeRLP encodes the essential data of the block using RLP.
func (b *Block) EncodeRLP(out io.Writer) error {
	// Only the block data needs to be encoded.
	return rlp.Encode(out, b.blockData)
}

// DecodeRLP decodes a RLP encoded block.
func (b *Block) DecodeRLP(in *rlp.Stream) error {
	b.hash.Store(nil)
	return in.Decode(&b.blockData)
}

// GetEthereumHeader returns the Ethereum header corresponding to this block.
func (b *Block) GetEthereumHeader() *types.Header {
	return &types.Header{
		ParentHash:  b.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{}, // < in Sonic, the coinbase is always 0
		Root:        b.StateRoot,
		TxHash:      b.TransactionsHashRoot,
		ReceiptHash: b.ReceiptsHashRoot,
		Bloom:       b.LogBloom,
		Difficulty:  big.NewInt(int64(b.Difficulty)),
		Number:      big.NewInt(int64(b.Number)),
		GasLimit:    b.GasLimit,
		GasUsed:     b.GasUsed,
		Time:        uint64(b.Time.Time().Unix()),
		Extra: EncodeExtraData(
			b.Time.Time(),
			time.Duration(b.Duration)*time.Nanosecond,
		),
		MixDigest: b.PrevRandao,
		Nonce:     types.BlockNonce{}, // constant 0 in Ethereum
		BaseFee:   b.BaseFee,

		// Sonic does not have a beacon chain and no withdrawals.
		WithdrawalsHash: &types.EmptyWithdrawalsHash,

		// Sonic does not support blobs, so no blob gas is used and there is
		// no excess blob gas.
		BlobGasUsed:   new(uint64), // = 0
		ExcessBlobGas: new(uint64), // = 0
	}
}

// EncodeExtraData produces the ExtraData field encoding Sonic-specific data
// in the Ethereum block header. This data includes:
//   - the nano-second part of the block's timestamp, for sub-second precision;
//   - the duration of the block, in nanoseconds, defined as the time elapsed
//     between the predecessor block's timestamp and this block's timestamp.
//     This is used for the computation of gas rates to adjust the base fee.
func EncodeExtraData(time time.Time, duration time.Duration) []byte {
	if duration < 0 {
		duration = 0
	}
	extra := make([]byte, 12)
	binary.BigEndian.PutUint32(extra[:4], uint32(time.Nanosecond()))
	binary.BigEndian.PutUint64(extra[4:], uint64(duration.Nanoseconds()))
	return extra
}

// DecodeExtraData decodes the ExtraData field encoding Sonic-specific data
// in the Ethereum block header. See EncodeExtraData for details.
func DecodeExtraData(extra []byte) (
	nanos int,
	duration time.Duration,
	err error,
) {
	if len(extra) != 12 {
		return 0, 0, errors.New("extra data must be 12 bytes long")
	}
	return int(binary.BigEndian.Uint32(extra[:4])),
		time.Duration(binary.BigEndian.Uint64(extra[4:])),
		nil
}

func (b *Block) EstimateSize() int {
	return int(unsafe.Sizeof(*b)) +
		len(b.TransactionHashes)*int(unsafe.Sizeof(common.Hash{}))
}

// ----------------------------------------------------------------------------
// BlockBuilder
// ----------------------------------------------------------------------------

type BlockBuilder struct {
	block        blockData
	transactions types.Transactions
	receipts     types.Receipts
}

func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{}
}

func (b *BlockBuilder) WithNumber(number uint64) *BlockBuilder {
	b.block.Number = number
	return b
}

func (b *BlockBuilder) WithParentHash(hash common.Hash) *BlockBuilder {
	b.block.ParentHash = hash
	return b
}

func (b *BlockBuilder) WithStateRoot(hash common.Hash) *BlockBuilder {
	b.block.StateRoot = hash
	return b
}

func (b *BlockBuilder) GetTransactions() types.Transactions {
	return slices.Clone(b.transactions)
}

func (b *BlockBuilder) AddTransaction(
	transaction *types.Transaction,
	receipt *types.Receipt,
) *BlockBuilder {
	b.transactions = append(b.transactions, transaction)
	b.receipts = append(b.receipts, receipt)
	return b
}

func (b *BlockBuilder) WithTime(time Timestamp) *BlockBuilder {
	b.block.Time = time
	return b
}

func (b *BlockBuilder) WithDuration(duration time.Duration) *BlockBuilder {
	if duration < 0 {
		duration = 0
	}
	b.block.Duration = uint64(duration.Nanoseconds())
	return b
}

func (b *BlockBuilder) WithDifficulty(difficulty uint64) *BlockBuilder {
	b.block.Difficulty = difficulty
	return b
}

func (b *BlockBuilder) WithGasLimit(gasLimit uint64) *BlockBuilder {
	b.block.GasLimit = gasLimit
	return b
}

func (b *BlockBuilder) WithGasUsed(gasUsed uint64) *BlockBuilder {
	b.block.GasUsed = gasUsed
	return b
}

func (b *BlockBuilder) WithBaseFee(baseFee *big.Int) *BlockBuilder {
	b.block.BaseFee = new(big.Int).Set(baseFee)
	return b
}

func (b *BlockBuilder) WithPrevRandao(prevRandao common.Hash) *BlockBuilder {
	b.block.PrevRandao = prevRandao
	return b
}

func (b *BlockBuilder) WithEpoch(epoch idx.Epoch) *BlockBuilder {
	b.block.Epoch = epoch
	return b
}

func (b *BlockBuilder) Build() *Block {
	res := new(Block)
	res.blockData = b.block

	res.TransactionsHashRoot = types.DeriveSha(
		b.transactions,
		trie.NewStackTrie(nil),
	)
	res.ReceiptsHashRoot = types.DeriveSha(b.receipts, trie.NewStackTrie(nil))
	res.LogBloom = types.MergeBloom(b.receipts)

	for _, tx := range b.transactions {
		res.TransactionHashes = append(res.TransactionHashes, tx.Hash())
	}

	return res
}
