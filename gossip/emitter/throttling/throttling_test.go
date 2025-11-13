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

package throttling

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestThrottling_SkippEventEmission_DoNotSkipIfBelongingToDominantSet(t *testing.T) {

	tests := map[string]struct {
		validatorID idx.ValidatorID
		stakes      *pos.Validators
	}{
		"validator is equivalent to dominant cuttoff": {
			validatorID: 1,
			stakes:      makeValidators(75, 25),
		},
		"validator belongs to dominant set": {
			validatorID: 2,
			stakes: makeValidators(
				750, 750, // 75% owned by first two validators
				125, 125, 125, 125,
			),
		},
		"non-first validator belongs to dominant set": {
			validatorID: 2,
			stakes: makeValidators(
				750, 750, // 75% owned by first two validators
				125, 125, 125, 125,
			),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			state := NewThrottlingState(test.validatorID, 0.75)
			state.OnNewEpoch(test.stakes, opera.Rules{})

			ctrl := gomock.NewController(t)
			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions().Return(types.Transactions{}).AnyTimes()
			event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()
			event.EXPECT().CreationTime().Return(inter.Timestamp(1000)).AnyTimes()

			skip := state.SkipEventEmission(event)

			require.False(t, skip)
		})
	}
}

func TestThrottling_SkippEventEmission_SkipIfNotBelongingToDominantSet(t *testing.T) {

	stakes := makeValidators(
		750, 750, // 75% owned by first two validators
		125, 125, 125, 125,
	)

	state := NewThrottlingState(3, 0.75)
	state.OnNewEpoch(stakes, opera.Rules{})

	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(types.Transactions{})
	event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()
	event.EXPECT().CreationTime().Return(inter.Timestamp(1000))

	skip := state.SkipEventEmission(event)

	require.True(t, skip)
}

func TestThrottling_UniformStakeDistribution_EventsAreNotSkipped(t *testing.T) {

	stakes := makeValidators(
		200, 200, 200, 200, 200,
	)

	for _, id := range stakes.IDs() {
		t.Run(fmt.Sprintf("validator=%d", id), func(t *testing.T) {
			state := NewThrottlingState(id, 120.0) // 120% threshold to force all validators being in dominant set
			state.OnNewEpoch(stakes, opera.Rules{})

			ctrl := gomock.NewController(t)
			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions().Return(types.Transactions{})
			event.EXPECT().Frame().Return(idx.Frame(1)).Times(2)
			event.EXPECT().CreationTime().Return(inter.Timestamp(1000)).Times(2)

			skip := state.SkipEventEmission(event)

			require.False(t, skip)
		})
	}
}

func TestThrottling_DoNotSkip_WhenEventCarriesTransactions(t *testing.T) {
	stakes := makeValidators(
		500, 300, 200,
	)

	state := NewThrottlingState(3, 0.75)
	state.OnNewEpoch(stakes, opera.Rules{})

	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(
		types.Transactions{types.NewTx(&types.LegacyTx{})})
	event.EXPECT().Frame()
	event.EXPECT().CreationTime()

	skip := state.SkipEventEmission(event)
	require.False(t, skip)
}

func TestThrottling_DoNotSkip_WhenEventBelongsToTheSameFrame(t *testing.T) {
	ctrl := gomock.NewController(t)
	stakes := makeValidators(
		500, 300, 200,
	)

	state := NewThrottlingState(3, 0.75)
	state.OnNewEpoch(stakes, opera.Rules{})

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(
		types.Transactions{types.NewTx(&types.LegacyTx{})})
	event.EXPECT().Frame().Return(idx.Frame(1))
	event.EXPECT().CreationTime()

	// First event to set the last frame
	// Cannot be skipped because it carries transactions
	skip := state.SkipEventEmission(event)
	require.False(t, skip)

	// Second event in the same frame
	event2 := inter.NewMockEventPayloadI(ctrl)
	event2.EXPECT().Transactions()
	event2.EXPECT().Frame().Return(idx.Frame(1)).Times(2)
	event2.EXPECT().CreationTime()

	skip = state.SkipEventEmission(event2)
	require.False(t, skip)
}

func TestThrottling_DoNotSkip_IfTimestampDeltaIsTooLarge(t *testing.T) {

	tests := map[string]struct {
		stalledInterval inter.Timestamp
		timeDelta       inter.Timestamp
		expectSkip      bool
	}{
		"delta less than 1/2 stalled interval": {
			stalledInterval: 100,
			timeDelta:       50,
			expectSkip:      false,
		},
		"delta equal to 1/2 stalled interval": {
			stalledInterval: 100,
			timeDelta:       50,
			expectSkip:      false,
		},
		"delta greater than 1/2 stalled interval": {
			stalledInterval: 100,
			timeDelta:       51,
			expectSkip:      true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			stakes := makeValidators(
				500, 300, 200,
			)

			state := NewThrottlingState(3, 0.75)
			state.OnNewEpoch(stakes, opera.Rules{
				Emitter: opera.EmitterRules{
					StalledInterval: test.stalledInterval,
				},
			})
			state.lastEventTime = inter.Timestamp(1000)
			state.lastEventFrame = idx.Frame(1)

			times := 2
			if test.expectSkip {
				times = 1
			}

			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions()
			event.EXPECT().Frame().Return(idx.Frame(2)).Times(times)
			event.EXPECT().CreationTime().Return(inter.Timestamp(1000 + test.timeDelta)).Times(times)

			skip := state.SkipEventEmission(event)
			require.Equal(t, test.expectSkip, skip)
		})
	}
}
