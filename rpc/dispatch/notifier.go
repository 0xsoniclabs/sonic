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
)

// ErrSubscriptionNotFound is the answer to an unsubscribe request for an id the
// connection does not hold.
var ErrSubscriptionNotFound = errors.New("subscription not found")

// ErrTooManySubscriptions is the answer to a subscribe request on a connection
// that holds as many subscriptions as it may.
var ErrTooManySubscriptions = errors.New("too many subscriptions on this connection")

// newSubscriptionID returns an id nobody can guess: 128 random bits as a hex
// quantity, the form of go-ethereum's ids.
func newSubscriptionID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("rpc: no randomness for subscription ids: %v", err))
	}
	id := strings.TrimLeft(hex.EncodeToString(raw[:]), "0")
	if id == "" {
		id = "0"
	}
	return "0x" + id
}

// NewFilterID returns a fresh unguessable id for a filter.
func NewFilterID() string { return newSubscriptionID() }

// Sink carries the notifications of a connection to its client. Send must not
// block: it returns an error if the client does not keep up or the connection
// is closed.
type Sink interface {
	Send(msg any) error
}

// Notifier is the connection-scoped handle through which subscriptions deliver
// notifications. HTTP requests have none.
type Notifier struct {
	d       *Dispatcher
	sink    Sink
	ctx     context.Context
	cancel  context.CancelCauseFunc
	maxSubs int

	wg sync.WaitGroup // producers

	mu     sync.Mutex
	subs   map[string]*Subscription
	closed bool
}

// NewNotifier creates the notifier of a connection. maxSubs bounds the
// subscriptions the connection may hold, 0 = unbounded.
func (d *Dispatcher) NewNotifier(parent context.Context, sink Sink, maxSubs int) *Notifier {
	ctx, cancel := context.WithCancelCause(parent)
	return &Notifier{d: d, sink: sink, ctx: ctx, cancel: cancel, maxSubs: maxSubs, subs: map[string]*Subscription{}}
}

type notifierKey struct{}

// WithNotifier returns a context carrying the notifier.
func WithNotifier(ctx context.Context, n *Notifier) context.Context {
	return context.WithValue(ctx, notifierKey{}, n)
}

// NotifierFrom returns the notifier of the connection a request came on.
func NotifierFrom(ctx context.Context) (*Notifier, bool) {
	n, ok := ctx.Value(notifierKey{}).(*Notifier)
	return n, ok
}

type namespaceKey struct{}

// Subscription is one subscription of a connection.
type Subscription struct {
	id        string
	kind      string
	namespace string
	n         *Notifier
	ctx       context.Context
	cancel    context.CancelFunc
	active    chan struct{}
	once      sync.Once
}

// NewSubscription registers a subscription of the given kind on the
// connection. The handler of eth_subscribe calls it, installs its feeds, and
// returns the id; no notification leaves before the client has the id.
func (n *Notifier) NewSubscription(ctx context.Context, kind string) (*Subscription, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil, errors.New("connection closed")
	}
	if n.maxSubs > 0 && len(n.subs) >= n.maxSubs {
		return nil, ErrTooManySubscriptions
	}
	ns, _ := ctx.Value(namespaceKey{}).(string)
	sctx, cancel := context.WithCancel(n.ctx)
	sub := &Subscription{
		id: newSubscriptionID(), kind: kind, namespace: ns, n: n,
		ctx: sctx, cancel: cancel, active: make(chan struct{}),
	}
	n.subs[sub.id] = sub
	n.d.metrics.Subscriptions.WithLabelValues(kind).Inc()
	return sub, nil
}

// ID is the id the client knows the subscription by.
func (s *Subscription) ID() string { return s.id }

// Done is closed when the subscription ended: it was unsubscribed or the
// connection closed.
func (s *Subscription) Done() <-chan struct{} { return s.ctx.Done() }

// Context ends with the subscription.
func (s *Subscription) Context() context.Context { return s.ctx }

// Close ends the subscription.
func (s *Subscription) Close() {
	s.once.Do(func() {
		s.cancel()
		s.n.mu.Lock()
		if _, ok := s.n.subs[s.id]; ok {
			delete(s.n.subs, s.id)
			s.n.d.metrics.Subscriptions.WithLabelValues(s.kind).Dec()
		}
		s.n.mu.Unlock()
	})
}

// Activate lets the notifications of the subscription out. The connection
// calls it after it wrote the response that carries the id.
func (n *Notifier) Activate(id string) {
	n.mu.Lock()
	sub := n.subs[id]
	n.mu.Unlock()
	if sub != nil {
		select {
		case <-sub.active:
		default:
			close(sub.active)
		}
	}
}

type notification struct {
	Version string             `json:"jsonrpc"`
	Method  string             `json:"method"`
	Params  notificationParams `json:"params"`
}

type notificationParams struct {
	Subscription string `json:"subscription"`
	Result       any    `json:"result"`
}

// Notify sends one event to the client. It waits until the client has the id
// of the subscription. A client that does not keep up fails the connection.
func (s *Subscription) Notify(payload any) error {
	select {
	case <-s.active:
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	metrics := s.n.d.metrics
	err := s.n.sink.Send(&notification{
		Version: "2.0",
		Method:  s.namespace + "_subscription",
		Params:  notificationParams{Subscription: s.id, Result: payload},
	})
	if err != nil {
		metrics.SubscriptionEvents.WithLabelValues(s.kind, "failed").Inc()
		s.n.Abort(err)
		return err
	}
	metrics.SubscriptionEvents.WithLabelValues(s.kind, "sent").Inc()
	return nil
}

// Unsubscribe ends the subscription with the id; false if the connection holds
// none.
func (n *Notifier) Unsubscribe(id string) bool {
	n.mu.Lock()
	sub := n.subs[id]
	n.mu.Unlock()
	if sub == nil {
		return false
	}
	sub.Close()
	return true
}

// Go runs a producer of the connection under the connection's recover group.
// The producer ends with the connection; a panic in it fails the connection.
func (n *Notifier) Go(fn func()) bool {
	n.wg.Add(1)
	ok := n.d.Go("subscription", func(context.Context) {
		defer n.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				n.Abort(fmt.Errorf("subscription producer panicked: %v", r))
				panic(r) // recorded by the dispatcher
			}
		}()
		fn()
	})
	if !ok {
		n.wg.Done()
	}
	return ok
}

// Abort fails the connection: every subscription ends and the transport closes
// the connection.
func (n *Notifier) Abort(cause error) { n.cancel(cause) }

// Failed is closed when the connection failed or was closed.
func (n *Notifier) Failed() <-chan struct{} { return n.ctx.Done() }

// Cause tells why the connection ended.
func (n *Notifier) Cause() error { return context.Cause(n.ctx) }

// Close ends the connection's subscriptions: it cancels first and then waits
// for the producers, never the other way round.
func (n *Notifier) Close() {
	n.mu.Lock()
	n.closed = true
	subs := make([]*Subscription, 0, len(n.subs))
	for _, s := range n.subs {
		subs = append(subs, s)
	}
	n.mu.Unlock()
	n.cancel(errors.New("connection closed"))
	n.wg.Wait()
	for _, s := range subs {
		s.Close()
	}
}

// -- subscription methods --

// AddSubscription registers a subscription kind of a namespace. The handler
// receives the parameters after the kind; it returns the id of the new
// subscription. The first parameter name is that of the kind.
func (d *Dispatcher) AddSubscription(ns, kind string, h Handler, params ...string) {
	m := &method{name: ns + "_subscribe", handler: h}
	all := append([]string{"subscription"}, params...)
	m.params = nil
	for i, p := range all {
		if strings.HasSuffix(p, "?") {
			p = strings.TrimSuffix(p, "?")
		} else {
			if i != m.minArgs {
				panic(fmt.Sprintf("rpc: subscription %s/%s: required parameter %q after an optional one", ns, kind, p))
			}
			m.minArgs++
		}
		m.params = append(m.params, p)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	key := ns + "/" + kind
	if _, exists := d.subs[key]; exists {
		panic(fmt.Sprintf("rpc: subscription %s registered twice", key))
	}
	d.subs[key] = &subscription{method: m}
}

// subscription is a subscription kind registered with AddSubscription.
type subscription struct {
	method *method
}

// Subscribe serves an eth_subscribe request: it finds the kind named by the
// first parameter, counts the parameters and calls the handler, which installs
// its feeds before it returns the id. A namespace that is not allowed on the
// transport has no kinds.
func (d *Dispatcher) Subscribe(ctx context.Context, ns string, allowed bool, params json.RawMessage) (json.RawMessage, *Error) {
	label := ns + "_subscribe"
	if !d.admit(&d.inflight) {
		return nil, NewError(CodeDefault, "server is shutting down")
	}
	defer d.inflight.Done()

	raw, err := SplitParams(params)
	if err != nil || (len(params) == 0 || string(params) == "null") {
		return nil, InvalidParams("non-array args")
	}
	var kind string
	if len(raw) == 0 || json.Unmarshal(raw[0], &kind) != nil {
		return nil, InvalidParams("expected subscription name as first argument")
	}
	d.mu.RLock()
	sub := d.subs[ns+"/"+kind]
	d.mu.RUnlock()
	// A namespace that is not enabled on the transport has no subscriptions.
	if sub == nil || !allowed {
		return nil, NewError(CodeMethodNotFound, "no %q subscription in %s namespace", kind, ns)
	}
	args, perr := sub.method.bind(params)
	if perr != nil {
		d.metrics.Requests.WithLabelValues(label, ResultInvalidParams).Inc()
		return nil, perr
	}
	ctx = context.WithValue(ctx, namespaceKey{}, ns)
	ctx = context.WithValue(ctx, methodKey{}, label)
	result, herr := d.run(ctx, sub.method, args)
	if herr != nil {
		res := ResultError
		if isPanic(herr) {
			res = ResultPanic
		}
		d.metrics.Requests.WithLabelValues(label, res).Inc()
		return nil, panicToError(herr)
	}
	encoded, merr := json.Marshal(result)
	if merr != nil {
		d.metrics.Requests.WithLabelValues(label, ResultError).Inc()
		return nil, NewError(CodeInternal, "%v", merr)
	}
	d.metrics.Requests.WithLabelValues(label, ResultOK).Inc()
	return encoded, nil
}

// AddSub0 registers a subscription kind without parameters.
func AddSub0(d *Dispatcher, ns, kind string, h func(ctx context.Context) (string, error)) {
	d.AddSubscription(ns, kind, func(ctx context.Context, args Args) (any, error) {
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx)
	})
}

// AddSub1 registers a subscription kind with one parameter after the kind.
func AddSub1[A any](d *Dispatcher, ns, kind string, h func(ctx context.Context, a A) (string, error), param string) {
	d.AddSubscription(ns, kind, func(ctx context.Context, args Args) (any, error) {
		a, err := decode[A](args, 1)
		if err != nil {
			return nil, err
		}
		if err := args.Check(); err != nil {
			return nil, err
		}
		return h(ctx, a)
	}, param)
}

// SubscriptionKinds returns the registered subscription kinds as
// "namespace/kind".
func (d *Dispatcher) SubscriptionKinds() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	kinds := make([]string, 0, len(d.subs))
	for key := range d.subs {
		kinds = append(kinds, key)
	}
	return kinds
}
