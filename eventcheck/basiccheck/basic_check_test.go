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

package basiccheck

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestChecker_checkTxs_AcceptsValidTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)

	valid := types.NewTx(&types.LegacyTx{To: &common.Address{}, Gas: 21000})
	require.NoError(t, validateTx(valid))

	event.EXPECT().Transactions().Return(types.Transactions{valid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()

	err := New().checkTxs(event)
	require.NoError(t, err)
}

func TestChecker_checkTxs_DetectsIssuesInTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := inter.NewMockEventPayloadI(ctrl)

	invalid := types.NewTx(&types.LegacyTx{
		Value: big.NewInt(-1),
	})

	event.EXPECT().Transactions().Return(types.Transactions{invalid}).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()

	err := New().checkTxs(event)
	require.Error(t, err)
}

func TestChecker_IntrinsicGas_LegacyCalculationDoesNotAccountForInitDataOrAuthList(t *testing.T) {
	tests := map[string]*types.Transaction{
		"legacyTx": types.NewTx(&types.LegacyTx{
			To:  nil,
			Gas: 21_000,
			// some data that takes
			Data: make([]byte, params.MaxInitCodeSize),
		}),
		"setCodeTx": types.NewTx(&types.SetCodeTx{
			To:       common.Address{},
			Gas:      21_000,
			AuthList: []types.SetCodeAuthorization{{}}}),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			costLegacy, err := intrinsicGasLegacy(tx.Data(), tx.AccessList(), tx.To() == nil)
			require.NoError(t, err)

			// in sonic, Homestead, Istanbul and Shanghai are always active
			costNew, err := core.IntrinsicGas(tx.Data(), tx.AccessList(),
				tx.SetCodeAuthorizations(), tx.To() == nil, true, true, true)
			require.NoError(t, err)

			require.Greater(t, costNew, costLegacy)
		})
	}
}

func TestChecker_IntrinsicGas_LegacyIsCheaperOrSameForAllRevisionCombinations(t *testing.T) {
	trueFalse := []bool{true, false}
	for _, homestead := range trueFalse {
		for _, istanbul := range trueFalse {
			for _, shanghai := range trueFalse {
				t.Run(makeTestName(homestead, istanbul, shanghai), func(t *testing.T) {
					costLegacy, err := intrinsicGasLegacy([]byte("test"), nil, false)
					require.NoError(t, err)
					costNew, err := core.IntrinsicGas([]byte("test"), nil, nil, false, homestead, istanbul, shanghai)
					require.NoError(t, err)
					require.GreaterOrEqual(t, costNew, costLegacy)
				})
			}
		}
	}
}

func makeTestName(homestead, istanbul, shanghai bool) string {
	name := ""
	withWithout := func(fork bool) string {
		if fork {
			return "With"
		}
		return "Without"
	}
	name += withWithout(homestead) + "Homestead"
	name += withWithout(istanbul) + "Istanbul"
	name += withWithout(shanghai) + "Shanghai"
	return name
}

func setupBasicEventMock(ctrl *gomock.Controller) *inter.MockEventPayloadI {
	event := inter.NewMockEventPayloadI(ctrl)

	// Base checker fields (lachesis-base basiccheck)
	event.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	event.EXPECT().Epoch().Return(idx.Epoch(1)).AnyTimes()
	event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()
	event.EXPECT().Lamport().Return(idx.Lamport(1)).AnyTimes()
	event.EXPECT().Parents().Return(nil).AnyTimes()

	// Sonic basiccheck fields
	event.EXPECT().NetForkID().Return(uint16(0)).AnyTimes()
	event.EXPECT().GasPowerUsed().Return(uint64(0)).AnyTimes()
	event.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	event.EXPECT().CreationTime().Return(inter.Timestamp(1)).AnyTimes()
	event.EXPECT().MedianTime().Return(inter.Timestamp(1)).AnyTimes()
	event.EXPECT().Transactions().Return(nil).AnyTimes()
	event.EXPECT().Payload().Return(&inter.Payload{}).AnyTimes()

	return event
}

func TestChecker_Validate_RejectsTooManyBlockHashes(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := setupBasicEventMock(ctrl)

	hashes := make([]hash.Hash, MaxBlockHashesPerEvent+1)
	event.EXPECT().BlockHashes().Return(inter.BlockHashes{
		Start:  1,
		Epoch:  1,
		Hashes: hashes,
	}).AnyTimes()

	err := New().Validate(event)
	require.ErrorIs(t, err, ErrTooManyBlockHash)
}

func TestChecker_Validate_AcceptsMaxBlockHashes(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := setupBasicEventMock(ctrl)

	hashes := make([]hash.Hash, MaxBlockHashesPerEvent)
	event.EXPECT().BlockHashes().Return(inter.BlockHashes{
		Start:  1,
		Epoch:  1,
		Hashes: hashes,
	}).AnyTimes()

	err := New().Validate(event)
	require.NoError(t, err)
}

func TestChecker_Validate_AcceptsEmptyBlockHashesWithZeroMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := setupBasicEventMock(ctrl)

	event.EXPECT().BlockHashes().Return(inter.BlockHashes{}).AnyTimes()

	err := New().Validate(event)
	require.NoError(t, err)
}

func TestChecker_Validate_RejectsBlockHashesWithZeroStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := setupBasicEventMock(ctrl)

	event.EXPECT().BlockHashes().Return(inter.BlockHashes{
		Start:  0,
		Epoch:  1,
		Hashes: []hash.Hash{{}},
	}).AnyTimes()

	err := New().Validate(event)
	require.ErrorIs(t, err, ErrBlockHashZeroStart)
}

func TestChecker_Validate_RejectsBlockHashesWithZeroEpoch(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := setupBasicEventMock(ctrl)

	event.EXPECT().BlockHashes().Return(inter.BlockHashes{
		Start:  1,
		Epoch:  0,
		Hashes: []hash.Hash{{}},
	}).AnyTimes()

	err := New().Validate(event)
	require.ErrorIs(t, err, ErrBlockHashZeroEpoch)
}

func TestChecker_Validate_RejectsBlockHashesWithMismatchedEpoch(t *testing.T) {
	ctrl := gomock.NewController(t)
	event := setupBasicEventMock(ctrl)

	// setupBasicEventMock sets event Epoch to 1, so use a different epoch here
	event.EXPECT().BlockHashes().Return(inter.BlockHashes{
		Start:  1,
		Epoch:  2,
		Hashes: []hash.Hash{{}},
	}).AnyTimes()

	err := New().Validate(event)
	require.ErrorIs(t, err, ErrBlockHashEpochMismatch)
}
