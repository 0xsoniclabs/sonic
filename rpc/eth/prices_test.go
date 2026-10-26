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

	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestGasPrice_IsTheBaseFeeOfTheHeadWithAMargin(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.AppendAndPublish(tests.BlockSpec{BaseFee: big.NewInt(1000)})
	harness := newEth(t, fake)
	harness.WaitForHead(1)
	require.JSONEq(t, `"0x44c"`, string(harness.Result("eth_gasPrice")))
}

func TestMaxPriorityFeePerGas_IsTheTipOfTheOracle(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	fake.SetOracleInputs(core.OracleInputs{PendingRules: rules, MinTip: big.NewInt(1_000_000_000_000_000_000)})
	require.JSONEq(t, `"0xde0b6b3a7640000"`, string(harness.Result("eth_maxPriorityFeePerGas")), "the minimum tip of the pool floors the tip")
}

func TestBlobBaseFee_IsTheMinimum(t *testing.T) {
	harness := newEth(t, tests.NewFakeCore(t))
	require.JSONEq(t, `"0x1"`, string(harness.Result("eth_blobBaseFee")))
}

func TestFeeHistory_RepeatsTheBaseFeeAndTheTips(t *testing.T) {
	fake := tests.NewFakeCore(t)
	for range 3 {
		fake.AppendBlock(tests.BlockSpec{BaseFee: big.NewInt(1000)})
	}
	fake.PublishHead(3)
	harness := newEth(t, fake)
	answer := decodeMap(t, harness.Result("eth_feeHistory", "0x2", "latest", []float64{10, 90}))
	require.Equal(t, "0x2", answer["oldestBlock"])
	require.Equal(t, []any{"0x44c", "0x44c", "0x44c"}, answer["baseFeePerGas"])
	require.Equal(t, []any{0.99, 0.99}, answer["gasUsedRatio"])
	require.Equal(t, []any{"0x1", "0x1", "0x1"}, answer["baseFeePerBlobGas"])
	require.Len(t, answer["reward"], 2)
	require.Len(t, answer["reward"].([]any)[0], 2)

	answer = decodeMap(t, harness.Result("eth_feeHistory", 10, "0x1", []float64{}))
	require.Equal(t, "0x0", answer["oldestBlock"], "the history starts at genesis")
	require.Len(t, answer["gasUsedRatio"], 2)
	require.Equal(t, []any{[]any{}, []any{}}, answer["reward"], "no percentiles, an empty reward per block")

	answer = decodeMap(t, harness.Result("eth_feeHistory", "0x800", "latest", nil))
	require.Len(t, answer["gasUsedRatio"], 4, "a count above 1024 is clamped, and the chain is shorter")
}

func TestFeeHistory_OfNoBlockIsEmpty(t *testing.T) {
	harness := newEth(t, tests.NewFakeCore(t))
	require.JSONEq(t, `{"oldestBlock":"0x0","gasUsedRatio":null}`, string(harness.Result("eth_feeHistory", "0x0", "latest", nil)))
}

func TestFeeHistory_ChecksItsArguments(t *testing.T) {
	harness := newEth(t, tests.NewFakeCore(t))
	tooMany := make([]float64, 101)
	cases := map[string]struct {
		parameters []any
		message    string
	}{
		"too many percentiles":   {[]any{"0x1", "latest", tooMany}, "invalid reward percentile: over the query limit 100"},
		"negative percentile":    {[]any{"0x1", "latest", []float64{-1}}, "invalid reward percentile: -1.000000"},
		"percentile above 100":   {[]any{"0x1", "latest", []float64{101}}, "invalid reward percentile: 101.000000"},
		"unordered percentiles":  {[]any{"0x1", "latest", []float64{50, 10}}, "invalid reward percentile: #0:50.000000 > #1:10.000000"},
		"a block above the head": {[]any{"0x1", "0x5", nil}, "block 5 not found; latest block is 0"},
	}
	for name, test := range cases {
		require.Equal(t, test.message, harness.Failure("eth_feeHistory", test.parameters...).Message, name)
	}
	require.Equal(t, "missing value for required argument 2", harness.Failure("eth_feeHistory", "0x1", "latest").Message)
}
