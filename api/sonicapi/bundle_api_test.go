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
		wantEarliest  uint64
		wantLatest    uint64
		errorContains string
	}{
		"nil both defaults from current block": {
			currentBlock: 10,
			wantEarliest: 11,
			wantLatest:   10 + bundle.MaxBlockRange,
		},
		"only earliest": {
			currentBlock: 10,
			blockRange:   &RPCRange{Earliest: hexN(50)},
			wantEarliest: 50,
			wantLatest:   50 + bundle.MaxBlockRange - 1,
		},
		"explicit latest": {
			currentBlock: 10,
			blockRange:   &RPCRange{Latest: hexN(200)},
			wantEarliest: 11,
			wantLatest:   200,
		},
		"range exceeds MaxBlockRange when only latest set": {
			currentBlock:  10,
			blockRange:    &RPCRange{Latest: hexN(10 + bundle.MaxBlockRange + 100)},
			errorContains: "invalid block range",
		},
		"both explicit": {
			currentBlock: 10,
			blockRange:   &RPCRange{Earliest: hexN(5), Latest: hexN(20)},
			wantEarliest: 5,
			wantLatest:   20,
		},
		"current block zero earliest is one": {
			currentBlock: 0,
			wantEarliest: 1,
			wantLatest:   bundle.MaxBlockRange,
		},
		"latest is less than earliest": {
			currentBlock:  100,
			blockRange:    &RPCRange{Earliest: hexN(50), Latest: hexN(40)},
			errorContains: "invalid block range",
		},
		"latest before implicit earliest from current block": {
			currentBlock:  10,
			blockRange:    &RPCRange{Latest: hexN(5)},
			errorContains: "invalid block range",
		},
		"greater than Max block range": {
			currentBlock:  100,
			blockRange:    &RPCRange{Earliest: hexN(50), Latest: hexN(50 + bundle.MaxBlockRange + 1)},
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
				require.EqualValues(t, tc.wantEarliest, r.Earliest)
				require.EqualValues(t, tc.wantLatest, r.Latest)
			}
		})
	}
}
