// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gossip

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/evmcore"
)

// enrEntry is the ENR entry which advertises `eth` protocol on the discovery.
type enrEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e enrEntry) ENRKey() string {
	return "opera"
}

// StartENRUpdater starts the `opera` ENR updater loop, which listens for chain
// head events and updates the requested node record whenever a fork is passed.
func StartENRUpdater(svc *Service, ln *enode.LocalNode) {
	var newHead = make(chan evmcore.ChainHeadNotify, 10)
	sub := svc.feed.SubscribeNewBlock(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case head := <-newHead:
				ln.Set(currentENREntry(
					svc,
					idx.Block(head.Block.Number.Uint64()),
					uint64(head.Block.Time.Unix()),
				))
			case <-sub.Err():
				// Would be nice to sync with Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

// currentENREntry constructs an `eth` ENR entry based on the current state of the chain.
func currentENREntry(svc *Service, blockHeigh idx.Block, time uint64) *enrEntry {
	genesisHash := *svc.store.GetGenesisID()
	genesisTime := svc.store.GetGenesisTime()
	return &enrEntry{
		ForkID: forkid.NewId(
			svc.store.GetEvmChainConfig(blockHeigh),
			common.Hash(genesisHash),
			uint64(genesisTime.Unix()),
			uint64(svc.store.GetLatestBlockIndex()),
			time),
	}
}
