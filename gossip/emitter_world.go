package gossip

import (
	"github.com/0xsoniclabs/consensus/dagindexer"
	"sync/atomic"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/utils/wgmutex"
	"github.com/0xsoniclabs/sonic/valkeystore"
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
	valkeystore.SignerI
	types.Signer
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

func (ew *emitterWorldProc) DagIndex() *dagindexer.Index {
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

func (ew *emitterWorldProc) IsSynced() bool {
	return ew.s.handler.syncStatus.AcceptEvents()
}

func (ew *emitterWorldProc) PeersNum() int {
	return ew.s.handler.peers.Len()
}

func (ew *emitterWorldRead) GetHeads(epoch consensus.Epoch) consensus.EventHashes {
	return ew.Store.GetHeadsSlice(epoch)
}

func (ew *emitterWorldRead) GetLastEvent(epoch consensus.Epoch, from consensus.ValidatorID) *consensus.EventHash {
	return ew.Store.GetLastEvent(epoch, from)
}

func (ew *emitterWorldRead) GetBlockEpoch(block consensus.BlockID) consensus.Epoch {
	return ew.Store.FindBlockEpoch(block)
}
