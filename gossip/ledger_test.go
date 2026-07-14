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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

// TestNet146Block8054923GasLimit_IsCanonicalHeaderValue pins the hard-coded gas
// limit to the canonical header value of net-146 block 8054923. This value is
// what protects historical byte-equivalence; it must never silently change.
func TestNet146Block8054923GasLimit_IsCanonicalHeaderValue(t *testing.T) {
	require.Equal(t, uint64(0x12a05f200), uint64(net146Block8054923GasLimit))
}

// TestBlockLedger_Finalize_AppliesNet146GasLimitException verifies that
// Finalize stamps the hard-coded net-146 gas limit for block 8054923 on network
// 146 regardless of the rules' MaxBlockGas, and leaves the announced gas limit
// untouched for any other block.
func TestBlockLedger_Finalize_AppliesNet146GasLimitException(t *testing.T) {
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

			candidate := processor.Finalize()
			require.Equal(test.want, candidate.Block.GasLimit)
		})
	}
}

// finalizedBlockProcessor builds a blockProcessor holding a finalized candidate
// backed by the given staged block, as Finalize would leave it. The ledger back
// pointer is supplied so Commit and Publish can reach the store and the feed.
func finalizedBlockProcessor(ledger *ledger, staged state.StagedBlock, blockTime inter.Timestamp) *blockProcessor {
	block := inter.NewBlockBuilder().
		WithNumber(1).
		WithTime(blockTime).
		Build()
	return &blockProcessor{
		l:            ledger,
		blockBuilder: inter.NewBlockBuilder().WithNumber(1),
		params:       BlockParams{Number: 1},
		candidate: &BlockCandidate{
			Block: block,
			// A non-empty TxHash marks the block as complete, which the EVM block
			// cache insists on.
			EvmBlock: &evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{TxHash: types.EmptyTxsHash},
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
	processor.Commit()
}

func TestBlockProcessor_Publish_WaitsForTheStagedBlockOfARecentBlock(t *testing.T) {
	ctrl := gomock.NewController(t)

	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Wait().Return(nil)

	processor := finalizedBlockProcessor(&ledger{}, staged, inter.Timestamp(time.Now().UnixNano()))
	processor.Publish()
}

func TestBlockProcessor_Publish_WaitsEvenWithoutAFeed(t *testing.T) {
	ctrl := gomock.NewController(t)

	// A ledger without a feed -- the replay harness, or a ledger opened by a tool --
	// still has to observe the outcome of its archive write.
	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Wait().Return(nil)

	processor := finalizedBlockProcessor(&ledger{config: LedgerConfig{Feed: nil}}, staged, inter.Timestamp(time.Now().UnixNano()))
	processor.Publish()
}

func TestBlockProcessor_Publish_DoesNotWaitForABlockOlderThanOneHour(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Catching up on history must not be paced by the archive, so an old block is
	// left to reach it asynchronously.
	staged := state.NewMockStagedBlock(ctrl)
	// No Wait is expected.

	old := time.Now().Add(-1*time.Hour - time.Second)
	processor := finalizedBlockProcessor(&ledger{}, staged, inter.Timestamp(old.UnixNano()))
	processor.Publish()
}

func TestBlockProcessor_Rollback_RollsBackTheStagedBlockAndDropsTheCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)

	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Rollback().Return(nil)

	processor := finalizedBlockProcessor(&ledger{}, staged, inter.Timestamp(time.Now().UnixNano()))
	require.NoError(t, processor.Rollback())
	require.Nil(t, processor.candidate, "a rolled back block must not be left behind as a candidate")

	// A second rollback has nothing to take back and must say so rather than
	// silently succeed.
	require.Error(t, processor.Rollback())
}

func TestBlockProcessor_Rollback_ReportsTheErrorOfTheStagedBlock(t *testing.T) {
	ctrl := gomock.NewController(t)

	injected := fmt.Errorf("injected error")
	staged := state.NewMockStagedBlock(ctrl)
	staged.EXPECT().Rollback().Return(injected)

	processor := finalizedBlockProcessor(&ledger{}, staged, inter.Timestamp(time.Now().UnixNano()))
	require.ErrorIs(t, processor.Rollback(), injected)
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
	require.NoError(t, processor.Rollback())

	// Commit is what writes a block to the store, and a rolled back block never
	// reaches it.
	require.Nil(t, store.GetBlock(1), "a rolled back block must not be in the store")
	require.Nil(t, store.GetBlockIndex(hash.Event(blockHash)), "a rolled back block must not be indexed")
}
