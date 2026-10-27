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

package opera

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestForkID_DependsOnTheGenesisTheHeightAndTheUpgrades(t *testing.T) {
	upgrade := UpgradeHeight{Upgrades: GetSonicUpgrades(), Height: 5}
	id, err := ForkID(upgrade, common.Hash{1})
	require.NoError(t, err)
	again, err := ForkID(upgrade, common.Hash{1})
	require.NoError(t, err)
	require.Equal(t, id, again)

	for name, other := range map[string]struct {
		upgrade UpgradeHeight
		genesis common.Hash
	}{
		"another genesis":  {upgrade, common.Hash{2}},
		"another height":   {UpgradeHeight{Upgrades: GetSonicUpgrades(), Height: 6}, common.Hash{1}},
		"another upgrades": {UpgradeHeight{Upgrades: GetAllegroUpgrades(), Height: 5}, common.Hash{1}},
	} {
		otherID, err := ForkID(other.upgrade, other.genesis)
		require.NoError(t, err)
		require.NotEqual(t, id, otherID, name)
	}
}
