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
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Writer writes one message to a stream connection, encoded as JSON. It is
// safe for concurrent use.
type Writer func(ctx context.Context, message any) error

// Notifier holds the subscriptions of one stream connection and sends their
// notifications. The producers of the notifications run under Notifier.Go:
// they end with the connection, and Close cancels them first and then waits
// for them.
type Notifier struct {
	dispatcher *Dispatcher
	ctx        context.Context
	cancel     context.CancelFunc
	write      Writer
	disconnect func(reason error)
	limit      int

	mutex         sync.Mutex
	closed        bool
	subscriptions map[string]*Subscription
	producers     sync.WaitGroup
}

// NewNotifier returns the notifier of a connection whose context is ctx.
// write sends a message on the connection; disconnect closes it, telling the
// client why when it can. A connection holds at most limit subscriptions.
func NewNotifier(ctx context.Context, dispatcher *Dispatcher, write Writer, disconnect func(reason error), limit int) *Notifier {
	ctx, cancel := context.WithCancel(ctx)
	return &Notifier{
		dispatcher:    dispatcher,
		ctx:           ctx,
		cancel:        cancel,
		write:         write,
		disconnect:    disconnect,
		limit:         limit,
		subscriptions: map[string]*Subscription{},
	}
}

// Close ends every subscription of the connection: it cancels their producers
// first and then waits for them.
func (notifier *Notifier) Close() {
	notifier.mutex.Lock()
	notifier.closed = true
	subscriptions := notifier.subscriptions
	notifier.subscriptions = map[string]*Subscription{}
	notifier.mutex.Unlock()

	notifier.cancel()
	for _, subscription := range subscriptions {
		subscription.end()
	}
	notifier.producers.Wait()
}

// Unsubscribe ends the subscription with the given id and tells whether the
// connection had one.
func (notifier *Notifier) Unsubscribe(identifier string) bool {
	notifier.mutex.Lock()
	subscription, found := notifier.subscriptions[identifier]
	delete(notifier.subscriptions, identifier)
	notifier.mutex.Unlock()
	if found {
		subscription.end()
	}
	return found
}

// Call is the view of a notifier one call has: the subscriptions it creates
// stay silent until the transport wrote the response that carries their id.
type Call struct {
	notifier *Notifier
	mutex    sync.Mutex
	created  []*Subscription
}

// NewCall returns the view of the notifier for one call.
func (notifier *Notifier) NewCall() *Call {
	return &Call{notifier: notifier}
}

type callKey struct{}

// WithCall puts the notifier view of a call on its context.
func WithCall(ctx context.Context, call *Call) context.Context {
	return context.WithValue(ctx, callKey{}, call)
}

// CallFrom returns the notifier view a stream connection put on the context
// of a call. HTTP calls have none.
func CallFrom(ctx context.Context) (*Call, bool) {
	call, ok := ctx.Value(callKey{}).(*Call)
	return call, ok
}

// ErrTooManySubscriptions refuses a subscription above the cap of a
// connection.
var ErrTooManySubscriptions = errors.New("too many subscriptions on this connection")

// Subscribe creates a subscription of the given kind whose notifications go
// out as namespace_subscription. Its producer is started with Go.
func (call *Call) Subscribe(namespace, kind string) (*Subscription, error) {
	notifier := call.notifier
	notifier.mutex.Lock()
	defer notifier.mutex.Unlock()
	if notifier.closed {
		return nil, errors.New("connection closed")
	}
	if len(notifier.subscriptions) >= notifier.limit {
		return nil, fmt.Errorf("%w, the limit is %d", ErrTooManySubscriptions, notifier.limit)
	}
	ctx, cancel := context.WithCancel(notifier.ctx)
	subscription := &Subscription{
		ID:       NewID(),
		notifier: notifier,
		method:   namespace + "_subscription",
		kind:     kind,
		ctx:      ctx,
		cancel:   cancel,
		active:   make(chan struct{}),
	}
	notifier.subscriptions[subscription.ID] = subscription
	notifier.dispatcher.metrics.Subscriptions.WithLabelValues(kind).Inc()

	call.mutex.Lock()
	call.created = append(call.created, subscription)
	call.mutex.Unlock()
	return subscription, nil
}

// Activate lets the subscriptions the call created send their notifications.
// The transport calls it after it wrote the response of the call.
func (call *Call) Activate() {
	call.mutex.Lock()
	created := call.created
	call.created = nil
	call.mutex.Unlock()
	for _, subscription := range created {
		subscription.activate()
	}
}

// Subscription is one subscription of a stream connection.
type Subscription struct {
	ID string

	notifier     *Notifier
	method       string
	kind         string
	ctx          context.Context
	cancel       context.CancelFunc
	active       chan struct{}
	activateOnce sync.Once
	endOnce      sync.Once
}

func (subscription *Subscription) activate() {
	subscription.activateOnce.Do(func() { close(subscription.active) })
}

// Go starts the producer of the subscription. It runs until the subscription
// or the connection ends; a panic in it ends the connection with the error.
func (subscription *Subscription) Go(run func(ctx context.Context)) {
	notifier := subscription.notifier
	notifier.producers.Add(1)
	started := notifier.dispatcher.Go(subscription.ctx, subscription.kind, func(ctx context.Context) {
		defer notifier.producers.Done()
		defer func() {
			if value := recover(); value != nil {
				notifier.dispatcher.recovered(subscription.kind, value)
				subscription.Fail(ErrPanic)
			}
		}()
		run(ctx)
	})
	if !started {
		notifier.producers.Done()
		subscription.Fail(ErrStopping)
	}
}

// Done is closed when the subscription ended.
func (subscription *Subscription) Done() <-chan struct{} {
	return subscription.ctx.Done()
}

// Notify sends one notification. It waits until the response carrying the id
// of the subscription was written.
func (subscription *Subscription) Notify(ctx context.Context, result any) error {
	select {
	case <-subscription.active:
	case <-subscription.ctx.Done():
		return errors.New("subscription ended")
	case <-ctx.Done():
		return ctx.Err()
	}
	return subscription.notifier.write(ctx, notification{
		Version: "2.0",
		Method:  subscription.method,
		Parameters: notificationParameters{
			Subscription: subscription.ID,
			Result:       result,
		},
	})
}

// Fail ends the subscription and its connection: a subscription that lost
// events, or whose producer failed, cannot go on, and the client learns it
// by losing the connection with the reason.
func (subscription *Subscription) Fail(reason error) {
	notifier := subscription.notifier
	notifier.mutex.Lock()
	delete(notifier.subscriptions, subscription.ID)
	notifier.mutex.Unlock()
	subscription.end()
	notifier.disconnect(fmt.Errorf("subscription %s ended: %w", subscription.ID, reason))
}

func (subscription *Subscription) end() {
	subscription.endOnce.Do(func() {
		subscription.cancel()
		subscription.notifier.dispatcher.metrics.Subscriptions.WithLabelValues(subscription.kind).Dec()
	})
}

type notification struct {
	Version    string                 `json:"jsonrpc"`
	Method     string                 `json:"method"`
	Parameters notificationParameters `json:"params"`
}

type notificationParameters struct {
	Subscription string `json:"subscription"`
	Result       any    `json:"result"`
}

// NewID returns an unguessable identifier in go-ethereum's format: a quantity
// of 128 random bits.
func NewID() string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		panic(fmt.Sprintf("no randomness for an id: %v", err))
	}
	encoded := strings.TrimLeft(hex.EncodeToString(random[:]), "0")
	if encoded == "" {
		encoded = "0"
	}
	return "0x" + encoded
}

// Unsubscribe ends a subscription of the connection the call belongs to and
// tells whether the connection had it.
func (call *Call) Unsubscribe(identifier string) bool {
	return call.notifier.Unsubscribe(identifier)
}
