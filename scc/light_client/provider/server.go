package provider

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rpc"
)

// Server implements the Provider interface and provides methods
// making RPC calls through an RPC client.
type Server struct {
	// client is the RPC client used for making RPC calls.
	client RpcClient
}

// NewServerFromClient creates a new Server with the given
// RPC client. The resulting Server takes ownership of the client and
// will close it when the Server is closed.
// The resulting Server must be closed after use.
//
// Parameters:
// - client: The RPC client to use for RPC calls.
//
// Returns:
// - *Server: A new instance of Server.
// - error: An error if the client is nil.
func NewServerFromClient(client RpcClient) (*Server, error) {
	if client == nil {
		return nil, fmt.Errorf("cannot start a Server with a nil client")
	}
	return &Server{
		client: client,
	}, nil
}

// NewServerFromURL creates a new instance of Server with a new RPC client
// connected to the given URL.
// The resulting Server must be closed after use.
//
// Parameters:
// - url: The URL of the RPC node to connect to.
//
// Returns:
// - *Server: A new instance of Server.
// - error: An error if the connection fails.
func NewServerFromURL(url string) (*Server, error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	return NewServerFromClient(client)
}

// Close closes the Server.
// Closing an already closed Server has no effect
func (s *Server) Close() {
	if s.IsClosed() {
		return
	}
	s.client.Close()
	s.client = nil
}

// IsClosed returns true if the Server is closed.
func (s Server) IsClosed() bool {
	return s.client == nil
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
//   - error: Not nil if the provider failed to obtain the requested certificates.
func (s Server) GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	if maxResults == 0 {
		return nil, nil
	}
	if s.IsClosed() {
		return nil, fmt.Errorf("no client available")
	}

	results := []ethapi.CommitteeCertificate{}
	err := s.client.Call(
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
	certs := make([]cert.CommitteeCertificate, len(results))
	currentPeriod := first
	for i, res := range results {
		if res.Period != uint64(currentPeriod) {
			return nil, fmt.Errorf("committee certificates out of order")
		}
		currentPeriod++
		certs[i] = res.ToCertificate()
	}
	return certs, nil
}

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
func (s Server) GetBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
	if maxResults == 0 {
		return nil, nil
	}
	if s.IsClosed() {
		return nil, fmt.Errorf("no client available")
	}

	var firstString string
	if first == LatestBlock {
		firstString = "latest"
	} else {
		firstString = fmt.Sprintf("0x%x", first)
	}
	results := []ethapi.BlockCertificate{}
	err := s.client.Call(
		&results,
		"sonic_getBlockCertificates",
		firstString,
		fmt.Sprintf("0x%x", maxResults),
	)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no block certificates found")
	}
	// if too many certificates are returned, drop the excess
	if uint64(len(results)) > maxResults {
		results = results[:maxResults]
	}
	certs := make([]cert.BlockCertificate, len(results))
	var currentBlock idx.Block
	if first == LatestBlock {
		currentBlock = idx.Block(results[0].Number)
	} else {
		currentBlock = first
	}
	for i, res := range results {
		if res.Number != uint64(currentBlock) {
			return nil, fmt.Errorf("block certificates out of order")
		}
		currentBlock++
		certs[i] = res.ToCertificate()
	}
	return certs, nil
}
