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
	"bytes"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// lockedBuffer is a log output the request goroutines and the test share.
type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (output *lockedBuffer) Write(written []byte) (int, error) {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	return output.buffer.Write(written)
}

func (output *lockedBuffer) String() string {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	return output.buffer.String()
}

func captureLog(t *testing.T) *lockedBuffer {
	t.Helper()
	output := &lockedBuffer{}
	previous := log.Root()
	log.SetDefault(log.NewLogger(slog.NewTextHandler(output, nil)))
	t.Cleanup(func() { log.SetDefault(previous) })
	return output
}

// TestStop_WaitsForAToscaRunAtTheGasCap holds Q1 and D10: Tosca's
// interpreter does not stop when a request is cancelled, so Stop waits for
// an eth_call that runs to the gas cap, logs what it waits for, and returns
// once the run ends, within the bound the gas cap sets.
func TestStop_WaitsForAToscaRunAtTheGasCap(t *testing.T) {
	output := captureLog(t)
	chain := newStateChain(t, callAccounts, func(configs *tests.Configs) {
		configs.Dispatch.RequestTimeout = 50 * time.Millisecond
		configs.Exec.RPCGasCap = 200_000_000
	})
	states := chain.core.Calls("State")
	answered := make(chan struct{})
	go func() {
		defer close(answered)
		_, _ = chain.harness.Call("eth_call", callTo(looper, "gas", "0xbebc200"))
	}()
	tests.Eventually(t, func() bool { return chain.core.Calls("State") > states }, "the call to open its state")

	start := time.Now()
	chain.harness.Dispatcher.Stop()
	require.Less(t, time.Since(start), 30*time.Second, "Stop returns once the run reaches the gas cap")
	select {
	case <-answered:
	case <-time.After(time.Second):
		t.Fatal("Stop returned before the call it waited for")
	}
	require.Contains(t, output.String(), "RPC stop waits for a request that ignores cancellation")
	require.Contains(t, output.String(), "method=eth_call")
}

// TestStop_WaitsForABlockedCoreCall holds Q1 and D10 for a call of the node
// that does not return: Stop logs the request it waits for and returns once
// the call returns.
func TestStop_WaitsForABlockedCoreCall(t *testing.T) {
	output := captureLog(t)
	chain := newStateChain(t, callAccounts, func(configs *tests.Configs) {
		configs.Dispatch.RequestTimeout = 50 * time.Millisecond
	})
	release := chain.core.Hold("Block")
	answered := make(chan struct{})
	go func() {
		defer close(answered)
		_, _ = chain.harness.Call("eth_getBlockByNumber", "0x1", false)
	}()
	tests.Eventually(t, func() bool { return chain.core.Calls("Block") > 0 }, "the request to reach the node")

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		chain.harness.Dispatcher.Stop()
	}()
	tests.Eventually(t, func() bool {
		return bytes.Contains([]byte(output.String()), []byte("method=eth_getBlockByNumber"))
	}, "Stop to log the request it waits for")
	select {
	case <-stopped:
		t.Fatal("Stop returned while the node's call was blocked")
	case <-time.After(100 * time.Millisecond):
	}
	release()
	<-stopped
	<-answered
}
