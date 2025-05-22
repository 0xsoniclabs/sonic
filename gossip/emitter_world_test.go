package gossip

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestEmitterWorldProc_GetUpgradeHeights_TakesResultOfUnderlyingStore(t *testing.T) {
	world := &emitterWorldProc{
		s: &Service{
			store: initStoreForTests(t),
		},
	}

	got := world.GetUpgradeHeights()
	want := world.s.store.GetUpgradeHeights()
	require.Equal(t, want, got)
}

func TestEmitterWorldProc_GetHeader_UsesStateReaderToResolveHeader(t *testing.T) {
	store := initStoreForTests(t)
	world := &emitterWorldProc{s: &Service{store: store}}

	got := world.GetHeader(common.Hash{}, 0)
	require.NotNil(t, got)
	want := store.GetBlock(0)
	require.Equal(t, big.NewInt(0), got.Number)
	require.Equal(t, want.Time, got.Time)
	require.Equal(t, want.GasLimit, got.GasLimit)
	require.Equal(t, want.Hash(), got.Hash)
	require.Equal(t, want.ParentHash, got.ParentHash)
}

func TestEmitterWorldRead_GetEpochStartBlock_ReturnsKnownEpochStarts(t *testing.T) {
	world := &emitterWorldRead{Store: initStoreForTests(t)}

	// some known values for epochs created during genesis
	require.Equal(t, idx.Block(0), world.GetEpochStartBlock(0))
	require.Equal(t, idx.Block(1), world.GetEpochStartBlock(1))
	require.Equal(t, idx.Block(2), world.GetEpochStartBlock(2))
}

func initStoreForTests(t *testing.T) *Store {
	t.Helper()
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	genStore := makefakegenesis.FakeGenesisStoreWithRulesAndStart(
		2,
		utils.ToFtm(genesisBalance),
		utils.ToFtm(genesisStake),
		opera.FakeNetRules(opera.SonicFeatures),
		2,
		2,
	)
	genesis := genStore.Genesis()
	require.NoError(store.ApplyGenesis(genesis))
	return store
}
