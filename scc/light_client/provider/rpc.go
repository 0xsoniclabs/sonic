// Package provider contains the implementation of the Provider interface
package provider

import (
	"fmt"
	"math"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RPCProvider implements the Provider interface and provides methods for
// making RPC calls through an Ethereum client.
type RPCProvider struct {
	// client is the Ethereum client used for making RPC calls.
	client *ethclient.Client
}

// NewRPCProvider creates a new instance of RPCProvider with the given
// Ethereum client. Returns nil if the client could not be created.
// The resulting Provider must be closed after use.
//
// Parameters:
// - client: The Ethereum client to use for RPC calls.
//
// Returns:
// - *RPCProvider: A new instance of RPCProvider.
func NewRPCProvider(client *ethclient.Client) *RPCProvider {
	// TODO: get real URL and make my own client?
	return &RPCProvider{
		client: client,
	}
}

// Close closes the RPCProvider and its underlying Ethereum client.
func (rpcp *RPCProvider) Close() {
	rpcp.client.Close()
}

// GetCommitteeCertificate retrieves committee certificates for a given period
// and a maximum number of results.
//
// Parameters:
// - first: The starting period for which to retrieve committee certificates.
// - maxResults: The maximum number of committee certificates to retrieve.
//
// Returns:
// - []cert.CommitteeCertificate: A slice of committee certificates.
// - error: An error if the retrieval fails.
func (rpcp *RPCProvider) GetCommitteeCertificate(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	if rpcp.client == nil {
		return nil, fmt.Errorf("No client available")
	}

	maxString := fmt.Sprintf("%x", maxResults)
	if maxResults == math.MaxUint64 {
		maxString = "max"
	}
	results := []ethapi.CommitteeCertificateJson{}
	err := rpcp.client.Client().Call(
		&results, "sonic_getCommitteeCertificates",
		fmt.Sprintf("%x", first),
		maxString,
	)
	if err != nil {
		return nil, err
	}
	certs := []cert.CommitteeCertificate{}
	for _, res := range results {
		certs = append(certs, res.ToCertificate())
	}
	return certs, nil
}

// GetBlockCertificate returns the block certificate for the given block number.
//
// Parameters:
// - number: The block number for which to retrieve the block certificate.
//
// Returns:
// - cert.BlockCertificate: The block certificate for the given block number.
// - error: An error if the retrieval fails.
func GetBlockCertificate(number idx.Block) (cert.BlockCertificate, error) {
	return cert.BlockCertificate{}, nil
}
