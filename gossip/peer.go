package gossip

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/utils/datasemaphore"
	mapset "github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/gossip/protocols/dag/dagstream"
	"github.com/0xsoniclabs/sonic/inter"
)

var (
	errNotRegistered = errors.New("peer is not registered")
)

var (
	sentTxsPromotedCounter    = metrics.GetOrRegisterCounter("p2p_sent_txs_promoted", nil)
	droppedTxsPromotedCounter = metrics.GetOrRegisterCounter("p2p_dropped_txs_promoted", nil)
	sentTxsRequestedCounter   = metrics.GetOrRegisterCounter("p2p_sent_txs_requested", nil)
	sentTxHashesCounter       = metrics.GetOrRegisterCounter("p2p_sent_tx_hashes", nil)
)

const (
	handshakeTimeout = 5 * time.Second
)

// PeerInfo represents a short summary of the sub-protocol metadata known
// about a connected peer.
type PeerInfo struct {
	Version     uint      `json:"version"` // protocol version negotiated
	Epoch       idx.Epoch `json:"epoch"`
	NumOfBlocks idx.Block `json:"blocks"`
}

type broadcastItem struct {
	Code uint64
	Raw  rlp.RawValue
}

type peer struct {
	id string

	cfg PeerCacheConfig

	*p2p.Peer
	rw p2p.MsgReadWriter

	version uint // Protocol version negotiated

	knownTxs            mapset.Set         // Set of transaction hashes known to be known by this peer
	knownEvents         mapset.Set         // Set of event hashes known to be known by this peer
	queue               chan broadcastItem // queue of items to send
	queuedDataSemaphore *datasemaphore.DataSemaphore
	term                chan struct{} // Termination channel to stop the broadcaster

	progress PeerProgress

	useless uint32

	sync.RWMutex

	endPoint atomic.Pointer[peerEndPointInfo]
}

type peerEndPointInfo struct {
	enode     enode.Node
	timestamp time.Time
}

func (p *peer) Useless() bool {
	return atomic.LoadUint32(&p.useless) != 0
}

func (p *peer) SetUseless() {
	atomic.StoreUint32(&p.useless, 1)
}

func (p *peer) SetProgress(x PeerProgress) {
	p.Lock()
	defer p.Unlock()

	p.progress = x
}

func (p *peer) GetProgress() PeerProgress {
	p.RLock()
	defer p.RUnlock()

	return p.progress
}

func (p *peer) InterestedIn(h hash.Event) bool {
	e := h.Epoch()

	p.RLock()
	defer p.RUnlock()

	return e != 0 &&
		p.progress.Epoch != 0 &&
		(e == p.progress.Epoch || e == p.progress.Epoch+1) &&
		!p.knownEvents.Contains(h)
}

func (a *PeerProgress) Less(b PeerProgress) bool {
	if a.Epoch != b.Epoch {
		return a.Epoch < b.Epoch
	}
	return a.LastBlockIdx < b.LastBlockIdx
}

func newPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, cfg PeerCacheConfig) *peer {
	peer := &peer{
		cfg:                 cfg,
		Peer:                p,
		rw:                  rw,
		version:             version,
		id:                  p.ID().String(),
		knownTxs:            mapset.NewSet(),
		knownEvents:         mapset.NewSet(),
		queue:               make(chan broadcastItem, cfg.MaxQueuedItems),
		queuedDataSemaphore: datasemaphore.New(dag.Metric{Num: cfg.MaxQueuedItems, Size: cfg.MaxQueuedSize}, getSemaphoreWarningFn("Peers queue")),
		term:                make(chan struct{}),
	}

	go peer.broadcast(peer.queue)

	return peer
}

// broadcast is a write loop that multiplexes event propagations, announcements
// and transaction broadcasts into the remote peer. The goal is to have an async
// writer that does not lock up node internals.
func (p *peer) broadcast(queue chan broadcastItem) {
	for {
		select {
		case item := <-queue:
			_ = p2p.Send(p.rw, item.Code, item.Raw)
			p.queuedDataSemaphore.Release(memSize(item.Raw))

		case <-p.term:
			return
		}
	}
}

// Close signals the broadcast goroutine to terminate.
func (p *peer) Close() {
	p.queuedDataSemaphore.Terminate()
	close(p.term)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	p.RLock()
	defer p.RUnlock()

	return &PeerInfo{
		Version:     p.version,
		Epoch:       p.progress.Epoch,
		NumOfBlocks: p.progress.LastBlockIdx,
	}
}

// MarkEvent marks a event as known for the peer, ensuring that the event will
// never be propagated to this particular peer.
func (p *peer) MarkEvent(hash hash.Event) {
	// If we reached the memory allowance, drop a previously known event hash
	for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
		p.knownEvents.Pop()
	}
	p.knownEvents.Add(hash)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Cardinality() >= p.cfg.MaxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// SendTransactionHashes sends transaction hashess to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendTransactionHashes(txids []common.Hash) error {
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for _, txid := range txids {
		p.knownTxs.Add(txid)
	}
	for p.knownTxs.Cardinality() >= p.cfg.MaxKnownTxs {
		p.knownTxs.Pop()
	}
	sentTxHashesCounter.Inc(int64(len(txids)))
	return p2p.Send(p.rw, NewEvmTxHashesMsg, txids)
}

func memSize(v rlp.RawValue) dag.Metric {
	return dag.Metric{Num: 1, Size: uint64(len(v) + 1024)}
}

func (p *peer) asyncSendEncodedItem(raw rlp.RawValue, code uint64, queue chan broadcastItem) bool {
	if !p.queuedDataSemaphore.TryAcquire(memSize(raw)) {
		return false
	}
	item := broadcastItem{
		Code: code,
		Raw:  raw,
	}
	select {
	case queue <- item:
		return true
	case <-p.term:
	default:
	}
	p.queuedDataSemaphore.Release(memSize(raw))
	return false
}

func (p *peer) asyncSendNonEncodedItem(value interface{}, code uint64, queue chan broadcastItem) bool {
	raw, err := rlp.EncodeToBytes(value)
	if err != nil {
		return false
	}
	return p.asyncSendEncodedItem(raw, code, queue)
}

func (p *peer) enqueueSendEncodedItem(raw rlp.RawValue, code uint64, queue chan broadcastItem) {
	if !p.queuedDataSemaphore.Acquire(memSize(raw), 10*time.Second) {
		return
	}
	item := broadcastItem{
		Code: code,
		Raw:  raw,
	}
	select {
	case queue <- item:
		return
	case <-p.term:
	}
	p.queuedDataSemaphore.Release(memSize(raw))
}

func (p *peer) enqueueSendNonEncodedItem(value interface{}, code uint64, queue chan broadcastItem) {
	raw, err := rlp.EncodeToBytes(value)
	if err != nil {
		return
	}
	p.enqueueSendEncodedItem(raw, code, queue)
}

func SplitTransactions(txs types.Transactions, fn func(types.Transactions)) {
	// divide big batch into smaller ones
	for len(txs) > 0 {
		batchSize := 0
		var batch types.Transactions
		for i, tx := range txs {
			batchSize += int(tx.Size()) + 1024
			batch = txs[:i+1]
			if batchSize >= softResponseLimitSize || i+1 >= softLimitItems {
				break
			}
		}
		txs = txs[len(batch):]
		fn(batch)
	}
}

// AsyncSendTransactions queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the transactions are silently dropped.
func (p *peer) AsyncSendTransactions(txs types.Transactions, queue chan broadcastItem) {
	if p.asyncSendNonEncodedItem(txs, EvmTxsMsg, queue) {
		sentTxsPromotedCounter.Inc(int64(len(txs)))
		// Mark all the transactions as known, but ensure we don't overflow our limits
		for _, tx := range txs {
			p.knownTxs.Add(tx.Hash())
		}
		for p.knownTxs.Cardinality() >= p.cfg.MaxKnownTxs {
			p.knownTxs.Pop()
		}
	} else {
		droppedTxsPromotedCounter.Inc(int64(len(txs)))
		p.Log().Debug("Dropping transactions propagation", "count", len(txs))
	}
}

// AsyncSendTransactionHashes queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the transactions are silently dropped.
func (p *peer) AsyncSendTransactionHashes(txids []common.Hash, queue chan broadcastItem) {
	if p.asyncSendNonEncodedItem(txids, NewEvmTxHashesMsg, queue) {
		sentTxHashesCounter.Inc(int64(len(txids)))
		// Mark all the transactions as known, but ensure we don't overflow our limits
		for _, tx := range txids {
			p.knownTxs.Add(tx)
		}
		for p.knownTxs.Cardinality() >= p.cfg.MaxKnownTxs {
			p.knownTxs.Pop()
		}
	} else {
		p.Log().Debug("Dropping tx announcement", "count", len(txids))
	}
}

// EnqueueSendTransactions queues list of transactions propagation to a remote
// peer.
// The method is blocking in a case if the peer's broadcast queue is full.
func (p *peer) EnqueueSendTransactions(txs types.Transactions, queue chan broadcastItem) {
	p.enqueueSendNonEncodedItem(txs, EvmTxsMsg, queue)
	sentTxsRequestedCounter.Inc(int64(len(txs)))
	// Mark all the transactions as known, but ensure we don't overflow our limits
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	for p.knownTxs.Cardinality() >= p.cfg.MaxKnownTxs {
		p.knownTxs.Pop()
	}
}

// SendEventIDs announces the availability of a number of events through
// a hash notification.
func (p *peer) SendEventIDs(hashes []hash.Event) error {
	// Mark all the event hashes as known, but ensure we don't overflow our limits
	for _, hash := range hashes {
		p.knownEvents.Add(hash)
	}
	for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
		p.knownEvents.Pop()
	}
	return p2p.Send(p.rw, NewEventIDsMsg, hashes)
}

// AsyncSendEventIDs queues the availability of a event for propagation to a
// remote peer. If the peer's broadcast queue is full, the event is silently
// dropped.
func (p *peer) AsyncSendEventIDs(ids hash.Events, queue chan broadcastItem) {
	if p.asyncSendNonEncodedItem(ids, NewEventIDsMsg, queue) {
		// Mark all the event hash as known, but ensure we don't overflow our limits
		for _, id := range ids {
			p.knownEvents.Add(id)
		}
		for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
			p.knownEvents.Pop()
		}
	} else {
		p.Log().Debug("Dropping event announcement", "count", len(ids))
	}
}

// SendEvents propagates a batch of events to a remote peer.
func (p *peer) SendEvents(events inter.EventPayloads) error {
	// Mark all the event hash as known, but ensure we don't overflow our limits
	for _, event := range events {
		p.knownEvents.Add(event.ID())
		for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
			p.knownEvents.Pop()
		}
	}
	return p2p.Send(p.rw, EventsMsg, events)
}

// SendEventsRLP propagates a batch of RLP events to a remote peer.
func (p *peer) SendEventsRLP(events []rlp.RawValue, ids []hash.Event) error {
	// Mark all the event hash as known, but ensure we don't overflow our limits
	for _, id := range ids {
		p.knownEvents.Add(id)
		for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
			p.knownEvents.Pop()
		}
	}
	return p2p.Send(p.rw, EventsMsg, events)
}

// AsyncSendEvents queues an entire event for propagation to a remote peer.
// If the peer's broadcast queue is full, the events are silently dropped.
func (p *peer) AsyncSendEvents(events inter.EventPayloads, queue chan broadcastItem) bool {
	if p.asyncSendNonEncodedItem(events, EventsMsg, queue) {
		// Mark all the event hash as known, but ensure we don't overflow our limits
		for _, event := range events {
			p.knownEvents.Add(event.ID())
		}
		for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
			p.knownEvents.Pop()
		}
		return true
	}
	p.Log().Debug("Dropping event propagation", "count", len(events))
	return false
}

// EnqueueSendEventsRLP queues an entire RLP event for propagation to a remote peer.
// The method is blocking in a case if the peer's broadcast queue is full.
func (p *peer) EnqueueSendEventsRLP(events []rlp.RawValue, ids []hash.Event, queue chan broadcastItem) {
	p.enqueueSendNonEncodedItem(events, EventsMsg, queue)
	// Mark all the event hash as known, but ensure we don't overflow our limits
	for _, id := range ids {
		p.knownEvents.Add(id)
	}
	for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
		p.knownEvents.Pop()
	}
}

// AsyncSendProgress queues a progress propagation to a remote peer.
// If the peer's broadcast queue is full, the progress is silently dropped.
func (p *peer) AsyncSendProgress(progress PeerProgress, queue chan broadcastItem) {
	if !p.asyncSendNonEncodedItem(progress, ProgressMsg, queue) {
		p.Log().Debug("Dropping peer progress propagation")
	}
}

func (p *peer) RequestEvents(ids hash.Events) error {
	// divide big batch into smaller ones
	for start := 0; start < len(ids); start += softLimitItems {
		end := len(ids)
		if end > start+softLimitItems {
			end = start + softLimitItems
		}
		p.Log().Debug("Fetching batch of events", "count", len(ids[start:end]))
		err := p2p.Send(p.rw, GetEventsMsg, ids[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *peer) RequestTransactions(txids []common.Hash) error {
	// divide big batch into smaller ones
	for start := 0; start < len(txids); start += softLimitItems {
		end := len(txids)
		if end > start+softLimitItems {
			end = start + softLimitItems
		}
		p.Log().Debug("Fetching batch of transactions", "count", len(txids[start:end]))
		err := p2p.Send(p.rw, GetEvmTxsMsg, txids[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *peer) SendEventsStream(r dagstream.Response, ids hash.Events) error {
	// Mark all the event hash as known, but ensure we don't overflow our limits
	for _, id := range ids {
		p.knownEvents.Add(id)
		for p.knownEvents.Cardinality() >= p.cfg.MaxKnownEvents {
			p.knownEvents.Pop()
		}
	}
	return p2p.Send(p.rw, EventsStreamResponse, r)
}

func (p *peer) RequestEventsStream(r dagstream.Request) error {
	return p2p.Send(p.rw, RequestEventsStream, r)
}

// Handshake executes the protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis object.
func (p *peer) Handshake(network uint64, progress PeerProgress, genesis common.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var handshake handshakeData // safe to read after two values have been received from errc

	go func() {
		// send both HandshakeMsg and ProgressMsg
		err := p2p.Send(p.rw, HandshakeMsg, &handshakeData{
			ProtocolVersion: uint32(p.version),
			NetworkID:       network,
			Genesis:         genesis,
		})
		if err != nil {
			errc <- err
		}
		errc <- p.SendProgress(progress)
	}()
	go func() {
		errc <- p.readStatus(network, &handshake, genesis)
		// do not expect ProgressMsg here, because eth62 clients won't send it
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	return nil
}

func (p *peer) SendProgress(progress PeerProgress) error {
	return p2p.Send(p.rw, ProgressMsg, progress)
}

func (p *peer) readStatus(network uint64, handshake *handshakeData, genesis common.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != HandshakeMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, HandshakeMsg)
	}
	if msg.Size > protocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, protocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&handshake); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	// TODO: rm after all the nodes updated to #184
	if handshake.NetworkID == 0 {
		handshake.NetworkID = network
	}

	if handshake.Genesis != genesis {
		return errResp(ErrGenesisMismatch, "%x (!= %x)", handshake.Genesis[:8], genesis[:8])
	}
	if handshake.NetworkID != network {
		return errResp(ErrNetworkIDMismatch, "%d (!= %d)", handshake.NetworkID, network)
	}
	if uint(handshake.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", handshake.ProtocolVersion, p.version)
	}
	return nil
}

// SendPeerInfoRequest sends a request to the peer asking for an update of
// its list of peers.
func (p *peer) SendPeerInfoRequest() error {
	// If the peer doesn't support the peer info protocol, don't bother
	// sending the request. This request would lead to a disconnect
	// if the peer doesn't understand it.
	if !p.Peer.RunningCap(ProtocolName, []uint{_Sonic_64, _Sonic_65}) {
		return nil
	}
	return p2p.Send(p.rw, GetPeerInfosMsg, struct{}{})
}

// SendEndPointUpdateRequest sends a request to the peer asking for the peer's
// public enode address to be used to establish a connection to this peer.
func (p *peer) SendEndPointUpdateRequest() error {
	// If the peer doesn't support version 65 of this protocol, don't bother
	// sending the request. This request would lead to a disconnect
	// if the peer doesn't understand it.
	if !p.Peer.RunningCap(ProtocolName, []uint{_Sonic_65}) {
		return nil
	}
	return p2p.Send(p.rw, GetEndPointMsg, struct{}{})
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("opera/%2d", p.version),
	)
}
