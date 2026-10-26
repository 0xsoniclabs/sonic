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
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

const (
	// streamWriteTimeout bounds a write to a stream connection, as
	// go-ethereum's codec bounds one without a deadline of its own; a client
	// that does not read in time loses its connection.
	streamWriteTimeout = 10 * time.Second
	// maxSubscriptionsPerConnection bounds the subscriptions of a stream
	// connection (D4; section 6).
	maxSubscriptionsPerConnection = 10_000
)

// codec reads and writes the messages of one stream connection. read is
// called by one reader at a time, write by many.
type codec interface {
	// read returns the next message.
	read() (json.RawMessage, error)
	// write sends a value encoded as JSON by the deadline.
	write(value any, deadline time.Time) error
	// close ends the connection, telling the client why when it can.
	close(reason error)
}

// stream serves a stream connection, WebSocket or unix socket, as
// go-ethereum's server does: the messages are read in order and their
// calls served concurrently, each answered when it is done; a message that
// does not parse is answered with the parse error and ends the connection.
// The subscriptions of the connection notify only after the response that
// carries their id.
type stream struct {
	served     connection
	codec      codec
	dispatcher *dispatch.Dispatcher

	writing sync.Mutex
	closing sync.Once
}

func newStream(router Router, transport string, limits BatchLimits, connection codec, dispatcher *dispatch.Dispatcher) *stream {
	return &stream{
		served:     connectionOf(router, transport, limits),
		codec:      connection,
		dispatcher: dispatcher,
	}
}

func connectionOf(router Router, transport string, limits BatchLimits) connection {
	return connection{router: router, transport: transport, limits: limits}
}

// serve serves the connection until it ends or ctx does. It returns when
// every call and subscription of the connection ended.
func (served *stream) serve(ctx context.Context) {
	connections := served.dispatcher.Metrics().Connections.WithLabelValues(served.served.transport)
	connections.Inc()
	defer connections.Dec()

	// The connection closes when ctx ends, which unblocks the read. Ending
	// it cancels first and then waits for its calls and producers.
	ctx, cancel := context.WithCancel(ctx)
	stopClosing := context.AfterFunc(ctx, func() { served.close(dispatch.ErrStopping) })
	notifier := dispatch.NewNotifier(ctx, served.dispatcher, served.notify, served.close, maxSubscriptionsPerConnection)
	var calls sync.WaitGroup
	defer func() {
		stopClosing()
		cancel()
		notifier.Close()
		calls.Wait()
		served.close(nil)
	}()
	for {
		raw, err := served.codec.read()
		if err != nil {
			served.answerReadError(err)
			return
		}
		messages, isBatch := parseMessages(raw)
		calls.Add(1)
		started := served.dispatcher.Go(ctx, served.served.transport+" call", func(ctx context.Context) {
			defer calls.Done()
			served.answer(ctx, notifier.NewCall(), messages, isBatch)
		})
		if !started {
			calls.Done()
			return
		}
	}
}

// answer serves the messages of one read and then lets the subscriptions
// they created notify.
func (served *stream) answer(ctx context.Context, call *dispatch.Call, messages []*message, isBatch bool) {
	ctx = dispatch.WithCall(ctx, call)
	write := func(response any, _ bool) { served.send(response) }
	if isBatch {
		served.served.serveBatch(ctx, messages, 0, write)
	} else {
		served.served.serveSingle(ctx, messages[0], 0, write)
	}
	call.Activate()
}

// answerReadError answers a message that does not parse, as go-ethereum's
// stream reader does; any other failure to read just ends the connection.
func (served *stream) answerReadError(err error) {
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		served.send(errorMessage(&dispatch.Error{Code: dispatch.CodeParseError, Message: err.Error()}))
	}
}

// send writes one message; a connection whose client does not take it in
// time is closed.
func (served *stream) send(value any) {
	if err := served.write(value); err != nil {
		log.Debug("RPC stream write failed", "transport", served.served.transport, "err", err)
		served.close(err)
	}
}

func (served *stream) write(value any) error {
	served.writing.Lock()
	defer served.writing.Unlock()
	return served.codec.write(value, time.Now().Add(streamWriteTimeout))
}

// notify writes a notification of a subscription.
func (served *stream) notify(ctx context.Context, notification any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := served.write(notification)
	if err != nil {
		served.close(err)
	}
	return err
}

// close ends the connection once.
func (served *stream) close(reason error) {
	served.closing.Do(func() { served.codec.close(reason) })
}
