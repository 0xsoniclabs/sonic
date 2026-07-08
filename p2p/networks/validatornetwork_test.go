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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/p2p"
)

// *p2p.Node must satisfy the ValidatorNode interface the composition needs.
var _ ValidatorNode = (*p2p.Node)(nil)

// fastNetworkConfig makes convergence driven by publish-on-join and
// new-member-discovery re-publish rather than the periodic backstop.
var fastNetworkConfig = ValidatorNetworkConfig{
	Directory: ValidatorDirectoryConfig{
		RePublishInterval: 500 * time.Millisecond,
		MaxJitter:         20 * time.Millisecond,
		Debounce:          20 * time.Millisecond,
	},
}

func TestValidatorNetwork_TwoValidators_FormMesh(t *testing.T) {
	membership := membershipOf(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := []*p2p.Node{
		startValidator(t, ctx, membership, 0),
		startValidator(t, ctx, membership, 1),
	}
	bootstrap(t, ctx, nodes[1], nodes[0])

	require.True(t, waitForFullMesh(nodes, 20*time.Second), "expected the two validators to form a mesh")
}

func TestValidatorNetwork_FiveValidatorsJoiningSequentially_FormFullMesh(t *testing.T) {
	const count = 5
	membership := membershipOf(t, count)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var nodes []*p2p.Node
	for i := 0; i < count; i++ {
		node := startValidator(t, ctx, membership, i)
		if i > 0 {
			bootstrap(t, ctx, node, nodes[0]) // each joiner bootstraps to the first node
		}
		nodes = append(nodes, node)
		time.Sleep(200 * time.Millisecond) // stagger the joins
	}

	require.True(t, waitForFullMesh(nodes, 40*time.Second), "expected all %d validators to converge to a full mesh", count)
}

func TestValidatorNetwork_NonValidatorJoins_NotPulledIntoMesh(t *testing.T) {
	membership := membershipOf(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	validators := []*p2p.Node{
		startValidator(t, ctx, membership, 0),
		startValidator(t, ctx, membership, 1),
		startValidator(t, ctx, membership, 2),
	}
	bootstrap(t, ctx, validators[1], validators[0])
	bootstrap(t, ctx, validators[2], validators[0])

	require.True(t, waitForFullMesh(validators, 30*time.Second), "expected the three validators to form a full mesh")

	// A non-validator node joins the network by bootstrapping to a validator. It
	// is on the gossip network but not in the validators' membership.
	outsider := newTestNode(t)
	require.NoError(t, outsider.Start(), "failed to start outsider")
	t.Cleanup(func() { _ = outsider.Stop() })
	bootstrap(t, ctx, outsider, validators[0])

	// Give the network ample time; it must NOT pull the outsider into the mesh.
	time.Sleep(3 * time.Second)

	require.Less(t, connectedCount(outsider, validators), len(validators), "outsider must not be fully meshed with validators")
	require.True(t, fullyMeshed(validators), "the three validators must remain fully meshed")
}

// --- helpers ---

type staticMembership struct {
	members []Member
}

func (m *staticMembership) Members() []Member               { return m.members }
func (m *staticMembership) Epoch() uint64                   { return 1 }
func (m *staticMembership) OnChange(func()) (cancel func()) { return func() {} }

// memberKeys are the consensus keys generated for a test membership, indexed by
// validator position.
var memberKeys = map[*staticMembership][]Signer{}

func membershipOf(t *testing.T, count int) *staticMembership {
	t.Helper()
	membership := &staticMembership{}
	signers := make([]Signer, count)
	for i := 0; i < count; i++ {
		key, err := crypto.GenerateKey()
		require.NoError(t, err, "failed to generate consensus key")
		signer := NewSecp256k1Signer(key)
		signers[i] = signer
		membership.members = append(membership.members, Member{ID: uint32(i + 1), PublicKey: signer.PublicKey()})
	}
	memberKeys[membership] = signers
	return membership
}

func startValidator(t *testing.T, ctx context.Context, membership *staticMembership, index int) *p2p.Node {
	t.Helper()
	return startValidatorWithConfig(t, ctx, membership, index, fastNetworkConfig)
}

func startValidatorWithConfig(t *testing.T, ctx context.Context, membership *staticMembership, index int, config ValidatorNetworkConfig) *p2p.Node {
	t.Helper()
	node := newTestNode(t)
	signer := memberKeys[membership][index]
	validatorNetwork := NewValidatorNetwork(node, membership, signer, NewSecp256k1Verifier(), uint32(index+1), config)
	require.NoError(t, node.Start(), "failed to start node")
	validatorNetwork.Start(ctx)
	t.Cleanup(func() {
		validatorNetwork.Stop()
		_ = node.Stop()
	})
	return node
}

func newTestNode(t *testing.T) *p2p.Node {
	t.Helper()
	config := p2p.DefaultConfig()
	config.ListenAddresses = []string{
		"/ip4/127.0.0.1/udp/0/quic-v1",
		"/ip4/127.0.0.1/tcp/0",
	}
	node, err := p2p.New(config, log.Root(), prometheus.NewRegistry())
	require.NoError(t, err, "failed to create node")
	return node
}

func bootstrap(t *testing.T, ctx context.Context, from, to *p2p.Node) {
	t.Helper()
	info := peer.AddrInfo{ID: to.ID(), Addrs: to.Host().Addrs()}
	require.NoError(t, from.Connect(ctx, info), "bootstrap connect failed")
}

func waitForFullMesh(nodes []*p2p.Node, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fullyMeshed(nodes) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fullyMeshed(nodes)
}

func fullyMeshed(nodes []*p2p.Node) bool {
	for i, node := range nodes {
		for j, other := range nodes {
			if i == j {
				continue
			}
			if node.Host().Network().Connectedness(other.ID()) != network.Connected {
				return false
			}
		}
	}
	return true
}

func connectedCount(node *p2p.Node, others []*p2p.Node) int {
	count := 0
	for _, other := range others {
		if node.Host().Network().Connectedness(other.ID()) == network.Connected {
			count++
		}
	}
	return count
}
