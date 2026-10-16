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

package eth

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// spins is JUMPDEST PUSH1 0 JUMP: a contract that runs until it is stopped.
var spins = []byte{0x5b, 0x60, 0x00, 0x56}

// spinsInGeth is a spinning contract that exceeds the on-chain code size limit,
// which makes the call run on the go-ethereum interpreter. Only that one
// stops at a timeout.
func spinsInGeth() map[string]any {
	code := append(append([]byte{}, spins...), make([]byte, 30_000)...)
	return map[string]any{contract.Hex(): map[string]any{"code": hexutil.Encode(code)}}
}

func newTimeoutHarness(t *testing.T, mutate func(*dispatch.Config)) (*tests.Harness, *tests.CarmenState) {
	t.Helper()
	cfg := tests.Default()
	mutate(&cfg)
	fc, cs := tests.NewCarmenCore(t, tests.WithBlocks(2))
	h := tests.NewHarness(t, fc, cfg, func(d *dispatch.Dispatcher, chain *dispatch.Chain) {
		Register(d, exec.NewEnv(chain, oracle.New(oracle.Config{}, fc)), "eth")
	})
	cs.SetCode(contract, spins)
	return h, cs
}

// O2: the EVM stops at the configured timeout, with the message of the
// reference release.
func TestCall_EVMTimeoutStopsTheInterpreter(t *testing.T) {
	h, _ := newTimeoutHarness(t, func(c *dispatch.Config) {
		c.EVMTimeout = 30 * time.Millisecond
		c.Timeout = 10 * time.Second
		c.GasCap = 0
	})
	start := time.Now()
	err := h.Fails("eth_call", map[string]any{"to": contract, "gas": hexutil.Uint64(1 << 40)}, "latest", spinsInGeth())
	require.Contains(t, err.Message, "execution aborted (timeout = 30ms)")
	require.Less(t, time.Since(start), 5*time.Second)

}

// O2: the request deadline wins when it is shorter than the EVM timeout.
func TestCall_RequestDeadlineStopsTheCall(t *testing.T) {
	h, _ := newTimeoutHarness(t, func(c *dispatch.Config) {
		c.EVMTimeout = 10 * time.Second
		c.Timeout = 50 * time.Millisecond
		c.GasCap = 0
	})
	start := time.Now()
	err := h.Fails("eth_call", map[string]any{"to": contract, "gas": hexutil.Uint64(1 << 40)}, "latest", spinsInGeth())
	require.NotNil(t, err)
	require.Less(t, time.Since(start), 5*time.Second)

	// The estimate runs without an EVM timeout of its own, as in the reference
	// release: only the request deadline stops it.
	start = time.Now()
	err = h.Fails("eth_estimateGas", map[string]any{"to": contract, "gas": hexutil.Uint64(1 << 40)}, "latest", spinsInGeth())
	require.NotNil(t, err)
	require.Less(t, time.Since(start), 5*time.Second)
}

// O1: the gas cap bounds a call that does not set its own gas.
func TestCall_GasCapBoundsTheWork(t *testing.T) {
	h, _ := newTimeoutHarness(t, func(c *dispatch.Config) {
		c.GasCap = 100_000
	})
	start := time.Now()
	err := h.Fails("eth_call", map[string]any{"to": contract}, "latest")
	require.NotNil(t, err)
	require.Less(t, time.Since(start), 5*time.Second)
}
