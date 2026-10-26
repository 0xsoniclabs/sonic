// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// countingChain is a chain whose block k has the hash k+1, and
// counts the headers read.
type countingChain struct {
	length uint64
	reads  int
}

func (c *countingChain) Header(_ common.Hash, number uint64) *EvmHeader {
	c.reads++
	if number >= c.length {
		return nil
	}
	return &EvmHeader{Number: new(big.Int).SetUint64(number), ParentHash: common.BigToHash(new(big.Int).SetUint64(number))}
}

func TestGetHashFn_ABlockAtOrAboveTheReferenceReadsNoHeader(t *testing.T) {
	chain := &countingChain{length: 1000}
	ref := &EvmHeader{Number: big.NewInt(1000), ParentHash: common.BigToHash(big.NewInt(1000))}
	get := GetHashFn(ref, chain)
	require.Equal(t, common.Hash{}, get(1000))
	require.Equal(t, common.Hash{}, get(5000))
	require.Zero(t, chain.reads)

	require.Equal(t, ref.ParentHash, get(999))
	require.Equal(t, common.BigToHash(big.NewInt(999)), get(998), "the hash of block 998")
	require.Equal(t, 1, chain.reads)
}
