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

// Package transport carries JSON-RPC 2.0 over HTTP, WebSocket and unix
// sockets. All transports share one message layer and one Front, which routes
// a call to the dispatcher or, while methods are being ported, to the node's
// legacy handler.
package transport

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

const (
	version             = "2.0"
	maxMethodNameLength = 2048
)

var null = json.RawMessage("null")

// Message is a JSON-RPC request, notification, response or error response.
// Which one it is depends on the fields.
type Message struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *dispatch.Error `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`

	// activate lets the notifications of a new subscription out, once the
	// response that carries its id was written.
	activate func()
}

func (m *Message) hasValidVersion() bool { return m.Version == version }

func (m *Message) hasValidID() bool {
	return len(m.ID) > 0 && m.ID[0] != '{' && m.ID[0] != '['
}

func (m *Message) isNotification() bool {
	return m.hasValidVersion() && m.ID == nil && m.Method != ""
}

func (m *Message) isCall() bool {
	return m.hasValidVersion() && m.hasValidID() && m.Method != ""
}

func (m *Message) isResponse() bool {
	return m.hasValidVersion() && m.hasValidID() && m.Method == "" && m.Params == nil &&
		(m.Result != nil || m.Error != nil)
}

func (m *Message) isSubscribe() bool { return strings.HasSuffix(m.Method, "_subscribe") }

func (m *Message) isUnsubscribe() bool { return strings.HasSuffix(m.Method, "_unsubscribe") }

// namespace returns the part of the method name before the first underscore.
func (m *Message) namespace() string { return namespaceOf(m.Method) }

// namespaceOf returns the part of a method name before the first underscore.
func namespaceOf(method string) string {
	before, _, _ := strings.Cut(method, "_")
	return before
}

func (m *Message) errorResponse(err *dispatch.Error) *Message {
	resp := errorMessage(err)
	resp.ID = m.ID
	return resp
}

func (m *Message) response(result json.RawMessage) *Message {
	return &Message{Version: version, ID: m.ID, Result: result}
}

func errorMessage(err *dispatch.Error) *Message {
	return &Message{Version: version, ID: null, Error: err}
}

// parseMessages parses raw bytes as a (batch of) JSON-RPC message(s). The raw
// message has already been syntax-checked, so there are no error checks: any
// non-JSON-RPC message yields the zero Message.
func parseMessages(raw json.RawMessage) (msgs []*Message, batch bool) {
	if !isBatch(raw) {
		msgs = []*Message{{}}
		_ = json.Unmarshal(raw, &msgs[0])
		if msgs[0] == nil {
			msgs[0] = new(Message)
		}
		return msgs, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	_, _ = dec.Token() // skip '['
	for dec.More() {
		msg := new(Message)
		_ = dec.Decode(msg)
		msgs = append(msgs, msg)
	}
	return msgs, true
}

// isBatch reports whether the first non-whitespace character is '['.
func isBatch(raw json.RawMessage) bool {
	for _, c := range raw {
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}
