// Copyright (c) 2024 Sonic Labs
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

package gossip

import (
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/gossip/pb"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
	"github.com/0xsoniclabs/sonic/inter"

	"github.com/Fantom-foundation/lachesis-base/gossip/basestream"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
)

// marshalHandshake serializes a handshakeData into protobuf wire format.
func marshalHandshake(h *handshakeData) ([]byte, error) {
	return proto.Marshal(&pb.Handshake{
		ProtocolVersion: h.ProtocolVersion,
		NetworkId:       h.NetworkID,
		Genesis:         h.Genesis[:],
	})
}

// unmarshalHandshake deserializes a handshakeData from protobuf wire format.
func unmarshalHandshake(b []byte) (*handshakeData, error) {
	var m pb.Handshake
	if err := proto.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	var genesis common.Hash
	copy(genesis[:], m.Genesis)
	return &handshakeData{
		ProtocolVersion: m.ProtocolVersion,
		NetworkID:       m.NetworkId,
		Genesis:         genesis,
	}, nil
}

// marshalProgress serializes a PeerProgress into protobuf wire format.
func marshalProgress(p PeerProgress) ([]byte, error) {
	return proto.Marshal(&pb.PeerProgress{
		Epoch:            uint32(p.Epoch),
		LastBlockIdx:     uint64(p.LastBlockIdx),
		LastBlockAtropos: p.LastBlockAtropos.Bytes(),
		HighestLamport:   uint32(p.HighestLamport),
	})
}

// unmarshalProgress deserializes a PeerProgress from protobuf wire format.
func unmarshalProgress(b []byte) (PeerProgress, error) {
	var m pb.PeerProgress
	if err := proto.Unmarshal(b, &m); err != nil {
		return PeerProgress{}, err
	}
	var atropos hash.Event
	atropos.SetBytes(m.LastBlockAtropos)
	return PeerProgress{
		Epoch:            idx.Epoch(m.Epoch),
		LastBlockIdx:     idx.Block(m.LastBlockIdx),
		LastBlockAtropos: atropos,
		HighestLamport:   idx.Lamport(m.HighestLamport),
	}, nil
}

// marshalTransactions serializes a batch of transactions into protobuf wire format.
func marshalTransactions(txs types.Transactions) ([]byte, error) {
	encoded := make([][]byte, len(txs))
	for i, tx := range txs {
		b, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("tx %d: %w", i, err)
		}
		encoded[i] = b
	}
	return proto.Marshal(&pb.Transactions{Txs: encoded})
}

// unmarshalTransactions deserializes a batch of transactions from protobuf wire format.
func unmarshalTransactions(b []byte) (types.Transactions, error) {
	var m pb.Transactions
	if err := proto.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	txs := make(types.Transactions, len(m.Txs))
	for i, raw := range m.Txs {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(raw); err != nil {
			return nil, fmt.Errorf("tx %d: %w", i, err)
		}
		txs[i] = tx
	}
	return txs, nil
}

// marshalHashes serializes a slice of common.Hash into protobuf wire format.
func marshalHashes(hashes []common.Hash) ([]byte, error) {
	encoded := make([][]byte, len(hashes))
	for i, h := range hashes {
		encoded[i] = h.Bytes()
	}
	return proto.Marshal(&pb.Hashes{Hashes: encoded})
}

// unmarshalHashes deserializes a slice of common.Hash from protobuf wire format.
func unmarshalHashes(b []byte) ([]common.Hash, error) {
	var m pb.Hashes
	if err := proto.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	hashes := make([]common.Hash, len(m.Hashes))
	for i, raw := range m.Hashes {
		hashes[i] = common.BytesToHash(raw)
	}
	return hashes, nil
}

// marshalEventIDs serializes a slice of event IDs into protobuf wire format.
func marshalEventIDs(ids hash.Events) ([]byte, error) {
	encoded := make([][]byte, len(ids))
	for i, id := range ids {
		encoded[i] = id.Bytes()
	}
	return proto.Marshal(&pb.EventIDs{Ids: encoded})
}

// unmarshalEventIDs deserializes a slice of event IDs from protobuf wire format.
func unmarshalEventIDs(b []byte) (hash.Events, error) {
	var m pb.EventIDs
	if err := proto.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	ids := make(hash.Events, len(m.Ids))
	for i, raw := range m.Ids {
		ids[i] = hash.BytesToEvent(raw)
	}
	return ids, nil
}

// marshalEvents serializes event payloads into protobuf wire format.
func marshalEvents(events inter.EventPayloads) ([]byte, error) {
	encoded := make([][]byte, len(events))
	for i, e := range events {
		b, err := e.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
		encoded[i] = b
	}
	return proto.Marshal(&pb.Events{Events: encoded})
}

// unmarshalEvents deserializes event payloads from protobuf wire format.
func unmarshalEvents(b []byte) (inter.EventPayloads, error) {
	var m pb.Events
	if err := proto.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	events := make(inter.EventPayloads, len(m.Events))
	for i, raw := range m.Events {
		e := &inter.EventPayload{}
		if err := e.UnmarshalBinary(raw); err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
		events[i] = e
	}
	return events, nil
}

// marshalEventsRaw marshals pre-serialized event blobs (from store) into protobuf.
// The blobs are treated as opaque bytes.
func marshalEventsRaw(events [][]byte) ([]byte, error) {
	return proto.Marshal(&pb.Events{Events: events})
}

// marshalStreamRequest serializes a dagstream.Request into protobuf wire format.
func marshalStreamRequest(r dagstream.Request) ([]byte, error) {
	return proto.Marshal(&pb.StreamRequest{
		SessionId:    r.Session.ID,
		SessionStart: []byte(r.Session.Start),
		SessionStop:  []byte(r.Session.Stop),
		LimitNum:     uint64(r.Limit.Num),
		LimitSize:    r.Limit.Size,
		Type:         uint32(r.Type),
		MaxChunks:    r.MaxChunks,
	})
}

// unmarshalStreamRequest deserializes a dagstream.Request from protobuf wire format.
func unmarshalStreamRequest(b []byte) (dagstream.Request, error) {
	var m pb.StreamRequest
	if err := proto.Unmarshal(b, &m); err != nil {
		return dagstream.Request{}, err
	}
	return dagstream.Request{
		Session: dagstream.Session{
			ID:    m.SessionId,
			Start: dagstream.Locator(m.SessionStart),
			Stop:  dagstream.Locator(m.SessionStop),
		},
		Limit: dag.Metric{
			Num:  idx.Event(m.LimitNum),
			Size: m.LimitSize,
		},
		Type:      basestream.RequestType(m.Type),
		MaxChunks: m.MaxChunks,
	}, nil
}

// marshalStreamResponseRaw marshals a dagstream.Response (with pre-serialized [][]byte events).
func marshalStreamResponseRaw(r dagstream.Response) ([]byte, error) {
	ids := make([][]byte, len(r.IDs))
	for i, id := range r.IDs {
		ids[i] = id.Bytes()
	}
	return proto.Marshal(&pb.StreamResponse{
		SessionId: r.SessionID,
		Done:      r.Done,
		Ids:       ids,
		Events:    r.Events,
	})
}

// marshalStreamResponse serializes a dagChunk into protobuf wire format.
func marshalStreamResponse(chunk dagChunk) ([]byte, error) {
	ids := make([][]byte, len(chunk.IDs))
	for i, id := range chunk.IDs {
		ids[i] = id.Bytes()
	}
	events := make([][]byte, len(chunk.Events))
	for i, e := range chunk.Events {
		b, err := e.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("event %d: %w", i, err)
		}
		events[i] = b
	}
	return proto.Marshal(&pb.StreamResponse{
		SessionId: chunk.SessionID,
		Done:      chunk.Done,
		Ids:       ids,
		Events:    events,
	})
}

// unmarshalStreamResponse deserializes a dagChunk from protobuf wire format.
func unmarshalStreamResponse(b []byte) (dagChunk, error) {
	var m pb.StreamResponse
	if err := proto.Unmarshal(b, &m); err != nil {
		return dagChunk{}, err
	}
	ids := make(hash.Events, len(m.Ids))
	for i, raw := range m.Ids {
		ids[i] = hash.BytesToEvent(raw)
	}
	events := make(inter.EventPayloads, len(m.Events))
	for i, raw := range m.Events {
		e := &inter.EventPayload{}
		if err := e.UnmarshalBinary(raw); err != nil {
			return dagChunk{}, fmt.Errorf("event %d: %w", i, err)
		}
		events[i] = e
	}
	return dagChunk{
		SessionID: m.SessionId,
		Done:      m.Done,
		IDs:       ids,
		Events:    events,
	}, nil
}

// marshalPeerInfos serializes a peerInfoMsg into protobuf wire format.
func marshalPeerInfos(infos peerInfoMsg) ([]byte, error) {
	peers := make([]*pb.PeerInfoEntry, len(infos.Peers))
	for i, p := range infos.Peers {
		peers[i] = &pb.PeerInfoEntry{Enode: p.Enode}
	}
	return proto.Marshal(&pb.PeerInfos{Peers: peers})
}

// unmarshalPeerInfos deserializes a peerInfoMsg from protobuf wire format.
func unmarshalPeerInfos(b []byte) (peerInfoMsg, error) {
	var m pb.PeerInfos
	if err := proto.Unmarshal(b, &m); err != nil {
		return peerInfoMsg{}, err
	}
	peers := make([]peerInfo, len(m.Peers))
	for i, p := range m.Peers {
		peers[i] = peerInfo{Enode: p.Enode}
	}
	return peerInfoMsg{Peers: peers}, nil
}

// marshalEndPoint serializes an enode string into protobuf wire format.
func marshalEndPoint(enode string) ([]byte, error) {
	return proto.Marshal(&pb.EndPoint{Enode: enode})
}

// unmarshalEndPoint deserializes an enode string from protobuf wire format.
func unmarshalEndPoint(b []byte) (string, error) {
	var m pb.EndPoint
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	return m.Enode, nil
}
