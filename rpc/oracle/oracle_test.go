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
	"time"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func newOracle(t *testing.T, config oracle.Config) (*oracle.Oracle, *tests.FakeCore, *big.Int) {
	t.Helper()
	fake := tests.NewFakeCore()
	rules, err := fake.Rules(1)
	require.NoError(t, err)
	fake.SetOracleInputs(core.OracleInputs{NextEpochRules: *rules, MinTip: new(big.Int)})
	return oracle.New(config, fake, func() uint64 { return 0 }), fake, rules.Economy.MinGasPrice
}

// Without gas power left and without a sample, the constructive price is 25
// times the minimum gas price, so the tip is 24 times it.
func TestSuggestTip_PricesTheGasPowerLeft(t *testing.T) {
	suggester, _, minPrice := newOracle(t, oracle.DefaultConfig())
	tip, err := suggester.SuggestTip(oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Mul(minPrice, big.NewInt(24)), tip)
}

func TestSuggestTip_IsBoundedByTheConfigurationAndThePool(t *testing.T) {
	cases := map[string]struct {
		config func(minPrice *big.Int) oracle.Config
		minTip int64
		want   func(minPrice *big.Int) *big.Int
	}{
		"capped by the maximum price": {
			config: func(minPrice *big.Int) oracle.Config {
				return oracle.Config{MaxGasPrice: new(big.Int).Mul(minPrice, big.NewInt(2)), DefaultCertainty: oracle.DecimalUnit / 2}
			},
			want: func(minPrice *big.Int) *big.Int { return minPrice },
		},
		"floored by the minimum price": {
			config: func(minPrice *big.Int) oracle.Config {
				return oracle.Config{MinGasPrice: new(big.Int).Mul(minPrice, big.NewInt(100)), DefaultCertainty: oracle.DecimalUnit / 2}
			},
			want: func(minPrice *big.Int) *big.Int { return new(big.Int).Mul(minPrice, big.NewInt(99)) },
		},
		"at least the pool's smallest tip": {
			config: func(minPrice *big.Int) oracle.Config { return oracle.DefaultConfig() },
			minTip: 1 << 62,
			want:   func(*big.Int) *big.Int { return big.NewInt(1 << 62) },
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			fake := tests.NewFakeCore()
			rules, err := fake.Rules(1)
			require.NoError(t, err)
			minPrice := rules.Economy.MinGasPrice
			fake.SetOracleInputs(core.OracleInputs{NextEpochRules: *rules, MinTip: big.NewInt(test.minTip)})
			suggester := oracle.New(test.config(minPrice), fake, func() uint64 { return 0 })
			tip, err := suggester.SuggestTip(oracle.DefaultCertainty)
			require.NoError(t, err)
			require.Equal(t, test.want(minPrice), tip)
		})
	}
}

func TestSuggestTip_KeepsASuggestionForASecond(t *testing.T) {
	suggester, fake, _ := newOracle(t, oracle.DefaultConfig())
	first, err := suggester.SuggestTip(oracle.DecimalUnit / 2)
	require.NoError(t, err)
	reads := fake.Calls("OracleInputs")
	second, err := suggester.SuggestTip(oracle.DecimalUnit/2 + 1)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, reads, fake.Calls("OracleInputs"), "the cached suggestion reads nothing")
}

func TestSuggestTip_FailsWithoutTheCurrentRules(t *testing.T) {
	suggester, fake, _ := newOracle(t, oracle.DefaultConfig())
	fake.SetMode("Rules", tests.Fail)
	_, err := suggester.SuggestTip(oracle.DefaultCertainty)
	require.ErrorIs(t, err, tests.ErrInjected)
}

// A sample of the pool raises the suggestion for a pool whose bids are
// higher, and reads the pool and the rules, not the gas power.
func TestSample_ReadsThePoolNotTheGasPower(t *testing.T) {
	suggester, fake, minPrice := newOracle(t, oracle.Config{MaxGasPrice: new(big.Int).Lsh(big.NewInt(1), 80), DefaultCertainty: oracle.DecimalUnit})
	account := tests.NewAccount()
	tip := new(big.Int).Mul(minPrice, big.NewInt(1000))
	for nonce := range uint64(4) {
		fake.AddPoolTransaction(account.Sign(&types.DynamicFeeTx{
			ChainID: big.NewInt(tests.ChainID), Nonce: nonce, Gas: 10000000,
			GasFeeCap: new(big.Int).Add(tip, minPrice), GasTipCap: tip,
		}))
	}
	before, err := suggester.SuggestTip(oracle.DecimalUnit)
	require.NoError(t, err)
	reads := fake.Calls("OracleInputs")
	suggester.SampleOnce()
	require.Equal(t, reads, fake.Calls("OracleInputs"))
	time.Sleep(time.Second) // past the cached suggestion
	after, err := suggester.SuggestTip(oracle.DecimalUnit)
	require.NoError(t, err)
	require.Equal(t, 1, after.Cmp(before), "the pool's bids raise the suggestion")
}

func TestSample_SkipsWithoutTheCurrentRules(t *testing.T) {
	suggester, fake, _ := newOracle(t, oracle.DefaultConfig())
	fake.SetMode("Rules", tests.Fail)
	pool := fake.Calls("PoolNextTransactions")
	suggester.SampleOnce()
	require.Equal(t, pool, fake.Calls("PoolNextTransactions"))
}

func TestSample_RunsUntilTheContextEnds(t *testing.T) {
	suggester, fake, _ := newOracle(t, oracle.DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		suggester.Sample(ctx)
		close(done)
	}()
	require.Eventually(t, func() bool { return fake.Calls("PoolNextTransactions") > 0 }, 3*time.Second, 10*time.Millisecond)
	cancel()
	<-done
}

func TestNew_SanitizesTheConfiguration(t *testing.T) {
	suggester, _, minPrice := newOracle(t, oracle.Config{DefaultCertainty: 2 * oracle.DecimalUnit})
	tip, err := suggester.SuggestTip(oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Mul(minPrice, big.NewInt(24)), tip, "the defaults stand in for missing bounds")
}
