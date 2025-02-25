package itemsfetcher

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/0xsoniclabs/consensus/utils/wlru"
	"github.com/0xsoniclabs/consensus/utils/workers"
)

/*
 * Fetcher is a network agent, which handles basic hash-based items sync.
 * The core mechanic is very simple: interested hash arrived => request it.
 * Fetcher has additional code to protect itself (and other nodes) against DoS.
 */

var (
	errTerminated = errors.New("terminated")
)

// ItemsRequesterFn is a callback type for sending a item retrieval request.
type ItemsRequesterFn func([]interface{}) error

type announceData struct {
	time       time.Time // Timestamp of the announcement
	peer       string    // Identifier of the peer originating the notification
	fetchItems ItemsRequesterFn
}

type announcesBatch struct {
	announceData
	ids []interface{} // Hashes of the items being announced
}

type fetchingItem struct {
	announce     announceData
	fetchingTime time.Time
}

// Fetcher is responsible for accumulating item announcements from various peers
// and scheduling them for retrieval.
type Fetcher struct {
	cfg Config

	// Various item channels
	notifications chan announcesBatch
	receivedItems chan []interface{}
	quit          chan struct{}

	// Callbacks
	callback Callback

	// Announce states
	announces *wlru.Cache // Announced items, scheduled for fetching

	fetching map[interface{}]fetchingItem // Announced items, currently fetching
	wg       sync.WaitGroup

	parallelTasks *workers.Workers
}

type Callback struct {
	// FilterInterested returns only item which may be requested.
	OnlyInterested func(ids []interface{}) []interface{}
	Suspend        func() bool
}

// New creates a item fetcher to retrieve items based on hash announcements.
func New(cfg Config, callback Callback) *Fetcher {
	f := &Fetcher{
		cfg:           cfg,
		notifications: make(chan announcesBatch, cfg.MaxQueuedBatches),
		receivedItems: make(chan []interface{}, cfg.MaxQueuedBatches),
		quit:          make(chan struct{}),
		fetching:      make(map[interface{}]fetchingItem),
		callback:      callback,
	}
	f.announces, _ = wlru.NewWithEvict(uint(cfg.HashLimit), cfg.HashLimit, func(key interface{}, _ interface{}) {
		delete(f.fetching, key)
	})
	f.parallelTasks = workers.New(&f.wg, f.quit, f.cfg.MaxParallelRequests*2)
	return f
}

// Start boots up the items fetcher.
func (f *Fetcher) Start() {
	f.parallelTasks.Start(f.cfg.MaxParallelRequests)
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.loop()
	}()
}

// Stop interrupts the fetcher, canceling all the pending operations.
// Stop waits until all the internal goroutines have finished.
func (f *Fetcher) Stop() {
	close(f.quit)
	f.parallelTasks.Drain()
	f.wg.Wait()
}

// Overloaded returns true if too much items are being requested
func (f *Fetcher) Overloaded() bool {
	return len(f.receivedItems) > f.cfg.MaxQueuedBatches*3/4 ||
		len(f.notifications) > f.cfg.MaxQueuedBatches*3/4 ||
		f.announces.Len() > f.cfg.HashLimit/2
}

// NotifyAnnounces announces the fetcher of the potential availability of a new item in
// the network.
func (f *Fetcher) NotifyAnnounces(peer string, ids []interface{}, time time.Time, fetchItems ItemsRequesterFn) error {
	// divide big batch into smaller ones
	for start := 0; start < len(ids); start += f.cfg.MaxBatch {
		end := len(ids)
		if end > start+f.cfg.MaxBatch {
			end = start + f.cfg.MaxBatch
		}
		op := announcesBatch{
			announceData: announceData{
				time:       time,
				peer:       peer,
				fetchItems: fetchItems,
			},
			ids: ids[start:end],
		}
		select {
		case <-f.quit:
			return errTerminated
		case f.notifications <- op:
			continue
		}
	}
	return nil
}

func (f *Fetcher) NotifyReceived(ids []interface{}) error {
	// divide big batch into smaller ones
	for start := 0; start < len(ids); start += f.cfg.MaxBatch {
		end := len(ids)
		if end > start+f.cfg.MaxBatch {
			end = start + f.cfg.MaxBatch
		}
		select {
		case <-f.quit:
			return errTerminated
		case f.receivedItems <- ids[start:end]:
			continue
		}
	}
	return nil
}

func (f *Fetcher) getAnnounces(id interface{}) []announceData {
	announces, ok := f.announces.Get(id)
	if !ok {
		return []announceData{}
	}
	return announces.([]announceData)
}

func (f *Fetcher) processNotification(notification announcesBatch, fetchTimer *time.Timer) {
	first := len(f.fetching) == 0

	// filter only not known
	notification.ids = f.callback.OnlyInterested(notification.ids)
	if len(notification.ids) == 0 {
		return
	}

	noFetching := f.callback.Suspend()

	toFetch := make([]interface{}, 0, len(notification.ids))
	now := time.Now()
	for _, id := range notification.ids {
		// add new announcement. other peers may already have announced it, so it's an array
		anns := f.getAnnounces(id)
		anns = append(anns, notification.announceData)
		f.announces.Add(id, append(anns, notification.announceData), uint(len(anns)))
		// if it wasn't announced before, then schedule for fetching this time
		if !noFetching {
			if _, ok := f.fetching[id]; !ok {
				f.fetching[id] = fetchingItem{
					announce:     notification.announceData,
					fetchingTime: now,
				}
				toFetch = append(toFetch, id)
			}
		}
	}

	if len(toFetch) != 0 {
		// Create a closure of the fetch and schedule in on a new thread
		fetchItems, hashes := notification.fetchItems, toFetch
		_ = f.parallelTasks.Enqueue(func() {
			_ = fetchItems(hashes)
		})
	}

	if first && len(f.fetching) != 0 {
		f.rescheduleFetch(fetchTimer)
	}
}

// Loop is the main fetcher loop, checking and processing various notifications
func (f *Fetcher) loop() {
	// Iterate the item fetching until a quit is requested
	fetchTimer := time.NewTimer(0)
	defer fetchTimer.Stop()

	for {
		// Wait for an outside item to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notifications:
			f.processNotification(notification, fetchTimer)

		case ids := <-f.receivedItems:
			for _, id := range ids {
				f.forgetHash(id)
			}

		case <-fetchTimer.C:
			now := time.Now()
			// At least one item's timer ran out, check for needing retrieval
			request := make(map[string][]interface{})
			requestFns := make(map[string]ItemsRequesterFn)

			// Find not arrived items
			all := append([]interface{}(nil), f.announces.Keys()...)
			notArrived := f.callback.OnlyInterested(all)

			for _, id := range notArrived {
				// Re-fetch not arrived items
				announces := f.getAnnounces(id)
				if len(announces) == 0 {
					continue
				}

				oldest := announces[0] // first is the oldest
				if time.Since(oldest.time) > f.cfg.ForgetTimeout {
					// Forget too old announces
					f.forgetHash(id)
				} else if time.Since(f.fetching[id].fetchingTime) > f.cfg.ArriveTimeout-f.cfg.GatherSlack {
					// The item still didn't arrive, queue for fetching from a random peer
					announce := announces[rand.Intn(len(announces))] // nolint:gosec
					request[announce.peer] = append(request[announce.peer], id)
					requestFns[announce.peer] = announce.fetchItems
					f.fetching[id] = fetchingItem{
						announce:     announce,
						fetchingTime: now,
					}
				}
			}

			// Forget arrived items.
			// It's possible to get here only if item arrived out-of-fetcher, via another channel.
			// Also may be possible after a change of an epoch.
			notArrivedMap := make(map[interface{}]bool, len(notArrived))
			for _, id := range notArrived {
				notArrivedMap[id] = true
			}
			for _, id := range all {
				if !notArrivedMap[id] {
					f.forgetHash(id)
				}
			}

			// Send out all item requests
			for peer, req := range request {
				// Create a closure of the fetch and schedule in on a new thread
				fetchItems, hashes := requestFns[peer], req
				_ = f.parallelTasks.Enqueue(func() {
					_ = fetchItems(hashes)
				})
			}
			// Schedule the next fetch if items are still pending
			f.rescheduleFetch(fetchTimer)
		}
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

// rescheduleFetch resets the specified fetch timer to the next announce timeout.
func (f *Fetcher) rescheduleFetch(fetch *time.Timer) {
	// Short circuit if no items are announced
	if f.announces.Len() == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	i := 0
	maxChecks := f.cfg.HashLimit / 32
	for _, fetch := range f.fetching {
		if earliest.After(fetch.fetchingTime) {
			earliest = fetch.fetchingTime
		}
		if i >= maxChecks {
			// no need to scan all the entries
			break
		}
		i++
	}
	// limit minimum duration to prevent spinning too often
	fetch.Reset(maxDuration(f.cfg.ArriveTimeout-time.Since(earliest), f.cfg.ArriveTimeout/8))
}

// forgetHash removes all traces of a item announcement from the fetcher's
// internal state.
func (f *Fetcher) forgetHash(id interface{}) {
	f.announces.Remove(id) // f.fetching is deleted inside the evict callback
}
