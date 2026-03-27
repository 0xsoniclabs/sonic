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

package genesis

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
)

func fakeHash() hash.Hash {
	return hash.Of(hash.FakeHash().Bytes())
}

func TestHashes_Includes(t *testing.T) {
	h := fakeHash()
	h1 := Hashes{"a": h}
	h2 := Hashes{"a": h, "b": fakeHash()}

	if !h1.Includes(h2) {
		t.Fatal("h1 should include h2 (h2 has all of h1's keys)")
	}
	if h2.Includes(h1) {
		t.Fatal("h2 should not include h1 (h1 is missing key 'b')")
	}
}

func TestHashes_Includes_Empty(t *testing.T) {
	empty := Hashes{}
	h := Hashes{"a": fakeHash()}

	if !empty.Includes(h) {
		t.Fatal("empty should include everything")
	}
	if !empty.Includes(empty) {
		t.Fatal("empty should include empty")
	}
}

func TestHashes_Includes_Mismatch(t *testing.T) {
	h1 := Hashes{"a": fakeHash()}
	h2 := Hashes{"a": fakeHash()} // different hash for same key

	if h1.Includes(h2) {
		t.Fatal("h1 should not include h2 when hashes differ")
	}
}

func TestHashes_Equal(t *testing.T) {
	fh := fakeHash()
	h1 := Hashes{"a": fh}
	h2 := Hashes{"a": fh}

	if !h1.Equal(h2) {
		t.Fatal("identical hashes should be equal")
	}
}

func TestHashes_Equal_DifferentSize(t *testing.T) {
	fh := fakeHash()
	h1 := Hashes{"a": fh}
	h2 := Hashes{"a": fh, "b": fakeHash()}

	if h1.Equal(h2) {
		t.Fatal("different size hashes should not be equal")
	}
}

func TestHashes_Equal_Empty(t *testing.T) {
	if !(Hashes{}).Equal(Hashes{}) {
		t.Fatal("empty hashes should be equal")
	}
}

func TestHeader_Equal(t *testing.T) {
	h1 := Header{
		GenesisID:   fakeHash(),
		NetworkID:   1,
		NetworkName: "test",
	}
	h2 := h1

	if !h1.Equal(h2) {
		t.Fatal("identical headers should be equal")
	}
}

func TestHeader_Equal_Different(t *testing.T) {
	h1 := Header{
		GenesisID:   fakeHash(),
		NetworkID:   1,
		NetworkName: "test",
	}
	h2 := Header{
		GenesisID:   fakeHash(),
		NetworkID:   2,
		NetworkName: "different",
	}

	if h1.Equal(h2) {
		t.Fatal("different headers should not be equal")
	}
}
