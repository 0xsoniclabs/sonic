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

package topicsdb

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"runtime/debug"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils/dbutil/threads"
)

func TestMain(m *testing.M) {
	debug.SetMaxThreads(20)

	os.Exit(m.Run())
}

func newTestIndex() *index {
	return newIndex(memorydb.New())
}

func TestIndexSearchMultyVariants(t *testing.T) {
	logger.SetTestMode(t)
	var (
		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
		addr1 = randAddress()
		addr2 = randAddress()
		addr3 = randAddress()
		addr4 = randAddress()
	)
	testdata := []*types.Log{{
		BlockNumber: 1,
		Address:     addr1,
		Topics:      []common.Hash{hash1, hash1, hash1},
	}, {
		BlockNumber: 3,
		Address:     addr2,
		Topics:      []common.Hash{hash2, hash2, hash2},
	}, {
		BlockNumber: 998,
		Address:     addr3,
		Topics:      []common.Hash{hash3, hash3, hash3},
	}, {
		BlockNumber: 999,
		Address:     addr4,
		Topics:      []common.Hash{hash4, hash4, hash4},
	},
	}

	index := newTestIndex()

	for _, l := range testdata {
		err := index.Push(l)
		require.NoError(t, err)
	}

	// require.ElementsMatchf(testdata, got, "") doesn't work properly here,
	// so use check()
	check := func(require *require.Assertions, got []*types.Log) {
		count := 0
		for _, a := range got {
			for _, b := range testdata {
				if b.Address == a.Address {
					require.ElementsMatch(a.Topics, b.Topics)
					count++
					break
				}
			}
		}
	}

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		t.Run(dsc, func(t *testing.T) {

			t.Run("With no addresses", func(t *testing.T) {
				require := require.New(t)
				got, err := method(nil, 0, 1000, [][]common.Hash{
					{},
					{hash1, hash2, hash3, hash4},
					{},
					{hash1, hash2, hash3, hash4},
				})
				require.NoError(err)
				require.Equal(4, len(got))
				check(require, got)
			})

			t.Run("With addresses", func(t *testing.T) {
				require := require.New(t)
				got, err := method(nil, 0, 1000, [][]common.Hash{
					{common.BytesToHash(addr1[:]), common.BytesToHash(addr2[:]), common.BytesToHash(addr3[:]), common.BytesToHash(addr4[:])},
					{hash1, hash2, hash3, hash4},
					{},
					{hash1, hash2, hash3, hash4},
				})
				require.NoError(err)
				require.Equal(4, len(got))
				check(require, got)
			})

			t.Run("With block range", func(t *testing.T) {
				require := require.New(t)
				got, err := method(nil, 2, 998, [][]common.Hash{
					{common.BytesToHash(addr1[:]), common.BytesToHash(addr2[:]), common.BytesToHash(addr3[:]), common.BytesToHash(addr4[:])},
					{hash1, hash2, hash3, hash4},
					{},
					{hash1, hash2, hash3, hash4},
				})
				require.NoError(err)
				require.Equal(2, len(got))
				check(require, got)
			})

			t.Run("With addresses and blocks", func(t *testing.T) {
				require := require.New(t)

				got1, err := method(nil, 2, 998, [][]common.Hash{
					{common.BytesToHash(addr1[:]), common.BytesToHash(addr2[:]), common.BytesToHash(addr3[:]), common.BytesToHash(addr4[:])},
					{hash1, hash2, hash3, hash4},
					{},
					{hash1, hash2, hash3, hash4},
				})
				require.NoError(err)
				require.Equal(2, len(got1))
				check(require, got1)

				got2, err := method(nil, 2, 998, [][]common.Hash{
					{common.BytesToHash(addr1[:]), common.BytesToHash(addr2[:]), common.BytesToHash(addr3[:]), common.BytesToHash(addr4[:])},
					{hash1, hash2, hash3, hash4},
					{},
					{hash1, hash2, hash3, hash4},
				})
				require.NoError(err)
				require.ElementsMatch(got1, got2)
			})

		})
	}
}

func TestIndexSearchShortCircuits(t *testing.T) {
	logger.SetTestMode(t)
	var (
		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
		addr1 = randAddress()
		addr2 = randAddress()
	)
	testdata := []*types.Log{{
		BlockNumber: 1,
		Address:     addr1,
		Topics:      []common.Hash{hash1, hash2},
	}, {
		BlockNumber: 3,
		Address:     addr1,
		Topics:      []common.Hash{hash1, hash2, hash3},
	}, {
		BlockNumber: 998,
		Address:     addr2,
		Topics:      []common.Hash{hash1, hash2, hash4},
	}, {
		BlockNumber: 999,
		Address:     addr1,
		Topics:      []common.Hash{hash1, hash2, hash4},
	},
	}

	index := newTestIndex()

	for _, l := range testdata {
		err := index.Push(l)
		require.NoError(t, err)
	}

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		t.Run(dsc, func(t *testing.T) {

			t.Run("topics count 1", func(t *testing.T) {
				require := require.New(t)
				got, err := method(nil, 0, 1000, [][]common.Hash{
					{common.BytesToHash(addr1[:])},
					{},
					{},
					{hash3},
				})
				require.NoError(err)
				require.Equal(1, len(got))
			})

			t.Run("topics count 2", func(t *testing.T) {
				require := require.New(t)
				got, err := method(nil, 0, 1000, [][]common.Hash{
					{common.BytesToHash(addr1[:])},
					{},
					{},
					{hash3, hash4},
				})
				require.NoError(err)
				require.Equal(2, len(got))
			})

			t.Run("block range", func(t *testing.T) {
				require := require.New(t)
				got, err := method(nil, 3, 998, [][]common.Hash{
					{common.BytesToHash(addr1[:])},
					{},
					{},
					{hash3, hash4},
				})
				require.NoError(err)
				require.Equal(1, len(got))
			})

		})
	}
}

func TestIndexSearchSingleVariant(t *testing.T) {
	logger.SetTestMode(t)

	topics, recs, topics4rec := genTestData(100)

	index := newTestIndex()

	for _, rec := range recs {
		err := index.Push(rec)
		require.NoError(t, err)
	}

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		t.Run(dsc, func(t *testing.T) {
			require := require.New(t)

			for i := 0; i < len(topics); i++ {
				from, to := topics4rec(i)
				tt := topics[from : to-1]

				qq := make([][]common.Hash, len(tt)+1)
				for pos, t := range tt {
					qq[pos+1] = []common.Hash{t}
				}

				got, err := method(nil, 0, 1000, qq)
				require.NoError(err)

				var expect []*types.Log
				for j, rec := range recs {
					if f, t := topics4rec(j); f != from || t != to {
						continue
					}
					expect = append(expect, rec)
				}

				require.ElementsMatchf(expect, got, "step %d", i)
			}

		})
	}
}

func TestIndexSearchSimple(t *testing.T) {
	logger.SetTestMode(t)

	var (
		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
		addr  = randAddress()
	)
	testdata := []*types.Log{{
		BlockNumber: 1,
		Address:     addr,
		Topics:      []common.Hash{hash1},
	}, {
		BlockNumber: 2,
		Address:     addr,
		Topics:      []common.Hash{hash2},
	}, {
		BlockNumber: 998,
		Address:     addr,
		Topics:      []common.Hash{hash3},
	}, {
		BlockNumber: 999,
		Address:     addr,
		Topics:      []common.Hash{hash4},
	},
	}

	index := newTestIndex()

	for _, l := range testdata {
		err := index.Push(l)
		require.NoError(t, err)
	}

	var (
		got []*types.Log
		err error
	)

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		t.Run(dsc, func(t *testing.T) {
			require := require.New(t)

			got, err = method(nil, 0, 0xffffffff, [][]common.Hash{
				{common.BytesToHash(addr[:])},
				{hash1},
			})
			require.NoError(err)
			require.Equal(1, len(got))

			got, err = method(nil, 0, 0xffffffff, [][]common.Hash{
				{common.BytesToHash(addr[:])},
				{hash2},
			})
			require.NoError(err)
			require.Equal(1, len(got))

			got, err = method(nil, 0, 0xffffffff, [][]common.Hash{
				{common.BytesToHash(addr[:])},
				{hash3},
			})
			require.NoError(err)
			require.Equal(1, len(got))
		})
	}

}

func TestMaxTopicsCount(t *testing.T) {
	logger.SetTestMode(t)

	testdata := &types.Log{
		BlockNumber: 1,
		Address:     randAddress(),
		Topics:      make([]common.Hash, maxTopicsCount),
	}
	pattern := make([][]common.Hash, maxTopicsCount+1)
	pattern[0] = []common.Hash{common.BytesToHash(testdata.Address[:])}
	for i := range testdata.Topics {
		testdata.Topics[i] = common.BytesToHash([]byte(fmt.Sprintf("topic%d", i)))
		pattern[0] = append(pattern[0], testdata.Topics[i])
		pattern[i+1] = []common.Hash{testdata.Topics[i]}
	}

	index := newTestIndex()
	err := index.Push(testdata)
	require.NoError(t, err)

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		t.Run(dsc, func(t *testing.T) {
			require := require.New(t)

			got, err := method(nil, 0, 0xffffffff, pattern)
			require.NoError(err)
			require.Equal(1, len(got))
			require.Equal(maxTopicsCount, len(got[0].Topics))
		})
	}

	require.Equal(t, maxTopicsCount+1, len(pattern))
	require.Equal(t, maxTopicsCount+1, len(pattern[0]))
}

func TestPatternLimit(t *testing.T) {
	logger.SetTestMode(t)
	require := require.New(t)

	data := []struct {
		pattern [][]common.Hash
		exp     [][]common.Hash
		err     error
	}{
		{
			pattern: [][]common.Hash{},
			exp:     [][]common.Hash{},
			err:     ErrEmptyTopics,
		},
		{
			pattern: [][]common.Hash{{}, {}, {}},
			exp:     [][]common.Hash{{}, {}, {}},
			err:     ErrEmptyTopics,
		},
		{
			pattern: [][]common.Hash{
				{hash.FakeHash(1), hash.FakeHash(1)}, {hash.FakeHash(2), hash.FakeHash(2)}, {hash.FakeHash(3), hash.FakeHash(4)}},
			exp: [][]common.Hash{
				{hash.FakeHash(1)}, {hash.FakeHash(2)}, {hash.FakeHash(3), hash.FakeHash(4)}},
			err: nil,
		},
		{
			pattern: [][]common.Hash{
				{hash.FakeHash(1), hash.FakeHash(2)}, {hash.FakeHash(3), hash.FakeHash(4)}, {hash.FakeHash(5), hash.FakeHash(6)}},
			exp: [][]common.Hash{
				{hash.FakeHash(1), hash.FakeHash(2)}, {hash.FakeHash(3), hash.FakeHash(4)}, {hash.FakeHash(5), hash.FakeHash(6)}},
			err: nil,
		},
		{
			pattern: append(append(make([][]common.Hash, maxTopicsCount), []common.Hash{hash.FakeHash(1)}), []common.Hash{hash.FakeHash(1)}),
			exp:     append(make([][]common.Hash, maxTopicsCount), []common.Hash{hash.FakeHash(1)}),
			err:     nil,
		},
	}

	for i, x := range data {
		got, err := limitPattern(x.pattern)
		require.Equal(len(x.exp), len(got))
		for j := range got {
			require.ElementsMatch(x.exp[j], got[j], i, j)
		}
		require.Equal(x.err, err, i)
	}
}

func TestKvdbThreadsPoolLimit(t *testing.T) {
	logger.SetTestMode(t)

	const N = 100

	_, recs, _ := genTestData(N)
	index := newTestIndex()
	for _, rec := range recs {
		err := index.Push(rec)
		require.NoError(t, err)
	}

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		t.Run(dsc, func(t *testing.T) {
			require := require.New(t)

			topics := make([]common.Hash, threads.GlobalPool.Cap()+1)
			for i := range topics {
				topics[i] = hash.FakeHash(int64(i))
			}
			require.Less(threads.GlobalPool.Cap(), len(topics))
			qq := make([][]common.Hash, 3)

			// one big pattern
			qq[1] = topics
			got, err := method(nil, 0, 1000, qq)
			require.NoError(err)
			require.Equal(N, len(got))

			// more than one big pattern
			qq[1], qq[2] = topics, topics
			got, err = method(nil, 0, 1000, qq)
			switch dsc {
			case "index":
				require.NoError(err)
				require.Equal(N, len(got))
			case "pooled":
				require.Equal(ErrTooBigTopics, err)
				require.Equal(0, len(got))

			}

		})
	}
}

func genTestData(count int) (
	topics []common.Hash,
	recs []*types.Log,
	topics4rec func(rec int) (from, to int),
) {
	const (
		period = 5
	)

	topics = make([]common.Hash, period)
	for i := range topics {
		topics[i] = hash.FakeHash(int64(i))
	}

	topics4rec = func(rec int) (from, to int) {
		from = rec % (period - 3)
		to = from + 3
		return
	}

	recs = make([]*types.Log, count)
	for i := range recs {
		from, to := topics4rec(i)
		r := &types.Log{
			BlockNumber: uint64(i / period),
			BlockHash:   hash.FakeHash(int64(i / period)),
			TxHash:      hash.FakeHash(int64(i % period)),
			Index:       uint(i % period),
			Address:     randAddress(),
			Topics:      topics[from:to],
			Data:        make([]byte, i),
		}
		_, _ = rand.Read(r.Data)
		recs[i] = r
	}

	return
}

func randAddress() (addr common.Address) {
	n, err := rand.Read(addr[:])
	if err != nil {
		panic(err)
	}
	if n != common.AddressLength {
		panic("address is not filled")
	}
	return
}
