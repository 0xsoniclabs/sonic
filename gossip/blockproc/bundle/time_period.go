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

package bundle

import (
	"io"
	"math"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/rlp"
)

// TimePeriod defines a real-time period.
type TimePeriod struct {
	// Start is the begin of the period (unix nanoseconds).
	Start inter.Timestamp
	// Duration is the duration of the period (in nanoseconds).
	Duration uint64
}

// MakeUnrestrictedTimePeriod creates a TimePeriod that is valid for the maximum
// possible duration, starting from the unix epoch (0). This corresponds to the
// half-open interval [0, math.MaxUint64).
func MakeUnrestrictedTimePeriod() TimePeriod {
	return TimePeriod{
		Start:    0,
		Duration: math.MaxUint64,
	}
}

// IsInPeriod checks if the given time is within this time period. The period is
// the half-open interval [Start, Start+Duration).
func (p TimePeriod) IsInPeriod(time inter.Timestamp) bool {
	return !p.IsBeforePeriod(time) && !p.IsAfterPeriod(time)
}

// IsBeforePeriod checks if the given time is before this time period, meaning
// that it is less than the start time of the period.
func (p TimePeriod) IsBeforePeriod(time inter.Timestamp) bool {
	return time < p.Start
}

// IsAfterPeriod checks if the given time is after this time period.
func (p TimePeriod) IsAfterPeriod(time inter.Timestamp) bool {
	// p.Start + p.Duration may overflow, so we subtract the start
	return time >= p.Start && time-p.Start >= inter.Timestamp(p.Duration)
}

func (p TimePeriod) encode(writer io.Writer) error {
	// The common case of an unrestricted period should be encoded efficiently,
	// without the need to encode the full struct.
	wrapped := encodedTimePeriod{TimePeriod: &p}
	if p == MakeUnrestrictedTimePeriod() {
		wrapped.TimePeriod = nil
	}
	return rlp.Encode(writer, wrapped)
}

func (p *TimePeriod) decode(reader io.Reader) error {
	var wrapped encodedTimePeriod
	if err := rlp.Decode(reader, &wrapped); err != nil {
		return err
	}
	if wrapped.TimePeriod == nil {
		*p = MakeUnrestrictedTimePeriod()
	} else {
		*p = *wrapped.TimePeriod
	}
	return nil
}

type encodedTimePeriod struct {
	*TimePeriod `rlp:"nil"`
}
