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

package verwatcher

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
)

func TestNewStore(t *testing.T) {
	db := memorydb.New()
	s := NewStore(db)
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
}

func TestStore_NetworkVersion_DefaultZero(t *testing.T) {
	db := memorydb.New()
	s := NewStore(db)

	v := s.GetNetworkVersion()
	if v != 0 {
		t.Errorf("expected default network version 0, got %d", v)
	}
}

func TestStore_SetAndGetNetworkVersion(t *testing.T) {
	db := memorydb.New()
	s := NewStore(db)

	s.SetNetworkVersion(42)
	if got := s.GetNetworkVersion(); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestStore_NetworkVersion_PersistsAcrossInstances(t *testing.T) {
	db := memorydb.New()

	s1 := NewStore(db)
	s1.SetNetworkVersion(100)

	// Create a new store instance using the same DB.
	s2 := NewStore(db)
	if got := s2.GetNetworkVersion(); got != 100 {
		t.Errorf("expected 100 from new instance, got %d", got)
	}
}

func TestStore_MissedVersion_DefaultZero(t *testing.T) {
	db := memorydb.New()
	s := NewStore(db)

	v := s.GetMissedVersion()
	if v != 0 {
		t.Errorf("expected default missed version 0, got %d", v)
	}
}

func TestStore_SetAndGetMissedVersion(t *testing.T) {
	db := memorydb.New()
	s := NewStore(db)

	s.SetMissedVersion(77)
	if got := s.GetMissedVersion(); got != 77 {
		t.Errorf("expected 77, got %d", got)
	}
}

func TestStore_MissedVersion_PersistsAcrossInstances(t *testing.T) {
	db := memorydb.New()

	s1 := NewStore(db)
	s1.SetMissedVersion(200)

	s2 := NewStore(db)
	if got := s2.GetMissedVersion(); got != 200 {
		t.Errorf("expected 200 from new instance, got %d", got)
	}
}

func TestStore_NetworkAndMissedVersion_Independent(t *testing.T) {
	db := memorydb.New()
	s := NewStore(db)

	s.SetNetworkVersion(10)
	s.SetMissedVersion(20)

	if got := s.GetNetworkVersion(); got != 10 {
		t.Errorf("expected network version 10, got %d", got)
	}
	if got := s.GetMissedVersion(); got != 20 {
		t.Errorf("expected missed version 20, got %d", got)
	}
}
