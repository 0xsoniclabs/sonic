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

package chain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func startChain(t *testing.T, fake *tests.FakeCore, config chain.Config) (*chain.Chain, *prometheus.Registry) {
	t.Helper()
	registry := prometheus.NewRegistry()
	dispatcher := dispatch.New(dispatch.Config{RequestTimeout: time.Second}, registry)
	served := chain.New(config, fake, registry)
	require.NoError(t, served.Start(dispatcher))
	t.Cleanup(dispatcher.Stop)
	return served, registry
}

func waitForHead(t *testing.T, served *chain.Chain, number uint64) {
	t.Helper()
	require.Eventually(t, func() bool { return served.Head().NumberU64() == number }, time.Second, time.Millisecond)
}

func TestChain_ServesTheHeadOfTheSubscription(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.AddBlock()
	served, _ := startChain(t, fake, chain.Config{})
	require.Equal(t, uint64(0), served.Head().NumberU64())

	fake.SetHead(2)
	waitForHead(t, served, 2)
}

func TestChain_NeverGoesBack(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.AddBlock()
	served, _ := startChain(t, fake, chain.Config{})
	fake.SetHead(2)
	waitForHead(t, served, 2)
	fake.SetHead(1)
	fake.SetHead(2) // the feed delivers in order, so 1 was taken before
	require.Equal(t, uint64(2), served.Head().NumberU64())
}

func TestChain_KeepsTheLastHeadWhenTheSubscriptionIsLost(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.SetHead(1)
	served, _ := startChain(t, fake, chain.Config{})
	fake.StopFeeds()
	require.Equal(t, uint64(1), served.Head().NumberU64())
}

func TestChain_FailsToStartWithoutAHead(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.SetMode("Head", tests.Fail)
	dispatcher := dispatch.New(dispatch.DefaultConfig(), prometheus.NewRegistry())
	defer dispatcher.Stop()
	err := chain.New(chain.Config{}, fake, prometheus.NewRegistry()).Start(dispatcher)
	require.ErrorIs(t, err, tests.ErrInjected)
}

func TestChain_ReadsTheHeadAgainAfterAPanicOfItsLoop(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	served, _ := startChain(t, fake, chain.Config{})
	require.Eventually(t, func() bool { return fake.Calls("SubscribeHead") == 1 }, time.Second, time.Millisecond)

	// The loop takes the head from the event and asks the node's progress for
	// the lag; a panic there restarts the loop.
	fake.SetMode("Progress", tests.Panic)
	fake.SetHead(1)
	require.Eventually(t, func() bool { return fake.Calls("SubscribeHead") >= 2 }, 2*time.Second, time.Millisecond)
	fake.SetMode("Progress", tests.Normal)
	waitForHead(t, served, 1)
}

func TestChain_ReportsTheServedHeadAndItsLag(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.AddBlock()
	fake.SetProgress(core.Progress{CurrentBlock: 2})
	served, registry := startChain(t, fake, chain.Config{})
	fake.SetHead(1)
	waitForHead(t, served, 1)

	families, err := registry.Gather()
	require.NoError(t, err)
	values := map[string]float64{}
	for _, family := range families {
		values[family.GetName()] = family.GetMetric()[0].GetGauge().GetValue()
	}
	require.Equal(t, 1.0, values["rpc_head_served"])
	require.Equal(t, 1.0, values["rpc_head_lag"])
}

func TestResolve_TurnsReferencesIntoNumbers(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.AddBlock()
	fake.SetHead(2)
	served, _ := startChain(t, fake, chain.Config{HistoryCutoff: 1})
	hashOfOne := blockHash(t, fake, 1)
	unknown := common.Hash{0x01}
	number := func(n chain.BlockNumber) chain.BlockReference { return chain.ReferenceTo(n) }

	tests := map[string]struct {
		reference chain.BlockReference
		want      uint64
		err       error
	}{
		"number":           {number(1), 1, nil},
		"number above":     {number(9), 9, nil},
		"latest":           {number(chain.LatestBlockNumber), 2, nil},
		"pending":          {number(chain.PendingBlockNumber), 2, nil},
		"safe":             {number(chain.SafeBlockNumber), 2, nil},
		"finalized":        {number(chain.FinalizedBlockNumber), 2, nil},
		"earliest":         {number(chain.EarliestBlockNumber), 1, nil},
		"invalid tag":      {number(-6), 0, chain.ErrInvalidTag},
		"hash":             {chain.BlockReference{Hash: &hashOfOne}, 1, nil},
		"unknown hash":     {chain.BlockReference{Hash: &unknown}, 0, chain.ErrUnknownHash},
		"missing selector": {chain.BlockReference{}, 0, chain.ErrMissingSelector},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := served.Resolve(served.Head().NumberU64(), test.reference)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestResolve_PassesOnAFailureOfTheHashLookup(t *testing.T) {
	fake := tests.NewFakeCore()
	served, _ := startChain(t, fake, chain.Config{})
	fake.SetMode("BlockNumberByHash", tests.Fail)
	_, err := served.Resolve(0, chain.BlockReference{Hash: &common.Hash{}})
	require.ErrorIs(t, err, tests.ErrInjected)
}

func blockHash(t *testing.T, fake *tests.FakeCore, number uint64) common.Hash {
	t.Helper()
	block, err := fake.Block(number)
	require.NoError(t, err)
	return block.Hash
}

// The decoding of block numbers and references is go-ethereum's, which is
// the reference for answers and errors alike.
var referenceInputs = []string{
	`"latest"`, `"earliest"`, `"pending"`, `"safe"`, `"finalized"`, `latest`, `"newest"`,
	`"0x10"`, `"0x010"`, `"10"`, `""`, `"0x8000000000000000"`, `"0x10000000000000000"`,
	` "0x1" `, `5`, `"0xzz"`, `null`,
	`"0x1100000000000000000000000000000000000000000000000000000000000000"`,
	`"0xzz00000000000000000000000000000000000000000000000000000000000000"`,
	`{"blockNumber":"0x5"}`, `{"blockNumber":5}`, `{"blockNumber":null}`, `{}`,
	`{"blockHash":"0x1100000000000000000000000000000000000000000000000000000000000000","requireCanonical":true}`,
	`{"blockNumber":"0x5","blockHash":"0x1100000000000000000000000000000000000000000000000000000000000000"}`,
	`{"requireCanonical":"yes"}`, `{"blockNumber":"0xzz"}`,
}

func TestBlockNumber_DecodesAsGoEthereum(t *testing.T) {
	for _, input := range referenceInputs {
		t.Run(input, func(t *testing.T) {
			var want rpc.BlockNumber
			wantErr := json.Unmarshal([]byte(input), &want)
			var got chain.BlockNumber
			gotErr := json.Unmarshal([]byte(input), &got)
			require.Equal(t, errorText(wantErr), errorText(gotErr))
			require.Equal(t, int64(want), int64(got))
		})
	}
}

func TestBlockReference_DecodesAsGoEthereum(t *testing.T) {
	for _, input := range referenceInputs {
		t.Run(input, func(t *testing.T) {
			var want rpc.BlockNumberOrHash
			wantErr := json.Unmarshal([]byte(input), &want)
			var got chain.BlockReference
			gotErr := json.Unmarshal([]byte(input), &got)
			require.Equal(t, errorText(wantErr), errorText(gotErr))
			require.Equal(t, (*int64)(want.BlockNumber), (*int64)(got.Number))
			require.Equal(t, want.BlockHash, got.Hash)
		})
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
