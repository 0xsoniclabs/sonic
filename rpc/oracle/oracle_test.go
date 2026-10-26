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

package oracle_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newOracle(t *testing.T, cfg oracle.Config) (*tests.FakeCore, *oracle.Oracle, *opera.Rules) {
	t.Helper()
	fc := tests.NewFakeCore()
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	fc.SetOracleInputs(core.OracleInputs{PendingRules: rules, TotalGasPowerLeft: 0, MinTip: new(big.Int)})
	return fc, oracle.New(cfg, fc, func() (*opera.Rules, error) { r := rules.Copy(); return &r, nil }), &rules
}

func TestSuggestTip_TheConfigurationBoundsTheSuggestion(t *testing.T) {
	_, o, rules := newOracle(t, oracle.DefaultConfig())
	tip, err := o.SuggestTip(context.Background(), oracle.AsDefaultCertainty)
	require.NoError(t, err)
	require.Positive(t, tip.Sign(), "no gas power left: the constructive price is above the minimum")

	// ADR9: the configured minimum gas price is the floor of a suggestion.
	floor := new(big.Int).Mul(rules.Economy.MinGasPrice, big.NewInt(1000))
	cfg := oracle.DefaultConfig()
	cfg.MinGasPrice = floor
	_, o, rules = newOracle(t, cfg)
	tip, err = o.SuggestTip(context.Background(), oracle.AsDefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Sub(floor, rules.Economy.MinGasPrice), tip)

	// The maximum gas price is the ceiling.
	cfg = oracle.DefaultConfig()
	cfg.MaxGasPrice = new(big.Int).Add(rules.Economy.MinGasPrice, big.NewInt(5))
	_, o, _ = newOracle(t, cfg)
	tip, err = o.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(5), tip)
}

func TestSuggestTip_TheMinimumTipIsTheFloor(t *testing.T) {
	fc, o, rules := newOracle(t, oracle.DefaultConfig())
	fc.SetOracleInputs(core.OracleInputs{PendingRules: *rules, TotalGasPowerLeft: 1 << 62, MinTip: big.NewInt(1 << 40)})
	tip, err := o.SuggestTip(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1<<40), tip)
}

func TestSuggestTip_IsCachedForASecond(t *testing.T) {
	fc, o, _ := newOracle(t, oracle.DefaultConfig())
	_, err := o.SuggestTip(context.Background(), oracle.AsDefaultCertainty)
	require.NoError(t, err)
	reads := fc.Calls("OracleInputs")
	_, err = o.SuggestTip(context.Background(), oracle.AsDefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, reads, fc.Calls("OracleInputs"))
	_, err = o.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)
	require.Equal(t, reads+1, fc.Calls("OracleInputs"), "another certainty bucket")
}

func TestSuggestTip_FailuresAndCancellation(t *testing.T) {
	fc := tests.NewFakeCore()
	o := oracle.New(oracle.DefaultConfig(), fc, func() (*opera.Rules, error) { return nil, tests.ErrFake })
	_, err := o.SuggestTip(context.Background(), 0)
	require.ErrorIs(t, err, tests.ErrFake)
	o = oracle.New(oracle.DefaultConfig(), fc, func() (*opera.Rules, error) { return nil, nil })
	_, err = o.SuggestTip(context.Background(), 0)
	require.EqualError(t, err, "no rules of the open epoch")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, o, _ = newOracle(t, oracle.DefaultConfig())
	_, err = o.SuggestTip(ctx, 0)
	require.ErrorIs(t, err, context.Canceled)

	// Inputs without pending rules are not a panic.
	fc, o, _ = newOracle(t, oracle.DefaultConfig())
	fc.SetOracleInputs(core.OracleInputs{})
	_, err = o.SuggestTip(context.Background(), 0)
	require.NoError(t, err)
}

// TestRun_ATickReadsThePoolAndNotTheGasPower is I10: a sample reads the heads
// of the pool and the open rules, never OracleInputs, and the samples raise
// the reactive part of a suggestion.
func TestRun_ATickReadsThePoolAndNotTheGasPower(t *testing.T) {
	fc, o, rules := newOracle(t, oracle.DefaultConfig())
	fc.SetOracleInputs(core.OracleInputs{PendingRules: *rules, TotalGasPowerLeft: 1 << 62, MinTip: new(big.Int)})
	quiet, err := o.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)

	pending := map[common.Address]types.Transactions{}
	for i := range 50 {
		w := tests.NewWallet(t)
		pending[w.Address] = types.Transactions{w.Sign(t, &types.DynamicFeeTx{
			ChainID: big.NewInt(tests.ChainID), Gas: 1_000_000,
			GasFeeCap: new(big.Int).Mul(rules.Economy.MinGasPrice, big.NewInt(100)),
			GasTipCap: new(big.Int).Mul(rules.Economy.MinGasPrice, big.NewInt(int64(10+i))),
		})}
	}
	fc.SetPool(pending, nil)
	reads := fc.Calls("OracleInputs")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); o.Run(ctx) }()
	tests.WaitFor(t, func() bool { return fc.Calls("PendingHeads") >= 2 })
	cancel()
	<-done
	require.Equal(t, reads, fc.Calls("OracleInputs"))

	busy, err := o.SuggestTip(context.Background(), oracle.DecimalUnit-1)
	require.NoError(t, err)
	require.Positive(t, busy.Cmp(quiet), "the samples raise the suggestion")
}

func TestNew_SanitizesTheConfiguration(t *testing.T) {
	cfg := oracle.Config{MaxGasPrice: nil, MinGasPrice: nil, DefaultCertainty: 2 * oracle.DecimalUnit}
	fc := tests.NewFakeCore()
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	o := oracle.New(cfg, fc, func() (*opera.Rules, error) { return &rules, nil })
	tip, err := o.SuggestTip(context.Background(), oracle.AsDefaultCertainty)
	require.NoError(t, err)
	require.NotNil(t, tip)
}
