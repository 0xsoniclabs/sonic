package opera

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVmConfig_SingleProposerModeDisablesExcessGasCharging(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("SingleProposerModeEnabled=%t", enabled), func(t *testing.T) {
			require := require.New(t)
			rules := Rules{
				Upgrades: Upgrades{
					SingleProposerBlockFormation: enabled,
				},
			}

			vmConfig := GetVmConfig(rules)

			require.NotEqual(
				enabled,
				vmConfig.ChargeExcessGas,
				"Expected ChargeExcessGas to be %t when SingleProposerBlockFormation is %t",
				!enabled,
				enabled,
			)
		})
	}
}
