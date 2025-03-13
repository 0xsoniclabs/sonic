package basepeerleecher

import (
	"time"
)

type EpochDownloaderConfig struct {
	RecheckInterval        time.Duration
	DefaultChunkItemsNum   uint32
	DefaultChunkItemsSize  uint64
	ParallelChunksDownload int
}
