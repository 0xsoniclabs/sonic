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
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

// ArchiveTopic is the gossipsub rendezvous topic archives advertise on and
// joining nodes subscribe to in order to discover a source of historic blocks.
const ArchiveTopic = "/sonic/archive-directory"

// Publisher publishes a message to a gossip topic; *p2p.Node satisfies it.
type Publisher interface {
	Publish(ctx context.Context, topic string, message []byte) error
}

// ArchiveInfo is a discovered archive and the block range it serves.
type ArchiveInfo struct {
	Peer         peer.ID
	Addresses    []string
	HistoryStart uint64
	HistoryEnd   uint64
	ExpiresAt    time.Time
}

// ArchiveDirectory implements p2p.GossipTopic for archive discovery. It
// validates advertisements (TTL and range sanity) and maintains a local
// directory of live archives. It is safe for concurrent use.
type ArchiveDirectory struct {
	logger  logger.Logger
	now     func() time.Time
	mutex   sync.Mutex
	entries map[peer.ID]ArchiveInfo
}

// NewArchiveDirectory creates an empty archive directory.
func NewArchiveDirectory(log logger.Logger) *ArchiveDirectory {
	return &ArchiveDirectory{
		logger:  log,
		now:     time.Now,
		entries: make(map[peer.ID]ArchiveInfo),
	}
}

// Topic implements p2p.GossipTopic.
func (d *ArchiveDirectory) Topic() string { return ArchiveTopic }

// Validate rejects malformed, expired, or nonsensical advertisements before
// they propagate, and is the anti-spam gate for the topic.
func (d *ArchiveDirectory) Validate(_ p2p.PeerID, message []byte) p2p.ValidationResult {
	advertisement, info, ok := d.parse(message)
	if !ok || advertisement == nil {
		return p2p.ValidationReject
	}
	if !info.ExpiresAt.After(d.now()) {
		return p2p.ValidationIgnore
	}
	if info.HistoryStart > info.HistoryEnd {
		return p2p.ValidationReject
	}
	return p2p.ValidationAccept
}

// Deliver records a validated advertisement in the directory.
func (d *ArchiveDirectory) Deliver(_ p2p.PeerID, message []byte) {
	_, info, ok := d.parse(message)
	if !ok {
		return
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.entries[info.Peer] = info
	d.logger.Debug("archive advertised", "peer", info.Peer,
		"start", info.HistoryStart, "end", info.HistoryEnd)
}

// Archives returns the currently live (non-expired) archives.
func (d *ArchiveDirectory) Archives() []ArchiveInfo {
	now := d.now()
	d.mutex.Lock()
	defer d.mutex.Unlock()
	result := make([]ArchiveInfo, 0, len(d.entries))
	for peerID, info := range d.entries {
		if !info.ExpiresAt.After(now) {
			delete(d.entries, peerID)
			continue
		}
		result = append(result, info)
	}
	return result
}

// Advertise publishes this node's archive advertisement to the topic, valid for
// the given time-to-live.
func Advertise(ctx context.Context, publisher Publisher, self peer.ID, addresses []string, historyStart, historyEnd uint64, ttl time.Duration) error {
	advertisement := &pb.ArchiveAdvertisement{
		PeerId:        []byte(self),
		Addresses:     addresses,
		HistoryStart:  historyStart,
		HistoryEnd:    historyEnd,
		ExpiresAtUnix: time.Now().Add(ttl).Unix(),
	}
	message, err := proto.Marshal(advertisement)
	if err != nil {
		return err
	}
	return publisher.Publish(ctx, ArchiveTopic, message)
}

func (d *ArchiveDirectory) parse(message []byte) (*pb.ArchiveAdvertisement, ArchiveInfo, bool) {
	var advertisement pb.ArchiveAdvertisement
	if err := proto.Unmarshal(message, &advertisement); err != nil {
		return nil, ArchiveInfo{}, false
	}
	peerID, err := peer.IDFromBytes(advertisement.PeerId)
	if err != nil {
		return nil, ArchiveInfo{}, false
	}
	info := ArchiveInfo{
		Peer:         peerID,
		Addresses:    advertisement.Addresses,
		HistoryStart: advertisement.HistoryStart,
		HistoryEnd:   advertisement.HistoryEnd,
		ExpiresAt:    time.Unix(advertisement.ExpiresAtUnix, 0),
	}
	return &advertisement, info, true
}
