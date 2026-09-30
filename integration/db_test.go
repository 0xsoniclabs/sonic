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

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/stretchr/testify/require"
)

func TestOpenStore_MissingSubdirectory_ReturnsError(t *testing.T) {
	tests := map[string]struct {
		existing string
		wantErr  string
	}{
		"missing chaindata": {existing: "carmen", wantErr: "does not contain a chaindata directory"},
		"missing carmen":    {existing: "chaindata", wantErr: "does not contain a carmen directory"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dataDir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dataDir, test.existing), 0700))
			_, _, err := OpenStore(dataDir, cachescale.Identity, nil)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestOpenStore_GenesisInitializedDir_OpensStoreAndEvmStore(t *testing.T) {
	require := require.New(t)
	dataDir := initFakeGenesisDataDir(t)

	store, dbs, err := OpenStore(dataDir, cachescale.Identity, nil)
	require.NoError(err)

	headIdx := store.GetLatestBlockIndex()
	head := store.GetBlock(headIdx)
	require.NotNil(head)
	require.NoError(store.EvmStore().CheckLiveStateHash(headIdx, hash.Hash(head.StateRoot)))

	require.NoError(store.Close())
	require.NoError(dbs.Close())
}

func TestOpenStore_EvmStoreFailure_ReportsErrorAndReleasesTheDatabases(t *testing.T) {
	require := require.New(t)
	dataDir := initFakeGenesisDataDir(t)

	// The genesis data directory has an archive, so the EVM store refuses to open
	// it with the archive disabled.
	_, _, err := OpenStore(dataDir, cachescale.Identity, func(cfg *gossip.StoreConfig) {
		cfg.EVM.StateDb.Archive = carmen.NoArchive
	})
	require.ErrorContains(err, "disabled archive")

	store, dbs, err := OpenStore(dataDir, cachescale.Identity, nil)
	require.NoError(err)
	require.NoError(store.Close())
	require.NoError(dbs.Close())
}
