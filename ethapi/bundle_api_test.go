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
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
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
	bundleGasLimit := gasLimit + hexutil.Uint64(params.TxAccessListAddressGas) +
		hexutil.Uint64(params.TxAccessListStorageKeyGas)
	require.Equal(t, slices.Repeat([]hexutil.Uint64{bundleGasLimit}, numTransactions), gasLimits)
}

func TestGetPooledBundles_ReturnsNonEmptyNonError_WhenNoBundlesArePooled(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)
	api := NewPublicBundleAPI(backend)

	backend.EXPECT().ChainID().Return(big.NewInt(1)).AnyTimes()
	backend.EXPECT().GetPooledBundles()

	res, err := api.GetPooledBundles(t.Context())
	require.NoError(t, err)
	require.Empty(t, res)
}

func TestGetPooledBundles_(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	api := NewPublicBundleAPI(backend)
	chainId := big.NewInt(1)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	envelope, txBundle, plan := bundle.NewBuilder().
		With(
			bundle.Step(key, &types.AccessListTx{ChainID: chainId}),
		).BuildEnvelopeBundleAndPlan()

	backend.EXPECT().ChainID().Return(chainId).AnyTimes()
	backend.EXPECT().GetPooledBundles().Return(
		map[common.Hash]common.Hash{
			plan.Hash(): envelope.Hash(),
		},
	)
	backend.EXPECT().GetPoolTransaction(envelope.Hash()).Return(envelope)

	result, err := api.GetPooledBundles(t.Context())
	require.NoError(t, err)

	require.Len(t, result, 1)
	require.Equal(t, plan.Hash(), result[0].PlanHash)
	require.Len(t, result[0].Transactions, 1)
	require.Equal(t, result[0].Transactions[0].Hash, txBundle.Transactions[0].Hash())
}

func TestGetPooledBundles_IgnoresInvalidQueuedBundles(t *testing.T) {

	invalidTx := types.NewTx(&types.LegacyTx{
		Data: []byte{0x1, 0x2},
	})

	cases := map[string]*types.Transaction{
		"no envelope":      nil,
		"invalid envelope": invalidTx,
	}

	for name, tx := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)

			api := NewPublicBundleAPI(backend)
			chainId := big.NewInt(1)

			key, err := crypto.GenerateKey()
			require.NoError(t, err)

			envelope, _, plan := bundle.NewBuilder().
				With(
					bundle.Step(key, &types.AccessListTx{ChainID: chainId}),
				).BuildEnvelopeBundleAndPlan()

			backend.EXPECT().ChainID().Return(chainId).AnyTimes()
			backend.EXPECT().GetPooledBundles().Return(
				map[common.Hash]common.Hash{
					plan.Hash(): envelope.Hash(),
				},
			)
			backend.EXPECT().GetPoolTransaction(envelope.Hash()).Return(tx)

			result, err := api.GetPooledBundles(t.Context())
			require.NoError(t, err)
			require.Empty(t, result)
		})
	}
}
