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
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/logger"
)

//go:generate go tool mockgen -destination=kvdb_mock_test.go -package=evmstore github.com/Fantom-foundation/lachesis-base/kvdb Store

func TestFindReceiptGasAmendments_ReturnsDifferenceToDerivedGasUsed(t *testing.T) {
	tests := map[string]struct {
		cumulative []uint64
		gasUsed    []uint64
		want       []ReceiptGasAmendment
	}{
		"NoReceipts": {},
		"NoLeak": {
			cumulative: []uint64{21_000, 42_000, 63_000},
			gasUsed:    []uint64{21_000, 21_000, 21_000},
		},
		"LeakBeforeFirst": {
			cumulative: []uint64{71_000, 92_000},
			gasUsed:    []uint64{21_000, 21_000},
			want:       []ReceiptGasAmendment{{Index: 0, GasExcess: 50_000}},
		},
		"LeakBeforeMiddle": {
			cumulative: []uint64{21_000, 92_000, 113_000},
			gasUsed:    []uint64{21_000, 21_000, 21_000},
			want:       []ReceiptGasAmendment{{Index: 1, GasExcess: 50_000}},
		},
		"LeakBeforeLast": {
			cumulative: []uint64{21_000, 42_000, 113_000},
			gasUsed:    []uint64{21_000, 21_000, 21_000},
			want:       []ReceiptGasAmendment{{Index: 2, GasExcess: 50_000}},
		},
		"MultipleLeaks": {
			cumulative: []uint64{31_000, 52_000, 103_000},
			gasUsed:    []uint64{21_000, 21_000, 21_000},
			want: []ReceiptGasAmendment{
				{Index: 0, GasExcess: 10_000},
				{Index: 2, GasExcess: 30_000},
			},
		},
		"GasUsedAboveDerivedWraps": {
			cumulative: []uint64{21_000, 41_000},
			gasUsed:    []uint64{21_000, 21_000},
			want:       []ReceiptGasAmendment{{Index: 1, GasExcess: math.MaxUint64 - 1_000 + 1}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			receipts := makeReceipts(test.cumulative, test.gasUsed)
			require.Equal(t, test.want, findReceiptGasAmendments(receipts))
		})
	}
}

func TestFindReceiptGasAmendments_AmendmentsRestoreGasUsed(t *testing.T) {
	gasUsed := []uint64{21_000, 30_000, 0, math.MaxUint64}
	cumulative := []uint64{100_000, 50_000, 50_000, 7}
	receipts := makeReceipts(cumulative, gasUsed)

	amendments := findReceiptGasAmendments(receipts)

	derived := makeReceipts(cumulative, deriveGasUsed(cumulative))
	for _, a := range amendments {
		derived[a.Index].GasUsed -= a.GasExcess
	}
	for i := range receipts {
		require.Equal(t, gasUsed[i], derived[i].GasUsed, "receipt %d", i)
	}
}

func TestStore_setReceiptGasAmendments_StoresNothingWithoutLeaks(t *testing.T) {
	store, _, _ := storeTableLogMocks(t)

	// The mocked table and logger fail the test on any call.
	store.setReceiptGasAmendments(1, makeReceipts(
		[]uint64{21_000, 42_000},
		[]uint64{21_000, 21_000},
	))
}

func TestStore_setReceiptGasAmendments_StoresAmendmentsOfBlock(t *testing.T) {
	logger.SetTestMode(t)
	store := cachedStore()

	for _, block := range []idx.Block{0, 1, math.MaxUint64} {
		t.Run(fmt.Sprintf("Block=%d", block), func(t *testing.T) {
			receipts := makeReceipts(
				[]uint64{21_000, 92_000, 113_000},
				[]uint64{21_000, 21_000, 21_000},
			)
			store.setReceiptGasAmendments(block, receipts)

			want := []ReceiptGasAmendment{{Index: 1, GasExcess: 50_000}}
			require.Equal(t, want, store.getReceiptGasAmendments(block))
		})
	}
	require.Nil(t, store.getReceiptGasAmendments(2))
}

func TestStore_setReceiptGasAmendments_WarnsAboutAmendments(t *testing.T) {
	store, table, log := storeTableLogMocks(t)

	want := []ReceiptGasAmendment{{Index: 1, GasExcess: 50_000}}
	encoded, err := rlp.EncodeToBytes(want)
	require.NoError(t, err)

	log.EXPECT().Warn("Receipt gas differs from cumulative gas", "block", idx.Block(5), "amendments", want)
	table.EXPECT().Put(idx.Block(5).Bytes(), encoded)

	store.setReceiptGasAmendments(5, makeReceipts(
		[]uint64{21_000, 92_000},
		[]uint64{21_000, 21_000},
	))
}

func TestStore_setReceiptGasAmendments_LogsOnPutError(t *testing.T) {
	store, table, log := storeTableLogMocks(t)

	injectedErr := errors.New("put error")
	log.EXPECT().Warn(gomock.Any(), gomock.Any())
	table.EXPECT().Put(gomock.Any(), gomock.Any()).Return(injectedErr)

	expectCrit(log, "Failed to store receipt gas amendments", "err", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("Failed to store receipt gas amendments: %v", []any{"err", injectedErr}),
		func() {
			store.setReceiptGasAmendments(1, makeReceipts([]uint64{71_000}, []uint64{21_000}))
		})
}

func TestStore_getReceiptGasAmendments_ReturnsNilForUnknownBlock(t *testing.T) {
	logger.SetTestMode(t)
	store := cachedStore()
	require.Nil(t, store.getReceiptGasAmendments(1))
}

func TestStore_getReceiptGasAmendments_LogsOnGetError(t *testing.T) {
	store, table, log := storeTableLogMocks(t)

	injectedErr := errors.New("get error")
	table.EXPECT().Get(idx.Block(1).Bytes()).Return(nil, injectedErr)

	expectCrit(log, "Failed to get receipt gas amendments", "err", injectedErr)
	// In production, a Crit log call causes the logger to exit the process.
	// To prevent the test from exiting, the mock logger is configured to panic instead.
	require.PanicsWithValue(t,
		fmt.Sprintf("Failed to get receipt gas amendments: %v", []any{"err", injectedErr}),
		func() { store.getReceiptGasAmendments(1) })
}

func TestStore_getReceiptGasAmendments_LogsOnInvalidEncoding(t *testing.T) {
	store, table, log := storeTableLogMocks(t)

	table.EXPECT().Get(gomock.Any()).Return([]byte{1, 2, 3}, nil)

	log.EXPECT().Crit("Failed to decode receipt gas amendments", "err", gomock.Any()).
		Do(func(msg string, ctx ...any) { panic(msg) })
	require.PanicsWithValue(t, "Failed to decode receipt gas amendments",
		func() { store.getReceiptGasAmendments(1) })
}

func TestStore_applyReceiptGasAmendments_SubtractsAmendedGas(t *testing.T) {
	logger.SetTestMode(t)
	store := cachedStore()

	cumulative := []uint64{31_000, 52_000, 103_000}
	gasUsed := []uint64{21_000, 21_000, 21_000}
	store.setReceiptGasAmendments(1, makeReceipts(cumulative, gasUsed))

	derived := makeReceipts(cumulative, deriveGasUsed(cumulative))
	store.applyReceiptGasAmendments(1, derived)
	for i, r := range derived {
		require.Equal(t, gasUsed[i], r.GasUsed, "receipt %d", i)
	}
}

func TestStore_applyReceiptGasAmendments_LeavesReceiptsOfOtherBlocksUntouched(t *testing.T) {
	logger.SetTestMode(t)
	store := cachedStore()

	store.setReceiptGasAmendments(1, makeReceipts([]uint64{71_000}, []uint64{21_000}))

	receipts := makeReceipts([]uint64{71_000}, []uint64{71_000})
	store.applyReceiptGasAmendments(2, receipts)
	require.EqualValues(t, 71_000, receipts[0].GasUsed)
}

func TestStore_applyReceiptGasAmendments_SkipsAmendmentsBeyondReceipts(t *testing.T) {
	logger.SetTestMode(t)
	store := cachedStore()
	store.putReceiptGasAmendments(1,
		[]ReceiptGasAmendment{
			{Index: 0, GasExcess: 1},
			{Index: 2, GasExcess: 1},
		})

	receipts := makeReceipts([]uint64{10, 20}, []uint64{10, 10})
	store.applyReceiptGasAmendments(1, receipts)
	require.EqualValues(t, 9, receipts[0].GasUsed)
	require.EqualValues(t, 10, receipts[1].GasUsed)

	// Amendments of a block without receipts.
	require.NotPanics(t, func() { store.applyReceiptGasAmendments(1, types.Receipts{}) })
}

func makeReceipts(cumulative, gasUsed []uint64) types.Receipts {
	receipts := make(types.Receipts, len(cumulative))
	for i := range cumulative {
		receipts[i] = &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: cumulative[i],
			GasUsed:           gasUsed[i],
			Logs:              []*types.Log{},
		}
	}
	return receipts
}

// deriveGasUsed mirrors the GasUsed derivation of types.Receipts.DeriveFields.
func deriveGasUsed(cumulative []uint64) []uint64 {
	gasUsed := make([]uint64, len(cumulative))
	var prev uint64
	for i, c := range cumulative {
		gasUsed[i] = c - prev
		prev = c
	}
	return gasUsed
}

func storeTableLogMocks(t *testing.T) (*Store, *MockStore, *logger.MockLogger) {
	ctrl := gomock.NewController(t)
	store := &Store{}
	table := NewMockStore(ctrl)
	store.table.ReceiptGasAmendments = table
	log := logger.NewMockLogger(ctrl)
	store.Log = log
	return store, table, log
}

// expectCrit sets up the given mock logger to expect a Crit call with the given
// message and arguments, and to panic with a message containing both.
func expectCrit(log *logger.MockLogger, msg string, args ...any) {
	log.EXPECT().Crit(msg, args...).
		Do(func(msg string, ctx ...any) {
			panic(fmt.Sprintf("%v: %v", msg, ctx))
		})
}
