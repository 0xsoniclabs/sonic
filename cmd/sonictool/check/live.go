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
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/log"
)

func CheckLiveStateDb(ctx context.Context, dataDir string, cacheRatio cachescale.Func) error {
	// compare with the last block in the gdb
	if err := checkLiveBlockRoot(dataDir, cacheRatio); err != nil {
		return err
	}
	log.Info("The live state hash matches with the last block in the gdb")

	liveDir := filepath.Join(dataDir, "carmen", "live")
	info, err := io.CheckMptDirectoryAndGetInfo(liveDir)
	if err != nil {
		return fmt.Errorf("failed to check live state dir: %w", err)
	}
	if err := mpt.VerifyFileLiveTrie(ctx, liveDir, info.Config, verificationObserver{}); err != nil {
		return fmt.Errorf("live state verification failed: %w", err)
	}
	log.Info("Verification of the live state succeeded")
	return nil
}

func checkLiveBlockRoot(dataDir string, cacheRatio cachescale.Func) (err error) {
	gdb, dbs, err := createGdb(dataDir, cacheRatio, carmen.NoArchive, true)
	if err != nil {
		return fmt.Errorf("failed to create gdb and db producer: %w", err)
	}
	defer caution.CloseAndReportError(&err, gdb, "failed to close gossip db")
	defer caution.CloseAndReportError(&err, dbs, "failed to close db producer")

	lastBlockIdx := gdb.GetLatestBlockIndex()
	lastBlock := gdb.GetBlock(lastBlockIdx)
	if lastBlock == nil {
		return fmt.Errorf("verification failed - unable to get the last block (%d) from gdb", lastBlockIdx)
	}
	err = gdb.EvmStore().CheckLiveStateHash(lastBlockIdx, hash.Hash(lastBlock.StateRoot))
	if err != nil {
		return fmt.Errorf("checking live state failed: %w", err)
	}
	log.Info("Live block root verification OK", "block", lastBlockIdx)
	return nil
}
