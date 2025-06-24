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

package check

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/0xsoniclabs/carmen/go/database/mpt"
	"github.com/0xsoniclabs/carmen/go/database/mpt/io"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/log"
)

func CheckArchiveStateDb(ctx context.Context, dataDir string, cacheRatio cachescale.Func) error {
	// compare with blocks in the gdb
	if err := checkArchiveBlockRoots(dataDir, cacheRatio); err != nil {
		return err
	}
	log.Info("The archive states hashes matches with blocks in the gdb")

	archiveDir := filepath.Join(dataDir, "carmen", "archive")
	info, err := io.CheckMptDirectoryAndGetInfo(archiveDir)
	if err != nil {
		return fmt.Errorf("failed to check archive state dir: %w", err)
	}
	if err := mpt.VerifyArchiveTrie(ctx, archiveDir, info.Config, verificationObserver{}); err != nil {
		return fmt.Errorf("archive state verification failed: %w", err)
	}
	log.Info("Verification of the archive state succeed")
	return nil
}

func checkArchiveBlockRoots(dataDir string, cacheRatio cachescale.Func) (err error) {
	gdb, dbs, err := createGdb(dataDir, cacheRatio, carmen.S5Archive, false)
	if err != nil {
		return fmt.Errorf("failed to create gdb and db producer: %w", err)
	}
	defer caution.CloseAndReportError(&err, gdb, "failed to close gossip db")
	defer caution.CloseAndReportError(&err, dbs, "failed to close db producer")

	invalidBlocks := 0
	lastBlockIdx := gdb.GetLatestBlockIndex()
	for i := idx.Block(1); i <= lastBlockIdx; i++ {
		block := gdb.GetBlock(i)
		if block == nil {
			return fmt.Errorf("verification failed - unable to get block %d from gdb", i)
		}
		err = gdb.EvmStore().CheckArchiveStateHash(i, hash.Hash(block.StateRoot))
		if err != nil {
			log.Error("Block root verification failed", "block", i, "err", err)
			invalidBlocks++
		}
		if i%1000 == 0 {
			log.Info("Block root verification OK", "block", i)
		}
	}
	if invalidBlocks != 0 {
		return fmt.Errorf("block root verification failed for %d blocks (from %d total blocks)", invalidBlocks, lastBlockIdx)
	}
	log.Info("Block root verification OK for all blocks", "blocks", lastBlockIdx)
	return nil
}
