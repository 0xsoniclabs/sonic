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

// Package filters serves the logs of the RPC module: log queries, the polled
// filters and the subscriptions of the eth namespace.
package filters

import (
	"context"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/core/types"
)

// Node is what the namespace asks the node.
type Node interface {
	core.LogIndex
}

type handlers struct {
	environment *exec.Environment
	node        Node
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node) {
	h := &handlers{environment: environment, node: node}
	dispatch.Add(dispatcher, "eth_getLogs", h.getLogs)
}

type logsParams struct {
	Criteria criteria
}

// getLogs answers the logs of the served blocks that the criteria select.
func (h *handlers) getLogs(ctx context.Context, params logsParams) ([]*types.Log, error) {
	logs, err := h.node.Logs(ctx, params.Criteria.query(h.environment.Head().NumberU64()))
	if err != nil {
		return nil, err
	}
	return answered(logs), nil
}
