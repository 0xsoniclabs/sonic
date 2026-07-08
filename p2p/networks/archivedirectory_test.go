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
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/pb"
)

func TestArchiveDirectory_ValidAdvertisement_AcceptedAndStored(t *testing.T) {
	directory := newTestDirectory()
	archive := newTestPeerID(t)
	message := advertisement(t, archive, 1000, 100, 200)

	require.Equal(t, p2p.ValidationAccept, directory.Validate("sender", message), "expected accept")
	directory.Deliver("sender", message)

	archives := directory.Archives()
	require.Len(t, archives, 1, "expected one stored archive for %s", archive)
	require.Equal(t, archive, archives[0].Peer)
}

func TestArchiveDirectory_ExpiredAdvertisement_Ignored(t *testing.T) {
	directory := newTestDirectory()
	message := advertisement(t, newTestPeerID(t), -10, 100, 200) // already expired

	require.Equal(t, p2p.ValidationIgnore, directory.Validate("sender", message), "expected ignore for expired advertisement")
}

func TestArchiveDirectory_InvalidRange_Rejected(t *testing.T) {
	directory := newTestDirectory()
	message := advertisement(t, newTestPeerID(t), 1000, 500, 200) // start > end

	require.Equal(t, p2p.ValidationReject, directory.Validate("sender", message), "expected reject for start>end")
}

func TestArchiveDirectory_MalformedMessage_Rejected(t *testing.T) {
	directory := newTestDirectory()
	require.Equal(t, p2p.ValidationReject, directory.Validate("sender", []byte("not a protobuf")), "expected reject for malformed message")
}

func TestArchiveDirectory_ExpiredEntry_PrunedFromResults(t *testing.T) {
	directory := newTestDirectory()
	current := time.Unix(1_000_000, 0)
	directory.now = func() time.Time { return current }

	message := advertisement(t, newTestPeerID(t), 60, 100, 200) // expires 60s later
	directory.Deliver("sender", message)
	require.Len(t, directory.Archives(), 1, "expected the archive to be live initially")

	current = current.Add(120 * time.Second)
	require.Empty(t, directory.Archives(), "expected expired archive to be pruned")
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
	require.NoError(t, err, "failed to marshal advertisement")
	return message
}
