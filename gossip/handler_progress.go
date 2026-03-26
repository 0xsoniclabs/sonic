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

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// handleHandshakeMsg rejects handshake messages received after initial handshake.
func (h *handler) handleHandshakeMsg(p *peer, raw []byte) error {
	return errResp(ErrExtraStatusMsg, "uncontrolled status message")
}

// handleProgressMsg processes a progress update from a peer.
func (h *handler) handleProgressMsg(p *peer, raw []byte) error {
	progress, err := unmarshalProgress(raw)
	if err != nil {
		return errResp(ErrDecode, "%v", err)
	}
	p.SetProgress(progress)
	return nil
}

// myProgress returns this node's current synchronization progress.
func (h *handler) myProgress() PeerProgress {
	bs := h.store.GetBlockState()
	epoch := h.store.GetEpoch()
	return PeerProgress{
		Epoch:            epoch,
		LastBlockIdx:     bs.LastBlock.Idx,
		LastBlockAtropos: bs.LastBlock.Atropos,
	}
}

// highestPeerProgress returns the progress of the most advanced known peer.
func (h *handler) highestPeerProgress() PeerProgress {
	peers := h.peers.List()
	max := h.myProgress()
	for _, peer := range peers {
		peerProgress := peer.GetProgress()
		if max.LastBlockIdx < peerProgress.LastBlockIdx {
			max = peerProgress
		}
	}
	return max
}

// broadcastProgress sends the current progress to all connected peers.
func (h *handler) broadcastProgress() {
	progress := h.myProgress()
	for _, peer := range h.peers.List() {
		peer.AsyncSendProgress(progress, peer.queue)
	}
}

// progressBroadcastLoop periodically broadcasts progress to peers, but only
// when progress has actually changed since the last broadcast.
func (h *handler) progressBroadcastLoop() {
	ticker := time.NewTicker(h.config.Protocol.ProgressBroadcastPeriod)
	defer ticker.Stop()
	defer h.loopsWg.Done()
	var lastBroadcast PeerProgress
	for {
		select {
		case <-ticker.C:
			current := h.myProgress()
			if current != lastBroadcast {
				h.broadcastProgress()
				lastBroadcast = current
			}
		case <-h.quitProgressBradcast:
			return
		}
	}
}

// onNewEpochLoop handles epoch transitions by clearing the DAG processor and updating the leecher.
func (h *handler) onNewEpochLoop() {
	defer h.loopsWg.Done()
	for {
		select {
		case myEpoch := <-h.newEpochsCh:
			h.dagProcessor.Clear()
			h.dagLeecher.OnNewEpoch(myEpoch)
		// Err() channel will be closed when unsubscribing.
		case <-h.newEpochsSub.Err():
			return
		}
	}
}

// NodeInfo represents a short summary of the sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network     uint64      `json:"network"` // network ID
	Genesis     common.Hash `json:"genesis"` // SHA3 hash of the host's genesis object
	Epoch       idx.Epoch   `json:"epoch"`
	NumOfBlocks idx.Block   `json:"blocks"`
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (h *handler) NodeInfo() *NodeInfo {
	numOfBlocks := h.store.GetLatestBlockIndex()
	return &NodeInfo{
		Network:     h.NetworkID,
		Genesis:     common.Hash(h.store.GetGenesisID()),
		Epoch:       h.store.GetEpoch(),
		NumOfBlocks: numOfBlocks,
	}
}
