package provider

import (
	"math"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=provider.go -package=provider -destination=provider_mock.go

// Provider is an interface to access certificates of the Sonic Certification Chain.
type Provider interface {

	// GetCommitteeCertificates returns up to `maxResults` consecutive committee
	// certificates starting from the given period.
	//
	// Parameters:
	// - first: The starting period for which to retrieve committee certificates.
	// - maxResults: The maximum number of committee certificates to retrieve.
	//
	// Returns:
	//   - []cert.CommitteeCertificate: A slice of committee certificates.
	//   - error: Not nil if the provider failed to obtain the requested certificates.
	GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error)

	// GetBlockCertificates returns up to `maxResults` consecutive block
	// certificates starting from the given block number.
	//
	// Parameters:
	//   - number: The starting block number for which to retrieve the block certificate.
	//     Can be LatestBlock to retrieve the latest certificates.
	//   - maxResults: The maximum number of block certificates to retrieve.
	//
	// Returns:
	//   - cert.BlockCertificate: The block certificates for the given block number
	//     and the following blocks.
	//   - error: Not nil if the provider failed to obtain the requested certificates.
	GetBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error)

	// Close closes the Provider.
	// Closing an already closed provider has no effect
	Close()
}

// LatestBlock is a constant used to indicate the latest block.
const LatestBlock = idx.Block(math.MaxUint64)
