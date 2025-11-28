package emitter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitterConfig_ValidateConfig_ReportsError_ForInvalidDominatingThreshold(t *testing.T) {

	cfg := DefaultConfig()
	for _, value := range []float64{-0.1, -1, 1.1, 2} {
		cfg.ThrottleDominantThreshold = value

		require.Error(t, cfg.Validate())
	}
}
