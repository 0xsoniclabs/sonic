package provider

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RPCProvider implements the Provider interface and provides methods
// making RPC calls through an Ethereum client.
type RPCProvider struct {
	// client is the Ethereum client used for making RPC calls.
	client *ethclient.Client
}

// NewRPCProviderFromClient creates a new instance of RPCProvider with the given
// Ethereum client.
// The resulting Provider must be closed after use.
//
// Parameters:
// - client: The Ethereum client to use for RPC calls.
//
// Returns:
// - *RPCProvider: A new instance of RPCProvider.
func NewRPCProviderFromClient(client *ethclient.Client) *RPCProvider {
	return &RPCProvider{
		client: client,
	}
}

// NewRPCProviderFromURL creates a new instance of RPCProvider with a new Ethereum client
// connected to the given URL.
// The resulting Provider must be closed after use.
//
// Parameters:
// - url: The URL of the RPC node to connect to.
//
// Returns:
// - *RPCProvider: A new instance of RPCProvider.
// - error: An error if the connection fails.
func NewRPCProviderFromURL(url string) (*RPCProvider, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	return NewRPCProviderFromClient(client), nil
}

// Close closes the RPCProvider and its underlying Ethereum client.
func (rpcp *RPCProvider) Close() {
	rpcp.client.Close()
}

// GetCommitteeCertificates returns up to `maxResults` consecutive committee
// certificates starting from the given period.
//
// Parameters:
// - first: The starting period for which to retrieve committee certificates.
// - maxResults: The maximum number of committee certificates to retrieve.
//
// Returns:
// - []cert.CommitteeCertificate: A slice of committee certificates.
// - error: An error if the client is nil or the retrieval fails.
func (rpcp RPCProvider) GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	if rpcp.client == nil {
		return nil, fmt.Errorf("No client available")
	}

	results := []ethapi.CommitteeCertificateJson{}
	err := rpcp.client.Client().Call(
		&results, "sonic_getCommitteeCertificates",
		fmt.Sprintf("0x%x", first),
		fmt.Sprintf("0x%x", maxResults),
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

// GetBlockCertificates returns up to `maxResults` consecutive block
// certificates starting from the given block number.
//
// Parameters:
// - number: The starting block number for which to retrieve the block certificate.
// - maxResults: The maximum number of block certificates to retrieve.
//
// Returns:
// - cert.BlockCertificate: The block certificates for the given block number and the following blocks.
// - error: An error if the client is nil or the retrieval fails.
func (rpcp RPCProvider) GetBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
	if rpcp.client == nil {
		return nil, fmt.Errorf("No client available")
	}

	results := []ethapi.BlockCertificateJson{}
	err := rpcp.client.Client().Call(
		&results, "sonic_getBlockCertificates",
		fmt.Sprintf("0x%x", first),
		fmt.Sprintf("0x%x", maxResults),
	)
	if err != nil {
		return nil, err
	}
	certs := []cert.BlockCertificate{}
	for _, res := range results {
		certs = append(certs, res.ToCertificate())
	}
	return certs, nil
}
