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

package eth_test

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func newGasHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.AddBlock()
	fake.SetHead(2)
	rules := fake.OracleInputs().NextEpochRules
	fake.SetOracleInputs(core.OracleInputs{NextEpochRules: rules, MinTip: big.NewInt(7)})
	return newHarness(t, fake)
}

func TestGasPrices_AnswerTheSuggestions(t *testing.T) {
	h := newGasHarness(t)
	// The head's base fee of 1 gwei with a margin of 10%.
	require.Equal(t, `"0x4190ab00"`, h.Result(t, "eth_gasPrice"))
	require.Equal(t, `"0x1"`, h.Result(t, "eth_blobBaseFee"))
	var tip hexutil.Big
	h.Call(t, &tip, "eth_maxPriorityFeePerGas")
	require.GreaterOrEqual(t, tip.ToInt().Int64(), int64(7), "the tip is at least the pool's smallest")
}

func TestGasPrices_PassOnAFailureOfTheRules(t *testing.T) {
	h := newGasHarness(t)
	h.Core.SetMode("Rules", tests.Fail)
	require.Contains(t, h.Fail(t, "eth_maxPriorityFeePerGas").Message, tests.ErrInjected.Error())
}

func TestFeeHistory_AnswersEveryBlockOfTheRange(t *testing.T) {
	h := newGasHarness(t)
	type history struct {
		OldestBlock  *hexutil.Big     `json:"oldestBlock"`
		Reward       [][]*hexutil.Big `json:"reward"`
		BaseFee      []*hexutil.Big   `json:"baseFeePerGas"`
		GasUsedRatio []float64        `json:"gasUsedRatio"`
	}
	cases := map[string]struct {
		count, last string
		oldest      int64
		blocks      int
	}{
		"one block":           {"0x1", "latest", 2, 1},
		"the whole chain":     {"0x3", "latest", 0, 3},
		"more than the chain": {"0x10", "0x1", 0, 2},
		"clamped count":       {"0x500", "latest", 0, 3},
		"decimal count":       {"2", "latest", 1, 2},
		"no block":            {"0x0", "latest", 0, 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			var result history
			h.Call(t, &result, "eth_feeHistory", test.count, test.last, []float64{10, 90})
			require.Equal(t, test.oldest, result.OldestBlock.ToInt().Int64())
			require.Len(t, result.GasUsedRatio, test.blocks)
			require.Len(t, result.Reward, test.blocks)
			if test.blocks > 0 {
				require.Len(t, result.BaseFee, test.blocks+1, "the base fees hold the next block too")
				require.Len(t, result.Reward[0], 2)
			}
		})
	}
}

func TestFeeHistory_RefusesWhatIsOutOfRange(t *testing.T) {
	h := newGasHarness(t)
	tooMany := make([]float64, 101)
	cases := map[string]struct {
		params []any
		err    *dispatch.Error
	}{
		"block above the head": {[]any{"0x1", "0x5", []float64{}},
			&dispatch.Error{Code: dispatch.CodeDefault, Message: "block 5 not found; latest block is 2"}},
		"percentile above 100": {[]any{"0x1", "latest", []float64{101}},
			&dispatch.Error{Code: dispatch.CodeDefault, Message: "invalid reward percentile: 101.000000"}},
		"decreasing percentiles": {[]any{"0x1", "latest", []float64{50, 10}},
			&dispatch.Error{Code: dispatch.CodeDefault, Message: "invalid reward percentile: #0:50.000000 > #1:10.000000"}},
		"too many percentiles": {[]any{"0x1", "latest", tooMany},
			&dispatch.Error{Code: dispatch.CodeDefault, Message: "invalid reward percentile: over the query limit 100"}},
		"percentiles required": {[]any{"0x1", "latest"},
			&dispatch.Error{Code: dispatch.CodeInvalidParams, Message: "missing value for required argument 2"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.err, h.Fail(t, "eth_feeHistory", test.params...))
		})
	}
}
