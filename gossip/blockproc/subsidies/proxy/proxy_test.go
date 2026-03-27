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

package proxy

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGetSlotForImplementation(t *testing.T) {
	slot := GetSlotForImplementation()
	expected := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	if slot != expected {
		t.Fatalf("unexpected slot: %s", slot.Hex())
	}
}

func TestGetCode(t *testing.T) {
	code := GetCode()
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}

	// Verify it returns a copy, not the original
	code2 := GetCode()
	code[0] = 0xff
	if code2[0] == 0xff {
		t.Fatal("GetCode should return a copy")
	}
}
