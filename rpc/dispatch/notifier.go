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
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// maxSubscriptionsPerConnection bounds the subscriptions of a connection.
const maxSubscriptionsPerConnection = 10_000

// ErrSubscriptionNotFound is the answer to an unsubscription of an id the
// connection has no subscription of, in go-ethereum's words.
var ErrSubscriptionNotFound = errors.New("subscription not found")

// Notifier holds the subscriptions of one stream connection and sends their
// notifications on it.
type Notifier struct {
	dispatcher *Dispatcher
	// send writes a message to the connection, end ends the connection.
	send func(message any) error
	end  func(reason error)
	ctx  context.Context

	mutex         sync.Mutex
	subscriptions map[ID]*Subscription
	closed        bool
	producers     sync.WaitGroup
}

// NewNotifier creates the notifier of a connection, whose context ends with
// the connection. send writes a message to the connection; end ends the
// connection and must not wait for the notifier.
func NewNotifier(ctx context.Context, dispatcher *Dispatcher, send func(message any) error, end func(reason error)) *Notifier {
	return &Notifier{dispatcher: dispatcher, send: send, end: end, ctx: ctx, subscriptions: map[ID]*Subscription{}}
}

// Subscription is a subscription of a connection. Its notifications are
// sent once its id has been answered.
type Subscription struct {
	ID        ID
	namespace string
	kind      string
	notifier  *Notifier
	ctx       context.Context
	cancel    context.CancelFunc
	active    chan struct{}
	activate  sync.Once
}

// add creates a subscription of the connection.
func (n *Notifier) add(namespace, kind string) (*Subscription, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.closed {
		return nil, errors.New("the connection is closed")
	}
	if len(n.subscriptions) >= maxSubscriptionsPerConnection {
		return nil, fmt.Errorf("too many subscriptions on this connection, the limit is %d", maxSubscriptionsPerConnection)
	}
	ctx, cancel := context.WithCancel(n.ctx)
	added := &Subscription{ID: NewID(), namespace: namespace, kind: kind, notifier: n, ctx: ctx, cancel: cancel, active: make(chan struct{})}
	n.subscriptions[added.ID] = added
	n.dispatcher.metrics.subscriptions.WithLabelValues(kind).Inc()
	return added, nil
}

// Unsubscribe ends a subscription of the connection and tells whether there
// was one.
func (n *Notifier) Unsubscribe(id ID) bool {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	found, exists := n.subscriptions[id]
	if exists {
		n.removeLocked(found)
	}
	return exists
}

func (n *Notifier) removeLocked(subscription *Subscription) {
	if _, exists := n.subscriptions[subscription.ID]; !exists {
		return
	}
	delete(n.subscriptions, subscription.ID)
	subscription.cancel()
	n.dispatcher.metrics.subscriptions.WithLabelValues(subscription.kind).Dec()
}

func (n *Notifier) remove(subscription *Subscription) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.removeLocked(subscription)
}

// Close ends every subscription of the connection, then waits for their
// producers; the reverse order would wait for producers that wait to send.
func (n *Notifier) Close() {
	n.mutex.Lock()
	n.closed = true
	for _, subscription := range n.subscriptions {
		n.removeLocked(subscription)
	}
	n.mutex.Unlock()
	n.producers.Wait()
}

// Activate lets the notifications of the subscription be sent; the transport
// calls it once it has written the subscription's id.
func (s *Subscription) Activate() {
	s.activate.Do(func() { close(s.active) })
}

// Go runs a producer of the subscription's notifications in a goroutine of
// the dispatcher, under its recover. Its context ends with the subscription,
// and the connection waits for it when it closes. Go answers false, without
// running anything, once the connection or the dispatcher is closing.
func (s *Subscription) Go(produce func(ctx context.Context)) bool {
	n := s.notifier
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.closed {
		return false
	}
	n.producers.Add(1)
	started := n.dispatcher.Go("subscription "+s.kind, func(context.Context) {
		defer n.producers.Done()
		produce(s.ctx)
	})
	if !started {
		n.producers.Done()
	}
	return started
}

// notification is the message of a notification.
type notification struct {
	Version string             `json:"jsonrpc"`
	Method  string             `json:"method"`
	Params  notificationParams `json:"params"`
}

type notificationParams struct {
	Subscription ID  `json:"subscription"`
	Result       any `json:"result"`
}

// Notify sends a notification, once the subscription's id has been
// answered. It fails once the subscription or its connection has ended; a
// connection that fails to write is ended.
func (s *Subscription) Notify(result any) error {
	select {
	case <-s.active:
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	err := s.notifier.send(&notification{
		Version: "2.0",
		Method:  s.namespace + "_subscription",
		Params:  notificationParams{Subscription: s.ID, Result: result},
	})
	outcome := "sent"
	if err != nil {
		outcome = "failed"
		s.notifier.end(fmt.Errorf("failed to send a notification of subscription %s: %w", s.ID, err))
	}
	s.notifier.dispatcher.metrics.subscriptionEvents.WithLabelValues(s.kind, outcome).Inc()
	return err
}

// Drop ends the subscription's connection, as the subscription has lost
// events: a client that goes on would miss them unawares.
func (s *Subscription) Drop(reason error) {
	log.Warn("RPC subscription dropped with its connection", "subscription", s.ID, "kind", s.kind, "reason", reason)
	s.notifier.end(fmt.Errorf("subscription %s dropped: %w", s.ID, reason))
}
