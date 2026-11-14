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

// Package testonly serves the test namespace of the RPC module: methods for
// the integration tests of the node, which do their work only while the node
// enables them.
package testonly

import (
	"context"
	"errors"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type handlers struct {
	enabled bool
	node    core.TestOnly
}

// Register adds the methods of the namespace to the dispatcher, whether the
// node enables them or not.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node core.TestOnly) {
	h := &handlers{enabled: environment.Config().EnableTestOnlyApi, node: node}
	dispatch.Add(dispatcher, "test_proposeTransactions", h.proposeTransactions)
	dispatch.Add(dispatcher, "test_addTransactions", h.addTransactions)
}

// transactionsParams holds RLP-encoded transactions, each in base64 as JSON
// encodes bytes, as the legacy handlers take them.
type transactionsParams struct {
	Transactions [][]byte
}

// proposeTransactions has the node, a validator, emit the transactions in a
// new event as they are, without validating them.
func (h *handlers) proposeTransactions(_ context.Context, params transactionsParams) (any, error) {
	return nil, h.pass(params, h.node.EmitTransactions)
}

// addTransactions puts the transactions into the pool in one call, so that
// the emitter sees them together; the pool validates and orders them.
func (h *handlers) addTransactions(_ context.Context, params transactionsParams) (any, error) {
	return nil, h.pass(params, h.node.SendTransactions)
}

func (h *handlers) pass(params transactionsParams, to func(types.Transactions) error) error {
	if !h.enabled {
		return errors.New("test-only API is not enabled")
	}
	transactions := make(types.Transactions, len(params.Transactions))
	for i, encoded := range params.Transactions {
		transactions[i] = new(types.Transaction)
		if err := rlp.DecodeBytes(encoded, transactions[i]); err != nil {
			return err
		}
	}
	return to(transactions)
}
