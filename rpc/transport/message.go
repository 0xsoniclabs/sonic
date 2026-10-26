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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// The JSON-RPC envelope, as go-ethereum reads and writes it: the version, an
// id that is neither an object nor an array, a method for a request, and a
// result or an error for a response.
type message struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (m *message) hasValidVersion() bool { return m.Version == "2.0" }

func (m *message) hasValidID() bool {
	return len(m.ID) > 0 && m.ID[0] != '{' && m.ID[0] != '['
}

func (m *message) isNotification() bool {
	return m.hasValidVersion() && m.ID == nil && m.Method != ""
}

func (m *message) isCall() bool {
	return m.hasValidVersion() && m.hasValidID() && m.Method != ""
}

func (m *message) isResponse() bool {
	return m.hasValidVersion() && m.hasValidID() && m.Method == "" && m.Params == nil && (m.Result != nil || m.Error != nil)
}

// ignored reports a message a server does not answer: a response, or the
// notification of a subscription, which only a client expects.
func (m *message) ignored() bool {
	return m.isResponse() || (m.isNotification() && strings.HasSuffix(m.Method, "_subscription"))
}

// parseFrame reads the messages of one frame: a single message or a batch. A
// message that does not decode is left empty, as in go-ethereum.
func parseFrame(raw json.RawMessage) ([]*message, bool) {
	if !isBatch(raw) {
		msgs := []*message{{}}
		_ = json.Unmarshal(raw, msgs[0])
		return msgs, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	_, _ = dec.Token() // the '['
	var msgs []*message
	for dec.More() {
		msgs = append(msgs, new(message))
		_ = dec.Decode(msgs[len(msgs)-1])
	}
	return msgs, true
}

func isBatch(raw json.RawMessage) bool {
	for _, c := range raw {
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// Transport-level errors of the envelope.
var (
	errTimeout          = &dispatch.Error{Code: dispatch.CodeTimeout, Message: "request timed out"}
	errResponseTooLarge = &dispatch.Error{Code: dispatch.CodeResponseTooLarge, Message: "response too large"}
	errBatchTooLarge    = &dispatch.Error{Code: dispatch.CodeInvalidRequest, Message: "batch too large"}
	errEmptyBatch       = &dispatch.Error{Code: dispatch.CodeInvalidRequest, Message: "empty batch"}
	errInvalidRequest   = &dispatch.Error{Code: dispatch.CodeInvalidRequest, Message: "invalid request"}
)

var nullID = json.RawMessage("null")

// response is one answer; a nil id is left out, as for a notification.
type response struct {
	id     json.RawMessage
	result json.RawMessage
	err    error
}

// encode writes a response as go-ethereum does: version, id, then the error
// or the result. The result is written as the dispatcher encoded it.
func (r *response) encode() []byte {
	var b bytes.Buffer
	b.WriteString(`{"jsonrpc":"2.0"`)
	if r.id != nil {
		b.WriteString(`,"id":`)
		b.Write(compactID(r.id))
	}
	if r.err != nil {
		b.WriteString(`,"error":`)
		enc, err := json.Marshal(dispatch.ToErrorObject(r.err))
		if err != nil {
			enc, _ = json.Marshal(dispatch.ErrorObject{Code: dispatch.CodeInternal, Message: err.Error()})
		}
		b.Write(enc)
	} else {
		b.WriteString(`,"result":`)
		b.Write(r.result)
	}
	b.WriteByte('}')
	return b.Bytes()
}

// compactID writes an id the way go-ethereum's encoder echoes it: compacted,
// with HTML characters escaped.
func compactID(id json.RawMessage) []byte {
	enc, err := json.Marshal(id)
	if err != nil {
		return nullID
	}
	return enc
}

func encodeBatch(responses []*response) []byte {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, r := range responses {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(r.encode())
	}
	b.WriteByte(']')
	return b.Bytes()
}

// writer sends the answer to one frame. A transport error is an answer of the
// envelope rather than of a method; HTTP writes it unbuffered.
type writer interface {
	write(msg []byte, transportError bool)
}

// frames serves the frames of one endpoint.
type frames struct {
	router Router
	kind   Kind
	limits Limits
}

// serveFrame answers one frame. For a stream, notifier is the request's
// notifier; timeout, when set, is the point at which a call still running is
// answered "request timed out", as go-ethereum does before the HTTP write
// timeout.
func (f *frames) serveFrame(ctx context.Context, raw json.RawMessage, w writer, timeout time.Duration, newNotifier func() *dispatch.RequestNotifier) {
	msgs, batch := parseFrame(raw)
	if batch {
		f.serveBatch(ctx, msgs, w, timeout, newNotifier)
		return
	}
	if msgs[0].ignored() {
		return
	}
	f.serveSingle(ctx, msgs[0], w, timeout, newNotifier)
}

// serveSingle answers one call with exactly one write: the answer, or the
// timeout error if the call is still running at the timeout.
func (f *frames) serveSingle(ctx context.Context, msg *message, w writer, timeout time.Duration, newNotifier func() *dispatch.RequestNotifier) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		once     sync.Once
		timedOut bool
	)
	if timeout > 0 {
		timer := time.AfterFunc(timeout, func() {
			cancel()
			once.Do(func() {
				timedOut = true
				w.write((&response{id: msg.ID, err: errTimeout}).encode(), true)
			})
		})
		defer timer.Stop()
	}
	n := newRequestNotifier(newNotifier)
	answer := f.handle(withNotifier(ctx, n), msg)
	once.Do(func() {
		if answer != nil {
			w.write(answer.encode(), false)
		}
	})
	finish(n, timedOut || answer == nil || answer.err != nil)
}

// batchBuffer holds the calls of a batch not answered yet and the answers so
// far; the batch and its timer write it once.
type batchBuffer struct {
	mu        sync.Mutex
	calls     []*message
	responses []*response
	wrote     bool
}

func (b *batchBuffer) next() *message {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.calls) == 0 {
		return nil
	}
	return b.calls[0]
}

func (b *batchBuffer) push(r *response) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if r != nil {
		b.responses = append(b.responses, r)
	}
	b.calls = b.calls[1:]
}

// fail answers every call not answered yet with err and writes the batch.
func (b *batchBuffer) fail(w writer, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, msg := range b.calls {
		if !msg.isNotification() {
			b.responses = append(b.responses, &response{id: msg.ID, err: err})
		}
	}
	b.writeLocked(w, true)
}

func (b *batchBuffer) write(w writer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.writeLocked(w, false)
}

func (b *batchBuffer) writeLocked(w writer, transportError bool) {
	if b.wrote {
		return
	}
	b.wrote = true
	if len(b.responses) > 0 {
		w.write(encodeBatch(b.responses), transportError)
	}
}

// serveBatch answers the calls of a batch one after the other, within the
// batch limits, and writes the answers once.
func (f *frames) serveBatch(ctx context.Context, msgs []*message, w writer, timeout time.Duration, newNotifier func() *dispatch.RequestNotifier) {
	if len(msgs) == 0 {
		w.write((&response{id: nullID, err: errEmptyBatch}).encode(), true)
		return
	}
	if limit := f.limits.BatchRequestLimit; limit != 0 && len(msgs) > limit {
		r := &response{id: nullID, err: errBatchTooLarge}
		for _, msg := range msgs {
			if msg.isCall() {
				r.id = msg.ID
				break
			}
		}
		w.write(encodeBatch([]*response{r}), true)
		return
	}
	calls := make([]*message, 0, len(msgs))
	for _, msg := range msgs {
		if !msg.ignored() {
			calls = append(calls, msg)
		}
	}
	if len(calls) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := &batchBuffer{calls: calls, responses: make([]*response, 0, len(calls))}
	if timeout > 0 {
		timer := time.AfterFunc(timeout, func() {
			cancel()
			buf.fail(w, errTimeout)
		})
		defer timer.Stop()
	}
	var notifiers []*dispatch.RequestNotifier
	failed := false
	responseBytes := 0
	for ctx.Err() == nil {
		msg := buf.next()
		if msg == nil {
			break
		}
		n := newRequestNotifier(newNotifier)
		if n != nil {
			notifiers = append(notifiers, n)
		}
		r := f.handle(withNotifier(ctx, n), msg)
		buf.push(r)
		if r != nil && r.err == nil && f.limits.BatchResponseMaxSize != 0 {
			responseBytes += len(r.result)
			if responseBytes > f.limits.BatchResponseMaxSize {
				buf.fail(w, errResponseTooLarge)
				failed = true
				break
			}
		}
	}
	buf.write(w)
	for _, n := range notifiers {
		n.Finish(failed || ctx.Err() != nil)
	}
}

// handle answers one message of a frame; nil for a notification.
func (f *frames) handle(ctx context.Context, msg *message) *response {
	switch {
	case msg.isNotification():
		_, _ = f.router.Serve(ctx, f.kind, dispatch.Call{Method: msg.Method, Params: msg.Params})
		return nil
	case msg.isCall():
		result, err := f.router.Serve(ctx, f.kind, dispatch.Call{Method: msg.Method, Params: msg.Params})
		return &response{id: msg.ID, result: result, err: err}
	case msg.hasValidID():
		return &response{id: msg.ID, err: errInvalidRequest}
	}
	return &response{id: nullID, err: errInvalidRequest}
}

func newRequestNotifier(newNotifier func() *dispatch.RequestNotifier) *dispatch.RequestNotifier {
	if newNotifier == nil {
		return nil
	}
	return newNotifier()
}

func withNotifier(ctx context.Context, n *dispatch.RequestNotifier) context.Context {
	if n == nil {
		return ctx
	}
	return dispatch.WithNotifier(ctx, n)
}

func finish(n *dispatch.RequestNotifier, failed bool) {
	if n != nil {
		n.Finish(failed)
	}
}

// readErrorMessage is the answer to a frame that could not be read, as
// go-ethereum words it; nothing is answered at the end of the input.
func readErrorMessage(err error) string {
	var netErr net.Error
	switch {
	case errors.As(err, &netErr) && netErr.Timeout():
		return "read timeout"
	case errors.As(err, &netErr):
		return "read error"
	case err == io.EOF:
		return ""
	}
	return "parse error"
}
