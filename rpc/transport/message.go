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

// Package transport serves JSON-RPC 2.0 over HTTP, WebSocket and a unix
// socket, with the envelope, batches, limits and security checks of
// go-ethereum's server, and routes every call through a Router.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

const (
	version             = "2.0"
	maxMethodNameLength = 2048
	subscribeSuffix     = "_subscribe"
	unsubscribeSuffix   = "_unsubscribe"
)

var null = json.RawMessage("null")

// message is a JSON-RPC request, notification or response.
type message struct {
	Version    string          `json:"jsonrpc,omitempty"`
	ID         json.RawMessage `json:"id,omitempty"`
	Method     string          `json:"method,omitempty"`
	Parameters json.RawMessage `json:"params,omitempty"`
	Error      *errorObject    `json:"error,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
}

type errorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (incoming *message) hasValidVersion() bool {
	return incoming.Version == version
}

func (incoming *message) hasValidID() bool {
	return len(incoming.ID) > 0 && incoming.ID[0] != '{' && incoming.ID[0] != '['
}

func (incoming *message) isNotification() bool {
	return incoming.hasValidVersion() && incoming.ID == nil && incoming.Method != ""
}

func (incoming *message) isCall() bool {
	return incoming.hasValidVersion() && incoming.hasValidID() && incoming.Method != ""
}

func (incoming *message) isResponse() bool {
	return incoming.hasValidVersion() && incoming.hasValidID() && incoming.Method == "" && incoming.Parameters == nil &&
		(incoming.Result != nil || incoming.Error != nil)
}

// errorResponse answers the message with an error, under its id.
func (incoming *message) errorResponse(err *dispatch.Error) *message {
	response := errorMessage(err)
	response.ID = incoming.ID
	return response
}

// errorMessage is an error response with a null id.
func errorMessage(err *dispatch.Error) *message {
	return &message{
		Version: version,
		ID:      null,
		Error:   &errorObject{Code: err.Code, Message: err.Message, Data: err.Data},
	}
}

func invalidRequest(text string) *dispatch.Error {
	return &dispatch.Error{Code: dispatch.CodeInvalidRequest, Message: text}
}

// parseMessages splits the raw input into its messages. Like go-ethereum, it
// keeps what decodes of a malformed message, which then answers "invalid
// request", and treats JSON null as an empty message.
func parseMessages(raw json.RawMessage) ([]*message, bool) {
	if !isBatch(raw) {
		single := new(message)
		_ = json.Unmarshal(raw, single)
		return []*message{single}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_, _ = decoder.Token()
	var messages []*message
	for decoder.More() {
		element := new(message)
		_ = decoder.Decode(element)
		messages = append(messages, element)
	}
	return messages, true
}

func isBatch(raw json.RawMessage) bool {
	for _, character := range raw {
		if character == 0x20 || character == 0x09 || character == 0x0a || character == 0x0d {
			continue
		}
		return character == '['
	}
	return false
}

// readErrorText is the text a failure to read a message is answered with,
// empty for the end of the input.
func readErrorText(err error) string {
	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return "read timeout"
		}
		return "read error"
	}
	if errors.Is(err, io.EOF) {
		return ""
	}
	return "parse error"
}

// Router serves the calls a transport receives. The transport names itself
// so that the router serves only what is enabled on it.
type Router interface {
	Route(ctx context.Context, transport string, method string, parameters json.RawMessage) (json.RawMessage, *dispatch.Error)
}

// connection is what the calls of one request or stream share: the router,
// the transport name and the batch limits.
type connection struct {
	router    Router
	transport string
	limits    BatchLimits
}

// BatchLimits are the limits of a batch: the number of its calls and the
// size of their results together, 0 for no limit.
type BatchLimits struct {
	RequestLimit    int
	ResponseMaxSize int
}

// answerCall serves one message of a request and returns its response, nil
// for a notification or a response.
func (served *connection) answerCall(ctx context.Context, incoming *message) *message {
	switch {
	case incoming.isNotification():
		served.call(ctx, incoming)
		return nil
	case incoming.isCall():
		return served.call(ctx, incoming)
	case incoming.hasValidID():
		return incoming.errorResponse(invalidRequest("invalid request"))
	}
	return errorMessage(invalidRequest("invalid request"))
}

func (served *connection) call(ctx context.Context, incoming *message) *message {
	isSubscription := strings.HasSuffix(incoming.Method, subscribeSuffix) || strings.HasSuffix(incoming.Method, unsubscribeSuffix)
	if !isSubscription && len(incoming.Method) > maxMethodNameLength {
		return incoming.errorResponse(invalidRequest(fmt.Sprintf("method name too long: %d > %d", len(incoming.Method), maxMethodNameLength)))
	}
	result, err := served.router.Route(ctx, served.transport, incoming.Method, incoming.Parameters)
	if err != nil {
		return incoming.errorResponse(err)
	}
	return &message{Version: version, ID: incoming.ID, Result: result}
}

// batch collects the responses of the calls of a batch. The calls are
// answered in order; when the write timeout or the response limit ends the
// batch, every call without an answer, the running one included, is answered
// with the error, and the responses so far go out with them.
type batch struct {
	mutex     sync.Mutex
	calls     []*message
	responses []*message
	written   bool
}

func (pending *batch) next() *message {
	pending.mutex.Lock()
	defer pending.mutex.Unlock()
	if len(pending.calls) == 0 {
		return nil
	}
	return pending.calls[0]
}

func (pending *batch) answered(response *message) {
	pending.mutex.Lock()
	defer pending.mutex.Unlock()
	if response != nil {
		pending.responses = append(pending.responses, response)
	}
	pending.calls = pending.calls[1:]
}

// finish writes the responses once, answering the remaining calls with err
// when it is not nil.
func (pending *batch) finish(write func(responses []*message, isError bool), err *dispatch.Error) {
	pending.mutex.Lock()
	defer pending.mutex.Unlock()
	if pending.written {
		return
	}
	pending.written = true
	if err != nil {
		for _, call := range pending.calls {
			if !call.isNotification() {
				pending.responses = append(pending.responses, call.errorResponse(err))
			}
		}
	}
	if len(pending.responses) > 0 {
		write(pending.responses, err != nil)
	}
}

// writer writes one response or a batch of responses; an error response
// written at the deadline is flushed at once.
type writer func(response any, isError bool)

// serveBatch answers the messages of a batch. writeTimeout, when not zero,
// is when the calls still unanswered are answered with ErrTimeout.
func (served *connection) serveBatch(ctx context.Context, messages []*message, writeTimeout time.Duration, write writer) {
	if len(messages) == 0 {
		write(errorMessage(invalidRequest("empty batch")), true)
		return
	}
	if served.limits.RequestLimit != 0 && len(messages) > served.limits.RequestLimit {
		write([]*message{batchTooLarge(messages)}, true)
		return
	}
	calls := make([]*message, 0, len(messages))
	for _, incoming := range messages {
		if !incoming.isResponse() {
			calls = append(calls, incoming)
		}
	}
	if len(calls) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pending := &batch{calls: calls}
	writeAll := func(responses []*message, isError bool) { write(responses, isError) }
	stopTimer := afterTimeout(writeTimeout, func() {
		cancel()
		pending.finish(writeAll, dispatch.ErrTimeout)
	})
	responseBytes := 0
	for ctx.Err() == nil {
		call := pending.next()
		if call == nil {
			break
		}
		response := served.answerCall(ctx, call)
		pending.answered(response)
		if response == nil || served.limits.ResponseMaxSize == 0 {
			continue
		}
		responseBytes += len(response.Result)
		if responseBytes > served.limits.ResponseMaxSize {
			pending.finish(writeAll, dispatch.ErrResponseTooLarge)
			break
		}
	}
	stopTimer()
	pending.finish(writeAll, nil)
}

// afterTimeout runs the function after the timeout, unless the returned stop
// is called first. A zero timeout runs nothing.
func afterTimeout(timeout time.Duration, function func()) (stop func()) {
	if timeout <= 0 {
		return func() {}
	}
	timer := time.AfterFunc(timeout, function)
	return func() { timer.Stop() }
}

// batchTooLarge answers a batch above the limit, under the id of its first
// call, the best a batch-wide error can do.
func batchTooLarge(messages []*message) *message {
	response := errorMessage(invalidRequest("batch too large"))
	for _, incoming := range messages {
		if incoming.isCall() {
			response.ID = incoming.ID
			break
		}
	}
	return response
}

// serveSingle answers one message. When writeTimeout is not zero, a call
// still running then is answered with ErrTimeout, and so is a notification;
// exactly one response is written.
func (served *connection) serveSingle(ctx context.Context, incoming *message, writeTimeout time.Duration, write writer) {
	if incoming.isResponse() {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var once sync.Once
	stopTimer := afterTimeout(writeTimeout, func() {
		cancel()
		once.Do(func() { write(incoming.errorResponse(dispatch.ErrTimeout), true) })
	})
	response := served.answerCall(ctx, incoming)
	stopTimer()
	once.Do(func() {
		if response != nil {
			write(response, false)
		}
	})
}
