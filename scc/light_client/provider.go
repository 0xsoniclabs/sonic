package light_client

import (
	"math"

	"github.com/0xsoniclabs/carmen/go/carmen"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -source=provider.go -package=light_client -destination=provider_mock.go

// provider is an interface to access certificates of the Sonic Certification Chain.
type provider interface {

	// getCommitteeCertificates returns up to `maxResults` consecutive committee
	// certificates starting from the given period.
	//
	// Parameters:
	// - first: The starting period for which to retrieve committee certificates.
	// - maxResults: The maximum number of committee certificates to retrieve.
	//
	// Returns:
	//   - []cert.CommitteeCertificate: A slice of committee certificates.
	//   - error: Not nil if the provider failed to obtain the requested certificates.
	getCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error)

	// getBlockCertificates returns up to `maxResults` consecutive block
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
	getBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error)

	// GetAccountProof returns the account proof corresponding to the
	// given address at the given height.
	//
	// Parameters:
	// - address: The address of the account.
	// - height: The block height of the state.
	//
	// Returns:
	// - WitnessProof: witness proof for the account proof.
	// - error: Not nil if the provider failed to obtain the requested account proof.
	getAccountProof(address common.Address, height idx.Block) (carmen.WitnessProof, error)

	// close closes the Provider.
	// Closing an already closed provider has no effect
	close()
}

// LatestBlock is a constant used to indicate the latest block.
const LatestBlock = idx.Block(math.MaxUint64)
