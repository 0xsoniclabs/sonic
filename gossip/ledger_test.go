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

package gossip

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

func TestLedger_Depth_CountsTheBlocksInFlight(t *testing.T) {
	l := &ledger{}
	require.Zero(t, l.Depth())

	l.stack = []*blockProcessor{{}, {}}
	require.Equal(t, 2, l.Depth())
}

func TestLedger_Commit_TakesTheOldestBlockAndPopsIt(t *testing.T) {
	ctrl := gomock.NewController(t)
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	// Only the oldest block may be committed: the archive is append-only, so the
	// committed chain can only ever advance in order.
	oldest := state.NewMockStagedBlock(ctrl)
	oldest.EXPECT().Commit().Return(nil)
	newest := state.NewMockStagedBlock(ctrl) // no Commit expected

	l := &ledger{store: store}
	l.stack = []*blockProcessor{
		finalizedBlockProcessorAt(l, oldest, 1),
		finalizedBlockProcessorAt(l, newest, 2),
	}

	committed, err := l.Commit()
	require.NoError(t, err)
	require.Equal(t, uint64(1), committed.Block.Number, "Commit must take the oldest block")
	require.Equal(t, 1, l.Depth(), "a committed block must leave the stack")
	require.Equal(t, uint64(2), l.stack[0].params.Number, "the newer block must remain")
}

func TestLedger_Commit_NothingInFlight_IsRejected(t *testing.T) {
	l := &ledger{}
	_, err := l.Commit()
	require.Error(t, err, "committing without a block must be reported, not ignored")
}

func TestLedger_Commit_UnfinalizedBlock_IsRejected(t *testing.T) {
	l := &ledger{}
	l.stack = []*blockProcessor{{params: BlockParams{Number: 1}}}

	_, err := l.Commit()
	require.Error(t, err, "a block that was never finalized has no content to commit")
	require.Equal(t, 1, l.Depth(), "a rejected commit must not pop the block")
}

func TestLedger_RevertTo_DiscardsFromTheTopNewestFirst(t *testing.T) {
	ctrl := gomock.NewController(t)

	// The order is structural, not conventional: a block is applied on top of the
	// one below it, so only the newest can be taken back out of the live state.
	rolledBack := []uint64{}
	l := &ledger{}
	for _, number := range []uint64{1, 2, 3} {
		staged := state.NewMockStagedBlock(ctrl)
		staged.EXPECT().Rollback().DoAndReturn(func() error {
			rolledBack = append(rolledBack, number)
			return nil
		})
		l.stack = append(l.stack, finalizedBlockProcessorAt(l, staged, number))
	}

	require.NoError(t, l.RevertTo(0))
	require.Equal(t, []uint64{3, 2, 1}, rolledBack, "blocks must be rolled back newest first")
	require.Zero(t, l.Depth())
}

func TestLedger_RevertTo_KeepsTheSurvivingPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Reverting a mispredicted suffix and rebuilding on what survives is the
	// ordinary path, so a partial revert must leave the prefix untouched.
	kept := state.NewMockStagedBlock(ctrl) // no Rollback expected
	dropped := state.NewMockStagedBlock(ctrl)
	dropped.EXPECT().Rollback().Return(nil)

	l := &ledger{}
	l.stack = []*blockProcessor{
		finalizedBlockProcessorAt(l, kept, 1),
		finalizedBlockProcessorAt(l, dropped, 2),
	}

	require.NoError(t, l.RevertTo(1))
	require.Equal(t, 1, l.Depth())
	require.Equal(t, uint64(1), l.stack[0].params.Number)
}

func TestLedger_RevertTo_OutOfRangeDepth_IsRejected(t *testing.T) {
	l := &ledger{}
	l.stack = []*blockProcessor{{params: BlockParams{Number: 1}}}

	require.Error(t, l.RevertTo(-1), "a negative depth is not a prefix of the stack")
	require.Error(t, l.RevertTo(2), "a depth beyond the stack cannot be reverted to")
	require.Equal(t, 1, l.Depth(), "a rejected revert must not change the stack")
}

func TestLedger_RevertTo_CurrentDepth_KeepsEverything(t *testing.T) {
	ctrl := gomock.NewController(t)

	staged := state.NewMockStagedBlock(ctrl) // no Rollback expected
	l := &ledger{}
	l.stack = []*blockProcessor{finalizedBlockProcessorAt(l, staged, 1)}

	require.NoError(t, l.RevertTo(1))
	require.Equal(t, 1, l.Depth())
}

func TestLedger_Run_NoBlockInFlight_Panics(t *testing.T) {
	// Assembling without a block is a programming error. Returning a zero value
	// would quietly build the wrong block instead of reporting it.
	l := &ledger{}
	require.Panics(t, func() { l.StateDB() })
	require.Panics(t, func() { l.Run(nil) })
	require.Panics(t, func() { l.runInternal(nil) })
	require.Panics(t, func() { l.Finalize() })
}

func TestLedger_ParentFor_SpeculativeTip_IsTheParent(t *testing.T) {
	// The point of the whole stack: the parent of the next block is one the ledger
	// has executed but not committed, so it is in no store to be read from.
	l := &ledger{}
	l.stack = []*blockProcessor{finalizedBlockProcessorAt(l, nil, 7)}

	parent, err := l.parentFor(8)
	require.NoError(t, err)
	require.NotNil(t, parent)
	require.Equal(t, uint64(7), parent.Number.Uint64())
}

func TestLedger_ParentFor_NotTheTipSuccessor_IsRejected(t *testing.T) {
	// A caller and the ledger disagreeing on the height means one of them is
	// working from a stale head, which must not produce a block at all.
	l := &ledger{}
	l.stack = []*blockProcessor{finalizedBlockProcessorAt(l, nil, 7)}

	_, err := l.parentFor(9)
	require.Error(t, err, "a block may not skip a height")
	_, err = l.parentFor(7)
	require.Error(t, err, "a block may not re-use its parent's height")
}

func TestLedger_ParentFor_UnfinalizedTip_IsRejected(t *testing.T) {
	l := &ledger{}
	l.stack = []*blockProcessor{{params: BlockParams{Number: 7}}}

	_, err := l.parentFor(8)
	require.Error(t, err, "a block still being assembled has no hash to chain onto")
}

func TestLedger_ParentFor_GenesisBlock_HasNoParent(t *testing.T) {
	l := &ledger{}
	parent, err := l.parentFor(0)
	require.NoError(t, err)
	require.Nil(t, parent, "block 0 chains onto nothing")
}

func TestLedger_StagedHeaders_SkipsABlockStillBeingAssembled(t *testing.T) {
	l := &ledger{}
	l.stack = []*blockProcessor{
		finalizedBlockProcessorAt(l, nil, 1),
		{params: BlockParams{Number: 2}}, // still being assembled
	}

	headers := l.stagedHeaders()
	require.Len(t, headers, 1, "a block without a hash yet cannot serve as an ancestor")
	require.Equal(t, uint64(1), headers[0].Number.Uint64())
}

// TestNet146Block8054923GasLimit_IsCanonicalHeaderValue pins the hard-coded gas
// limit to the canonical header value of net-146 block 8054923. This value is
// what protects historical byte-equivalence; it must never silently change.
func TestNet146Block8054923GasLimit_IsCanonicalHeaderValue(t *testing.T) {
	require.Equal(t, uint64(0x12a05f200), uint64(net146Block8054923GasLimit))
}

// TestBlockProcessor_Finalize_AppliesNet146GasLimitException verifies that
// Finalize stamps the hard-coded net-146 gas limit for block 8054923 on network
// 146 regardless of the rules' MaxBlockGas, and leaves the announced gas limit
// untouched for any other block.
func TestBlockProcessor_Finalize_AppliesNet146GasLimitException(t *testing.T) {
	const announcedGasLimit = uint64(123)

	tests := map[string]struct {
		networkID uint64
		number    uint64
		want      uint64
	}{
		"net-146 block 8054923 uses the hard-coded exception": {
			networkID: 146,
			number:    8054923,
			want:      net146Block8054923GasLimit,
		},
		"different block on net 146 keeps the announced limit": {
			networkID: 146,
			number:    8054924,
			want:      announcedGasLimit,
		},
		"same block number on a different network keeps the announced limit": {
			networkID: 250,
			number:    8054923,
			want:      announcedGasLimit,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{BaseFee: big.NewInt(0)},
			}, 0, types.Receipts{}, nil)

			processor := &blockProcessor{
				evmProcessor: evmProcessor,
				blockBuilder: inter.NewBlockBuilder().
					WithNumber(test.number).
					WithGasLimit(announcedGasLimit),
				params: BlockParams{
					Number: test.number,
					// A MaxBlockGas distinct from the hard-coded constant proves
					// the stamped value comes from the constant, not the rules.
					Rules: opera.Rules{
						NetworkID: test.networkID,
						Blocks:    opera.BlocksRules{MaxBlockGas: net146Block8054923GasLimit + 1},
					},
				},
			}

			candidate := processor.finalize()
			require.Equal(test.want, candidate.Block.GasLimit)
		})
	}
}

// finalizedBlockProcessor builds a blockProcessor for block 1 holding a finalized
// candidate backed by the given staged block, as Finalize would leave it. The
// ledger back pointer is supplied so commit can reach the store.
func finalizedBlockProcessor(ledger *ledger, staged state.StagedBlock, blockTime inter.Timestamp) *blockProcessor {
	bp := finalizedBlockProcessorAt(ledger, staged, 1)
	bp.candidate.Block = inter.NewBlockBuilder().
		WithNumber(1).
		WithTime(blockTime).
		Build()
	return bp
}

// finalizedBlockProcessorAt builds a blockProcessor for the given block number
// holding a finalized candidate, as Finalize would leave it.
func finalizedBlockProcessorAt(ledger *ledger, staged state.StagedBlock, number uint64) *blockProcessor {
	block := inter.NewBlockBuilder().WithNumber(number).Build()
	return &blockProcessor{
		l:            ledger,
		blockBuilder: inter.NewBlockBuilder().WithNumber(number),
		params:       BlockParams{Number: number},
		candidate: &BlockCandidate{
			Block: block,
			// A non-empty TxHash marks the block as complete, which the EVM block
			// cache insists on.
			EvmBlock: &evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{
					Number: new(big.Int).SetUint64(number),
					Hash:   block.Hash(),
					TxHash: types.EmptyTxsHash,
				},
			},
			staged: staged,
		},
	}
}

func TestBlockProcessor_Commit_CommitsTheStagedBlockWithoutWaitingForIt(t *testing.T) {
	ctrl := gomock.NewController(t)
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	// Wait belongs to Publish: committing must not be paced by the archive, or a
	// consensus executing ahead of finality would be stalled by it.
	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Commit().Return(nil)

	processor := finalizedBlockProcessor(&ledger{store: store}, staged, inter.Timestamp(time.Now().UnixNano()))
	processor.commit()
}

func TestLedger_Publish_WaitsForTheStagedBlockOfARecentBlock(t *testing.T) {
	ctrl := gomock.NewController(t)

	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Wait().Return(nil)

	ledger := &ledger{}
	processor := finalizedBlockProcessor(ledger, staged, inter.Timestamp(time.Now().UnixNano()))
	ledger.Publish(processor.candidate)
}

func TestLedger_Publish_WaitsEvenWithoutAFeed(t *testing.T) {
	ctrl := gomock.NewController(t)

	// A ledger without a feed -- the replay harness, or a ledger opened by a tool --
	// still has to observe the outcome of its archive write.
	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Wait().Return(nil)

	ledger := &ledger{config: LedgerConfig{Feed: nil}}
	processor := finalizedBlockProcessor(ledger, staged, inter.Timestamp(time.Now().UnixNano()))
	ledger.Publish(processor.candidate)
}

func TestLedger_Publish_DoesNotWaitForABlockOlderThanOneHour(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Catching up on history must not be paced by the archive, so an old block is
	// left to reach it asynchronously.
	staged := state.NewMockStagedBlock(ctrl)
	// No Wait is expected.

	old := time.Now().Add(-1*time.Hour - time.Second)
	ledger := &ledger{}
	processor := finalizedBlockProcessor(ledger, staged, inter.Timestamp(old.UnixNano()))
	ledger.Publish(processor.candidate)
}

func TestBlockProcessor_Rollback_RollsBackTheStagedBlockAndDropsTheCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)

	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Rollback().Return(nil)

	processor := finalizedBlockProcessor(&ledger{}, staged, inter.Timestamp(time.Now().UnixNano()))
	require.NoError(t, processor.rollback())
	require.Nil(t, processor.candidate, "a rolled back block must not be left behind as a candidate")

	// A second rollback has nothing to take back and must say so rather than
	// silently succeed.
	require.Error(t, processor.rollback())
}

func TestBlockProcessor_Rollback_ReportsTheErrorOfTheStagedBlock(t *testing.T) {
	ctrl := gomock.NewController(t)

	injected := fmt.Errorf("injected error")
	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Rollback().Return(injected)

	processor := finalizedBlockProcessor(&ledger{}, staged, inter.Timestamp(time.Now().UnixNano()))
	require.ErrorIs(t, processor.rollback(), injected)
}

func TestBlockProcessor_Rollback_DoesNotWriteTheBlockToTheStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Rollback().Return(nil)

	processor := finalizedBlockProcessor(&ledger{store: store}, staged, inter.Timestamp(time.Now().UnixNano()))
	blockHash := processor.candidate.Block.Hash()
	require.NoError(t, processor.rollback())

	// Commit is what writes a block to the store, and a rolled back block never
	// reaches it.
	require.Nil(t, store.GetBlock(1), "a rolled back block must not be in the store")
	require.Nil(t, store.GetBlockIndex(hash.Event(blockHash)), "a rolled back block must not be indexed")
}

func TestStagedChain_Header_PrefersASpeculativeBlockOverTheStore(t *testing.T) {
	// A block that is staged but not committed is in the live state and in no
	// store. Serving it here is what lets the BLOCKHASH walk cross it rather than
	// stop at it.
	staged := &evmcore.EvmHeader{
		Number: big.NewInt(2),
		Hash:   common.Hash{0x22},
	}
	chain := &stagedChain{
		staged:   []*evmcore.EvmHeader{staged},
		fallback: headerChain{2: {Number: big.NewInt(2), Hash: common.Hash{0xff}}},
	}

	got := chain.Header(common.Hash{}, 2)
	require.NotNil(t, got)
	require.Equal(t, common.Hash{0x22}, got.Hash, "the speculative block must win over a stale store entry")
}

func TestStagedChain_Header_FallsBackToTheStoreBelowTheStack(t *testing.T) {
	// The walk must continue past the staged blocks into committed history; only
	// the few blocks in flight are missing from the store.
	chain := &stagedChain{
		staged:   []*evmcore.EvmHeader{{Number: big.NewInt(2), Hash: common.Hash{0x22}}},
		fallback: headerChain{1: {Number: big.NewInt(1), Hash: common.Hash{0x11}}},
	}

	got := chain.Header(common.Hash{}, 1)
	require.NotNil(t, got)
	require.Equal(t, common.Hash{0x11}, got.Hash)
}

func TestStagedChain_Header_UnknownBlock_ReturnsNil(t *testing.T) {
	chain := &stagedChain{fallback: headerChain{}}
	require.Nil(t, chain.Header(common.Hash{}, 9))
}

func TestStagedChain_Header_VerificationHashNamingAnotherBlock_ReturnsNil(t *testing.T) {
	// The verification hash is how a caller pins which block it means; a
	// speculative block must honour it exactly as the store does.
	chain := &stagedChain{
		staged: []*evmcore.EvmHeader{{Number: big.NewInt(2), Hash: common.Hash{0x22}}},
	}

	require.Nil(t, chain.Header(common.Hash{0x33}, 2), "a mismatching verification hash names another block")
	require.NotNil(t, chain.Header(common.Hash{0x22}, 2), "the matching verification hash must be accepted")
}

// headerChain is a DummyChain serving a fixed set of headers by number, standing
// in for the store.
type headerChain map[uint64]*evmcore.EvmHeader

func (c headerChain) Header(verificationHash common.Hash, number uint64) *evmcore.EvmHeader {
	header, found := c[number]
	if !found {
		return nil
	}
	if (verificationHash != common.Hash{}) && verificationHash != header.Hash {
		return nil
	}
	return header
}
