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

package opera

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVmConfig_SingleProposerModeDisablesExcessGasCharging(t *testing.T) {
	for _, singleProposerMode := range []bool{true, false} {
		t.Run(fmt.Sprintf("SingleProposerModeEnabled=%t", singleProposerMode), func(t *testing.T) {
			require := require.New(t)
			rules := Rules{
				Upgrades: Upgrades{
					SingleProposerBlockFormation: singleProposerMode,
				},
			}

			vmConfig := GetVmConfig(rules)

			require.NotEqual(singleProposerMode, vmConfig.ChargeExcessGas)
		})
	}
}
