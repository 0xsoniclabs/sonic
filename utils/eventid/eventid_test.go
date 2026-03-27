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

package eventid

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

func TestNewCache(t *testing.T) {
	c := NewCache(100)
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
	if c.maxSize != 100 {
		t.Fatalf("expected maxSize 100, got %d", c.maxSize)
	}
}

func TestCache_BeforeReset(t *testing.T) {
	c := NewCache(100)

	has, ok := c.Has(hash.ZeroEvent)
	if ok {
		t.Fatal("expected ok=false before Reset")
	}
	if has {
		t.Fatal("expected has=false before Reset")
	}

	if c.Add(hash.ZeroEvent) {
		t.Fatal("expected Add to return false before Reset")
	}
}

func TestCache_Reset(t *testing.T) {
	c := NewCache(100)
	c.Reset(hash.FakeEpoch())

	if c.epoch != hash.FakeEpoch() {
		t.Fatalf("expected epoch %d, got %d", hash.FakeEpoch(), c.epoch)
	}
}

func TestCache_AddAndHas(t *testing.T) {
	c := NewCache(100)
	c.Reset(hash.FakeEpoch())

	e := hash.FakeEvent()

	has, ok := c.Has(e)
	if !ok {
		t.Fatal("expected ok=true after Reset with correct epoch")
	}
	if has {
		t.Fatal("expected has=false for missing event")
	}

	if !c.Add(e) {
		t.Fatal("expected Add to succeed")
	}

	has, ok = c.Has(e)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !has {
		t.Fatal("expected has=true after Add")
	}
}

func TestCache_WrongEpoch(t *testing.T) {
	c := NewCache(100)
	c.Reset(idx.Epoch(1)) // epoch 1

	e := hash.FakeEvent() // uses FakeEpoch (123456)

	has, ok := c.Has(e)
	if ok {
		t.Fatal("expected ok=false for wrong epoch")
	}
	if has {
		t.Fatal("expected has=false for wrong epoch")
	}

	if c.Add(e) {
		t.Fatal("expected Add to return false for wrong epoch")
	}
}

func TestCache_Remove(t *testing.T) {
	c := NewCache(100)
	c.Reset(hash.FakeEpoch())

	e := hash.FakeEvent()
	c.Add(e)

	c.Remove(e)

	has, ok := c.Has(e)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if has {
		t.Fatal("expected has=false after Remove")
	}
}

func TestCache_Remove_BeforeReset(t *testing.T) {
	c := NewCache(100)
	// Should not panic
	c.Remove(hash.ZeroEvent)
}

func TestCache_MaxSize(t *testing.T) {
	c := NewCache(2)
	c.Reset(hash.FakeEpoch())

	e1 := hash.FakeEvent()
	e2 := hash.FakeEvent()
	e3 := hash.FakeEvent()

	if !c.Add(e1) {
		t.Fatal("expected Add e1 to succeed")
	}
	if !c.Add(e2) {
		t.Fatal("expected Add e2 to succeed")
	}

	// Adding third should fail and nil out the ids map
	if c.Add(e3) {
		t.Fatal("expected Add e3 to fail (over capacity)")
	}

	has, ok := c.Has(e1)
	if ok {
		t.Fatal("expected ok=false after overflow")
	}
	if has {
		t.Fatal("expected has=false after overflow")
	}
}

func TestCache_ResetClearsData(t *testing.T) {
	c := NewCache(100)
	c.Reset(hash.FakeEpoch())

	e := hash.FakeEvent()
	c.Add(e)

	c.Reset(idx.Epoch(999))

	has, ok := c.Has(e)
	if ok {
		t.Fatal("expected ok=false for old epoch event")
	}
	if has {
		t.Fatal("expected has=false for old epoch event")
	}
}
