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

	switch {
	case len(bundlesByBlock) > 0 && hasHistory:
		return s.replayProcessedBundles(historyHashes, bundlesByBlock)
	case len(bundlesByBlock) > 0:
		s.Log.Crit("Bundles were processed but no history hash was found in genesis")
		return errors.New("bundles were processed but no history hash was found in genesis")
	case hasHistory:
		// the history hash is updated for every block even if no bundles are retained,
		// we need to restore it to produce correct epoch state hash at the epoch sealing
		return s.restoreBundleHistoryHash(historyHashes.Latest)
	default:
		s.Log.Info("No processed bundles in genesis, skipping import")
		return nil
	}
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

// restoreBundleHistoryHash sets the latest bundle history hash without replaying any processed bundles
// as the genesis does not contain any within the retained window.
func (s *Store) restoreBundleHistoryHash(latest bundle.HistoryHash) error {
	s.Log.Info("No processed bundles in genesis, restoring bundle history hash",
		"latestBlockNum", latest.BlockNumber, "latestHash", latest.Hash)
	s.SetProcessedBundlesHistoryHash(latest.BlockNumber, latest.Hash)

	// keep the per-block entry, so a genesis can be exported from this state
	err := s.table.ProcessedBundles.Put(
		getBundleHistoryHashKey(latest.BlockNumber),
		latest.Hash.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("failed to restore bundle history hash: %w", err)
	}
	return nil
}

// replayProcessedBundles replays the retained processed bundles,
// the history hash chain is computed and verified against the genesis record.
func (s *Store) replayProcessedBundles(
	historyHashes bundle.BundleGenesisHistoryHashes,
	bundlesByBlock map[uint64][]bundle.ExecutionInfo,
) error {
	s.Log.Info("Importing processed bundles from genesis", "count", len(bundlesByBlock))
	s.Log.Info("found bundle history hashes in genesis",
		"oldestBlockNum", historyHashes.Oldest.BlockNumber, "oldestHash", historyHashes.Oldest.Hash,
		"latestBlockNum", historyHashes.Latest.BlockNumber, "latestHash", historyHashes.Latest.Hash)

	// replay blocks to latest and add retained bundles
	// the execution plan chain is updated block by block
	startBlock := s.initBundleHistoryReplay(historyHashes)
	for block := startBlock; block <= historyHashes.Latest.BlockNumber; block++ {
		bundlesPerBlock, err := positionsByExecutionPlan(bundlesByBlock[block])
		if err != nil {
			s.Log.Crit("Invalid processed bundles in genesis", "block", block, "err", err)
			return fmt.Errorf("invalid processed bundles in genesis at block %d: %w", block, err)
		}
		s.AddProcessedBundles(block, bundlesPerBlock)
	}

	return s.verifyBundleHistoryHash(historyHashes.Latest)
}

// initBundleHistoryReplay returns the first block to replay.
// The oldest hash is used as the base for a range of 1024+ blocks (the retained window, inclusive);
// otherwise the replay starts from a zero hash. We know the previous history hash is zero
// because there have been less than 1024 blocks since the first block which contains processed bundles.
func (s *Store) initBundleHistoryReplay(historyHashes bundle.BundleGenesisHistoryHashes) uint64 {
	oldest, latest := historyHashes.Oldest, historyHashes.Latest
	if latest.BlockNumber-oldest.BlockNumber < 1023 {
		return oldest.BlockNumber
	}
	s.SetProcessedBundlesHistoryHash(oldest.BlockNumber, oldest.Hash)
	return oldest.BlockNumber + 1
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
