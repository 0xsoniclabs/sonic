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

package networks

import (
	"context"
	"sync"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
)

func TestValidatorMesh_InitialSet_DialsAllValidators(t *testing.T) {
	host := newFakeMeshHost(t)
	mesh := NewValidatorMesh(host)

	a := validatorAt(t, host, 1)
	b := validatorAt(t, host, 2)
	mesh.Reconcile(context.Background(), []Validator{a, b})

	host.assertConnected(t, a.Peer.ID, b.Peer.ID)
}

func TestValidatorMesh_ValidatorRemoved_DisconnectsRemovedWithReason(t *testing.T) {
	host := newFakeMeshHost(t)
	mesh := NewValidatorMesh(host)

	a := validatorAt(t, host, 1)
	b := validatorAt(t, host, 2)
	c := validatorAt(t, host, 3)
	mesh.Reconcile(context.Background(), []Validator{a, b})

	// b leaves, c joins.
	mesh.Reconcile(context.Background(), []Validator{a, c})

	host.assertConnected(t, a.Peer.ID, c.Peer.ID)
	if reason, ok := host.closedReason(b.Peer.ID); !ok || reason != "removed-from-set" {
		t.Fatalf("expected b closed with reason removed-from-set, got %q ok=%v", reason, ok)
	}
	if _, dialedAgain := host.dialCount(a.Peer.ID); dialedAgain != 1 {
		t.Fatalf("expected a dialed exactly once across reconciles, got %d", dialedAgain)
	}
}

func TestValidatorMesh_SkipsSelf(t *testing.T) {
	host := newFakeMeshHost(t)
	mesh := NewValidatorMesh(host)

	self := Validator{ID: 9, Peer: peer.AddrInfo{ID: host.id}}
	other := validatorAt(t, host, 2)
	mesh.Reconcile(context.Background(), []Validator{self, other})

	host.assertConnected(t, other.Peer.ID)
}

func TestValidatorMesh_SetUpdate_TriggersReconcile(t *testing.T) {
	host := newFakeMeshHost(t)
	mesh := NewValidatorMesh(host)

	a := validatorAt(t, host, 1)
	b := validatorAt(t, host, 2)
	set := &fakeValidatorSet{current: []Validator{a}}

	mesh.Track(context.Background(), set)
	host.assertConnected(t, a.Peer.ID)

	set.pushUpdate([]Validator{a, b})
	host.assertConnected(t, a.Peer.ID, b.Peer.ID)
	mesh.Stop()
}

// --- fakes ---

func newFakeMeshHost(t *testing.T) *fakeMeshHost {
	t.Helper()
	controller := gomock.NewController(t)
	log := logger.NewMockLogger(controller)
	log.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	log.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	return &fakeMeshHost{
		id:      newTestPeerID(t),
		logger:  log,
		dialed:  make(map[peer.ID]int),
		closed:  make(map[peer.ID]string),
		present: make(map[peer.ID]struct{}),
	}
}

type fakeMeshHost struct {
	id      peer.ID
	logger  logger.Logger
	mutex   sync.Mutex
	dialed  map[peer.ID]int
	closed  map[peer.ID]string
	present map[peer.ID]struct{}
}

func (h *fakeMeshHost) ID() p2p.PeerID { return h.id }

func (h *fakeMeshHost) Connect(_ context.Context, info peer.AddrInfo) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.dialed[info.ID]++
	h.present[info.ID] = struct{}{}
	return nil
}

func (h *fakeMeshHost) ClosePeer(target p2p.PeerID, reason string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.closed[target] = reason
	delete(h.present, target)
	return nil
}

func (h *fakeMeshHost) Logger() logger.Logger { return h.logger }

func (h *fakeMeshHost) assertConnected(t *testing.T, expected ...peer.ID) {
	t.Helper()
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if len(h.present) != len(expected) {
		t.Fatalf("expected %d connections, have %d", len(expected), len(h.present))
	}
	for _, id := range expected {
		if _, ok := h.present[id]; !ok {
			t.Fatalf("expected connection to %s", id)
		}
	}
}

func (h *fakeMeshHost) closedReason(id peer.ID) (string, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	reason, ok := h.closed[id]
	return reason, ok
}

func (h *fakeMeshHost) dialCount(id peer.ID) (peer.ID, int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return id, h.dialed[id]
}

func validatorAt(t *testing.T, _ *fakeMeshHost, id uint32) Validator {
	t.Helper()
	return Validator{
		ID:        id,
		PublicKey: []byte{byte(id)},
		Peer:      peer.AddrInfo{ID: newTestPeerID(t)},
	}
}

type fakeValidatorSet struct {
	mutex    sync.Mutex
	current  []Validator
	callback func([]Validator)
}

func (s *fakeValidatorSet) Current() []Validator {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.current
}

func (s *fakeValidatorSet) Epoch() uint64 { return 1 }

func (s *fakeValidatorSet) OnUpdate(callback func([]Validator)) func() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.callback = callback
	return func() {}
}

func (s *fakeValidatorSet) pushUpdate(validators []Validator) {
	s.mutex.Lock()
	s.current = validators
	callback := s.callback
	s.mutex.Unlock()
	if callback != nil {
		callback(validators)
	}
}
