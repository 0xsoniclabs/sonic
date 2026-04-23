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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// This file implements the storage and management of processed bundles in
// the Store. Processed bundles are tracked to prevent re-processing the same
// bundle multiple times. The store keeps track of recently processed bundles
// by indexing the hashes of their execution plans, along with the block number
// and position in which they were executed. The store also maintains a hash of
// the history of processed bundles, which is updated after every block, to
// cross-validate that validators remain aligned on their bundle processing
// history.
//
// Bundles need to be indexed by the execution plan hash instead of the hash of
// the bundled transactions enclosing them, since otherwise the same plan may be
// resubmitted multiple times using different envelop transactions.
//
// In the underlying table, the following keys are used:
//  - key: [] 							 -> [uint64, hash]          // last block and hash for which the processed bundles have been stored
//  - key: ['e']<execPlanHash> 			 -> [block,position,count]  // for a recently processed bundle (auto-pruned)
//  - key: ['h']<blockNum> 				 -> hash                    // cumulative history hash at that block (auto-pruned)
//  - key: ['i']<blockNum, execPlanHash> -> []         				// for a processed bundle at a specific block number, to handle cleanups
//
// The hash of the processed bundle's history is computed as follows:
//  - initially, the hash is zero
//  - the zero hash is kept until the first executed bundle
//  - thereafter, for every block, the hash is updated as follows:
//      addedExecPlanHash = Xor(<hashes of newly added execution plans>)
//      newHash = Keccak256(oldHash || addedExecPlanHash || blockNum)
//
// The hash can be used to verify that validators remain aligned on their bundle
// processing history.

// AddProcessedBundles adds the given bundle execution information for the given
// block number. This should be called after every block, listing the bundles
// that got accepted in the block.
func (s *Store) AddProcessedBundles(
	blockNum uint64,
	executedBundles map[common.Hash]bundle.PositionInBlock,
) {
	// Make sure there is only one update at any time.
	s.processedBundleMutex.Lock()
	defer s.processedBundleMutex.Unlock()

	_, oldHash := s.GetProcessedBundleHistoryHash()

	// keep the zero hash until a bundle is executed and from then onwards,
	// compute it for every block.
	if oldHash == (common.Hash{}) && len(executedBundles) == 0 {
		return
	}

	// Register and index new hashes.
	table := s.table.ProcessedBundles
	batch := table.NewBatch()
	addedHash := s.addNewBundles(blockNum, executedBundles, batch)

	// Delete outdated hashes.
	s.deleteOutdatedBundles(blockNum, batch)

	// Update the history hash of processed bundles.
	newHash := computeNewBundleStateHash(oldHash, addedHash, blockNum)

	err := errors.Join(
		batch.Put(nil, append(
			bigendian.Uint64ToBytes(blockNum),
			newHash.Bytes()...,
		)),
		batch.Put(getBlockHistoryHashKey(blockNum), newHash.Bytes()),
	)
	if err != nil {
		s.Log.Crit("failed to update hash of processed bundles", "error", err)
	}

	// Write all changes to the store.
	if err := batch.Write(); err != nil {
		s.Log.Crit("failed to write batch for updating processed bundles", "error", err)
	}
}

// addNewBundles adds the given bundle execution information to the store, and returns
// the XOR of the hashes of the added bundles to update the history hash.
func (s *Store) addNewBundles(
	blockNum uint64,
	executedBundles map[common.Hash]bundle.PositionInBlock,
	batch kvdb.Batch,
) common.Hash {

	addedHash := common.Hash{}
	for hash, info := range executedBundles {
		data := make([]byte, 16)
		binary.BigEndian.PutUint64(data[:8], blockNum)
		binary.BigEndian.PutUint32(data[8:12], info.Offset)
		binary.BigEndian.PutUint32(data[12:], info.Count)

		err := errors.Join(
			batch.Put(getEntryKey(hash), data),
			batch.Put(getIndexKey(blockNum, hash), []byte{0}),
		)
		if err != nil {
			s.Log.Crit("failed to add processed bundle hash to batch", "error", err)
		}
		addedHash = xorHash(addedHash, hash)
	}
	return addedHash
}

// deleteOutdatedBundles deletes the entries of processed bundles that were
// processed too far in the past.
func (s *Store) deleteOutdatedBundles(blockNum uint64, batch kvdb.Batch) {
	if blockNum >= bundle.MaxBlockRange-1 {
		// enough blocks have passed to start cleaning up the store
		highestOutdatedBlockNumber := blockNum - bundle.MaxBlockRange + 1
		it := s.table.ProcessedBundles.NewIterator([]byte{'i'}, nil)
		defer it.Release()
		for it.Next() {
			key := it.Key()
			// size of the key used to index processed bundle is:
			//  - 1 byte for the prefix
			//  - 8 bytes for the block number
			//  - 32 bytes for the hash
			if len(key) != 1+8+32 {
				continue
			}
			oldBundleBlockNumber := binary.BigEndian.Uint64(key[1 : 1+8])
			if oldBundleBlockNumber > highestOutdatedBlockNumber {
				// the bundle is not old enough to be deleted
				break
			}
			hash := common.BytesToHash(key[1+8:])
			err := errors.Join(
				batch.Delete(getIndexKey(oldBundleBlockNumber, hash)),
				batch.Delete(getEntryKey(hash)),
				batch.Delete(getBlockHistoryHashKey(oldBundleBlockNumber)),
			)
			if err != nil {
				s.Log.Crit("failed to delete old processed bundle hash", "error", err)
			}
		}
	}
}

// computeNewBundleStateHash computes the new hash of the processed bundles history
// based on the previous hash, the added plans hash and the block number of the update.
//
// This hash is used to verify that clients remain aligned on their bundle
// processing history
func computeNewBundleStateHash(
	oldHash common.Hash,
	addedPlans common.Hash,
	blockNumber uint64,
) common.Hash {
	// size of the update buffer is:
	//  - 32 bytes for the previous hash
	//  - 32 bytes for the added hashes
	//  - 8 bytes for the block number
	update := make([]byte, 2*32+8)
	copy(update[:32], oldHash.Bytes())
	copy(update[32:64], addedPlans.Bytes())
	binary.BigEndian.PutUint64(update[64:], blockNumber)
	newHash := common.Hash(crypto.Keccak256(update))

	return newHash
}

// HasBundleRecentlyBeenProcessed checks if a bundle execution plan with the
// given hash has been processed recently. This is used to prevent re-processing
// the same bundle multiple times.
//
// Note: the store only keeps track of the bundles being executed in the last
// bundle.MaxBlockRange blocks, so this function returns false for bundles
// that were processed too far in the past and have been cleaned up from the
// store.
func (s *Store) HasBundleRecentlyBeenProcessed(execPlanHash common.Hash) bool {
	res, err := s.table.ProcessedBundles.Get(getEntryKey(execPlanHash))
	if err != nil {
		s.Log.Crit("failed to check processed bundle", "error", err)
	}
	return res != nil
}

// GetBundleExecutionInfo returns the execution info for a processed execution
// plan, if it is present in the store. Note that execution info is being
// automatically removed from the store after bundle.MaxBlockRange blocks,
// so this function returns nil for bundles that were processed too far in the
// past.
func (s *Store) GetBundleExecutionInfo(execPlanHash common.Hash) *bundle.ExecutionInfo {
	res, err := s.table.ProcessedBundles.Get(getEntryKey(execPlanHash))
	if err != nil {
		s.Log.Crit("failed to get execution info for bundle", "error", err)
	}
	if res == nil {
		return nil
	}
	if len(res) != 16 {
		s.Log.Crit("invalid data length for execution info", "length", len(res))
	}
	blockNum := binary.BigEndian.Uint64(res[:8])
	startPosition := binary.BigEndian.Uint32(res[8:12])
	count := binary.BigEndian.Uint32(res[12:])
	return &bundle.ExecutionInfo{
		ExecutionPlanHash: execPlanHash,
		BlockNumber:       blockNum,
		Position: bundle.PositionInBlock{
			Offset: startPosition,
			Count:  count,
		},
	}
}

// GetProcessedBundleHistoryHash returns the block number of the last update
// and the current hash of the processed bundles history.
func (s *Store) GetProcessedBundleHistoryHash() (uint64, common.Hash) {
	state, err := s.table.ProcessedBundles.Get(nil)
	if err != nil {
		s.Log.Crit("failed to get hash of processed bundles", "error", err)
	}
	if state == nil {
		return 0, common.Hash{}
	}

	// size of the value used stored in the processed bundles history:
	//  - 8 bytes for the block number
	//  - 32 bytes for the hash
	if len(state) != 8+32 {
		s.Log.Crit("invalid state length for processed bundles", "length", len(state))
	}
	blockNum := binary.BigEndian.Uint64(state[:8])
	hash := common.BytesToHash(state[8:])
	return blockNum, hash
}

// GetOldestRetainedBundleHistoryHash returns the block number and bundle
// history hash for the oldest block still retained in the store.
// Returns ok=false when no per-block hashes have been stored yet (i.e. no
// bundle has ever been executed).
func (s *Store) GetOldestRetainedBundleHistoryHash() (blockNum uint64, hash common.Hash, ok bool) {
	it := s.table.ProcessedBundles.NewIterator([]byte{'h'}, nil)
	defer it.Release()
	if !it.Next() {
		return 0, common.Hash{}, false
	}
	key := it.Key()
	// key layout: 1 byte prefix + 8 bytes block number
	if len(key) != 1+8 {
		s.Log.Crit("invalid per-block history hash key length", "length", len(key))
	}
	blockNum = binary.BigEndian.Uint64(key[1:])
	hash = common.BytesToHash(it.Value())
	return blockNum, hash, true
}

// SetProcessedBundlesHistoryHash sets the block number and hash of the
// processed bundles history. This should be used only during genesis initialization.
func (s *Store) SetProcessedBundlesHistoryHash(blockNum uint64, hash common.Hash) {
	// Make sure there is only one update at any time.
	s.processedBundleMutex.Lock()
	defer s.processedBundleMutex.Unlock()

	err := s.table.ProcessedBundles.Put(nil, append(
		bigendian.Uint64ToBytes(blockNum),
		hash.Bytes()...,
	))
	if err != nil {
		s.Log.Crit("failed to set hash of processed bundles", "error", err)
	}
}

// EnumerateProcessedBundles returns a list of all recently processed bundle
// execution infos currently tracked by the store.
func (s *Store) EnumerateProcessedBundles() []bundle.ExecutionInfo {
	// Make sure there is only one update at any time.
	s.processedBundleMutex.Lock()
	defer s.processedBundleMutex.Unlock()

	result := make([]bundle.ExecutionInfo, 0)

	// get all recently processed bundles
	it := s.table.ProcessedBundles.NewIterator([]byte{'e'}, nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		value := it.Value()
		if len(key) != 1+32 || len(value) != 16 {
			s.Log.Crit(
				"invalid key or value length for processed bundle entry during export",
				"keyLength", len(key),
				"valueLength", len(value))
		}

		result = append(result, bundle.ExecutionInfo{
			ExecutionPlanHash: common.BytesToHash(key[1:]),
			BlockNumber:       binary.BigEndian.Uint64(value[:8]),
			Position: bundle.PositionInBlock{
				Offset: binary.BigEndian.Uint32(value[8:12]),
				Count:  binary.BigEndian.Uint32(value[12:]),
			},
		})
	}
	if it.Error() != nil {
		s.Log.Crit("failed to export processed bundles", "error", it.Error())
	}
	return result
}

// --- utility functions for processed bundles management ---

// getEntryKey returns the key used to store the presence of a processed bundle
// hash and its associated execution infos.
func getEntryKey(hash common.Hash) []byte {
	return append([]byte{'e'}, hash.Bytes()...)
}

// getIndexKey returns the key used to index a processed bundle hash at a
// specific block number, to handle cleanups.
func getIndexKey(blockNum uint64, hash common.Hash) []byte {
	return append(
		append([]byte{'i'}, bigendian.Uint64ToBytes(blockNum)...),
		hash.Bytes()...)
}

// getBlockHistoryHashKey returns the key used to store the cumulative history hash
// for a specific block number.
func getBlockHistoryHashKey(blockNum uint64) []byte {
	return append([]byte{'h'}, bigendian.Uint64ToBytes(blockNum)...)
}

// xorHash returns the XOR of two hashes.
func xorHash(a, b common.Hash) common.Hash {
	var res common.Hash
	for i := 0; i < len(res); i++ {
		res[i] = a[i] ^ b[i]
	}
	return res
}
