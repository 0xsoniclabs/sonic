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

package gsignercache

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"
)

var (
	globalCache, _ = lru.New(40000)
)

type WlruCache struct {
	Cache *lru.Cache
}

func (w *WlruCache) Add(txid common.Hash, c CachedSender) {
	w.Cache.Add(txid, c)
}

func (w *WlruCache) Get(txid common.Hash) *CachedSender {
	ic, ok := w.Cache.Get(txid)
	if !ok {
		return nil
	}
	c := ic.(CachedSender)
	return &c
}

func Wrap(signer types.Signer) types.Signer {
	return WrapWithCachedSigner(signer, &WlruCache{globalCache})
}

type CachedSender struct {
	From   common.Address
	Signer types.Signer
}

type SenderCache interface {
	Add(txid common.Hash, c CachedSender)
	Get(txid common.Hash) *CachedSender
}

type CachedSigner struct {
	types.Signer
	cache SenderCache
}

func WrapWithCachedSigner(signer types.Signer, cache SenderCache) CachedSigner {
	return CachedSigner{
		Signer: signer,
		cache:  cache,
	}
}

func (cs CachedSigner) Equal(s2 types.Signer) bool {
	cs2, ok := s2.(CachedSigner)
	if ok {
		// unwrap the signer
		return cs.Signer.Equal(cs2.Signer)
	}
	return cs.Signer.Equal(s2)
}

func (cs CachedSigner) Sender(tx *types.Transaction) (common.Address, error) {
	// try to load the sender from the global cache
	cached := cs.cache.Get(tx.Hash())
	if cached != nil && cached.Signer.Equal(cs.Signer) {
		return cached.From, nil
	}
	from, err := cs.Signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	cs.cache.Add(tx.Hash(), CachedSender{
		From:   from,
		Signer: cs.Signer,
	})
	return from, nil
}
