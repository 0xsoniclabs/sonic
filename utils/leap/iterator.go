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

package leap

//go:generate mockgen -source=iterator.go -package=leap -destination=iterator_mock.go

// Iterator defines a generic iterator over a sequence of ordered elements of
// type T. It is a general abstraction of a data source that can be utilized
// by the algorithms in this package.
type Iterator[T any] interface {
	// Next advances the iterator to the next element. Initially, the iterator
	// is positioned before the first element, so Next must be called to advance
	// it to the first element. Next returns true if the iterator was advanced
	// to a valid element, and false if the iterator is exhausted.
	Next() bool

	// Seek advances the iterator to the smallest element greater than or equal
	// to the given target value. It returns true if such an element was found,
	// and false if the iterator is exhausted. If exhausted, the iterator
	// points to the position after the last element, invalid for Current.
	//
	// Backward seeks are not supported. If the seek target is less than the
	// current element, the iterator is not moved (i.e., Seek is a no-op in
	// that case).
	Seek(T) bool

	// Current returns the current element. It must only be called after Next or
	// Seek has returned true.
	Current() T

	// Release releases any resources held by the iterator. It must be called
	// when the iterator is no longer needed.
	Release()
}
