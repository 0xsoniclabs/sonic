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

// Package transport carries the JSON-RPC 2.0 messages of the RPC module over
// HTTP and IPC. Its envelope, batches, errors and limits answer as
// go-ethereum's server does, which its tests use as the reference.
package transport

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

const (
	version                  = "2.0"
	subscribeMethodSuffix    = "_subscribe"
	unsubscribeMethodSuffix  = "_unsubscribe"
	notificationMethodSuffix = "_subscription"
	maxMethodNameLength      = 2048
)

var null = json.RawMessage("null")

// message is a JSON-RPC message: a call, a notification, a response or an
// error, depending on its fields.
type message struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *dispatch.Error `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	// written is called once the answer has been written: it activates the
	// subscription a call installed.
	written func()
}

func (m *message) hasValidVersion() bool {
	return m.Version == version
}

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

func (m *message) namespace() string {
	namespace, _, _ := strings.Cut(m.Method, "_")
	return namespace
}

// answer is the response to the message with a result or an error.
func (m *message) answer(response dispatch.Response) *message {
	if response.Error != nil {
		return m.errorResponse(response.Error)
	}
	return &message{Version: version, ID: m.ID, Result: response.Result}
}

func (m *message) errorResponse(err *dispatch.Error) *message {
	response := errorMessage(err)
	response.ID = m.ID
	return response
}

// errorMessage is an error response without an id to answer.
func errorMessage(err *dispatch.Error) *message {
	return &message{Version: version, ID: null, Error: err}
}

// invalidMessage is the error of a message that cannot be read.
func invalidMessage(text string) *dispatch.Error {
	return &dispatch.Error{Code: dispatch.CodeParse, Message: text}
}

func invalidRequest(text string) *dispatch.Error {
	return &dispatch.Error{Code: dispatch.CodeInvalidRequest, Message: text}
}

var (
	errNotificationsUnsupported = &dispatch.Error{Code: dispatch.CodeMethodNotFound, Message: "notifications not supported"}
	errTimeout                  = &dispatch.Error{Code: dispatch.CodeTimeout, Message: "request timed out"}
	errResponseTooLarge         = &dispatch.Error{Code: dispatch.CodeResponseTooLarge, Message: "response too large"}
	errBatchTooLarge            = invalidRequest("batch too large")
	errEmptyBatch               = invalidRequest("empty batch")
	errInvalidRequest           = invalidRequest("invalid request")
)

// parseMessages reads raw bytes, already checked to be JSON, as one message
// or a batch. As go-ethereum's server does, it ignores what does not fit a
// message, so such an element is answered as an invalid request.
func parseMessages(raw json.RawMessage) (messages []*message, batch bool) {
	if !isBatch(raw) {
		single := new(message)
		_ = json.Unmarshal(raw, single)
		return []*message{single}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_, _ = decoder.Token() // the opening bracket
	for decoder.More() {
		element := new(message)
		_ = decoder.Decode(&element)
		if element == nil {
			element = new(message) // a null element
		}
		messages = append(messages, element)
	}
	return messages, true
}

// isBatch tells whether the first character that is not white space is an
// opening bracket.
func isBatch(raw json.RawMessage) bool {
	for _, character := range raw {
		if character == 0x20 || character == 0x09 || character == 0x0a || character == 0x0d {
			continue
		}
		return character == '['
	}
	return false
}
