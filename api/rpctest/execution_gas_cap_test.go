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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// gasLeftCode returns the gas left at its start as a 32-byte word.
var gasLeftCode = []byte{
	byte(vm.GAS), byte(vm.PUSH1), 0, byte(vm.MSTORE),
	byte(vm.PUSH1), 32, byte(vm.PUSH1), 0, byte(vm.RETURN),
}

func newExecutionGasCapBackend(t *testing.T, upgrades opera.Upgrades, contract common.Address) *fakeBackend {
	return NewBackendBuilder(t).
		WithUpgrade(0, upgrades).
		WithAccount(contract, AccountState{Code: gasLeftCode}).
		WithBlockHistory([]Block{
			{Number: 1, Hash: common.HexToHash("0x1"), PrevRandao: common.Hash{1}, BaseFee: big.NewInt(0)},
			{Number: 2, Hash: common.HexToHash("0x2"), ParentHash: common.HexToHash("0x1"), PrevRandao: common.Hash{2}, BaseFee: big.NewInt(0)},
		}).
		Build()
}

func Test_Calls_ExecuteWithTheirWholeGasLimit(t *testing.T) {
	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	calls := map[string]func(*testing.T, *fakeBackend, ethapi.TransactionArgs) []byte{
		"eth_call": func(t *testing.T, be *fakeBackend, args ethapi.TransactionArgs) []byte {
			out, err := ethapi.NewPublicBlockChainAPI(be).Call(t.Context(), args, &latest, nil, nil)
			require.NoError(t, err)
			return out
		},
		"debug_traceCall": func(t *testing.T, be *fakeBackend, args ethapi.TransactionArgs) []byte {
			res, err := ethapi.NewPublicDebugAPI(be, 100_000, 0, false).TraceCall(t.Context(), args, latest, nil)
			require.NoError(t, err)
			encoded, err := json.Marshal(res)
			require.NoError(t, err)
			var result struct{ ReturnValue hexutil.Bytes }
			require.NoError(t, json.Unmarshal(encoded, &result))
			return result.ReturnValue
		},
		"trace_call": func(t *testing.T, be *fakeBackend, args ethapi.TransactionArgs) []byte {
			res, err := ethapi.NewPublicTxTraceAPI(be, 100_000, 1000).Call(t.Context(), args, []string{"trace"}, latest, nil)
			require.NoError(t, err)
			return res.Output
		},
		"trace_callMany": func(t *testing.T, be *fakeBackend, args ethapi.TransactionArgs) []byte {
			calls := []ethapi.CallRequest{{Args: args, TraceTypes: []string{"trace"}}}
			res, err := ethapi.NewPublicTxTraceAPI(be, 100_000, 1000).CallMany(t.Context(), calls, latest, nil)
			require.NoError(t, err)
			require.Len(t, res, 1)
			return res[0].Output
		},
	}
	upgrades := map[string]opera.Upgrades{
		"Brio":  opera.GetBrioUpgrades(),
		"Canto": opera.GetCantoUpgrades(),
	}

	for upgradeName, upgrades := range upgrades {
		for callName, call := range calls {
			t.Run(upgradeName+"/"+callName, func(t *testing.T) {
				contract := common.Address{0x42}
				be := newExecutionGasCapBackend(t, upgrades, contract)
				maxTxGas := be.MaxGasLimit()
				require.Greater(t, be.RPCGasCap(), maxTxGas)

				args := ethapi.TransactionArgs{To: &contract, Gas: ToHexUint64(be.RPCGasCap())}
				gasLeft := new(big.Int).SetBytes(call(t, be, args))
				require.Greater(t, gasLeft.Uint64(), maxTxGas)
			})
		}
	}
}

func Test_DoCall_CapsTheExecutionGasFromCantoOnUnlessSkipped(t *testing.T) {
	tests := map[string]struct {
		upgrades            opera.Upgrades
		skipExecutionGasCap bool
		capped              bool
	}{
		"Brio":                {upgrades: opera.GetBrioUpgrades()},
		"Canto":               {upgrades: opera.GetCantoUpgrades(), capped: true},
		"Canto without a cap": {upgrades: opera.GetCantoUpgrades(), skipExecutionGasCap: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			contract := common.Address{0x42}
			be := newExecutionGasCapBackend(t, test.upgrades, contract)
			maxTxGas := be.MaxGasLimit()

			args := ethapi.TransactionArgs{To: &contract, Gas: ToHexUint64(be.RPCGasCap())}
			res, err := ethapi.DoCall(
				t.Context(), be, args, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, nil, 0, be.RPCGasCap(), nil, test.skipExecutionGasCap,
			)
			require.NoError(t, err)
			require.NoError(t, res.Err)

			gasLeft := new(big.Int).SetBytes(res.Return()).Uint64()
			if test.capped {
				require.Less(t, gasLeft, maxTxGas)
			} else {
				require.Greater(t, gasLeft, maxTxGas)
			}
		})
	}
}
