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
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/p2p"
	"github.com/0xsoniclabs/sonic/p2p/guard"
)

// TestValidatorHandshake_FailedHandshake_DisconnectsPeer has a non-validator
// actively open the validator-handshake stream and present a non-member proof.
// The validator must disconnect it.
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

	_ = attemptBadHandshake(ctx, attacker, validator)

	if !waitForConnectedness(validator, attacker.ID(), network.NotConnected, 10*time.Second) {
		t.Fatal("expected the validator to disconnect a peer that failed the handshake")
	}
}

// TestValidatorHandshake_SustainedFailures_BansPeer floods a validator with
// failed handshakes and asserts the attacker is eventually banned so re-dials are
// refused, and that the disconnect metric is recorded.
func TestValidatorHandshake_SustainedFailures_BansPeer(t *testing.T) {
	membership := membershipOf(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	registry := prometheus.NewRegistry()
	config := p2p.DefaultConfig()
	config.ListenAddresses = []string{"/ip4/127.0.0.1/udp/0/quic-v1", "/ip4/127.0.0.1/tcp/0"}
	validator, err := p2p.New(config, log.Root(), registry)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}
	validatorNetwork := NewValidatorNetwork(validator, membership, memberKeys[membership][0], NewSecp256k1Verifier(), 1,
		ValidatorNetworkConfig{
			Directory:            fastNetworkConfig.Directory,
			HandshakeFailures:    guard.FailureLimitConfig{FailureBurst: 2},
			HandshakeBanDuration: 3 * time.Second,
		})
	if err := validator.Start(); err != nil {
		t.Fatalf("failed to start validator: %v", err)
	}
	validatorNetwork.Start(ctx)
	t.Cleanup(func() { validatorNetwork.Stop(); _ = validator.Stop() })

	attacker := newTestNode(t)
	if err := attacker.Start(); err != nil {
		t.Fatalf("failed to start attacker: %v", err)
	}
	t.Cleanup(func() { _ = attacker.Stop() })
	bootstrap(t, ctx, attacker, validator)

	for i := 0; i < 10 && !validator.Gater().IsBanned(attacker.ID()); i++ {
		_ = attemptBadHandshake(ctx, attacker, validator)
		time.Sleep(100 * time.Millisecond)
	}

	if !validator.Gater().IsBanned(attacker.ID()) {
		t.Fatal("expected the attacker to be banned after sustained handshake failures")
	}
	if got := counterValue(t, registry, "sonic_p2p_peer_disconnects_total", "reason", "handshake-failure"); got < 1 {
		t.Fatalf("expected the handshake-failure disconnect metric to be recorded, got %v", got)
	}
}

// TestValidatorHandshake_NonAttemptingPeer_StaysConnected proves the change is
// scoped to handshake attempts: an archive-like non-validator that connects to a
// validator but never opens the handshake stream is neither disconnected nor
// banned.
func TestValidatorHandshake_NonAttemptingPeer_StaysConnected(t *testing.T) {
	membership := membershipOf(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validator := startValidator(t, ctx, membership, 0)

	archive := newTestNode(t) // non-validator; never opens the handshake stream
	if err := archive.Start(); err != nil {
		t.Fatalf("failed to start archive: %v", err)
	}
	t.Cleanup(func() { _ = archive.Stop() })
	bootstrap(t, ctx, archive, validator)

	// Wait out a window comparable to the handshake-abuse handling.
	time.Sleep(3 * time.Second)

	if validator.Host().Network().Connectedness(archive.ID()) != network.Connected {
		t.Fatal("expected a peer that never attempts the handshake to remain connected")
	}
	if validator.Gater().IsBanned(archive.ID()) {
		t.Fatal("a peer that never attempts the handshake must not be banned")
	}
}

// attemptBadHandshake opens the validator-handshake stream from `from` to `to`
// and sends a proof signed by a key that is not in the validator set. It returns
// any error opening/writing (e.g. once the attacker is banned).
func attemptBadHandshake(ctx context.Context, from, to *p2p.Node) error {
	key, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	proof, err := CreateBindingProof(NewSecp256k1Signer(key), from.ID(), 999, 1, nonce)
	if err != nil {
		return err
	}
	stream, err := from.OpenStream(ctx, to.ID(), HandshakeProtocolID)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()
	return stream.WriteMessage(proof, maxHandshakeSize)
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

// counterValue returns the value of the counter named metric whose label matches
// (labelName, labelValue), or 0 if absent.
func counterValue(t *testing.T, registry *prometheus.Registry, metric, labelName, labelValue string) float64 {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() != metric {
			continue
		}
		for _, entry := range family.GetMetric() {
			for _, label := range entry.GetLabel() {
				if label.GetName() == labelName && label.GetValue() == labelValue {
					return entry.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}
