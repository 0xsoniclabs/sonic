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
