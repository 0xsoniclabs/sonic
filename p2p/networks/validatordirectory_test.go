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
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

func TestDirectoryDigest_DifferentInputs_DifferentHash(t *testing.T) {
	base := directoryDigest([]byte("peer"), []string{"ab", "c"}, 1)
	if base != directoryDigest([]byte("peer"), []string{"ab", "c"}, 1) {
		t.Fatal("identical inputs must hash equally")
	}
	// Length-prefixing must distinguish a re-split address list.
	if base == directoryDigest([]byte("peer"), []string{"a", "bc"}, 1) {
		t.Fatal("re-split address list must change the digest")
	}
	if base == directoryDigest([]byte("peer"), []string{"c", "ab"}, 1) {
		t.Fatal("reordered address list must change the digest")
	}
	if base == directoryDigest([]byte("peer"), []string{"ab", "c"}, 2) {
		t.Fatal("changed sequence must change the digest")
	}
	if base == directoryDigest([]byte("PEER"), []string{"ab", "c"}, 1) {
		t.Fatal("changed peer ID must change the digest")
	}
}

func TestValidatorDirectory_ValidAdvertisement_Accepted(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})

	message := buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	if got := fixture.directory.Validate("from", message); got != p2p.ValidationAccept {
		t.Fatalf("expected accept, got %v", got)
	}
}

func TestValidatorDirectory_NonMember_Rejected(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, _ := newTestSigner(t) // signer's key is not in the membership
	message := buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	if got := fixture.directory.Validate("from", message); got != p2p.ValidationReject {
		t.Fatalf("expected reject for non-member, got %v", got)
	}
}

func TestValidatorDirectory_BadSignature_Rejected(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})

	message := buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1)
	var advertisement pb.ValidatorAdvertisement
	if err := proto.Unmarshal(message, &advertisement); err != nil {
		t.Fatal(err)
	}
	advertisement.Signature[0] ^= 0xff
	tampered, _ := proto.Marshal(&advertisement)
	if got := fixture.directory.Validate("from", tampered); got != p2p.ValidationReject {
		t.Fatalf("expected reject for bad signature, got %v", got)
	}
}

func TestValidatorDirectory_StaleSequence_Ignored(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	peerID := newTestPeerID(t)

	fixture.directory.Deliver("from", buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 5))

	stale := buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 3)
	if got := fixture.directory.Validate("from", stale); got != p2p.ValidationIgnore {
		t.Fatalf("expected ignore for stale sequence, got %v", got)
	}
}

func TestValidatorDirectory_MalformedMessage_Rejected(t *testing.T) {
	fixture := newDirectoryFixture(t)
	if got := fixture.directory.Validate("from", []byte("not a protobuf")); got != p2p.ValidationReject {
		t.Fatalf("expected reject for malformed message, got %v", got)
	}
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
	if !ok || info.ID != peerID {
		t.Fatalf("expected to resolve %s, got %+v ok=%v", peerID, info, ok)
	}
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
	if _, ok := fixture.directory.Resolve(fixture.signer.PublicKey()); ok {
		t.Fatal("expected own advertisement to be skipped")
	}
}

func TestValidatorDirectory_PruneNonMembers_RemovesEntry(t *testing.T) {
	fixture := newDirectoryFixture(t)
	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	peerID := newTestPeerID(t)
	fixture.directory.Deliver("from", buildAdvertisement(t, signer, peerID, []string{"/ip4/127.0.0.1/tcp/4002"}, 1))
	if _, ok := fixture.directory.Resolve(publicKey); !ok {
		t.Fatal("expected entry present before prune")
	}

	fixture.membership.set(nil) // validator removed from the set
	fixture.directory.pruneNonMembers()
	if _, ok := fixture.directory.Resolve(publicKey); ok {
		t.Fatal("expected entry pruned after member removal")
	}
}

func TestValidatorDirectory_Start_PublishesOnJoin(t *testing.T) {
	fixture := newDirectoryFixture(t)
	fixture.directory.Start(context.Background())
	defer fixture.directory.Stop()

	if !waitFor(func() bool { return fixture.publisher.count() >= 1 }, 2*time.Second) {
		t.Fatal("expected an advertisement to be published on join")
	}
	// The published advertisement is self-consistent and verifiable.
	var advertisement pb.ValidatorAdvertisement
	if err := proto.Unmarshal(fixture.publisher.last(), &advertisement); err != nil {
		t.Fatal(err)
	}
	digest := directoryDigest(advertisement.PeerId, advertisement.Addresses, advertisement.Sequence)
	if !NewSecp256k1Verifier().Verify(advertisement.ValidatorPublicKey, digest[:], advertisement.Signature) {
		t.Fatal("published advertisement failed verification")
	}
}

func TestValidatorDirectory_NewMemberDiscovery_TriggersRepublish(t *testing.T) {
	fixture := newDirectoryFixture(t)
	fixture.directory.Start(context.Background())
	defer fixture.directory.Stop()

	if !waitFor(func() bool { return fixture.publisher.count() >= 1 }, 2*time.Second) {
		t.Fatal("expected the initial publish-on-join")
	}
	initial := fixture.publisher.count()

	signer, publicKey := newTestSigner(t)
	fixture.membership.set([]Member{{ID: 2, PublicKey: publicKey}})
	fixture.directory.Deliver("from", buildAdvertisement(t, signer, newTestPeerID(t), []string{"/ip4/127.0.0.1/tcp/4002"}, 1))

	if !waitFor(func() bool { return fixture.publisher.count() > initial }, 2*time.Second) {
		t.Fatal("expected discovering a new member to trigger a re-publish")
	}
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
	if err != nil {
		t.Fatalf("failed to sign advertisement: %v", err)
	}
	message, err := proto.Marshal(&pb.ValidatorAdvertisement{
		ValidatorPublicKey: signer.PublicKey(),
		PeerId:             []byte(peerID),
		Addresses:          addresses,
		Sequence:           sequence,
		Signature:          signature,
	})
	if err != nil {
		t.Fatalf("failed to marshal advertisement: %v", err)
	}
	return message
}

func mustAddr(t *testing.T, s string) ma.Multiaddr {
	t.Helper()
	address, err := ma.NewMultiaddr(s)
	if err != nil {
		t.Fatalf("bad multiaddr %q: %v", s, err)
	}
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
