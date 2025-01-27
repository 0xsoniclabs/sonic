package chain

import (
	"github.com/0xsoniclabs/sonic/cmd/sonictool/db"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"io"
	"path/filepath"
	"time"

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

func ExportEvents(gdbParams db.GossipDbParameters, w io.Writer, from, to idx.Epoch) (err error) {
	chaindataDir := filepath.Join(gdbParams.DataDir, "chaindata")
	dbs, err := db.MakeDbProducer(chaindataDir, cachescale.Identity)
	if err != nil {
		return err
	}
	defer dbs.Close()

	// Fill the rest of the params
	gdbParams.Dbs = dbs
	gdbParams.CacheRatio = cachescale.Identity

	gdb, err := db.MakeGossipDb(gdbParams)
	if err != nil {
		return err
	}
	defer gdb.Close()

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
