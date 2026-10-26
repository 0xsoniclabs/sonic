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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// fixture is an oracle over a FakeCore whose open epoch has the given
// minimum gas price and whose validators have the given gas power left.
type fixture struct {
	core   *tests.FakeCore
	oracle *oracle.Oracle
	rules  opera.Rules
}

func newFixture(t *testing.T, config oracle.Config, gasPowerLeft uint64) fixture {
	t.Helper()
	fake := tests.NewFakeCore(t)
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	rules.Economy.MinGasPrice = big.NewInt(1000)
	fake.SetOracleInputs(core.OracleInputs{PendingRules: rules, TotalGasPowerLeft: gasPowerLeft, MinTip: big.NewInt(0)})
	openRules := func(context.Context) (opera.Rules, error) { return rules, nil }
	return fixture{core: fake, oracle: oracle.New(config, fake, openRules), rules: rules}
}

// fullGasPower is the gas power of the fixture's validators with nothing
// spent.
func fullGasPower(rules opera.Rules) uint64 {
	return rules.Economy.LongGasPower.AllocPerSec * uint64(rules.Economy.LongGasPower.MaxAllocPeriod) / uint64(time.Second)
}

// TestSuggestTip_KeepsAMarginWithAllGasPowerLeft: with an empty pool and all
// gas power left, the default certainty of one half spends 2.5% of the gas
// power, which asks for 2.5% above the minimum price.
func TestSuggestTip_KeepsAMarginWithAllGasPowerLeft(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	fixture := newFixture(t, oracle.DefaultConfig(), fullGasPower(rules))
	tip, err := fixture.oracle.SuggestTip(context.Background(), oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(25), tip)
}

func TestSuggestTip_RisesAsTheGasPowerRunsOut(t *testing.T) {
	fixture := newFixture(t, oracle.DefaultConfig(), 0)
	tip, err := fixture.oracle.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(24*1000), tip, "with no gas power left the price is 25 times the minimum")
}

func TestSuggestTip_IsCappedByTheMaximumPrice(t *testing.T) {
	config := oracle.DefaultConfig()
	config.MaxGasPrice = big.NewInt(5000)
	fixture := newFixture(t, config, 0)
	tip, err := fixture.oracle.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(4000), tip)
}

// TestSuggestTip_IsFlooredByTheMinimumPrice holds ADR9: the GPO key
// MinGasPrice keeps its effect.
func TestSuggestTip_IsFlooredByTheMinimumPrice(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	config := oracle.DefaultConfig()
	config.MinGasPrice = big.NewInt(7000)
	fixture := newFixture(t, config, fullGasPower(rules))
	tip, err := fixture.oracle.SuggestTip(context.Background(), oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(6000), tip)
}

func TestSuggestTip_IsAtLeastTheMinimumTipOfThePool(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	fixture := newFixture(t, oracle.DefaultConfig(), fullGasPower(rules))
	fixture.core.SetOracleInputs(core.OracleInputs{PendingRules: fixture.rules, TotalGasPowerLeft: fullGasPower(rules), MinTip: big.NewInt(42)})
	tip, err := fixture.oracle.SuggestTip(context.Background(), oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(42), tip)
}

// TestSuggestTip_IsCachedForAPeriod holds I10: the walk over the last events
// behind OracleInputs is not repeated for every request.
func TestSuggestTip_IsCachedForAPeriod(t *testing.T) {
	fixture := newFixture(t, oracle.DefaultConfig(), 0)
	first, err := fixture.oracle.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)
	before := fixture.core.Calls("OracleInputs")
	again, err := fixture.oracle.SuggestTip(context.Background(), oracle.DecimalUnit+1)
	require.NoError(t, err)
	require.Equal(t, first, again)
	require.Equal(t, before, fixture.core.Calls("OracleInputs"), "a certainty of the same band is answered from the cache")
	again.SetInt64(-1)
	cached, err := fixture.oracle.SuggestTip(context.Background(), oracle.DecimalUnit)
	require.NoError(t, err)
	require.Equal(t, first, cached, "the cache hands out copies")
}

func TestSuggestTip_FailsWhenTheRulesCannotBeRead(t *testing.T) {
	failure := errors.New("no rules")
	broken := oracle.New(oracle.DefaultConfig(), tests.NewFakeCore(t), func(context.Context) (opera.Rules, error) {
		return opera.Rules{}, failure
	})
	_, err := broken.SuggestTip(context.Background(), oracle.DefaultCertainty)
	require.ErrorIs(t, err, failure)
}

func TestNew_SanitizesTheConfiguration(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	fixture := newFixture(t, oracle.Config{DefaultCertainty: 2 * oracle.DecimalUnit}, fullGasPower(rules))
	tip, err := fixture.oracle.SuggestTip(context.Background(), oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(45), tip, "a missing maximum and minimum take their defaults, a certainty above one is one")
	require.Equal(t, uint64(0.5*oracle.DecimalUnit), oracle.DefaultConfig().DefaultCertainty)
}

// TestSample_ReadsThePoolAndTheRulesButNotTheGasPower holds I10: a tick of
// the sampler does not call OracleInputs.
func TestSample_ReadsThePoolAndTheRulesButNotTheGasPower(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	fixture := newFixture(t, oracle.DefaultConfig(), fullGasPower(rules))
	signer := tests.NewSigner(t)
	recipient := common.HexToAddress("0x1")
	var pending types.Transactions
	for nonce := range 5 {
		pending = append(pending, signer.Sign(t, &types.LegacyTx{Nonce: uint64(nonce), To: &recipient, Gas: 10_000_000, GasPrice: big.NewInt(1_000_000)}))
	}
	other := tests.NewSigner(t)
	fixture.core.SetPool(map[common.Address]types.Transactions{signer.Address: pending, other.Address: {other.Sign(t, &types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(500)})}}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- fixture.oracle.Sample(ctx) }()
	tests.Eventually(t, func() bool { return fixture.core.Calls("PendingHeads") > 0 }, "the first sample")
	cancel()
	require.NoError(t, <-done)
	require.Zero(t, fixture.core.Calls("OracleInputs"), "a sample walks no validator's events")

	tip, err := fixture.oracle.SuggestTip(context.Background(), 0)
	require.NoError(t, err)
	require.Positive(t, tip.Sign(), "the reactive price of an uncertain suggestion follows the pool")
}

func TestSample_StopsWhenTheRulesCannotBeRead(t *testing.T) {
	failure := errors.New("no rules")
	broken := oracle.New(oracle.DefaultConfig(), tests.NewFakeCore(t), func(context.Context) (opera.Rules, error) {
		return opera.Rules{}, failure
	})
	require.ErrorIs(t, broken.Sample(context.Background()), failure)
}
