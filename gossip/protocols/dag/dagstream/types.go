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

package dagstream

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/lachesis-base/gossip/basestream"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
)

// Request represents a DAG stream request specifying a session, size limit, type, and chunk count.
type Request struct {
	Session   Session
	Limit     dag.Metric
	Type      basestream.RequestType
	MaxChunks uint32
}

// Response represents a DAG stream response containing event IDs and serialized event data.
type Response struct {
	SessionID uint32
	Done      bool
	IDs       hash.Events
	Events    [][]byte
}

// Session identifies a streaming session with a unique ID and a start/stop locator range.
type Session struct {
	ID    uint32
	Start Locator
	Stop  Locator
}

// Locator is a byte-slice key used to identify a position in the DAG stream.
type Locator []byte

// Compare returns the lexicographic comparison of two locators.
func (l Locator) Compare(b basestream.Locator) int {
	return bytes.Compare(l, b.(Locator))
}

// Inc returns a new locator incremented by one, preserving the original byte length.
func (l Locator) Inc() basestream.Locator {
	nextBn := new(big.Int).SetBytes(l)
	nextBn.Add(nextBn, common.Big1)
	return Locator(common.LeftPadBytes(nextBn.Bytes(), len(l)))
}

// Payload holds event IDs, serialized event data, and their cumulative size.
type Payload struct {
	IDs    hash.Events
	Events [][]byte
	Size   uint64
}

// AddEvent appends an event ID and its serialized bytes to the payload.
func (p *Payload) AddEvent(id hash.Event, eventB []byte) {
	p.IDs = append(p.IDs, id)
	p.Events = append(p.Events, eventB)
	p.Size += uint64(len(eventB))
}

// AddID appends an event ID to the payload without storing the event bytes.
func (p *Payload) AddID(id hash.Event, size int) {
	p.IDs = append(p.IDs, id)
	p.Size += uint64(size)
}

// Len returns the number of event IDs in the payload.
func (p Payload) Len() int {
	return len(p.IDs)
}

// TotalSize returns the cumulative byte size of all events in the payload.
func (p Payload) TotalSize() uint64 {
	return p.Size
}

// TotalMemSize returns the estimated in-memory size of the payload including ID overhead.
func (p Payload) TotalMemSize() int {
	if len(p.Events) != 0 {
		return int(p.Size) + len(p.IDs)*128
	}
	return len(p.IDs) * 128
}

const (
	RequestIDs    basestream.RequestType = 0
	RequestEvents basestream.RequestType = 2
)
