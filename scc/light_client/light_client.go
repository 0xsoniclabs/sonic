package light_client

import (
	"fmt"
	"net/url"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// LightClient is the main entry point for the light client.
// It is responsible for managing the light client state and
// interacting with the provider.
type LightClient struct {
	provider provider
	state    state
}

// Config is used to configure the LightClient.
// It requires a list of urls for the certificate providers and an initial committee.
type Config struct {
	Url     []*url.URL
	Genesis scc.Committee
	// By default, requests are retried up to 1024 times to reach a 10-second timeout.
	Retries uint
	Timeout time.Duration
}

// NewLightClient creates a new LightClient with the given config.
// Returns an error if the config does not contain a valid provider url or committee.
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
// Closing an already closed client has no effect
func (c *LightClient) Close() {
	c.provider.close()
}

// Sync updates the light client state using certificates from the provider.
// This serves as the primary method for synchronizing the light client state
// with the network.
func (c *LightClient) Sync() (idx.Block, error) {
	return c.state.sync(c.provider)
}
