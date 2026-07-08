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
	"crypto/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/network"

	"github.com/0xsoniclabs/sonic/p2p"
)

// TestValidatorHandshake_FailedHandshake_DisconnectsPeer has a non-validator
// actively open the validator-handshake stream and present a non-member proof.
// The validator must disconnect it. Until Handle disconnects on failure it only
// resets the stream, leaving the peer connected, so this test fails.
func TestValidatorHandshake_FailedHandshake_DisconnectsPeer(t *testing.T) {
	membership := membershipOf(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validator := startValidator(t, ctx, membership, 0)

	attacker := newTestNode(t)
	if err := attacker.Start(); err != nil {
		t.Fatalf("failed to start attacker: %v", err)
	}
	t.Cleanup(func() { _ = attacker.Stop() })
	bootstrap(t, ctx, attacker, validator)

	sendBadHandshake(t, ctx, attacker, validator)

	if !waitForConnectedness(validator, attacker.ID(), network.NotConnected, 10*time.Second) {
		t.Fatal("expected the validator to disconnect a peer that failed the handshake")
	}
}

// sendBadHandshake opens the validator-handshake stream from `from` to `to` and
// sends a well-formed proof signed by a key that is not in the validator set.
func sendBadHandshake(t *testing.T, ctx context.Context, from, to *p2p.Node) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("failed to read nonce: %v", err)
	}
	proof, err := CreateBindingProof(NewSecp256k1Signer(key), from.ID(), 999, 1, nonce)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}
	stream, err := from.OpenStream(ctx, to.ID(), HandshakeProtocolID)
	if err != nil {
		t.Fatalf("failed to open handshake stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	if err := stream.WriteMessage(proof, maxHandshakeSize); err != nil {
		t.Fatalf("failed to write handshake: %v", err)
	}
}

func waitForConnectedness(node *p2p.Node, target p2p.PeerID, want network.Connectedness, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if node.Host().Network().Connectedness(target) == want {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
