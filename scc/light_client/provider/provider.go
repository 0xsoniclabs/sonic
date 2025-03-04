package provider

import (
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=provider.go -package=provider -destination=provider_mock.go

// Provider is an interface to access certificates of the Sonic Certification Chain.
type Provider interface {
	// GetCommitteeCertificate returns the committee certificate for the given chain ID and period.
	GetCommitteeCertificate(first scc.Period, maxResults uint64) (cert.CommitteeCertificate, error)
	// GetBlockCertificate returns the block certificate for the given chain ID and block number.
	GetBlockCertificate(number idx.Block) (cert.BlockCertificate, error)
}
