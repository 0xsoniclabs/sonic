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

// func TestThrottling_SkipEventEmission_DoNotSkipIfBelongingToDominantSet(t *testing.T) {

// 	tests := map[string]struct {
// 		validatorID idx.ValidatorID
// 		validators  *pos.Validators
// 	}{
// 		"validator is equivalent to dominant cutoff": {
// 			validatorID: 1,
// 			validators:  makeValidators(75, 25),
// 		},
// 		"validator belongs to dominant set": {
// 			validatorID: 2,
// 			validators: makeValidators(
// 				750, 750, // 75% owned by first two validators
// 				125, 125, 125, 125,
// 			),
// 		},
// 		"non-first validator belongs to dominant set": {
// 			validatorID: 2,
// 			validators: makeValidators(
// 				750, 750, // 75% owned by first two validators
// 				125, 125, 125, 125,
// 			),
// 		},
// 	}

// 	for name, test := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			ctrl := gomock.NewController(t)

// 			lastEventHash, lastEvent := createTestEventWithFrame(idx.Frame(1))

// 			world := NewMockWorldReader(ctrl)
// 			world.EXPECT().GetEpochValidators().Return(test.validators, idx.Epoch(0))
// 			world.EXPECT().GetRules().Return(opera.Rules{})
// 			world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0)).AnyTimes()
// 			world.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).Return(&lastEventHash).AnyTimes()
// 			world.EXPECT().GetEvent(gomock.Any()).Return(&lastEvent).AnyTimes()

// 			state := NewThrottlingState(test.validatorID, 0.75, 3, 10, world)

// 			event := inter.NewMockEventPayloadI(ctrl)
// 			event.EXPECT().Transactions().Return(types.Transactions{}).AnyTimes()
// 			event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()
// 			event.EXPECT().SelfParent().Return(&hash.Event{1})

// 			skip := state.CanSkipEventEmission(event)
// 			require.Equal(t, DoNotSkipEvent_DominantStake, skip)
// 		})
// 	}
// }

// func TestThrottling_SkipEventEmission_SkipIfNotBelongingToDominantSet(t *testing.T) {

// 	stakes := makeValidators(
// 		750, 750, // 75% owned by first two validators
// 		125, 125, 125, 125,
// 	)

// 	ctrl := gomock.NewController(t)
// 	world := NewMockWorldReader(ctrl)
// 	world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0))
// 	world.EXPECT().GetRules().Return(opera.Rules{})
// 	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0)).AnyTimes()
// 	lastEventHash, lastEvent := createTestEventWithFrame(idx.Frame(1))
// 	world.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).Return(&lastEventHash).AnyTimes()
// 	world.EXPECT().GetEvent(gomock.Any()).Return(&lastEvent).AnyTimes()

// 	state := NewThrottlingState(3, 0.75, 1, 10, world)

// 	event := inter.NewMockEventPayloadI(ctrl)
// 	event.EXPECT().Transactions().Return(types.Transactions{})
// 	event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()
// 	event.EXPECT().SelfParent().Return(&hash.Event{1})

// 	skip := state.CanSkipEventEmission(event)
// 	require.Equal(t, SkipEventEmission, skip)
// }

// func TestThrottling_DoNotSkip_WhenEventCarriesTransactions(t *testing.T) {

// 	ctrl := gomock.NewController(t)
// 	world := NewMockWorldReader(ctrl)
// 	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0))

// 	state := NewThrottlingState(3, 0.75, 0, 10, world)

// 	event := inter.NewMockEventPayloadI(ctrl)
// 	event.EXPECT().Transactions().Return(
// 		types.Transactions{types.NewTx(&types.LegacyTx{})})
// 	event.EXPECT().Frame().MinTimes(1)

// 	skip := state.CanSkipEventEmission(event)
// 	require.Equal(t, DoNotSkipEvent_CarriesTransactions, skip)
// }

// func TestThrottling_DoNotSkip_WhenEventBelongsToTheSameFrame(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	// stakes are dominated by validators 1 and 2,
// 	// this allows validator 3 to be throttled for this test
// 	stakes := makeValidators(500, 300, 200)

// 	world := NewMockWorldReader(ctrl)
// 	world.EXPECT().GetEpochValidators().Return(stakes, idx.Epoch(0)).AnyTimes()
// 	world.EXPECT().GetRules().AnyTimes()
// 	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0)).AnyTimes()
// 	world.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).AnyTimes()

// 	// repeat test for a variety of maxRepeatedFrames settings
// 	for _, maxRepeatedFrames := range []uint{1, 2, 3, 4, 80} {

// 		state := NewThrottlingState(3, 0.75, maxRepeatedFrames, 10, world)
// 		repeatedFrame := idx.Frame(7) // any frame number, repeatedly used

// 		for range maxRepeatedFrames {

// 			repeatedFrameEvent := inter.NewMockEventPayloadI(ctrl)
// 			repeatedFrameEvent.EXPECT().Transactions()
// 			repeatedFrameEvent.EXPECT().Frame().Return(repeatedFrame).MinTimes(1)
// 			repeatedFrameEvent.EXPECT().SelfParent()
// 			skip := state.CanSkipEventEmission(repeatedFrameEvent)
// 			require.Equal(t, SkipEventEmission, skip)
// 		}

// 		oneTooManyEvent := inter.NewMockEventPayloadI(ctrl)
// 		oneTooManyEvent.EXPECT().Transactions()
// 		oneTooManyEvent.EXPECT().Frame().Return(repeatedFrame).MinTimes(1)
// 		skip := state.CanSkipEventEmission(oneTooManyEvent)
// 		require.Equal(t, DoNotSkipEvent_SameFrameExceeded, skip)
// 	}
// }

// func TestThrottling_DoNotSkip_IfTooManyBlocksAreSkipped(t *testing.T) {
// 	ctrl := gomock.NewController(t)

// 	world := NewMockWorldReader(ctrl)
// 	world.EXPECT().GetRules().Return(
// 		opera.Rules{
// 			Economy: opera.EconomyRules{
// 				BlockMissedSlack: 50,
// 			},
// 		}).AnyTimes()
// 	world.EXPECT().GetEpochValidators().
// 		Return(makeValidators(500, 300, 200),
// 			idx.Epoch(0)).AnyTimes()

// 	lastEventHash, lastEvent := createTestEventWithFrame(idx.Frame(1))
// 	world.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).Return(&lastEventHash).AnyTimes()
// 	world.EXPECT().GetEvent(gomock.Any()).Return(&lastEvent).AnyTimes()

// 	throttler := NewThrottlingState(3, 0.75, 10, 10, world)
// 	throttler.lastEmissionBlockNumber = idx.Block(17)

// 	event := inter.NewMockEventPayloadI(ctrl)
// 	event.EXPECT().Transactions().AnyTimes()
// 	event.EXPECT().Frame().Return(idx.Frame(2)).AnyTimes()
// 	event.EXPECT().SelfParent().AnyTimes()

// 	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(17 + 50)).Times(2)
// 	skip := throttler.CanSkipEventEmission(event)
// 	require.Equal(t, DoNotSkipEvent_TooManyBlocksMissed, skip,
// 		"Event missing so many blocks should NOT be skipped")

// 	// one more than the last time
// 	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(17 + 50 + 1)).MinTimes(1)
// 	skip = throttler.CanSkipEventEmission(event)
// 	require.Equal(t, SkipEventEmission, skip,
// 		"Event missing less than max allowed blocks should be skipped")
// }

// func TestThrottling_DoNotSkip_RespectHeartbeatEvents(t *testing.T) {

// 	validators := makeValidators(10, 10, 10, 10) // one suppressed validator

// 	ctrl := gomock.NewController(t)

// 	world := NewMockWorldReader(ctrl)
// 	world.EXPECT().GetRules().Return(opera.Rules{}).AnyTimes()
// 	world.EXPECT().GetEpochValidators().
// 		Return(validators, idx.Epoch(0)).AnyTimes()
// 	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0)).AnyTimes()

// 	eventHash, event := createTestEventWithFrame(idx.Frame(1))
// 	world.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).Return(&eventHash)
// 	world.EXPECT().GetEvent(gomock.Any()).Return(&event)

// 	throttler := NewThrottlingState(
// 		4,
// 		0.75, // first three validators dominate stake
// 		1000, // large enough to not interfere with this test
// 		3,    // number for frames to emit heartbeats
// 		world)

// 	// Event 1 should be considered a heartbeat
// 	event1 := inter.NewMockEventPayloadI(ctrl)
// 	event1.EXPECT().Transactions().Return(types.Transactions{})
// 	event1.EXPECT().Frame().Return(idx.Frame(4)).AnyTimes()

// 	skip := throttler.CanSkipEventEmission(event1)
// 	require.Equal(t, DoNotSkipEvent_Heartbeat, skip,
// 		"Heartbeat event should NOT be skipped")

// 	// Event 2 is created shortly after Event 2 with the next frame number.
// 	// It can be skipped.
// 	eventHash, event = createTestEventWithFrame(idx.Frame(4))
// 	world.EXPECT().GetLastEvent(gomock.Any(), gomock.Any()).Return(&eventHash).AnyTimes()
// 	world.EXPECT().GetEvent(gomock.Any()).Return(&event).AnyTimes()
// 	event2 := inter.NewMockEventPayloadI(ctrl)
// 	event2.EXPECT().Transactions().Return(types.Transactions{})
// 	event2.EXPECT().Frame().Return(idx.Frame(5)).AnyTimes()
// 	event2.EXPECT().SelfParent().Return(&hash.Event{42})

// 	skip = throttler.CanSkipEventEmission(event2)
// 	require.Equal(t, SkipEventEmission, skip)
// }

// func createTestEventWithFrame(frame idx.Frame) (hash.Event, inter.Event) {
// 	lastEventHash := hash.Event{1}
// 	lastEventBuilder := &inter.MutableEventPayload{}
// 	lastEventBuilder.SetFrame(idx.Frame(frame))
// 	lastEvent := lastEventBuilder.Build().Event
// 	return lastEventHash, lastEvent
// }

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
