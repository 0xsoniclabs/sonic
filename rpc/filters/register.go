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

// Package filters serves the log queries, the polled filters and the
// subscriptions of the eth namespace, which the node also serves as ftm.
package filters

import (
	"context"
	"encoding/json"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

type handlers struct {
	environment *exec.Environment
	logIndex    core.LogIndex
	polled      *polledFilters
	headers     *headers
}

// Register adds the methods of the package under each namespace name (eth
// and ftm), which share their filters, as in the reference, and starts the
// loops that feed and expire the polled filters.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, logIndex core.LogIndex, namespaces ...string) {
	polled := newPolledFilters(environment.Config().FilterTimeout, journalLimit)
	built := &headers{environment: environment, hub: chain.NewHub[*evmcore.EvmHeaderJson](headerQueue)}
	handler := &handlers{environment: environment, logIndex: logIndex, polled: polled, headers: built}
	criteria := dispatch.Required[Criteria]("criteria")
	identifier := dispatch.Required[string]("filter")
	for _, namespace := range namespaces {
		add := func(name string, function dispatch.Handler, parameters ...dispatch.Parameter) {
			dispatcher.Add(namespace+"_"+name, function, parameters...)
		}
		add("getLogs", handler.getLogs, criteria)
		add("newFilter", handler.newFilter, criteria)
		add("newBlockFilter", handler.newBlockFilter)
		add("newPendingTransactionFilter", handler.newPendingTransactionFilter, dispatch.Optional[bool]("fullTransactions"))
		add("getFilterChanges", handler.getFilterChanges, identifier)
		add("getFilterLogs", handler.getFilterLogs, identifier)
		add("uninstallFilter", handler.uninstallFilter, identifier)
		add("subscribe", handler.subscribe(namespace), dispatch.Required[string]("subscription"), dispatch.Optional[json.RawMessage]("argument"))
	}
	dispatcher.Loop("subscriptions-heads", built.follow(environment.Heads()))
	dispatcher.Loop("filters-heads", polled.followHeads(environment.Heads()))
	dispatcher.Loop("filters-pool", polled.followPool(environment.Transactions()))
	dispatcher.Loop("filters-expiry", polled.expire)
}

func (handler *handlers) getLogs(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return handler.findLogs(head, dispatch.Argument[Criteria](arguments, 0))
}
