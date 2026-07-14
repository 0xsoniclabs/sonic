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
	"math"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// VerifyBlocks verifies the entire chain of blocks starting from the given
// genesis. It processes each block in sequence, applying all transactions and
// ensuring that the resulting block hashes match the expected values.
//
// Verification runs over three independent paths that must all reproduce the
// recorded block hashes:
//   - verifyBlocksOnState replays the blocks directly on a Carmen state and
//     re-derives each block hash by hand;
//   - verifyBlocksOnLedger replays them through the production block-processing
//     pipeline (a real gossip ledger over a real store); and
//   - verifyBlocksOnLedgerWithRollback replays them through that same pipeline
//     while taking a run of blocks back again mid-chain, which must not change
//     the blocks it then produces.
func VerifyBlocks(
	t *testing.T,
	genesis *makefakegenesis.GenesisJson,
	blocks []*types.Block,
) {
	require := require.New(t)
	require.NotEmpty(blocks)
	require.Equal(uint64(0), blocks[0].NumberU64())

	verifyBlocksOnState(t, genesis, blocks)
	verifyBlocksOnLedger(t, genesis, blocks)
	verifyBlocksOnLedgerWithRollback(t, genesis, blocks)
}

// verifyBlocksOnState verifies the chain of blocks by replaying them directly on
// a standalone Carmen state, checking the state root, gas usage, and receipts
// hash of each block, and re-deriving and comparing the full block hash.
func verifyBlocksOnState(
	t *testing.T,
	genesis *makefakegenesis.GenesisJson,
	blocks []*types.Block,
) {
	require := require.New(t)

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
			genesis.Rules,
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
			require.Equal(usedGas, r.CumulativeGasUsed)
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
			WithPrevRandao(block.MixDigest()).
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

// verifyBlocksOnLedger verifies the chain of blocks by replaying them through the
// production block-processing pipeline: a real gossip ledger driven over a real
// store (opened through integration.Ledger in replay mode). For each block it
// drives BeginBlock -> Run -> Finalize and asserts that the assembled block
// reproduces the recorded state root, gas usage, and -- decisively -- the block
// hash. Each block is committed so the next block's base fee (derived from the
// parent header) and state root can be reproduced.
//
// Replay mode is required: a live ledger re-generates the post-execution
// transactions that subsidies and bundles produce, but a historical block
// already contains them, so re-generating would diverge. Because those
// transactions are already present, all of a block's transactions can be replayed
// in their recorded order as a single batch with a generous (block-gas-limit,
// unbounded-size) budget that never skips any.
func verifyBlocksOnLedger(
	t *testing.T,
	genesis *makefakegenesis.GenesisJson,
	blocks []*types.Block,
) {
	require := require.New(t)

	dataDir := initLedgerDataDirFromGenesis(t, genesis)

	// Replay mode: historical blocks already contain the post-execution
	// transactions that subsidies and bundles generate, so the ledger must not
	// re-generate them.
	ledger, err := integration.OpenLedger(dataDir, gossip.LedgerConfig{Replay: true})
	require.NoError(err, "failed to open ledger")
	defer func() {
		require.NoError(ledger.Close())
	}()

	// The genesis bakes one or more blocks into the store (block 0 plus the
	// chain-initialization block(s)); the ledger only produces blocks after them.
	// Replay therefore starts at the first block past the genesis head, opening
	// against its finalized state root (the Carmen world-state root that
	// GetHeadBlock returns and verifies against the live state).
	head, err := ledger.GetHeadBlock()
	require.NoError(err, "failed to read ledger head")
	firstReplayed := head.Number + 1
	liveStateRoot := head.StateRoot
	require.Less(firstReplayed, uint64(len(blocks)),
		"no blocks beyond the genesis head to replay")

	// The last genesis-baked block must match the recorded chain.
	require.Equal(blocks[firstReplayed-1].Hash(), head.Hash(),
		"genesis head block hash mismatch")

	// Replay every block after the genesis head in order, chaining on the live
	// state root produced by each block.
	for i := firstReplayed; i < uint64(len(blocks)); i++ {
		processor, candidate := replayBlockOnLedger(t, ledger, genesis, blocks[i], liveStateRoot)

		// Persist the block so the next block can read it as its parent (e.g. for
		// the base fee), and chain on the state root just produced.
		processor.Commit()
		processor.Publish()
		liveStateRoot = candidate.Block.StateRoot
	}
}

// replayBlockOnLedger drives one recorded block through the ledger and asserts
// that the assembled candidate reproduces it exactly. The block is left finalized:
// its content is in the live state, but the caller decides whether to commit it or
// roll it back again.
func replayBlockOnLedger(
	t *testing.T,
	ledger *integration.Ledger,
	genesis *makefakegenesis.GenesisJson,
	block *types.Block,
	parentStateRoot common.Hash,
) (gossip.BlockProcessor, *gossip.BlockCandidate) {
	t.Helper()
	require := require.New(t)

	nanos, duration, err := inter.DecodeExtraData(block.Header().Extra)
	require.NoError(err, "block %d: failed to decode extra data", block.NumberU64())

	processor, err := ledger.BeginBlock(gossip.BlockParams{
		Number: block.NumberU64(),
		Time:   inter.Timestamp(block.Time()*1e9 + uint64(nanos)),
		// Epoch is left zero: it does not feed the block hash.
		ParentHash:      block.ParentHash(),
		PrevRandao:      block.MixDigest(),
		ParentStateRoot: parentStateRoot,
		GasLimit:        block.GasLimit(),
		// Unused by the single system batch below; set for clarity.
		UserGasLimit: block.GasLimit(),
		Duration:     duration,
		Rules:        genesis.Rules,
	}, func(*core_types.Log) {})
	require.NoError(err, "block %d: failed to begin block", block.NumberU64())

	// Replay all of the block's transactions in their recorded order as a
	// single batch. The replaying EVM module executes them faithfully without
	// re-generating subsidy/bundle post-transactions (which the block already
	// contains, interleaved with the user transactions), so the order — and
	// thus the transactions and receipts roots — is reproduced exactly. Only
	// already-included transactions are fed, with the user gas limit set to the
	// full block gas limit and the full block size budget, so neither limit
	// binds and no transaction is skipped.
	processor.Run(block.Transactions())
	candidate := processor.Finalize()

	require.Zero(candidate.NumSkipped,
		"block %d: some transactions were skipped", block.NumberU64())
	require.Equal(block.Root(), common.Hash(candidate.Block.StateRoot),
		"block %d: state root mismatch", block.NumberU64())
	require.Equal(block.GasUsed(), candidate.Block.GasUsed,
		"block %d: gas used mismatch", block.NumberU64())
	require.Equal(block.Hash(), candidate.Block.Hash(),
		"block %d: block hash mismatch", block.NumberU64())

	return processor, candidate
}

// verifyBlocksOnLedgerWithRollback verifies that the ledger can execute a block,
// take it back again, and reproduce it exactly on a second attempt.
//
// This is what a consensus that certifies a block before finalizing it needs: it
// must execute a block to obtain the hash a quorum certifies, and discard it again
// when the round fails to certify. The decisive properties are that a rolled back
// and re-executed block is byte-identical to the first attempt, and that the
// archive -- which is append-only and rejects a height it has already seen -- only
// ever receives the blocks that were committed, so the rolled back height can be
// produced again.
//
// The store this runs over keeps an archive (DefaultStoreConfig), so the archive
// path is exercised rather than assumed.
func verifyBlocksOnLedgerWithRollback(
	t *testing.T,
	genesis *makefakegenesis.GenesisJson,
	blocks []*types.Block,
) {
	require := require.New(t)

	dataDir := initLedgerDataDirFromGenesis(t, genesis)
	ledger, err := integration.OpenLedger(dataDir, gossip.LedgerConfig{Replay: true})
	require.NoError(err, "failed to open ledger")
	defer func() {
		require.NoError(ledger.Close())
	}()

	head, err := ledger.GetHeadBlock()
	require.NoError(err, "failed to read ledger head")
	firstReplayed := head.Number + 1
	liveStateRoot := head.StateRoot

	last := uint64(len(blocks)) - 1
	require.GreaterOrEqual(last, firstReplayed, "no blocks beyond the genesis head to replay")

	// Only one block is taken back at a time, because that is as deep as this seam
	// currently reaches. Carmen stages any number of blocks, but BeginBlock derives
	// the parent hash and the next base fee from a header it reads out of the store
	// (see EvmStateReader in ledger.go and EVMModule.Start), and only Commit puts a
	// block there. Beginning block N+1 while N is merely staged therefore finds no
	// parent. Executing several blocks ahead of the decision to keep them -- which a
	// consensus certifying blocks before finalizing them does -- needs the parent
	// header of a staged block to be reachable; that belongs with the speculation
	// stack rather than here.
	const window = uint64(1)
	// Take the block back mid-chain rather than at the tip, so the rollback acts on
	// a trie that already holds history and the chain visibly continues past it.
	pivot := (firstReplayed + last) / 2
	if pivot+window > last {
		pivot = last - window
	}

	// Replay and commit the chain up to the pivot.
	for i := firstReplayed; i <= pivot; i++ {
		processor, candidate := replayBlockOnLedger(t, ledger, genesis, blocks[i], liveStateRoot)
		processor.Commit()
		processor.Publish()
		liveStateRoot = candidate.Block.StateRoot
	}
	pivotStateRoot := liveStateRoot

	// Execute the next blocks without deciding their fate, as a consensus does
	// while it is still waiting for them to be certified. Each of them must be the
	// block that was recorded, which replayBlockOnLedger asserts.
	speculate := func() ([]gossip.BlockProcessor, common.Hash) {
		processors := []gossip.BlockProcessor{}
		root := pivotStateRoot
		for i := pivot + 1; i <= pivot+window; i++ {
			processor, candidate := replayBlockOnLedger(t, ledger, genesis, blocks[i], root)
			processors = append(processors, processor)
			root = candidate.Block.StateRoot
		}
		return processors, root
	}

	processors, speculatedRoot := speculate()

	// Take them all back, newest first, and the live state must return to the
	// pivot.
	for i := len(processors) - 1; i >= 0; i-- {
		require.NoError(processors[i].Rollback(),
			"block %d: failed to roll back", blocks[pivot+1+uint64(i)].NumberU64())
	}

	// Nothing of the rolled back blocks may have reached the archive, so the whole
	// run can be repeated -- and must reproduce exactly the same blocks.
	processors, reSpeculatedRoot := speculate()
	require.Equal(speculatedRoot, reSpeculatedRoot,
		"re-executing the rolled back blocks produced a different state root")

	// Keep them this time, oldest first -- the order the append-only archive
	// requires.
	for _, processor := range processors {
		processor.Commit()
		processor.Publish()
	}
	liveStateRoot = reSpeculatedRoot

	// The archive must hold every committed block and nothing else: had a rolled
	// back block reached it, re-committing its height would have been rejected as
	// already present.
	archiveHeight, empty, err := ledger.Store().EvmStore().GetArchiveBlockHeight()
	require.NoError(err, "failed to read the archive height")
	require.False(empty, "the archive is empty after committing blocks")
	require.Equal(pivot+window, archiveHeight,
		"the archive does not hold exactly the committed blocks")

	// The chain continues to work after the rollback: the remaining blocks replay
	// on top of it as usual.
	for i := pivot + window + 1; i <= last; i++ {
		processor, candidate := replayBlockOnLedger(t, ledger, genesis, blocks[i], liveStateRoot)
		processor.Commit()
		processor.Publish()
		liveStateRoot = candidate.Block.StateRoot
	}
}

// initLedgerDataDirFromGenesis creates a temporary Sonic data directory and
// initializes an on-disk gossip store in it from the given JSON genesis,
// returning the data directory for integration.OpenLedger to reopen.
//
// It mirrors the production genesis-import flow (open the store WITHOUT opening
// the EVM store, ApplyGenesis, Commit), since the genesis import requires an
// empty carmen directory — which OpenLedger's EvmStore().Open() would populate.
func initLedgerDataDirFromGenesis(t *testing.T, genesis *makefakegenesis.GenesisJson) string {
	t.Helper()
	require := require.New(t)

	dataDir := t.TempDir()
	chaindataDir := filepath.Join(dataDir, "chaindata")
	carmenDir := filepath.Join(dataDir, "carmen")
	require.NoError(os.MkdirAll(chaindataDir, 0700))
	require.NoError(os.MkdirAll(carmenDir, 0700))

	genStore, err := makefakegenesis.ApplyGenesisJson(genesis, dataDir)
	require.NoError(err, "failed to build genesis store")

	dbs, err := integration.GetDbProducer(chaindataDir, integration.DBCacheConfig{
		Cache:   480 << 20,
		Fdlimit: 100,
	})
	require.NoError(err)

	cfg := gossip.DefaultStoreConfig(cachescale.Identity)
	cfg.EVM.StateDb.Directory = carmenDir
	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(err)

	require.NoError(store.ApplyGenesis(genStore.Genesis()))
	require.NoError(store.Commit())
	require.NoError(store.Close())
	require.NoError(dbs.Close())
	return dataDir
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

		var balance amount.Amount
		if account.Balance != nil {
			balance = amount.NewFromUint256(account.Balance)
		}
		s.db.AddBalance(cc.Address(address), balance)
		s.db.SetNonce(cc.Address(address), account.Nonce)
		for key, value := range account.Storage {
			s.db.SetState(cc.Address(address), cc.Key(key), cc.Value(value))
		}
	}
	s.db.EndTransaction()
	staged, err := s.db.EndBlock(0)
	if err != nil {
		return fmt.Errorf("failed to apply the genesis block: %w", err)
	}
	// The genesis state is never taken back.
	if err := staged.Commit(); err != nil {
		return fmt.Errorf("failed to commit the genesis block: %w", err)
	}
	return s.db.Check()
}

// ApplyBlock applies the given block to this state, processing all transactions
// and updating the state accordingly. It returns the receipts of the transactions
// in the block, or an error if the block could not be processed.
func (s *State) ApplyBlock(
	rules opera.Rules,
	block *types.Block,
) (types.Receipts, error) {

	chainConfig := opera.CreateTransientEvmChainConfig(
		rules.NetworkID,
		[]opera.UpgradeHeight{{Height: 0, Upgrades: rules.Upgrades}},
		idx.Block(block.NumberU64()),
	)

	processor := evmcore.NewStateProcessorForReplay(
		chainConfig,
		historyAdapter{history: s.blockHashHistory},
		rules.Upgrades,
	)

	evmBlock := &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:      block.Number(),
			ParentHash:  block.ParentHash(),
			Time:        inter.Timestamp(block.Time() * 1e9),
			GasLimit:    block.GasLimit(),
			PrevRandao:  block.Header().MixDigest,
			BaseFee:     block.BaseFee(),
			BlobBaseFee: big.NewInt(1),
		},
		Transactions: block.Transactions(),
	}

	stateDb := evmstore.CreateCarmenStateDb(s.db, nil)

	vmConfig := opera.GetVmConfig(rules)
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
		math.MaxUint64, // the blocks have already been produced, the size limit is not relevant for the replay
	).ProcessedTransactions

	receipts := types.Receipts{}
	for i, cur := range processed {
		if cur.Receipt == nil {
			return nil, fmt.Errorf("failed to process tx %d in block %d", i, block.NumberU64())
		}
		receipts = append(receipts, cur.Receipt)
	}

	staged, err := s.db.EndBlock(block.NumberU64())
	if err != nil {
		return nil, fmt.Errorf("failed to apply block %d: %w", block.NumberU64(), err)
	}
	// This replay path only moves forwards, so every block it applies is kept.
	if err := staged.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit block %d: %w", block.NumberU64(), err)
	}
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
