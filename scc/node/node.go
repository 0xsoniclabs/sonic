// Copyright 2025 Sonic Operations Ltd
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

package node

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
)

// Node is a node in the Sonic Certification Chain. It is responsible for
// handling the progression of the chain by responding to new block statements
// and creating new certificates.
type Node struct {
	store Store
	state State
}

// NewNode creates a new node with the given store and committee. The provided
// committee is expected to be the active committee of the current block.
func NewNode(store Store, committee scc.Committee) *Node {
	return &Node{
		store: store,
		state: newInMemoryState(committee),
	}
}

// NewBlock should be called after a new block is added to the Sonic chain. It
// starts the creation of a corresponding block certificate and, if the block is
// the last one of the period, a new committee certificate for the following
// period.
func (n *Node) NewBlock(stmt cert.BlockStatement) error {
	blockCert := cert.NewCertificate(stmt)
	if err := n.store.UpdateBlockCertificate(blockCert); err != nil {
		return fmt.Errorf("failed to create block certificate, %w", err)
	}
	if !scc.IsLastBlockOfPeriod(stmt.Number) {
		return nil
	}
	committeeCert := cert.NewCertificate(cert.CommitteeStatement{
		Period:    scc.GetPeriod(stmt.Number) + 1,
		Committee: n.state.GetCurrentCommittee(),
	})
	return n.store.UpdateCommitteeCertificate(committeeCert)
}
