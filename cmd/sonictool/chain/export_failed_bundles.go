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

package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// failedBundleEntry is the JSON encoding of a failed bundle.
type failedBundleEntry struct {
	Block uint64        `json:"block"`
	Pos   int           `json:"pos"`
	Tx    hexutil.Bytes `json:"tx"`
}

// ExportFailedBundles writes the failed bundles found in the database to the
// given file as a JSON list of {"block", "pos", "tx"} entries, where tx is the
// binary encoding of the envelope. If appendToFile is set, the entries of the
// file are kept and the export starts after their last block.
func ExportFailedBundles(gdbParams db.GossipDbParameters, filename string, appendToFile bool) (err error) {
	entries := []failedBundleEntry{}
	start := idx.Block(0)
	if appendToFile {
		data, err := os.ReadFile(filename)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &entries); err != nil {
				return fmt.Errorf("failed to parse %v: %w", filename, err)
			}
		}
		if len(entries) > 0 {
			start = idx.Block(entries[len(entries)-1].Block + 1)
		}
	}

	chaindataDir := filepath.Join(gdbParams.DataDir, "chaindata")
	dbs, err := db.MakeDbProducer(chaindataDir, cachescale.Identity)
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, dbs, "failed to close db producer")

	gdbParams.Dbs = dbs
	gdbParams.CacheRatio = cachescale.Identity

	gdb, err := db.MakeGossipDb(gdbParams)
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, gdb, "failed to close gossip db")

	log.Info("Searching failed bundles", "start", start, "kept", len(entries))
	failed, err := gdb.FindFailedBundles(start)
	if err != nil {
		return err
	}
	for _, f := range failed {
		tx, err := f.Envelope.MarshalBinary()
		if err != nil {
			return err
		}
		entries = append(entries, failedBundleEntry{Block: uint64(f.Block), Pos: f.Pos, Tx: tx})
	}
	log.Info("Exporting failed bundles", "count", len(entries), "new", len(failed))
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	// The file is replaced only after a successful search.
	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}
