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
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBundleEstimateGas_BasicFunctionallyIsProvided(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBackend := NewMockBackend(ctrl)
	mockState := state.NewMockStateDB(ctrl)
	mockHeader := &evmcore.EvmHeader{
		Number: big.NewInt(1),
		Root:   common.Hash{123},
	}
	mockBlock := &evmcore.EvmBlock{EvmHeader: *mockHeader}

	blkNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	any := gomock.Any()
	mockBackend.EXPECT().GetNetworkRules(any, idx.Block(1)).Return(&opera.Rules{}, nil).AnyTimes()
	mockBackend.EXPECT().StateAndBlockByNumberOrHash(any, blkNr).Return(mockState, mockBlock, nil).AnyTimes()
	mockBackend.EXPECT().RPCGasCap().Return(uint64(10000000))
	mockBackend.EXPECT().MaxGasLimit().Return(uint64(10000000)).Times(2)
	mockBackend.EXPECT().HeaderByNumber(any, any).Return(mockHeader, nil).Times(2)
	mockBackend.EXPECT().ChainConfig(gomock.Any()).Return(&params.ChainConfig{}).Times(2)
	mockBackend.EXPECT().GetEVM(any, any, any, any, any).DoAndReturn(getEvmFunc(mockState)).AnyTimes()
	setExpectedStateCalls(mockState)

	api := NewPublicBundleAPI(mockBackend)
	args := []TransactionArgs{getTxArgs(t), getTxArgs(t)}

	gas, err := api.EstimateGasForTransactions(context.Background(), args, &blkNr, nil, nil)
	require.NoError(t, err, "failed to estimate gas")
	require.Equal(t, len(args), len(gas.GasLimits))
	require.Greater(t, gas.GasLimits[0], uint64(0))
	require.Greater(t, gas.GasLimits[1], uint64(0))
}
