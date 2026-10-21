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

package oracle

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// node is a fixed set of inputs of the oracle.
type node struct {
	inputs      core.OracleInputs
	heads       []core.PendingHead
	inputsCalls int
}

func (n *node) OracleInputs() core.OracleInputs {
	n.inputsCalls++
	return n.inputs
}
func (n *node) PendingHeads(int) []core.PendingHead { return n.heads }
func (n *node) Progress() core.Progress             { return core.Progress{} }
func (n *node) Rules(uint64) (*opera.Rules, error) {
	rules := n.inputs.Rules
	return &rules, nil
}

func newNode() *node {
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	long := rules.Economy.LongGasPower
	return &node{inputs: core.OracleInputs{
		Rules: rules, PendingRules: rules, MinTip: new(big.Int),
		TotalGasPowerLeft: new(big.Int).Div(
			new(big.Int).Mul(new(big.Int).SetUint64(long.AllocPerSec), new(big.Int).SetUint64(uint64(long.MaxAllocPeriod))),
			big.NewInt(int64(time.Second))).Uint64(),
	}}
}

func tx(tip, feeCap int64, gas uint64) core.PendingHead {
	return core.PendingHead{Tx: types.NewTx(&types.DynamicFeeTx{
		ChainID: big.NewInt(1), GasTipCap: big.NewInt(tip), GasFeeCap: big.NewInt(feeCap), Gas: gas,
		To: &common.Address{1},
	})}
}

// Golden tips of the oracle the node had before the module took over: the
// answers of its constructive price at five levels of free gas power and five
// certainties, as measured on the node's own implementation (release 2.2.4).
var goldenTips = [5][5]string{
	{"24000000000", "24000000000", "24000000000", "24000000000", "24000000000"},
	{"18933344000", "19146672000", "20000000000", "20853344000", "21066672000"},
	{"2881250000", "2986250000", "3406250000", "3826250000", "3931250000"},
	{"214999000", "227000000", "275000000", "322999000", "335000000"},
	{"5000000", "9000000", "25000000", "41000000", "45000000"},
}

func TestSuggestTip_MatchesTheNodesFormerOracle(t *testing.T) {
	n := newNode()
	maxGas := n.inputs.TotalGasPowerLeft
	require.EqualValues(t, 150_000_000, maxGas)
	for li, left := range []uint64{0, maxGas / 10, maxGas / 2, maxGas * 9 / 10, maxGas} {
		for ci, certainty := range []uint64{0, 100_000, 500_000, 900_000, 1_000_000} {
			n.inputs.TotalGasPowerLeft = left
			want, _ := new(big.Int).SetString(goldenTips[li][ci], 10)
			require.Equal(t, want, New(Config{}, n).SuggestTip(certainty), "left=%d certainty=%d", left, certainty)
		}
	}
	// ... and for the default certainty.
	n.inputs.TotalGasPowerLeft = maxGas
	require.Equal(t, big.NewInt(25_000_000), New(Config{}, n).SuggestTip(AsDefaultCertainty))
}

func TestSuggestTip_MinTipIsAFloor(t *testing.T) {
	n := newNode()
	n.inputs.MinTip = big.NewInt(12345678901)
	require.Equal(t, big.NewInt(12345678901), New(Config{}, n).SuggestTip(0))
}

func TestSuggestTip_MinGasPriceIsAFloor(t *testing.T) {
	n := newNode() // all gas power is free: the lowest tip
	require.Equal(t, big.NewInt(5_000_000), New(Config{}, n).SuggestTip(0))
	floor := new(big.Int).Add(orZero(n.inputs.Rules.Economy.MinGasPrice), big.NewInt(7_000_000_000))
	require.Equal(t, big.NewInt(7_000_000_000), New(Config{MinGasPrice: floor}, n).SuggestTip(0))
}

func TestTick_ReadsOnlyTheRulesOfTheNode(t *testing.T) {
	n := newNode()
	n.heads = []core.PendingHead{tx(1, 2, 21000)}
	o := New(Config{}, n)
	o.tick()
	require.Zero(t, n.inputsCalls, "the pool is sampled without the full oracle inputs")
}

func TestSuggestTip_ReactsToThePool(t *testing.T) {
	n := newNode()
	min := n.inputs.Rules.Economy.MinGasPrice.Int64()
	o := New(Config{}, n)
	quiet := o.SuggestTip(0)

	// Many pending transactions that pay well above the minimal price: the
	// price percentiles rise after the oracle sampled the pool.
	for range 200 {
		n.heads = append(n.heads, tx(min*50, min*60, 5_000_000))
	}
	o.tick()
	o.tCache.Purge()
	busy := o.SuggestTip(0)
	require.Greater(t, busy.Cmp(quiet), 0, "quiet %v busy %v", quiet, busy)

	// A tip never exceeds what the operator allows.
	capped := New(Config{MaxGasPrice: big.NewInt(min * 2)}, n)
	capped.tick()
	require.LessOrEqual(t, new(big.Int).Add(capped.SuggestTip(0), big.NewInt(min)).Cmp(big.NewInt(min*2)), 0)
}

func TestSuggestTip_IsCachedForAMoment(t *testing.T) {
	n := newNode()
	o := New(Config{}, n)
	first := o.SuggestTip(0)
	n.inputs.TotalGasPowerLeft = 0
	require.Equal(t, first, o.SuggestTip(0))
	// The copy returned is the caller's.
	first.SetInt64(7)
	require.NotEqual(t, int64(7), o.SuggestTip(0).Int64())
}

func TestSuggestTip_ToleratesMissingNumbers(t *testing.T) {
	n := &node{inputs: core.OracleInputs{}}
	o := New(Config{}, n)
	require.NotPanics(t, func() { _ = o.SuggestTip(0) })
	require.NotPanics(t, func() { o.tick() })
	n.heads = []core.PendingHead{{Tx: nil}, tx(1, 2, 21000)}
	require.NotPanics(t, func() { o.tick() })
	require.NotPanics(t, func() { _ = New(Config{}, nil).SuggestTip(0) })
}

func TestConfigIsSanitized(t *testing.T) {
	o := New(Config{MaxGasPrice: big.NewInt(0), DefaultCertainty: 5_000_000}, newNode())
	require.Equal(t, DefaultMaxGasPrice(), o.cfg.MaxGasPrice)
	require.EqualValues(t, DecimalUnit, o.cfg.DefaultCertainty)
	o = New(Config{}, newNode())
	require.EqualValues(t, DecimalUnit/2, o.cfg.DefaultCertainty)
}

func TestRun_SamplesUntilTheContextEnds(t *testing.T) {
	n := newNode()
	o := New(Config{}, n)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { o.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
}

func TestSuggestedGasPrice_MatchesTheNodesFunction(t *testing.T) {
	for _, fee := range []*big.Int{nil, big.NewInt(0), big.NewInt(1), big.NewInt(1000), big.NewInt(1e18), new(big.Int).Lsh(big.NewInt(1), 200)} {
		require.Equal(t, gaspricelimits.GetSuggestedGasPriceForNewTransactions(fee), SuggestedGasPrice(fee), "%v", fee)
	}
}
