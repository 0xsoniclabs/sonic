// Copyright 2026 Sonic Operations Ltd
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

package dispatch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// Notifier is what a subscription handler sees of its connection. Only stream
// connections (WebSocket, IPC) put one on a request's context.
type Notifier interface {
	// Subscribe opens a subscription of the given kind on the connection. Its
	// notifications are held back until the answer to the subscribing request
	// was written, and the handler installs its feed before it returns the id.
	Subscribe(namespace, kind string) (Subscription, error)
	// Unsubscribe ends a subscription of the connection; it reports false if
	// the connection holds none with that id.
	Unsubscribe(id string) bool
	// Go runs a producer under the connection's recover group. The context
	// ends with the connection; a panic in f closes the connection.
	Go(f func(ctx context.Context)) bool
}

// Subscription is one subscription of a stream connection.
type Subscription interface {
	ID() string
	// Notify queues one notification without blocking. A subscription whose
	// client does not read falls behind its buffer; it is ended, and its
	// connection closed with the reason.
	Notify(result any)
	// Done is closed when the subscription ended.
	Done() <-chan struct{}
	// Drop ends the subscription and its connection with the reason: its
	// consumer lost events, and its client must learn of the gap.
	Drop(reason string)
}

type notifierKey struct{}

// WithNotifier puts the notifier of a stream connection on a request context.
func WithNotifier(ctx context.Context, n Notifier) context.Context {
	return context.WithValue(ctx, notifierKey{}, n)
}

// NotifierFrom returns the notifier of a request's connection, if it has one.
func NotifierFrom(ctx context.Context) (Notifier, bool) {
	n, ok := ctx.Value(notifierKey{}).(Notifier)
	return n, ok
}

// NewID returns an unguessable id in go-ethereum's form: a hex quantity of 16
// random bytes.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("dispatch: no randomness for an id: %v", err))
	}
	id := strings.TrimLeft(hex.EncodeToString(b[:]), "0")
	if id == "" {
		id = "0"
	}
	return "0x" + id
}

// StreamLimits bound what one stream connection may hold.
type StreamLimits struct {
	MaxSubscriptions int // subscriptions per connection
	Buffer           int // queued notifications per subscription
}

// Stream is the state of one stream connection: its subscriptions and their
// producers. The transport creates one per connection.
type Stream struct {
	d      *Dispatcher
	limits StreamLimits
	write  func(msg []byte) error
	closer func(reason string)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu     sync.Mutex
	subs   map[string]*subscription
	closed bool
}

// NewStream creates the state of a stream connection. write sends one
// message on the connection, and closeConn closes it with a reason.
func (d *Dispatcher) NewStream(ctx context.Context, limits StreamLimits, write func(msg []byte) error, closeConn func(reason string)) *Stream {
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(d.ctx, cancel)
	s := &Stream{
		d:      d,
		limits: limits,
		write:  write,
		closer: closeConn,
		ctx:    ctx,
		cancel: func() { stop(); cancel() },
		subs:   make(map[string]*subscription),
	}
	return s
}

// Context ends when the connection closes or the dispatcher stops.
func (s *Stream) Context() context.Context {
	return s.ctx
}

// Close ends every subscription of the connection: it cancels the producers
// first and then waits for them.
func (s *Stream) Close() {
	s.mu.Lock()
	s.closed = true
	subs := s.subs
	s.subs = make(map[string]*subscription)
	s.mu.Unlock()
	s.cancel()
	for _, sub := range subs {
		sub.end()
	}
	s.wg.Wait()
}

// Request returns the notifier of one request of the connection. Finish
// releases the notifications of the subscriptions the request opened once
// its answer was written, or ends them if the request failed.
func (s *Stream) Request() *RequestNotifier {
	return &RequestNotifier{s: s}
}

// Go runs a goroutine of the connection under the dispatcher. A panic in it
// closes the connection.
func (s *Stream) Go(name string, f func(ctx context.Context)) bool {
	s.wg.Add(1)
	ok := s.d.Go(name, func(context.Context) {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.d.recovered(name, r)
				s.closer(fmt.Sprintf("panic in %s", name))
			}
		}()
		f(s.ctx)
	})
	if !ok {
		s.wg.Done()
	}
	return ok
}

// RequestNotifier is the Notifier of one request of a stream connection.
type RequestNotifier struct {
	s       *Stream
	mu      sync.Mutex
	created []*subscription
}

var errStreamClosed = errors.New("connection closed")

func (n *RequestNotifier) Subscribe(namespace, kind string) (Subscription, error) {
	s := n.s
	sub := &subscription{
		s:         s,
		id:        NewID(),
		namespace: namespace,
		kind:      kind,
		queue:     make(chan []byte, s.limits.Buffer),
		active:    make(chan struct{}),
		done:      make(chan struct{}),
	}
	s.mu.Lock()
	switch {
	case s.closed:
		s.mu.Unlock()
		return nil, errStreamClosed
	case len(s.subs) >= s.limits.MaxSubscriptions:
		s.mu.Unlock()
		return nil, fmt.Errorf("too many subscriptions, the limit is %d", s.limits.MaxSubscriptions)
	}
	s.subs[sub.id] = sub
	s.mu.Unlock()

	if !s.Go("subscription "+kind, sub.deliver) {
		s.remove(sub.id)
		return nil, errStopped
	}
	s.d.metrics.Subscriptions(kind).Inc()
	n.mu.Lock()
	n.created = append(n.created, sub)
	n.mu.Unlock()
	return sub, nil
}

func (n *RequestNotifier) Unsubscribe(id string) bool {
	sub := n.s.remove(id)
	if sub == nil {
		return false
	}
	sub.end()
	return true
}

func (n *RequestNotifier) Go(f func(ctx context.Context)) bool {
	return n.s.Go("subscription producer", f)
}

// Finish releases the notifications of the subscriptions the request opened,
// once its answer was written; if the request failed, it ends them.
func (n *RequestNotifier) Finish(failed bool) {
	n.mu.Lock()
	created := n.created
	n.created = nil
	n.mu.Unlock()
	for _, sub := range created {
		if failed {
			n.Unsubscribe(sub.id)
			continue
		}
		close(sub.active)
	}
}

func (s *Stream) remove(id string) *subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub := s.subs[id]
	delete(s.subs, id)
	return sub
}

// subscription delivers the notifications of one subscription in order.
type subscription struct {
	s                   *Stream
	id, namespace, kind string
	queue               chan []byte
	active              chan struct{}
	done                chan struct{}
	endOnce             sync.Once
}

func (sub *subscription) ID() string            { return sub.id }
func (sub *subscription) Done() <-chan struct{} { return sub.done }

func (sub *subscription) Notify(result any) {
	select {
	case <-sub.done:
		return
	default:
	}
	msg, err := sub.encode(result)
	if err != nil {
		log.Error("RPC subscription result not encodable", "kind", sub.kind, "err", err)
		return
	}
	select {
	case sub.queue <- msg:
		sub.s.d.metrics.SubscriptionEvents(sub.kind, OutcomeSent).Inc()
	default:
		sub.s.d.metrics.SubscriptionEvents(sub.kind, OutcomeDropped).Inc()
		sub.Drop(fmt.Sprintf("subscription %s fell behind by %d notifications", sub.id, cap(sub.queue)))
	}
}

func (sub *subscription) Drop(reason string) {
	sub.s.remove(sub.id)
	sub.end()
	sub.s.closer(reason)
}

func (sub *subscription) end() {
	sub.endOnce.Do(func() {
		close(sub.done)
		sub.s.d.metrics.Subscriptions(sub.kind).Dec()
	})
}

// notification is a subscription message as go-ethereum writes it.
type notification struct {
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  notifyPayload `json:"params"`
}

type notifyPayload struct {
	ID     string          `json:"subscription"`
	Result json.RawMessage `json:"result"`
}

func (sub *subscription) encode(result any) ([]byte, error) {
	enc, ok := result.(json.RawMessage)
	if !ok {
		var err error
		if enc, err = json.Marshal(result); err != nil {
			return nil, err
		}
	}
	return json.Marshal(notification{
		Version: "2.0",
		Method:  sub.namespace + "_subscription",
		Params:  notifyPayload{ID: sub.id, Result: enc},
	})
}

// deliver writes the queued notifications once the subscription is active.
func (sub *subscription) deliver(ctx context.Context) {
	select {
	case <-sub.active:
	case <-sub.done:
		return
	case <-ctx.Done():
		return
	}
	for {
		select {
		case msg := <-sub.queue:
			if err := sub.s.write(msg); err != nil {
				sub.Drop(fmt.Sprintf("writing a notification failed: %v", err))
				return
			}
		case <-sub.done:
			return
		case <-ctx.Done():
			return
		}
	}
}
