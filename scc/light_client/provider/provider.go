package provider

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=provider.go -package=provider -destination=provider_mock.go

// Provider is an interface to access certificates of the Sonic Certification Chain.
type Provider interface {
	// GetCommitteeCertificates returns the committee certificates for the
	// period speciefied as first until the maximum number of results specified as maxResults.
	GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error)

	// GetBlockCertificate returns the block certificates starting from first
	// until the maximum number of results specified as maxResults.
	GetBlockCertificate(first idx.Block, maxResults uint64) (cert.BlockCertificate, error)
}
