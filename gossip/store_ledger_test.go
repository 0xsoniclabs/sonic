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

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestStore_Header_ReturnsHeaderOfStoredBlockMatchingHash(t *testing.T) {
	const number = idx.Block(12)
	block := inter.NewBlockBuilder().WithNumber(uint64(number)).Build()

	tests := map[string]struct {
		hash common.Hash
		want bool
	}{
		"no hash":        {want: true},
		"matching hash":  {hash: block.Hash(), want: true},
		"different hash": {hash: common.Hash{1}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store, err := NewMemStore(t)
			require.NoError(t, err)
			store.SetBlock(number, block)
			setLatestBlockIndex(store, number)

			header := store.Header(test.hash, uint64(number))

			if !test.want {
				require.Nil(t, header)
				return
			}
			require.Equal(t, block.Hash(), header.Hash)
			require.Equal(t, uint64(number), header.Number.Uint64())
		})
	}
}
