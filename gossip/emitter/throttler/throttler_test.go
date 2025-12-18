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
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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

func TestThrottler_updateAttendance_SuppressedValidatorsAreOffline_AfterNonDominatingTimeout(t *testing.T) {
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
					NonDominatingTimeout: DominatingTimeout,
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
				DominatingTimeout:      3,
				NonDominatingTimeout:   test.NonDominatingTimeout,
			}

			attendanceList := newAttendanceList()
			attendanceList.attendance[1] = test.lastAttendance

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
