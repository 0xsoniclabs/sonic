package provider

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rpc"
)

// RpcProvider implements the Provider interface and provides methods
// making RPC calls through an RPC client.
type RpcProvider struct {
	// client is the RPC client used for making RPC calls.
	client RpcClient
}

// NewRpcProviderFromClient creates a new RpcProvider with the given
// RPC client. The resulting Provider takes ownership of the client and
// will close it when the Provider is closed.
// The resulting Provider must be closed after use.
//
// Parameters:
// - client: The RPC client to use for RPC calls.
//
// Returns:
// - *RpcProvider: A new instance of RpcProvider.
// - error: An error if the client is nil.
func NewRpcProviderFromClient(client RpcClient) (*RpcProvider, error) {
	if client == nil {
		return nil, fmt.Errorf("cannot start a provider with a nil client")
	}
	return &RpcProvider{
		client: client,
	}, nil
}

// NewRpcProviderFromURL creates a new instance of RpcProvider with a new RPC client
// connected to the given URL.
// The resulting Provider must be closed after use.
//
// Parameters:
// - url: The URL of the RPC node to connect to.
//
// Returns:
// - *RpcProvider: A new instance of RpcProvider.
// - error: An error if the connection fails.
func NewRpcProviderFromURL(url string) (*RpcProvider, error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	return NewRpcProviderFromClient(client)
}

// Close closes the RpcProvider.
// Closing an already closed provider has no effect
func (rpcp *RpcProvider) Close() {
	if rpcp.IsClosed() {
		return
	}
	rpcp.client.Close()
	rpcp.client = nil
}

// IsClosed returns true if the RpcProvider is closed.
func (rpcp RpcProvider) IsClosed() bool {
	return rpcp.client == nil
}

// GetCommitteeCertificates returns up to `maxResults` consecutive committee
// certificates starting from the given period.
//
// Parameters:
// - first: The starting period for which to retrieve committee certificates.
// - maxResults: The maximum number of committee certificates to retrieve.
//
// Returns:
//   - []cert.CommitteeCertificate: A slice of committee certificates.
//   - error: An error if the call fails or the certificates are out of order.
func (rpcp RpcProvider) GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	if rpcp.IsClosed() {
		return nil, fmt.Errorf("no client available")
	}

	results := []ethapi.CommitteeCertificate{}
	err := rpcp.client.Call(
		&results,
		"sonic_getCommitteeCertificates",
		fmt.Sprintf("0x%x", first),
		fmt.Sprintf("0x%x", maxResults),
	)
	if err != nil {
		return nil, err
	}
	// if too many certificates are returned, drop the excess
	if uint64(len(results)) > maxResults {
		results = results[:maxResults]
	}
	certs := []cert.CommitteeCertificate{}
	currentPeriod := first
	for _, res := range results {
		if res.Period != uint64(currentPeriod) {
			return nil, fmt.Errorf("committee certificates out of order")
		}
		currentPeriod++
		certs = append(certs, res.ToCertificate())
	}
	return certs, nil
}

// GetBlockCertificates returns up to `maxResults` consecutive block
// certificates starting from the given block number.
//
// Parameters:
//   - number: The starting block number for which to retrieve the block certificate.
//     Can be LatestPeriod to retrieve the latest certificates.
//   - maxResults: The maximum number of block certificates to retrieve.
//
// Returns:
//   - cert.BlockCertificate: The block certificates for the given block number
//     and the following blocks.
//   - error: An error if the client is nil, the call fails, the
//     certificates are out of order or more than requested.
func (rpcp RpcProvider) GetBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
	if rpcp.IsClosed() {
		return nil, fmt.Errorf("no client available")
	}

	var firstString string
	if first == LatestBlock {
		firstString = "latest"
	} else {
		firstString = fmt.Sprintf("0x%x", first)
	}
	results := []ethapi.BlockCertificate{}
	err := rpcp.client.Call(
		&results,
		"sonic_getBlockCertificates",
		firstString,
		fmt.Sprintf("0x%x", maxResults),
	)
	if err != nil {
		return nil, err
	}
	// if too many certificates are returned, drop the excess
	if uint64(len(results)) > maxResults {
		results = results[:maxResults]
	}
	certs := []cert.BlockCertificate{}
	var currentBlock idx.Block
	if first == LatestBlock {
		currentBlock = idx.Block(results[0].Number)
	} else {
		currentBlock = first
	}
	for _, res := range results {
		if res.Number != uint64(currentBlock) {
			return nil, fmt.Errorf("block certificates out of order")
		}
		currentBlock++
		certs = append(certs, res.ToCertificate())
	}
	return certs, nil
}
