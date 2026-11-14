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
	"strings"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// BatchLimits bound a batch: the number of its messages and the bytes of
// its results. Zero means no bound.
type BatchLimits struct {
	Items         int
	ResponseBytes int
}

// writer delivers the answers of a connection or of an HTTP request. An
// error answer may need to reach the client before the server's write
// timeout cuts it off.
type writer interface {
	write(value any, isError bool)
}

// session serves the messages a connection or an HTTP request reads.
type session struct {
	front  *Front
	out    writer
	limits BatchLimits
	// timeout is when a call that has not answered is answered "request
	// timed out", zero for never: on HTTP, just before the server's write
	// timeout.
	timeout time.Duration
}

// isClientMessage tells whether a message is one a client receives, a
// response or a subscription notification, which a server ignores.
func isClientMessage(msg *message) bool {
	return msg.isResponse() || msg.isNotification() && strings.HasSuffix(msg.Method, notificationMethodSuffix)
}

// serveMessage serves one message that is not part of a batch.
func (s *session) serveMessage(ctx context.Context, msg *message) {
	if isClientMessage(msg) {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var responded sync.Once
	if s.timeout > 0 {
		timer := time.AfterFunc(s.timeout, func() {
			cancel()
			responded.Do(func() { s.out.write(msg.errorResponse(errTimeout), true) })
		})
		defer timer.Stop()
	}
	answer := s.serveCall(ctx, msg)
	// Done without an answer too, so that the timer's answer is not written
	// after the request has ended.
	responded.Do(func() {
		if answer != nil {
			s.out.write(answer, false)
			wrote(answer)
		}
	})
}

// serveBatch serves the messages of a batch in order, and answers them in one
// array. When the timeout or the bound on the results is reached, the calls
// not answered yet are answered with the error, together with the answers so
// far.
func (s *session) serveBatch(ctx context.Context, messages []*message) {
	if len(messages) == 0 {
		s.out.write(errorMessage(errEmptyBatch), true)
		return
	}
	if s.limits.Items != 0 && len(messages) > s.limits.Items {
		s.out.write([]*message{batchTooLarge(messages)}, true)
		return
	}
	var calls []*message
	for _, msg := range messages {
		if !isClientMessage(msg) {
			calls = append(calls, msg)
		}
	}
	if len(calls) == 0 {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	buffer := &batchBuffer{calls: calls, answers: make([]*message, 0, len(calls))}
	if s.timeout > 0 {
		timer := time.AfterFunc(s.timeout, func() {
			cancel()
			buffer.answerRest(s.out, errTimeout)
		})
		defer timer.Stop()
	}
	responseBytes := 0
	for ctx.Err() == nil {
		msg := buffer.next()
		if msg == nil {
			break
		}
		answer := s.serveCall(ctx, msg)
		buffer.push(answer)
		if answer == nil || s.limits.ResponseBytes == 0 {
			continue
		}
		if responseBytes += len(answer.Result); responseBytes > s.limits.ResponseBytes {
			buffer.answerRest(s.out, errResponseTooLarge)
			break
		}
	}
	buffer.write(s.out)
}

// batchTooLarge answers a batch over the bound with the id of its first call,
// the best the protocol allows to report an error of a whole batch.
func batchTooLarge(messages []*message) *message {
	answer := errorMessage(errBatchTooLarge)
	for _, msg := range messages {
		if msg.isCall() {
			answer.ID = msg.ID
			break
		}
	}
	return answer
}

// serveCall answers a call, nil for a notification.
func (s *session) serveCall(ctx context.Context, msg *message) *message {
	switch {
	case msg.isNotification():
		s.front.call(ctx, msg)
		return nil
	case msg.isCall():
		return s.front.call(ctx, msg)
	case msg.hasValidID():
		return msg.errorResponse(errInvalidRequest)
	default:
		return errorMessage(errInvalidRequest)
	}
}

// batchBuffer holds the calls of a batch not answered yet and the answers so
// far. The loop serving the batch and its timer share it.
type batchBuffer struct {
	mutex   sync.Mutex
	calls   []*message
	answers []*message
	wrote   bool
}

// next is the call being served, which stays among the unanswered until its
// answer is pushed.
func (b *batchBuffer) next() *message {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.calls) == 0 {
		return nil
	}
	return b.calls[0]
}

func (b *batchBuffer) push(answer *message) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if answer != nil {
		b.answers = append(b.answers, answer)
	}
	b.calls = b.calls[1:]
}

// answerRest answers every call not answered yet with the error and writes
// the batch.
func (b *batchBuffer) answerRest(out writer, err *dispatch.Error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, msg := range b.calls {
		if !msg.isNotification() {
			b.answers = append(b.answers, msg.errorResponse(err))
		}
	}
	b.writeLocked(out, true)
}

func (b *batchBuffer) write(out writer) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.writeLocked(out, false)
}

// writeLocked writes the answers once; the caller holds the lock.
func (b *batchBuffer) writeLocked(out writer, isError bool) {
	if b.wrote {
		return
	}
	b.wrote = true
	if len(b.answers) > 0 {
		out.write(b.answers, isError)
	}
	for _, answer := range b.answers {
		wrote(answer)
	}
}

// wrote tells an answer that it has been written.
func wrote(answer *message) {
	if answer.written != nil {
		answer.written()
	}
}
