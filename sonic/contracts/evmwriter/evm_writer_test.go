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

package evmwriter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	require := require.New(t)

	require.Equal([]byte{0xe3, 0x04, 0x43, 0xbc}, setBalanceMethodID)
	require.Equal([]byte{0xd6, 0xa0, 0xc7, 0xaf}, copyCodeMethodID)
	require.Equal([]byte{0x07, 0x69, 0x0b, 0x2a}, swapCodeMethodID)
	require.Equal([]byte{0x39, 0xe5, 0x03, 0xab}, setStorageMethodID)
	require.Equal([]byte{0x79, 0xbe, 0xad, 0x38}, incNonceMethodID)

}
