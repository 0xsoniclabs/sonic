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

package basestreamseeder

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/sonic/gossip/basestream"
)

type testLocator struct {
	B []byte
}

func (l testLocator) Compare(b basestream.Locator) int {
	return bytes.Compare(l.B, b.(testLocator).B)
}

func (l testLocator) Inc() basestream.Locator {
	nextBn := new(big.Int).SetBytes(l.B)
	nextBn.Add(nextBn, common.Big1)
	return testLocator{
		B: nextBn.Bytes(),
	}
}

type testPayload struct {
	IDs    consensus.EventHashes
	Events consensus.Events
	Size   uint64
}

func (p testPayload) AddEvent(id consensus.EventHash, event consensus.Event) {
	p.IDs = append(p.IDs, id)          // nolint:staticcheck
	p.Events = append(p.Events, event) // nolint:staticcheck
	p.Size += uint64(event.Size())     // nolint:staticcheck
}

func (p testPayload) Len() int {
	return len(p.IDs)
}

func (p testPayload) TotalSize() uint64 {
	return p.Size
}

func (p testPayload) TotalMemSize() int {
	return int(p.Size) + len(p.IDs)*128
}
