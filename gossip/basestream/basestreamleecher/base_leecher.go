package basestreamleecher

import (
	"sync"
	"time"
)

// BaseLeecher is a generic items downloader
type BaseLeecher struct {
	// Callbacks
	callback Callbacks

	recheckInterval time.Duration

	Peers map[string]struct{}

	Quit chan struct{}

	Wg sync.WaitGroup

	Mu *sync.RWMutex

	Terminated bool
}

// New creates a generic items downloader
func New(recheckInterval time.Duration, callback Callbacks) *BaseLeecher {
	return &BaseLeecher{
		callback:        callback,
		recheckInterval: recheckInterval,
		Peers:           make(map[string]struct{}),
		Quit:            make(chan struct{}),
		Mu:              new(sync.RWMutex),
	}
}

type Callbacks struct {
	SelectSessionPeerCandidates func() []string
	ShouldTerminateSession      func() bool
	StartSession                func(candidates []string)
	TerminateSession            func()
	OngoingSession              func() bool
	OngoingSessionPeer          func() string
}

func (d *BaseLeecher) Start() {
	d.Wg.Add(1)
	go func() {
		defer d.Wg.Done()
		d.loop()
	}()
}

func (d *BaseLeecher) Routine() {
	if d.Terminated {
		return
	}
	if d.callback.OngoingSession() && d.callback.ShouldTerminateSession() {
		d.callback.TerminateSession()
	}
	if !d.callback.OngoingSession() {
		candidates := d.callback.SelectSessionPeerCandidates()
		if len(candidates) != 0 {
			d.callback.StartSession(candidates)
		}
	}
}

func (d *BaseLeecher) loop() {
	syncTicker := time.NewTicker(d.recheckInterval)
	defer syncTicker.Stop()
	for {
		select {
		case <-d.Quit:
			return
		case <-syncTicker.C:
			d.Mu.Lock()
			d.Routine()
			d.Mu.Unlock()
		}
	}
}

// RegisterPeer injects a new download peer to download items from.
func (d *BaseLeecher) RegisterPeer(peer string) error {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	if d.Terminated {
		return nil
	}
	d.Peers[peer] = struct{}{}

	return nil
}

func (d *BaseLeecher) PeersNum() int {
	d.Mu.RLock()
	defer d.Mu.RUnlock()

	return len(d.Peers)
}

// UnregisterPeer removes a peer from the known list, preventing current or any future sessions with the peer
func (d *BaseLeecher) UnregisterPeer(peer string) error {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	if d.callback.OngoingSessionPeer() == peer {
		d.callback.TerminateSession()
		d.Routine()
	}
	delete(d.Peers, peer)
	return nil
}

func (d *BaseLeecher) Terminate() {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	d.Terminated = true
	close(d.Quit)
	d.callback.TerminateSession()
}

// Stop interrupts the leecher, canceling all the pending operations.
// Stop waits until all the internal goroutines have finished.
func (d *BaseLeecher) Stop() {
	d.Terminate()
	d.Wg.Wait()
}
