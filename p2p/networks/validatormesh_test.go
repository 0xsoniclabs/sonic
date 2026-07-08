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
	"errors"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
)

func TestHandshakeProtocol_ValidProof_Authenticates(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	membership := &fakeMembership{members: []Member{{ID: 2, PublicKey: publicKey}}}
	peerID := newTestPeerID(t)
	proof, err := CreateBindingProof(signer, peerID, 2, membership.Epoch(), newNonce(t))
	require.NoError(t, err)

	var authenticated bool
	var failure error
	handshake := NewHandshakeProtocol(newTestPeerID(t), signer, NewSecp256k1Verifier(), membership, 2, log.Root(),
		func(peer.ID, uint32) { authenticated = true },
		func(_ peer.ID, err error) { failure = err })

	stream := &fakeStream{peer: peerID, payload: proof}
	handshake.Handle(stream)

	require.True(t, authenticated, "expected authentication")
	require.NoError(t, failure, "expected authentication")
	require.True(t, stream.closed, "expected a clean close")
	require.False(t, stream.reset, "expected a clean close")
}

func TestHandshakeProtocol_NonMemberProof_ReportsFailure(t *testing.T) {
	member, publicKey := newTestSigner(t)
	membership := &fakeMembership{members: []Member{{ID: 2, PublicKey: publicKey}}}
	outsider, _ := newTestSigner(t) // not in the membership
	peerID := newTestPeerID(t)
	proof, err := CreateBindingProof(outsider, peerID, 999, membership.Epoch(), newNonce(t))
	require.NoError(t, err)

	var authenticated bool
	var failure error
	handshake := NewHandshakeProtocol(newTestPeerID(t), member, NewSecp256k1Verifier(), membership, 2, log.Root(),
		func(peer.ID, uint32) { authenticated = true },
		func(_ peer.ID, err error) { failure = err })

	stream := &fakeStream{peer: peerID, payload: proof}
	handshake.Handle(stream)

	require.False(t, authenticated, "a non-member proof must not authenticate")
	require.ErrorIs(t, failure, ErrHandshakeNotValidator)
	require.True(t, stream.reset, "expected the stream to be reset on failure")
}

func TestHandshakeProtocol_ReadError_ReportsFailure(t *testing.T) {
	signer, publicKey := newTestSigner(t)
	membership := &fakeMembership{members: []Member{{ID: 2, PublicKey: publicKey}}}

	var failure error
	handshake := NewHandshakeProtocol(newTestPeerID(t), signer, NewSecp256k1Verifier(), membership, 2, log.Root(),
		nil, func(_ peer.ID, err error) { failure = err })

	stream := &fakeStream{peer: newTestPeerID(t), readErr: errors.New("boom")}
	handshake.Handle(stream)

	require.Error(t, failure, "expected a read error to be reported via onFailure")
}

// fakeStream is a minimal p2p.Stream returning a preset message.
type fakeStream struct {
	peer    peer.ID
	payload proto.Message
	readErr error
	reset   bool
	closed  bool
}

func (s *fakeStream) Peer() p2p.PeerID { return s.peer }

func (s *fakeStream) ReadMessage(message proto.Message, _ int) error {
	if s.readErr != nil {
		return s.readErr
	}
	proto.Reset(message)
	proto.Merge(message, s.payload)
	return nil
}

func (s *fakeStream) WriteMessage(proto.Message, int) error { return nil }
func (s *fakeStream) Close() error                          { s.closed = true; return nil }
func (s *fakeStream) Reset() error                          { s.reset = true; return nil }

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
	reason, ok := host.closedReason(infoB.ID)
	require.True(t, ok, "expected b to be closed")
	require.Equal(t, "removed-from-set", reason, "expected b closed with reason removed-from-set")
	require.Equal(t, 1, host.dialCount(infoA.ID), "expected a dialed exactly once across reconciles")
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
	require.Len(t, h.present, len(expected))
	for _, id := range expected {
		require.Contains(t, h.present, id, "expected connection to %s", id)
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
