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

package ethapi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"testing"

	"github.com/0xsoniclabs/sonic/api/rpctest"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var (
	storageAddr1 = common.HexToAddress("0x1111111111111111111111111111111111111111")
	storageAddr2 = common.HexToAddress("0x2222222222222222222222222222222222222222")
	unknownAddr  = common.HexToAddress("0xdead000000000000000000000000000000000000")

	slot0 = common.BigToHash(big.NewInt(0))
	slot1 = common.BigToHash(big.NewInt(1))
	slot2 = common.BigToHash(big.NewInt(2))

	val0 = common.BigToHash(big.NewInt(42))
	val1 = common.BigToHash(big.NewInt(100))
	val2 = common.BigToHash(big.NewInt(200))
)

// newStorageValuesAPI builds a fake backend with two accounts holding
// pre-defined storage and returns the block chain API handler on top of it.
func newStorageValuesAPI(t *testing.T) *PublicBlockChainAPI {
	be := rpctest.NewBackendBuilder(t).
		WithAccount(storageAddr1, rpctest.AccountState{
			Balance: big.NewInt(1e18),
			Store: map[common.Hash]common.Hash{
				slot0: val0,
				slot1: val1,
			},
		}).
		WithAccount(storageAddr2, rpctest.AccountState{
			Balance: big.NewInt(1e18),
			Store: map[common.Hash]common.Hash{
				slot2: val2,
			},
		}).
		Build()
	return NewPublicBlockChainAPI(be)
}

func TestGetStorageValues_ReturnsRequestedValues(t *testing.T) {
	maxSlots := make([]common.Hash, maxGetStorageSlots)
	maxSlotsWant := make([]common.Hash, maxGetStorageSlots)
	for i := range maxSlots {
		maxSlots[i] = common.BigToHash(big.NewInt(int64(i)))
	}
	maxSlotsWant[0] = val0
	maxSlotsWant[1] = val1

	tests := map[string]struct {
		requests map[common.Address][]common.Hash
		want     map[common.Address][]common.Hash
	}{
		"single address, single slot": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {slot0},
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: {val0},
			},
		},
		"multiple addresses, multiple slots": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {slot0, slot1},
				storageAddr2: {slot2},
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: {val0, val1},
				storageAddr2: {val2},
			},
		},
		"missing slot returns zero": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {common.HexToHash("0xff")},
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: {{}},
			},
		},
		"nonexistent account returns zeros": {
			requests: map[common.Address][]common.Hash{
				unknownAddr: {slot0, slot1},
			},
			want: map[common.Address][]common.Hash{
				unknownAddr: {{}, {}},
			},
		},
		"same slot requested for different accounts": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {slot2},
				storageAddr2: {slot2},
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: {{}},
				storageAddr2: {val2},
			},
		},
		"duplicated slot in one request": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {slot0, slot0},
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: {val0, val0},
			},
		},
		"empty key list next to non-empty one": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {},
				storageAddr2: {slot2},
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: {},
				storageAddr2: {val2},
			},
		},
		"exactly the maximum number of slots is accepted": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: maxSlots,
			},
			want: map[common.Address][]common.Hash{
				storageAddr1: maxSlotsWant,
			},
		},
	}

	api := newStorageValuesAPI(t)
	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := api.GetStorageValues(t.Context(), test.requests, latest)
			require.NoError(t, err)
			require.Len(t, result, len(test.want))
			for addr, wantVals := range test.want {
				require.Contains(t, result, addr)
				require.Len(t, result[addr], len(wantVals))
				for i, want := range wantVals {
					// values are always full 32-byte words
					require.Len(t, []byte(result[addr][i]), common.HashLength)
					require.Equal(t, want, common.BytesToHash(result[addr][i]))
				}
			}
		})
	}
}

func TestGetStorageValues_SupportsBlockTagsAndHashes(t *testing.T) {
	// the default block history of the fake backend contains
	// a single block with number 1 and hash 0x1
	tests := map[string]rpc.BlockNumberOrHash{
		"latest":    rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
		"pending":   rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber),
		"earliest":  rpc.BlockNumberOrHashWithNumber(rpc.EarliestBlockNumber),
		"finalized": rpc.BlockNumberOrHashWithNumber(rpc.FinalizedBlockNumber),
		"safe":      rpc.BlockNumberOrHashWithNumber(rpc.SafeBlockNumber),
		"number":    rpc.BlockNumberOrHashWithNumber(1),
		"hash":      rpc.BlockNumberOrHashWithHash(common.HexToHash("0x1"), false),
	}

	api := newStorageValuesAPI(t)

	for name, block := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := api.GetStorageValues(t.Context(), map[common.Address][]common.Hash{
				storageAddr1: {slot0},
			}, block)
			require.NoError(t, err)
			require.Equal(t, val0, common.BytesToHash(result[storageAddr1][0]))
		})
	}
}

func TestGetStorageValues_InvalidRequestsAreRejected(t *testing.T) {
	tooManySlots := make([]common.Hash, maxGetStorageSlots+1)
	for i := range tooManySlots {
		tooManySlots[i] = common.BigToHash(big.NewInt(int64(i)))
	}
	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	tests := map[string]struct {
		requests    map[common.Address][]common.Hash
		block       rpc.BlockNumberOrHash
		wantCode    int
		wantMessage string
	}{
		"nil request": {
			requests:    nil,
			block:       latest,
			wantCode:    -32602,
			wantMessage: "empty request",
		},
		"empty request": {
			requests:    map[common.Address][]common.Hash{},
			block:       latest,
			wantCode:    -32602,
			wantMessage: "empty request",
		},
		"addresses without any slots": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: {},
				storageAddr2: nil,
			},
			block:       latest,
			wantCode:    -32602,
			wantMessage: "empty request",
		},
		"too many slots for one address": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: tooManySlots,
			},
			block:       latest,
			wantCode:    -38026,
			wantMessage: fmt.Sprintf("too many slots (max %d)", maxGetStorageSlots),
		},
		"too many slots across addresses": {
			requests: map[common.Address][]common.Hash{
				storageAddr1: tooManySlots[:maxGetStorageSlots],
				storageAddr2: {slot0},
			},
			block:       latest,
			wantCode:    -38026,
			wantMessage: fmt.Sprintf("too many slots (max %d)", maxGetStorageSlots),
		},
	}

	api := newStorageValuesAPI(t)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := api.GetStorageValues(t.Context(), test.requests, test.block)
			require.Nil(t, result)
			require.Error(t, err)
			require.Equal(t, test.wantMessage, err.Error())
			var coded rpc.Error
			require.ErrorAs(t, err, &coded)
			require.Equal(t, test.wantCode, coded.ErrorCode())
		})
	}
}

func TestGetStorageValues_UnknownBlocksAreRejected(t *testing.T) {
	tests := map[string]rpc.BlockNumberOrHash{
		"unknown block number": rpc.BlockNumberOrHashWithNumber(9999),
		"unknown block hash":   rpc.BlockNumberOrHashWithHash(common.HexToHash("0xbeef"), false),
	}

	api := newStorageValuesAPI(t)

	for name, block := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := api.GetStorageValues(t.Context(), map[common.Address][]common.Hash{
				storageAddr1: {slot0},
			}, block)
			require.Nil(t, result)
			require.Error(t, err)
		})
	}
}

func TestGetStorageValues_OversizedRequestIsRejectedWithoutStateAccess(t *testing.T) {
	// The slot limit must be enforced before the state is resolved, so an
	// oversized request cannot cause any state work. Requesting an unknown
	// block must still fail with the limit error, not a block lookup error.
	tooManySlots := make([]common.Hash, maxGetStorageSlots+1)
	for i := range tooManySlots {
		tooManySlots[i] = common.BigToHash(big.NewInt(int64(i)))
	}

	api := newStorageValuesAPI(t)

	_, err := api.GetStorageValues(t.Context(), map[common.Address][]common.Hash{
		storageAddr1: tooManySlots,
	}, rpc.BlockNumberOrHashWithNumber(9999))
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("too many slots (max %d)", maxGetStorageSlots), err.Error())
}

func TestGetStorageValues_ResultMarshalsToPaddedHexWords(t *testing.T) {
	api := newStorageValuesAPI(t)

	result, err := api.GetStorageValues(t.Context(), map[common.Address][]common.Hash{
		storageAddr1: {slot0, common.HexToHash("0xff")},
	}, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	require.NoError(t, err)

	encoded, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded map[common.Address][]string
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Len(t, decoded[storageAddr1], 2)
	for _, value := range decoded[storageAddr1] {
		// every value is a 0x-prefixed 32-byte hex string
		require.Regexp(t, regexp.MustCompile("^0x[0-9a-f]{64}$"), value)
	}
}
