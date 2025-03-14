package provider

import (
	"math"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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

	// GetAccountInfo returns the account info corresponding to the
	// given address at the given height.
	//
	// Parameters:
	// - address: The address of the account.
	// - height: The block height of the state.
	//
	// Returns:
	// - AccountInfo: The AccountInfo of the account at the given height.
	// - error: Not nil if the provider failed to obtain the requested account info.
	GetAccountInfo(address common.Address, height idx.Block) (AccountInfo, error)

	// Close closes the Provider.
	// Closing an already closed provider has no effect
	Close()
}

// LatestBlock is a constant used to indicate the latest block.
const LatestBlock = idx.Block(math.MaxUint64)

// AccountInfo represents proof data for an account's state.
// It includes the account's proof, balance, and nonce.
//
// Fields:
// - AccountProof: array of serialized nodes that prove the account's existence.
// - Balance: The account's balance in Wei.
// - Nonce: The nonce of the related account.
type AccountInfo struct {
	AccountProof []string
	Balance      uint256.Int
	Nonce        uint64
}
