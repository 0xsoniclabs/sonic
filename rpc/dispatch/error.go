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

import "fmt"

// The error codes of JSON-RPC 2.0 and the ones go-ethereum's server adds.
const (
	CodeParse            = -32700
	CodeInvalidRequest   = -32600
	CodeMethodNotFound   = -32601
	CodeInvalidParams    = -32602
	CodeInternal         = -32603
	CodeDefault          = -32000
	CodeTimeout          = -32002
	CodeResponseTooLarge = -32003
)

// Error is a JSON-RPC error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// ErrorCode makes an *Error keep its code when a handler returns it.
func (e *Error) ErrorCode() int { return e.Code }

// ErrorData makes an *Error keep its data when a handler returns it.
func (e *Error) ErrorData() any { return e.Data }

// codedError is an error that names its JSON-RPC code.
type codedError interface {
	error
	ErrorCode() int
}

// dataError is an error that carries JSON-RPC error data.
type dataError interface {
	error
	ErrorData() any
}

// ToError turns an error into its JSON-RPC form: code -32000 unless the error
// names its own, with the error's data if it carries any. As go-ethereum's
// server does, only the error itself is asked, not the errors it wraps.
func ToError(err error) *Error {
	result := &Error{Code: CodeDefault, Message: err.Error()}
	if coded, ok := err.(codedError); ok {
		result.Code = coded.ErrorCode()
	}
	if data, ok := err.(dataError); ok {
		result.Data = data.ErrorData()
	}
	return result
}

// MethodNotFound is the error of a method no one serves.
func MethodNotFound(method string) *Error {
	return &Error{Code: CodeMethodNotFound, Message: fmt.Sprintf("the method %s does not exist/is not available", method)}
}

// InvalidParams is the error of parameters that cannot be decoded.
func InvalidParams(message string) *Error {
	return &Error{Code: CodeInvalidParams, Message: message}
}

var errHandlerCrashed = &Error{Code: CodeInternal, Message: "method handler crashed"}

var errShuttingDown = &Error{Code: CodeInternal, Message: "the RPC server is shutting down"}
