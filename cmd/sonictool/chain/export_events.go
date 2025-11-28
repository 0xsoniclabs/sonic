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

package chain

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/keycard-go/hexutils"
)

var (
	eventsFileHeader  = hexutils.HexToBytes("7e995678")
	eventsFileVersion = hexutils.HexToBytes("00010001")
)

// statsReportLimit is the time limit during import and export after which we
// always print out progress. This avoids the user wondering what's going on.
const statsReportLimit = 8 * time.Second

func ExportEvents(gdbParams db.GossipDbParameters, w io.Writer, importAll bool, from, to idx.Epoch) (err error) {
	chaindataDir := filepath.Join(gdbParams.DataDir, "chaindata")
	dbs, err := db.MakeDbProducer(chaindataDir, cachescale.Identity)
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, dbs, "failed to close db producer")

	// Fill the rest of the params
	gdbParams.Dbs = dbs
	gdbParams.CacheRatio = cachescale.Identity

	gdb, err := db.MakeGossipDb(gdbParams)
	if err != nil {
		return err
	}
	defer caution.CloseAndReportError(&err, gdb, "failed to close gossip db")

	if !importAll {
		err = validateRange(gdb, from, to)
		if err != nil {
			return err
		}
	}

	// Write header and version
	_, err = w.Write(append(eventsFileHeader, eventsFileVersion...))
	if err != nil {
		return err
	}

	start, reported := time.Now(), time.Time{}

	var (
		counter int
		last    hash.Event
	)
	gdb.ForEachEventRLP(from.Bytes(), func(id hash.Event, event rlp.RawValue) bool {
		if to >= from && id.Epoch() > to {
			return false
		}
		counter++
		_, err = w.Write(event)
		if err != nil {
			return false
		}
		last = id
		if counter%100 == 1 && time.Since(reported) >= statsReportLimit {
			log.Info("Exporting events", "last", last.String(), "exported", counter, "elapsed", common.PrettyDuration(time.Since(start)))
			reported = time.Now()
		}
		return true
	})
	log.Info("Exported events", "last", last.String(), "exported", counter, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// validateRange checks that the first and last epoch of the requested
// range are older than the current epoch and have events in the database.
// If either condition does not hold, it returns an error.
// Note that if not epoch range was given from the commandline, the check is skipped.
func validateRange(gdb *gossip.Store, from, to idx.Epoch) error {

	if from > to {
		err := fmt.Errorf("invalid requested range, initial epoch %v older than last %v", from, to)
		log.Error(err.Error())
		return err
	}

	currentEpoch := gdb.GetEpoch()
	if to > currentEpoch || from > currentEpoch {
		err := fmt.Errorf("requested to export events from %d to %d, but last known event is %d", from, to, currentEpoch)
		log.Error(err.Error())
		return err
	}

	verifyHasEventFromEpoch := func(epoch idx.Epoch) error {
		event := gdb.FirstEventFromEpoch(epoch)
		if event == nil || event.Epoch() != epoch {
			err := fmt.Errorf("could not find events for epoch %d", epoch)
			log.Error(err.Error())
			return err
		}
		return nil
	}

	err := verifyHasEventFromEpoch(from)
	if err != nil {
		return err
	}

	err = verifyHasEventFromEpoch(to)
	if err != nil {
		return err
	}

	return nil
}
