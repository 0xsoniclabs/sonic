// Copyright 2026 Sonic Operations Ltd
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

package epochcheck

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	base "github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	pos "github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestChecker_Validate_SingleProposerIntroducesNewFormat(t *testing.T) {

	versions := map[bool]uint8{
		false: 2, // old format
		true:  3, // new format
	}

	for enabled, version := range versions {
		t.Run(fmt.Sprintf("singleProposer=%t", enabled), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			reader := NewMockReader(ctrl)
			event := inter.NewMockEventPayloadI(ctrl)

			creator := idx.ValidatorID(1)
			event.EXPECT().Epoch().AnyTimes()
			event.EXPECT().Parents().AnyTimes()
			event.EXPECT().Extra().AnyTimes()
			event.EXPECT().GasPowerUsed().AnyTimes()
			event.EXPECT().Transactions().AnyTimes()
			event.EXPECT().TransactionsToMeter().AnyTimes()
			event.EXPECT().MisbehaviourProofs().AnyTimes()
			event.EXPECT().BlockVotes().AnyTimes()
			event.EXPECT().BlockHashes().AnyTimes()
			event.EXPECT().EpochVote().AnyTimes()
			event.EXPECT().Creator().Return(creator).AnyTimes()

			builder := pos.NewBuilder()
			builder.Set(creator, 10)
			validators := builder.Build()

			reader.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()

			rules := opera.Rules{Upgrades: opera.Upgrades{
				Sonic:                        true,
				SingleProposerBlockFormation: enabled,
			}}
			reader.EXPECT().GetEpochRules().Return(rules, idx.Epoch(0)).AnyTimes()

			checker := Checker{
				Base:   base.New(reader),
				reader: reader,
			}

			// Check that the correct version is fine.
			event.EXPECT().Version().Return(version)
			require.NoError(t, checker.Validate(event))

			// Check that the wrong version fails.
			event.EXPECT().Version().Return(version + 1)
			require.ErrorIs(t, checker.Validate(event), ErrWrongVersion)
		})
	}
}

func TestChecker_Validate_BlockHashesOnEventsIntroducesVersion4(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)
	event := inter.NewMockEventPayloadI(ctrl)

	creator := idx.ValidatorID(1)
	event.EXPECT().Epoch().AnyTimes()
	event.EXPECT().Parents().AnyTimes()
	event.EXPECT().Extra().AnyTimes()
	event.EXPECT().GasPowerUsed().AnyTimes()
	event.EXPECT().Transactions().AnyTimes()
	event.EXPECT().TransactionsToMeter().AnyTimes()
	event.EXPECT().MisbehaviourProofs().AnyTimes()
	event.EXPECT().BlockVotes().AnyTimes()
	event.EXPECT().BlockHashes().AnyTimes()
	event.EXPECT().EpochVote().AnyTimes()
	event.EXPECT().Creator().Return(creator).AnyTimes()

	builder := pos.NewBuilder()
	builder.Set(creator, 10)
	validators := builder.Build()

	reader.EXPECT().GetEpochValidators().Return(validators, idx.Epoch(0)).AnyTimes()

	rules := opera.Rules{Upgrades: opera.Upgrades{
		Sonic:                        true,
		SingleProposerBlockFormation: true,
		BlockHashesOnEvents:          true,
	}}
	reader.EXPECT().GetEpochRules().Return(rules, idx.Epoch(0)).AnyTimes()

	checker := Checker{
		Base:   base.New(reader),
		reader: reader,
	}

	// Version 4 should be accepted.
	event.EXPECT().Version().Return(uint8(4))
	require.NoError(t, checker.Validate(event))

	// Version 3 should be rejected.
	event.EXPECT().Version().Return(uint8(3))
	require.ErrorIs(t, checker.Validate(event), ErrWrongVersion)
}

func TestCalcGasPowerUsed_IncludesBlockHashesGas(t *testing.T) {
	ctrl := gomock.NewController(t)

	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())

	// Without block hashes
	eventWithout := inter.NewMockEventPayloadI(ctrl)
	eventWithout.EXPECT().Parents().Return(nil).AnyTimes()
	eventWithout.EXPECT().Extra().Return(nil).AnyTimes()
	eventWithout.EXPECT().TransactionsToMeter().Return(nil).AnyTimes()
	eventWithout.EXPECT().MisbehaviourProofs().Return(nil).AnyTimes()
	eventWithout.EXPECT().BlockVotes().Return(inter.LlrBlockVotes{}).AnyTimes()
	eventWithout.EXPECT().EpochVote().Return(inter.LlrEpochVote{}).AnyTimes()
	eventWithout.EXPECT().BlockHashes().Return(inter.BlockHashes{}).AnyTimes()
	gasWithout := CalcGasPowerUsed(eventWithout, rules)

	// With block hashes
	eventWith := inter.NewMockEventPayloadI(ctrl)
	eventWith.EXPECT().Parents().Return(nil).AnyTimes()
	eventWith.EXPECT().Extra().Return(nil).AnyTimes()
	eventWith.EXPECT().TransactionsToMeter().Return(nil).AnyTimes()
	eventWith.EXPECT().MisbehaviourProofs().Return(nil).AnyTimes()
	eventWith.EXPECT().BlockVotes().Return(inter.LlrBlockVotes{}).AnyTimes()
	eventWith.EXPECT().EpochVote().Return(inter.LlrEpochVote{}).AnyTimes()
	eventWith.EXPECT().BlockHashes().Return(inter.BlockHashes{
		Start:  1,
		Epoch:  1,
		Hashes: []hash.Hash{{1}, {2}, {3}},
	}).AnyTimes()
	gasWith := CalcGasPowerUsed(eventWith, rules)

	require.Greater(t, gasWith, gasWithout)
}
