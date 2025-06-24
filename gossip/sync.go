// Copyright 2025 Sonic Operations Ltd
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

package gossip

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var isMaybeSyncedGauge = metrics.GetOrRegisterGauge("chain/maybeSynced", nil)

type syncStatus struct {
	maybeSynced uint32
}

func (ss *syncStatus) MaybeSynced() bool {
	return atomic.LoadUint32(&ss.maybeSynced) != 0
}

func (ss *syncStatus) MarkMaybeSynced() {
	atomic.StoreUint32(&ss.maybeSynced, uint32(1))
	isMaybeSyncedGauge.Update(int64(1))
}

func (ss *syncStatus) AcceptEvents() bool {
	return true
}

func (ss *syncStatus) AcceptBlockRecords() bool {
	return false
}

func (ss *syncStatus) AcceptTxs() bool {
	return ss.MaybeSynced()
}

func (ss *syncStatus) RequestLLR() bool {
	return ss.MaybeSynced()
}

type txsync struct {
	p     *peer
	txids []common.Hash
}

// syncTransactions starts sending all currently pending transactions to the given peer.
func (h *handler) syncTransactions(p *peer, txids []common.Hash) {
	if len(txids) == 0 {
		return
	}
	select {
	case h.txsyncCh <- &txsync{p, txids}:
	case <-h.quitSync:
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (h *handler) txsyncLoop() {
	var (
		pending = make(map[enode.ID]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		pack.p = s.p
		pack.txids = pack.txids[:0]
		for i := 0; i < len(s.txids) && len(pack.txids) < softLimitItems; i++ {
			pack.txids = append(pack.txids, s.txids[i])
		}
		// Remove the transactions that will be sent.
		s.txids = s.txids[len(pack.txids):]
		if len(s.txids) == 0 {
			delete(pending, s.p.ID())
		}
		// Send the pack in the background.
		s.p.Log().Trace("Sending batch of transaction hashes", "count", len(pack.txids))
		sending = true
		go func() {
			if len(pack.txids) != 0 {
				done <- pack.p.SendTransactionHashes(pack.txids)
			} else {
				done <- nil
			}
		}()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.IntN(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-h.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				pack.p.Log().Debug("Transaction send failed", "err", err)
				delete(pending, pack.p.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-h.quitSync:
			return
		}
	}
}
