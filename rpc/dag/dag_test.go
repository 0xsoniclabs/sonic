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

package dag_test

import (
	"encoding/json"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/rpc/dag"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func newEvent(epoch idx.Epoch, lamport idx.Lamport, creator idx.ValidatorID) *inter.EventPayload {
	builder := &inter.MutableEventPayload{}
	builder.SetVersion(3)
	builder.SetEpoch(epoch)
	builder.SetLamport(lamport)
	builder.SetCreator(creator)
	return builder.Build()
}

// newHarness serves a head in epoch 3, whose heads the node keeps, and three
// events: two of one epoch and lamport, one of another.
func newHarness(t *testing.T) (*tests.Harness, []*inter.EventPayload) {
	t.Helper()
	fake := tests.NewFakeCore()
	fake.SetEpoch(3)
	fake.AddBlock()
	fake.SetHead(1)
	events := []*inter.EventPayload{newEvent(3, 7, 1), newEvent(3, 7, 2), newEvent(3, 8, 1)}
	for _, event := range events {
		fake.AddEvent(event)
	}
	fake.SetLiveEpochHeads(3, []common.Hash{common.Hash(events[0].ID()), common.Hash(events[2].ID())})
	return tests.NewHarness(t, fake, func(h *tests.Harness) {
		dag.Register(h.Dispatcher, h.Environment, h.Core)
	}), events
}

func shortID(event *inter.EventPayload) string {
	return event.ID().String()
}

func TestGetEvent_FindsAnEventByItsHashOrShortID(t *testing.T) {
	h, events := newHarness(t)
	event := events[2]
	for _, name := range []string{hexutil.Encode(event.ID().Bytes()), shortID(event)} {
		var answer struct {
			ID      hexutil.Bytes  `json:"id"`
			Lamport hexutil.Uint64 `json:"lamport"`
		}
		h.Call(t, &answer, "dag_getEvent", name)
		require.Equal(t, event.ID().Bytes(), []byte(answer.ID), name)
		require.Equal(t, hexutil.Uint64(8), answer.Lamport)
	}

	var payload map[string]json.RawMessage
	h.Call(t, &payload, "dag_getEventPayload", shortID(event), true)
	require.Equal(t, "[]", string(payload["transactions"]))
	h.Call(t, &payload, "dag_getEventPayload", shortID(event), false)
	require.Contains(t, payload, "size")
}

func TestGetEvent_RefusesWhatNamesNoOneEvent(t *testing.T) {
	h, _ := newHarness(t)
	unknown := hexutil.Encode(make([]byte, 32))
	cases := map[string]string{
		unknown:  "event " + unknown + " not found",
		"zz":     "full hash parsing error: hex string without 0x prefix",
		"1:2":    "incorrect format of short event ID (need Epoch:Lamport:Hash",
		"x:7:ab": `short hash parsing error (lamport): strconv.ParseUint: parsing "x": invalid syntax`,
		"3:y:ab": `short hash parsing error (lamport): strconv.ParseUint: parsing "y": invalid syntax`,
		"3:9:":   "event not found by short ID",
		"3:7:":   "there're multiple events with the same short ID, please use full ID",
	}
	for name, err := range cases {
		for _, method := range []string{"dag_getEvent", "dag_getEventPayload"} {
			params := []any{name}
			if method == "dag_getEventPayload" {
				params = append(params, false)
			}
			require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: err}, h.Fail(t, method, params...), "%s %s", method, name)
		}
	}
	h.Core.SetMode("Event", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "dag_getEvent", unknown).Message)
}

func TestGetHeads_AnswersTheOpenEpochOnly(t *testing.T) {
	h, events := newHarness(t)
	heads := `["` + hexutil.Encode(events[0].ID().Bytes()) + `","` + hexutil.Encode(events[2].ID().Bytes()) + `"]`
	require.Equal(t, heads, h.Result(t, "dag_getHeads", "pending"))
	require.Equal(t, heads, h.Result(t, "dag_getHeads", "0x3"))
	cases := map[string]string{
		"latest":   "epoch 2: heads are not available",
		"earliest": "epoch 0: heads are not available",
		"0x4":      "epoch is not in range",
	}
	for selector, err := range cases {
		require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: err}, h.Fail(t, "dag_getHeads", selector), selector)
	}
	h.Core.SetMode("LiveEpochHeads", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "dag_getHeads", "pending").Message)
	h.Core.SetMode("EpochOf", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "dag_getHeads", "pending").Message)
}
