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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

func TestArchiveDirectory_ValidAdvertisement_AcceptedAndStored(t *testing.T) {
	directory := newTestDirectory()
	archive := newTestPeerID(t)
	message := advertisement(t, archive, 1000, 100, 200)

	if result := directory.Validate("sender", message); result != p2p.ValidationAccept {
		t.Fatalf("expected accept, got %v", result)
	}
	directory.Deliver("sender", message)

	archives := directory.Archives()
	if len(archives) != 1 || archives[0].Peer != archive {
		t.Fatalf("expected one stored archive for %s, got %+v", archive, archives)
	}
}

func TestArchiveDirectory_ExpiredAdvertisement_Ignored(t *testing.T) {
	directory := newTestDirectory()
	message := advertisement(t, newTestPeerID(t), -10, 100, 200) // already expired

	if result := directory.Validate("sender", message); result != p2p.ValidationIgnore {
		t.Fatalf("expected ignore for expired advertisement, got %v", result)
	}
}

func TestArchiveDirectory_InvalidRange_Rejected(t *testing.T) {
	directory := newTestDirectory()
	message := advertisement(t, newTestPeerID(t), 1000, 500, 200) // start > end

	if result := directory.Validate("sender", message); result != p2p.ValidationReject {
		t.Fatalf("expected reject for start>end, got %v", result)
	}
}

func TestArchiveDirectory_MalformedMessage_Rejected(t *testing.T) {
	directory := newTestDirectory()
	if result := directory.Validate("sender", []byte("not a protobuf")); result != p2p.ValidationReject {
		t.Fatalf("expected reject for malformed message, got %v", result)
	}
}

func TestArchiveDirectory_ExpiredEntry_PrunedFromResults(t *testing.T) {
	directory := newTestDirectory()
	current := time.Unix(1_000_000, 0)
	directory.now = func() time.Time { return current }

	message := advertisement(t, newTestPeerID(t), 60, 100, 200) // expires 60s later
	directory.Deliver("sender", message)
	if len(directory.Archives()) != 1 {
		t.Fatal("expected the archive to be live initially")
	}

	current = current.Add(120 * time.Second)
	if archives := directory.Archives(); len(archives) != 0 {
		t.Fatalf("expected expired archive to be pruned, got %+v", archives)
	}
}

func newTestDirectory() *ArchiveDirectory {
	directory := NewArchiveDirectory(log.Root())
	directory.now = func() time.Time { return time.Unix(1_000_000, 0) }
	return directory
}

// advertisement builds a marshalled ArchiveAdvertisement expiring ttlSeconds
// after the directory's fixed test clock (time.Unix(1_000_000, 0)).
func advertisement(t *testing.T, archive peer.ID, ttlSeconds int64, start, end uint64) []byte {
	t.Helper()
	message, err := proto.Marshal(&pb.ArchiveAdvertisement{
		PeerId:        []byte(archive),
		Addresses:     []string{"/ip4/127.0.0.1/tcp/4002"},
		HistoryStart:  start,
		HistoryEnd:    end,
		ExpiresAtUnix: time.Unix(1_000_000, 0).Add(time.Duration(ttlSeconds) * time.Second).Unix(),
	})
	if err != nil {
		t.Fatalf("failed to marshal advertisement: %v", err)
	}
	return message
}
