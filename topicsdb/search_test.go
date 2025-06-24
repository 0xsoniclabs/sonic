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
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearch(b *testing.B) {
	topics, recs, topics4rec := genTestData(1000)

	index := newTestIndex()

	for _, rec := range recs {
		err := index.Push(rec)
		require.NoError(b, err)
	}

	var query [][][]common.Hash
	for i := 0; i < len(topics); i++ {
		from, to := topics4rec(i)
		tt := topics[from : to-1]

		qq := make([][]common.Hash, len(tt))
		for pos, t := range tt {
			qq[pos] = []common.Hash{t}
		}

		query = append(query, qq)
	}

	pooled := withThreadPool{index}

	for dsc, method := range map[string]func(context.Context, idx.Block, idx.Block, [][]common.Hash) ([]*types.Log, error){
		"index":  index.FindInBlocks,
		"pooled": pooled.FindInBlocks,
	} {
		b.Run(dsc, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				qq := query[i%len(query)]
				_, err := method(nil, 0, 0xffffffff, qq)
				require.NoError(b, err)
			}
		})
	}
}
