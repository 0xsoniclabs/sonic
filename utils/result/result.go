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

package result

// T is a result of an operation that can return a value or an error.
type T[V any] struct {
	value V
	err   error
}

// New creates a new result with a value. This is the type of result that is
// returned when the operation is successful.
func New[V any](value V) T[V] {
	return T[V]{value: value}
}

// Error creates a new result with an error. This is the type of result that is
// returned when the operation is unsuccessful. Note, passing nil as an error
// will result in a non-error result containing the zero value.
func Error[V any](err error) T[V] {
	return T[V]{err: err}
}

// Unwrap returns the value and error of the result. If the operation was a
// success, the error will be nil. If the operation was a failure, the value
// will be the zero value of the type and the error will describe the failure.
func (r T[V]) Unwrap() (V, error) {
	return r.value, r.err
}

// IsError returns true if the result is an error.
func (r T[V]) IsError() bool {
	return r.err != nil
}
