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
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
)

func TestLedger_Head_ReturnsHandleToBlockAtHeadPointer(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	headIdx := store.GetLatestBlockIndex()

	head, err := NewLedger(store).Head()
	require.NoError(t, err)
	require.Equal(t, store.GetBlock(headIdx).Hash(), head.Block().Hash())
}

func TestLedger_Head_HeadBlockMissingFromStore_ReportsError(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	bs := store.GetBlockState()
	bs.LastBlock.Idx++
	store.SetBlockEpochState(bs, store.GetEpochState())

	head, err := NewLedger(store).Head()
	require.ErrorContains(t, err, "not found in store")
	require.Nil(t, head)
}

func TestHead_BeginBlock_OpensLiveStateAtHeadStateRoot(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store).Head()
	require.NoError(t, err)

	next := BlockParams{Number: head.Block().Number + 1}
	executing, err := head.BeginBlock(next)
	require.NoError(t, err)
	require.Equal(t, head.Block().StateRoot, executing.stateDB().GetStateHash())
}

func TestHead_BeginBlock_SecondCallOnSameHead_ReportsError(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store).Head()
	require.NoError(t, err)
	next := BlockParams{Number: head.Block().Number + 1}
	_, err = head.BeginBlock(next)
	require.NoError(t, err)

	executing, err := head.BeginBlock(next)
	require.ErrorContains(t, err, "head already consumed")
	require.Nil(t, executing)
}

func TestHead_BeginBlock_FailedCallDoesNotConsumeHead(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store).Head()
	require.NoError(t, err)
	_, err = head.BeginBlock(BlockParams{Number: head.Block().Number + 2})
	require.Error(t, err)

	_, err = head.BeginBlock(BlockParams{Number: head.Block().Number + 1})
	require.NoError(t, err)
}

func TestHead_BeginBlock_RejectsBlockNotOnTheLedgerHead(t *testing.T) {
	tests := map[string]struct {
		setup   func(l *ledger, h Head) BlockParams
		wantErr string
	}{
		"number does not follow the head": {
			setup: func(_ *ledger, h Head) BlockParams {
				return BlockParams{Number: h.Block().Number + 2}
			},
			wantErr: "does not follow head",
		},
		"head pointer moved on": {
			setup: func(l *ledger, h Head) BlockParams {
				bs := l.store.GetBlockState()
				bs.LastBlock.Idx++
				l.store.SetBlockEpochState(bs, l.store.GetEpochState())
				return BlockParams{Number: h.Block().Number + 1}
			},
			wantErr: "no longer the ledger's head",
		},
		"head block replaced": {
			setup: func(l *ledger, h Head) BlockParams {
				number := h.Block().Number
				l.store.SetBlock(idx.Block(number), inter.NewBlockBuilder().
					WithNumber(number).
					Build())
				return BlockParams{Number: h.Block().Number + 1}
			},
			wantErr: "no longer the ledger's head",
		},
		"live state does not match the head": {
			setup: func(l *ledger, h Head) BlockParams {
				block := inter.NewBlockBuilder().
					WithNumber(h.Block().Number).
					WithStateRoot(common.Hash{1}).
					Build()
				l.store.SetBlock(idx.Block(block.Number), block)
				h.(*head).block = block
				return BlockParams{Number: block.Number + 1}
			},
			wantErr: "unexpected state root",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newInMemoryStoreWithGenesisData(
				t, opera.GetSonicUpgrades(), 1, 2,
			)
			l := &ledger{store: store}
			head, err := l.Head()
			require.NoError(t, err)

			executing, err := head.BeginBlock(test.setup(l, head))
			require.ErrorContains(t, err, test.wantErr)
			require.Nil(t, executing)
		})
	}
}
