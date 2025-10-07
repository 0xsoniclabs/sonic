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
	globalCache, _ = lru.New(100_000) // ~40 bytes per entry => ~4MB
)

type lruCache struct {
	cache *lru.Cache
}

func (w *lruCache) add(txid common.Hash, c cachedSender) {
	w.cache.Add(txid, c)
}

func (w *lruCache) get(txid common.Hash) *cachedSender {
	ic, ok := w.cache.Get(txid)
	if !ok {
		return nil
	}
	c := ic.(cachedSender)
	return &c
}

func Wrap(signer types.Signer) types.Signer {
	return WrapWithCachedSigner(signer, &lruCache{globalCache})
}

type cachedSender struct {
	from   common.Address
	signer types.Signer
}

type senderCache interface {
	add(txid common.Hash, c cachedSender)
	get(txid common.Hash) *cachedSender
}

type cachedSigner struct {
	types.Signer
	cache senderCache
}

func WrapWithCachedSigner(signer types.Signer, cache senderCache) cachedSigner {
	return cachedSigner{
		Signer: signer,
		cache:  cache,
	}
}

func (cs cachedSigner) Equal(s2 types.Signer) bool {
	cs2, ok := s2.(cachedSigner)
	if ok {
		// unwrap the signer
		return cs.Signer.Equal(cs2.Signer)
	}
	return cs.Signer.Equal(s2)
}

func (cs cachedSigner) Sender(tx *types.Transaction) (common.Address, error) {
	// try to load the sender from the global cache
	cached := cs.cache.get(tx.Hash())
	if cached != nil && cached.signer.Equal(cs.Signer) {
		return cached.from, nil
	}
	from, err := cs.Signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	cs.cache.add(tx.Hash(), cachedSender{
		from:   from,
		signer: cs.Signer,
	})
	return from, nil
}
