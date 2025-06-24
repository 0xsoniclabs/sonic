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

package inter

import (
	"time"

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
)

type (
	// Timestamp is a UNIX nanoseconds timestamp
	Timestamp uint64
)

// Bytes gets the byte representation of the index.
func (t Timestamp) Bytes() []byte {
	return bigendian.Uint64ToBytes(uint64(t))
}

func FromUnix(t int64) Timestamp {
	return Timestamp(int64(t) * int64(time.Second))
}

// Unix returns t as a Unix time, the number of seconds elapsed
// since January 1, 1970 UTC. The result does not depend on the
// location associated with t.
func (t Timestamp) Unix() int64 {
	return int64(t) / int64(time.Second)
}

func (t Timestamp) Time() time.Time {
	return time.Unix(int64(t)/int64(time.Second), int64(t)%int64(time.Second))
}

// MaxTimestamp return max value.
func MaxTimestamp(x, y Timestamp) Timestamp {
	if x > y {
		return x
	}
	return y
}
