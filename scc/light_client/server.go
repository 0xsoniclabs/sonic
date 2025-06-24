// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package light_client

import (
	"fmt"

	"github.com/0xsoniclabs/carmen/go/carmen"
	"github.com/0xsoniclabs/carmen/go/common/immutable"
	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	idx "github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// server implements the Provider interface and provides methods
// making RPC calls through an RPC client.
type server struct {
	// client is the RPC client used for making RPC calls.
	client rpcClient
}

// newServerFromClient creates a new Server with the given
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
func newServerFromClient(client rpcClient) (*server, error) {
	if client == nil {
		return nil, fmt.Errorf("cannot start a Server with a nil client")
	}
	return &server{
		client: client,
	}, nil
}

// newServerFromURL creates a new instance of Server with a new RPC client
// connected to the given URL.
// The resulting Server must be closed after use.
//
// Parameters:
// - url: The URL of the RPC node to connect to.
//
// Returns:
// - *Server: A new instance of Server.
// - error: An error if the connection fails.
func newServerFromURL(url string) (*server, error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	return newServerFromClient(client)
}

// close closes the Server.
// Closing an already closed Server has no effect
func (s *server) close() {
	if s.IsClosed() {
		return
	}
	s.client.Close()
	s.client = nil
}

// IsClosed returns true if the Server is closed.
func (s server) IsClosed() bool {
	return s.client == nil
}

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
func (s server) getCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
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
func (s server) getBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
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

// GetAccountProof returns the account proof corresponding to the
// given address at the given height.
//
// Parameters:
// - address: The address of the account.
// - height: The block height of the state.
//
// Returns:
// - AccountProof: The proof of the account at the given height.
// - error: Not nil if the provider failed to obtain the requested account proof.
func (s server) getAccountProof(address common.Address, height idx.Block) (carmen.WitnessProof, error) {
	heightString := fmt.Sprintf("0x%x", height)
	if height == LatestBlock {
		heightString = "latest"
	}
	var result struct {
		AccountProof []string
	}
	err := s.client.Call(
		&result,
		"eth_getProof",
		fmt.Sprintf("%v", address),
		[]string{},
		heightString,
	)
	if err != nil {
		return nil, err
	}

	// decode elements for the proof.
	elements := []carmen.Bytes{}
	for _, element := range result.AccountProof {
		data, err := hexutil.Decode(element)
		if err != nil {
			return nil, fmt.Errorf("failed to decode proof element: %v", err)
		}
		elements = append(elements, immutable.NewBytes(data))
	}
	proof := carmen.CreateWitnessProofFromNodes(elements...)
	if !proof.IsValid() {
		return nil, fmt.Errorf("invalid proof")
	}
	return proof, nil
}
