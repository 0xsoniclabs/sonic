package gossip

import (
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/stretchr/testify/require"
)

func TestStore_GetCommitteeCertificate_FailsIfNotPresent(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)
	_, err = store.GetCommitteeCertificate(1)
	require.ErrorContains(err, "no such certificate")
}

func TestStore_GetCommitteeCertificate_RetrievesPresentEntries(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	original := cert.NewCertificate(cert.CommitteeStatement{
		Period: 1,
	})

	require.NoError(store.UpdateCommitteeCertificate(original))

	restored, err := store.GetCommitteeCertificate(1)
	require.NoError(err)
	require.Equal(original, restored)
}

func TestStore_GetCommitteeCertificate_DistinguishesBetweenPeriods(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	original1 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(),
	})

	original2 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    2,
		Committee: scc.NewCommittee(scc.Member{}),
	})
	require.NotEqual(original1, original2)

	require.NoError(store.UpdateCommitteeCertificate(original1))
	require.NoError(store.UpdateCommitteeCertificate(original2))

	restored1, err := store.GetCommitteeCertificate(1)
	require.NoError(err)
	require.Equal(original1, restored1)

	restored2, err := store.GetCommitteeCertificate(2)
	require.NoError(err)
	require.Equal(original2, restored2)
}

func TestStore_UpdateCommitteeCertificate_CanOverrideExisting(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	original1 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(scc.Member{}),
	})
	require.NoError(store.UpdateCommitteeCertificate(original1))

	original2 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(scc.Member{}, scc.Member{}),
	})
	require.NotEqual(original1, original2)
	require.NoError(store.UpdateCommitteeCertificate(original2))

	restored, err := store.GetCommitteeCertificate(1)
	require.NoError(err)
	require.Equal(original2, restored)
}

func TestStore_GetBlockCertificate_FailsIfNotPresent(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)
	_, err = store.GetBlockCertificate(1)
	require.ErrorContains(err, "no such certificate")
}

func TestStore_GetBlockCertificate_RetrievesPresentEntries(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	original := cert.NewCertificate(cert.BlockStatement{
		Number: 1,
	})

	require.NoError(store.UpdateBlockCertificate(original))

	restored, err := store.GetBlockCertificate(1)
	require.NoError(err)
	require.Equal(original, restored)
}

func TestStore_GetBlockCertificate_DistinguishesBetweenPeriods(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	original1 := cert.NewCertificate(cert.BlockStatement{
		Number: 1,
		Hash:   [32]byte{1, 2, 3},
	})

	original2 := cert.NewCertificate(cert.BlockStatement{
		Number: 2,
		Hash:   [32]byte{4, 5, 6},
	})
	require.NotEqual(original1, original2)

	require.NoError(store.UpdateBlockCertificate(original1))
	require.NoError(store.UpdateBlockCertificate(original2))

	restored1, err := store.GetBlockCertificate(1)
	require.NoError(err)
	require.Equal(original1, restored1)

	restored2, err := store.GetBlockCertificate(2)
	require.NoError(err)
	require.Equal(original2, restored2)
}

func TestStore_UpdateBlockCertificate_CanOverrideExisting(t *testing.T) {
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	original1 := cert.NewCertificate(cert.BlockStatement{
		Number: 1,
		Hash:   [32]byte{1, 2, 3},
	})
	require.NoError(store.UpdateBlockCertificate(original1))

	original2 := cert.NewCertificate(cert.BlockStatement{
		Number: 1,
		Hash:   [32]byte{4, 5, 6},
	})
	require.NotEqual(original1, original2)
	require.NoError(store.UpdateBlockCertificate(original2))

	restored, err := store.GetBlockCertificate(1)
	require.NoError(err)
	require.Equal(original2, restored)
}
