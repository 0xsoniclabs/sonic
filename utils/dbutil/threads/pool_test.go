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

package threads

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	debug.SetMaxThreads(10)

	os.Exit(m.Run())
}

func TestThreadPool(t *testing.T) {

	for name, pool := range map[string]*ThreadPool{
		"global": &GlobalPool,
		"local":  {},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			require.Equal(8, pool.Cap())

			got, release := pool.Lock(0)
			require.Equal(0, got)
			release(1)

			gotA, releaseA := pool.Lock(10)
			require.Equal(8, gotA)
			releaseA(1)

			gotB, releaseB := pool.Lock(10)
			require.Equal(1, gotB)
			releaseB(gotB)

			releaseA(gotA)
			gotB, releaseB = pool.Lock(10)
			require.Equal(8, gotB)

			// don't releaseB(gotB) to check pools isolation
			_ = releaseB
		})
	}
}
