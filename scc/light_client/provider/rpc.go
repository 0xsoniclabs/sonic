package provider

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RPCProvider implements the Provider interface and adds methods for RPC calls.
type RPCProvider struct {
	// Embed Provider interface to enbable decorator pattern
	Provider

	// client is the Ethereum client to use for RPC calls.
	client *ethclient.Client
}

// NewRPCProvider creates a new RPCProvider with the given Ethereum client.
// Returns nil if the client could not be created.
func NewRPCProvider(provider Provider, client *ethclient.Client) *RPCProvider {
	// TODO: get real client
	// client, err := ethclient.Dial(fmt.Sprintf("http://localhost:%d", 0))
	// if err != nil {
	// 	return nil
	// }
	return &RPCProvider{
		Provider: provider,
		client:   client,
	}
}

func (rpcp *RPCProvider) GetCommitteeCertificate(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	if rpcp.client == nil && rpcp.Provider == nil {
		return nil, fmt.Errorf("Can not fetch certificates from provider nor network")
	}
	results := []cert.CommitteeCertificate{}
	if rpcp.client != nil {
		err := rpcp.client.Client().Call(&results, "sonic_getCommitteeCertificates", "0x0", "max")
		if err != nil {
			return nil, err
		}
		return results, nil
	}
	if rpcp.Provider != nil {

	}
	return nil, nil
}

// GetBlockCertificate returns the block certificate for the given chain ID and block number.
func GetBlockCertificate(number idx.Block) (cert.BlockCertificate, error) {
	return cert.BlockCertificate{}, nil
}
