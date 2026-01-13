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

package throttler

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestThrottling_CanSkipEventEmission_SkipEmission_WhenValidatorIsNonDominant(t *testing.T) {
	t.Parallel()

	stakes := makeValidatorsFromStakes(
		750, 750, // 75% owned by first two validators
		125, 125, 125, 125,
	)

	// iterate over non-dominant validator IDs
	for _, id := range []idx.ValidatorID{3, 4, 5, 6} {
		t.Run(fmt.Sprintf("validatorID=%d", id), func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0)).AnyTimes()
			world.EXPECT().GetRules().Return(opera.Rules{
				Economy: opera.EconomyRules{
					BlockMissedSlack: 50,
				},
			})
			lastEvent := makeEventWithSeq(123)
			world.EXPECT().GetLastEvent(gomock.Any()).Return(lastEvent).AnyTimes()

			state := NewThrottlingState(
				id,
				config.ThrottlerConfig{
					Enabled:                true,
					DominantStakeThreshold: 0.75, // this test is stake agnostic
					DominatingTimeout:      1,
					NonDominatingTimeout:   10,
				},
				world)

			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions().Return(types.Transactions{})
			event.EXPECT().SelfParent().Return(&hash.Event{1}).MinTimes(1)

			skip := state.CanSkipEventEmission(event)
			require.Equal(t, SkipEventEmission, skip)
		})
	}
}

func TestThrottling_CanSkipEventEmission_DoNotSkip_WhenValidatorIsPartOfTheDominantSet(t *testing.T) {
	t.Parallel()

	stakes := makeValidatorsFromStakes(10, 10, 10)

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{
		Economy: opera.EconomyRules{
			BlockMissedSlack: 50,
		},
	})
	lastEvent := makeEventWithSeq(123)
	world.EXPECT().GetLastEvent(gomock.Any()).Return(lastEvent).AnyTimes()

	state := NewThrottlingState(3,
		config.ThrottlerConfig{
			Enabled:                true,
			DominantStakeThreshold: 0.75, // this test is stake agnostic
			DominatingTimeout:      1,
			NonDominatingTimeout:   10,
		},
		world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(types.Transactions{})
	event.EXPECT().SelfParent().Return(&hash.Event{1}).MinTimes(1)

	skip := state.CanSkipEventEmission(event)
	require.Equal(t, DoNotSkipEvent_DominantStake, skip)
}

func TestThrottling_CanSkipEventEmission_DoNotSkip_WhenValidatorBelongsToDominantSet(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		validatorID idx.ValidatorID
		validators  *pos.Validators
	}{
		"validator stake is equivalent to dominant threshold": {
			validatorID: 1,
			validators:  makeValidatorsFromStakes(75, 25),
		},
		"validator belongs to dominant set": {
			validatorID: 1,
			validators: makeValidatorsFromStakes(
				750, 750, // 75% owned by first two validators
				125, 125, 125, 125,
			),
		},
		"non-first validator belongs to dominant set": {
			validatorID: 2,
			validators: makeValidatorsFromStakes(
				750, 750, // 75% owned by first two validators
				125, 125, 125, 125,
			),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			lastEvent := makeEventWithSeq(123)

			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(test.validators, idx.Epoch(0)).MinTimes(1)
			world.EXPECT().GetRules().Return(opera.Rules{
				Economy: opera.EconomyRules{
					BlockMissedSlack: 50,
				},
			})
			world.EXPECT().GetLastEvent(gomock.Any()).Return(lastEvent).MinTimes(1)

			state := NewThrottlingState(
				test.validatorID,
				config.ThrottlerConfig{
					Enabled:                true,
					DominantStakeThreshold: 0.75,
					DominatingTimeout:      3,
					NonDominatingTimeout:   10,
				},
				world)

			event := inter.NewMockEventPayloadI(ctrl)
			event.EXPECT().Transactions().Return(types.Transactions{})
			event.EXPECT().SelfParent().Return(&hash.Event{1}).MinTimes(1)

			skip := state.CanSkipEventEmission(event)
			require.Equal(t, DoNotSkipEvent_DominantStake, skip)
		})
	}
}

func TestThrottling_CanSkipEventEmission_DoNotSkip_WhenValidatorHasNotParticipatedInBlocksForTooLong(t *testing.T) {
	t.Parallel()

	// this test assumes local validator is non-dominant, use id 4
	validatorId := idx.ValidatorID(4)
	validators := makeValidatorsFromStakes(10, 10, 10, 10)

	for nonDominatingTimeout := config.Attempt(0); nonDominatingTimeout <= 100; nonDominatingTimeout += 5 {
		for blockMissedSlack := idx.Block(0); blockMissedSlack <= 100; blockMissedSlack += 10 {
			t.Run(fmt.Sprintf("nonDominatingTimeout=%d,blockMissedSlack=%d",
				nonDominatingTimeout, blockMissedSlack), func(t *testing.T) {
				t.Parallel()

				rules := opera.Rules{
					Economy: opera.EconomyRules{
						BlockMissedSlack: blockMissedSlack,
					},
				}
				throttlerConfig := config.ThrottlerConfig{
					Enabled:                true,
					DominantStakeThreshold: 0.75,
					DominatingTimeout:      0,
					NonDominatingTimeout:   nonDominatingTimeout,
				}

				ctrl := gomock.NewController(t)
				world := NewMockWorldReader(ctrl)
				world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()
				world.EXPECT().GetRules().Return(rules).AnyTimes()

				state := NewThrottlingState(validatorId, throttlerConfig, world)

				event := inter.NewMockEventPayloadI(ctrl)
				event.EXPECT().Transactions().Return(types.Transactions{}).AnyTimes()
				event.EXPECT().SelfParent().Return(&hash.Event{42}).AnyTimes()

				// heartbeat timeout is the minimum between
				// half of NonDominatingTimeout and half of BlockMissedSlack
				timeout := min(
					config.Attempt(rules.Economy.BlockMissedSlack)/2,
					state.config.NonDominatingTimeout/2,
				)

				attempt := 0
				for range int(timeout) - 1 {

					keepNodesOnline(validators, attempt, world)

					skip := state.CanSkipEventEmission(event)
					require.Equal(t, SkipEventEmission, skip)
					attempt++
				}

				keepNodesOnline(validators, attempt, world)

				// after timeout attempts, heartbeat should be respected
				skip := state.CanSkipEventEmission(event)
				require.Equal(t, DoNotSkipEvent_Heartbeat, skip)
				attempt++

				// for all block slack which is larger than 1 attempt,
				// the next attempt should be skipped again
				if timeout > 1 {

					keepNodesOnline(validators, attempt, world)

					skip = state.CanSkipEventEmission(event)
					require.Equal(t, SkipEventEmission, skip)
				}
			})
		}
	}
}

// keepNodesOnline makes all validators appear online, this helps to isolate
// the heartbeat logic in tests.
func keepNodesOnline(validators *pos.Validators, attempt int, world *MockWorldReader) {
	for _, id := range validators.IDs() {
		// all validators return next event, to be considered online
		lastEvent := makeEventWithSeq(1 + idx.Event(attempt+1))
		world.EXPECT().GetLastEvent(id).Return(lastEvent)
	}
}

func TestThrottling_CanSkipEventEmission_DoNotSkip_WhenEventCarriesTransactions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(makeValidatorsFromStakes(500, 300, 200), idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{}).AnyTimes()
	lastEvent := makeEventWithSeq(123)
	world.EXPECT().GetLastEvent(gomock.Any()).Return(lastEvent).AnyTimes()

	state := NewThrottlingState(3,
		config.ThrottlerConfig{
			Enabled:                true,
			DominantStakeThreshold: 0.75,
			DominatingTimeout:      0,
			NonDominatingTimeout:   10,
		},
		world)

	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Transactions().Return(
		types.Transactions{types.NewTx(&types.LegacyTx{})})
	event.EXPECT().SelfParent().Return(&hash.Event{42}).AnyTimes()

	skip := state.CanSkipEventEmission(event)
	require.Equal(t, DoNotSkipEvent_CarriesTransactions, skip)
}

func TestThrottling_CanSkipEventEmission_DoNotSkip_GenesisEvents(t *testing.T) {
	t.Parallel()

	// All validators shall emit, including non-dominant ones
	validators := makeValidatorsFromStakes(500, 300, 200)

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetRules().Return(opera.Rules{}).AnyTimes()
	lastEvent := makeEventWithSeq(123)
	world.EXPECT().GetLastEvent(gomock.Any()).Return(lastEvent).AnyTimes()

	for id := idx.ValidatorID(1); id <= idx.ValidatorID(validators.Len()); id++ {

		state := NewThrottlingState(id,
			config.ThrottlerConfig{
				Enabled:                true,
				DominantStakeThreshold: 0.75,
				DominatingTimeout:      0,
				NonDominatingTimeout:   10,
			},
			world)

		event := inter.NewMockEventPayloadI(ctrl)
		event.EXPECT().Transactions()
		event.EXPECT().SelfParent().MinTimes(1)

		skip := state.CanSkipEventEmission(event)
		require.Equal(t, DoNotSkipEvent_Genesis, skip)
	}
}

func TestThrottling_ResetState_ZeroesStateValues(t *testing.T) {
	t.Parallel()

	state := NewThrottlingState(1, config.ThrottlerConfig{}, nil)

	state.attempt = 100
	state.lastEmission = 100
	state.attendanceList = attendanceList{
		attendance: map[idx.ValidatorID]validatorAttendance{
			1: {},
			2: {},
			3: {},
		},
	}

	state.resetState()

	require.Zero(t, state.attempt)
	require.Zero(t, state.lastEmission)
	require.Empty(t, state.attendanceList.attendance)
}

func TestThrottling_CanSkipEventEmission_DoNotSkip_RespectHeartbeatEvents(t *testing.T) {
	t.Parallel()

	for _, NonDominatingTimeout := range []config.Attempt{4, 5, 10, 25} {
		t.Run(fmt.Sprintf("NonDominatingTimeout=%d", NonDominatingTimeout),
			func(t *testing.T) {
				t.Parallel()

				validators := makeValidatorsFromStakes(10, 10, 10, 10) // one suppressed validator

				ctrl := gomock.NewController(t)

				world := NewMockWorldReader(ctrl)
				world.EXPECT().GetRules().Return(opera.Rules{
					Economy: opera.EconomyRules{
						BlockMissedSlack: 1000, // large enough to not interfere with this test
					},
				}).AnyTimes()
				world.EXPECT().GetEpochValidators().
					Return(validators, idx.Epoch(0)).AnyTimes()
				otherPeersEvents := makeEventWithSeq(1)
				world.EXPECT().GetLastEvent(gomock.Any()).Return(otherPeersEvents).Times(int(validators.Len())).AnyTimes()

				throttler := NewThrottlingState(
					4, // last validator, suppressed
					config.ThrottlerConfig{
						Enabled:                true,
						DominantStakeThreshold: 0.75, // first three validators dominate stake
						DominatingTimeout:      1000, // large enough to not interfere with this test
						NonDominatingTimeout:   NonDominatingTimeout,
					},
					world)

				// Event 1 should be considered a heartbeat
				event := inter.NewMockEventPayloadI(ctrl)
				event.EXPECT().Transactions().Return(types.Transactions{}).AnyTimes()
				event.EXPECT().SelfParent().Return(&hash.Event{42}).AnyTimes()

				// events in between should be skipped
				for range int(NonDominatingTimeout)/2 - 1 {
					skip := throttler.CanSkipEventEmission(event)
					require.Equal(t, SkipEventEmission, skip)
				}

				// after NonDominatingTimeout attempts, heartbeat should be respected
				skip := throttler.CanSkipEventEmission(event)
				require.Equal(t, DoNotSkipEvent_Heartbeat, skip)

				// one more attempt, should be skipped again
				skip = throttler.CanSkipEventEmission(event)
				require.Equal(t, SkipEventEmission, skip)
			})
	}
}

func TestThrottling_FeatureCanBeDisabled(t *testing.T) {

	t.Parallel()

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)

	throttler := NewThrottlingState(
		4, // last validator, suppressed
		config.ThrottlerConfig{
			Enabled: false,
		},
		world)

	// Event 1 should be considered a heartbeat
	event := inter.NewMockEventPayloadI(ctrl)

	skip := throttler.CanSkipEventEmission(event)
	require.Equal(t, DoNotSkipEvent_ThrottlerDisabled, skip)
}

func TestThrottler_updateAttendance_DominatingValidatorsAreOffline_AfterDominatingTimeout(t *testing.T) {
	t.Parallel()

	const currentAttempt = 15

	type testCase struct {
		dominatingTimeout config.Attempt
		lastAttendance    validatorAttendance
		expectedOnline    bool
	}
	tests := make(map[string]testCase)
	for lastSeenAt := config.Attempt(1); lastSeenAt <= currentAttempt; lastSeenAt++ {
		for _, DominatingTimeout := range []config.Attempt{1, 2, 3, 4, 5} {
			tests[fmt.Sprintf(
				"lastSeenAt=%d stalledTimeout=%d",
				lastSeenAt,
				DominatingTimeout,
			)] = testCase{
				dominatingTimeout: DominatingTimeout,
				lastAttendance: validatorAttendance{
					lastSeenSeq: 123,
					lastSeenAt:  lastSeenAt,
					online:      true,
				},
				expectedOnline: lastSeenAt+DominatingTimeout > currentAttempt,
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			validators := makeValidatorsFromStakes(100)
			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0))
			world.EXPECT().GetLastEvent(idx.ValidatorID(1)).Return(makeEventWithSeq(123))

			config := config.ThrottlerConfig{
				Enabled:                true,
				DominantStakeThreshold: 0.75,
				DominatingTimeout:      test.dominatingTimeout,
				NonDominatingTimeout:   100, // fix long timeout
			}

			attendanceList := newAttendanceList()
			attendanceList.attendance[1] = test.lastAttendance
			lastDominantSet := makeSet(1)

			attendanceList.updateAttendance(world, config, lastDominantSet, currentAttempt)

			require.Equal(t, test.expectedOnline, attendanceList.isOnline(1))
		})
	}
}

func TestThrottler_updateAttendance_NonDominantValidatorsAreOffline_AfterNonDominatingTimeout(t *testing.T) {
	t.Parallel()

	const currentAttempt = 101

	type testCase struct {
		NonDominatingTimeout config.Attempt
		lastAttendance       validatorAttendance
		expectedOnline       bool
	}
	tests := make(map[string]testCase)
	for lastSeenAt := config.Attempt(1); lastSeenAt <= currentAttempt; lastSeenAt++ {
		for _, NonDominatingTimeout := range []config.Attempt{1, 2, 3, 8, 10, 20, 50, 100} {

			tests[fmt.Sprintf(
				"lastSeenAt=%d NonDominatingTimeout=%d",
				lastSeenAt,
				NonDominatingTimeout,
			)] = testCase{
				NonDominatingTimeout: NonDominatingTimeout,
				lastAttendance: validatorAttendance{
					lastSeenSeq: 123,
					lastSeenAt:  lastSeenAt,
					online:      true,
				},
				expectedOnline: lastSeenAt+NonDominatingTimeout > currentAttempt,
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			validators := makeValidatorsFromStakes(100)
			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0))
			world.EXPECT().GetLastEvent(idx.ValidatorID(1)).Return(makeEventWithSeq(123))

			config := config.ThrottlerConfig{
				Enabled:                true,
				DominantStakeThreshold: 0.75, // this test is stake agnostic
				DominatingTimeout:      3,    // fix short timeout
				NonDominatingTimeout:   test.NonDominatingTimeout,
			}

			attendanceList := newAttendanceList()
			attendanceList.attendance[1] = test.lastAttendance

			// notice empty lastDominantSet - validator is suppressed
			attendanceList.updateAttendance(world, config, nil, currentAttempt)

			require.Equal(t, test.expectedOnline, attendanceList.isOnline(1))
		})
	}
}

func TestThrottler_updateAttendance_OfflineValidatorsComeBackOnlineWithAnyNewSeqNumber(t *testing.T) {
	t.Parallel()

	const currentAttempt = 100
	const lastSeenSeq = 122

	type testCase struct {
		DominatingTimeout    config.Attempt
		NonDominatingTimeout config.Attempt
		lastAttendance       validatorAttendance
	}
	tests := make(map[string]testCase)
	for lastSeenAt := config.Attempt(1); lastSeenAt <= currentAttempt; lastSeenAt++ {
		for _, DominatingTimeout := range []config.Attempt{1, 2, 3, 8, 10} {
			for _, NonDominatingTimeout := range []config.Attempt{1, 2, 3, 8, 10} {

				tests[fmt.Sprintf(
					"lastSeenAt=%d NonDominatingTimeout=%d DominatingTimeout=%d",
					lastSeenAt,
					NonDominatingTimeout,
					DominatingTimeout,
				)] = testCase{
					DominatingTimeout:    DominatingTimeout,
					NonDominatingTimeout: NonDominatingTimeout,
					lastAttendance: validatorAttendance{
						lastSeenSeq: lastSeenSeq,
						lastSeenAt:  lastSeenAt,
					},
				}
			}
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			validators := makeValidatorsFromStakes(100)
			world := NewMockWorldReader(ctrl)
			world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0))
			world.EXPECT().GetLastEvent(idx.ValidatorID(1)).
				Return(makeEventWithSeq(lastSeenSeq + 1))

			config := config.ThrottlerConfig{
				Enabled:                true,
				DominantStakeThreshold: 0.75,
				DominatingTimeout:      test.DominatingTimeout,
				NonDominatingTimeout:   test.NonDominatingTimeout,
			}

			attendanceList := newAttendanceList()
			attendanceList.attendance[1] = test.lastAttendance
			require.False(t, attendanceList.isOnline(1))

			// notice empty lastDominantSet - offline validator can not have dominant stake
			attendanceList.updateAttendance(world, config, nil, currentAttempt)

			require.True(t, attendanceList.isOnline(1))
		})
	}
}

func TestThrottler_updateAttendance_ValidatorsRemainOffline_IfNoEventIsReceived(t *testing.T) {

	ctrl := gomock.NewController(t)
	validators := makeValidatorsFromStakes(100)
	world := NewMockWorldReader(ctrl)
	world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0))
	world.EXPECT().GetLastEvent(idx.ValidatorID(1))

	config := config.ThrottlerConfig{
		Enabled:                true,
		DominantStakeThreshold: 0.75,
		DominatingTimeout:      3,
		NonDominatingTimeout:   100,
	}

	attendanceList := newAttendanceList()

	// notice empty lastDominantSet - offline validator can not have dominant stake
	attendanceList.updateAttendance(world, config, nil, 15)

	require.False(t, attendanceList.isOnline(1))
}

func Test_AttendanceList_DoesNotFlipFlop(t *testing.T) {
	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)

	validators := makeValidatorsFromStakes(1)
	world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()
	world.EXPECT().GetLastEvent(gomock.Any()).Return(makeEventWithSeq(1)).Times(5)

	config := config.ThrottlerConfig{
		DominatingTimeout:    3,
		NonDominatingTimeout: 10,
	}

	list := newAttendanceList()
	list.attendance = map[idx.ValidatorID]validatorAttendance{
		1: {
			lastSeenSeq: 1,
			lastSeenAt:  0,
			online:      true,
		},
	}

	dominantSet := makeSet(1)

	// Initially the dominant validator is online.
	list.updateAttendance(world, config, dominantSet, 1)
	require.True(t, list.isOnline(1))
	list.updateAttendance(world, config, dominantSet, 2)
	require.True(t, list.isOnline(1))

	// After 3 attempts, the dominant validator is considered offline.
	list.updateAttendance(world, config, dominantSet, 3)
	require.False(t, list.isOnline(1))

	// The validator is no longer dominating.
	dominantSet = makeSet()

	// But it should stay offline until it makes progress.
	list.updateAttendance(world, config, dominantSet, 4)
	require.False(t, list.isOnline(1))
	list.updateAttendance(world, config, dominantSet, 5)
	require.False(t, list.isOnline(1))

	// It is considered back online as a new event shows up.
	world.EXPECT().GetLastEvent(gomock.Any()).Return(makeEventWithSeq(2))
	list.updateAttendance(world, config, dominantSet, 6)
	require.True(t, list.isOnline(1))
}

func TestThrottling_getOnlineValidators_InitiallyAllValidatorsAreOfflineExceptLocal(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)

	validators := makeValidatorsFromStakes(100, 200, 300)

	localValidator := idx.ValidatorID(1)
	state := NewThrottlingState(localValidator, config.ThrottlerConfig{}, world)
	onlineValidators := state.getOnlineValidators(validators)
	require.Len(t, onlineValidators.IDs(), 1)
	require.Contains(t, onlineValidators.IDs(), localValidator)
}

func TestThrottling_computeOnlineDominantSet_InitiallyAllValidatorsAreConsideredOfflineExceptLocal(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	world := NewMockWorldReader(ctrl)

	validators := makeValidatorsFromStakes(100, 200, 300)
	world.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()

	localValidator := idx.ValidatorID(1)
	state := NewThrottlingState(localValidator,
		config.ThrottlerConfig{
			DominantStakeThreshold: 0.8,
		}, world)
	onlineValidators := state.computeOnlineDominantSet()
	require.Len(t, onlineValidators, 1)
	require.Contains(t, onlineValidators, localValidator)
}

func makeEventWithSeq(seq idx.Event) *inter.Event {
	builder := &inter.MutableEventPayload{}
	builder.SetSeq(seq)
	return &builder.Build().Event
}

// makeSet is a helper to create a dominantSet from a list of validator IDs.
func makeSet(ids ...idx.ValidatorID) dominantSet {
	res := make(dominantSet)
	for _, id := range ids {
		res[id] = struct{}{}
	}
	return res
}
