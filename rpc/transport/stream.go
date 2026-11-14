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
	"sync"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// stream is a connection that carries messages both ways, as IPC and
// WebSocket do, so that it can carry notifications.
type stream interface {
	writer
	// read answers the next message. A message that is not JSON is answered
	// by read and ends the connection, as go-ethereum's server does.
	read() (json.RawMessage, error)
	// send writes a notification.
	send(message any) error
	// end ends the connection, telling the client the reason if it can.
	// It does not wait for the connection's work.
	end(reason error)
	// close closes the connection.
	close()
}

// serveStream serves the messages of a stream connection until it ends,
// within the batch limits:
// each in a goroutine of the dispatcher, as their answers may come in any
// order, with the connection's notifier on their context. Ending a
// connection cancels its calls and subscriptions first, then waits for
// them; the reverse order would wait for work that waits to write.
func serveStream(ctx context.Context, dispatcher *dispatch.Dispatcher, front *Front, limits BatchLimits, connection stream) {
	ctx, cancel := context.WithCancel(ctx)
	notifier := dispatch.NewNotifier(ctx, dispatcher, connection.send, connection.end)
	var calls sync.WaitGroup
	defer func() {
		cancel()
		calls.Wait()
		notifier.Close()
		connection.close()
	}()
	ctx = withNotifier(ctx, notifier)
	served := &session{front: front, out: connection, limits: limits}
	for {
		raw, err := connection.read()
		if err != nil {
			return
		}
		messages, batch := parseMessages(raw)
		calls.Add(1)
		started := dispatcher.Go("stream call", func(context.Context) {
			defer calls.Done()
			if batch {
				served.serveBatch(ctx, messages)
			} else {
				served.serveMessage(ctx, messages[0])
			}
		})
		if !started {
			calls.Done()
			return
		}
	}
}
