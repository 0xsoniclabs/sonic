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
	"math/big"
	"math/rand/v2"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"

	"github.com/0xsoniclabs/sonic/evmcore"
)

// dummyTxPool is a fake, helper transaction pool for testing purposes
type dummyTxPool struct {
	txFeed notify.Feed
	pool   []*types.Transaction        // Collection of all transactions
	added  chan<- []*types.Transaction // Notification channel for new transactions

	signer types.Signer

	lock sync.RWMutex
}

// AddRemotes appends a batch of transactions to the pool, and notifies any
// listeners if the addition channel is non nil
func (p *dummyTxPool) AddRemotes(txs []*types.Transaction) []error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.pool = append(p.pool, txs...)
	if p.added != nil {
		p.added <- txs
	}
	return make([]error, len(txs))
}

func (p *dummyTxPool) AddLocals(txs []*types.Transaction) []error {
	return p.AddRemotes(txs)
}

func (p *dummyTxPool) AddLocal(tx *types.Transaction) error {
	return p.AddLocals([]*types.Transaction{tx})[0]
}

func (p *dummyTxPool) Nonce(addr common.Address) uint64 {
	return 0
}

func (p *dummyTxPool) Stats() (int, int) {
	return p.Count(), 0
}

func (p *dummyTxPool) Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return nil, nil
}

func (p *dummyTxPool) ContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return nil, nil
}

// Pending returns all the transactions known to the pool
func (p *dummyTxPool) Pending(enforceTips bool) (map[common.Address]types.Transactions, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	batches := make(map[common.Address]types.Transactions)
	for _, tx := range p.pool {
		from, _ := types.Sender(p.signer, tx)
		batches[from] = append(batches[from], tx)
	}
	for _, batch := range batches {
		sort.Sort(types.TxByNonce(batch))
	}
	return batches, nil
}

func (p *dummyTxPool) MinTip() *big.Int {
	return big.NewInt(0)
}

func (p *dummyTxPool) SubscribeNewTxsNotify(ch chan<- evmcore.NewTxsNotify) notify.Subscription {
	return p.txFeed.Subscribe(ch)
}

func (p *dummyTxPool) Map() map[common.Hash]*types.Transaction {
	p.lock.RLock()
	defer p.lock.RUnlock()
	res := make(map[common.Hash]*types.Transaction, len(p.pool))
	for _, tx := range p.pool {
		res[tx.Hash()] = tx
	}
	return nil
}

func (p *dummyTxPool) Get(txid common.Hash) *types.Transaction {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, tx := range p.pool {
		if tx.Hash() == txid {
			return tx
		}
	}
	return nil
}

func (p *dummyTxPool) Has(txid common.Hash) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for _, tx := range p.pool {
		if tx.Hash() == txid {
			return true
		}
	}
	return false
}

func (p *dummyTxPool) OnlyNotExisting(txids []common.Hash) []common.Hash {
	m := p.Map()
	notExisting := make([]common.Hash, 0, len(txids))
	for _, txid := range txids {
		if m[txid] == nil {
			notExisting = append(notExisting, txid)
		}
	}
	return notExisting
}

func (p *dummyTxPool) SampleHashes(max int) []common.Hash {
	p.lock.RLock()
	defer p.lock.RUnlock()
	res := make([]common.Hash, 0, max)
	skip := 0
	if len(p.pool) > max {
		skip = rand.IntN(len(p.pool) - max)
	}
	for _, tx := range p.pool {
		if len(res) >= max {
			break
		}
		if skip > 0 {
			skip--
			continue
		}
		res = append(res, tx.Hash())
	}
	return res
}

func (p *dummyTxPool) Count() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return len(p.pool)
}

func (p *dummyTxPool) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.pool) != 0 {
		p.pool = p.pool[:0]
	}
}

func (p *dummyTxPool) Delete(needle common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.pool) == 0 {
		return
	}
	notErased := make([]*types.Transaction, 0, len(p.pool)-1)
	for _, tx := range p.pool {
		if tx.Hash() != needle {
			notErased = append(notErased, tx)
		}
	}
	p.pool = notErased
}

func (p *dummyTxPool) Stop() {}
