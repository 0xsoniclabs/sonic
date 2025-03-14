package basepeerleecher

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errTerminated = errors.New("terminated")
)

type IsProcessed func(id interface{}) bool

type receivedChunk struct {
	id interface{}
}

type EpochDownloaderCallbacks struct {
	IsProcessed IsProcessed

	RequestChunks func(maxNum uint32, maxSize uint64, maxChunks uint32) error

	Suspend func() bool

	Done func() bool
}

// BasePeerLeecher is responsible for scheduling items for retrieval.
type BasePeerLeecher struct {
	cfg EpochDownloaderConfig

	totalRequested int
	totalProcessed int

	processingChunks []receivedChunk

	notifyReceivedChunk chan *receivedChunk

	quit chan struct{}
	done uint32

	wg *sync.WaitGroup

	// Callbacks
	callback EpochDownloaderCallbacks
}

// New creates an items fetcher to retrieve items chunk-by-chunk. Works only with 1 peer.
func New(wg *sync.WaitGroup, cfg EpochDownloaderConfig, callback EpochDownloaderCallbacks) *BasePeerLeecher {
	quit := make(chan struct{})
	return &BasePeerLeecher{
		processingChunks:    make([]receivedChunk, 0, cfg.ParallelChunksDownload*2),
		notifyReceivedChunk: make(chan *receivedChunk, cfg.ParallelChunksDownload*2),
		quit:                quit,
		cfg:                 cfg,
		callback:            callback,
		wg:                  wg,
	}
}

// Start boots up the announcement based synchroniser, accepting and processing
// fetches until termination requested.
func (d *BasePeerLeecher) Start() {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.loop()
	}()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (d *BasePeerLeecher) Stop() {
	d.Terminate()
	d.wg.Wait()
}

func (d *BasePeerLeecher) Terminate() {
	// set done, close the chan if it was not done yet
	if atomic.SwapUint32(&d.done, 1) == 0 {
		close(d.quit)
	}
}

func (d *BasePeerLeecher) Stopped() bool {
	return atomic.LoadUint32(&d.done) != 0
}

// NotifyChunkReceived injects new pack infos from a peer
func (d *BasePeerLeecher) NotifyChunkReceived(id interface{}) error {
	op := &receivedChunk{
		id: id,
	}
	select {
	case d.notifyReceivedChunk <- op:
		return nil
	case <-d.quit:
		return errTerminated
	}
}

// Loop is the main leecher's loop, fetching chunks according to the progress of their arrival
func (d *BasePeerLeecher) loop() {
	syncTicker := time.NewTicker(d.cfg.RecheckInterval)
	defer syncTicker.Stop()

	for {
		select {
		case <-d.quit:
			// terminating, abort all operations
			return

		case op := <-d.notifyReceivedChunk:
			if d.Stopped() {
				continue
			}
			if len(d.processingChunks) < d.cfg.ParallelChunksDownload*2 {
				d.processingChunks = append(d.processingChunks, *op)
				d.routine()
			}

		case <-syncTicker.C:
			d.routine()
		}
	}
}

func (d *BasePeerLeecher) routine() {
	if d.callback.Done() {
		d.Terminate()
		return
	}
	d.processingChunks = d.sweepProcessedChunks()
	d.tryToSync()
}

func (d *BasePeerLeecher) sweepProcessedChunks() []receivedChunk {
	notProcessed := make([]receivedChunk, 0, len(d.processingChunks))
	for _, op := range d.processingChunks {
		if d.callback.IsProcessed(op.id) {
			d.totalProcessed++
		} else {
			notProcessed = append(notProcessed, op)
		}
	}
	return notProcessed
}

func (d *BasePeerLeecher) tryToSync() {
	if d.callback.Suspend() {
		return
	}

	if d.totalRequested < d.totalProcessed+d.cfg.ParallelChunksDownload {
		requestsToSend := (d.totalProcessed + d.cfg.ParallelChunksDownload) - d.totalRequested
		d.totalRequested += requestsToSend
		_ = d.callback.RequestChunks(d.cfg.DefaultChunkItemsNum, d.cfg.DefaultChunkItemsSize, uint32(requestsToSend))
	}
}
