package gossip

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
)

func TestStore_GetLatestBlock_ReportsLatestBlock(t *testing.T) {
	require := require.New(t)
	store := initStoreForTests(t)

	require.Equal(idx.Block(2), store.GetLatestBlockIndex())
	got := store.GetLatestBlock()
	want := store.GetBlock(idx.Block(2))
	require.Equal(uint64(2), got.Number)
	require.Equal(want, got)
}
