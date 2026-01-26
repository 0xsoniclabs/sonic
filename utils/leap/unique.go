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

// Unique wraps an iterator and filters out duplicate consecutive values.
func Unique[T comparable](iterator Iterator[T]) *unique[T] {
	return &unique[T]{
		iterator: iterator,
	}
}

type unique[T comparable] struct {
	iterator Iterator[T]
	last     T
}

func (it *unique[T]) Next() bool {
	for it.iterator.Next() {
		cur := it.iterator.Cur()
		if cur != it.last {
			it.last = cur
			return true
		}
	}
	return false
}

func (it *unique[T]) Cur() T {
	return it.iterator.Cur()
}

func (it *unique[T]) Seek(value T) bool {
	if !it.iterator.Seek(value) {
		return false
	}
	it.last = it.iterator.Cur()
	return true
}

func (it *unique[T]) Release() {
	it.iterator.Release()
}
