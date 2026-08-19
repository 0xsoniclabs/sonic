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

package rpctest

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var (
	estimateSender    = common.Address{0x1}
	estimateRecipient = common.Address{0x2}
	estimateContract  = common.Address{0x3}
)

// setAndClearSlotCode sets storage slot 0 and clears it again, which earns a
// refund, so the gas used is below the gas limit the call needs.
var setAndClearSlotCode = []byte{
	byte(vm.PUSH1), 1, byte(vm.PUSH1), 0, byte(vm.SSTORE),
	byte(vm.PUSH1), 0, byte(vm.PUSH1), 0, byte(vm.SSTORE),
}

func newEstimateGasBackend(t *testing.T, upgrades opera.Upgrades) *fakeBackend {
	return NewBackendBuilder(t).
		WithUpgrade(0, upgrades).
		WithAccount(estimateSender, AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(estimateRecipient, AccountState{Balance: big.NewInt(1)}).
		WithAccount(estimateContract, AccountState{Code: setAndClearSlotCode}).
		WithBlockHistory([]Block{
			{Number: 1, Hash: common.HexToHash("0x1"), PrevRandao: common.Hash{1}, BaseFee: big.NewInt(0)},
		}).
		Build()
}

func Test_DoEstimateGas_PlainTransfersAreEstimatedAtTheirPrice(t *testing.T) {
	tests := map[string]struct {
		upgrades opera.Upgrades
		to       common.Address
		value    int64
		want     uint64
	}{
		"Brio call":              {upgrades: opera.GetBrioUpgrades(), to: estimateRecipient, want: params.TxGas},
		"Brio transfer":          {upgrades: opera.GetBrioUpgrades(), to: estimateRecipient, value: 1, want: params.TxGas},
		"Canto call":             {upgrades: opera.GetCantoUpgrades(), to: estimateRecipient, want: 15_000},
		"Canto transfer":         {upgrades: opera.GetCantoUpgrades(), to: estimateRecipient, value: 1, want: 21_000},
		"Canto transfer to self": {upgrades: opera.GetCantoUpgrades(), to: estimateSender, value: 1, want: 12_000},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			be := newEstimateGasBackend(t, test.upgrades)
			args := ethapi.TransactionArgs{
				From:  &estimateSender,
				To:    &test.to,
				Value: (*hexutil.Big)(big.NewInt(test.value)),
			}
			gas, err := ethapi.DoEstimateGas(t.Context(), be, args,
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber), nil, nil, be.RPCGasCap(), nil)
			require.NoError(t, err)
			require.Equal(t, test.want, uint64(gas))
		})
	}
}

func Test_DoEstimateGas_ReturnsTheLowestGasLimitTheCallSucceedsWith(t *testing.T) {
	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	for name, upgrades := range map[string]opera.Upgrades{
		"Brio":  opera.GetBrioUpgrades(),
		"Canto": opera.GetCantoUpgrades(),
	} {
		t.Run(name, func(t *testing.T) {
			be := newEstimateGasBackend(t, upgrades)
			args := ethapi.TransactionArgs{From: &estimateSender, To: &estimateContract}
			gas, err := ethapi.DoEstimateGas(t.Context(), be, args, latest, nil, nil, be.RPCGasCap(), nil)
			require.NoError(t, err)

			succeeds := func(gas uint64) bool {
				args.Gas = ToHexUint64(gas)
				res, err := ethapi.DoCall(t.Context(), be, args, latest, nil, nil, 0, be.RPCGasCap(), nil, false)
				require.NoError(t, err)
				return !res.Failed()
			}
			require.True(t, succeeds(uint64(gas)))
			require.False(t, succeeds(uint64(gas)-1))
		})
	}
}
