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
