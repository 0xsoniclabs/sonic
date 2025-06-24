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

package basiccheck

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
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
