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

package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericNameOfTmpDB(t *testing.T) {
	require := require.New(t)

	for name, exp := range map[string]string{
		"":              "",
		"main":          "main",
		"main-single":   "main-single",
		"lachesis-0":    "lachesis-tmp",
		"lachesis-0999": "lachesis-tmp",
		"gossip-50":     "gossip-tmp",
		"epoch-1":       "epoch-tmp",
		"xxx-1a":        "xxx-1a",
		"123":           "123",
	} {
		got := genericNameOfTmpDB(name)
		require.Equal(exp, got, name)
	}
}
