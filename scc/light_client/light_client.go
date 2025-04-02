package light_client

import (
	"fmt"
	"net/url"
	"time"

	"github.com/0xsoniclabs/carmen/go/carmen"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// LightClient is the main entry point for the light client.
// It is responsible for managing the light client state and
// interacting with the provider.
type LightClient struct {
	provider provider
	state    state
}

// Config is used to configure the LightClient.
// It requires a list of URLs for the certificate providers and an initial committee.
type Config struct {
	Url     []*url.URL
	Genesis scc.Committee
	// By default, requests are retried up to 1024 times to reach a 10-second timeout.
	Retries uint
	Timeout time.Duration
}

// NewLightClient creates a new LightClient with the given config.
// Returns an error if the config does not contain a valid provider URL or committee.
func NewLightClient(config Config) (*LightClient, error) {
	if err := config.Genesis.Validate(); err != nil {
		return nil, fmt.Errorf("invalid committee provided: %w", err)
	}
	providers := make([]provider, len(config.Url))
	for _, u := range config.Url {
		var p provider
		p, err := newServerFromURL(u.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create provider: %w", err)
		}
		providers = append(providers, newRetry(p, config.Retries, config.Timeout))
	}
	p, err := newMultiplexer(providers...)
	if err != nil {
		return nil, fmt.Errorf("failed to create multiplexer: %w", err)
	}
	return &LightClient{
		state:    *newState(config.Genesis),
		provider: p,
	}, nil
}

// Close closes the light client provider.
// Closing an already closed client has no effect.
func (c *LightClient) Close() {
	c.provider.close()
}

// Sync updates the light client state using certificates from the provider.
// This serves as the primary method for synchronizing the light client state
// with the network.
func (c *LightClient) Sync() (idx.Block, error) {
	return c.state.sync(c.provider)
}

// getAccountProof retrieves and verifies the proof for the given address.
// Returns an error if the light client has not been synced or if the proof cannot be obtained.
func (c *LightClient) getAccountProof(address common.Address) (carmen.WitnessProof, error) {
	if !c.state.hasSynced {
		return nil, fmt.Errorf("light client has not yet synced")
	}
	proof, err := c.provider.getAccountProof(address, c.state.headNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get account proof: %w", err)
	}
	if proof == nil {
		return nil, fmt.Errorf("nil account proof for address %v", address)
	}
	return proof, nil
}

// GetBalance returns the balance of the given address.
// It returns an error if the light client has not been synced,
// the balance could not be proven or there was any error
// in getting or verifying the proof.
func (c *LightClient) GetBalance(address common.Address) (*uint256.Int, error) {
	proof, err := c.getAccountProof(address)
	if err != nil {
		return nil, fmt.Errorf("failed to get account proof: %w", err)
	}
	// it is safe to use the state.headRoot here because if the state had not been synced,
	// getAccountProof would have returned an error earlier.
	value, proven, err := proof.GetBalance(carmen.Hash(c.state.headRoot),
		carmen.Address(address))
	if err != nil {
		return nil, fmt.Errorf("failed to get balance from proof: %w", err)
	}
	if !proven {
		return nil,
			fmt.Errorf("balance could not be proven for account %v", address)
	}
	balance := value.Uint256()
	return &balance, err

}
