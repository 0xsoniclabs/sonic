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

// Package net serves the net namespace of the RPC module.
package net

import (
	"context"
	"strconv"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Node is what the namespace asks the node.
type Node interface {
	PeerCount() int
}

type handlers struct {
	environment *exec.Environment
	node        Node
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node) {
	h := &handlers{environment: environment, node: node}
	dispatch.Add(dispatcher, "net_version", h.version)
	dispatch.Add(dispatcher, "net_listening", listening)
	dispatch.Add(dispatcher, "net_peerCount", h.peerCount)
}

// version is the network id, which is the chain id.
func (h *handlers) version(context.Context, struct{}) (string, error) {
	return strconv.FormatUint(h.environment.ChainID().Uint64(), 10), nil
}

// listening answers true: the node always listens for peers.
func listening(context.Context, struct{}) (bool, error) {
	return true, nil
}

func (h *handlers) peerCount(context.Context, struct{}) (hexutil.Uint, error) {
	return hexutil.Uint(h.node.PeerCount()), nil
}
