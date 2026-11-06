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

package evmstore

import (
	"fmt"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLoggingStateDB_AddLog_ForwardsLogsAndFiresHookAfterInnerStateDB(t *testing.T) {
	tests := map[string]struct {
		// assignedIndexes are the indexes the inner StateDB assigns to the
		// consecutive logs added to the wrapper.
		assignedIndexes []uint
		withHook        bool
	}{
		"no logs":                  {assignedIndexes: nil, withHook: true},
		"single log":               {assignedIndexes: []uint{0}, withHook: true},
		"multiple logs":            {assignedIndexes: []uint{0, 1, 2}, withHook: true},
		"logs late in block":       {assignedIndexes: []uint{5, 6, 7, 8}, withHook: true},
		"single log without hook":  {assignedIndexes: []uint{0}, withHook: false},
		"multiple logs w/o hook":   {assignedIndexes: []uint{3, 4}, withHook: false},
		"hook is optional (empty)": {assignedIndexes: nil, withHook: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			inner := state.NewMockStateDB(ctrl)

			var events []string
			hooks := &tracing.Hooks{}
			if test.withHook {
				hooks.OnLog = func(log *types.Log) {
					events = append(events, fmt.Sprintf("hook:%d", log.Index))
				}
			}
			wrapped := WrapStateDbWithLogger(inner, hooks)

			var want []string
			for _, index := range test.assignedIndexes {
				log := &types.Log{Address: common.Address{1}, Data: []byte{byte(index)}}
				inner.EXPECT().AddLog(log).Do(func(log *types.Log) {
					events = append(events, "inner")
					log.Index = index
				})
				want = append(want, "inner")
				if test.withHook {
					want = append(want, fmt.Sprintf("hook:%d", index))
				}
				wrapped.AddLog(log)
			}

			require.Equal(want, events)
		})
	}
}

func TestLoggingStateDB_AddLog_HookIndexesMatchLogsOfRealStateDB(t *testing.T) {
	tests := map[string]struct {
		// logsPerTx is the number of logs emitted by each transaction of the
		// block, in order.
		logsPerTx []int
	}{
		"single tx without logs":     {logsPerTx: []int{0}},
		"single tx with one log":     {logsPerTx: []int{1}},
		"single tx with many logs":   {logsPerTx: []int{5}},
		"multiple txs one log each":  {logsPerTx: []int{1, 1, 1}},
		"multiple txs many logs":     {logsPerTx: []int{2, 1, 3}},
		"empty tx between log txs":   {logsPerTx: []int{2, 0, 2}},
		"leading empty txs":          {logsPerTx: []int{0, 0, 3}},
		"trailing empty txs":         {logsPerTx: []int{3, 0, 0}},
		"many txs growing log count": {logsPerTx: []int{1, 2, 3, 4, 5}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			db := newInMemoryCarmenStateDB(t)
			var hookIndexes []uint
			wrapped := WrapStateDbWithLogger(db, &tracing.Hooks{
				OnLog: func(log *types.Log) {
					hookIndexes = append(hookIndexes, log.Index)
				},
			})

			wrapped.BeginBlock(1)

			var wantIndexes []uint
			var receiptIndexes []uint
			for txIndex, numLogs := range test.logsPerTx {
				txHash := common.Hash{byte(txIndex + 1)}
				wrapped.SetTxContext(txHash, txIndex)

				for i := range numLogs {
					wrapped.AddLog(&types.Log{
						Address: common.Address{byte(txIndex + 1)},
						Data:    []byte{byte(i)},
					})
					wantIndexes = append(wantIndexes, uint(len(wantIndexes)))
				}

				// This is the source of the log index reported in receipts.
				logs := wrapped.GetLogs(txHash, common.Hash{})
				require.Len(logs, numLogs)
				for _, log := range logs {
					receiptIndexes = append(receiptIndexes, log.Index)
				}
				wrapped.EndTransaction()
			}

			require.Equal(wantIndexes, hookIndexes, "block-level index seen by the hook")
			require.Equal(receiptIndexes, hookIndexes, "index seen by the hook differs from receipt")
		})
	}
}

func TestLoggingStateDB_AddLog_RevertedLogsFreeTheirIndexForLaterLogs(t *testing.T) {
	tests := map[string]struct {
		// logsBeforeSnapshot, logsReverted and logsAfterRevert are numbers of
		// logs added to a single transaction around a snapshot revert.
		logsBeforeSnapshot int
		logsReverted       int
		logsAfterRevert    int
	}{
		"revert at start of tx":         {0, 2, 1},
		"revert after one log":          {1, 1, 1},
		"revert multiple logs":          {2, 3, 2},
		"revert without following logs": {2, 2, 0},
		"revert nothing":                {2, 0, 2},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			db := newInMemoryCarmenStateDB(t)
			var hookIndexes []uint
			wrapped := WrapStateDbWithLogger(db, &tracing.Hooks{
				OnLog: func(log *types.Log) {
					hookIndexes = append(hookIndexes, log.Index)
				},
			})

			txHash := common.Hash{1}
			wrapped.BeginBlock(1)
			wrapped.SetTxContext(txHash, 0)

			addLogs := func(n int) []uint {
				hookIndexes = nil
				for range n {
					wrapped.AddLog(&types.Log{Address: common.Address{1}})
				}
				return hookIndexes
			}

			addLogs(test.logsBeforeSnapshot)
			snapshot := wrapped.Snapshot()
			reverted := addLogs(test.logsReverted)
			wrapped.RevertToSnapshot(snapshot)
			afterRevert := addLogs(test.logsAfterRevert)

			// Indexes handed out to reverted logs are handed out again.
			for i := range reverted {
				require.Equal(uint(test.logsBeforeSnapshot+i), reverted[i])
			}
			for i := range afterRevert {
				require.Equal(uint(test.logsBeforeSnapshot+i), afterRevert[i])
			}

			// Surviving logs carry exactly the indexes the hook saw for them.
			logs := wrapped.GetLogs(txHash, common.Hash{})
			require.Len(logs, test.logsBeforeSnapshot+test.logsAfterRevert)
			for i, log := range logs {
				require.Equal(uint(i), log.Index)
			}
		})
	}
}

// newInMemoryCarmenStateDB creates a CarmenStateDB on top of a real in-memory
// Carmen state, such that log indexes are assigned by the real implementation.
func newInMemoryCarmenStateDB(t *testing.T) *CarmenStateDB {
	t.Helper()
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:   "go-memory",
		Schema:    carmen.Schema(5),
		Archive:   carmen.NoArchive,
		Directory: t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, carmenState.Close())
	})
	return CreateCarmenStateDb(carmen.CreateStateDBUsing(carmenState), nil)
}
