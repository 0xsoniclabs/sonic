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
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestThrottling_SkipEventEmission_DoNotSkipIfBelongingToDominantSet(t *testing.T) {

	tests := map[string]struct {
		validatorID idx.ValidatorID
		validators  *pos.Validators
	}{
		"validator is equivalent to dominant cutoff": {
			validatorID: 1,
			validators:  makeValidators(75, 25),
		},
		"validator belongs to dominant set": {
			validatorID: 2,
			validators: makeValidators(
				750, 750, // 75% owned by first two validators
				125, 125, 125, 125,
			),
		},
		"non-first validator belongs to dominant set": {
			validatorID: 2,
			validators: makeValidators(
				750, 750, // 75% owned by first two validators
				125, 125, 125, 125,
			),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(test.validators, idx.Epoch(0))
			world.EXPECT().GetRules().Return(opera.Rules{})
			world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0)).AnyTimes()

			state := NewThrottlingState(test.validatorID, 0.75, 3, world)

			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions().Return(types.Transactions{}).AnyTimes()
			event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()

			skip := state.SkipEventEmission(event)
			require.False(t, skip)
		})
	}
}

func TestThrottling_SkipEventEmission_SkipIfNotBelongingToDominantSet(t *testing.T) {

	stakes := makeValidators(
		750, 750, // 75% owned by first two validators
		125, 125, 125, 125,
	)

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0))
	world.EXPECT().GetRules().Return(opera.Rules{})
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0))

	state := NewThrottlingState(3, 0.75, 1, world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(types.Transactions{})
	event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()

	skip := state.SkipEventEmission(event)
	require.True(t, skip)
}

func TestThrottling_DoNotSkip_WhenEventCarriesTransactions(t *testing.T) {

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0))

	state := NewThrottlingState(3, 0.75, 0, world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(
		types.Transactions{types.NewTx(&types.LegacyTx{})})
	event.EXPECT().Frame()

	skip := state.SkipEventEmission(event)
	require.False(t, skip)
}

func TestThrottling_DoNotSkip_WhenEventBelongsToTheSameFrame(t *testing.T) {
	ctrl := gomock.NewController(t)
	// stakes are dominated by validators 1 and 2,
	// this allows validator 3 to be throttled for this test
	stakes := makeValidators(
		500, 300, 200,
	)

	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{
		Emitter: opera.EmitterRules{
			Interval: 170,
		},
	}).AnyTimes()
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0)).AnyTimes()

	// repeat test for a variety of maxRepeatedFrames settings
	for _, maxRepeatedFrames := range []uint{1, 2, 3, 4, 80} {

		state := NewThrottlingState(3, 0.75, maxRepeatedFrames, world)
		repeatedFrame := idx.Frame(7) // any frame number, repeatedly used

		for range maxRepeatedFrames {
			repeatedFrameEvent := inter.NewMockEventPayloadI(ctrl)
			repeatedFrameEvent.EXPECT().Transactions()
			repeatedFrameEvent.EXPECT().Frame().Return(repeatedFrame).Times(2)
			skip := state.SkipEventEmission(repeatedFrameEvent)
			require.True(t, skip)
		}

		oneTooManyEvent := inter.NewMockEventPayloadI(ctrl)
		oneTooManyEvent.EXPECT().Transactions()
		oneTooManyEvent.EXPECT().Frame().Return(repeatedFrame).Times(2)
		skip := state.SkipEventEmission(oneTooManyEvent)
		require.False(t, skip)
	}
}

func TestThrottling_DoNotSkip_IfTooManyBlocksAreSkipped(t *testing.T) {
	ctrl := gomock.NewController(t)

	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetRules().Return(
		opera.Rules{
			Economy: opera.EconomyRules{
				BlockMissedSlack: 50,
			},
		}).AnyTimes()
	world.EXPECT().GetEpochValidators().Return(makeValidators(
		500, 300, 200,
	), idx.Epoch(0)).AnyTimes()

	throttler := NewThrottlingState(3, 0.75, 10, world)
	throttler.lastEmissionBlockNumber = idx.Block(17)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().AnyTimes()
	event.EXPECT().Frame().Return(idx.Frame(2)).AnyTimes()

	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(17 + 50)).Times(2)
	skip := throttler.SkipEventEmission(event)
	require.False(t, skip, "Event missing so many blocks should NOT be skipped")

	// one more than the last time
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(17 + 50 + 1)).MinTimes(1)
	skip = throttler.SkipEventEmission(event)
	require.True(t, skip, "Event missing less than max allowed blocks should be skipped")
}
