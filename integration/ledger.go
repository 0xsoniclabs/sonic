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
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
)

// Ledger wraps a gossip.Ledger together with the on-disk store it runs on:
// OpenLedger opens the databases and the Carmen state of a data directory, and
// Close commits and releases them again. As it embeds the gossip.Ledger, it can
// replay and process blocks.
type Ledger struct {
	*gossip.Ledger
	store *gossip.Store
	dbs   kvdb.FullDBProducer
}

// OpenLedger opens the on-disk store of dataDir with OpenStore and returns a
// Ledger over it. The returned Ledger owns the opened resources and must be
// closed with Close.
func OpenLedger(dataDir string, config gossip.LedgerConfig) (*Ledger, error) {
	store, dbs, err := OpenStore(dataDir, cachescale.Identity, nil)
	if err != nil {
		return nil, err
	}
	return &Ledger{
		Ledger: gossip.NewLedger(store, config),
		store:  store,
		dbs:    dbs,
	}, nil
}

// AdvanceHead moves the head pointer to block after its Commit by setting the last
// block (without an Atropos) and the finalized state root; the rest of the block
// and epoch state stays as it is. The block then is the parent the next block
// reads and the block Head returns.
func (l *Ledger) AdvanceHead(block *inter.Block) {
	bs := l.store.GetBlockState().Copy()
	bs.LastBlock = iblockproc.BlockCtx{Idx: idx.Block(block.Number), Time: block.Time}
	bs.FinalizedStateRoot = hash.Hash(block.StateRoot)
	l.store.SetBlockEpochState(bs, l.store.GetEpochState())
}

// Close commits the gossip store, so the blocks and the head written since
// opening persist, and releases the resources owned by the ledger: the gossip
// store and then the database producer. Errors from all three are reported.
func (l *Ledger) Close() (err error) {
	caution.ExecuteAndReportError(&err, l.store.Commit, "failed to commit gossip store")
	caution.CloseAndReportError(&err, l.store, "failed to close gossip store")
	caution.CloseAndReportError(&err, l.dbs, "failed to close database producer")
	return err
}
