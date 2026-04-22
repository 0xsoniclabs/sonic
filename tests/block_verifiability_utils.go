// Copyright 2026 Sonic Operations Ltd
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

package tests

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// VerifyBlocks verifies the entire chain of blocks starting from the given
// genesis. It processes each block in sequence, applying all transactions and
// ensuring that the resulting block hashes match the expected values.
func VerifyBlocks(
	t *testing.T,
	genesis *makefakegenesis.GenesisJson,
	blocks []*types.Block,
) {
	require := require.New(t)
	require.NotEmpty(blocks)
	require.Equal(uint64(0), blocks[0].NumberU64())

	// Create a new state-DB instance.
	state, err := NewState(t.TempDir())
	require.NoError(err)
	defer func() {
		require.NoError(state.Close())
	}()

	// Load the genesis into the state-DB.
	require.NoError(state.ApplyGenesis(genesis))
	require.Equal(blocks[0].Root(), state.GetStateRoot())

	// Verify all blocks by replaying them on the state-DB.
	for i, block := range blocks {
		receipts, err := state.ApplyBlock(
			genesis.Rules.NetworkID,
			genesis.Rules.Upgrades,
			block,
		)
		require.NoError(err, "failed to apply block %d", block.NumberU64())
		require.Equal(len(block.Transactions()), len(receipts))

		// Check the state root.
		require.Equal(block.Root(), state.GetStateRoot(),
			"block %d: state root mismatch", block.NumberU64(),
		)

		// Check the reported gas used.
		usedGas := uint64(0)
		for _, r := range receipts {
			usedGas += r.GasUsed
		}
		require.Equal(block.GasUsed(), usedGas,
			"block %d, tx %d: gas used mismatch", block.NumberU64(), i,
		)

		// Check the receipts hash.
		receiptsHash := types.DeriveSha(receipts, trie.NewStackTrie(nil))
		require.Equal(block.ReceiptHash(), receiptsHash,
			"block %d, tx %d: receipts hash mismatch", block.NumberU64(), i,
		)

		// Check the full block hash.
		nanos, duration, err := inter.DecodeExtraData(block.Header().Extra)
		require.NoError(err, "block %d: failed to decode extra data", block.NumberU64())
		builder := inter.NewBlockBuilder().
			WithNumber(block.NumberU64()).
			WithParentHash(block.ParentHash()).
			WithTime(inter.Timestamp(block.Time()*1e9 + uint64(nanos))).
			WithDuration(duration).
			WithGasLimit(block.GasLimit()).
			WithBaseFee(block.BaseFee()).
			WithPrevRandao(block.MixDigest())

		builder.
			WithStateRoot(state.GetStateRoot()).
			WithGasUsed(usedGas)
		for i, tx := range block.Transactions() {
			builder.AddTransaction(tx, receipts[i])
		}

		restored := builder.Build()
		require.Equal(restored.GetEthereumHeader(), block.Header())
		require.Equal(restored.GetEthereumHeader().Time, block.Time(),
			"block %d: timestamp mismatch", block.NumberU64(),
		)

		require.Equal(block.Hash(), restored.Hash(),
			"block %d: block hash mismatch", block.NumberU64(),
		)
	}
}

// --- Block Replay Infrastructure ---

// State is an abstraction of the Chain State Database. It tracks the balances,
// nonces, codes, and storage states of accounts in the blockchain and provides
// transaction support for modifying these states.
//
// This type is an adapter for the Carmen state database, providing custom top
// level methods for managing instances in the context of the replay tool.
type State struct {
	db               carmen.StateDB
	blockHashHistory *blockHashHistory
}

// StateParameters is a configuration struct for creating a new State instance.
type StateParameters struct {
	Directory string
}

// NewState creates a new State instance with the given parameters. The
// resulting state database is empty.
//
// Successfully created instances must be closed using the Close method.
func NewState(dir string) (*State, error) {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create state dir %q; %v", dir, err)
	}

	archive := carmen.NoArchive

	state, err := carmen.NewState(carmen.Parameters{
		Directory:    dir,
		Variant:      "go-file",
		Schema:       carmen.Schema(5),
		Archive:      archive,
		LiveCache:    100 * 1024 * 1024, // 100MB
		ArchiveCache: 100 * 1024 * 1024, // 100MB
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %v", err)
	}
	db := carmen.CreateCustomStateDBUsing(state, 0)
	return &State{db: db, blockHashHistory: &blockHashHistory{}}, nil
}

// Close closes the state database and releases any resources associated with it.
// After calling Close, the State instance should not be used anymore.
// If the state database was already closed, this method has no effect.
func (s *State) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// GetStateRoot returns the current state root hash of the state database.
func (s *State) GetStateRoot() common.Hash {
	return common.Hash(s.db.GetHash())
}

// ApplyGenesis applies the genesis data from the specified file on this state.
func (s *State) ApplyGenesis(genesis *makefakegenesis.GenesisJson) error {
	// apply the genesis accounts to the state
	s.db.BeginBlock()
	s.db.BeginTransaction()
	for _, account := range genesis.Accounts {
		address := account.Address
		if len(account.Code) != 0 {
			s.db.SetCode(cc.Address(address), account.Code)
		}
		balance, err := amount.NewFromBigInt(account.Balance)
		if err != nil {
			return fmt.Errorf("invalid balance for account %s: %w", address.Hex(), err)
		}
		s.db.AddBalance(cc.Address(address), balance)
		s.db.SetNonce(cc.Address(address), account.Nonce)
		for key, value := range account.Storage {
			s.db.SetState(cc.Address(address), cc.Key(key), cc.Value(value))
		}
	}
	s.db.EndTransaction()
	s.db.EndBlock(0)
	return s.db.Check()
}

// ApplyBlock applies the given block to this state, processing all transactions
// and updating the state accordingly. It returns the receipts of the transactions
// in the block, or an error if the block could not be processed.
func (s *State) ApplyBlock(
	chainId uint64,
	upgrades opera.Upgrades,
	block *types.Block,
) (types.Receipts, error) {

	chainConfig := opera.CreateTransientEvmChainConfig(
		chainId,
		nil,
		idx.Block(block.NumberU64()),
	)

	// In verification mode, gas subsidies are disabled to avoid introducing
	// a second charge for sponsored transactions.
	upgrades.GasSubsidies = false

	processor := evmcore.NewStateProcessor(
		chainConfig,
		historyAdapter{history: s.blockHashHistory},
		upgrades,
	)

	evmBlock := &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:      block.Number(),
			ParentHash:  block.ParentHash(),
			Time:        inter.Timestamp(block.Time() * 1e9),
			GasLimit:    block.GasLimit(),
			PrevRandao:  block.Header().MixDigest,
			BaseFee:     big.NewInt(0), // < zero, to circumvent base-fee limits for sponsored txs
			BlobBaseFee: big.NewInt(1),
		},
		Transactions: block.Transactions(),
	}

	stateDb := evmstore.CreateCarmenStateDb(s.db, nil)

	vmConfig := opera.GetVmConfig(opera.Rules{})
	gasLimit := block.GasLimit()

	s.blockHashHistory.SetBlockHash(block.NumberU64()-1, block.ParentHash())

	s.db.BeginBlock()
	var usedGas uint64
	processed := processor.Process(
		evmBlock,
		stateDb,
		vmConfig,
		gasLimit,
		&usedGas,
		0, // tx index offset
		nil,
	).ProcessedTransactions

	if false { // Debug
		fmt.Printf("block %d: used gas %d / %d\n", block.NumberU64(), usedGas, block.GasUsed())
		signer := types.LatestSignerForChainID(big.NewInt(int64(chainId)))
		for _, cur := range processed {
			from, _ := signer.Sender(cur.Transaction)
			to := cur.Transaction.To()
			if cur.Receipt != nil {
				fmt.Printf("  tx %v -> %v, nonce %d (gas used %d)\n", from.Hex(), to.Hex(), cur.Transaction.Nonce(), cur.Receipt.GasUsed)
			} else {
				fmt.Printf("  tx %v -> %v, nonce %d (skipped)\n", from.Hex(), to.Hex(), cur.Transaction.Nonce())
			}
		}
	}

	receipts := types.Receipts{}
	for i, cur := range processed {
		if cur.Receipt == nil {
			return nil, fmt.Errorf("failed to process tx %d in block %d", i, block.NumberU64())
		}
		receipts = append(receipts, cur.Receipt)
	}

	s.db.EndBlock(block.NumberU64())
	return receipts, s.db.Check()
}

// --- block hash history tracking ---

// blockHashHistory keeps track of the last 256 block hashes. This is required
// for the BLOCKHASH opcode in the EVM.
type blockHashHistory struct {
	historicHashes [256]common.Hash
}

func (b *blockHashHistory) GetBlockHash(number uint64) common.Hash {
	return b.historicHashes[number%256]
}

func (b *blockHashHistory) SetBlockHash(number uint64, hash common.Hash) {
	b.historicHashes[number%256] = hash
}

// --- block hash history adapter ---

// historyAdapter implements the evmcore.DummyChain interface, allowing it to
// be used with the EVM state processor to serve historic block hashes.
type historyAdapter struct {
	history *blockHashHistory
}

func (h historyAdapter) Header(_ common.Hash, number uint64) *evmcore.EvmHeader {
	// The only information required from the header is the block number, the
	// block's hash, and the parent hash. Everything else is ignored by the EVM.
	return &evmcore.EvmHeader{
		Number:     big.NewInt(int64(number)),
		Hash:       h.history.GetBlockHash(number),
		ParentHash: h.history.GetBlockHash(number - 1),
	}
}
