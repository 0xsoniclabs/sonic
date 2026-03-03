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
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// This file implements the storage and management of processed bundle hashes in
// the Store. Processed bundles are tracked to prevent re-processing the same
// bundle multiple times. The store keeps track of recently processed bundle
// hashes, up to a maximum block range defined by bundle.MaxBlockRange.
//
// In the underlying table, the following keys are used:
//  - key: [] -> [uint64, hash]                        // last block and hash for which the processed bundles have been stored
//  - key: ['e']<execPlanHash> -> [block,position]     // for a processed bundle
//  - key: ['i']<blockNum, execPlanHash> -> []         // for a processed bundle at a specific block number, to handle cleanups
//
// The hash of the processed bundle's history is computed as follows:
//  - initially, the hash is zero
//  - for every update, the hash is updated as follows:
//      addedExecPlanHash = Xor(<hashes of newly added bundles>)
//      deletedExecPlanHash = Xor(<hashes of deleted bundles>)
//      newHash = Keccak256(oldHash || addedExecPlanHash || deletedExecPlanHash || blockNum)
//
// The hash can be used to verify that validators remain aligned on their bundle
// processing history.

// AddProcessedBundles adds the given bundle hashes as processed for the given
// block number. This should be called after every block, listing the bundles
// that got accepted in the block.
func (s *Store) AddProcessedBundles(blockNum uint64, executedBundles []bundle.ExecutionInfo) error {

	// TODO: add a mutex to avoid concurrent updates.

	// Steps to be conducted:
	//  - add new hashes to the store
	//  - remove old hashes from the store
	//  - update the state hash

	// Register and index new hashes.
	table := s.table.ProcessedBundles
	batch := table.NewBatch()
	addedHash := common.Hash{}
	for _, info := range executedBundles {
		hash := info.Hash

		data := make([]byte, 12)
		binary.BigEndian.PutUint64(data[:8], info.BlockNum)
		binary.BigEndian.PutUint32(data[8:], info.Position)

		err := errors.Join(
			batch.Put(getEntryKey(hash), data),
			batch.Put(getIndexKey(blockNum, hash), []byte{0}),
		)
		if err != nil {
			return fmt.Errorf("failed to add processed bundle hash: %v", err)
		}
		addedHash = xorHash(addedHash, hash)
	}

	// Delete out-dated hashes.
	deletedHash := common.Hash{}
	if blockNum > bundle.MaxBlockRange {
		oldBlockNum := blockNum - bundle.MaxBlockRange
		it := table.NewIterator([]byte{'i'}, nil)
		for it.Next() {
			key := it.Key()
			if len(key) != 1+8+32 {
				continue
			}
			num := binary.BigEndian.Uint64(key[1 : 1+8])
			if num > oldBlockNum {
				break
			}
			hash := common.BytesToHash(key[1+8:])
			err := errors.Join(
				batch.Delete(getIndexKey(num, hash)),
				batch.Delete(getEntryKey(hash)),
			)
			if err != nil {
				return fmt.Errorf("failed to delete old processed bundle hash: %v", err)
			}
			deletedHash = xorHash(deletedHash, hash)
		}
	}

	// Update the state hash.
	_, oldHash, err := s.GetProcessedBundleHistoryHash()
	if err != nil {
		return fmt.Errorf("failed to get current hash of processed bundles: %v", err)
	}

	update := make([]byte, 3*32+8)
	copy(update[:32], oldHash.Bytes())
	copy(update[32:64], addedHash.Bytes())
	copy(update[64:96], deletedHash.Bytes())
	binary.BigEndian.PutUint64(update[96:], blockNum)
	newHash := common.Hash(crypto.Keccak256(update))

	err = batch.Put(nil, append(
		binary.BigEndian.AppendUint64(nil, blockNum),
		newHash.Bytes()...,
	))
	if err != nil {
		return fmt.Errorf("failed to update hash of processed bundles: %v", err)
	}

	// Write all changes to the store.
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to create batch for processed bundles: %v", err)
	}
	return nil
}

// HasRecentlyBeenProcessed checks if the given bundle hash has been processed
// recently, i.e., if it is present in the store. Note that this does not check
// for bundles that have been processed too far in the past, as those are
// removed from the store after bundle.MaxBlockRange blocks.
func (s *Store) HasRecentlyBeenProcessed(execPlanHash common.Hash) (bool, error) {
	res, err := s.table.ProcessedBundles.Get(getEntryKey(execPlanHash))
	if err != nil {
		return false, fmt.Errorf("failed to check processed bundle: %v", err)
	}
	return res != nil, nil
}

// GetBundleExecutionInfo returns the execution info for a processed bundle hash, if
// it is present in the store.
func (s *Store) GetBundleExecutionInfo(execPlanHash common.Hash) (*bundle.ExecutionInfo, error) {
	res, err := s.table.ProcessedBundles.Get(getEntryKey(execPlanHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get execution info for bundle: %v", err)
	}
	if res == nil {
		return nil, nil
	}
	if len(res) != 12 {
		return nil, fmt.Errorf("invalid data length for execution info: %d", len(res))
	}
	blockNum := binary.BigEndian.Uint64(res[:8])
	position := binary.BigEndian.Uint32(res[8:])
	return &bundle.ExecutionInfo{
		Hash:     execPlanHash,
		BlockNum: blockNum,
		Position: position,
	}, nil
}

// GetProcessedBundleHistoryHash returns the current hash of the processed
// bundles history, along with the block number of the last update.
func (s *Store) GetProcessedBundleHistoryHash() (uint64, common.Hash, error) {
	state, err := s.table.ProcessedBundles.Get(nil)
	if err != nil {
		return 0, common.Hash{}, fmt.Errorf("failed to get hash of processed bundles: %v", err)
	}
	if state == nil {
		return 0, common.Hash{}, nil
	}
	if len(state) != 32+8 {
		return 0, common.Hash{}, fmt.Errorf("invalid state length for processed bundles: %d", len(state))
	}
	blockNum := binary.BigEndian.Uint64(state[:8])
	hash := common.BytesToHash(state[8:])
	return blockNum, hash, nil
}

// getEntryKey returns the key used to store the presence of a processed bundle
// hash.
func getEntryKey(hash common.Hash) []byte {
	return append([]byte{'e'}, hash.Bytes()...)
}

// getIndexKey returns the key used to index a processed bundle hash at a
// specific block number, to handle cleanups.
func getIndexKey(blockNum uint64, hash common.Hash) []byte {
	return append(append([]byte{'i'}, binary.BigEndian.AppendUint64(nil, blockNum)...), hash.Bytes()...)
}

// xorHash returns the XOR of two hashes.
func xorHash(a, b common.Hash) common.Hash {
	var res common.Hash
	for i := 0; i < len(res); i++ {
		res[i] = a[i] ^ b[i]
	}
	return res
}
