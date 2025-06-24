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

package utils

import (
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNumQueue(t *testing.T) {

	t.Run("Simple", func(t *testing.T) {
		N := uint64(100)
		q := NewNumQueue(0)
		for i := uint64(1); i <= N; i++ {
			var iter sync.WaitGroup
			iter.Add(1)
			go func(i uint64) {
				defer iter.Done()
				q.WaitFor(i)
			}(i)

			q.Done(i)
			iter.Wait()
		}
	})

	t.Run("Random", func(t *testing.T) {
		require := require.New(t)
		N := 100

		q := NewNumQueue(0)
		output := make(chan uint64, 10)
		nums := rand.Perm(N)

		for _, n := range nums {
			go func(n uint64) {
				q.WaitFor(n - 1)
				output <- n
				if n == uint64(N) {
					close(output)
				}
				q.Done(n)

			}(uint64(n + 1))
		}

		var prev uint64
		for got := range output {
			require.Less(prev, got)
			prev = got
		}
	})
}
