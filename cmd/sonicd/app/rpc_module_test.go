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

package app

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

func TestTakeOverRPC_HTTPAndWebSocketOnOneEndpoint(t *testing.T) {
	cfg := &node.Config{
		HTTPHost: "127.0.0.1", HTTPPort: 18545, HTTPPathPrefix: "/rpc",
		HTTPCors: []string{"http://a"}, HTTPVirtualHosts: []string{"localhost"}, HTTPModules: []string{"eth", "net"},
		WSHost: "127.0.0.1", WSPort: 18545, WSPathPrefix: "/ws", WSOrigins: []string{"http://b"}, WSModules: []string{"eth"},
		BatchRequestLimit: 7, BatchResponseMaxSize: 700,
		IPCPath: "sonic.ipc", DataDir: t.TempDir(),
		HTTPTimeouts: gethrpc.HTTPTimeouts{ReadTimeout: 5 * time.Second},
	}
	takeover := takeOverRPC(cfg)

	require.Empty(t, cfg.HTTPHost)
	require.Empty(t, cfg.WSHost)
	require.Empty(t, cfg.IPCPath)

	require.NotNil(t, takeover.http)
	require.Nil(t, takeover.ws, "the same endpoint serves both")
	require.Equal(t, "127.0.0.1", takeover.http.Host)
	require.Equal(t, 18545, takeover.http.Port)
	require.Equal(t, "/rpc", takeover.http.Prefix)
	require.Equal(t, []string{"http://a"}, takeover.http.Cors)
	require.Equal(t, []string{"localhost"}, takeover.http.VHosts)
	require.Equal(t, []string{"eth", "net"}, takeover.http.Modules)
	require.Equal(t, 7, takeover.http.BatchLimit)
	require.Equal(t, 700, takeover.http.BatchResponseMax)
	require.Equal(t, 5*time.Second, takeover.http.Timeouts.Read)
	require.NotNil(t, takeover.http.WS)
	require.Equal(t, "/ws", takeover.http.WSPrefix)
	require.Equal(t, []string{"http://b"}, takeover.http.WS.Origins)
	require.Equal(t, []string{"eth"}, takeover.http.WS.Modules)
	require.Equal(t, 7, takeover.http.WS.BatchLimit)

	require.NotNil(t, takeover.ipc)
	require.Contains(t, takeover.ipc.Path, "sonic.ipc")

	// The settings are put back for the places that report the configuration.
	takeover.restore(cfg)
	require.Equal(t, "127.0.0.1", cfg.HTTPHost)
	require.Equal(t, "127.0.0.1", cfg.WSHost)
	require.Equal(t, 18545, cfg.WSPort)
	require.Equal(t, []string{"http://b"}, cfg.WSOrigins)
	require.Equal(t, "sonic.ipc", cfg.IPCPath)
}

func TestTakeOverRPC_WebSocketOnItsOwnEndpoint(t *testing.T) {
	cfg := &node.Config{
		HTTPHost: "127.0.0.1", HTTPPort: 8545,
		WSHost: "127.0.0.1", WSPort: 8546, WSOrigins: []string{"*"},
	}
	takeover := takeOverRPC(cfg)
	require.NotNil(t, takeover.http)
	require.Nil(t, takeover.http.WS)
	require.NotNil(t, takeover.ws)
	require.Equal(t, 8546, takeover.ws.Port)
	require.NotNil(t, takeover.ws.WS)
	require.Equal(t, []string{"*"}, takeover.ws.WS.Origins)
}

func TestTakeOverRPC_OnlyWhatIsConfiguredIsServed(t *testing.T) {
	takeover := takeOverRPC(&node.Config{})
	require.Nil(t, takeover.http)
	require.Nil(t, takeover.ws)
	require.Nil(t, takeover.ipc)

	takeover = takeOverRPC(&node.Config{WSHost: "127.0.0.1", WSPort: 8546})
	require.Nil(t, takeover.http)
	require.NotNil(t, takeover.ws)
}

func TestLimitsFromConfig(t *testing.T) {
	g := gossip.DefaultConfig(cachescale.Identity)
	g.RPCTimeout = 3 * time.Second
	g.RPCEVMTimeout = 4 * time.Second
	g.RPCGasCap = 12345
	g.RPCTxFeeCap = 2.5
	g.AllowUnprotectedTxs = true
	g.MaxResponseSize = 999
	g.StructLogLimit = 17
	g.MaxTraceFilterRange = 33
	g.AllowJSTracers = true
	g.FilterAPI.LogQueryParameterLimit = 11
	g.FilterAPI.LogQueryResultLimit = 12
	g.GPO = gasprice.Config{MaxGasPrice: big.NewInt(77), DefaultCertainty: 5}

	limits := limitsFromConfig(g)
	require.Equal(t, 3*time.Second, limits.Timeout)
	require.Equal(t, 4*time.Second, limits.EVMTimeout)
	require.EqualValues(t, 12345, limits.GasCap)
	require.Equal(t, 2.5, limits.TxFeeCap)
	require.True(t, limits.UnprotectedAllowed)
	require.Equal(t, 999, limits.MaxResponseSize)
	require.Equal(t, 17, limits.StructLogLimit)
	require.EqualValues(t, 33, limits.MaxTraceFilterRange)
	require.True(t, limits.AllowJSTracers)
	require.EqualValues(t, 11, limits.LogQueryParameterLimit)
	require.EqualValues(t, 12, limits.LogQueryResultLimit)
	require.Equal(t, big.NewInt(77), limits.OracleMaxGasPrice)
	require.EqualValues(t, 5, limits.OracleDefaultCertainty)

	// The oracle keeps its defaults where the configuration has none.
	g.GPO = gasprice.Config{}
	limits = limitsFromConfig(g)
	require.Equal(t, dispatch.DefaultConfig().OracleMaxGasPrice, limits.OracleMaxGasPrice)
	require.Equal(t, dispatch.DefaultConfig().OracleDefaultCertainty, limits.OracleDefaultCertainty)
}

func TestLegacyError_KeepsCodeAndData(t *testing.T) {
	err := legacyError(&jsonError{code: -32000, message: "boom", data: "0x01"})
	require.Equal(t, -32000, err.Code)
	require.Equal(t, "boom", err.Message)
	require.Equal(t, "0x01", err.Data)

	err = legacyError(&jsonError{code: 3, message: "reverted"})
	require.Equal(t, 3, err.Code)
	require.Nil(t, err.Data)

	err = legacyError(errors.New("connection lost"))
	require.Equal(t, dispatch.CodeInternal, err.Code)
	require.Equal(t, "connection lost", err.Message)
}

func TestForward_WithoutANodeAnswersInternalError(t *testing.T) {
	takeover := &rpcTakeover{}
	_, err := takeover.Forward(context.Background(), "admin_peers", nil)
	require.NotNil(t, err)
	require.Equal(t, dispatch.CodeInternal, err.Code)
	require.Equal(t, errNoLegacy.Error(), err.Message)
}

type jsonError struct {
	code    int
	message string
	data    any
}

func (e *jsonError) Error() string          { return e.message }
func (e *jsonError) ErrorCode() int         { return e.code }
func (e *jsonError) ErrorData() interface{} { return e.data }

var _ gethrpc.DataError = (*jsonError)(nil)
