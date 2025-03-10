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

func TestNode_ProcessNewBlock_CreatesBlockCertificate(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockStore(ctrl)

	stmt := cert.BlockStatement{
		Number:    1,
		Hash:      common.Hash{1, 2, 3},
		StateRoot: common.Hash{4, 5},
	}
	store.EXPECT().UpdateBlockCertificate(cert.NewCertificate(stmt)).Return(nil)

	node := NewNode(store, scc.Committee{})
	_, _, err := node.ProcessNewBlock(stmt)
	require.NoError(t, err)
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

	_, _, err := node.ProcessNewBlock(blockStmt)
	require.NoError(t, err)
}

func TestNode_NewBlock_ReportsCertificateCreationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockStore(ctrl)

	stmt := cert.BlockStatement{}
	issue := fmt.Errorf("injected issue")
	store.EXPECT().UpdateBlockCertificate(cert.NewCertificate(stmt)).Return(issue)

	node := NewNode(store, scc.Committee{})
	_, _, err := node.ProcessNewBlock(stmt)
	require.ErrorIs(t, err, issue)
}
