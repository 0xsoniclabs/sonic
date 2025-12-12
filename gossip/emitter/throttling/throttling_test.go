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
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestThrottler_updateAttendance_DominatingValidatorsAreOffline_AfterShortTimeout(t *testing.T) {
	t.Parallel()

	const currentAttempt = 15

	type testCase struct {
		shortTimeout   attempt
		lastAttendance validatorAttendance
		expectedOnline bool
	}
	tests := make(map[string]testCase)
	for lastSeenAt := attempt(1); lastSeenAt <= currentAttempt; lastSeenAt++ {
		for _, shortTimeout := range []attempt{1, 2, 3, 4, 5} {
			tests[fmt.Sprintf(
				"lastSeenAt=%d shortTimeout=%d",
				lastSeenAt,
				shortTimeout,
			)] = testCase{
				shortTimeout: shortTimeout,
				lastAttendance: validatorAttendance{
					lastSeenSeq: 123,
					lastSeenAt:  lastSeenAt,
					online:      true,
				},
				expectedOnline: lastSeenAt+shortTimeout > currentAttempt,
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := testFakeWorld{
				1: makeTestEvent(123),
			}

			state := NewThrottlingState(
				13,   // non-existing validator ID, this test does not depend on it
				0.75, // this test is stake agnostic
				uint64(test.shortTimeout),
				100, // fix long timeout
				world,
			)
			state.attempt = currentAttempt
			state.attendanceList[1] = test.lastAttendance
			state.lastDominatingSet = makeSet(1)

			state.updateAttendance()

			attendance, found := state.attendanceList[1]
			online := found && attendance.online
			require.Equal(t, test.expectedOnline, online)
		})
	}
}

func TestThrottler_updateAttendance_SuppressedValidatorsAreOffline_AfterLongTimeout(t *testing.T) {
	t.Parallel()

	const currentAttempt = 101

	type testCase struct {
		longTimeout    attempt
		lastAttendance validatorAttendance
		expectedOnline bool
	}
	tests := make(map[string]testCase)
	for lastSeenAt := attempt(1); lastSeenAt <= currentAttempt; lastSeenAt++ {
		for _, longTimeout := range []attempt{1, 2, 3, 8, 10, 20, 50, 100} {

			tests[fmt.Sprintf(
				"lastSeenAt=%d longTimeout=%d",
				lastSeenAt,
				longTimeout,
			)] = testCase{
				longTimeout: longTimeout,
				lastAttendance: validatorAttendance{
					lastSeenSeq: 123,
					lastSeenAt:  lastSeenAt,
					online:      true,
				},
				expectedOnline: lastSeenAt+longTimeout > currentAttempt,
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := testFakeWorld{
				1: makeTestEvent(123),
			}

			state := NewThrottlingState(
				13,   // non-existing validator ID, this test does not depend on it
				0.75, // this test is stake agnostic
				3,    // fix short timeout
				uint64(test.longTimeout),
				world,
			)
			state.attempt = currentAttempt
			state.attendanceList[1] = test.lastAttendance

			state.updateAttendance()

			attendance, found := state.attendanceList[1]
			online := found && attendance.online
			require.Equal(t, test.expectedOnline, online)
		})
	}
}

func TestThrottler_updateAttendance_OfflineValidatorsComeBackOnlineWithAnyNewSeqNumber(t *testing.T) {
	t.Parallel()

	const currentAttempt = 100

	type testCase struct {
		longTimeout    attempt
		lastAttendance validatorAttendance
	}
	tests := make(map[string]testCase)
	for lastSeenAt := attempt(1); lastSeenAt <= currentAttempt; lastSeenAt++ {
		for _, shortTimeout := range []attempt{1, 2, 3, 8, 10} {
			for _, longTimeout := range []attempt{1, 2, 3, 8, 10} {

				tests[fmt.Sprintf(
					"lastSeenAt=%d longTimeout=%d shortTimeout=%d",
					lastSeenAt,
					longTimeout,
					shortTimeout,
				)] = testCase{
					longTimeout: shortTimeout,
					lastAttendance: validatorAttendance{
						lastSeenSeq: 122, // one less the event about to be seen
						lastSeenAt:  lastSeenAt,
						online:      false,
					},
				}
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := testFakeWorld{
				1: makeTestEvent(123),
			}

			state := NewThrottlingState(
				13,   // non-existing validator ID, this test does not depend on it
				0.75, // this test is stake agnostic
				3,
				uint64(test.longTimeout),
				world,
			)
			state.attempt = currentAttempt
			state.attendanceList[1] = test.lastAttendance

			state.updateAttendance()

			attendance, found := state.attendanceList[1]
			online := found && attendance.online
			require.True(t, online)
		})
	}

}

func makeTestEvent(seq idx.Event) inter.Event {
	builder := &inter.MutableEventPayload{}
	builder.SetSeq(seq)
	return builder.Build().Event
}

// testFakeWorld is a simple implementation of WorldReader for testing purposes.
// Is is a collection of last events per validator.
type testFakeWorld map[idx.ValidatorID]inter.Event

func (fw testFakeWorld) GetEpochValidators() (*pos.Validators, idx.Epoch) {
	builder := pos.NewBuilder()
	for id := range fw {
		builder.Set(id, 100)
	}
	return builder.Build(), idx.Epoch(0)
}

func (fw testFakeWorld) GetLastEvent(validatorID idx.ValidatorID) *inter.Event {
	event, found := fw[validatorID]
	if !found {
		return nil
	}
	return &event
}

func (fw testFakeWorld) GetRules() opera.Rules {
	return opera.Rules{}
}

func TestThrottling_CanSkipEventEmission_DoNotSkipIfBelongingToDominantSet(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			ctrl := gomock.NewController(t)

			lastEvent := makeTestEvent(123)

			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(test.validators, idx.Epoch(0)).MinTimes(1)
			world.EXPECT().GetRules().Return(opera.Rules{
				Economy: opera.EconomyRules{
					BlockMissedSlack: 50,
				},
			})
			world.EXPECT().GetLastEvent(gomock.Any()).Return(&lastEvent).MinTimes(1)

			state := NewThrottlingState(test.validatorID, 0.75, 3, 10, world)

			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions().Return(types.Transactions{})
			event.EXPECT().SelfParent().Return(&hash.Event{1}).MinTimes(1)

			skip := state.CanSkipEventEmission(event)
			require.Equal(t, DoNotSkipEvent_DominantStake, skip)
		})
	}
}

func TestThrottling_SkipEventEmission_SkipIfNotBelongingToDominantSet(t *testing.T) {
	t.Parallel()

	stakes := makeValidators(
		750, 750, // 75% owned by first two validators
		125, 125, 125, 125,
	)

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{
		Economy: opera.EconomyRules{
			BlockMissedSlack: 50,
		},
	})
	lastEvent := makeTestEvent(123)
	world.EXPECT().GetLastEvent(gomock.Any()).Return(&lastEvent).AnyTimes()

	state := NewThrottlingState(3, 0.75, 1, 10, world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(types.Transactions{})
	event.EXPECT().SelfParent().Return(&hash.Event{1}).MinTimes(1)

	skip := state.CanSkipEventEmission(event)
	require.Equal(t, SkipEventEmission, skip)
}

func TestThrottling_DoNotSkip_WhenEventCarriesTransactions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(makeValidators(500, 300, 200), idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{}).AnyTimes()
	lastEvent := makeTestEvent(123)
	world.EXPECT().GetLastEvent(gomock.Any()).Return(&lastEvent).AnyTimes()

	state := NewThrottlingState(3, 0.75, 0, 10, world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(
		types.Transactions{types.NewTx(&types.LegacyTx{})})
	event.EXPECT().SelfParent().Return(&hash.Event{42}).AnyTimes()

	skip := state.CanSkipEventEmission(event)
	require.Equal(t, DoNotSkipEvent_CarriesTransactions, skip)
}

func TestThrottling_DoNotSkip_GenesisEvents(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(makeValidators(500, 300, 200), idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{}).AnyTimes()
	lastEvent := makeTestEvent(123)
	world.EXPECT().GetLastEvent(gomock.Any()).Return(&lastEvent).AnyTimes()

	state := NewThrottlingState(3, 0.75, 0, 10, world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions()
	event.EXPECT().SelfParent().MinTimes(1)

	skip := state.CanSkipEventEmission(event)
	require.Equal(t, DoNotSkipEvent_Genesis, skip)
}

func TestThrottling_DoNotSkip_RespectHeartbeatEvents(t *testing.T) {
	t.Parallel()

	for _, longTimeout := range []uint64{4, 5, 10} {
		t.Run(fmt.Sprintf("longTimeout=%d", longTimeout),
			func(t *testing.T) {
				t.Parallel()

				validators := makeValidators(10, 10, 10, 10) // one suppressed validator

				ctrl := gomock.NewController(t)

				world := NewMockWorldReader(ctrl)
				world.EXPECT().GetRules().Return(opera.Rules{
					Economy: opera.EconomyRules{
						BlockMissedSlack: 1000, // large enough to not interfere with this test
					},
				}).AnyTimes()
				world.EXPECT().GetEpochValidators().
					Return(validators, idx.Epoch(0)).AnyTimes()
				otherPeersEvents := makeTestEvent(1)
				world.EXPECT().GetLastEvent(gomock.Any()).Return(&otherPeersEvents).Times(int(validators.Len())).AnyTimes()

				throttler := NewThrottlingState(
					4,    // last validator, suppressed
					0.75, // first three validators dominate stake
					1000, // large enough to not interfere with this test
					longTimeout,
					world)

				// Event 1 should be considered a heartbeat
				event := inter.NewMockEventPayloadI(ctrl)
				event.EXPECT().Transactions().Return(types.Transactions{}).AnyTimes()
				event.EXPECT().SelfParent().Return(&hash.Event{42}).AnyTimes()

				for range int(longTimeout)/2 - 1 {
					skip := throttler.CanSkipEventEmission(event)
					require.Equal(t, SkipEventEmission, skip)
				}

				skip := throttler.CanSkipEventEmission(event)
				require.Equal(t, DoNotSkipEvent_Heartbeat, skip)

				// one more attempt, should be skipped again
				skip = throttler.CanSkipEventEmission(event)
				require.Equal(t, SkipEventEmission, skip)
			})
	}
}

// func TestThrottler_filterOfflineValidators_preservesStakesAndIdsOfOnlineValidators(t *testing.T) {

// 	tests := map[string]struct {
// 		validators         *pos.Validators
// 		onlineValidatorIDs []idx.ValidatorID
// 	}{}

// 	for i := range 10 {
// 		for j := range i {

// 			name := fmt.Sprintf("%d validators, %d online", i, j)
// 			validators := makeValidators(slices.Repeat([]int64{100}, i)...)
// 			onlineIDs := make([]idx.ValidatorID, 0, j)
// 			for k := 0; k < j; k++ {
// 				onlineIDs = append(onlineIDs, idx.ValidatorID(k+1))
// 			}
// 			tests[name] = struct {
// 				validators         *pos.Validators
// 				onlineValidatorIDs []idx.ValidatorID
// 			}{
// 				validators:         validators,
// 				onlineValidatorIDs: onlineIDs,
// 			}
// 		}
// 	}

// 	for name, test := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			ctrl := gomock.NewController(t)

// 			epoch := idx.Epoch(0)

// 			world := NewMockWorldReader(ctrl)

// 			for _, id := range test.validators.IDs() {
// 				if slices.Contains(test.onlineValidatorIDs, id) {
// 					eventHash, event := createTestEventWithFrame(idx.Frame(10))
// 					world.EXPECT().GetLastEvent(epoch, id).Return(&eventHash)
// 					world.EXPECT().GetEvent(eventHash).Return(&event)
// 				} else {
// 					world.EXPECT().GetLastEvent(epoch, id)
// 				}
// 			}

// 			event := inter.NewMockEventPayloadI(ctrl)
// 			event.EXPECT().SelfParent().Return(&hash.Event{1})
// 			event.EXPECT().Frame().Return(idx.Frame(11)).AnyTimes()

// 			throttler := NewThrottlingState(0, 0.8, 0, 6, world)

// 			onlineSet := throttler.filterOfflineValidators(
// 				test.validators, event, epoch)

// 			require.ElementsMatch(t, test.onlineValidatorIDs, onlineSet.IDs())

// 			accumulatedStake := pos.Weight(0)
// 			for _, id := range test.onlineValidatorIDs {
// 				require.Equal(t,
// 					test.validators.Get(id),
// 					onlineSet.Get(id),
// 					"stake for online validator %d should match",
// 					id)
// 				accumulatedStake += onlineSet.Get(id)
// 			}
// 			require.Equal(t, accumulatedStake, onlineSet.TotalWeight(),
// 				"total stake of online set should match sum of individual stakes")
// 		})
// 	}
// }

func makeSet(ids ...idx.ValidatorID) dominantSet {
	res := make(dominantSet)
	for _, id := range ids {
		res[id] = struct{}{}
	}
	return res
}
