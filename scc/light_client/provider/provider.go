package provider

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=provider.go -package=provider -destination=provider_mock.go

// Provider is an interface to access certificates of the Sonic Certification Chain.
type Provider interface {
	// GetCommitteeCertificates returns up to `maxResults` consecutive committee
	// certificates starting from the given period.
	GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error)

	// GetBlockCertificates returns up to `maxResults` consecutive block
	// certificates starting from the given block number.
	GetBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error)
}
