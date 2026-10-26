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
	"fmt"
)

// The JSON-RPC error codes the runtime answers with.
const (
	CodeParseError        = -32700
	CodeInvalidRequest    = -32600
	CodeMethodNotFound    = -32601
	CodeInvalidParameters = -32602
	CodeInternalError     = -32603
	CodeDefault           = -32000
	CodeTimeout           = -32002
	CodeResponseTooLarge  = -32003
)

// Error is a JSON-RPC error. A handler that needs a code other than the
// default -32000 returns one, or any error with the methods ErrorCode and
// ErrorData, as go-ethereum's server reads them.
type Error struct {
	Code    int
	Message string
	Data    any
}

func (err *Error) Error() string {
	return err.Message
}

// ErrorCode returns the JSON-RPC code of the error.
func (err *Error) ErrorCode() int {
	return err.Code
}

// ErrorData returns the data of the error, nil if there is none.
func (err *Error) ErrorData() any {
	return err.Data
}

// InvalidParameters is an error of code -32602.
func InvalidParameters(format string, values ...any) *Error {
	return &Error{Code: CodeInvalidParameters, Message: fmt.Sprintf(format, values...)}
}

// MethodNotFound is the error of a method no Register names.
func MethodNotFound(method string) *Error {
	return &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("the method %s does not exist/is not available", method)}
}

// ErrNotificationsUnsupported answers a subscription on a transport without
// notifications.
var ErrNotificationsUnsupported = &Error{Code: CodeMethodNotFound, Message: "notifications not supported"}

// ErrTimeout answers a call that did not finish before the write timeout of
// its transport.
var ErrTimeout = &Error{Code: CodeTimeout, Message: "request timed out"}

// ErrResponseTooLarge answers the calls of a batch whose responses together
// exceed the configured limit.
var ErrResponseTooLarge = &Error{Code: CodeResponseTooLarge, Message: "response too large"}

// ErrPanic answers a call whose handler panicked.
var ErrPanic = &Error{Code: CodeInternalError, Message: "method handler crashed"}

// ErrStopping answers a call that arrives after Stop began.
var ErrStopping = &Error{Code: CodeDefault, Message: "the RPC server is stopping"}

type coded interface {
	ErrorCode() int
}

type withData interface {
	ErrorData() any
}

// ToError turns any error into an Error the way go-ethereum's server does:
// the code of an error with ErrorCode, -32000 otherwise, and the data of an
// error with ErrorData.
func ToError(err error) *Error {
	if converted, ok := err.(*Error); ok {
		return converted
	}
	result := &Error{Code: CodeDefault, Message: err.Error()}
	if withCode, ok := err.(coded); ok {
		result.Code = withCode.ErrorCode()
	}
	if withErrorData, ok := err.(withData); ok {
		result.Data = withErrorData.ErrorData()
	}
	return result
}
