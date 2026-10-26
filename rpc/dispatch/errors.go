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
	"errors"
	"fmt"
)

// JSON-RPC error codes the runtime answers with.
const (
	CodeDefault          = -32000
	CodeTimeout          = -32002
	CodeResponseTooLarge = -32003
	CodeParse            = -32700
	CodeInvalidRequest   = -32600
	CodeMethodNotFound   = -32601
	CodeInvalidParams    = -32602
	CodeInternal         = -32603
)

// Error is a JSON-RPC error with a code and optional data. A handler returns
// it for an answer whose code is not -32000.
type Error struct {
	Code    int
	Message string
	Data    any
}

func (e *Error) Error() string  { return e.Message }
func (e *Error) ErrorCode() int { return e.Code }
func (e *Error) ErrorData() any { return e.Data }

// Coded is an error that carries its JSON-RPC code. Like go-ethereum, the
// dispatcher looks at the returned error itself and does not unwrap it: an
// error wrapped with %w is answered with -32000.
type Coded interface {
	error
	ErrorCode() int
}

// WithData is an error that carries data for the error object.
type WithData interface {
	error
	ErrorData() any
}

// InvalidParams is the error of a parameter a method refuses as malformed.
func InvalidParams(format string, args ...any) *Error {
	return &Error{Code: CodeInvalidParams, Message: fmt.Sprintf(format, args...)}
}

// ErrNotificationsUnsupported answers a subscription over a transport that
// cannot deliver notifications.
var ErrNotificationsUnsupported = &Error{Code: CodeMethodNotFound, Message: "notifications not supported"}

// ErrSubscriptionNotFound answers an unsubscription of an id the connection
// does not hold.
var ErrSubscriptionNotFound = errors.New("subscription not found")

var (
	errPanic   = &Error{Code: CodeInternal, Message: "method handler crashed"}
	errStopped = &Error{Code: CodeInternal, Message: "server is shutting down"}
)

// MethodNotFound answers a method the transport does not serve.
func MethodNotFound(method string) *Error {
	return &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("the method %s does not exist/is not available", method)}
}

// SubscriptionNotFound answers a subscription of a kind the namespace does not
// have on the connection's transport.
func SubscriptionNotFound(namespace, kind string) *Error {
	return &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("no %q subscription in %s namespace", kind, namespace)}
}

// ErrorObject is the error member of a response.
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ToErrorObject maps an error to its JSON-RPC form: the code and data of the
// error itself, -32000 otherwise.
func ToErrorObject(err error) ErrorObject {
	obj := ErrorObject{Code: CodeDefault, Message: err.Error()}
	if c, ok := err.(Coded); ok {
		obj.Code = c.ErrorCode()
	}
	if d, ok := err.(WithData); ok {
		obj.Data = d.ErrorData()
	}
	return obj
}
