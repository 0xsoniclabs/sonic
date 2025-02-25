package itemsfetcher

import (
	"time"

	"github.com/0xsoniclabs/consensus/utils/cachescale"
)

type Config struct {
	ForgetTimeout time.Duration // Time before an announced event is forgotten
	ArriveTimeout time.Duration // Time allowance before an announced event is explicitly requested
	GatherSlack   time.Duration // Interval used to collate almost-expired announces with fetches
	HashLimit     int           // Maximum number of unique events a peer may have announced

	MaxBatch int // Maximum number of hashes in an announce batch (batch is divided if exceeded)

	MaxParallelRequests int // Maximum number of parallel requests

	// MaxQueuedHashesBatches is the maximum number of announce batches to queue up before
	// dropping incoming hashes.
	MaxQueuedBatches int
}

func DefaultConfig(scale cachescale.Func) Config {
	return Config{
		ForgetTimeout:       1 * time.Minute,
		ArriveTimeout:       1000 * time.Millisecond,
		GatherSlack:         100 * time.Millisecond,
		HashLimit:           20000,
		MaxBatch:            scale.I(512),
		MaxQueuedBatches:    scale.I(32),
		MaxParallelRequests: 256,
	}
}
