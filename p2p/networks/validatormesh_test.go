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

func TestValidatorMesh_ResolvableMembers_AllDialed(t *testing.T) {
	host := newFakeMeshHost(t)
	resolver := newFakeResolver()
	mesh := NewValidatorMesh(host, resolver, nil)

	a, infoA := memberAt(t, 1)
	b, infoB := memberAt(t, 2)
	resolver.add(a.PublicKey, infoA)
	resolver.add(b.PublicKey, infoB)

	mesh.Reconcile(context.Background(), []Member{a, b})
	host.assertConnected(t, infoA.ID, infoB.ID)
}

func TestValidatorMesh_UnresolvedMember_Skipped(t *testing.T) {
	host := newFakeMeshHost(t)
	resolver := newFakeResolver()
	mesh := NewValidatorMesh(host, resolver, nil)

	a, infoA := memberAt(t, 1)
	b, _ := memberAt(t, 2) // address not yet known
	resolver.add(a.PublicKey, infoA)

	mesh.Reconcile(context.Background(), []Member{a, b})
	host.assertConnected(t, infoA.ID)
}

func TestValidatorMesh_MemberRemoved_DisconnectsWithReason(t *testing.T) {
	host := newFakeMeshHost(t)
	resolver := newFakeResolver()
	mesh := NewValidatorMesh(host, resolver, nil)

	a, infoA := memberAt(t, 1)
	b, infoB := memberAt(t, 2)
	c, infoC := memberAt(t, 3)
	resolver.add(a.PublicKey, infoA)
	resolver.add(b.PublicKey, infoB)
	resolver.add(c.PublicKey, infoC)

	mesh.Reconcile(context.Background(), []Member{a, b})
	mesh.Reconcile(context.Background(), []Member{a, c})

	host.assertConnected(t, infoA.ID, infoC.ID)
	if reason, ok := host.closedReason(infoB.ID); !ok || reason != "removed-from-set" {
		t.Fatalf("expected b closed with reason removed-from-set, got %q ok=%v", reason, ok)
	}
	if dialed := host.dialCount(infoA.ID); dialed != 1 {
		t.Fatalf("expected a dialed exactly once across reconciles, got %d", dialed)
	}
}

func TestValidatorMesh_SkipsSelf(t *testing.T) {
	host := newFakeMeshHost(t)
	resolver := newFakeResolver()
	mesh := NewValidatorMesh(host, resolver, nil)

	self, _ := memberAt(t, 9)
	resolver.add(self.PublicKey, peer.AddrInfo{ID: host.id})
	other, infoOther := memberAt(t, 2)
	resolver.add(other.PublicKey, infoOther)

	mesh.Reconcile(context.Background(), []Member{self, other})
	host.assertConnected(t, infoOther.ID)
}

func TestValidatorMesh_MembershipChange_TriggersReconcile(t *testing.T) {
	host := newFakeMeshHost(t)
	resolver := newFakeResolver()
	mesh := NewValidatorMesh(host, resolver, nil)

	a, infoA := memberAt(t, 1)
	b, infoB := memberAt(t, 2)
	resolver.add(a.PublicKey, infoA)
	resolver.add(b.PublicKey, infoB)
	membership := &fakeMembership{members: []Member{a}}

	mesh.Track(context.Background(), membership)
	host.assertConnected(t, infoA.ID)

	membership.set([]Member{a, b})
	host.assertConnected(t, infoA.ID, infoB.ID)
	mesh.Stop()
}

func TestValidatorMesh_AddressDiscovered_TriggersDial(t *testing.T) {
	host := newFakeMeshHost(t)
	resolver := newFakeResolver()
	mesh := NewValidatorMesh(host, resolver, nil)

	a, infoA := memberAt(t, 1)
	b, infoB := memberAt(t, 2)
	resolver.add(a.PublicKey, infoA) // b's address not yet known
	membership := &fakeMembership{members: []Member{a, b}}

	mesh.Track(context.Background(), membership)
	host.assertConnected(t, infoA.ID)

	resolver.add(b.PublicKey, infoB) // fires OnDiscovery -> reconcile
	host.assertConnected(t, infoA.ID, infoB.ID)
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

func (h *fakeMeshHost) dialCount(id peer.ID) int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.dialed[id]
}

func memberAt(t *testing.T, id uint32) (Member, peer.AddrInfo) {
	t.Helper()
	return Member{ID: id, PublicKey: []byte{byte(id), 0xaa, 0xbb}}, peer.AddrInfo{ID: newTestPeerID(t)}
}

// fakeResolver is a hand-rolled AddressResolver recording discovery callbacks.
type fakeResolver struct {
	mutex       sync.Mutex
	entries     map[string]peer.AddrInfo
	subscribers []func()
}

func newFakeResolver() *fakeResolver {
	return &fakeResolver{entries: make(map[string]peer.AddrInfo)}
}

func (r *fakeResolver) Resolve(publicKey []byte) (peer.AddrInfo, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	info, ok := r.entries[string(publicKey)]
	return info, ok
}

func (r *fakeResolver) OnDiscovery(callback func()) func() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.subscribers = append(r.subscribers, callback)
	return func() {}
}

func (r *fakeResolver) add(publicKey []byte, info peer.AddrInfo) {
	r.mutex.Lock()
	r.entries[string(publicKey)] = info
	callbacks := append([]func(){}, r.subscribers...)
	r.mutex.Unlock()
	for _, callback := range callbacks {
		callback()
	}
}

// fakeMembership is a hand-rolled Membership with a single change subscriber.
type fakeMembership struct {
	mutex    sync.Mutex
	members  []Member
	callback func()
}

func (m *fakeMembership) Members() []Member {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.members
}

func (m *fakeMembership) Epoch() uint64 { return 1 }

func (m *fakeMembership) OnChange(callback func()) func() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.callback = callback
	return func() {}
}

func (m *fakeMembership) set(members []Member) {
	m.mutex.Lock()
	m.members = members
	callback := m.callback
	m.mutex.Unlock()
	if callback != nil {
		callback()
	}
}
