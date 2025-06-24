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
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNode_NewBlock_CreatesBlockCertificate(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockStore(ctrl)

	stmt := cert.BlockStatement{
		Number:    1,
		Hash:      common.Hash{1, 2, 3},
		StateRoot: common.Hash{4, 5},
	}
	store.EXPECT().UpdateBlockCertificate(cert.NewCertificate(stmt)).Return(nil)

	node := NewNode(store, scc.Committee{})
	require.NoError(t, node.NewBlock(stmt))
}

func TestNode_NewBlock_CreatesCommitteeCertificateAtEndOfPeriod(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockStore(ctrl)
	state := NewMockState(ctrl)

	committeeA := scc.NewCommittee()
	committeeB := scc.NewCommittee(scc.Member{})

	blockStmt := cert.BlockStatement{
		Number:    2*scc.BLOCKS_PER_PERIOD - 1,
		Hash:      common.Hash{1, 2, 3},
		StateRoot: common.Hash{4, 5},
	}
	require.True(t, scc.IsLastBlockOfPeriod(blockStmt.Number))

	store.EXPECT().UpdateBlockCertificate(cert.NewCertificate(blockStmt)).Return(nil)
	store.EXPECT().UpdateCommitteeCertificate(cert.NewCertificate(cert.CommitteeStatement{
		Period:    2,
		Committee: committeeB,
	}))
	state.EXPECT().GetCurrentCommittee().Return(committeeB)

	node := NewNode(store, committeeA)
	node.state = state

	require.NoError(t, node.NewBlock(blockStmt))
}

func TestNode_NewBlock_ReportsCertificateCreationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockStore(ctrl)

	stmt := cert.BlockStatement{}
	issue := fmt.Errorf("injected issue")
	store.EXPECT().UpdateBlockCertificate(cert.NewCertificate(stmt)).Return(issue)

	node := NewNode(store, scc.Committee{})
	require.ErrorIs(t, node.NewBlock(stmt), issue)
}
