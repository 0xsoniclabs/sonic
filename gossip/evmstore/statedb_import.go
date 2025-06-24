// Copyright 2025 Sonic Operations Ltd
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

package evmstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	mptio "github.com/0xsoniclabs/carmen/go/database/mpt/io"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/utils/adapters/kvdb2ethdb"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/kvdb/nokeyiserr"
	"github.com/Fantom-foundation/lachesis-base/kvdb/pebble"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

var emptyCodeHash = crypto.Keccak256(nil)

// ImportLiveWorldState imports Sonic World State data from the live state genesis section.
// Must be called before the first Open call.
func (s *Store) ImportLiveWorldState(liveReader io.Reader) error {
	liveDir := filepath.Join(s.parameters.Directory, "live")
	if err := os.MkdirAll(liveDir, 0700); err != nil {
		return fmt.Errorf("failed to create carmen dir during FWS import; %v", err)
	}
	if err := mptio.ImportLiveDb(mptio.NewLog(), liveDir, liveReader); err != nil {
		return fmt.Errorf("failed to import LiveDB; %v", err)
	}
	return nil
}

// ImportArchiveWorldState imports Sonic World State data from the archive state genesis section.
// Must be called before the first Open call.
func (s *Store) ImportArchiveWorldState(archiveReader io.Reader) error {
	if s.parameters.Archive == carmen.NoArchive {
		return nil // skip if the archive is disabled
	}
	if s.parameters.Archive == carmen.S5Archive {
		archiveDir := filepath.Join(s.parameters.Directory, "archive")
		if err := os.MkdirAll(archiveDir, 0700); err != nil {
			return fmt.Errorf("failed to create carmen archive dir during FWS import; %v", err)
		}
		if err := mptio.ImportArchive(mptio.NewLog(), archiveDir, archiveReader); err != nil {
			return fmt.Errorf("failed to initialize Archive; %v", err)
		}
		return nil
	}
	return fmt.Errorf("archive is used, but cannot be initialized from FWS live genesis section")
}

// InitializeArchiveWorldState imports Sonic World State data from the live state genesis section.
// Must be called before the first Open call.
func (s *Store) InitializeArchiveWorldState(liveReader io.Reader, blockNum uint64) error {
	if s.parameters.Archive == carmen.NoArchive {
		return nil // skip if the archive is disabled
	}
	if s.parameters.Archive == carmen.S5Archive {
		archiveDir := filepath.Join(s.parameters.Directory, "archive")
		if err := os.MkdirAll(archiveDir, 0700); err != nil {
			return fmt.Errorf("failed to create carmen archive dir during FWS import; %v", err)
		}
		if err := mptio.InitializeArchive(mptio.NewLog(), archiveDir, liveReader, blockNum); err != nil {
			return fmt.Errorf("failed to initialize Archive; %v", err)
		}
		return nil
	}
	return fmt.Errorf("archive is used, but cannot be initialized from FWS live genesis section")
}

// ExportLiveWorldState exports Sonic World State data for the live state genesis section.
// The Store must be closed during the call.
func (s *Store) ExportLiveWorldState(ctx context.Context, out io.Writer) error {
	liveDir := filepath.Join(s.parameters.Directory, "live")
	if err := mptio.Export(ctx, mptio.NewLog(), liveDir, out); err != nil {
		return fmt.Errorf("failed to export Live StateDB; %v", err)
	}
	return nil
}

// ExportArchiveWorldState exports Sonic World State data for the archive state genesis section.
// The Store must be closed during the call.
func (s *Store) ExportArchiveWorldState(ctx context.Context, out io.Writer) error {
	archiveDir := filepath.Join(s.parameters.Directory, "archive")
	if err := mptio.ExportArchive(ctx, mptio.NewLog(), archiveDir, out); err != nil {
		return fmt.Errorf("failed to export Archive StateDB; %v", err)
	}
	return nil
}

func (s *Store) ImportLegacyEvmData(evmItems genesis.EvmItems, blockNum uint64, root common.Hash) (err error) {
	if err = s.Open(); err != nil {
		return fmt.Errorf("failed to open EvmStore for legacy EVM data import; %w", err)
	}
	defer caution.CloseAndReportError(&err, s, "failed to close EvmStore after legacy EVM data import")

	carmenDir, err := os.MkdirTemp(s.parameters.Directory, "opera-tmp-import-legacy-genesis")
	if err != nil {
		panic(fmt.Errorf("failed to create temporary dir for legacy EVM data import: %w", err))
	}
	defer caution.ExecuteAndReportError(&err, func() error { return os.RemoveAll(carmenDir) },
		"failed to remove temporary directory for legacy EVM data import")

	s.Log.Info("Unpacking legacy EVM data into a temporary directory", "dir", carmenDir)
	db, err := pebble.New(carmenDir, 1024, 100, nil, nil)
	if err != nil {
		panic(fmt.Errorf("failed to open temporary database for legacy EVM data import: %w", err))
	}
	evmItems.ForEach(func(key, value []byte) bool {
		err := db.Put(key, value)
		return err == nil
	})

	s.Log.Info("Importing legacy EVM data into Carmen", "index", blockNum, "root", root)

	var currentBlock uint64 = 1
	var accountsCount, slotsCount uint64 = 0, 0
	bulk := s.liveStateDb.StartBulkLoad(currentBlock)

	restartBulkIfNeeded := func() error {
		if (accountsCount+slotsCount)%1_000_000 == 0 && currentBlock < blockNum {
			if err := bulk.Close(); err != nil {
				return err
			}
			currentBlock++
			bulk = s.liveStateDb.StartBulkLoad(currentBlock)
		}
		return nil
	}

	chaindb := rawdb.NewDatabase(kvdb2ethdb.Wrap(nokeyiserr.Wrap(db)))
	tdb := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: false, IsVerkle: false})
	t, err := trie.NewStateTrie(trie.StateTrieID(root), tdb)
	if err != nil {
		return fmt.Errorf("failed to open trie; %w", err)
	}
	preimages := table.New(db, []byte("secure-key-"))

	accIter, err := t.NodeIterator(nil)
	if err != nil {
		return fmt.Errorf("failed to open accounts iterator; %w", err)
	}
	for accIter.Next(true) {
		if accIter.Leaf() {

			addressBytes, err := preimages.Get(accIter.LeafKey())
			if err != nil || addressBytes == nil {
				return fmt.Errorf("missing preimage for account address hash %v; %w", accIter.LeafKey(), err)
			}
			address := cc.Address(common.BytesToAddress(addressBytes))

			var acc types.StateAccount
			if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
				return fmt.Errorf("invalid account encountered during traversal; %w", err)
			}

			bulk.CreateAccount(address)
			bulk.SetNonce(address, acc.Nonce)
			bulk.SetBalance(address, amount.NewFromUint256(acc.Balance))

			if !bytes.Equal(acc.CodeHash, emptyCodeHash) {
				code := rawdb.ReadCode(chaindb, common.BytesToHash(acc.CodeHash))
				if len(code) == 0 {
					return fmt.Errorf("missing code for account %v", address)
				}
				bulk.SetCode(address, code)
			}

			if acc.Root != types.EmptyRootHash {
				storageTrie, err := trie.NewStateTrie(trie.StateTrieID(acc.Root), tdb)
				if err != nil {
					return fmt.Errorf("failed to open storage trie for account %v; %w", address, err)
				}
				storageIt, err := storageTrie.NodeIterator(nil)
				if err != nil {
					return fmt.Errorf("failed to open storage iterator for account %v; %w", address, err)
				}
				for storageIt.Next(true) {
					if storageIt.Leaf() {
						keyBytes, err := preimages.Get(storageIt.LeafKey())
						if err != nil || keyBytes == nil {
							return fmt.Errorf("missing preimage for storage key hash %v; %w", storageIt.LeafKey(), err)
						}
						key := cc.Key(common.BytesToHash(keyBytes))

						_, valueBytes, _, err := rlp.Split(storageIt.LeafBlob())
						if err != nil {
							return fmt.Errorf("failed to decode storage; %w", err)
						}
						value := cc.Value(common.BytesToHash(valueBytes))

						bulk.SetState(address, key, value)
						slotsCount++
						if err := restartBulkIfNeeded(); err != nil {
							return err
						}
					}
				}
				if storageIt.Error() != nil {
					return fmt.Errorf("failed to iterate storage trie of account %v; %w", address, storageIt.Error())
				}
			}

			accountsCount++
			if err := restartBulkIfNeeded(); err != nil {
				return err
			}
		}
	}
	if accIter.Error() != nil {
		return fmt.Errorf("failed to iterate accounts trie; %w", accIter.Error())
	}

	if err := bulk.Close(); err != nil {
		return err
	}
	// add the empty genesis block into archive
	if currentBlock < blockNum {
		bulk = s.liveStateDb.StartBulkLoad(blockNum)
		if err := bulk.Close(); err != nil {
			return err
		}
	}
	return nil
}
