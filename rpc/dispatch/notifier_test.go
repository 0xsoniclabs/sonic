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
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sink records what a connection would write.
type sink struct {
	mu   sync.Mutex
	msgs []string
	fail error
}

func (s *sink) Send(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return s.fail
	}
	data, _ := json.Marshal(v)
	s.msgs = append(s.msgs, string(data))
	return nil
}

func (s *sink) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.msgs...)
}

func newNotifier(t *testing.T, maxSubs int) (*Dispatcher, *Notifier, *sink, context.Context) {
	t.Helper()
	d := newTestDispatcher(t, DefaultConfig())
	s := &sink{}
	n := d.NewNotifier(context.Background(), s, maxSubs)
	ctx := context.WithValue(WithNotifier(context.Background(), n), namespaceKey{}, "eth")
	return d, n, s, ctx
}

func TestSubscriptionIds_AreUnguessableAndDistinct(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		id := newSubscriptionID()
		require.True(t, strings.HasPrefix(id, "0x"))
		require.False(t, seen[id])
		seen[id] = true
	}
}

func TestNotifier_NothingIsSentBeforeTheIdIsActivated(t *testing.T) {
	_, n, s, ctx := newNotifier(t, 0)
	sub, err := n.NewSubscription(ctx, "newHeads")
	require.NoError(t, err)

	sent := make(chan error, 1)
	go func() { sent <- sub.Notify("early") }()
	time.Sleep(50 * time.Millisecond)
	require.Empty(t, s.all())

	n.Activate(sub.ID())
	require.NoError(t, <-sent)
	require.Len(t, s.all(), 1)
	require.JSONEq(t, `{"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"`+sub.ID()+`","result":"early"}}`, s.all()[0])
}

func TestNotifier_CapsSubscriptionsAndFreesTheSlot(t *testing.T) {
	_, n, _, ctx := newNotifier(t, 2)
	a, err := n.NewSubscription(ctx, "newHeads")
	require.NoError(t, err)
	_, err = n.NewSubscription(ctx, "logs")
	require.NoError(t, err)
	_, err = n.NewSubscription(ctx, "logs")
	require.ErrorIs(t, err, ErrTooManySubscriptions)

	require.True(t, n.Unsubscribe(a.ID()))
	require.False(t, n.Unsubscribe(a.ID()))
	_, err = n.NewSubscription(ctx, "logs")
	require.NoError(t, err)
}

func TestNotifier_UnsubscribeEndsTheSubscription(t *testing.T) {
	_, n, _, ctx := newNotifier(t, 0)
	sub, err := n.NewSubscription(ctx, "newHeads")
	require.NoError(t, err)
	n.Unsubscribe(sub.ID())
	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatal("subscription not ended")
	}
	require.Error(t, sub.Notify("late"))
}

func TestNotifier_ASinkFailureFailsTheConnection(t *testing.T) {
	_, n, s, ctx := newNotifier(t, 0)
	sub, err := n.NewSubscription(ctx, "newHeads")
	require.NoError(t, err)
	n.Activate(sub.ID())
	s.fail = errors.New("queue full")
	require.Error(t, sub.Notify("x"))
	select {
	case <-n.Failed():
		require.ErrorContains(t, n.Cause(), "queue full")
	case <-time.After(time.Second):
		t.Fatal("connection did not fail")
	}
}

func TestNotifier_CloseCancelsBeforeItWaits(t *testing.T) {
	d, n, _, ctx := newNotifier(t, 0)
	sub, err := n.NewSubscription(ctx, "newHeads")
	require.NoError(t, err)
	started := make(chan struct{})
	require.True(t, n.Go(func() {
		close(started)
		<-sub.Done() // a producer that only ends when the subscription is cancelled
	}))
	<-started

	closed := make(chan struct{})
	go func() { n.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close waited for a producer that waits for the cancellation")
	}
	_, err = n.NewSubscription(ctx, "x")
	require.Error(t, err)
	_ = d
}

func TestNotifier_APanickingProducerFailsTheConnection(t *testing.T) {
	d, n, _, _ := newNotifier(t, 0)
	require.True(t, n.Go(func() { panic("producer") }))
	select {
	case <-n.Failed():
	case <-time.After(time.Second):
		t.Fatal("connection did not fail")
	}
	require.Eventually(t, func() bool {
		return counter(d.metrics.Panics.WithLabelValues("task:subscription")) == 1
	}, time.Second, time.Millisecond)
	n.Close()
}

func TestSubscribe_Dispatch(t *testing.T) {
	d, n, _, _ := newNotifier(t, 0)
	AddSub0(d, "eth", "newHeads", func(ctx context.Context) (string, error) {
		sub, err := mustNotifier(ctx).NewSubscription(ctx, "newHeads")
		if err != nil {
			return "", err
		}
		return sub.ID(), nil
	})
	AddSub1(d, "eth", "logs", func(ctx context.Context, crit map[string]any) (string, error) {
		sub, err := mustNotifier(ctx).NewSubscription(ctx, "logs")
		if err != nil {
			return "", err
		}
		return sub.ID(), nil
	}, "criteria")
	ctx := WithNotifier(context.Background(), n)

	res, err := d.Subscribe(ctx, "eth", true, json.RawMessage(`["newHeads"]`))
	require.Nil(t, err)
	var id string
	require.NoError(t, json.Unmarshal(res, &id))
	require.True(t, n.Unsubscribe(id))

	_, err = d.Subscribe(ctx, "eth", true, json.RawMessage(`["logs",{"a":1}]`))
	require.Nil(t, err)

	for params, want := range map[string]string{
		`["nope"]`:           `no "nope" subscription in eth namespace`,
		``:                   `non-array args`,
		`null`:               `non-array args`,
		`{}`:                 `non-array args`,
		`[]`:                 `expected subscription name as first argument`,
		`[5]`:                `expected subscription name as first argument`,
		`["logs"]`:           `missing value for required argument 1`,
		`["newHeads","x"]`:   `too many arguments, want at most 1`,
		`["logs","bad"]`:     `parameter 1 (criteria)`,
		`["logs",{},"more"]`: `too many arguments, want at most 2`,
	} {
		var raw json.RawMessage
		if params != "" {
			raw = json.RawMessage(params)
		}
		_, err := d.Subscribe(ctx, "eth", true, raw)
		require.NotNil(t, err, params)
		require.Contains(t, err.Message, want, params)
	}
	// A namespace that is not allowed has no subscriptions.
	_, err = d.Subscribe(ctx, "eth", false, json.RawMessage(`["newHeads"]`))
	require.Equal(t, `no "newHeads" subscription in eth namespace`, err.Message)
	// Other namespaces have none either.
	_, err = d.Subscribe(ctx, "net", true, json.RawMessage(`["newHeads"]`))
	require.Equal(t, CodeMethodNotFound, err.Code)
}

func mustNotifier(ctx context.Context) *Notifier {
	n, _ := NotifierFrom(ctx)
	return n
}

func TestHub_DropsSlowConsumersWithoutWaiting(t *testing.T) {
	h := NewHub[int]()
	slow, fast := h.Subscribe(1), h.Subscribe(100)
	for i := range 10 {
		h.Publish(i)
	}
	require.True(t, slow.Dropped())
	require.False(t, fast.Dropped())
	require.Equal(t, 1, h.Len())
	var got []int
	for len(fast.Events()) > 0 {
		got = append(got, <-fast.Events())
	}
	require.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, got)
	fast.Close()
	fast.Close() // idempotent
	require.Zero(t, h.Len())
}
