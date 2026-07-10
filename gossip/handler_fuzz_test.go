// Copyright 2026 Sonic Operations Ltd
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
	"bytes"
	"errors"
	"io"
	"math/big"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
)

var fuzzMsgCodes = []uint64{
	HandshakeMsg,
	ProgressMsg,
	EvmTxsMsg,
	NewEvmTxHashesMsg,
	GetEvmTxsMsg,
	NewEventIDsMsg,
	GetEventsMsg,
	EventsMsg,
	RequestEventsStream,
	EventsStreamResponse,
	GetPeerInfosMsg,
	PeerInfosMsg,
	GetEndPointMsg,
	EndPointUpdateMsg,
}

func FuzzGossipHandlePeer(f *testing.F) {
	addHandlerMessageCorpus(f)

	// Note: this fuzzer has large memory requirements.
	// at the time of this message, one iteration requires 1.5 GiB of memory.
	//
	// To avoid OOM situations, use -parallel
	// > go test -fuzz FuzzGossipHandlePeer ./gossip/ -v -parallel=6
	f.Fuzz(func(t *testing.T, data []byte) {
		handler, err := makeFuzzedHandler(t)
		if err != nil {
			t.Fatalf("Failed to create fuzzed handler: %v", err)
		}

		msg, err := newFuzzMsg(data)
		if err != nil {
			t.Skip("input data is not a message, skip this run")
		}

		peerCfg := DefaultPeerCacheConfig(cachescale.Identity)
		input := &fuzzHandleReadWriter{
			networkID: handler.NetworkID,
			genesis:   common.Hash(handler.store.GetGenesisID()),
			version:   1,
			msg:       msg,
		}

		// Keep a Sonic-like name to avoid early rejection by isUseless.
		other := newPeer(1, p2p.NewPeer(randomID(), "Sonic/fuzz-peer", []p2p.Cap{}), input, peerCfg)

		// errors are ok, we are fuzzing for crash.
		_ = handler.handle(other)
	})
}

func makeFuzzedHandler(t *testing.T) (*handler, error) {
	const (
		genesisStakers = 3
		genesisBalance = 1e18
		genesisStake   = 2 * 4e6
	)

	upgrades := opera.GetSonicUpgrades()

	genStore := makefakegenesis.FakeGenesisStore(
		genesisStakers,
		utils.ToFtmU256(genesisBalance),
		utils.ToFtmU256(genesisStake),
		upgrades,
	)
	genesis := genStore.Genesis()

	store, err := NewMemStore(t)
	if err != nil {
		return nil, err
	}
	err = store.ApplyGenesis(genesis)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = store.Close() })

	var (
		heavyCheckReader    HeavyCheckReader
		gasPowerCheckReader GasPowerCheckReader
		proposalChecker     proposalCheckReader
	)

	mu := new(sync.RWMutex)

	chainId := big.NewInt(1234)
	txSigner := types.LatestSignerForChainID(chainId)
	config := DefaultConfig(cachescale.Identity)
	checkers := makeCheckers(config.HeavyCheck, txSigner, &heavyCheckReader, &gasPowerCheckReader, &proposalChecker, store)

	feed := new(ServiceFeed)
	chainconfig := opera.CreateTransientEvmChainConfig(
		1234,
		[]opera.UpgradeHeight{{
			Upgrades: upgrades,
			Height:   idx.Block(0),
		}},
		idx.Block(0),
	)
	txpool := evmcore.NewTxPool(
		evmcore.DefaultTxPoolConfig,
		chainconfig,
		&EvmStateReader{
			ServiceFeed: feed,
			store:       store,
		}, nil)
	t.Cleanup(txpool.Stop)

	h, err := newHandler(
		handlerConfig{
			config:   config,
			notifier: feed,
			txpool:   txpool,
			engineMu: mu,
			checkers: checkers,
			s:        store,
			process: processCallback{
				Event: func(event *inter.EventPayload) error {
					return nil
				},
			},
		})
	if err != nil {
		return nil, err
	}

	h.Start(3)
	t.Cleanup(h.Stop)
	return h, nil
}

func randomID() (id enode.ID) {
	for i := range id {
		id[i] = byte(rand.IntN(255))
	}
	return id
}

type fuzzMsgReadWriter struct {
	msg *p2p.Msg
}

type fuzzHandleReadWriter struct {
	networkID uint64
	genesis   common.Hash
	version   uint
	msg       *p2p.Msg

	mu        sync.Mutex
	readCount int
}

func addHandlerMessageCorpus(f *testing.F) {
	singleTx := makeSeedTx()
	singleHash := common.BytesToHash([]byte{1})
	singleEventID := hash.Event{1}
	nilTxEvent, okNilTxEvent := makeSeedEventSafe(nil)
	emptyTxEvent, okEmptyTxEvent := makeSeedEventSafe(types.Transactions{})
	singleTxEvent, okSingleTxEvent := makeSeedEventSafe(types.Transactions{singleTx})

	var nilTxs types.Transactions
	addCorpusInput(f, HandshakeMsg, &handshakeData{ProtocolVersion: 1, NetworkID: 1, Genesis: common.Hash{1}})
	addCorpusInput(f, ProgressMsg, &PeerProgress{})
	addCorpusInput(f, EvmTxsMsg, nilTxs)
	addCorpusInput(f, EvmTxsMsg, types.Transactions{})
	addCorpusInput(f, EvmTxsMsg, types.Transactions{singleTx})

	var nilTxHashes []common.Hash
	addCorpusInput(f, NewEvmTxHashesMsg, nilTxHashes)
	addCorpusInput(f, NewEvmTxHashesMsg, []common.Hash{})
	addCorpusInput(f, NewEvmTxHashesMsg, []common.Hash{singleHash})
	addCorpusInput(f, GetEvmTxsMsg, nilTxHashes)
	addCorpusInput(f, GetEvmTxsMsg, []common.Hash{})
	addCorpusInput(f, GetEvmTxsMsg, []common.Hash{singleHash})

	var nilEventIDs hash.Events
	addCorpusInput(f, NewEventIDsMsg, nilEventIDs)
	addCorpusInput(f, NewEventIDsMsg, hash.Events{})
	addCorpusInput(f, NewEventIDsMsg, hash.Events{singleEventID})
	addCorpusInput(f, GetEventsMsg, nilEventIDs)
	addCorpusInput(f, GetEventsMsg, hash.Events{})
	addCorpusInput(f, GetEventsMsg, hash.Events{singleEventID})

	var nilEvents inter.EventPayloads
	addCorpusInput(f, EventsMsg, nilEvents)
	addCorpusInput(f, EventsMsg, inter.EventPayloads{})
	if okNilTxEvent {
		addCorpusInput(f, EventsMsg, inter.EventPayloads{nilTxEvent})
	}
	if okEmptyTxEvent {
		addCorpusInput(f, EventsMsg, inter.EventPayloads{emptyTxEvent})
	}
	if okSingleTxEvent {
		addCorpusInput(f, EventsMsg, inter.EventPayloads{singleTxEvent})
	}

	addCorpusInput(f, RequestEventsStream, dagstream.Request{})
	addCorpusInput(f, RequestEventsStream, dagstream.Request{
		Session: dagstream.Session{ID: 1, Start: []byte{0x01}, Stop: []byte{0x02}},
		Type:    dagstream.RequestEvents,
		Limit:   dag.Metric{Num: 1, Size: 1024},
	})

	addCorpusInput(f, EventsStreamResponse, dagChunk{SessionID: 1, Done: true})
	addCorpusInput(f, EventsStreamResponse, dagChunk{SessionID: 1, IDs: hash.Events{singleEventID}})
	if okNilTxEvent {
		addCorpusInput(f, EventsStreamResponse, dagChunk{SessionID: 1, Events: inter.EventPayloads{nilTxEvent}})
	}
	if okSingleTxEvent {
		addCorpusInput(f, EventsStreamResponse, dagChunk{SessionID: 1, Events: inter.EventPayloads{singleTxEvent}})
	}

	f.Add(makeFuzzInput(GetPeerInfosMsg, nil))
	addCorpusInput(f, PeerInfosMsg, peerInfoMsg{Peers: []peerInfo{}})
	addCorpusInput(f, PeerInfosMsg, peerInfoMsg{Peers: []peerInfo{{Enode: "enode://invalid@127.0.0.1:30303"}}})
	f.Add(makeFuzzInput(GetEndPointMsg, nil))
	addCorpusInput(f, EndPointUpdateMsg, "")
	addCorpusInput(f, EndPointUpdateMsg, "enode://invalid@127.0.0.1:30303")
}

func makeSeedTx() *types.Transaction {
	var to common.Address
	to[0] = 1
	return types.NewTx(&types.LegacyTx{
		Nonce:    1,
		To:       &to,
		Value:    big.NewInt(1),
		Gas:      21_000,
		GasPrice: big.NewInt(1),
	})
}

func makeSeedEvent(txs types.Transactions) *inter.EventPayload {
	b := inter.MutableEventPayload{}
	b.SetEpoch(1)
	b.SetCreator(1)
	b.SetLamport(1)
	b.SetSeq(1)
	b.SetCreationTime(1)
	b.SetTxs(txs)
	return b.Build()
}

func makeSeedEventSafe(txs types.Transactions) (event *inter.EventPayload, ok bool) {
	defer func() {
		if recover() != nil {
			event = nil
			ok = false
		}
	}()
	return makeSeedEvent(txs), true
}

func addCorpusInput(f *testing.F, code uint64, payload interface{}) {
	if input, ok := fuzzInput(code, payload); ok {
		f.Add(input)
	}
}

func fuzzInput(code uint64, payload interface{}) ([]byte, bool) {
	if payload == nil {
		return makeFuzzInput(code, nil), true
	}
	defer func() {
		_ = recover()
	}()
	encoded, err := rlp.EncodeToBytes(payload)
	if err != nil {
		return nil, false
	}
	return makeFuzzInput(code, encoded), true
}

func makeFuzzInput(code uint64, payload []byte) []byte {
	selector := byte(0)
	for i, c := range fuzzMsgCodes {
		if c == code {
			selector = byte(i)
			break
		}
	}
	out := make([]byte, 1+len(payload))
	out[0] = selector
	copy(out[1:], payload)
	return out
}

func newFuzzMsg(data []byte) (*p2p.Msg, error) {
	if len(data) < 1 {
		return nil, errors.New("empty data")
	}

	code := fuzzMsgCodes[int(data[0])%len(fuzzMsgCodes)]
	data = data[1:]

	return &p2p.Msg{
		Code:    code,
		Size:    uint32(len(data)),
		Payload: bytes.NewReader(data),
	}, nil
}

func (rw *fuzzMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	return *rw.msg, nil
}

func (rw *fuzzMsgReadWriter) WriteMsg(p2p.Msg) error {
	return nil
}

func (rw *fuzzHandleReadWriter) ReadMsg() (p2p.Msg, error) {
	rw.mu.Lock()
	n := rw.readCount
	rw.readCount++
	rw.mu.Unlock()

	if n == 0 {
		encoded, err := rlp.EncodeToBytes(&handshakeData{
			ProtocolVersion: uint32(rw.version),
			NetworkID:       rw.networkID,
			Genesis:         rw.genesis,
		})
		if err != nil {
			return p2p.Msg{}, err
		}
		return p2p.Msg{
			Code:    HandshakeMsg,
			Size:    uint32(len(encoded)),
			Payload: bytes.NewReader(encoded),
		}, nil
	}

	if n == 1 {
		return *rw.msg, nil
	}

	return p2p.Msg{}, io.EOF
}

func (rw *fuzzHandleReadWriter) WriteMsg(p2p.Msg) error {
	return nil
}
