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

package emitter

import (
	"io"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/inter"
)

const (
	// maxBlockHashesPerEvent is the maximum number of block hashes
	// that can be included in a single event.
	maxBlockHashesPerEvent = 64
)

// addBlockHashes collects block hashes from recently processed blocks and
// adds them to the event. It follows the emission pattern from go-opera's
// addLlrBlockVotes but without voting and misbehaviour proof semantics.
//
// The function tracks which blocks have already been included in previous
// events using a persistent file (emittedBvsFile), and collects up to
// maxBlockHashesPerEvent consecutive block hashes within the current epoch.
func (em *Emitter) addBlockHashes(e *inter.MutableEventPayload) {
	// Determine the starting block
	prevEmittedBlock := em.readLastBlockHashesTip()
	start := prevEmittedBlock + 1

	latestBlock := em.world.GetLatestBlockIndex()
	if start > latestBlock {
		return // nothing new to report
	}

	// Collect block hashes for consecutive blocks within the same epoch
	hashes := make([]hash.Hash, 0, maxBlockHashesPerEvent)
	var epoch idx.Epoch
	for b := start; b <= latestBlock && len(hashes) < maxBlockHashesPerEvent; b++ {
		block := em.world.GetBlock(b)
		if block == nil {
			break
		}
		blockEpoch := block.Epoch
		if epoch == 0 {
			epoch = blockEpoch
		}
		if blockEpoch != epoch {
			break // stop at epoch boundaries
		}
		blockHash := block.Hash()
		hashes = append(hashes, hash.Hash(blockHash))
	}
	if len(hashes) == 0 {
		return
	}

	e.SetBlockHashes(inter.BlockHashes{
		Start:  start,
		Epoch:  epoch,
		Hashes: hashes,
	})

	// Persist the last emitted block hash tip
	em.writeLastBlockHashesTip(start + idx.Block(len(hashes)) - 1)
}

// readLastBlockHashesTip reads the last block number for which block hashes
// have been emitted from the persistent file (reuses emittedBvsFile).
func (em *Emitter) readLastBlockHashesTip() idx.Block {
	if em.emittedBvsFile == nil {
		return 0
	}
	buf := make([]byte, 8)
	_, err := em.emittedBvsFile.ReadAt(buf, 0)
	if err != nil {
		if err == io.EOF {
			return 0
		}
		log.Crit("Failed to read block hashes file",
			"file", em.config.PrevBlockVotesFile.Path, "err", err)
	}
	return idx.BytesToBlock(buf)
}

// writeLastBlockHashesTip writes the last block number for which block
// hashes have been emitted to the persistent file (reuses emittedBvsFile).
func (em *Emitter) writeLastBlockHashesTip(block idx.Block) {
	if em.emittedBvsFile == nil {
		return
	}
	_, err := em.emittedBvsFile.WriteAt(block.Bytes(), 0)
	if err != nil {
		log.Crit("Failed to write block hashes file",
			"file", em.config.PrevBlockVotesFile.Path, "err", err)
	}
}
