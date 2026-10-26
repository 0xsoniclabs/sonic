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
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// recordingConnection is a stream connection that records what is written
// to it and why it was closed.
type recordingConnection struct {
	mutex   sync.Mutex
	written []string
	reasons []error
}

func (connection *recordingConnection) write(_ context.Context, message any) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	connection.written = append(connection.written, string(encoded))
	return nil
}

func (connection *recordingConnection) disconnect(reason error) {
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	connection.reasons = append(connection.reasons, reason)
}

func (connection *recordingConnection) messages() []string {
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	return append([]string(nil), connection.written...)
}

func (connection *recordingConnection) closeReasons() []error {
	connection.mutex.Lock()
	defer connection.mutex.Unlock()
	return append([]error(nil), connection.reasons...)
}

func newTestNotifier(t *testing.T, limit int) (testDispatcher, *Notifier, *recordingConnection) {
	t.Helper()
	dispatcher := newTestDispatcher(t)
	connection := &recordingConnection{}
	notifier := NewNotifier(context.Background(), dispatcher.Dispatcher, connection.write, connection.disconnect, limit)
	t.Cleanup(notifier.Close)
	return dispatcher, notifier, connection
}

func TestNotifier_NotificationsWaitForTheResponseWithTheID(t *testing.T) {
	_, notifier, connection := newTestNotifier(t, 10)
	call := notifier.NewCall()
	subscription, err := call.Subscribe("eth", "newHeads")
	require.NoError(t, err)
	sent := make(chan error, 1)
	subscription.Go(func(ctx context.Context) {
		sent <- subscription.Notify(ctx, "first")
		<-ctx.Done()
	})
	select {
	case <-sent:
		t.Fatal("a notification went out before the response with the subscription id")
	case <-time.After(50 * time.Millisecond):
	}
	call.Activate()
	require.NoError(t, <-sent)
	require.Equal(t, []string{
		`{"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"` + subscription.ID + `","result":"first"}}`,
	}, connection.messages())
}

func TestNotifier_RefusesSubscriptionsAboveTheLimit(t *testing.T) {
	dispatcher, notifier, _ := newTestNotifier(t, 2)
	call := notifier.NewCall()
	_, err := call.Subscribe("eth", "logs")
	require.NoError(t, err)
	_, err = call.Subscribe("eth", "logs")
	require.NoError(t, err)
	_, err = call.Subscribe("eth", "logs")
	require.ErrorIs(t, err, ErrTooManySubscriptions)
	require.Equal(t, 2.0, dispatcher.scrape(t, "rpc_subscriptions", "kind", "logs"))
}

func TestNotifier_UnsubscribeEndsTheProducer(t *testing.T) {
	dispatcher, notifier, _ := newTestNotifier(t, 10)
	call := notifier.NewCall()
	subscription, err := call.Subscribe("eth", "newHeads")
	require.NoError(t, err)
	ended := make(chan struct{})
	subscription.Go(func(ctx context.Context) {
		<-ctx.Done()
		close(ended)
	})
	require.True(t, call.Unsubscribe(subscription.ID))
	require.False(t, call.Unsubscribe(subscription.ID), "an id is unsubscribed once")
	<-ended
	require.Error(t, subscription.Notify(context.Background(), "late"))
	require.Zero(t, dispatcher.scrape(t, "rpc_subscriptions", "kind", "newHeads"))
}

func TestNotifier_IDsBelongToTheirConnection(t *testing.T) {
	_, first, _ := newTestNotifier(t, 10)
	_, second, _ := newTestNotifier(t, 10)
	subscription, err := first.NewCall().Subscribe("eth", "newHeads")
	require.NoError(t, err)
	require.False(t, second.Unsubscribe(subscription.ID))
	require.True(t, first.Unsubscribe(subscription.ID))
}

func TestNotifier_CloseCancelsThenWaitsForTheProducers(t *testing.T) {
	_, notifier, _ := newTestNotifier(t, 10)
	call := notifier.NewCall()
	var finished atomic.Int32
	for range 3 {
		subscription, err := call.Subscribe("eth", "newHeads")
		require.NoError(t, err)
		subscription.Go(func(ctx context.Context) {
			<-ctx.Done()
			time.Sleep(10 * time.Millisecond)
			finished.Add(1)
		})
	}
	notifier.Close()
	require.Equal(t, int32(3), finished.Load(), "Close returned before its producers ended")
	_, err := call.Subscribe("eth", "newHeads")
	require.Error(t, err, "a closed connection takes no subscription")
}

func TestNotifier_AFailedSubscriptionClosesItsConnectionWithTheReason(t *testing.T) {
	_, notifier, connection := newTestNotifier(t, 10)
	subscription, err := notifier.NewCall().Subscribe("eth", "logs")
	require.NoError(t, err)
	lost := errors.New("events were lost")
	subscription.Fail(lost)
	reasons := connection.closeReasons()
	require.Len(t, reasons, 1)
	require.ErrorIs(t, reasons[0], lost)
	require.Contains(t, reasons[0].Error(), subscription.ID)
	require.False(t, notifier.Unsubscribe(subscription.ID))
}

func TestNotifier_APanickingProducerEndsTheConnection(t *testing.T) {
	dispatcher, notifier, connection := newTestNotifier(t, 10)
	subscription, err := notifier.NewCall().Subscribe("eth", "logs")
	require.NoError(t, err)
	subscription.Go(func(context.Context) { panic("producer") })
	eventually(t, func() bool { return len(connection.closeReasons()) == 1 }, "the connection to close")
	require.ErrorIs(t, connection.closeReasons()[0], ErrPanic)
	require.Equal(t, 1.0, dispatcher.scrape(t, "rpc_panics_total", "method", "logs"))
}

func TestNotifier_AProducerAfterStopFailsItsSubscription(t *testing.T) {
	dispatcher, notifier, connection := newTestNotifier(t, 10)
	subscription, err := notifier.NewCall().Subscribe("eth", "logs")
	require.NoError(t, err)
	dispatcher.Stop()
	subscription.Go(func(context.Context) { t.Error("a producer ran after Stop") })
	require.Len(t, connection.closeReasons(), 1)
	require.ErrorIs(t, connection.closeReasons()[0], ErrStopping)
}

func TestCallFrom_FindsTheCallOnTheContext(t *testing.T) {
	_, notifier, _ := newTestNotifier(t, 10)
	_, found := CallFrom(context.Background())
	require.False(t, found)
	call := notifier.NewCall()
	got, found := CallFrom(WithCall(context.Background(), call))
	require.True(t, found)
	require.Same(t, call, got)
}

func TestNewID_IsAQuantityOf128RandomBits(t *testing.T) {
	format := regexp.MustCompile(`^0x[1-9a-f][0-9a-f]{0,31}$`)
	seen := map[string]bool{}
	for range 1000 {
		identifier := NewID()
		require.Regexp(t, format, identifier)
		require.False(t, seen[identifier])
		seen[identifier] = true
	}
}
