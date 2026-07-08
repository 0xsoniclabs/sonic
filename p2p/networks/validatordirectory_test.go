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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

func TestDirectoryDigest_DifferentInputs_DifferentHash(t *testing.T) {
	base := directoryDigest([]byte("peer"), []string{"ab", "c"}, 1)
	require.Equal(t, base, directoryDigest([]byte("peer"), []string{"ab", "c"}, 1), "identical inputs must hash equally")
	// Length-prefixing must distinguish a re-split address list.
	require.NotEqual(t, base, directoryDigest([]byte("peer"), []string{"a", "bc"}, 1), "re-split address list must change the digest")
	require.NotEqual(t, base, directoryDigest([]byte("peer"), []string{"c", "ab"}, 1), "reordered address list must change the digest")
	require.NotEqual(t, base, directoryDigest([]byte("peer"), []string{"ab", "c"}, 2), "changed sequence must change the digest")
	require.NotEqual(t, base, directoryDigest([]byte("PEER"), []string{"ab", "c"}, 1), "changed peer ID must change the digest")
}

func TestValidatorDirectory_ValidAdvertisement_Accepted(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})

	message := buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	require.Equal(t, p2p.ValidationAccept, fixture.directory.Validate("from", message), "expected accept")
}

func TestValidatorDirectory_NonMember_Rejected(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, _ := newTestSigner(t) // signer's key is not in the membership
	message := buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	require.Equal(t, p2p.ValidationReject, fixture.directory.Validate("from", message), "expected reject for non-member")
}

func TestValidatorDirectory_BadSignature_Rejected(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})

	message := buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	var advertisement pb.ValidatorAdvertisement
	require.NoError(t, proto.Unmarshal(message, &advertisement))
	advertisement.Signature[0] ^= 0xff
	tampered, _ := proto.Marshal(&advertisement)
	require.Equal(t, p2p.ValidationReject, fixture.directory.Validate("from", tampered), "expected reject for bad signature")
}

func TestValidatorDirectory_StaleSequence_Ignored(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	peerID := newTestPeerID(t)

	fixture.directory.Deliver("from", buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 5))

	stale := buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 3)
	require.Equal(t, p2p.ValidationIgnore, fixture.directory.Validate("from", stale), "expected ignore for stale sequence")
}

func TestValidatorDirectory_MalformedMessage_Rejected(t *testing.T) {
	fixture := newDirectoryFixture(t)
	require.Equal(t, p2p.ValidationReject, fixture.directory.Validate("from", []byte("not a protobuf")), "expected reject for malformed message")
}

func TestValidatorDirectory_Deliver_ResolvableAndNotifiesDiscovery(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	peerID := newTestPeerID(t)

	discovered := make(chan struct{}, 1)
	fixture.directory.OnDiscovery(func() {
		select {
		case discovered <- struct{}{}:
		default:
		}
	})

	fixture.directory.Deliver("from", buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 1))

	info, ok := fixture.directory.Resolve(publicKey)
	require.True(t, ok, "expected to resolve %s", peerID)
	require.Equal(t, peerID, info.ID, "expected to resolve %s", peerID)
	select {
	case <-discovered:
	default:
		t.Fatal("expected OnDiscovery to fire")
	}
}

func TestValidatorDirectory_Deliver_SkipsSelf(t *testing.T) {
	fixture := newDirectoryFixture(t)
	// An advertisement signed by our own key must never be stored.
	message := buildAdvertisement(t, fixture.signer, fixture.local.id, []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	fixture.directory.Deliver("from", message)
	_, ok := fixture.directory.Resolve(fixture.signer.PublicKey())
	require.False(t, ok, "expected own advertisement to be skipped")
}

func TestValidatorDirectory_PruneNonMembers_RemovesEntry(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	peerID := newTestPeerID(t)
	fixture.directory.Deliver("from", buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 1))
	_, ok := fixture.directory.Resolve(publicKey)
	require.True(t, ok, "expected entry present before prune")

	fixture.membership.set(nil) // validator removed from the set
	fixture.directory.pruneNonMembers()
	_, ok = fixture.directory.Resolve(publicKey)
	require.False(t, ok, "expected entry pruned after member removal")
}

func TestValidatorDirectory_Start_PublishesOnJoin(t *testing.T) {
	fixture := newDirectoryFixture(t)
	fixture.directory.Start(context.Background())
	defer fixture.directory.Stop()

	require.True(t, waitFor(func() bool { return fixture.publisher.count() >= 1 }, 2*time.Second), "expected an advertisement to be published on join")
	// The published advertisement is self-consistent and verifiable.
	var advertisement pb.ValidatorAdvertisement
	require.NoError(t, proto.Unmarshal(fixture.publisher.last(), &advertisement))
	digest := directoryDigest(advertisement.PeerId, advertisement.Addresses, advertisement.Sequence)
	require.True(t, NewSecp256k1Verifier().Verify(advertisement.ValidatorPublicKey, digest[:], advertisement.Signature), "published advertisement failed verification")
}

func TestValidatorDirectory_NewMemberDiscovery_TriggersRepublish(t *testing.T) {
	fixture := newDirectoryFixture(t)
	fixture.directory.Start(context.Background())
	defer fixture.directory.Stop()

	require.True(t, waitFor(func() bool { return fixture.publisher.count() >= 1 }, 2*time.Second), "expected the initial publish-on-join")
	initial := fixture.publisher.count()

	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	fixture.directory.Deliver("from", buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1))

	require.True(t, waitFor(func() bool { return fixture.publisher.count() > initial }, 2*time.Second), "expected discovering a new member to trigger a re-publish")
}

// --- fixtures & fakes ---

type directoryFixture struct {
	directory  *ValidatorDirectory
	membership *fakeMembership
	publisher  *fakePublisher
	signer     Signer
	local      fakeLocalNode
}

func newDirectoryFixture(t *testing.T) directoryFixture {
	t.Helper()
	signer, _ := newTestSigner(t)
	membership := &fakeMembership{}
	publisher := &fakePublisher{}
	local := fakeLocalNode{id: newTestPeerID(t), addrs: []ma.Multiaddr{mustAddr(t, "/ip4/127.0.0.1/tcp/4002")}}
	directory := NewValidatorDirectory(
		membership, signer, NewSecp256k1Verifier(), publisher, local, log.Root(),
		ValidatorDirectoryConfig{RePublishInterval: time.Hour, MaxJitter: time.Millisecond, Debounce: time.Millisecond},
		100,
	)
	return directoryFixture{directory: directory, membership: membership, publisher: publisher, signer: signer, local: local}
}

func buildAdvertisement(t *testing.T, signer Signer, peerID peer.ID, addresses []string, sequence uint64) []byte {
	t.Helper()
	digest := directoryDigest([]byte(peerID), addresses, sequence)
	signature, err := signer.Sign(digest[:])
	require.NoError(t, err, "failed to sign advertisement")
	message, err := proto.Marshal(&pb.ValidatorAdvertisement{
		ValidatorPublicKey: signer.PublicKey(),
		PeerId:             []byte(peerID),
		Addresses:          addresses,
		Sequence:           sequence,
		Signature:          signature,
	})
	require.NoError(t, err, "failed to marshal advertisement")
	return message
}

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	t.Helper()
	address, err := ma.NewMultiaddr(s)
	require.NoError(t, err, "bad multiaddr %q", s)
	return address
}

func waitFor(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

type fakePublisher struct {
	mutex    sync.Mutex
	messages [][]byte
}

func (p *fakePublisher) Publish(_ context.Context, _ string, message []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.messages = append(p.messages, message)
	return nil
}

func (p *fakePublisher) count() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.messages)
}

func (p *fakePublisher) last() []byte {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.messages[len(p.messages)-1]
}

type fakeLocalNode struct {
	id    peer.ID
	addrs []ma.Multiaddr
}

func (n fakeLocalNode) ID() peer.ID               { return n.id }
func (n fakeLocalNode) Addresses() []ma.Multiaddr { return n.addrs }
