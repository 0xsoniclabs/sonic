// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package gossip

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

//go:generate mockgen -source=emitter_world_test.go -destination=emitter_world_mock.go -package=gossip

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

	got := world.Header(common.Hash{}, 0)
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

func TestEmitterWorldProc_HasBundleRecentlyBeenProcessed_ReturnsResultFromStore(t *testing.T) {
	for _, hasBeenProcessed := range []bool{true, false} {
		t.Run(fmt.Sprintf("hasBeenProcessed=%t", hasBeenProcessed), func(t *testing.T) {
			store := initStoreForTests(t)
			world := &emitterWorldProc{s: &Service{store: store}}

			hash := common.Hash{1, 2, 3}
			if hasBeenProcessed {
				store.AddProcessedBundles(0,
					map[common.Hash]bundle.PositionInBlock{
						hash: {},
					},
				)
			}

			got := world.HasBundleRecentlyBeenProcessed(hash)
			require.Equal(t, hasBeenProcessed, got)
		})
	}
}

// _table is an interface needed to generate a mock for a kvdb.Store.
type _table interface {
	kvdb.Store
}

var _ _table // to avoid _table unused warning.
