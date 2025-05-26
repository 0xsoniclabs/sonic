package gossip

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
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

func initStoreForTests(t *testing.T) *Store {
	t.Helper()
	require := require.New(t)
	store, err := NewMemStore(t)
	require.NoError(err)

	genStore := makefakegenesis.FakeGenesisStoreWithRulesAndStart(
		2,
		utils.ToFtm(genesisBalance),
		utils.ToFtm(genesisStake),
		opera.FakeNetRules(opera.GetSonicUpgrades()),
		2,
		2,
	)
	genesis := genStore.Genesis()
	require.NoError(store.ApplyGenesis(genesis))
	return store
}
