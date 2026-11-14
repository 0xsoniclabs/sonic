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

package transport

import (
	"context"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// The envelope tests serve the same methods through the module's transport
// and through go-ethereum's server, which is the reference.

// slowDuration is how long the slow method runs, ignoring cancellation as an
// EVM run of an interpreter that never polls does.
const slowDuration = 1500 * time.Millisecond

type codedError struct{}

func (codedError) Error() string  { return "execution reverted" }
func (codedError) ErrorCode() int { return 3 }
func (codedError) ErrorData() any { return "0x01" }

// referenceService is the reference's form of the test methods.
type referenceService struct{}

func (referenceService) Echo(text string) string { return text }

func (referenceService) Add(a, b int) int { return a + b }

func (referenceService) Fail() error { return codedError{} }

func (referenceService) Panic() string { panic("boom") }

func (referenceService) Nothing() *string { return nil }

func (referenceService) Large(size int) string { return string(make([]byte, size)) }

// Ticks notifies the numbers up to the count.
func (referenceService) Ticks(ctx context.Context, count int, label *string) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()
	go func() {
		for tick := range count {
			_ = notifier.Notify(subscription.ID, tick)
		}
	}()
	return subscription, nil
}

func (referenceService) Slow() string {
	time.Sleep(slowDuration)
	return "late"
}

func newReference(t testing.TB) *rpc.Server {
	t.Helper()
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("test", referenceService{}))
	return server
}

type echoParams struct{ Text string }
type addParams struct{ A, B int }
type sizeParams struct{ Size int }
type ticksParams struct {
	Count int
	Label *string
}

// newTestDispatcher serves the test methods and rpc_modules, as the module
// registers it.
func newTestDispatcher(t testing.TB) *dispatch.Dispatcher {
	t.Helper()
	dispatcher := dispatch.New(dispatch.DefaultConfig(), prometheus.NewRegistry())
	dispatch.Add(dispatcher, "test_echo", func(_ context.Context, p echoParams) (string, error) { return p.Text, nil })
	dispatch.Add(dispatcher, "test_add", func(_ context.Context, p addParams) (int, error) { return p.A + p.B, nil })
	dispatch.Add(dispatcher, "test_fail", func(context.Context, struct{}) (any, error) { return nil, codedError{} })
	dispatch.Add(dispatcher, "test_panic", func(context.Context, struct{}) (string, error) { panic("boom") })
	dispatch.Add(dispatcher, "test_nothing", func(context.Context, struct{}) (*string, error) { return nil, nil })
	dispatch.Add(dispatcher, "test_large", func(_ context.Context, p sizeParams) (string, error) { return string(make([]byte, p.Size)), nil })
	dispatch.Add(dispatcher, "test_slow", func(context.Context, struct{}) (string, error) {
		time.Sleep(slowDuration)
		return "late", nil
	})
	dispatch.AddSubscription(dispatcher, "test", "ticks", func(_ context.Context, subscription *dispatch.Subscription, p ticksParams) error {
		subscription.Go(func(context.Context) {
			for tick := range p.Count {
				_ = subscription.Notify(tick)
			}
		})
		return nil
	})
	dispatch.Add(dispatcher, "rpc_modules", func(ctx context.Context, _ struct{}) (map[string]string, error) {
		return Modules(ctx), nil
	})
	t.Cleanup(dispatcher.Stop)
	return dispatcher
}
