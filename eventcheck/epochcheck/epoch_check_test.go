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
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	base "github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	pos "github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
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

func TestCalcGasPowerUsed(t *testing.T) {
	preBrio := opera.Rules{
		Economy: opera.EconomyRules{Gas: opera.GasRules{
			EventGas:             1000,
			ParentGas:            100,
			ExtraDataGas:         10,
			MisbehaviourProofGas: 500,
			BlockVotesBaseGas:    200,
			BlockVoteGas:         50,
			EpochVoteGas:         300,
		}},
		Dag:      opera.DagRules{MaxFreeParents: 3},
		Upgrades: opera.Upgrades{Sonic: true, Allegro: true},
	}

	withBrio := preBrio
	withBrio.Upgrades.Brio = true

	// zeroBrio to test only component under the test
	zeroBrio := opera.Rules{
		Upgrades: opera.Upgrades{Brio: true},
	}

	t.Run("valid gas used pre-Brio", func(t *testing.T) {
		e := newTestEvent(types.Transactions{txWithGas(21000)}, 5, 7, 3, 4, true)
		gas, err := CalcGasPowerUsed(e, preBrio)
		require.NoError(t, err)
		require.Equal(t, uint64(24470), gas)
	})

	t.Run("valid gas used Brio", func(t *testing.T) {
		e := newTestEvent(types.Transactions{txWithGas(21000)}, 5, 7, 3, 4, true)
		gas, err := CalcGasPowerUsed(e, withBrio)
		require.NoError(t, err)
		require.Equal(t, uint64(24470), gas)
	})

	t.Run("maximum of txsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		e := newTestEvent(types.Transactions{txWithGas(math.MaxUint64)}, 0, 0, 0, 0, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of txsGas (Brio)", func(t *testing.T) {
		e := newTestEvent(types.Transactions{txWithGas(math.MaxUint64 - 500), txWithGas(600)}, 5, 7, 3, 4, true)
		_, err := CalcGasPowerUsed(e, withBrio)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})

	t.Run("maximum of parentsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.ParentGas = math.MaxUint64
		e := newTestEvent(nil, 1, 0, 0, 0, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of parentsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.ParentGas = math.MaxUint64
		e := newTestEvent(nil, 2, 0, 0, 0, false)
		_, err := CalcGasPowerUsed(e, rules)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})

	t.Run("maximum of extraGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.ExtraDataGas = math.MaxUint64
		// 1 byte extra: SafeMul(1, MaxUint64) = MaxUint64
		e := newTestEvent(nil, 0, 1, 0, 0, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of extraGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.ExtraDataGas = math.MaxUint64
		// 2 bytes extra: SafeMul(2, MaxUint64) overflows
		e := newTestEvent(nil, 0, 2, 0, 0, false)
		_, err := CalcGasPowerUsed(e, rules)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})

	t.Run("maximum of mpsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.MisbehaviourProofGas = math.MaxUint64
		// 1 proof: SafeMul(1, MaxUint64) = MaxUint64
		e := newTestEvent(nil, 0, 0, 1, 0, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of mpsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.MisbehaviourProofGas = math.MaxUint64
		// 2 proofs: SafeMul(2, MaxUint64) overflows
		e := newTestEvent(nil, 0, 0, 2, 0, false)
		_, err := CalcGasPowerUsed(e, rules)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})

	t.Run("maximum of blockVotesGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.BlockVoteGas = math.MaxUint64
		e := newTestEvent(nil, 0, 0, 0, 1, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of blockVotesGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.BlockVoteGas = math.MaxUint64
		e := newTestEvent(nil, 0, 0, 0, 2, false)
		_, err := CalcGasPowerUsed(e, rules)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})

	t.Run("maximum of bvsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.BlockVoteGas = math.MaxUint64 - 1
		rules.Economy.Gas.BlockVotesBaseGas = 1
		e := newTestEvent(nil, 0, 0, 0, 1, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of bvsGas (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.BlockVoteGas = math.MaxUint64
		rules.Economy.Gas.BlockVotesBaseGas = 1
		e := newTestEvent(nil, 0, 0, 0, 1, false)
		_, err := CalcGasPowerUsed(e, rules)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})

	t.Run("maximum of total (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.EventGas = math.MaxUint64
		e := newTestEvent(nil, 0, 0, 0, 0, false)
		gas, err := CalcGasPowerUsed(e, rules)
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), gas)
	})

	t.Run("overflow of total (Brio)", func(t *testing.T) {
		rules := zeroBrio
		rules.Economy.Gas.EventGas = math.MaxUint64
		e := newTestEvent(types.Transactions{txWithGas(1)}, 0, 0, 0, 0, false)
		_, err := CalcGasPowerUsed(e, rules)
		require.ErrorIs(t, err, ErrTooBigGasUsed)
	})
}

func newTestEvent(
	txs types.Transactions,
	parents int,
	extraLen int,
	mps int,
	blockVotes int,
	hasEpochVote bool,
) *inter.MutableEventPayload {
	e := &inter.MutableEventPayload{}
	e.SetTxs(txs)
	e.SetParents(make(hash.Events, parents))
	e.SetExtra(make([]byte, extraLen))
	e.SetMisbehaviourProofs(make([]inter.MisbehaviourProof, mps))
	bvs := inter.LlrBlockVotes{Votes: make([]hash.Hash, blockVotes)}
	if blockVotes > 0 {
		bvs.Start = idx.Block(1)
	}
	e.SetBlockVotes(bvs)
	if hasEpochVote {
		e.SetEpochVote(inter.LlrEpochVote{Epoch: idx.Epoch(1)})
	}
	return e
}

func txWithGas(gas uint64) *types.Transaction {
	return types.NewTx(&types.LegacyTx{Gas: gas})
}
