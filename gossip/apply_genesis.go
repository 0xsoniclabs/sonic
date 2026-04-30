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
	"github.com/0xsoniclabs/sonic/scc/cert"
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
	} else {
		s.Log.Info("StateDB imported successfully, stateRoot matches", "index", lastBlock.Idx, "root", lastBlock.StateRoot)
	}

	g.CommitteeCertificates.ForEach(func(c cert.Certificate[cert.CommitteeStatement]) bool {
		if err := s.UpdateCommitteeCertificate(c); err != nil {
			s.Log.Crit("Failed to write committee certificate", "err", err)
			return false
		}
		return true
	})

	g.BlockCertificates.ForEach(func(c cert.Certificate[cert.BlockStatement]) bool {
		if err := s.UpdateBlockCertificate(c); err != nil {
			s.Log.Crit("Failed to write block certificate", "err", err)
			return false
		}
		return true
	})

	if g.ProcessedBundles != nil {

		// accumulate all execution infos based on the block where they were processed
		bundlesByBlock := make(map[uint64][]bundle.ExecutionInfo)
		g.ProcessedBundles.ForEach(func(info bundle.ExecutionInfo) bool {
			bundlesByBlock[info.BlockNumber] = append(bundlesByBlock[info.BlockNumber], info)
			return true
		})

		if len(bundlesByBlock) == 0 {
			s.Log.Info("No processed bundles in genesis, skipping import")
			return nil
		}
		s.Log.Info("Importing processed bundles from genesis", "count", len(bundlesByBlock))

		// get history hashes from genesis so that the predecessor can be used
		// as the starting point and both oldest and latest can be verified.
		hh, found := g.ProcessedBundles.GetHistoryHashes()
		if !found {
			s.Log.Crit("Bundles were processed but no history hash was found in genesis")
		}
		s.Log.Info("found bundle history hashes in genesis",
			"oldestBlockNum", hh.Oldest.BlockNumber, "oldestHash", hh.Oldest.Hash,
			"latestBlockNum", hh.Latest.BlockNumber, "latestHash", hh.Latest.Hash)

		// When the block range is less than 1024, there have been less than
		// 1024 blocks since the first block which contains processed bundles.
		// Therefore, we know the previous history hash is zero and we can
		// start from the oldest block. Otherwise, there have been at least
		// 1024 blocks since the first block with processed bundles, so we use
		// the oldest hash as the starting point and start replay from the next
		// block. Because the block range is inclusive, we check for 1023.
		startBlock := hh.Oldest.BlockNumber
		if hh.Latest.BlockNumber-hh.Oldest.BlockNumber >= 1023 {
			s.SetProcessedBundlesHistoryHash(hh.Oldest.BlockNumber, hh.Oldest.Hash)
			startBlock = hh.Oldest.BlockNumber + 1
		}

		// Replay all blocks to latest, adding bundles
		// where present and nil otherwise, so the hash chain is correctly
		// computed for every block.
		for block := startBlock; block <= hh.Latest.BlockNumber; block++ {
			bundlesPerBlock := map[common.Hash]bundle.PositionInBlock{}
			for _, info := range bundlesByBlock[block] {
				if _, exists := bundlesPerBlock[info.ExecutionPlanHash]; exists {
					s.Log.Crit("Duplicate execution plan hash in genesis",
						"block", block,
						"hash", info.ExecutionPlanHash)
					return fmt.Errorf(
						"duplicate execution plan hash in genesis: block %d, hash %s",
						block, info.ExecutionPlanHash)
				}
				bundlesPerBlock[info.ExecutionPlanHash] = info.Position
			}
			s.AddProcessedBundles(block, bundlesPerBlock)
		}

		// Verify that the cumulative history hash for the latest block matches
		// the expected hash from the genesis file, confirming that the imported
		// entries are correct and complete.
		latestHash, ok := s.GetProcessedBundleHistoryHash(hh.Latest.BlockNumber)
		if !ok || latestHash != hh.Latest.Hash {
			return fmt.Errorf(
				"reproduced latest bundle history hash does not match genesis: "+
					"got block %d hash %s, want block %d hash %s",
				hh.Latest.BlockNumber, latestHash, hh.Latest.BlockNumber, hh.Latest.Hash)
		}
		s.Log.Info("Processed bundles imported successfully, all history hashes verified",
			"latestBlockNum", hh.Latest.BlockNumber, "latestHash", latestHash)
	}

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
