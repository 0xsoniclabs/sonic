package gossip

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
)

func TestEthApiBackend_GetNetworkRules_LoadsRulesFromEpoch(t *testing.T) {
	require := require.New(t)

	blockNumber := idx.Block(12)
	epoch := idx.Epoch(3)

	store, err := NewMemStore(t)
	require.NoError(err)

	store.SetBlock(
		blockNumber,
		inter.NewBlockBuilder().
			WithNumber(uint64(blockNumber)).
			WithEpoch(epoch).
			Build(),
	)
	require.True(store.HasBlock(blockNumber))

	rules := opera.FakeNetRules(opera.Upgrades{})
	rules.Name = "test-rules"

	store.SetHistoryBlockEpochState(
		epoch,
		iblockproc.BlockState{},
		iblockproc.EpochState{
			Epoch: epoch,
			Rules: rules,
		},
	)

	backend := &EthAPIBackend{
		svc: &Service{
			store: store,
		},
		state: &EvmStateReader{
			store: store,
		},
	}

	got, err := backend.GetNetworkRules(t.Context(), blockNumber)
	require.NoError(err)

	// Rules contain functions that cannot be compared directly,
	// so we compare their string representations.
	want := fmt.Sprintf("%+v", rules)
	have := fmt.Sprintf("%+v", got)
	require.Equal(want, have, "Network rules do not match")
}

func TestEthApiBackend_GetNetworkRules_MissingBlockReturnsNilRules(t *testing.T) {
	require := require.New(t)

	blockNumber := idx.Block(12)

	store, err := NewMemStore(t)
	require.NoError(err)
	require.False(store.HasBlock(blockNumber))

	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}

	rules, err := backend.GetNetworkRules(t.Context(), blockNumber)
	require.NoError(err)
	require.Nil(rules)
}
