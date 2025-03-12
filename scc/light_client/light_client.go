package light_client

import (
	"fmt"
	"slices"

	"github.com/0xsoniclabs/sonic/scc"
	lc_state "github.com/0xsoniclabs/sonic/scc/light_client/light_client_state"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// LightClient is the main entry point for the light client.
// It is responsible for managing the light client state and
// interacting with the provider.
type LightClient struct {
	provider provider.Provider
	state    *lc_state.State
}

// Config is used to configure the LightClient.
// It requires an url for the certificate providers and an initial committee.
type Config struct {
	Provider string
	Genesis  scc.Committee
}

// NewLightClient creates a new LightClient with the given config.
// returns an error if the config does not contain a valid provider or committee.
func NewLightClient(config Config) (*LightClient, error) {
	if err := config.Genesis.Validate(); err != nil {
		return nil, fmt.Errorf("invalid committee provided: %w", err)
	}
	p, err := provider.NewServerFromURL(config.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w\n", err)
	}
	return &LightClient{
		state: lc_state.NewState(
			scc.NewCommittee(
				slices.Clone(config.Genesis.Members())...)),
		provider: p,
	}, nil
}

// Close closes the light client provider.
func (c *LightClient) Close() {
	c.provider.Close()
}

// Sync updates the light client state using certificates from the provider.
// This serves as the primary method for synchronizing the light client state
// with the network.
func (c *LightClient) Sync() (idx.Block, error) {
	return c.state.Sync(c.provider)
}
