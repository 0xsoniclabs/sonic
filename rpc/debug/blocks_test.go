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

package debug_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/debug"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func newHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.SetHead(1)
	fake.AddBlock() // stored, not served
	return tests.NewHarness(t, fake, func(h *tests.Harness) { debug.Register(h.Dispatcher, h.Environment) })
}

func TestBlocks_AnswerTheBlockOfANumber(t *testing.T) {
	h := newHarness(t)
	block, err := h.Core.Block(1)
	require.NoError(t, err)
	// The block of the node is not RLP-serializable, so the reference answers
	// a found block with the encoder's error.
	_, encodingErr := rlp.EncodeToBytes(block)
	require.Error(t, encodingErr)
	cases := map[string]struct {
		number uint64
		found  bool
	}{
		"number":                    {1, true},
		"above the head":            {2, false},
		"cast to latest":            {math.MaxUint64 - 1, true},
		"cast to a tag of no block": {math.MaxUint64 - 9, false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if !test.found {
				notFound := "block #" + strconv.FormatUint(test.number, 10) + " not found"
				require.Equal(t, notFound, h.Fail(t, "debug_getBlockRlp", test.number).Message)
				require.Equal(t, notFound, h.Fail(t, "debug_printBlock", test.number).Message)
				return
			}
			require.Equal(t, encodingErr.Error(), h.Fail(t, "debug_getBlockRlp", test.number).Message)
			require.Contains(t, h.Result(t, "debug_printBlock", test.number), "EvmHeader")
		})
	}
}

// debug_printBlock passes a failed lookup on, debug_getBlockRlp takes it for
// a missing block, as their legacy handlers do.
func TestBlocks_AnswerAFailedLookup(t *testing.T) {
	h := newHarness(t)
	h.Core.SetMode("Block", tests.Fail)
	require.Equal(t, "block #1 not found", h.Fail(t, "debug_getBlockRlp", 1).Message)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "debug_printBlock", 1).Message)
}
