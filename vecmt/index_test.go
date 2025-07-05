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

package vecmt

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/kvdb/memorydb"

	"github.com/0xsoniclabs/sonic/inter"
)

var (
	testASCIIScheme = `
a1.0   b1.0   c1.0   d1.0   e1.0
║      ║      ║      ║      ║
║      ╠──────╫───── d2.0   ║
║      ║      ║      ║      ║
║      b2.1 ──╫──────╣      e2.1
║      ║      ║      ║      ║
║      ╠──────╫───── d3.1   ║
a2.1 ──╣      ║      ║      ║
║      ║      ║      ║      ║
║      b3.2 ──╣      ║      ║
║      ║      ║      ║      ║
║      ╠──────╫───── d4.2   ║
║      ║      ║      ║      ║
║      ╠───── c2.2   ║      e3.2
║      ║      ║      ║      ║
`
)

type eventWithCreationTime struct {
	consensus.Event
	creationTime inter.Timestamp
}

func (e *eventWithCreationTime) CreationTime() inter.Timestamp {
	return e.creationTime
}

func BenchmarkIndex_Add(b *testing.B) {
	b.StopTimer()
	ordered := make(consensus.Events, 0)
	nodes, _, _ := consensustest.ASCIIschemeForEach(testASCIIScheme, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			ordered = append(ordered, e)
		},
	})
	validatorsBuilder := consensus.NewBuilder()
	for _, peer := range nodes {
		validatorsBuilder.Set(peer, 1)
	}
	validators := validatorsBuilder.Build()
	events := make(map[consensus.EventHash]consensus.Event)
	getEvent := func(id consensus.EventHash) consensus.Event {
		return events[id]
	}
	for _, e := range ordered {
		events[e.ID()] = e
	}

	vecClock := NewIndex(func(err error) { panic(err) }, LiteConfig())
	vecClock.Reset(validators, memorydb.New(), getEvent)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		vecClock.Reset(validators, memorydb.New(), getEvent)
		b.StartTimer()
		for _, e := range ordered {
			err := vecClock.Add(&eventWithCreationTime{e, inter.Timestamp(e.Seq())})
			if err != nil {
				panic(err)
			}
			i++
			if i >= b.N {
				break
			}
		}
	}
}
