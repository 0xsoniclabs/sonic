package proclogger

import (
	"time"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/utils"
)

type dagSum struct {
	connected       idx.Event
	totalProcessing time.Duration
}

type llrSum struct {
	bvs idx.Block
	brs idx.Block
	evs idx.Epoch
	ers idx.Epoch
}

type Logger struct {
	// summary accumulators
	dagSum dagSum
	llrSum llrSum

	// latest logged data
	lastEpoch     idx.Epoch
	lastBlock     idx.Block
	lastID        hash.Event
	lastEventTime inter.Timestamp
	lastLlrTime   inter.Timestamp

	nextLogging time.Time

	emitting  bool
	noSummary bool

	logger.Instance
}

func (l *Logger) summary(now time.Time) {
	if l.noSummary {
		return
	}
	if now.After(l.nextLogging) {
		if l.llrSum != (llrSum{}) {
			age := utils.PrettyDuration(now.Sub(l.lastLlrTime.Time())).String()
			if l.lastLlrTime <= l.lastEventTime {
				age = "none"
			}
			l.Log.Info("New LLR summary", "last_epoch", l.lastEpoch, "last_block", l.lastBlock,
				"new_evs", l.llrSum.evs, "new_ers", l.llrSum.ers, "new_bvs", l.llrSum.bvs, "new_brs", l.llrSum.brs, "age", age)
		}
		if l.dagSum != (dagSum{}) {
			l.Log.Info("New DAG summary", "new", l.dagSum.connected, "last_id", l.lastID.String(),
				"age", utils.PrettyDuration(now.Sub(l.lastEventTime.Time())), "t", utils.PrettyDuration(l.dagSum.totalProcessing))
		}
		l.dagSum = dagSum{}
		l.llrSum = llrSum{}
		l.nextLogging = now.Add(8 * time.Second)
	}
}
