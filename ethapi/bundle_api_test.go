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

package ethapi

import (
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBundleEstimateGas_PreArgsAreConsideredForEveryTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	estimator := NewMockGasEstimator(ctrl)

	gasLimit := hexutil.Uint64(21000)
	numTransactions := 3

	for i := range numTransactions {
		estimator.EXPECT().EstimateGas(
			gomock.Any(),
			gomock.Any(),
		).DoAndReturn(func(args TransactionArgs, preArgs []TransactionArgs) (hexutil.Uint64, error) {
			require.Len(t, preArgs, i, "unexpected number of preArgs for transaction %d", i)
			return gasLimit, nil
		})
	}

	txArg := TransactionArgs{
		From: &common.Address{0x1},
		To:   &common.Address{0x2},
	}

	args := slices.Repeat([]TransactionArgs{txArg}, numTransactions)

	gasLimits, err := doEstimateGasForTransactions(args, estimator)
	require.NoError(t, err)
	bundleGasLimit := gasLimit + 2400 + 1900
	require.Equal(t, slices.Repeat([]hexutil.Uint64{bundleGasLimit}, numTransactions), gasLimits)
}
