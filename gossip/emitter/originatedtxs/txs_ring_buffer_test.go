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

package originatedtxs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNew(t *testing.T) {
	buf := New(10)
	if buf == nil {
		t.Fatal("New returned nil")
	}
	if !buf.Empty() {
		t.Error("new buffer should be empty")
	}
}

func TestInc_And_TotalOf(t *testing.T) {
	buf := New(10)
	addr := common.HexToAddress("0x01")

	if got := buf.TotalOf(addr); got != 0 {
		t.Errorf("expected 0 for unknown address, got %d", got)
	}

	buf.Inc(addr)
	if got := buf.TotalOf(addr); got != 1 {
		t.Errorf("expected 1 after one Inc, got %d", got)
	}

	buf.Inc(addr)
	buf.Inc(addr)
	if got := buf.TotalOf(addr); got != 3 {
		t.Errorf("expected 3 after three Inc calls, got %d", got)
	}
}

func TestDec(t *testing.T) {
	buf := New(10)
	addr := common.HexToAddress("0x02")

	// Dec on unknown address should be a no-op.
	buf.Dec(addr)
	if got := buf.TotalOf(addr); got != 0 {
		t.Errorf("expected 0 after Dec on unknown, got %d", got)
	}

	buf.Inc(addr)
	buf.Inc(addr)
	buf.Dec(addr)
	if got := buf.TotalOf(addr); got != 1 {
		t.Errorf("expected 1 after 2 Inc + 1 Dec, got %d", got)
	}

	// Dec to zero should remove the entry entirely.
	buf.Dec(addr)
	if got := buf.TotalOf(addr); got != 0 {
		t.Errorf("expected 0 after decrementing to zero, got %d", got)
	}
	if !buf.Empty() {
		t.Error("buffer should be empty after all entries removed")
	}
}

func TestClear(t *testing.T) {
	buf := New(10)
	addr1 := common.HexToAddress("0x01")
	addr2 := common.HexToAddress("0x02")

	buf.Inc(addr1)
	buf.Inc(addr2)
	buf.Clear()

	if !buf.Empty() {
		t.Error("buffer should be empty after Clear")
	}
	if got := buf.TotalOf(addr1); got != 0 {
		t.Errorf("expected 0 for addr1 after Clear, got %d", got)
	}
}

func TestEmpty(t *testing.T) {
	buf := New(10)
	if !buf.Empty() {
		t.Error("new buffer should be empty")
	}

	addr := common.HexToAddress("0x03")
	buf.Inc(addr)
	if buf.Empty() {
		t.Error("buffer should not be empty after Inc")
	}

	buf.Dec(addr)
	if !buf.Empty() {
		t.Error("buffer should be empty after removing last entry")
	}
}

func TestMultipleAddresses(t *testing.T) {
	buf := New(100)
	addr1 := common.HexToAddress("0x01")
	addr2 := common.HexToAddress("0x02")
	addr3 := common.HexToAddress("0x03")

	buf.Inc(addr1)
	buf.Inc(addr1)
	buf.Inc(addr2)
	buf.Inc(addr3)
	buf.Inc(addr3)
	buf.Inc(addr3)

	if got := buf.TotalOf(addr1); got != 2 {
		t.Errorf("addr1: expected 2, got %d", got)
	}
	if got := buf.TotalOf(addr2); got != 1 {
		t.Errorf("addr2: expected 1, got %d", got)
	}
	if got := buf.TotalOf(addr3); got != 3 {
		t.Errorf("addr3: expected 3, got %d", got)
	}
}

func TestLRU_Eviction(t *testing.T) {
	// With maxAddresses=2, adding a third should evict the least recently used.
	buf := New(2)
	addr1 := common.HexToAddress("0x01")
	addr2 := common.HexToAddress("0x02")
	addr3 := common.HexToAddress("0x03")

	buf.Inc(addr1)
	buf.Inc(addr2)
	// Access addr1 via TotalOf to make addr2 the least recently used.
	buf.TotalOf(addr1)
	buf.Inc(addr3) // should evict addr2

	if got := buf.TotalOf(addr2); got != 0 {
		t.Errorf("addr2 should have been evicted, got %d", got)
	}
	if got := buf.TotalOf(addr1); got != 1 {
		t.Errorf("addr1 should still be present, got %d", got)
	}
	if got := buf.TotalOf(addr3); got != 1 {
		t.Errorf("addr3 should be present, got %d", got)
	}
}
