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

package ethapi

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetVmConfig_RetrievesVmConfigFromRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	ctx := t.Context()
	height := idx.Block(12)

	rules := opera.Rules{}
	backend.EXPECT().GetNetworkRules(ctx, height).Return(&rules, nil).AnyTimes()

	config, err := GetVmConfig(ctx, backend, 12)
	require.NoError(t, err)

	// The config contains functions that can not be compared with require.Equal,
	// so we compare a debug print of the config instead.
	want := fmt.Sprintf("%+v", opera.GetVmConfig(rules))
	got := fmt.Sprintf("%+v", config)
	require.Equal(t, want, got)
}

func TestGetVmConfig_ForwardsErrorFromRuleRetrieval(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	any := gomock.Any()
	expectedErr := fmt.Errorf("injected error")
	backend.EXPECT().GetNetworkRules(any, any).Return(nil, expectedErr).AnyTimes()

	config, err := GetVmConfig(t.Context(), backend, 12)
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, vm.Config{}, config)
}

func TestGetVmConfig_ProducesAnErrorIfRulesAreNotAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	any := gomock.Any()
	backend.EXPECT().GetNetworkRules(any, any).Return(nil, nil).AnyTimes()

	config, err := GetVmConfig(t.Context(), backend, 12)
	require.ErrorContains(t, err, "no network rules found for block height 12")
	require.Equal(t, vm.Config{}, config)
}
