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

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

type handlers struct {
	environment *exec.Environment
	logIndex    core.LogIndex
}

// Register adds the methods of the package under the given namespace name
// (eth or ftm).
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, logIndex core.LogIndex, namespace string) {
	handler := &handlers{environment: environment, logIndex: logIndex}
	dispatcher.Add(namespace+"_getLogs", handler.getLogs, dispatch.Required[Criteria]("criteria"))
}

func (handler *handlers) getLogs(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return handler.findLogs(head, dispatch.Argument[Criteria](arguments, 0))
}
