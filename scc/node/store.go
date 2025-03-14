package node

import (
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
)

//go:generate mockgen -source=store.go -destination=store_mock.go -package=node

// Store is an interface for a persistent store of certificates. A store is
// required to facilitate the creation of certificates and the gradual
// aggregation of signatures.
type Store interface {
	// GetCommitteeCertificate retrieves the certificate for the given period.
	// If no certificate is found, an error is returned.
	GetCommitteeCertificate(scc.Period) (cert.CommitteeCertificate, error)

	// GetBlockCertificate retrieves the certificate for the given block.
	// If no certificate is found, an error is returned.
	GetBlockCertificate(idx.Block) (cert.BlockCertificate, error)

	// UpdateCommitteeCertificate adds or updates the certificate in the store.
	// If a certificate for the same period is already present, it is overwritten.
	UpdateCommitteeCertificate(cert.CommitteeCertificate) error

	// UpdateBlockCertificate adds or updates the certificate in the store.
	// If a certificate for the same block is already present, it is overwritten.
	UpdateBlockCertificate(cert.BlockCertificate) error

	// GetLatestCertificationChainState retrieves the most recent certification
	// chain state available in the store. States are ordered by their block
	// height. If there is none, an error is returned.
	GetLatestCertificationChainState() (State, error)

	// AddCertificationChainState adds a new certification chain state to the
	// store. States are indexed by their block height. If there is a state for
	// the same block hight already in the store, it is overwritten.
	AddCertificationChainState(State) error
}
