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
	filters     *filterSet
}

// Register adds the methods of the namespace to the dispatcher, and starts
// the loops that feed and expire the polled filters.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node) {
	h := &handlers{environment: environment, node: node, filters: newFilterSet(environment.Config().FilterTimeout)}
	dispatch.Add(dispatcher, "eth_getLogs", h.getLogs)
	dispatch.Add(dispatcher, "eth_newFilter", h.newFilter)
	dispatch.Add(dispatcher, "eth_newBlockFilter", h.newBlockFilter)
	dispatch.Add(dispatcher, "eth_newPendingTransactionFilter", h.newPendingTransactionFilter)
	dispatch.Add(dispatcher, "eth_getFilterChanges", h.getFilterChanges)
	dispatch.Add(dispatcher, "eth_getFilterLogs", h.getFilterLogs)
	dispatch.Add(dispatcher, "eth_uninstallFilter", h.uninstallFilter)
	h.startJournals(dispatcher)
	dispatcher.Loop("polled filters expiry", h.expireFilters)
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
