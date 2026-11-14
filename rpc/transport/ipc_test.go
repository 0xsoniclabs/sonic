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

package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// adminService stands for the node's own handler: its administration, which
// IPC forwards to.
type adminService struct{}

func (adminService) Datadir() string { return "/data" }

func (adminService) StartHTTP() bool { return true }

func startModuleIPC(t *testing.T) string {
	t.Helper()
	node := rpc.NewServer()
	require.NoError(t, node.RegisterName("admin", adminService{}))
	dispatcher := newTestDispatcher(t)
	path := filepath.Join(t.TempDir(), "sonic.ipc")
	server, err := StartIPC(dispatcher, path, NewIPCFront(dispatcher, node), prometheus.NewRegistry())
	require.NoError(t, err)
	t.Cleanup(server.Stop)
	return path
}

func startReferenceIPC(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "reference.ipc")
	listener, err := net.Listen("unix", path)
	require.NoError(t, err)
	server := newReference(t)
	go func() { _ = server.ServeListener(listener) }()
	t.Cleanup(func() {
		_ = listener.Close()
		server.Stop()
	})
	return path
}

// exchange writes a request to a socket and reads the answers it gets within
// a short while, in the order they arrive sorted, as calls on a stream are
// answered in any order.
func exchange(t *testing.T, path, request string, answers int) []string {
	t.Helper()
	connection, err := net.Dial("unix", path)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	_, err = connection.Write([]byte(request))
	require.NoError(t, err)
	reader := bufio.NewReader(connection)
	var got []string
	for range answers {
		require.NoError(t, connection.SetReadDeadline(time.Now().Add(5*time.Second)))
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		got = append(got, line)
	}
	require.NoError(t, connection.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	_, err = reader.ReadString('\n')
	require.Error(t, err, "no further answer")
	sort.Strings(got)
	return got
}

func TestIPC_AnswersAsGoEthereumServer(t *testing.T) {
	reference := startReferenceIPC(t)
	module := startModuleIPC(t)
	cases := map[string]struct {
		request string
		answers int
	}{
		"call":           {`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["hi"]}`, 1},
		"two calls":      {`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"test_add","params":[1,2]}`, 2},
		"concatenated":   {`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]}{"jsonrpc":"2.0","id":2,"method":"test_echo","params":["b"]}`, 2},
		"notification":   {`{"jsonrpc":"2.0","method":"test_echo","params":["hi"]}`, 0},
		"batch":          {envelopeBodies["batch"], 1},
		"empty batch":    {`[]`, 1},
		"unknown method": {`{"jsonrpc":"2.0","id":1,"method":"test_unknown"}`, 1},
		"syntax error":   {`{"jsonrpc":"2.0",]`, 1},
		"large batch":    {largeBatch(30), 1},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			want := exchange(t, reference, test.request, test.answers)
			got := exchange(t, module, test.request, test.answers)
			require.Equal(t, want, got)
		})
	}
}

// ids replaces the random ids of subscriptions with a placeholder.
var ids = regexp.MustCompile(`"0x[0-9a-f]{8,32}"`)

// A subscription is answered with its id before its notifications, and its
// calls are checked in go-ethereum's order and words.
func TestIPC_SubscribesAsGoEthereumServer(t *testing.T) {
	reference := startReferenceIPC(t)
	module := startModuleIPC(t)
	call := func(method, params string) string {
		return `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`
	}
	cases := map[string]struct {
		request string
		answers int
		ordered bool
	}{
		"subscription":            {call("test_subscribe", `["ticks",3]`), 4, true},
		"in a batch":              {"[" + call("test_subscribe", `["ticks",2]`) + "]", 3, true},
		"no array":                {call("test_subscribe", `{}`), 1, false},
		"no name":                 {call("test_subscribe", `[1]`), 1, false},
		"unknown kind":            {call("test_subscribe", `["other"]`), 1, false},
		"unknown namespace":       {call("other_subscribe", `["ticks",1]`), 1, false},
		"missing parameter":       {call("test_subscribe", `["ticks"]`), 1, false},
		"too many parameters":     {call("test_subscribe", `["ticks",1,"a",2]`), 1, false},
		"invalid parameter":       {call("test_subscribe", `["ticks","x"]`), 1, false},
		"long name":               {call(strings.Repeat("a", 2048)+"_subscribe", `["ticks",1]`), 1, false},
		"unsubscribe unknown":     {call("test_unsubscribe", `["0x1"]`), 1, false},
		"unsubscribe no id":       {call("test_unsubscribe", `[]`), 1, false},
		"unsubscribe invalid id":  {call("other_unsubscribe", `[1]`), 1, false},
		"unsubscribe long method": {call(strings.Repeat("a", 2048)+"_unsubscribe", `["0x1"]`), 1, false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			exchanged := exchange
			if test.ordered {
				exchanged = exchangeInOrder
			}
			want := exchanged(t, reference, test.request, test.answers)
			got := exchanged(t, module, test.request, test.answers)
			for i := range want {
				want[i], got[i] = ids.ReplaceAllString(want[i], `"id"`), ids.ReplaceAllString(got[i], `"id"`)
			}
			require.Equal(t, want, got)
		})
	}
}

// exchangeInOrder is exchange without sorting the answers.
func exchangeInOrder(t *testing.T, path, request string, answers int) []string {
	t.Helper()
	connection, err := net.Dial("unix", path)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	_, err = connection.Write([]byte(request))
	require.NoError(t, err)
	reader := bufio.NewReader(connection)
	var got []string
	for range answers {
		require.NoError(t, connection.SetReadDeadline(time.Now().Add(5*time.Second)))
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		got = append(got, line)
	}
	return got
}

// IPC has no batch limits, as go-ethereum's IPC server has none.
func largeBatch(size int) string {
	batch := "["
	for i := range size {
		if i > 0 {
			batch += ","
		}
		batch += `{"jsonrpc":"2.0","id":1,"method":"test_large","params":[100]}`
	}
	return batch + "]"
}

func TestIPC_ForwardsTheNodesAdministration(t *testing.T) {
	module := startModuleIPC(t)
	cases := map[string]struct {
		request, want string
	}{
		"administration": {`{"jsonrpc":"2.0","id":1,"method":"admin_datadir"}`, `{"jsonrpc":"2.0","id":1,"result":"/data"}` + "\n"},
		"endpoint control": {`{"jsonrpc":"2.0","id":1,"method":"admin_startHTTP"}`,
			`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method admin_startHTTP does not exist/is not available"}}` + "\n"},
		"unknown of the node": {`{"jsonrpc":"2.0","id":"x","method":"debug_nothing"}`,
			`{"jsonrpc":"2.0","id":"x","error":{"code":-32601,"message":"the method debug_nothing does not exist/is not available"}}` + "\n"},
		"other namespace": {`{"jsonrpc":"2.0","id":1,"method":"personal_listAccounts"}`,
			`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method personal_listAccounts does not exist/is not available"}}` + "\n"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, []string{test.want}, exchange(t, module, test.request, 1))
		})
	}
	require.Empty(t, exchange(t, module, `{"jsonrpc":"2.0","method":"admin_datadir"}`, 0), "a forwarded notification has no answer")
}

func TestIPC_ListsTheModulesOfTheModuleAndOfTheNode(t *testing.T) {
	module := startModuleIPC(t)
	answers := exchange(t, module, `{"jsonrpc":"2.0","id":1,"method":"rpc_modules"}`, 1)
	var response struct{ Result map[string]string }
	require.NoError(t, json.Unmarshal([]byte(answers[0]), &response))
	require.Equal(t, map[string]string{"admin": "1.0", "rpc": "1.0", "test": "1.0"}, response.Result)
}

func TestIPC_TheSocketIsTheNodeUsersOnly(t *testing.T) {
	path := startModuleIPC(t)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestIPC_ReplacesASocketLeftBehind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sonic.ipc")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	dispatcher := newTestDispatcher(t)
	server, err := StartIPC(dispatcher, path, NewIPCFront(dispatcher, rpc.NewServer()), prometheus.NewRegistry())
	require.NoError(t, err)
	server.Stop()
}

func TestIPC_StopEndsTheConnectionsAndTheirCalls(t *testing.T) {
	dispatcher := newTestDispatcher(t)
	path := filepath.Join(t.TempDir(), "sonic.ipc")
	server, err := StartIPC(dispatcher, path, NewIPCFront(dispatcher, rpc.NewServer()), prometheus.NewRegistry())
	require.NoError(t, err)
	connection, err := net.Dial("unix", path)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()
	_, err = connection.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a"]}`))
	require.NoError(t, err)
	_, err = bufio.NewReader(connection).ReadString('\n')
	require.NoError(t, err)

	server.Stop()
	require.NoError(t, connection.SetReadDeadline(time.Now().Add(time.Second)))
	_, err = connection.Read(make([]byte, 1))
	require.Error(t, err, "the connection is closed")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = (&net.Dialer{}).DialContext(ctx, "unix", path)
	require.Error(t, err, "the socket no longer accepts")
}
