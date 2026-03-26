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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// handleGetPeerInfosMsg processes a request for peer information.
func (h *handler) handleGetPeerInfosMsg(p *peer, raw []byte) error {
	infos := []peerInfo{}
	for _, peer := range h.peers.List() {
		if peer.Useless() {
			continue
		}
		info := peer.endPoint.Load()
		if info == nil {
			continue
		}
		infos = append(infos, peerInfo{
			Enode: info.enode.String(),
		})
	}
	responseBytes, err := marshalPeerInfos(peerInfoMsg{Peers: infos})
	if err != nil {
		return err
	}
	if err := sendBytes(p.rw, PeerInfosMsg, responseBytes); err != nil {
		return err
	}
	return nil
}

// handlePeerInfosMsg processes a peer info response.
func (h *handler) handlePeerInfosMsg(p *peer, raw []byte) error {
	infos, err := unmarshalPeerInfos(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	reportedPeers := []*enode.Node{}
	for _, info := range infos.Peers {
		var enodeNode enode.Node
		if err := enodeNode.UnmarshalText([]byte(info.Enode)); err != nil {
			h.Log.Warn("Failed to unmarshal enode", "enode", info.Enode, "err", err)
		} else {
			reportedPeers = append(reportedPeers, &enodeNode)
		}
	}
	h.connectionAdvisor.UpdatePeers(p.ID(), reportedPeers)
	return nil
}

// handleGetEndPointMsg processes a request for this node's public endpoint.
func (h *handler) handleGetEndPointMsg(p *peer, raw []byte) error {
	source := h.localEndPointSource
	if source == nil {
		return nil
	}
	enodeNode := source.GetLocalEndPoint()
	if enodeNode == nil {
		return nil
	}
	responseBytes, err := marshalEndPoint(enodeNode.String())
	if err != nil {
		return err
	}
	if err := sendBytes(p.rw, EndPointUpdateMsg, responseBytes); err != nil {
		return err
	}
	return nil
}

// handleEndPointUpdateMsg processes an endpoint update from a peer.
func (h *handler) handleEndPointUpdateMsg(p *peer, raw []byte) error {
	encoded, err := unmarshalEndPoint(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	var enodeNode enode.Node
	if err := enodeNode.UnmarshalText([]byte(encoded)); err != nil {
		h.Log.Warn("Failed to unmarshal enode", "enode", encoded, "err", err)
	} else {
		p.endPoint.Store(&peerEndPointInfo{
			enode:     enodeNode,
			timestamp: time.Now(),
		})
	}
	return nil
}

// peerInfoCollectionLoop periodically collects peer info, manages network topology,
// and suggests new peer connections.
func (h *handler) peerInfoCollectionLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(h.config.Protocol.PeerInfoCollectionPeriod)
	defer ticker.Stop()
	defer h.loopsWg.Done()
	for {
		select {
		case <-ticker.C:
			// Get a suggestion for a new peer.
			suggestion := h.connectionAdvisor.GetNewPeerSuggestion()
			if suggestion != nil {
				select {
				case h.nextSuggestedPeer <- suggestion:
				default:
				}
			}

			// Request updated peer information from current peers.
			peers := h.peers.List()
			for _, peer := range peers {
				// If we do not have the peer's end-point or it is too old, request it.
				if info := peer.endPoint.Load(); info == nil || time.Since(info.timestamp) > h.config.Protocol.PeerEndPointUpdatePeriod {
					if err := peer.SendEndPointUpdateRequest(); err != nil {
						log.Warn("Failed to send end-point update request", "peer", peer.id, "err", err)
						// If the end-point update request fails, do not send the peer info request.
						continue
					}
				}
				if err := peer.SendPeerInfoRequest(); err != nil {
					log.Warn("Failed to send peer info request", "peer", peer.id, "err", err)
				}
			}

			// Drop a redundant connection if there are too many connections.
			if suggestion != nil && len(peers) >= h.maxPeers-1 {
				redundant := h.connectionAdvisor.GetRedundantPeerSuggestion()
				if redundant != nil {
					for _, peer := range peers {
						if peer.Node().ID() == *redundant {
							peer.Disconnect(p2p.DiscTooManyPeers)
							break
						}
					}
				}
			}
		case <-stop:
			return
		}
	}
}

// GetSuggestedPeerIterator returns an enode iterator that yields peer connection suggestions.
func (h *handler) GetSuggestedPeerIterator() enode.Iterator {
	return &suggestedPeerIterator{
		handler: h,
		close:   make(chan struct{}),
	}
}

// suggestedPeerIterator wraps the nextSuggestedPeer channel as an enode.Iterator.
type suggestedPeerIterator struct {
	handler *handler
	next    *enode.Node
	close   chan struct{}
}

// Next blocks until a new suggested peer is available or the iterator is closed.
func (i *suggestedPeerIterator) Next() bool {
	select {
	case i.next = <-i.handler.nextSuggestedPeer:
		return true
	case <-i.close:
		return false
	}
}

// Node returns the most recently suggested peer enode.
func (i *suggestedPeerIterator) Node() *enode.Node {
	return i.next
}

// Close stops the iterator and unblocks any pending Next call.
func (i *suggestedPeerIterator) Close() {
	close(i.close)
}
