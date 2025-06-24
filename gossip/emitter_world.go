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
	"sync/atomic"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils/wgmutex"
	"github.com/0xsoniclabs/sonic/vecmt"
)

type emitterWorldProc struct {
	s *Service
}

type emitterWorldRead struct {
	*Store
}

// emitterWorld implements emitter.World interface
type emitterWorld struct {
	emitterWorldProc
	emitterWorldRead
	*wgmutex.WgMutex
	emitter.TxPool
}

func (ew *emitterWorldProc) Check(emitted *inter.EventPayload, parents inter.Events) error {
	// sanity check
	return ew.s.checkers.Validate(emitted, parents.Interfaces())
}

func (ew *emitterWorldProc) Process(emitted *inter.EventPayload) error {
	done := ew.s.procLogger.EventConnectionStarted(emitted, true)
	defer done()
	return ew.s.processEvent(emitted)
}

func (ew *emitterWorldProc) Broadcast(emitted *inter.EventPayload) {
	// PM listens and will broadcast it
	ew.s.feed.newEmittedEvent.Send(emitted)
}

func (ew *emitterWorldProc) Build(e *inter.MutableEventPayload, onIndexed func()) error {
	return ew.s.buildEvent(e, onIndexed)
}

func (ew *emitterWorldProc) DagIndex() *vecmt.Index {
	return ew.s.dagIndexer
}

func (ew *emitterWorldProc) IsBusy() bool {
	return atomic.LoadUint32(&ew.s.eventBusyFlag) != 0 || atomic.LoadUint32(&ew.s.blockBusyFlag) != 0
}

func (ew *emitterWorldProc) StateDB() state.StateDB {
	statedb, err := ew.s.store.evm.GetTxPoolStateDB()
	if err != nil {
		return nil
	}
	return statedb
}

func (ew *emitterWorldProc) GetUpgradeHeights() []opera.UpgradeHeight {
	return ew.s.store.GetUpgradeHeights()
}

func (ew *emitterWorldProc) GetHeader(h common.Hash, number uint64) *evmcore.EvmHeader {
	reader := &EvmStateReader{
		store: ew.s.store,
	}
	return reader.GetHeader(h, number)
}

func (ew *emitterWorldProc) IsSynced() bool {
	return ew.s.handler.syncStatus.AcceptEvents()
}

func (ew *emitterWorldProc) PeersNum() int {
	return ew.s.handler.peers.Len()
}

func (ew *emitterWorldRead) GetHeads(epoch idx.Epoch) hash.Events {
	return ew.GetHeadsSlice(epoch)
}

func (ew *emitterWorldRead) GetLastEvent(epoch idx.Epoch, from idx.ValidatorID) *hash.Event {
	return ew.Store.GetLastEvent(epoch, from)
}

func (ew *emitterWorldRead) GetBlockEpoch(block idx.Block) idx.Epoch {
	return ew.FindBlockEpoch(block)
}
