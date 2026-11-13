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

package net_test

import (
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/net"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/stretchr/testify/require"
)

func TestNet_AnswersTheNetworkAndItsPeers(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.SetPeerCount(3)
	h := tests.NewHarness(t, fake, func(h *tests.Harness) {
		net.Register(h.Dispatcher, h.Environment, h.Core)
	})
	cases := map[string]struct {
		method string
		want   string
	}{
		"version":    {"net_version", `"4003"`},
		"listening":  {"net_listening", "true"},
		"peer count": {"net_peerCount", `"0x3"`},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, h.Result(t, test.method))
		})
	}
}
