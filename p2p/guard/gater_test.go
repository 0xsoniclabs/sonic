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

package guard

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestGater_BannedPeer_RefusedAtDialAndSecured(t *testing.T) {
	gater := NewGater()
	banned := testPeerID(t)
	permitted := testPeerID(t)

	gater.Ban(banned)

	if gater.InterceptPeerDial(banned) {
		t.Fatal("expected outbound dial to a banned peer to be refused")
	}
	if gater.InterceptSecured(network.DirInbound, banned, nil) {
		t.Fatal("expected inbound secured connection from a banned peer to be refused")
	}
	if !gater.InterceptPeerDial(permitted) {
		t.Fatal("expected dial to a non-banned peer to be permitted")
	}
}

func TestGater_BanUntil_ExpiresAfterCooldown(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	gater := NewGater()
	gater.now = func() time.Time { return now }
	peerID := testPeerID(t)

	gater.BanUntil(peerID, now.Add(time.Minute))
	if !gater.IsBanned(peerID) {
		t.Fatal("expected peer to be banned before the cooldown elapses")
	}

	now = now.Add(2 * time.Minute)
	if gater.IsBanned(peerID) {
		t.Fatal("expected peer to be unbanned after the cooldown elapses")
	}
}

func TestGater_Ban_IsPermanent(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	gater := NewGater()
	gater.now = func() time.Time { return now }
	peerID := testPeerID(t)

	gater.Ban(peerID)
	now = now.Add(1000 * time.Hour)
	if !gater.IsBanned(peerID) {
		t.Fatal("expected a permanent ban to persist indefinitely")
	}
}

func TestGater_Unban_RemovesBan(t *testing.T) {
	gater := NewGater()
	peerID := testPeerID(t)
	gater.Ban(peerID)
	gater.Unban(peerID)
	if gater.IsBanned(peerID) {
		t.Fatal("expected peer to be permitted after being unbanned")
	}
}

func testPeerID(t *testing.T) peer.ID {
	t.Helper()
	_, public, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	id, err := peer.IDFromPublicKey(public)
	if err != nil {
		t.Fatalf("failed to derive peer ID: %v", err)
	}
	return id
}
