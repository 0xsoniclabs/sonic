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

package sonicapi

import (
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func Test_sanitizeBlockRange(t *testing.T) {
	hexN := func(n uint64) hexutil.Uint64 { b := hexutil.Uint64(n); return b }

	tests := map[string]struct {
		currentBlock  uint64
		blockRange    *RPCRange
		wantFirst     uint64
		wantLength    uint64
		errorContains string
	}{
		"nil both defaults from current block": {
			currentBlock: 10,
			wantFirst:    11,
			wantLength:   bundle.MaxBlockRangeLength,
		},
		"only first": {
			currentBlock: 10,
			blockRange:   &RPCRange{First: hexN(50)},
			wantFirst:    50,
			wantLength:   bundle.MaxBlockRangeLength,
		},
		"explicit length": {
			currentBlock: 10,
			blockRange:   &RPCRange{Length: hexN(200)},
			wantFirst:    11,
			wantLength:   200,
		},
		"length exceeds MaxBlockRange": {
			currentBlock:  10,
			blockRange:    &RPCRange{Length: hexN(bundle.MaxBlockRangeLength + 1)},
			errorContains: "invalid block range",
		},
		"both explicit": {
			currentBlock: 10,
			blockRange:   &RPCRange{First: hexN(5), Length: hexN(20)},
			wantFirst:    5,
			wantLength:   20,
		},
		"current block zero earliest is one": {
			currentBlock: 0,
			wantFirst:    1,
			wantLength:   bundle.MaxBlockRangeLength,
		},
		"no overflow of first": {
			currentBlock: math.MaxUint64,
			wantFirst:    math.MaxUint64,
			wantLength:   bundle.MaxBlockRangeLength,
		},
		"specified range is in the past": {
			currentBlock:  10,
			blockRange:    &RPCRange{First: hexN(5), Length: hexN(3)},
			errorContains: "invalid block range",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := sanitizeBlockRange(tc.currentBlock, tc.blockRange)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, tc.wantFirst, r.First)
				require.EqualValues(t, tc.wantLength, r.Length)
			}
		})
	}
}
