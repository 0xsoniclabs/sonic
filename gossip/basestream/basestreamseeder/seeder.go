package basestreamseeder

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsoniclabs/consensus/utils/workers"
	"github.com/0xsoniclabs/sonic/gossip/basestream"
)

var (
	ErrSelectorMismatch = errors.New("session selector mismatch")
	ErrTooManyChunks    = errors.New("too many request chunks")
	errTerminated       = errors.New("terminated")
)

type BaseSeeder struct {
	callback Callbacks

	peerSessions map[string][]uint32
	sessions     map[sessionIDAndPeer]sessionState

	notifyUnregisteredPeer chan string
	notifyReceivedRequest  chan *requestAndPeer
	quit                   chan struct{}
	done                   bool

	cfg Config

	wg sync.WaitGroup

	senders              []*workers.Workers
	pendingResponsesSize int64
	sessionsCounter      uint32
}

func New(cfg Config, callbacks Callbacks) *BaseSeeder {
	s := &BaseSeeder{
		callback:               callbacks,
		peerSessions:           make(map[string][]uint32),
		sessions:               make(map[sessionIDAndPeer]sessionState),
		notifyUnregisteredPeer: make(chan string, 128),
		notifyReceivedRequest:  make(chan *requestAndPeer, 16),
		senders:                make([]*workers.Workers, cfg.SenderThreads),
		quit:                   make(chan struct{}),
		cfg:                    cfg,
	}
	for i := 0; i < cfg.SenderThreads; i++ {
		s.senders[i] = workers.New(&s.wg, s.quit, s.cfg.MaxSenderTasks)
	}
	return s
}

type Callbacks struct {
	ForEachItem func(start basestream.Locator, rType basestream.RequestType, onKey func(key basestream.Locator) bool, onAppended func(items basestream.Payload) bool) basestream.Payload
}

type Peer struct {
	ID           string
	SendChunk    func(basestream.Response) error
	Misbehaviour func(error)
}

type sessionIDAndPeer struct {
	id   uint32
	peer string
}

type requestAndPeer struct {
	request basestream.Request
	peer    Peer
}

type sessionState struct {
	origSelector basestream.Locator
	next         basestream.Locator
	stop         basestream.Locator
	done         bool
	senderI      int
	sendChunk    func(basestream.Response) error
}

func (s *BaseSeeder) Start() {
	for i := 0; i < s.cfg.SenderThreads; i++ {
		s.senders[i].Start(1)
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.readerLoop()
	}()
}

// Stop interrupts the seeder, canceling all the pending operations.
// Stop waits until all the internal goroutines have finished.
func (s *BaseSeeder) Stop() {
	close(s.quit)
	s.done = true
	for i := 0; i < s.cfg.SenderThreads; i++ {
		s.senders[i].Drain()
	}
	s.wg.Wait()
}

func (s *BaseSeeder) NotifyRequestReceived(peer Peer, r basestream.Request) (err error, peerErr error) {
	if r.MaxChunks > s.cfg.MaxResponseChunks {
		return nil, ErrTooManyChunks
	}
	// sanitize maximum chunk limits
	if r.MaxPayloadNum > s.cfg.MaxResponsePayloadNum {
		r.MaxPayloadNum = s.cfg.MaxResponsePayloadNum
	}
	if r.MaxPayloadSize > s.cfg.MaxResponsePayloadSize {
		r.MaxPayloadSize = s.cfg.MaxResponsePayloadSize
	}
	// submit request
	op := &requestAndPeer{
		peer:    peer,
		request: r,
	}
	select {
	case s.notifyReceivedRequest <- op:
		return nil, nil
	case <-s.quit:
		return errTerminated, nil
	}
}

func (s *BaseSeeder) UnregisterPeer(peer string) error {
	select {
	case s.notifyUnregisteredPeer <- peer:
		return nil
	case <-s.quit:
		return errTerminated
	}
}

func (s *BaseSeeder) waitPendingResponsesBelowLimit() {
	for atomic.LoadInt64(&s.pendingResponsesSize) >= s.cfg.MaxPendingResponsesSize {
		if s.done {
			// terminating, abort all operations
			return
		}
		// we shouldn't get here normally, so it's fine to spin instead of a conditional variable
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *BaseSeeder) readerLoop() {
	for {
		// Wait for an outside event to occur
		select {
		case <-s.quit:
			// terminating, abort all operations
			return

		case peerID := <-s.notifyUnregisteredPeer:
			sessions := s.peerSessions[peerID]
			for _, sid := range sessions {
				delete(s.sessions, sessionIDAndPeer{sid, peerID})
			}
			delete(s.peerSessions, peerID)

		case op := <-s.notifyReceivedRequest:
			s.waitPendingResponsesBelowLimit()

			// prune oldest session
			sessions := s.peerSessions[op.peer.ID]
			if len(sessions) > 2 {
				oldest := sessions[0]
				sessions = sessions[1:]
				delete(s.sessions, sessionIDAndPeer{oldest, op.peer.ID})
			}

			// add session
			session, ok := s.sessions[sessionIDAndPeer{op.request.Session.ID, op.peer.ID}]
			if !ok {
				session.origSelector = op.request.Session.Start
				session.next = op.request.Session.Start
				session.stop = op.request.Session.Stop
				session.sendChunk = op.peer.SendChunk
				session.senderI = int(s.sessionsCounter % uint32(s.cfg.SenderThreads))
				sessions = append(sessions, op.request.Session.ID)
				s.peerSessions[op.peer.ID] = sessions
				s.sessionsCounter++
			}

			// sanity check (cannot change session parameters after it's created)
			if session.origSelector.Compare(op.request.Session.Start) != 0 {
				op.peer.Misbehaviour(ErrSelectorMismatch)
				continue
			}

			for i := uint32(0); i < op.request.MaxChunks && !session.done; i++ {
				allConsumed := true
				resp := basestream.Response{}
				lastKey := session.next
				resp.Payload = s.callback.ForEachItem(session.next, op.request.Type, func(key basestream.Locator) bool {
					if key.Compare(session.stop) >= 0 {
						return false
					}
					lastKey = key
					return true
				}, func(items basestream.Payload) bool {
					numReached := uint32(items.Len()) >= op.request.MaxPayloadNum
					sizeReached := items.TotalSize() >= op.request.MaxPayloadSize
					if numReached || sizeReached {
						allConsumed = false
						return false
					}
					return true
				})
				// update session
				session.next = lastKey.Inc()
				session.done = allConsumed
				s.sessions[sessionIDAndPeer{op.request.Session.ID, op.peer.ID}] = session

				resp.Done = allConsumed
				resp.SessionID = op.request.Session.ID

				memSize := resp.Payload.TotalMemSize()
				s.waitPendingResponsesBelowLimit()
				atomic.AddInt64(&s.pendingResponsesSize, int64(memSize))
				_ = s.senders[session.senderI].Enqueue(func() {
					_ = session.sendChunk(resp)
					atomic.AddInt64(&s.pendingResponsesSize, -int64(memSize))
				})
			}
		}
	}
}
