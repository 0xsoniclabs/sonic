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
	"bytes"
	"errors"
	"math/big"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
)

func FuzzGossipHandler(f *testing.F) {

	// Note: this fuzzer has large memory requirements.
	// at the time of this message, one iteration requires 1.5 GiB of memory.
	//
	// To avoid OOM situations, use -parallel
	// > go test -fuzz FuzzGossipHandler ./gossip/ -v -parallel=6
	f.Fuzz(func(t *testing.T, data []byte) {
		handler, err := makeFuzzedHandler(t)
		if err != nil {
			t.Fatalf("Failed to create fuzzed handler: %v", err)
		}

		msg, err := newFuzzMsg(data)
		if err != nil {
			t.Skip("input data is not a message, skip this run")
		}

		input := &fuzzMsgReadWriter{msg}
		other := &peer{
			version: 1,
			Peer:    p2p.NewPeer(randomID(), "fake-node-1", []p2p.Cap{}),
			rw:      input,
		}

		// errors are ok, we are fuzzing for crash
		// therefore we are interested in inputs that can both
		// be processed or generate an error
		_ = handler.handleMsg(other)
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
		utils.ToFtm(genesisBalance),
		utils.ToFtm(genesisStake),
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
		})
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

func newFuzzMsg(data []byte) (*p2p.Msg, error) {
	if len(data) < 1 {
		return nil, errors.New("empty data")
	}

	var (
		codes = []uint64{
			HandshakeMsg,
			EvmTxsMsg,
			ProgressMsg,
			NewEventIDsMsg,
			GetEventsMsg,
			EventsMsg,
			RequestEventsStream,
			EventsStreamResponse,
		}
		code = codes[int(data[0])%len(codes)]
	)
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
