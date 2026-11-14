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
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
)

// epochSnapshot is the states of an epoch with rules of the given name, the
// last block before it and its start and the start of the epoch before.
func epochSnapshot(name string, lastBlock uint64, previousStart, start inter.Timestamp) core.EpochSnapshot {
	var states core.EpochSnapshot
	states.EpochState.Rules = opera.Rules{Name: name}
	states.BlockState.LastBlock.Idx = idx.Block(lastBlock)
	states.EpochState.PrevEpochStart = previousStart
	states.EpochState.EpochStart = start
	return states
}

// newEpochHarness serves a head in epoch 3 with the states of epochs 0, 2
// and 3 and the node's live states.
func newEpochHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewFakeCore()
	fake.SetEpoch(3)
	fake.AddBlock()
	fake.SetHead(1)
	fake.SetEpochState(0, epochSnapshot("zero", 0, 0, 0))
	fake.SetEpochState(2, epochSnapshot("two", 4, 5, 10))
	fake.SetEpochState(3, epochSnapshot("three", 5, 10, 20))
	fake.SetLiveEpochState(epochSnapshot("live", 6, 10, 20))
	return newHarness(t, fake)
}

func TestCurrentEpoch_IsTheEpochOfTheNextBlock(t *testing.T) {
	h := newEpochHarness(t)
	require.Equal(t, `"0x3"`, h.Result(t, "eth_currentEpoch"))
	h.Core.SetMode("EpochOf", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_currentEpoch").Message)
}

func TestGetRules_ReadsTheStatesOfEachEpochSelector(t *testing.T) {
	h := newEpochHarness(t)
	for selector, name := range map[string]string{
		"latest": "three", "safe": "three", "finalized": "three", "pending": "live", "earliest": "zero", "0x2": "two",
	} {
		var rules opera.Rules
		h.Call(t, &rules, "eth_getRules", selector)
		require.Equal(t, name, rules.Name, selector)
	}
	require.Equal(t, "null", h.Result(t, "eth_getRules", "0x7"))
	h.Core.SetMode("EpochState", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "eth_getRules", "latest").Message)
}

func TestGetEpochBlock_IsTheLastBlockBeforeTheEpoch(t *testing.T) {
	h := newEpochHarness(t)
	require.Equal(t, `"0x5"`, h.Result(t, "eth_getEpochBlock", "latest"))
	require.Equal(t, `"0x6"`, h.Result(t, "eth_getEpochBlock", "pending"))
	require.Equal(t, `"0x0"`, h.Result(t, "eth_getEpochBlock", "0x7"))
}

func TestGetEpochStats_AnswersTheLastSealedEpochOnly(t *testing.T) {
	h := newEpochHarness(t)
	stats := `{"end":"0x14","epoch":"0x2","start":"0xa","totalBaseRewardWeight":"0x0","totalFee":"0x0","totalTxRewardWeight":"0x0"}`
	require.JSONEq(t, stats, h.Result(t, "eth_getEpochStats", "latest"))
	require.JSONEq(t, stats, h.Result(t, "eth_getEpochStats", "0x2"))
	for _, selector := range []string{"0x1", "0x3", "pending"} {
		require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "getEpochStats API call doesn't support retrieving previous sealed epochs"},
			h.Fail(t, "eth_getEpochStats", selector), selector)
	}
}
