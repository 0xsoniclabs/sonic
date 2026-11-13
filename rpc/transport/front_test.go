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
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func newNamespacesDispatcher(t *testing.T) *dispatch.Dispatcher {
	t.Helper()
	dispatcher := dispatch.New(dispatch.DefaultConfig(), prometheus.NewRegistry())
	for _, method := range []string{"eth_chainId", "debug_traceTransaction", "net_version"} {
		dispatch.Add(dispatcher, method, func(context.Context, struct{}) (string, error) { return "ok", nil })
	}
	dispatch.Add(dispatcher, "rpc_modules", func(ctx context.Context, _ struct{}) (map[string]string, error) {
		return Modules(ctx), nil
	})
	t.Cleanup(dispatcher.Stop)
	return dispatcher
}

func callFront(t *testing.T, front *Front, method string) *message {
	t.Helper()
	return front.call(context.Background(), &message{Version: version, ID: json.RawMessage("1"), Method: method})
}

// The API list enables namespaces per transport; the node's namespaces and
// the deprecated mirror of eth are ignored, and the metadata is always there.
func TestFront_ServesTheNamespacesOfItsList(t *testing.T) {
	dispatcher := newNamespacesDispatcher(t)
	cases := map[string]struct {
		list    []string
		served  []string
		refused []string
		modules map[string]string
	}{
		"all without a list": {
			list:    nil,
			served:  []string{"eth_chainId", "debug_traceTransaction", "net_version"},
			modules: map[string]string{"eth": "1.0", "debug": "1.0", "net": "1.0", "rpc": "1.0"},
		},
		"the listed only": {
			list:    []string{"eth"},
			served:  []string{"eth_chainId"},
			refused: []string{"debug_traceTransaction", "net_version"},
			modules: map[string]string{"eth": "1.0", "rpc": "1.0"},
		},
		"node namespaces and the eth mirror ignored": {
			list:    []string{"admin", "ftm", "net", "nope"},
			served:  []string{"net_version"},
			refused: []string{"eth_chainId", "ftm_chainId", "admin_datadir"},
			modules: map[string]string{"net": "1.0", "rpc": "1.0"},
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			front := NewFront(dispatcher, test.list, "http.api")
			for _, method := range test.served {
				require.Equal(t, `"ok"`, string(callFront(t, front, method).Result), method)
			}
			for _, method := range test.refused {
				require.Equal(t, dispatch.MethodNotFound(method), callFront(t, front, method).Error, method)
			}
			var modules map[string]string
			require.NoError(t, json.Unmarshal(callFront(t, front, "rpc_modules").Result, &modules))
			require.Equal(t, test.modules, modules)
		})
	}
}

func TestFront_ForwardsNothingWithoutTheNodesHandler(t *testing.T) {
	front := NewFront(newNamespacesDispatcher(t), nil, "http.api")
	require.Equal(t, dispatch.MethodNotFound("admin_datadir"), callFront(t, front, "admin_datadir").Error)
}

// A node's handler that answers no message is reported, not passed on.
func TestFront_ReportsAnUnreadableAnswerOfTheNodesHandler(t *testing.T) {
	front := NewIPCFront(newNamespacesDispatcher(t), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.Equal(t, dispatch.CodeInternal, callFront(t, front, "admin_datadir").Error.Code)
}
