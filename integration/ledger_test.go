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

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestOpenLedger_GenesisInitializedDir_OpensReadsAndCloses(t *testing.T) {
	dataDir := initFakeGenesisDataDir(t)

	ledger, err := OpenLedger(dataDir, gossip.LedgerConfig{})
	require.NoError(t, err)

	// The ledger opened over the applied genesis exposes a consistent head: the
	// genesis head block is readable and a block can begin on it, which checks
	// its state root against the Carmen live state.
	head, err := ledger.Head()
	require.NoError(t, err)
	require.NotZero(t, head.Block().StateRoot)
	_, err = head.BeginBlock(gossip.BlockParams{Number: head.Block().Number + 1, Time: head.Block().Time + 1}, nil)
	require.NoError(t, err)

	require.NoError(t, ledger.Close())
}

func TestLedger_AdvanceHead_MakesCommittedBlockTheHead(t *testing.T) {
	ledger, err := OpenLedger(initFakeGenesisDataDir(t), gossip.LedgerConfig{})
	require.NoError(t, err)
	defer func() { require.NoError(t, ledger.Close()) }()
	rules := ledger.store.GetRules()

	head, err := ledger.Head()
	require.NoError(t, err)
	executing, err := head.BeginBlock(gossip.BlockParams{
		Number: head.Block().Number + 1,
		Time:   head.Block().Time + 1,
		Rules:  rules,
	}, nil)
	require.NoError(t, err)
	finalized := executing.Finalize()
	block := finalized.Candidate().Block
	committed := finalized.Commit()
	ledger.AdvanceHead(block)
	committed.Publish()

	head, err = ledger.Head()
	require.NoError(t, err)
	require.Equal(t, block.Hash(), head.Block().Hash())

	_, err = head.BeginBlock(gossip.BlockParams{
		Number: block.Number + 1,
		Time:   block.Time + 1,
		Rules:  rules,
	}, nil)
	require.NoError(t, err)
}

func TestLedger_Close_PersistsTheHeadForReopening(t *testing.T) {
	dataDir := initFakeGenesisDataDir(t)
	ledger, err := OpenLedger(dataDir, gossip.LedgerConfig{})
	require.NoError(t, err)
	rules := ledger.store.GetRules()

	head, err := ledger.Head()
	require.NoError(t, err)
	executing, err := head.BeginBlock(gossip.BlockParams{
		Number: head.Block().Number + 1,
		Time:   head.Block().Time + 1,
		Rules:  rules,
	}, nil)
	require.NoError(t, err)
	finalized := executing.Finalize()
	block := finalized.Candidate().Block
	committed := finalized.Commit()
	ledger.AdvanceHead(block)
	committed.Publish()
	require.NoError(t, ledger.Close())

	ledger, err = OpenLedger(dataDir, gossip.LedgerConfig{})
	require.NoError(t, err)
	defer func() { require.NoError(t, ledger.Close()) }()

	head, err = ledger.Head()
	require.NoError(t, err)
	require.Equal(t, block.Hash(), head.Block().Hash())
	_, err = head.BeginBlock(gossip.BlockParams{Number: block.Number + 1, Time: block.Time + 1, Rules: rules}, nil)
	require.NoError(t, err)
}

// initFakeGenesisDataDir creates a temporary data directory and initializes an
// on-disk store in it from a fake genesis, returning the data directory. It
// mirrors the production genesis-import flow (open the store WITHOUT opening the
// EVM store, ApplyGenesis, Commit), since the genesis import requires an empty
// carmen directory — which OpenLedger's EvmStore().Open() would populate.
func initFakeGenesisDataDir(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	chaindataDir := filepath.Join(dataDir, "chaindata")
	carmenDir := filepath.Join(dataDir, "carmen")
	require.NoError(t, os.MkdirAll(chaindataDir, 0700))
	require.NoError(t, os.MkdirAll(carmenDir, 0700))

	genStore, err := makefakegenesis.FakeGenesisStore(
		1,
		uint256.NewInt(1e18),
		uint256.NewInt(1e18),
		opera.GetSonicUpgrades(),
		dataDir,
	)
	require.NoError(t, err)

	dbs, err := GetDbProducer(chaindataDir, DBCacheConfig{Cache: 480 << 20, Fdlimit: 100})
	require.NoError(t, err)

	cfg := gossip.DefaultStoreConfig(cachescale.Identity)
	cfg.EVM.StateDb.Directory = carmenDir
	store, err := gossip.NewStore(dbs, cfg)
	require.NoError(t, err)

	require.NoError(t, store.ApplyGenesis(genStore.Genesis()))
	require.NoError(t, store.Commit())
	require.NoError(t, store.Close())
	require.NoError(t, dbs.Close())
	return dataDir
}
