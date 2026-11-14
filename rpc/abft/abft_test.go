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

package abft_test

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/rpc/abft"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
)

func profiles(weight int64, key byte) map[idx.ValidatorID]drivertype.Validator {
	return map[idx.ValidatorID]drivertype.Validator{
		1: {Weight: big.NewInt(weight), PubKey: validatorpk.PubKey{Type: validatorpk.Types.Secp256k1, Raw: []byte{key, 0x01}}},
		2: {Weight: big.NewInt(weight), PubKey: validatorpk.PubKey{Type: validatorpk.Types.Secp256k1, Raw: []byte{key, 0x02}}},
	}
}

// states are the states of an epoch of validators 1 and 2: at block 10,
// validator 1 was last online at block 5, validator 2 at block 9, and the
// rules forgive 3 missed blocks.
func states() core.EpochSnapshot {
	var states core.EpochSnapshot
	states.EpochState.Validators = pos.EqualWeightValidators([]idx.ValidatorID{1, 2}, 1)
	states.EpochState.ValidatorProfiles = profiles(10, 0xaa)
	states.EpochState.Rules.Economy.BlockMissedSlack = 3
	states.BlockState.LastBlock = iblockproc.BlockCtx{Idx: 10, Time: 1000}
	states.BlockState.ValidatorStates = []iblockproc.ValidatorBlockState{
		{LastBlock: 5, LastOnlineTime: 400, Uptime: 77, Originated: big.NewInt(9)},
		{LastBlock: 9, LastOnlineTime: 900, Uptime: 88, Originated: big.NewInt(8)},
	}
	states.BlockState.NextValidatorProfiles = profiles(20, 0xbb)
	return states
}

func newHarness(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewFakeCore()
	fake.SetEpochState(1, states())
	fake.SetLiveEpochState(states())
	return tests.NewHarness(t, fake, func(h *tests.Harness) {
		abft.Register(h.Dispatcher, h.Environment, h.Core)
	})
}

func TestGetValidators_AnswersTheProfilesOfAnEpoch(t *testing.T) {
	h := newHarness(t)
	require.JSONEq(t, `{"0x1":{"pubkey":"0xc0aa01","weight":"0xa"},"0x2":{"pubkey":"0xc0aa02","weight":"0xa"}}`,
		h.Result(t, "abft_getValidators", "latest"))
	require.JSONEq(t, `{"0x1":{"pubkey":"0xc0bb01","weight":"0x14"},"0x2":{"pubkey":"0xc0bb02","weight":"0x14"}}`,
		h.Result(t, "abft_getValidators", "pending"), "pending has the profiles of the next epoch")
	require.Equal(t, "null", h.Result(t, "abft_getValidators", "0x7"))
	h.Core.SetMode("EpochState", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, "abft_getValidators", "latest").Message)
}

func TestValidatorRecords_AnswerTheOpenEpoch(t *testing.T) {
	h := newHarness(t)
	cases := map[string]struct {
		validator                    string
		downtime, uptime, originated string
	}{
		"offline beyond the slack": {"0x1", `{"offlineBlocks":"0x5","offlineTime":"0x258"}`, `"0x4d"`, `"0x9"`},
		"offline within the slack": {"0x2", `{"offlineBlocks":"0x0","offlineTime":"0x0"}`, `"0x58"`, `"0x8"`},
		"no validator":             {"0x9", `{"offlineBlocks":"0x0","offlineTime":"0x0"}`, `"0x0"`, `null`},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.JSONEq(t, test.downtime, h.Result(t, "abft_getDowntime", test.validator))
			require.Equal(t, test.uptime, h.Result(t, "abft_getEpochUptime", test.validator))
			require.Equal(t, test.originated, h.Result(t, "abft_getOriginatedEpochFee", test.validator))
		})
	}
}
