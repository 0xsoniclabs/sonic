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

// Defaults of the stream transports.
const (
	// DefaultQueueSize is the number of messages a connection may have waiting
	// to be written. A client that falls further behind is disconnected.
	DefaultQueueSize = 4096
	// DefaultMaxSubscriptions is the number of subscriptions one connection may
	// hold.
	DefaultMaxSubscriptions = 1024
	// writeTimeout bounds the write of one message to a client.
	writeTimeout = 10 * time.Second
)

// codec reads and writes the messages of one stream connection.
type codec interface {
	// readMessage returns the next JSON value of the stream. A syntax error
	// is not an error of the codec: it is returned as a parse error.
	readMessage() (json.RawMessage, error)
	// writeMessage writes one message, bounded by the deadline.
	writeMessage(data []byte, deadline time.Time) error
	// close closes the underlying connection.
	close() error
	// remote describes the peer.
	remote() string
}

// pinger is implemented by codecs of transports with keep-alive.
type pinger interface {
	codec
	// idleInterval is how long a connection may be idle before a ping.
	idleInterval() time.Duration
	ping() error
}

// StreamConfig configures the connections of a stream transport.
type StreamConfig struct {
	EndpointConfig
	// MaxSubscriptions bounds the subscriptions of one connection.
	MaxSubscriptions int
	// QueueSize bounds the messages waiting to be written to one client.
	QueueSize int
}

// streamServer serves the connections of a stream transport.
type streamServer struct {
	cfg      StreamConfig
	d        *dispatch.Dispatcher
	endpoint *Endpoint
}

func newStreamServer(cfg StreamConfig, front *Front) *streamServer {
	cfg.Notifications = true
	if cfg.MaxSubscriptions == 0 {
		cfg.MaxSubscriptions = DefaultMaxSubscriptions
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = DefaultQueueSize
	}
	d := front.Dispatcher()
	return &streamServer{
		cfg:      cfg,
		d:        d,
		endpoint: NewEndpoint(cfg.EndpointConfig, front),
	}
}

// errQueueFull is the failure of a connection whose client does not read.
var errQueueFull = errors.New("client does not read its messages: output queue is full")

// connection is one client of a stream transport.
type connection struct {
	s        *streamServer
	c        codec
	queue    chan []byte
	notifier *dispatch.Notifier
	ctx      context.Context
	cancel   context.CancelCauseFunc

	calls sync.WaitGroup // calls being served
	done  chan struct{}  // closed when the writer ended

	qmu     sync.RWMutex // guards closing the queue against senders
	qclosed bool
}

// closeQueue ends the writer after it wrote what is queued.
func (conn *connection) closeQueue() {
	conn.qmu.Lock()
	defer conn.qmu.Unlock()
	if !conn.qclosed {
		conn.qclosed = true
		close(conn.queue)
	}
}

// serve serves a connection until it ends and releases everything it owns.
// It returns after every goroutine of the connection has returned.
func (s *streamServer) serve(parent context.Context, c codec) {
	metrics := s.d.Metrics().Connections.WithLabelValues(s.cfg.Name)
	metrics.Inc()
	defer metrics.Dec()

	ctx, cancel := context.WithCancelCause(parent)
	conn := &connection{
		s:      s,
		c:      c,
		queue:  make(chan []byte, s.cfg.QueueSize),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	conn.notifier = s.d.NewNotifier(ctx, conn, s.cfg.MaxSubscriptions)
	callCtx := dispatch.WithNotifier(ctx, conn.notifier)

	if !s.d.Go("rpc-writer", func(context.Context) { conn.writeLoop() }) {
		_ = c.close()
		cancel(nil)
		return
	}

	// The connection fails when a subscription fails it or the module stops: the
	// codec is closed to unblock the reader. A normal end does not close it
	// here: the writer still has to flush what is queued.
	ended := make(chan struct{})
	s.d.Go("rpc-watch", func(context.Context) {
		select {
		case <-conn.notifier.Failed():
			select {
			case <-ended:
				return
			default:
			}
			conn.cancel(conn.notifier.Cause())
			_ = c.close()
		case <-parent.Done():
			_ = c.close()
		case <-ended:
		}
	})

	conn.readLoop(callCtx)

	// No more requests come in. Those in flight finish and are answered, bounded
	// by their own deadlines; a failed connection cancels them. Then the
	// subscriptions end: cancel first, then wait for the producers. The queue is
	// flushed before the codec closes.
	close(ended)
	conn.calls.Wait()
	cancel(errors.New("connection closed"))
	conn.notifier.Close()
	conn.closeQueue()
	<-conn.done
	_ = c.close()
}

// readLoop reads messages and serves each in its own call.
func (conn *connection) readLoop(ctx context.Context) {
	for {
		raw, err := conn.c.readMessage()
		if err != nil {
			if ctx.Err() == nil {
				if msg := messageForReadError(err); msg != "" {
					_ = conn.write(ctx, errorMessage(dispatch.NewError(dispatch.CodeParse, "%s", msg)), true)
				}
			}
			return
		}
		msgs, batch := parseMessages(raw)
		conn.calls.Add(1)
		if !conn.s.d.Go("rpc-call", func(context.Context) {
			defer conn.calls.Done()
			if batch {
				conn.s.endpoint.handleBatch(ctx, msgs, conn)
			} else {
				conn.s.endpoint.handleMsg(ctx, msgs[0], conn)
			}
		}) {
			conn.calls.Done()
			return
		}
	}
}

// write queues a response, waiting for room in the queue.
func (conn *connection) write(ctx context.Context, v any, _ bool) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	conn.qmu.RLock()
	defer conn.qmu.RUnlock()
	if conn.qclosed {
		return errors.New("connection closed")
	}
	select {
	case conn.queue <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Send queues a notification without waiting: a client that does not keep up
// fails the connection.
func (conn *connection) Send(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	conn.qmu.RLock()
	defer conn.qmu.RUnlock()
	if conn.qclosed {
		return errors.New("connection closed")
	}
	select {
	case conn.queue <- data:
		return nil
	default:
		return errQueueFull
	}
}

// writeLoop writes the queued messages to the client until the queue is
// closed. A write that fails ends the connection.
func (conn *connection) writeLoop() {
	defer close(conn.done)
	var idle <-chan time.Time
	var p pinger
	if pc, ok := conn.c.(pinger); ok {
		p = pc
		timer := time.NewTicker(pc.idleInterval())
		defer timer.Stop()
		idle = timer.C
	}
	failed := false
	for {
		select {
		case data, ok := <-conn.queue:
			if !ok {
				return
			}
			if failed {
				continue // drain what is left so no producer waits on the queue
			}
			if err := conn.c.writeMessage(data, time.Now().Add(writeTimeout)); err != nil {
				log.Debug("RPC connection write failed", "remote", conn.c.remote(), "err", err)
				failed = true
				conn.cancel(err)
				_ = conn.c.close()
			}
		case <-idle:
			if p != nil && !failed {
				if err := p.ping(); err != nil {
					failed = true
					conn.cancel(err)
					_ = conn.c.close()
				}
			}
		}
	}
}
