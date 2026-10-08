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
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/logger"
)

func equalStorageReceipts(t *testing.T, expect, got []*types.ReceiptForStorage) {
	assert.EqualValues(t, len(expect), len(got))
	for i := range expect {
		assert.EqualValues(t, expect[i].CumulativeGasUsed, got[i].CumulativeGasUsed)
		assert.EqualValues(t, expect[i].Logs, got[i].Logs)
		assert.EqualValues(t, expect[i].Status, got[i].Status)
	}
}

func TestStoreGetCachedReceipts(t *testing.T) {
	logger.SetTestMode(t)

	block, expect := fakeReceipts()
	store := cachedStore()
	store.SetRawReceipts(block, expect)

	got, _ := store.GetRawReceipts(block)
	assert.EqualValues(t, expect, got)
}

func TestStoreGetNonCachedReceipts(t *testing.T) {
	logger.SetTestMode(t)

	block, expect := fakeReceipts()
	store := nonCachedStore()
	store.SetRawReceipts(block, expect)

	got, _ := store.GetRawReceipts(block)
	equalStorageReceipts(t, expect, got)
}

func BenchmarkStoreGetRawReceipts(b *testing.B) {
	logger.SetTestMode(b)

	b.Run("cache on", func(b *testing.B) {
		benchStoreGetRawReceipts(b, cachedStore())
	})
	b.Run("cache off", func(b *testing.B) {
		benchStoreGetRawReceipts(b, nonCachedStore())
	})
}

func benchStoreGetRawReceipts(b *testing.B, store *Store) {
	block, receipt := fakeReceipts()

	store.SetRawReceipts(block, receipt)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if v, _ := store.GetRawReceipts(block); v == nil {
			b.Fatal("invalid result")
		}
	}
}

func BenchmarkStoreSetRawReceipts(b *testing.B) {
	logger.SetTestMode(b)

	b.Run("cache on", func(b *testing.B) {
		benchStoreSetRawReceipts(b, cachedStore())
	})
	b.Run("cache off", func(b *testing.B) {
		benchStoreSetRawReceipts(b, nonCachedStore())
	})
}

func benchStoreSetRawReceipts(b *testing.B, store *Store) {
	block, receipt := fakeReceipts()

	for i := 0; i < b.N; i++ {
		store.SetRawReceipts(block, receipt)
	}
}

func fakeReceipts() (idx.Block, []*types.ReceiptForStorage) {
	return idx.Block(1),
		[]*types.ReceiptForStorage{
			{
				PostState:         nil,
				Status:            0,
				CumulativeGasUsed: 0,
				Bloom:             types.Bloom{},
				Logs:              []*types.Log{},
				TxHash:            common.Hash{},
				ContractAddress:   common.Address{},
				GasUsed:           0,
				BlockHash:         common.Hash{},
				BlockNumber:       nil,
				TransactionIndex:  0,
			},
		}
}

func TestFindGasUsedOverrides(t *testing.T) {
	tests := map[string]struct {
		receipts types.Receipts
		expected []ibr.GasUsedOverride
	}{
		"no receipts": {},
		"single consistent receipt": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
			},
		},
		"consistent gas": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 30},
			},
		},
		"zero gas receipts are consistent": {
			receipts: types.Receipts{
				{GasUsed: 0, CumulativeGasUsed: 0},
				{GasUsed: 0, CumulativeGasUsed: 0},
			},
		},
		"surplus gas before first receipt": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 15},
				{GasUsed: 20, CumulativeGasUsed: 35},
			},
			expected: []ibr.GasUsedOverride{{Index: 0, GasUsed: 10}},
		},
		"surplus gas between receipts": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 35},
				{GasUsed: 5, CumulativeGasUsed: 40},
				{GasUsed: 5, CumulativeGasUsed: 55},
			},
			expected: []ibr.GasUsedOverride{{Index: 1, GasUsed: 20}, {Index: 3, GasUsed: 5}},
		},
		"surplus gas before every receipt": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 11},
				{GasUsed: 20, CumulativeGasUsed: 32},
				{GasUsed: 30, CumulativeGasUsed: 63},
			},
			expected: []ibr.GasUsedOverride{
				{Index: 0, GasUsed: 10},
				{Index: 1, GasUsed: 20},
				{Index: 2, GasUsed: 30},
			},
		},
		"surplus gas before receipt without gas used": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 0, CumulativeGasUsed: 50},
			},
			expected: []ibr.GasUsedOverride{{Index: 1, GasUsed: 0}},
		},
		"surplus gas after last receipt is not detectable": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
			},
		},
		"gas used larger than derived value is ignored": {
			receipts: types.Receipts{
				{GasUsed: 30, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 30},
			},
		},
		"decreasing cumulative gas is ignored": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 5},
			},
		},
		"recovers after decreasing cumulative gas": {
			receipts: types.Receipts{
				{GasUsed: 10, CumulativeGasUsed: 10},
				{GasUsed: 20, CumulativeGasUsed: 5},
				{GasUsed: 20, CumulativeGasUsed: 30},
			},
			expected: []ibr.GasUsedOverride{{Index: 2, GasUsed: 20}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, findGasUsedOverrides(test.receipts))
		})
	}
}

// makeTestReceipts creates receipts with the given gas used and cumulative gas
// used values, together with matching transactions.
func makeTestReceipts(gasUsed, cumulativeGasUsed []uint64) (types.Receipts, types.Transactions) {
	to := common.Address{1}
	receipts := make(types.Receipts, len(gasUsed))
	txs := make(types.Transactions, len(gasUsed))
	for i := range gasUsed {
		receipts[i] = &types.Receipt{
			Type:              types.LegacyTxType,
			GasUsed:           gasUsed[i],
			CumulativeGasUsed: cumulativeGasUsed[i],
		}
		txs[i] = types.NewTx(&types.LegacyTx{To: &to, Nonce: uint64(i)})
	}
	return receipts, txs
}

func gasUsedOf(receipts types.Receipts) []uint64 {
	var res []uint64
	for _, r := range receipts {
		res = append(res, r.GasUsed)
	}
	return res
}

func cumulativeGasUsedOf(receipts types.Receipts) []uint64 {
	var res []uint64
	for _, r := range receipts {
		res = append(res, r.CumulativeGasUsed)
	}
	return res
}

func TestStoreGetReceipts_RestoresGasUsedAfterGasSurplus(t *testing.T) {
	tests := map[string]struct {
		gasUsed           []uint64
		cumulativeGasUsed []uint64
		overridesStored   bool
	}{
		"empty block": {},
		"no surplus": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{10, 30},
		},
		"surplus before first receipt": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{110, 130},
			overridesStored:   true,
		},
		"surplus between receipts": {
			gasUsed:           []uint64{10, 20},
			cumulativeGasUsed: []uint64{10, 130}, // 100 surplus
			overridesStored:   true,
		},
		"multiple surpluses": {
			gasUsed:           []uint64{10, 20, 30, 40},
			cumulativeGasUsed: []uint64{10, 80, 110, 190}, // 50 and 40 surplus
			overridesStored:   true,
		},
		"surplus before every receipt": {
			gasUsed:           []uint64{10, 20, 30},
			cumulativeGasUsed: []uint64{11, 32, 63},
			overridesStored:   true,
		},
		"surplus before receipt without gas used": {
			gasUsed:           []uint64{10, 0, 30},
			cumulativeGasUsed: []uint64{10, 50, 80},
			overridesStored:   true,
		},
	}

	stores := map[string]func() *Store{
		"cached":    cachedStore,
		"nonCached": nonCachedStore,
	}

	for name, test := range tests {
		for storeName, newStore := range stores {
			t.Run(name+"/"+storeName, func(t *testing.T) {
				logger.SetTestMode(t)
				store := newStore()
				block := idx.Block(1)
				receipts, txs := makeTestReceipts(test.gasUsed, test.cumulativeGasUsed)

				store.SetReceipts(block, receipts)

				has, err := store.table.ReceiptGasUsedOverrides.Has(block.Bytes())
				require.NoError(t, err)
				require.Equal(t, test.overridesStored, has)

				// Read through the cache (if any) first, and then again after
				// evicting the cached value to force a load from the database.
				// Loading from the database populates the cache, so a third
				// read covers the cache hit on a restored value.
				store.cache.Receipts.Remove(block)
				for _, read := range []string{"database", "cache"} {
					got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
					require.Len(t, got, len(test.gasUsed), read)
					require.Equal(t, gasUsedOf(receipts), gasUsedOf(got), read)
					require.Equal(t, test.cumulativeGasUsed, cumulativeGasUsedOf(got), read)
				}
			})
		}
	}
}

func TestStoreSetReceipts_ReplacesOverridesOfPreviousExecution(t *testing.T) {
	tests := map[string]struct {
		first, second     []uint64 // cumulative gas used
		gasUsed           []uint64
		overridesStored   bool
		expectedGasUsed   []uint64
		expectedOverrides []ibr.GasUsedOverride
	}{
		"overrides are removed": {
			first:           []uint64{110, 130},
			second:          []uint64{10, 30},
			gasUsed:         []uint64{10, 20},
			expectedGasUsed: []uint64{10, 20},
		},
		"overrides are added": {
			first:             []uint64{10, 30},
			second:            []uint64{110, 130},
			gasUsed:           []uint64{10, 20},
			overridesStored:   true,
			expectedGasUsed:   []uint64{10, 20},
			expectedOverrides: []ibr.GasUsedOverride{{Index: 0, GasUsed: 10}},
		},
		"overrides are replaced": {
			first:             []uint64{110, 130},
			second:            []uint64{10, 130},
			gasUsed:           []uint64{10, 20},
			overridesStored:   true,
			expectedGasUsed:   []uint64{10, 20},
			expectedOverrides: []ibr.GasUsedOverride{{Index: 1, GasUsed: 20}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger.SetTestMode(t)
			store := nonCachedStore()
			block := idx.Block(1)

			first, _ := makeTestReceipts(test.gasUsed, test.first)
			second, txs := makeTestReceipts(test.gasUsed, test.second)
			store.SetReceipts(block, first)
			store.SetReceipts(block, second)

			buf, err := store.table.ReceiptGasUsedOverrides.Get(block.Bytes())
			require.NoError(t, err)
			if !test.overridesStored {
				require.Empty(t, buf)
			} else {
				var overrides []ibr.GasUsedOverride
				require.NoError(t, rlp.DecodeBytes(buf, &overrides))
				require.Equal(t, test.expectedOverrides, overrides)
			}

			got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
			require.Equal(t, test.expectedGasUsed, gasUsedOf(got))
		})
	}
}

func TestStoreGetReceipts_OverridesAreScopedToTheirBlock(t *testing.T) {
	logger.SetTestMode(t)
	store := nonCachedStore()

	withSurplus, txs1 := makeTestReceipts([]uint64{10}, []uint64{110})
	withoutSurplus, txs2 := makeTestReceipts([]uint64{10}, []uint64{10})
	store.SetReceipts(1, withSurplus)
	store.SetReceipts(2, withoutSurplus)

	got1 := store.GetReceipts(1, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs1)
	got2 := store.GetReceipts(2, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs2)
	require.Equal(t, []uint64{10}, gasUsedOf(got1))
	require.Equal(t, []uint64{10}, gasUsedOf(got2))

	has, err := store.table.ReceiptGasUsedOverrides.Has(idx.Block(2).Bytes())
	require.NoError(t, err)
	require.False(t, has)
}

func TestStoreGetReceipts_IgnoresOutOfRangeOverrides(t *testing.T) {
	tests := map[string]struct {
		overrides []ibr.GasUsedOverride
		expected  []uint64
	}{
		"index beyond receipts": {
			overrides: []ibr.GasUsedOverride{{Index: 5, GasUsed: 1}},
			expected:  []uint64{10},
		},
		"index equal to number of receipts": {
			overrides: []ibr.GasUsedOverride{{Index: 1, GasUsed: 1}},
			expected:  []uint64{10},
		},
		"out of range does not stop valid overrides": {
			overrides: []ibr.GasUsedOverride{{Index: 5, GasUsed: 1}, {Index: 0, GasUsed: 7}},
			expected:  []uint64{7},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger.SetTestMode(t)
			store := nonCachedStore()
			block := idx.Block(1)
			receipts, txs := makeTestReceipts([]uint64{10}, []uint64{10})

			store.SetReceipts(block, receipts)
			store.SetGasUsedOverrides(block, test.overrides)

			got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
			require.Equal(t, test.expected, gasUsedOf(got))
		})
	}
}

func TestStoreGetGasUsedOverrides(t *testing.T) {
	emptyList, err := rlp.EncodeToBytes([]ibr.GasUsedOverride{})
	require.NoError(t, err)

	tests := map[string]struct {
		stored   []byte
		expected []ibr.GasUsedOverride
	}{
		"nothing stored": {},
		"empty list stored": {
			stored: emptyList,
		},
		"overrides stored": {
			stored:   mustEncodeOverrides(t, []ibr.GasUsedOverride{{Index: 1, GasUsed: 2}, {Index: 3, GasUsed: 4}}),
			expected: []ibr.GasUsedOverride{{Index: 1, GasUsed: 2}, {Index: 3, GasUsed: 4}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger.SetTestMode(t)
			store := nonCachedStore()
			block := idx.Block(1)
			if test.stored != nil {
				require.NoError(t, store.table.ReceiptGasUsedOverrides.Put(block.Bytes(), test.stored))
			}
			// Results are compared with Equal, which distinguishes nil from
			// empty, since the latter alters genesis block records.
			require.Equal(t, test.expected, store.GetGasUsedOverrides(block))
		})
	}
}

func mustEncodeOverrides(t *testing.T, overrides []ibr.GasUsedOverride) []byte {
	t.Helper()
	buf, err := rlp.EncodeToBytes(overrides)
	require.NoError(t, err)
	return buf
}

// recordingStore records writes to a table, and runs a hook after each of them.
type recordingStore struct {
	kvdb.Store
	name     string
	writes   *[]string
	afterPut func()
}

func (r *recordingStore) Put(key, value []byte) error {
	err := r.Store.Put(key, value)
	*r.writes = append(*r.writes, r.name+":put")
	if r.afterPut != nil {
		r.afterPut()
	}
	return err
}

func (r *recordingStore) Delete(key []byte) error {
	err := r.Store.Delete(key)
	*r.writes = append(*r.writes, r.name+":delete")
	return err
}

func TestStoreSetReceipts_WritesOverridesBeforeReceipts(t *testing.T) {
	writers := map[string]func(*Store, idx.Block, types.Receipts){
		"SetReceipts": func(s *Store, n idx.Block, r types.Receipts) {
			s.SetReceipts(n, r)
		},
		"SetRawReceiptsWithGasUsedOverrides": func(s *Store, n idx.Block, r types.Receipts) {
			raw := make([]*types.ReceiptForStorage, len(r))
			for i := range r {
				raw[i] = (*types.ReceiptForStorage)(r[i])
			}
			s.SetRawReceiptsWithGasUsedOverrides(n, raw, findGasUsedOverrides(r))
		},
	}

	tests := map[string]struct {
		previous []uint64 // cumulative gas used of a previous execution, if any
		current  []uint64 // cumulative gas used
		expected []string
	}{
		"overrides are added": {
			current:  []uint64{10, 130},
			expected: []string{"overrides:put", "receipts:put"},
		},
		"overrides are replaced": {
			previous: []uint64{110, 130},
			current:  []uint64{10, 130},
			expected: []string{"overrides:put", "receipts:put"},
		},
		"overrides are removed": {
			previous: []uint64{110, 130},
			current:  []uint64{10, 30},
			expected: []string{"overrides:delete", "receipts:put"},
		},
		"no overrides": {
			current:  []uint64{10, 30},
			expected: []string{"receipts:put"},
		},
	}

	for writerName, write := range writers {
		for name, test := range tests {
			t.Run(writerName+"/"+name, func(t *testing.T) {
				logger.SetTestMode(t)
				store := nonCachedStore()
				block := idx.Block(1)
				gasUsed := []uint64{10, 20}

				if test.previous != nil {
					previous, _ := makeTestReceipts(gasUsed, test.previous)
					write(store, block, previous)
				}

				var writes []string
				store.table.Receipts = &recordingStore{Store: store.table.Receipts, name: "receipts", writes: &writes}
				store.table.ReceiptGasUsedOverrides = &recordingStore{Store: store.table.ReceiptGasUsedOverrides, name: "overrides", writes: &writes}

				current, _ := makeTestReceipts(gasUsed, test.current)
				write(store, block, current)

				require.Equal(t, test.expected, writes)
			})
		}
	}
}

// A reader seeing new receipts must also see their overrides. Otherwise, it
// could derive and cache wrong gas used values for them.
func TestStoreSetReceipts_ReaderSeesOverridesOfNewReceipts(t *testing.T) {
	logger.SetTestMode(t)
	store := cachedStore()
	block := idx.Block(1)
	receipts, txs := makeTestReceipts([]uint64{10, 20}, []uint64{10, 130})

	var writes []string
	var seenByReader []uint64
	store.table.ReceiptGasUsedOverrides = &recordingStore{Store: store.table.ReceiptGasUsedOverrides, name: "overrides", writes: &writes}
	store.table.Receipts = &recordingStore{
		Store:  store.table.Receipts,
		name:   "receipts",
		writes: &writes,
		// Runs right after the new receipts became visible in the database.
		afterPut: func() {
			got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
			seenByReader = gasUsedOf(got)
		},
	}

	store.SetReceipts(block, receipts)

	require.Equal(t, []uint64{10, 20}, seenByReader)
	got := store.GetReceipts(block, params.TestChainConfig, common.Hash{}, 0, nil, nil, txs)
	require.Equal(t, []uint64{10, 20}, gasUsedOf(got))
}

// failingStore fails the selected operations of a table with the given error.
type failingStore struct {
	kvdb.Store
	err        error
	failHas    bool
	failGet    bool
	failPut    bool
	failDelete bool
}

func (f *failingStore) Has(key []byte) (bool, error) {
	if f.failHas {
		return false, f.err
	}
	return f.Store.Has(key)
}

func (f *failingStore) Get(key []byte) ([]byte, error) {
	if f.failGet {
		return nil, f.err
	}
	return f.Store.Get(key)
}

func (f *failingStore) Put(key, value []byte) error {
	if f.failPut {
		return f.err
	}
	return f.Store.Put(key, value)
}

func (f *failingStore) Delete(key []byte) error {
	if f.failDelete {
		return f.err
	}
	return f.Store.Delete(key)
}

// In production, a Crit log call causes the logger to exit the process. To
// prevent the tests from exiting, the mock logger panics with the message.
func expectCrit(log *logger.MockLogger, msg string, ctx ...any) {
	log.EXPECT().Crit(msg, ctx...).
		Do(func(msg string, _ ...any) { panic(msg) })
}

func TestStoreGasUsedOverrides_DatabaseFailuresAreFatal(t *testing.T) {
	injectedErr := errors.New("injected error")
	block := idx.Block(1)
	overrides := []ibr.GasUsedOverride{{Index: 1, GasUsed: 2}}

	tests := map[string]struct {
		table   *failingStore
		stored  []byte // written to the table before the operation, if not nil
		crit    string
		critCtx []any
		run     func(*Store)
	}{
		"check for stale overrides fails": {
			table:   &failingStore{failHas: true},
			crit:    "Failed to check key",
			critCtx: []any{"err", injectedErr},
			run:     func(s *Store) { s.SetGasUsedOverrides(block, nil) },
		},
		"deletion of stale overrides fails": {
			table:   &failingStore{failDelete: true},
			stored:  mustEncodeOverrides(t, overrides),
			crit:    "Failed to delete key",
			critCtx: []any{"err", injectedErr},
			run:     func(s *Store) { s.SetGasUsedOverrides(block, nil) },
		},
		"storing overrides fails": {
			table:   &failingStore{failPut: true},
			crit:    "Failed to put key-value",
			critCtx: []any{"err", injectedErr},
			run:     func(s *Store) { s.SetGasUsedOverrides(block, overrides) },
		},
		"loading overrides fails": {
			table:   &failingStore{failGet: true},
			crit:    "Failed to get key-value",
			critCtx: []any{"err", injectedErr},
			run:     func(s *Store) { s.GetGasUsedOverrides(block) },
		},
		"stored overrides are corrupt": {
			table:   &failingStore{},
			stored:  []byte{0xff, 0x01},
			crit:    "Failed to decode rlp",
			critCtx: []any{"err", gomock.Any(), "size", 2},
			run:     func(s *Store) { s.GetGasUsedOverrides(block) },
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := nonCachedStore()
			if test.stored != nil {
				require.NoError(t, store.table.ReceiptGasUsedOverrides.Put(block.Bytes(), test.stored))
			}
			test.table.Store = store.table.ReceiptGasUsedOverrides
			test.table.err = injectedErr
			store.table.ReceiptGasUsedOverrides = test.table

			log := logger.NewMockLogger(gomock.NewController(t))
			store.Log = log
			expectCrit(log, test.crit, test.critCtx...)

			require.PanicsWithValue(t, test.crit, func() { test.run(store) })
		})
	}
}
