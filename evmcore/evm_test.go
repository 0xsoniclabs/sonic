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

package evmcore

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEVMBlockContext_DifficultyIsOne(t *testing.T) {
	header := &EvmHeader{
		Number: big.NewInt(12),
	}
	context := NewEVMBlockContext(header, nil, nil)
	require.Equal(t, big.NewInt(1), context.Difficulty)
}

func TestNewEVMBlockContextWithDifficulty_UsesProvidedDifficulty(t *testing.T) {
	header := &EvmHeader{
		Number: big.NewInt(12),
	}
	for i := range int64(10) {
		difficulty := big.NewInt(i)
		context := NewEVMBlockContextWithDifficulty(header, nil, nil, difficulty)
		require.Equal(t, difficulty, context.Difficulty)
	}
}
