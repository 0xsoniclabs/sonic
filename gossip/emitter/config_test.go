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

package emitter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitterConfig_ValidateConfig_ReportsError_ForInvalidDominatingThreshold(t *testing.T) {
	cfg := DefaultConfig()
	for _, value := range []float64{-0.1, -1, 0.69, 1.1, 2} {
		cfg.ThrottlerDominantThreshold = value
		require.Error(t, cfg.Validate())
	}
}

func TestEmitterConfig_ValidateConfig_ReportsError_ForInvalidSkipInSameFrame(t *testing.T) {
	cfg := DefaultConfig()
	for _, value := range []uint{0, 1} {
		cfg.ThrottlerSkipInSameFrame = value
		require.Error(t, cfg.Validate())
	}
}

func TestEmitterConfig_ValidateConfig_ReturnsNil_ForValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	validDominatingThresholds := []float64{0.7, 0.75, 0.8, 0.85, 0.9, 0.95, 1.0}
	validSkipInSameFrame := []uint{2, 3, 4, 5, 10, 20}

	for _, domThreshold := range validDominatingThresholds {
		for _, skipInSameFrame := range validSkipInSameFrame {
			cfg.ThrottlerDominantThreshold = domThreshold
			cfg.ThrottlerSkipInSameFrame = skipInSameFrame
			require.Nil(t, cfg.Validate())
		}
	}
}
