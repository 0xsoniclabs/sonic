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

package dispatch_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// connection records what a notifier sends and why it ends.
type connection struct {
	mutex  sync.Mutex
	sent   []any
	reason error
	fail   error
}

func (c *connection) send(message any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.fail != nil {
		return c.fail
	}
	c.sent = append(c.sent, message)
	return nil
}

func (c *connection) end(reason error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.reason = reason
}

func (c *connection) sentCount() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return len(c.sent)
}

type countParams struct {
	Count int
}

// newSubscribing registers a kind that notifies the numbers up to its count
// and a kind whose installation fails or panics.
func newSubscribing(t *testing.T) (*dispatch.Dispatcher, *prometheus.Registry) {
	t.Helper()
	registry := prometheus.NewRegistry()
	dispatcher := dispatch.New(dispatch.DefaultConfig(), registry)
	t.Cleanup(dispatcher.Stop)
	dispatch.AddSubscription(dispatcher, "test", "count", func(_ context.Context, subscription *dispatch.Subscription, params countParams) error {
		subscription.Go(func(ctx context.Context) {
			for i := range params.Count {
				if subscription.Notify(i) != nil {
					return
				}
			}
			<-ctx.Done()
		})
		return nil
	})
	dispatch.AddSubscription(dispatcher, "test", "failing", func(context.Context, *dispatch.Subscription, struct{}) error {
		return errors.New("refused")
	})
	dispatch.AddSubscription(dispatcher, "test", "panicking", func(context.Context, *dispatch.Subscription, struct{}) error {
		panic("boom")
	})
	dispatch.AddSubscription(dispatcher, "test", "dropping", func(_ context.Context, subscription *dispatch.Subscription, _ struct{}) error {
		subscription.Go(func(context.Context) { subscription.Drop(errors.New("lost events")) })
		return nil
	})
	return dispatcher, registry
}

func subscribe(t *testing.T, dispatcher *dispatch.Dispatcher, notifier *dispatch.Notifier, params string) (dispatch.Response, *dispatch.Subscription) {
	t.Helper()
	return dispatcher.Subscribe(context.Background(), notifier, "test_subscribe", json.RawMessage(params))
}

// Notifications wait until the subscription's id is answered, and stop when
// it ends.
func TestNotifier_SendsOnceActivatedAndUntilUnsubscribed(t *testing.T) {
	dispatcher, registry := newSubscribing(t)
	out := &connection{}
	notifier := dispatch.NewNotifier(context.Background(), dispatcher, out.send, out.end)
	response, subscription := subscribe(t, dispatcher, notifier, `["count",3]`)
	require.Nil(t, response.Error)
	require.Equal(t, `"`+string(subscription.ID)+`"`, string(response.Result))
	require.Equal(t, 1.0, liveSubscriptions(t, registry, "count"))
	time.Sleep(10 * time.Millisecond)
	require.Zero(t, out.sentCount(), "nothing before the id is answered")
	subscription.Activate()
	require.Eventually(t, func() bool { return out.sentCount() == 3 }, time.Second, time.Millisecond)
	encoded, err := json.Marshal(out.sent[0])
	require.NoError(t, err)
	require.JSONEq(t, `{"jsonrpc":"2.0","method":"test_subscription","params":{"subscription":"`+string(subscription.ID)+`","result":0}}`, string(encoded))

	require.True(t, notifier.Unsubscribe(subscription.ID))
	require.False(t, notifier.Unsubscribe(subscription.ID))
	require.ErrorIs(t, subscription.Notify(4), context.Canceled)
	require.Zero(t, liveSubscriptions(t, registry, "count"))
	notifier.Close()
}

// liveSubscriptions reads the gauge of the live subscriptions of a kind.
func liveSubscriptions(t *testing.T, registry *prometheus.Registry, kind string) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "rpc_subscriptions" {
			continue
		}
		for _, metric := range family.GetMetric() {
			if metric.GetLabel()[0].GetValue() == kind {
				return metric.GetGauge().GetValue()
			}
		}
	}
	return 0
}

func TestNotifier_RemovesASubscriptionWhoseInstallationFails(t *testing.T) {
	dispatcher, registry := newSubscribing(t)
	notifier := dispatch.NewNotifier(context.Background(), dispatcher, (&connection{}).send, (&connection{}).end)
	defer notifier.Close()
	response, subscription := subscribe(t, dispatcher, notifier, `["failing"]`)
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "refused"}, response.Error)
	require.Nil(t, subscription)
	response, subscription = subscribe(t, dispatcher, notifier, `["panicking"]`)
	require.Equal(t, dispatch.CodeInternal, response.Error.Code)
	require.Nil(t, subscription)
	for _, kind := range []string{"failing", "panicking"} {
		require.Zero(t, liveSubscriptions(t, registry, kind))
	}
}

func TestNotifier_BoundsTheSubscriptionsOfAConnection(t *testing.T) {
	dispatcher, _ := newSubscribing(t)
	notifier := dispatch.NewNotifier(context.Background(), dispatcher, (&connection{}).send, (&connection{}).end)
	defer notifier.Close()
	for range 10_000 {
		response, _ := subscribe(t, dispatcher, notifier, `["count",0]`)
		require.Nil(t, response.Error)
	}
	response, _ := subscribe(t, dispatcher, notifier, `["count",0]`)
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "too many subscriptions on this connection, the limit is 10000"}, response.Error)
}

// A subscription that loses events, and one that fails to send, ends its
// connection with the reason.
func TestNotifier_EndsTheConnectionOfASubscriptionThatFails(t *testing.T) {
	dispatcher, _ := newSubscribing(t)
	out := &connection{}
	notifier := dispatch.NewNotifier(context.Background(), dispatcher, out.send, out.end)
	defer notifier.Close()
	_, subscription := subscribe(t, dispatcher, notifier, `["dropping"]`)
	require.Eventually(t, func() bool {
		out.mutex.Lock()
		defer out.mutex.Unlock()
		return out.reason != nil
	}, time.Second, time.Millisecond)
	require.EqualError(t, out.reason, "subscription "+string(subscription.ID)+" dropped: lost events")

	failing := &connection{fail: errors.New("broken pipe")}
	notifier = dispatch.NewNotifier(context.Background(), dispatcher, failing.send, failing.end)
	defer notifier.Close()
	_, subscription = subscribe(t, dispatcher, notifier, `["count",1]`)
	subscription.Activate()
	require.Eventually(t, func() bool {
		failing.mutex.Lock()
		defer failing.mutex.Unlock()
		return failing.reason != nil
	}, time.Second, time.Millisecond)
	require.EqualError(t, failing.reason, "failed to send a notification of subscription "+string(subscription.ID)+": broken pipe")
}

// Closing a connection ends its subscriptions and waits for their
// producers, and accepts no new subscription.
func TestNotifier_CloseWaitsForTheProducers(t *testing.T) {
	dispatcher, _ := newSubscribing(t)
	notifier := dispatch.NewNotifier(context.Background(), dispatcher, (&connection{}).send, (&connection{}).end)
	_, subscription := subscribe(t, dispatcher, notifier, `["count",0]`)
	notifier.Close()
	require.False(t, subscription.Go(func(context.Context) {}), "no producer once closed")
	response, _ := subscribe(t, dispatcher, notifier, `["count",0]`)
	require.EqualError(t, response.Error, "the connection is closed")
}
