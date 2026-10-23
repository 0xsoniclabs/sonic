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

// SpinCode runs until it is out of gas or stopped: JUMPDEST PUSH1 0 JUMP.
func SpinCode() []byte { return []byte{0x5b, 0x60, 0x00, 0x56} }

// StoreCode writes 1 to slot 0: PUSH1 1 PUSH1 0 SSTORE STOP.
func StoreCode() []byte { return []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00} }
