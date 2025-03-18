package light_client

import (
	"fmt"
	"net/url"

	"github.com/0xsoniclabs/sonic/scc"
	lcs "github.com/0xsoniclabs/sonic/scc/light_client/light_client_state"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// LightClient is the main entry point for the light client.
// It is responsible for managing the light client state and
// interacting with the provider.
type LightClient struct {
	provider provider.Provider
	state    *lcs.State
}

// Config is used to configure the LightClient.
// It requires an url for the certificate provider and an initial committee.
type Config struct {
	Url     *url.URL
	Genesis scc.Committee
}

// NewLightClient creates a new LightClient with the given config.
// Returns an error if the config does not contain a valid provider url or committee.
func NewLightClient(config Config) (*LightClient, error) {
	if err := config.Genesis.Validate(); err != nil {
		return nil, fmt.Errorf("invalid committee provided: %w", err)
	}
	p, err := provider.NewServerFromURL(config.Url.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w\n", err)
	}
	return &LightClient{
		state:    lcs.NewState(config.Genesis),
		provider: p,
	}, nil
}

// Close closes the light client provider.
// Closing an already closed client has no effect
func (c *LightClient) Close() {
	c.provider.Close()
}

// Sync updates the light client state using certificates from the provider.
// This serves as the primary method for synchronizing the light client state
// with the network.
func (c *LightClient) Sync() (idx.Block, error) {
	return c.state.Sync(c.provider)
}
