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
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/utils/dbutil/autocompact"
	"github.com/Fantom-foundation/lachesis-base/kvdb/batched"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// ApplyGenesis writes initial state.
func (s *Store) ApplyGenesis(g genesis.Genesis) (err error) {
	// use batching wrapper for hot tables
	unwrap := s.WrapTablesAsBatched()
	defer unwrap()

	// write epochs
	var topEr *ier.LlrIdxFullEpochRecord
	g.Epochs.ForEach(func(er ier.LlrIdxFullEpochRecord) bool {
		if er.EpochState.Rules.NetworkID != g.NetworkID || er.EpochState.Rules.Name != g.NetworkName {
			err = errors.New("network ID/name mismatch")
			return false
		}
		if topEr == nil {
			topEr = &er
		}
		s.WriteFullEpochRecord(er)
		return true
	})
	if err != nil {
		return err
	}
	if topEr == nil {
		return errors.New("no ERs in genesis")
	}
	var prevEs *iblockproc.EpochState
	s.ForEachHistoryBlockEpochState(func(bs iblockproc.BlockState, es iblockproc.EpochState) bool {
		s.WriteUpgradeHeight(bs, es, prevEs)
		prevEs = &es
		return true
	})
	s.SetBlockEpochState(topEr.BlockState, topEr.EpochState)
	s.FlushBlockEpochState()

	s.SetGenesisID(g.GenesisID)
	s.SetGenesisBlockIndex(topEr.BlockState.LastBlock.Idx)

	// write blocks
	var lastBlock ibr.LlrIdxFullBlockRecord
	g.Blocks.ForEach(func(br ibr.LlrIdxFullBlockRecord) bool {
		err = s.WriteFullBlockRecord(br)
		if err != nil {
			s.Log.Crit(err.Error())
			return false
		}

		if br.Idx > lastBlock.Idx {
			lastBlock = br
		}
		return true
	})

	// write EVM items
	liveReader, err := g.FwsLiveSection.GetReader()
	if err != nil {
		s.Log.Info("Sonic World State Live data not available in the genesis", "err", err)
	}

	if liveReader != nil { // has S5 section - import S5 data
		s.Log.Info("Importing Sonic World State Live data from genesis")
		err = s.evm.ImportLiveWorldState(liveReader)
		if err != nil {
			return fmt.Errorf("failed to import Sonic World State data from genesis; %v", err)
		}

		// import S5 archive
		archiveReader, _ := g.FwsArchiveSection.GetReader()
		if archiveReader != nil { // has archive section
			s.Log.Info("Importing Sonic World State Archive data from genesis")
			err = s.evm.ImportArchiveWorldState(archiveReader)
			if err != nil {
				return fmt.Errorf("failed to import Sonic World State Archive data from genesis; %v", err)
			}
		} else { // no archive section - initialize archive from the live section
			s.Log.Info("No archive in the genesis file - initializing the archive from the live state", "blockNum", lastBlock.Idx)
			liveToArchiveReader, err := g.FwsLiveSection.GetReader() // second reader of the same section for the archive import
			if err != nil {
				return fmt.Errorf("failed to get second FWS section reader; %v", err)
			}
			err = s.evm.InitializeArchiveWorldState(liveToArchiveReader, uint64(lastBlock.Idx))
			if err != nil {
				return fmt.Errorf("failed to import Sonic World State data from genesis; %v", err)
			}
		}
	} else { // no S5 section in the genesis file
		// Import legacy EVM genesis section
		err = s.evm.ImportLegacyEvmData(g.RawEvmItems, uint64(lastBlock.Idx), common.Hash(lastBlock.StateRoot))
		if err != nil {
			return fmt.Errorf("import of legacy genesis data into StateDB failed; %v", err)
		}
	}

	if err := s.evm.Open(); err != nil {
		return fmt.Errorf("unable to open EvmStore to check imported state: %w", err)
	}
	if err := s.evm.CheckLiveStateHash(lastBlock.Idx, lastBlock.StateRoot); err != nil {
		return fmt.Errorf("checking imported live state failed: %w", err)
	}

	s.Log.Info("StateDB imported successfully, stateRoot matches", "index", lastBlock.Idx, "root", lastBlock.StateRoot)

	if g.ProcessedBundles != nil {
		if err := s.importProcessedBundles(g.ProcessedBundles); err != nil {
			return err
		}
	}
	return nil
}

// importProcessedBundles restores the processed bundles and the bundle
// history hash from the genesis file.
func (s *Store) importProcessedBundles(bundles genesis.ProcessedBundles) error {
	bundlesByBlock := groupBundlesByBlock(bundles)
	historyHashes, hasHistory := bundles.GetHistoryHashes()

	if !hasHistory && len(bundlesByBlock) == 0 {
		s.Log.Info("No processed bundles or bundle history in genesis, skipping bundle history import")
		return nil
	}
	return s.replayProcessedBundles(historyHashes, hasHistory, bundlesByBlock)
}

// groupBundlesByBlock accumulates all execution info based on the block where they were processed.
func groupBundlesByBlock(bundles genesis.ProcessedBundles) map[uint64][]bundle.ExecutionInfo {
	bundlesByBlock := make(map[uint64][]bundle.ExecutionInfo)
	bundles.ForEach(func(info bundle.ExecutionInfo) bool {
		bundlesByBlock[info.BlockNumber] = append(bundlesByBlock[info.BlockNumber], info)
		return true
	})
	return bundlesByBlock
}

// replayProcessedBundles replays the retained processed bundles on top of the
// oldest history hash of the genesis. The history hash chain is recomputed for
// every later block and verified against the latest genesis record.
//
// The history hash is updated for every block even if no bundles are retained,
// so it is replayed even without bundles to produce the correct epoch state
// hash at the next epoch sealing.
func (s *Store) replayProcessedBundles(
	historyHashes bundle.BundleGenesisHistoryHashes,
	hasHistory bool,
	bundlesByBlock map[uint64][]bundle.ExecutionInfo,
) error {
	if !hasHistory {
		return errors.New("bundles were processed but no history hash was found in genesis")
	}
	s.Log.Info("Importing processed bundles from genesis", "count", len(bundlesByBlock))
	s.Log.Info("found bundle history hashes in genesis",
		"oldestBlockNum", historyHashes.Oldest.BlockNumber, "oldestHash", historyHashes.Oldest.Hash,
		"latestBlockNum", historyHashes.Latest.BlockNumber, "latestHash", historyHashes.Latest.Hash)

	base := historyHashes.Oldest
	if err := s.restoreBundleHistoryBase(base, bundlesByBlock); err != nil {
		return err
	}

	// replay blocks to latest and add retained bundles
	// the execution plan chain is updated block by block
	for block := base.BlockNumber + 1; block <= historyHashes.Latest.BlockNumber; block++ {
		bundlesPerBlock, err := positionsByExecutionPlan(bundlesByBlock[block])
		if err != nil {
			return fmt.Errorf("invalid processed bundles in genesis at block %d: %w", block, err)
		}
		if err := s.addProcessedBundles(block, bundlesPerBlock); err != nil {
			return fmt.Errorf("failed to replay processed bundles of block %d: %w", block, err)
		}
	}

	return s.verifyBundleHistoryHash(historyHashes.Latest)
}

// restoreBundleHistoryBase stores the given history hash as the base of the
// replay, together with the bundles processed up to the base block. The oldest
// genesis hash is the history hash after its block, so bundles up to that
// block are only retained for replay protection and do not affect the hash.
func (s *Store) restoreBundleHistoryBase(
	base bundle.HistoryHash,
	bundlesByBlock map[uint64][]bundle.ExecutionInfo,
) error {
	s.processedBundleMutex.Lock()
	defer s.processedBundleMutex.Unlock()

	batch := s.table.ProcessedBundles.NewBatch()
	for block, infos := range bundlesByBlock {
		if block > base.BlockNumber {
			continue
		}
		positions, err := positionsByExecutionPlan(infos)
		if err != nil {
			return fmt.Errorf("invalid processed bundles in genesis at block %d: %w", block, err)
		}
		if _, err := s.addNewBundles(block, positions, batch); err != nil {
			return fmt.Errorf("failed to restore processed bundles of block %d: %w", block, err)
		}
	}

	err := errors.Join(
		putLatestBundleHistoryHash(batch, base.BlockNumber, base.Hash),
		batch.Put(getBundleHistoryHashKey(base.BlockNumber), base.Hash.Bytes()),
	)
	if err != nil {
		return fmt.Errorf("failed to restore bundle history base: %w", err)
	}
	if err := batch.Write(); err != nil {
		return err
	}
	s.retainedBundlesCounted = false // < recount the restored bundles on next use
	return nil
}

// positionsByExecutionPlan indexes the positions of the bundles processed in a single block
// by their execution plan hash.
func positionsByExecutionPlan(infos []bundle.ExecutionInfo) (map[common.Hash]bundle.PositionInBlock, error) {
	positions := make(map[common.Hash]bundle.PositionInBlock, len(infos))
	for _, info := range infos {
		if _, exists := positions[info.ExecutionPlanHash]; exists {
			return nil, fmt.Errorf("duplicate execution plan hash %s", info.ExecutionPlanHash)
		}
		positions[info.ExecutionPlanHash] = info.Position
	}
	return positions, nil
}

// verifyBundleHistoryHash checks if the cumulative history hash for the latest block
// matches the expected hash from the genesis file.
func (s *Store) verifyBundleHistoryHash(latest bundle.HistoryHash) error {
	got, ok := s.GetProcessedBundleHistoryHash(latest.BlockNumber)
	if !ok || got != latest.Hash {
		return fmt.Errorf(
			"reproduced latest bundle history hash does not match genesis: "+
				"got block %d hash %s, want block %d hash %s",
			latest.BlockNumber, got, latest.BlockNumber, latest.Hash)
	}
	s.Log.Info("Processed bundles imported successfully, all history hashes verified",
		"latestBlockNum", latest.BlockNumber, "latestHash", got)
	return nil
}

func (s *Store) WrapTablesAsBatched() (unwrap func()) {
	origTables := s.table

	batchedBlocks := batched.Wrap(autocompact.Wrap2M(s.table.Blocks, opt.GiB, 16*opt.GiB, false, "blocks"))
	s.table.Blocks = batchedBlocks

	batchedBlockHashes := batched.Wrap(s.table.BlockHashes)
	s.table.BlockHashes = batchedBlockHashes

	unwrapEVM := s.evm.WrapTablesAsBatched()
	return func() {
		unwrapEVM()
		_ = batchedBlocks.Flush()
		_ = batchedBlockHashes.Flush()
		s.table = origTables
	}
}
