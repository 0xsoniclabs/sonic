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

package tests

import "github.com/ethereum/go-ethereum/common/hexutil"

// Contract code for tests that run calls.
var (
	// ReturnsFortyTwo returns the word 42.
	ReturnsFortyTwo = hexutil.MustDecode("0x602a60005260206000f3")
	// RevertsWithData reverts with the data 0xdeadbeef.
	RevertsWithData = hexutil.MustDecode("0x63deadbeef6000526004601cfd")
	// WritesStorage writes 1 into slot 0.
	WritesStorage = hexutil.MustDecode("0x600160005500")
	// EmitsLog emits a log of the topic 0x01 without data.
	EmitsLog = hexutil.MustDecode("0x600160006000a100")
	// Spins loops until its gas runs out.
	Spins = hexutil.MustDecode("0x5b600056")
	// ReturnsBlockHash returns the BLOCKHASH of the number its call data
	// holds.
	ReturnsBlockHash = hexutil.MustDecode("0x60003540600052602060" + "00f3")
)
