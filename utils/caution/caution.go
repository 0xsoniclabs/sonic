// Copyright 2025 Sonic Operations Ltd
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

// package caution provides utility functions for handling errors in defer
// cleanup and closing resources.
package caution

import (
	"errors"
	"fmt"
	"io"
)

// ExecuteAndReportError is intended for use in production code to handle errors
// ignored in defer clean ups.
//
// - The first argument is the error variable where the error, if any, will be
// accumulated. If it already contains an error, the new error will be combined.
// - The second argument is a cleanup function that may return an error. This
// function will always be executed.
// - The third argument is a mandatory context message that will be used in the
// error if the cleanup function fails.
//
// Usage example:
//
// Original code:
//
//	func F(....) error {
//	    [...]
//	    defer f.CleanUpThatMayFail(someArg)
//	    [...]
//	}
//
// Refactored with the new functions:
//
//	func F(....) (err error) {
//	    [...]
//	    defer ExecuteAndReportError(&err, f() error {f.CleanUpThatMayFail(someArg) }, "failed to cleanup f")
//	    [...]
//	}
func ExecuteAndReportError(err *error, f func() error, message string) {
	fErr := f()
	if fErr != nil {
		*err = errors.Join(*err, fmt.Errorf("%s: %w", message, fErr))
	}
}

// CloseAndReportError is specialization of ExecuteAndReportError for types that
// implement the Closer interface, to add error management in the
// `defer f.Close()` pattern.
// Usage example:
//
// Original code:
//
//	func F(....) error {
//	    [...]
//	    defer f.Close()
//	    [...]
//	}
//
// Refactored with the new functions:
//
//	func F(....) (err error) {
//	    [...]
//	    defer CloseAndReportError(&err, f, "failed to close f")
//	    [...]
//	}
func CloseAndReportError(err *error, closer io.Closer, message string) {
	ExecuteAndReportError(err, func() error {
		return closer.Close()
	}, message)
}
