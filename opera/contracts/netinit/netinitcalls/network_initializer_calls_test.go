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

package netinitcall

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

func TestInitializeAll(t *testing.T) {
	data := InitializeAll(
		idx.Epoch(1),
		big.NewInt(1000000),
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
		common.HexToAddress("0x03"),
		common.HexToAddress("0x04"),
		common.HexToAddress("0x05"),
	)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
	if len(data) < 4 {
		t.Fatal("data too short for ABI-encoded call")
	}
}
