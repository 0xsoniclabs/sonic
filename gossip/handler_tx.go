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
	"math/rand/v2"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/utils/txtime"
)

// interfacesToTxids converts a slice of interface values to transaction hashes.
func interfacesToTxids(ids []interface{}) []common.Hash {
	res := make([]common.Hash, len(ids))
	for i, id := range ids {
		res[i] = id.(common.Hash)
	}
	return res
}

// txidsToInterfaces converts a slice of transaction hashes to interface values.
func txidsToInterfaces(ids []common.Hash) []interface{} {
	res := make([]interface{}, len(ids))
	for i, id := range ids {
		res[i] = id
	}
	return res
}

// handleEvmTxsMsg processes full transactions received from a peer.
func (h *handler) handleEvmTxsMsg(p *peer, raw []byte) error {
	// Transactions arrived, make sure we have a valid and fresh graph to handle them
	if !h.syncStatus.AcceptTxs() {
		return nil
	}
	// Transactions can be processed, parse all of them and deliver to the pool
	txs, err := unmarshalTransactions(raw)
	if err != nil {
		return errResp(ErrDecode, "msg: %v", err)
	}
	if err := checkLenLimits(len(txs), txs); err != nil {
		return err
	}
	txids := make([]interface{}, txs.Len())
	for i, tx := range txs {
		txids[i] = tx.Hash()
	}
	_ = h.txFetcher.NotifyReceived(txids)
	h.handleTxs(p, txs)
	return nil
}

// handleNewEvmTxHashesMsg processes transaction hash announcements from a peer.
func (h *handler) handleNewEvmTxHashesMsg(p *peer, raw []byte) error {
	// Transactions arrived, make sure we have a valid and fresh graph to handle them
	if !h.syncStatus.AcceptTxs() {
		return nil
	}
	// Transactions can be processed, parse all of them and deliver to the pool
	txHashes, err := unmarshalHashes(raw)
	if err != nil {
		return errResp(ErrDecode, "msg: %v", err)
	}
	if err := checkLenLimits(len(txHashes), txHashes); err != nil {
		return err
	}
	h.handleTxHashes(p, txHashes)
	return nil
}

// handleGetEvmTxsMsg processes a request for transactions by hash.
func (h *handler) handleGetEvmTxsMsg(p *peer, raw []byte) error {
	requests, err := unmarshalHashes(raw)
	if err != nil {
		return errResp(ErrDecode, "msg: %v", err)
	}
	if err := checkLenLimits(len(requests), requests); err != nil {
		return err
	}
	txs := make(types.Transactions, 0, len(requests))
	for _, txid := range requests {
		tx := h.txpool.Get(txid)
		if tx == nil {
			continue
		}
		txs = append(txs, tx)
	}
	SplitTransactions(txs, func(batch types.Transactions) {
		p.EnqueueSendTransactions(batch, p.queue)
	})
	return nil
}

// handleTxHashes marks transaction hashes as known at the peer and schedules retrieval.
func (h *handler) handleTxHashes(p *peer, announces []common.Hash) {
	// Mark the hashes as present at the remote node
	now := time.Now()
	for _, id := range announces {
		txtime.Saw(id, now)
		p.MarkTransaction(id)
	}
	// Schedule all the unknown hashes for retrieval
	requestTransactions := func(ids []interface{}) error {
		return p.RequestTransactions(interfacesToTxids(ids))
	}
	_ = h.txFetcher.NotifyAnnounces(p.id, txidsToInterfaces(announces), time.Now(), requestTransactions)
}

// handleTxs marks transactions as known at the peer and adds them to the pool.
func (h *handler) handleTxs(p *peer, txs types.Transactions) {
	// Mark the hashes as present at the remote node
	now := time.Now()
	for _, tx := range txs {
		txid := tx.Hash()
		txtime.Saw(txid, now)
		p.MarkTransaction(txid)
	}
	h.txpool.AddRemotes(txs)
}

// BroadcastTxs will propagate a batch of transactions to all peers which are not known to
// already have the given transaction.
func (h *handler) BroadcastTxs(txs types.Transactions) {
	broadcastedTxsCounter.Inc(int64(txs.Len()))
	var txset = make(map[*peer]types.Transactions)

	// Broadcast transactions to a batch of peers not knowing about it
	totalSize := common.StorageSize(0)
	for _, tx := range txs {
		peers := h.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			txset[peer] = append(txset[peer], tx)
		}
		totalSize += common.StorageSize(tx.Size())
		log.Trace("Broadcast transaction", "hash", tx.Hash(), "recipients", len(peers))
	}

	fullRecipients := h.decideBroadcastAggressiveness(int(totalSize), time.Second, len(txset))
	i := 0
	for peer, txs := range txset {
		SplitTransactions(txs, func(batch types.Transactions) {
			if i < fullRecipients {
				peer.AsyncSendTransactions(batch, peer.queue)
			} else {
				txids := make([]common.Hash, batch.Len())
				for i, tx := range batch {
					txids[i] = tx.Hash()
				}
				peer.AsyncSendTransactionHashes(txids, peer.queue)
			}
		})
		i++
	}
}

// txBroadcastLoop listens for new transactions and periodically sends random hashes to random peers.
func (h *handler) txBroadcastLoop() {
	basePeriod := h.config.Protocol.RandomTxHashesSendPeriod
	ticker := time.NewTicker(basePeriod)
	defer ticker.Stop()
	defer h.loopsWg.Done()
	for {
		select {
		case notify := <-h.txsCh:
			h.BroadcastTxs(notify.Txs)
		// Err() channel will be closed when unsubscribing.
		case <-h.txsSub.Err():
			return
		case <-ticker.C:
			if !h.syncStatus.AcceptTxs() {
				break
			}
			peers := h.peers.List()
			if len(peers) == 0 {
				continue
			}
			// Adaptive: scale period with peer count to reduce redundant broadcasts
			peerCount := len(peers)
			adaptivePeriod := basePeriod * time.Duration(max(1, peerCount/10))
			if adaptivePeriod > 5*basePeriod {
				adaptivePeriod = 5 * basePeriod
			}
			ticker.Reset(adaptivePeriod)
			randPeer := peers[rand.IntN(peerCount)]
			h.syncTransactions(randPeer, h.txpool.SampleHashes(h.config.Protocol.MaxRandomTxHashesSend))
		}
	}
}
