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
	"io"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/eventcheck"
	"github.com/0xsoniclabs/sonic/eventcheck/gaspowercheck"
	"github.com/0xsoniclabs/sonic/eventcheck/parentscheck"
	"github.com/0xsoniclabs/sonic/eventcheck/proposalcheck"
	"github.com/0xsoniclabs/sonic/inter"
	parentscheckbase "github.com/Fantom-foundation/lachesis-base/eventcheck/parentscheck"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover/discfilter"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateEventPropertiesDependingOnParents(t *testing.T) {

	tests := map[string]struct {
		modify   func(*inter.MutableEventPayload)
		expected error
	}{
		"valid event": {
			modify: func(event *inter.MutableEventPayload) {},
		},
		"parents check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetLamport(2)
			},
			expected: parentscheckbase.ErrWrongLamport,
		},
		"gas power check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetGasPowerLeft(inter.GasPowerLeft{
					Gas: [inter.GasPowerConfigs]uint64{1000, 2000},
				})
			},
			expected: gaspowercheck.ErrWrongGasPowerLeft,
		},
		"proposal check violation": {
			modify: func(event *inter.MutableEventPayload) {
				event.SetVersion(3)
				event.SetPayload(inter.Payload{
					ProposalSyncState: inter.ProposalSyncState{
						LastSeenProposalTurn: 75,
					},
				})
			},
			expected: proposalcheck.ErrSyncStateProgressionWithoutProposal,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			gasPowerCheckReader := gaspowercheck.NewMockReader(ctrl)
			proposalCheckReader := proposalcheck.NewMockReader(ctrl)

			checkers := &eventcheck.Checkers{
				Parentscheck:  parentscheck.New(),
				Gaspowercheck: gaspowercheck.New(gasPowerCheckReader),
				Proposalcheck: proposalcheck.New(proposalCheckReader),
			}

			epoch := idx.Epoch(12)

			creator := idx.ValidatorID(1)
			validatorsBuilder := pos.ValidatorsBuilder{}
			validatorsBuilder.Set(creator, pos.Weight(100))
			validators := validatorsBuilder.Build()

			// Create a parent event.
			builder := inter.MutableEventPayload{}
			builder.SetEpoch(epoch)
			builder.SetCreator(creator)
			builder.SetSeq(1)
			parent := builder.Build()

			// Create the event to be tested.
			builder = inter.MutableEventPayload{}
			builder.SetEpoch(epoch)
			builder.SetCreator(creator)
			builder.SetLamport(1)
			builder.SetSeq(2)
			builder.SetCreationTime(1)
			builder.SetParents([]hash.Event{parent.ID()})

			test.modify(&builder)

			event := builder.Build()

			// Set up the validation context.
			validationContext := &gaspowercheck.ValidationContext{
				Epoch:           epoch,
				Validators:      validators,
				ValidatorStates: []gaspowercheck.ValidatorState{{}},
			}
			gasPowerCheckReader.EXPECT().GetValidationContext().Return(validationContext).AnyTimes()

			proposalCheckReader.EXPECT().GetEventPayload(gomock.Any()).Return(inter.Payload{}).AnyTimes()

			// Run the actual check.
			require.ErrorIs(t, validateEventPropertiesDependingOnParents(
				checkers,
				event,
				[]inter.EventI{parent},
			), test.expected)
		})
	}
}

func TestHandleMsgEventsStreamResponse(t *testing.T) {
	h, err := makeFuzzedHandler(t)
	require.NoError(t, err)

	peerCfg := DefaultPeerCacheConfig(cachescale.Identity)
	makeTestPeer := func(rw p2p.MsgReadWriter) *peer {
		return newPeer(1, p2p.NewPeer(randomID(), "test-peer", []p2p.Cap{}), rw, peerCfg)
	}
	sendChunk := func(chunk dagChunk) error {
		encoded, err := rlp.EncodeToBytes(chunk)
		require.NoError(t, err)
		msg := &p2p.Msg{
			Code:    EventsStreamResponse,
			Size:    uint32(len(encoded)),
			Payload: bytes.NewReader(encoded),
		}
		return h.handleMsg(makeTestPeer(&fuzzMsgReadWriter{msg}))
	}

	t.Run("empty chunk with Done=false is rejected", func(t *testing.T) {
		err := sendChunk(dagChunk{SessionID: 1, Done: false})
		require.Error(t, err)
	})

	t.Run("empty chunk with Done=true is accepted", func(t *testing.T) {
		err := sendChunk(dagChunk{SessionID: 1, Done: true})
		require.NoError(t, err)
	})

	t.Run("chunk with IDs and Done=false is accepted", func(t *testing.T) {
		err := sendChunk(dagChunk{SessionID: 1, Done: false, IDs: hash.Events{hash.Event{}}})
		require.NoError(t, err)
	})

	t.Run("chunk with IDs and Done=true is accepted", func(t *testing.T) {
		err := sendChunk(dagChunk{SessionID: 1, Done: true, IDs: hash.Events{hash.Event{}}})
		require.NoError(t, err)
	})
}

// TestHandlePanicRecovery verifies that a panic anywhere inside handle()'s
// message loop is caught by its defer/recover and returned as an error,
// disconnecting only the offending peer rather than crashing the node.
func TestHandlePanicRecovery(t *testing.T) {
	h, err := makeFuzzedHandler(t)
	require.NoError(t, err)

	peerCfg := DefaultPeerCacheConfig(cachescale.Identity)
	rw := &handshakeThenPanicReadWriter{
		networkID: h.NetworkID,
		genesis:   common.Hash(h.store.GetGenesisID()),
		version:   1,
	}
	// Peer name must contain "sonic" or "opera" to not be rejected as useless
	// by isUseless() before the message loop is even reached.
	p := newPeer(1, p2p.NewPeer(randomID(), "Sonic/v1.0.0-test/linux-amd64/go1.25", []p2p.Cap{}), rw, peerCfg)

	err = h.handle(p)
	require.ErrorContains(t, err, "panic while handling peer")
}

func TestHandlePeerInputResilience(t *testing.T) {
	h, err := makeFuzzedHandler(t)
	require.NoError(t, err)

	peerCfg := DefaultPeerCacheConfig(cachescale.Identity)
	newTestPeer := func(rw p2p.MsgReadWriter) *peer {
		return newPeer(
			1,
			p2p.NewPeer(randomID(), "Sonic/resilience-test", []p2p.Cap{}),
			rw,
			peerCfg,
		)
	}

	makeMsg := func(code uint64, payload interface{}) *p2p.Msg {
		t.Helper()
		if payload == nil {
			return &p2p.Msg{Code: code, Size: 0, Payload: bytes.NewReader(nil)}
		}
		encoded, err := rlp.EncodeToBytes(payload)
		require.NoError(t, err)
		return &p2p.Msg{Code: code, Size: uint32(len(encoded)), Payload: bytes.NewReader(encoded)}
	}

	handshake := makeMsg(HandshakeMsg, &handshakeData{
		ProtocolVersion: 1,
		NetworkID:       h.NetworkID,
		Genesis:         common.Hash(h.store.GetGenesisID()),
	})

	tests := []struct {
		name            string
		steps           []readStep
		wantErrContains string
	}{
		{
			name: "invalid handshake code as first message",
			steps: []readStep{
				{msg: makeMsg(ProgressMsg, &PeerProgress{})},
			},
			wantErrContains: "No status message",
		},
		{
			name: "invalid handshake payload",
			steps: []readStep{
				{msg: &p2p.Msg{Code: HandshakeMsg, Size: 1, Payload: bytes.NewReader([]byte{0xff})}},
			},
			wantErrContains: "Invalid message",
		},
		{
			name: "successful handshake then malformed events payload",
			steps: []readStep{
				{msg: handshake},
				{msg: &p2p.Msg{Code: EventsMsg, Size: 1, Payload: bytes.NewReader([]byte{0xff})}},
			},
			wantErrContains: "Invalid message",
		},
		{
			name: "successful handshake then events containing nil event pointer",
			steps: []readStep{
				{msg: handshake},
				{msg: makeMsg(EventsMsg, inter.EventPayloads{nil})},
			},
			wantErrContains: "Invalid message",
		},
		{
			name: "successful handshake then event stream response with nil event pointer",
			steps: []readStep{
				{msg: handshake},
				{msg: makeMsg(EventsStreamResponse, dagChunk{SessionID: 1, Events: inter.EventPayloads{nil}})},
			},
			wantErrContains: "Invalid message",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rw := &scriptedReadWriter{steps: tc.steps}
			p := newTestPeer(rw)

			var gotErr error
			require.NotPanics(t, func() {
				gotErr = h.handle(p)
			})
			require.Error(t, gotErr)
			if tc.wantErrContains != "" {
				require.ErrorContains(t, gotErr, tc.wantErrContains)
			}
		})
	}
}

type readStep struct {
	msg *p2p.Msg
	err error
}

type scriptedReadWriter struct {
	mu    sync.Mutex
	steps []readStep
	pos   int
}

func (rw *scriptedReadWriter) ReadMsg() (p2p.Msg, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.pos >= len(rw.steps) {
		return p2p.Msg{}, io.EOF
	}
	step := rw.steps[rw.pos]
	rw.pos++
	if step.err != nil {
		return p2p.Msg{}, step.err
	}
	if step.msg == nil {
		return p2p.Msg{}, io.EOF
	}
	return *step.msg, nil
}

func (rw *scriptedReadWriter) WriteMsg(p2p.Msg) error {
	return nil
}

// handshakeThenPanicReadWriter satisfies p2p.MsgReadWriter. It returns a valid
// HandshakeMsg on the first ReadMsg call so that peer.Handshake() succeeds,
// then panics on subsequent calls to simulate a bug triggered by peer input.
type handshakeThenPanicReadWriter struct {
	networkID uint64
	genesis   common.Hash
	version   uint

	mu        sync.Mutex
	readCount int
}

func (rw *handshakeThenPanicReadWriter) ReadMsg() (p2p.Msg, error) {
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
			panic(err)
		}
		return p2p.Msg{
			Code:    HandshakeMsg,
			Size:    uint32(len(encoded)),
			Payload: bytes.NewReader(encoded),
		}, nil
	}

	panic("simulated peer-input panic")
}

func (rw *handshakeThenPanicReadWriter) WriteMsg(p2p.Msg) error {
	return nil
}

func TestIsUseless(t *testing.T) {
	validEnode := enode.MustParse("enode://3f4306c065eaa5d8079e17feb56c03a97577e67af3c9c17496bb8916f102f1ff603e87d2a4ebfa0a2f70b780b85db212618857ea4e9627b24a9b0dd2faeb826e@127.0.0.1:5050")
	sonicName := "Sonic/v1.0.0-a-61af51c2-1715085138/linux-amd64/go1.21.7"
	operaName := "go-opera/v1.1.2-rc.6-8e84c9dc-1688013329/linux-amd64/go1.19.11"
	invalidName := "bot"

	discfilter.Enable()
	if isUseless(validEnode, sonicName) {
		t.Errorf("sonic peer reported as useless")
	}
	if isUseless(validEnode, operaName) {
		t.Errorf("opera peer reported as useless")
	}
	if !isUseless(validEnode, invalidName) {
		t.Errorf("invalid peer not reported as useless")
	}
	if !isUseless(validEnode, operaName) {
		t.Errorf("peer not banned after marking as useless")
	}
}
