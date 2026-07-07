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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

//go:generate mockgen -source=validatordirectory.go -destination=validatordirectory_mock.go -package=networks

// ValidatorDirectoryTopic is the gossipsub topic validators advertise their
// network location on so peers can discover where to dial them.
const ValidatorDirectoryTopic = "/sonic/validator-directory"

// directoryDomain domain-separates advertisement signatures from every other use
// of the consensus key.
const directoryDomain = "sonic/p2p/validator-directory/v1\x00"

// maxAdvertisementSize bounds a ValidatorAdvertisement message.
const maxAdvertisementSize = 8 << 10

// LocalNode provides this node's own network identity and addresses, so the
// directory can advertise where to reach it.
type LocalNode interface {
	// ID returns the local libp2p peer identity.
	ID() peer.ID
	// Addresses returns the multiaddrs the local node is reachable on.
	Addresses() []ma.Multiaddr
}

// AddressResolver resolves a validator's consensus public key to its network
// location and notifies subscribers when new addresses are learned. It is the
// internal handoff from the directory to the mesh.
type AddressResolver interface {
	// Resolve returns the network location advertised for the given consensus
	// public key, if one has been discovered.
	Resolve(publicKey []byte) (peer.AddrInfo, bool)
	// OnDiscovery registers a callback fired whenever an address is learned or
	// updated. The returned function cancels the subscription.
	OnDiscovery(callback func()) (cancel func())
}

// ValidatorDirectoryConfig tunes advertisement publishing.
type ValidatorDirectoryConfig struct {
	// RePublishInterval is the periodic backstop re-publish interval.
	RePublishInterval time.Duration
	// MaxJitter is the maximum random delay added before a triggered re-publish.
	MaxJitter time.Duration
	// Debounce coalesces bursts of re-publish triggers into a single publish.
	Debounce time.Duration
}

func (c ValidatorDirectoryConfig) withDefaults() ValidatorDirectoryConfig {
	if c.RePublishInterval <= 0 {
		c.RePublishInterval = 30 * time.Second
	}
	if c.MaxJitter <= 0 {
		c.MaxJitter = 5 * time.Second
	}
	if c.Debounce <= 0 {
		c.Debounce = time.Second
	}
	return c
}

// ValidatorDirectory is a gossipsub topic on which validators advertise their
// network location. It verifies advertisements against the current membership,
// maintains a directory of discovered addresses (exposed as an AddressResolver),
// and publishes this node's own advertisement. It implements p2p.GossipTopic and
// AddressResolver, and is safe for concurrent use.
type ValidatorDirectory struct {
	membership Membership
	signer     Signer
	verifier   Verifier
	publisher  Publisher
	local      LocalNode
	logger     logger.Logger
	config     ValidatorDirectoryConfig

	mutex       sync.Mutex
	entries     map[string]directoryEntry // keyed by consensus public key
	sequence    uint64
	subscribers map[int]func()
	nextSubID   int

	republish     chan struct{}
	quit          chan struct{}
	wait          sync.WaitGroup
	cancelContext context.CancelFunc
	cancelChange  func()
}

// directoryEntry is a discovered validator location.
type directoryEntry struct {
	info     peer.AddrInfo
	sequence uint64
}

// NewValidatorDirectory creates a validator directory. seed seeds the local
// advertisement sequence; pass a wall-clock nanosecond timestamp so it stays
// monotonic across restarts.
func NewValidatorDirectory(
	membership Membership,
	signer Signer,
	verifier Verifier,
	publisher Publisher,
	local LocalNode,
	log logger.Logger,
	config ValidatorDirectoryConfig,
	seed uint64,
) *ValidatorDirectory {
	return &ValidatorDirectory{
		membership:  membership,
		signer:      signer,
		verifier:    verifier,
		publisher:   publisher,
		local:       local,
		logger:      log,
		config:      config.withDefaults(),
		entries:     make(map[string]directoryEntry),
		sequence:    seed,
		subscribers: make(map[int]func()),
		republish:   make(chan struct{}, 1),
		quit:        make(chan struct{}),
	}
}

// Topic implements p2p.GossipTopic.
func (d *ValidatorDirectory) Topic() string { return ValidatorDirectoryTopic }

// Validate is the anti-spam gate: it accepts an advertisement only if it comes
// from a current validator, carries a valid signature, and is fresher than the
// last one stored for that validator.
func (d *ValidatorDirectory) Validate(_ p2p.PeerID, message []byte) p2p.ValidationResult {
	if len(message) > maxAdvertisementSize {
		return p2p.ValidationReject
	}
	advertisement, _, ok := d.parse(message)
	if !ok {
		return p2p.ValidationReject
	}
	if !d.isMember(advertisement.ValidatorPublicKey) {
		return p2p.ValidationReject
	}
	digest := directoryDigest(advertisement.PeerId, advertisement.Addresses, advertisement.Sequence)
	if !d.verifier.Verify(advertisement.ValidatorPublicKey, digest[:], advertisement.Signature) {
		return p2p.ValidationReject
	}
	d.mutex.Lock()
	previous, existed := d.entries[string(advertisement.ValidatorPublicKey)]
	d.mutex.Unlock()
	if existed && advertisement.Sequence <= previous.sequence {
		return p2p.ValidationIgnore
	}
	return p2p.ValidationAccept
}

// Deliver records a validated advertisement, notifies discovery subscribers, and
// re-publishes our own advertisement when a previously-unknown validator appears
// so that a late joiner learns the existing set promptly.
func (d *ValidatorDirectory) Deliver(_ p2p.PeerID, message []byte) {
	advertisement, info, ok := d.parse(message)
	if !ok {
		return
	}
	if bytes.Equal(advertisement.ValidatorPublicKey, d.signer.PublicKey()) {
		return // never store our own entry
	}
	key := string(advertisement.ValidatorPublicKey)

	d.mutex.Lock()
	previous, existed := d.entries[key]
	if existed && advertisement.Sequence <= previous.sequence {
		d.mutex.Unlock()
		return
	}
	d.entries[key] = directoryEntry{info: info, sequence: advertisement.Sequence}
	d.mutex.Unlock()

	d.notifyDiscovery()
	if !existed {
		d.logger.Debug("validator discovered", "peer", info.ID)
		d.scheduleRepublish()
	}
}

// Resolve implements AddressResolver.
func (d *ValidatorDirectory) Resolve(publicKey []byte) (peer.AddrInfo, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	entry, ok := d.entries[string(publicKey)]
	if !ok {
		return peer.AddrInfo{}, false
	}
	return entry.info, true
}

// OnDiscovery implements AddressResolver.
func (d *ValidatorDirectory) OnDiscovery(callback func()) (cancel func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	id := d.nextSubID
	d.nextSubID++
	d.subscribers[id] = callback
	return func() {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		delete(d.subscribers, id)
	}
}

// Start begins publishing this node's advertisement: immediately (publish on
// join), on membership changes and new-validator discovery (debounced), and on a
// periodic backstop. Call Stop to end publishing.
func (d *ValidatorDirectory) Start(ctx context.Context) {
	runContext, cancel := context.WithCancel(ctx)
	d.cancelContext = cancel

	d.publish(runContext)
	d.cancelChange = d.membership.OnChange(func() {
		d.pruneNonMembers()
		d.scheduleRepublish()
	})

	d.wait.Add(1)
	go d.publishLoop(runContext)
}

// Stop ends publishing and unsubscribes from membership changes.
func (d *ValidatorDirectory) Stop() {
	if d.cancelChange != nil {
		d.cancelChange()
		d.cancelChange = nil
	}
	close(d.quit)
	if d.cancelContext != nil {
		d.cancelContext()
	}
	d.wait.Wait()
}

func (d *ValidatorDirectory) publishLoop(ctx context.Context) {
	defer d.wait.Done()
	ticker := time.NewTicker(d.config.RePublishInterval)
	defer ticker.Stop()

	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	defer debounce.Stop()
	armed := false

	for {
		select {
		case <-d.quit:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.publish(ctx)
		case <-d.republish:
			debounce.Reset(d.config.Debounce + d.jitter())
			armed = true
		case <-debounce.C:
			if armed {
				armed = false
				d.publish(ctx)
			}
		}
	}
}

// publish builds, signs, and publishes this node's advertisement.
func (d *ValidatorDirectory) publish(ctx context.Context) {
	d.mutex.Lock()
	d.sequence++
	sequence := d.sequence
	d.mutex.Unlock()

	addresses := make([]string, 0, len(d.local.Addresses()))
	for _, address := range d.local.Addresses() {
		addresses = append(addresses, address.String())
	}
	peerID := []byte(d.local.ID())
	digest := directoryDigest(peerID, addresses, sequence)
	signature, err := d.signer.Sign(digest[:])
	if err != nil {
		d.logger.Warn("failed to sign validator advertisement", "err", err)
		return
	}
	message, err := proto.Marshal(&pb.ValidatorAdvertisement{
		ValidatorPublicKey: d.signer.PublicKey(),
		PeerId:             peerID,
		Addresses:          addresses,
		Sequence:           sequence,
		Signature:          signature,
	})
	if err != nil {
		d.logger.Warn("failed to marshal validator advertisement", "err", err)
		return
	}
	if err := d.publisher.Publish(ctx, ValidatorDirectoryTopic, message); err != nil {
		d.logger.Debug("failed to publish validator advertisement", "err", err)
	}
}

func (d *ValidatorDirectory) scheduleRepublish() {
	select {
	case d.republish <- struct{}{}:
	default: // a re-publish is already pending; coalesce
	}
}

func (d *ValidatorDirectory) notifyDiscovery() {
	d.mutex.Lock()
	callbacks := make([]func(), 0, len(d.subscribers))
	for _, callback := range d.subscribers {
		callbacks = append(callbacks, callback)
	}
	d.mutex.Unlock()
	for _, callback := range callbacks {
		callback()
	}
}

// pruneNonMembers drops discovered entries for validators no longer in the set.
func (d *ValidatorDirectory) pruneNonMembers() {
	members := d.membership.Members()
	current := make(map[string]struct{}, len(members))
	for _, member := range members {
		current[string(member.PublicKey)] = struct{}{}
	}
	d.mutex.Lock()
	changed := false
	for key := range d.entries {
		if _, ok := current[key]; !ok {
			delete(d.entries, key)
			changed = true
		}
	}
	d.mutex.Unlock()
	if changed {
		d.notifyDiscovery()
	}
}

func (d *ValidatorDirectory) isMember(publicKey []byte) bool {
	for _, member := range d.membership.Members() {
		if bytes.Equal(member.PublicKey, publicKey) {
			return true
		}
	}
	return false
}

func (d *ValidatorDirectory) jitter() time.Duration {
	if d.config.MaxJitter <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(d.config.MaxJitter) + 1))
}

// parse decodes and structurally validates an advertisement, returning the
// message and the peer.AddrInfo it describes.
func (d *ValidatorDirectory) parse(message []byte) (*pb.ValidatorAdvertisement, peer.AddrInfo, bool) {
	var advertisement pb.ValidatorAdvertisement
	if err := proto.Unmarshal(message, &advertisement); err != nil {
		return nil, peer.AddrInfo{}, false
	}
	peerID, err := peer.IDFromBytes(advertisement.PeerId)
	if err != nil {
		return nil, peer.AddrInfo{}, false
	}
	addresses := make([]ma.Multiaddr, 0, len(advertisement.Addresses))
	for _, address := range advertisement.Addresses {
		parsed, err := ma.NewMultiaddr(address)
		if err != nil {
			return nil, peer.AddrInfo{}, false
		}
		addresses = append(addresses, parsed)
	}
	return &advertisement, peer.AddrInfo{ID: peerID, Addrs: addresses}, true
}

// directoryDigest computes the domain-separated, length-prefixed digest signed by
// an advertisement over the peer ID, address list, and sequence.
func directoryDigest(peerID []byte, addresses []string, sequence uint64) [32]byte {
	hasher := sha256.New()
	hasher.Write([]byte(directoryDomain))
	writeLengthPrefixed(hasher, peerID)
	writeUvarint(hasher, uint64(len(addresses)))
	for _, address := range addresses {
		writeLengthPrefixed(hasher, []byte(address))
	}
	var sequenceBytes [8]byte
	binary.BigEndian.PutUint64(sequenceBytes[:], sequence)
	hasher.Write(sequenceBytes[:])
	var digest [32]byte
	copy(digest[:], hasher.Sum(nil))
	return digest
}

func writeLengthPrefixed(hasher hash.Hash, data []byte) {
	writeUvarint(hasher, uint64(len(data)))
	hasher.Write(data)
}

func writeUvarint(hasher hash.Hash, value uint64) {
	var buffer [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buffer[:], value)
	hasher.Write(buffer[:n])
}
