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

// Package dag serves the dag namespace of the RPC module: the events of the
// node's DAG.
package dag

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Node is what the namespace asks the node's DAG.
type Node interface {
	LiveEpochHeads(epoch uint64) ([]common.Hash, error)
	FindEvents(epoch uint64, lamport uint32, prefix []byte) []common.Hash
	Event(id common.Hash) (*inter.Event, error)
	EventPayload(id common.Hash) (*inter.EventPayload, error)
}

type handlers struct {
	environment *exec.Environment
	node        Node
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node) {
	h := &handlers{environment: environment, node: node}
	dispatch.Add(dispatcher, "dag_getEvent", h.getEvent)
	dispatch.Add(dispatcher, "dag_getEventPayload", h.getEventPayload)
	dispatch.Add(dispatcher, "dag_getHeads", h.getHeads)
}

type eventParams struct {
	ID string
}

// getEvent answers the header of an event, named by its hash or short id.
func (h *handlers) getEvent(_ context.Context, params eventParams) (map[string]any, error) {
	event, err := lookUp(h, params.ID, h.node.Event)
	if err != nil {
		return nil, err
	}
	return inter.RPCMarshalEvent(event), nil
}

type payloadParams struct {
	ID                  string
	IncludeTransactions bool
}

// getEventPayload answers an event with its payload, its transactions as
// hashes or in full.
func (h *handlers) getEventPayload(_ context.Context, params payloadParams) (map[string]any, error) {
	payload, err := lookUp(h, params.ID, h.node.EventPayload)
	if err != nil {
		return nil, err
	}
	return inter.RPCMarshalEventPayload(payload, params.IncludeTransactions)
}

// lookUp reads an event, named by its hash or short id, with read.
func lookUp[Event any](h *handlers, name string, read func(common.Hash) (*Event, error)) (*Event, error) {
	id, err := h.eventID(name)
	if err != nil {
		return nil, err
	}
	event, err := exec.NoneIfMissing(read(id))
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, fmt.Errorf("event %s not found", name)
	}
	return event, nil
}

// eventID is the id an event's hash or short id names. A short id is
// "epoch:lamport:prefix", the prefix being the first bytes of the hash part
// of the id; it must name one event.
func (h *handlers) eventID(name string) (common.Hash, error) {
	parts := strings.Split(name, ":")
	if len(parts) == 1 {
		id, err := hexutil.Decode(name)
		if err != nil {
			return common.Hash{}, fmt.Errorf("full hash parsing error: %w", err)
		}
		return common.BytesToHash(id), nil
	}
	if len(parts) != 3 {
		return common.Hash{}, errors.New("incorrect format of short event ID (need Epoch:Lamport:Hash")
	}
	epoch, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return common.Hash{}, fmt.Errorf("short hash parsing error (lamport): %w", err)
	}
	lamport, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return common.Hash{}, fmt.Errorf("short hash parsing error (lamport): %w", err)
	}
	ids := h.node.FindEvents(epoch, uint32(lamport), common.FromHex(parts[2]))
	switch len(ids) {
	case 0:
		return common.Hash{}, errors.New("event not found by short ID")
	case 1:
		return ids[0], nil
	}
	return common.Hash{}, errors.New("there're multiple events with the same short ID, please use full ID")
}

type headsParams struct {
	Epoch chain.BlockNumber
}

// getHeads answers the events of an epoch without descendants, which the
// node keeps for the open epoch only.
func (h *handlers) getHeads(_ context.Context, params headsParams) ([]hexutil.Bytes, error) {
	epoch, err := h.environment.DAGEpoch(h.environment.Head(), params.Epoch)
	if err != nil {
		return nil, err
	}
	heads, err := h.node.LiveEpochHeads(epoch)
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("epoch %d: heads are not available", epoch)
	}
	if err != nil {
		return nil, err
	}
	encoded := make([]hexutil.Bytes, len(heads))
	for i, head := range heads {
		encoded[i] = head.Bytes()
	}
	return encoded, nil
}
