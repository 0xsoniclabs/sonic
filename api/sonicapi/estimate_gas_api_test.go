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

package sonicapi

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// bundleGasOverhead is the extra gas added per transaction in the bundle
// to account for the execution plan entries in the access list.
const bundleGasOverhead = params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas

func Test_EstimateGasForTransactions_SingleTransfer(t *testing.T) {
	acc1, err := rpctest.NewWallet()
	require.NoError(t, err)
	acc2, err := rpctest.NewWallet()
	require.NoError(t, err)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(*acc1.Address(), rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(*acc2.Address(), rpctest.AccountState{}).
		Build()

	api := NewPublicBundleAPI(be)

	args := []ethapi.TransactionArgs{
		{
			From:  acc1.Address(),
			To:    acc2.Address(),
			Value: rpctest.ToHexBigInt(big.NewInt(1e17)),
		},
	}

	result, err := api.EstimateGasForTransactions(t.Context(), args, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, result.GasLimits, 1)
	// Gas must be at least TxGas + bundle overhead
	require.GreaterOrEqual(t, uint64(result.GasLimits[0]), uint64(params.TxGas+bundleGasOverhead))
}

func Test_EstimateGasForTransactions_MultipleIndependentTransfers(t *testing.T) {
	acc1, err := rpctest.NewWallet()
	require.NoError(t, err)
	acc2, err := rpctest.NewWallet()
	require.NoError(t, err)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(*acc1.Address(), rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(*acc2.Address(), rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := []ethapi.TransactionArgs{
		{
			From:  acc1.Address(),
			To:    acc2.Address(),
			Nonce: rpctest.ToHexUint64(0),
			Value: rpctest.ToHexBigInt(big.NewInt(1e16)),
		},
		{
			From:  acc2.Address(),
			To:    acc1.Address(),
			Nonce: rpctest.ToHexUint64(0),
			Value: rpctest.ToHexBigInt(big.NewInt(1e16)),
		},
		{
			From:  acc1.Address(),
			To:    acc2.Address(),
			Nonce: rpctest.ToHexUint64(1),
			Value: rpctest.ToHexBigInt(big.NewInt(1e16)),
		},
	}

	result, err := api.EstimateGasForTransactions(t.Context(), args, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, result.GasLimits, 3)
	for i, gas := range result.GasLimits {
		require.GreaterOrEqual(t, uint64(gas), uint64(params.TxGas+bundleGasOverhead),
			"gas limit for tx %d is too low", i)
	}
}

func Test_EstimateGasForTransactions_TooManyTransactions(t *testing.T) {
	acc1, err := rpctest.NewWallet()
	require.NoError(t, err)
	acc2, err := rpctest.NewWallet()
	require.NoError(t, err)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(*acc1.Address(), rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	// Build 17 transactions (limit is 16)
	args := make([]ethapi.TransactionArgs, MAX_BUNDLE_TRANSACTIONS+1)
	for i := range args {
		args[i] = ethapi.TransactionArgs{
			From:  acc1.Address(),
			To:    acc2.Address(),
			Nonce: rpctest.ToHexUint64(uint64(i)),
			Value: rpctest.ToHexBigInt(big.NewInt(1)),
		}
	}

	_, err = api.EstimateGasForTransactions(t.Context(), args, nil, nil, nil)
	require.ErrorContains(t, err, "too many transactions")
}

func Test_EstimateGasForTransactions_EmptyArgs(t *testing.T) {
	be := rpctest.NewBackendBuilder(t).Build()
	api := NewPublicBundleAPI(be)

	result, err := api.EstimateGasForTransactions(t.Context(), []ethapi.TransactionArgs{}, nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, result.GasLimits)
}

func Test_EstimateGasForTransactions_WithExplicitBlockNumber(t *testing.T) {
	acc1, err := rpctest.NewWallet()
	require.NoError(t, err)
	acc2, err := rpctest.NewWallet()
	require.NoError(t, err)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(*acc1.Address(), rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithBlockHistory([]rpctest.Block{
			{Number: 1, Hash: common.HexToHash("0x1")},
			{Number: 2, Hash: common.HexToHash("0x2"), ParentHash: common.HexToHash("0x1")},
		}).
		Build()

	api := NewPublicBundleAPI(be)

	blockNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(1))
	args := []ethapi.TransactionArgs{
		{
			From:  acc1.Address(),
			To:    acc2.Address(),
			Value: rpctest.ToHexBigInt(big.NewInt(1e16)),
		},
	}

	result, err := api.EstimateGasForTransactions(t.Context(), args, &blockNrOrHash, nil, nil)
	require.NoError(t, err)
	require.Len(t, result.GasLimits, 1)
	require.GreaterOrEqual(t, uint64(result.GasLimits[0]), uint64(params.TxGas+bundleGasOverhead))
}

func Test_EstimateGasForTransactions_WithStateOverride(t *testing.T) {
	acc1, err := rpctest.NewWallet()
	require.NoError(t, err)
	acc2, err := rpctest.NewWallet()
	require.NoError(t, err)

	// acc1 starts with no balance, but a state override will fund it
	be := rpctest.NewBackendBuilder(t).
		WithAccount(*acc1.Address(), rpctest.AccountState{}).
		Build()

	api := NewPublicBundleAPI(be)

	overrideBalanceVal := hexutil.U256(*uint256.MustFromBig(big.NewInt(1e18)))
	overrideBalancePtr := &overrideBalanceVal
	overrides := ethapi.StateOverride{
		*acc1.Address(): ethapi.OverrideAccount{
			Balance: &overrideBalancePtr,
		},
	}

	args := []ethapi.TransactionArgs{
		{
			From:  acc1.Address(),
			To:    acc2.Address(),
			Value: rpctest.ToHexBigInt(big.NewInt(1e16)),
		},
	}

	result, err := api.EstimateGasForTransactions(t.Context(), args, nil, &overrides, nil)
	require.NoError(t, err)
	require.Len(t, result.GasLimits, 1)
	require.GreaterOrEqual(t, uint64(result.GasLimits[0]), uint64(params.TxGas+bundleGasOverhead))
}
