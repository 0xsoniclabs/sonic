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

	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestStore_Header_ReturnsHeaderOfBlockUpToHeadMatchingHash(t *testing.T) {
	const head = idx.Block(12)
	blocks := map[idx.Block]*inter.Block{}
	for _, n := range []idx.Block{head - 1, head, head + 1} {
		blocks[n] = inter.NewBlockBuilder().WithNumber(uint64(n)).Build()
	}

	tests := map[string]struct {
		number        idx.Block
		hash          common.Hash
		valueReturned bool
	}{
		"below head":     {number: head - 1, valueReturned: true},
		"head":           {number: head, valueReturned: true},
		"above head":     {number: head + 1},
		"matching hash":  {number: head, hash: blocks[head].Hash(), valueReturned: true},
		"different hash": {number: head, hash: common.Hash{1}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store, err := NewMemStore(t)
			require.NoError(t, err)
			for n, block := range blocks {
				store.SetBlock(n, block)
			}
			setLatestBlockIndex(store, head)

			header := store.Header(test.hash, uint64(test.number))

			if !test.valueReturned {
				require.Nil(t, header)
				return
			}
			require.Equal(t, blocks[test.number].Hash(), header.Hash)
			require.Equal(t, uint64(test.number), header.Number.Uint64())
		})
	}
}

func TestStore_GetLiveStateDb_NotOpen_ReportsError(t *testing.T) {
	_, err := (&Store{evm: &evmstore.Store{}}).GetLiveStateDb(hash.Hash{})
	require.ErrorContains(t, err, "EvmStore is not open")
}
