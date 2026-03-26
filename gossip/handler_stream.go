// Copyright 2024 The Sonic Authors
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

package gossip

import (
	"errors"

	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream/dagstreamseeder"
	"github.com/Fantom-foundation/lachesis-base/hash"
)

// handleRequestEventsStream processes a DAG stream request from a peer.
func (h *handler) handleRequestEventsStream(p *peer, raw []byte) error {
	request, err := unmarshalStreamRequest(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	if request.Limit.Num > hardLimitItems-1 {
		return errResp(ErrMsgTooLarge, "request limit too large")
	}
	if request.Limit.Size > protocolMaxMsgSize*2/3 {
		return errResp(ErrMsgTooLarge, "request size too large")
	}
	pid := p.id
	_, peerErr := h.dagSeeder.NotifyRequestReceived(dagstreamseeder.Peer{
		ID:        pid,
		SendChunk: p.SendEventsStream,
		Misbehaviour: func(err error) {
			h.peerMisbehaviour(pid, err)
		},
	}, request)
	if peerErr != nil {
		return peerErr
	}
	return nil
}

// handleEventsStreamResponse processes a DAG stream response chunk from a peer.
func (h *handler) handleEventsStreamResponse(p *peer, raw []byte) error {
	chunk, err := unmarshalStreamResponse(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	if err := checkLenLimits(len(chunk.Events)+len(chunk.IDs)+1, chunk); err != nil {
		return err
	}
	if (len(chunk.Events) != 0) && (len(chunk.IDs) != 0) {
		return errors.New("expected either events or event hashes")
	}
	var last hash.Event
	if len(chunk.IDs) != 0 {
		h.handleEventHashes(p, chunk.IDs)
		last = chunk.IDs[len(chunk.IDs)-1]
	}
	if len(chunk.Events) != 0 {
		h.handleEvents(p, chunk.Events.Bases(), true)
		last = chunk.Events[len(chunk.Events)-1].ID()
	}
	_ = h.dagLeecher.NotifyChunkReceived(chunk.SessionID, last, chunk.Done)
	return nil
}
