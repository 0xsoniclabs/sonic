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
// It first ensures the client is synchronized before querying for the proof.
// Returns an error if synchronization fails, if the proof cannot be obtained.
func (c *LightClient) getAccountProof(address common.Address) (carmen.WitnessProof, error) {
	// always sync before querying
	_, err := c.Sync()
	if err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}
	proof, err := c.provider.getAccountProof(address, LatestBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to get account proof: %w", err)
	}
	if proof == nil {
		return nil, fmt.Errorf("nil account proof for address %v", address)
	}
	return proof, nil
}

// GetBalance returns the balance of the given address.
// It returns an error if the balance could not be proven or there was any error
// in getting or verifying the proof.
func (c *LightClient) GetBalance(address common.Address) (*uint256.Int, error) {
	balance, err := getInfoFromProof(address, c, "balance",
		func(
			proof carmen.WitnessProof,
			address common.Address,
			rootHash common.Hash,
		) (carmen.Amount, bool, error) {
			return proof.GetBalance(carmen.Hash(rootHash), carmen.Address(address))
		},
	)
	if err != nil {
		return nil, err
	}
	balanceInt := balance.Uint256()
	return &balanceInt, nil
}

// getInfoFromProof runs a function `f` that takes a proof, the provided address
// and the provided state root hash, to retrieve a value of type T.
//
// If the proof is missing, invalid, or does not confirm the requested information,
// an error is returned.
//
// Parameters:
// - address: The address whose data is being queried.
// - c: The LightClient instance handling the request.
// - valueName: A string representing the type of value being retrieved (e.g., "balance").
// - f: A function that takes (proof, address, rootHash) and returns (T, proven, error).
//
// Returns:
// - The requested value of type T if proven successfully, otherwise an error.
func getInfoFromProof[T any](address common.Address, c *LightClient, valueName string,
	f func(carmen.WitnessProof, common.Address, common.Hash) (T, bool, error)) (T, error) {
	var zeroValue T
	proof, err := c.getAccountProof(address)
	if err != nil {
		return zeroValue, fmt.Errorf("failed to get account proof: %w", err)
	}
	// it is safe to ignore the hasSynced flag here because if there was an error
	// during sync, it would have triggered an early return.
	rootHash, _ := c.state.stateRoot()
	value, proven, err := f(proof, address, rootHash)
	if err != nil {
		return zeroValue, fmt.Errorf("failed to get %v from proof: %w", valueName, err)
	}
	if !proven {
		return zeroValue,
			fmt.Errorf("%v could not be proven from the proof and state root hash",
				valueName)
	}
	return value, err
}
