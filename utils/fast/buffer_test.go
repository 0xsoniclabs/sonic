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

package fast

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	const N = 100

	var (
		w  *Writer
		r  *Reader
		bb = []byte{0, 0, 0xFF, 9, 0}
	)

	t.Run("Writer", func(t *testing.T) {
		require := require.New(t)

		w = NewWriter(make([]byte, 0, N/2))
		for i := byte(0); i < N; i++ {
			w.MustWriteByte(i)
		}
		require.Equal(N, len(w.Bytes()))
		w.Write(bb)
		require.Equal(N+len(bb), len(w.Bytes()))
	})

	t.Run("Reader", func(t *testing.T) {
		require := require.New(t)

		r = NewReader(w.Bytes())
		require.Equal(N+len(bb), len(r.Bytes()))
		require.False(r.Empty())
		for exp := byte(0); exp < N; exp++ {
			got := r.MustReadByte()
			require.Equal(exp, got)
		}
		require.Equal(N, r.Position())
		got := r.Read(len(bb))
		require.Equal(bb, got)
		require.True(r.Empty())
	})
}

func Benchmark(b *testing.B) {
	b.Run("Write", func(b *testing.B) {
		b.Run("Std", func(b *testing.B) {
			w := bytes.NewBuffer(make([]byte, 0, b.N))
			for i := 0; i < b.N; i++ {
				w.WriteByte(byte(i))
			}
			require.Equal(b, b.N, len(w.Bytes()))
		})
		b.Run("Fast", func(b *testing.B) {
			w := NewWriter(make([]byte, 0, b.N))
			for i := 0; i < b.N; i++ {
				w.MustWriteByte(byte(i))
			}
			require.Equal(b, b.N, len(w.Bytes()))
		})
	})

	b.Run("Read", func(b *testing.B) {
		src := make([]byte, 1000)
		_, _ = rand.Read(src)

		b.Run("Std", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r := bytes.NewReader(src)
				for j := 0; j < len(src); j++ {
					_, _ = r.ReadByte()
				}
			}
		})
		b.Run("Fast", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r := NewReader(src)
				for j := 0; j < len(src); j++ {
					_ = r.MustReadByte()
				}
			}
		})
	})
}
