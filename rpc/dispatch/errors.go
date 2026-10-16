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

// Package dispatch is the runtime of the RPC module: the method table, the
// request pipeline (lookup, decode, deadline, recover, metrics, encode), the
// Group that owns request-scoped goroutines, and the types every namespace
// shares (block references, the served head, configuration).
package dispatch

import "fmt"

// JSON-RPC 2.0 error codes used by the module. The codes match go-ethereum's
// rpc.Server, so clients see the same numbers as before.
const (
	CodeParse          = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternal       = -32603
	CodeDefault        = -32000
	CodeTimeout        = -32002
	CodeTooLarge       = -32003
)

// CodedError is an error that carries a JSON-RPC error code. It is the
// structural twin of go-ethereum's rpc.Error: errors of the node's legacy
// packages satisfy it without importing this package.
type CodedError interface {
	error
	ErrorCode() int
}

// DataError is an error that carries data for the JSON-RPC error object. It is
// the twin of go-ethereum's rpc.DataError.
type DataError interface {
	error
	ErrorData() any
}

// Error is a JSON-RPC error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("json-rpc error %d", e.Code)
	}
	return e.Message
}

func (e *Error) ErrorCode() int { return e.Code }

func (e *Error) ErrorData() any { return e.Data }

// NewError returns an error with the given code and message.
func NewError(code int, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// InvalidParams returns a -32602 error.
func InvalidParams(format string, args ...any) *Error {
	return NewError(CodeInvalidParams, format, args...)
}

// MethodNotFound returns the -32601 error of an unknown method.
func MethodNotFound(method string) *Error {
	return NewError(CodeMethodNotFound, "the method %s does not exist/is not available", method)
}

// ErrNotificationsUnsupported is the answer to a subscription request on a
// transport without a connection to notify.
var ErrNotificationsUnsupported = &Error{Code: CodeMethodNotFound, Message: "notifications not supported"}

// ToError converts the error of a handler into the error object of the
// response: errors that carry a code and data keep them, any other error is a
// plain -32000, as go-ethereum answers it. Like go-ethereum it looks at the
// error itself and does not unwrap it, so wrapping an error changes its code.
func ToError(err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	}
	res := &Error{Code: CodeDefault, Message: err.Error()}
	if coded, ok := err.(CodedError); ok {
		res.Code = coded.ErrorCode()
	}
	if data, ok := err.(DataError); ok {
		res.Data = data.ErrorData()
	}
	return res
}
