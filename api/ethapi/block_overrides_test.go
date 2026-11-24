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
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBlockOverrides_baseFeeOr_ReturnsOverrideOrFallback(t *testing.T) {
	fallback := big.NewInt(100)
	override := big.NewInt(1000)

	tests := map[string]struct {
		overrides *BlockOverrides
		want      *big.Int
	}{
		"no overrides": {
			overrides: nil,
			want:      fallback,
		},
		"base fee not overridden": {
			overrides: &BlockOverrides{Number: (*hexutil.Big)(big.NewInt(5))},
			want:      fallback,
		},
		"base fee overridden": {
			overrides: &BlockOverrides{BaseFee: (*hexutil.Big)(override)},
			want:      override,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, test.overrides.baseFeeOr(fallback))
		})
	}
}

// gasPriceAndBaseFeeCode returns GASPRICE and BASEFEE as two 32-byte words.
var gasPriceAndBaseFeeCode = []byte{
	byte(vm.GASPRICE), byte(vm.PUSH0), byte(vm.MSTORE),
	byte(vm.BASEFEE), byte(vm.PUSH1), 0x20, byte(vm.MSTORE),
	byte(vm.PUSH1), 0x40, byte(vm.PUSH0), byte(vm.RETURN),
}

// gasPriceTestCase is a call on a block with base fee 100, optionally
// overridden to 1000, with the GASPRICE and BASEFEE the call must see.
type gasPriceTestCase struct {
	overrides    *BlockOverrides
	wantBaseFee  *big.Int
	wantGasPrice *big.Int
}

var (
	gasPriceTestHeaderBaseFee = big.NewInt(100)
	gasPriceTestCases         = map[string]gasPriceTestCase{
		"no overrides": {
			overrides:    nil,
			wantBaseFee:  big.NewInt(100),
			wantGasPrice: big.NewInt(100),
		},
		"base fee overridden": {
			overrides:    &BlockOverrides{BaseFee: (*hexutil.Big)(big.NewInt(1000))},
			wantBaseFee:  big.NewInt(1000),
			wantGasPrice: big.NewInt(1000),
		},
	}
)

// gasPriceTestArgs is a dynamic-fee call without a tip, which pays exactly
// the base fee.
func gasPriceTestArgs(contract common.Address) TransactionArgs {
	gas := hexutil.Uint64(100_000)
	return TransactionArgs{
		From:                 &common.Address{0x1},
		To:                   &contract,
		Gas:                  &gas,
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1_000_000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(0)),
	}
}

func requireGasPriceAndBaseFee(t *testing.T, test gasPriceTestCase, returnData []byte) {
	t.Helper()
	require.Len(t, returnData, 64)
	gasPrice := new(big.Int).SetBytes(returnData[:32])
	baseFee := new(big.Int).SetBytes(returnData[32:])
	require.Equal(t, test.wantBaseFee, baseFee, "BASEFEE")
	require.Equal(t, test.wantGasPrice, gasPrice, "GASPRICE")
}

func TestDoCall_GasPriceFollowsOverriddenBaseFee(t *testing.T) {
	for name, test := range gasPriceTestCases {
		t.Run(name, func(t *testing.T) {
			contract := common.Address{0x42}
			backend := newCallTestBackend(t, gasPriceTestHeaderBaseFee, contract, gasPriceAndBaseFeeCode)

			result, err := DoCall(t.Context(), backend, gasPriceTestArgs(contract),
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, test.overrides, time.Second, 10_000_000, nil)
			require.NoError(t, err)
			require.False(t, result.Failed(), "call failed: %v", result.Err)
			requireGasPriceAndBaseFee(t, test, result.ReturnData)
		})
	}
}

func TestTraceCall_GasPriceFollowsOverriddenBaseFee(t *testing.T) {
	for name, test := range gasPriceTestCases {
		t.Run(name, func(t *testing.T) {
			contract := common.Address{0x42}
			backend := newCallTestBackend(t, gasPriceTestHeaderBaseFee, contract, gasPriceAndBaseFeeCode)
			api := NewPublicDebugAPI(backend, 10000, 10000, false)

			tracer := "callTracer"
			config := &TraceCallConfig{BlockOverrides: test.overrides}
			config.Tracer = &tracer
			res, err := api.TraceCall(t.Context(), gasPriceTestArgs(contract),
				rpc.BlockNumberOrHashWithNumber(1), config)
			require.NoError(t, err)

			var trace struct {
				Error  string        `json:"error"`
				Output hexutil.Bytes `json:"output"`
			}
			raw, ok := res.(json.RawMessage)
			require.True(t, ok, "unexpected result type %T", res)
			require.NoError(t, json.Unmarshal(raw, &trace))
			require.Empty(t, trace.Error)
			requireGasPriceAndBaseFee(t, test, trace.Output)
		})
	}
}

// newCallTestBackend builds a mock Backend able to run DoCall and TraceCall
// against a single contract with the given code, on a block with the given base fee.
// The EVM uses the block context the caller passes, so block overrides apply.
func newCallTestBackend(t *testing.T, baseFee *big.Int, contract common.Address, code []byte) *MockBackend {
	t.Helper()
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)
	stateDb := state.NewMockStateDB(ctrl)

	header := &evmcore.EvmHeader{
		Number:     big.NewInt(1),
		BaseFee:    baseFee,
		PrevRandao: common.Hash{0x1},
	}
	block := &evmcore.EvmBlock{EvmHeader: *header}
	upgrades := opera.GetSonicUpgrades()
	chainConfig := makeChainConfig(upgrades)

	any := gomock.Any()
	backend.EXPECT().StateAndBlockByNumberOrHash(any, any).Return(stateDb, block, nil).AnyTimes()
	backend.EXPECT().BlockByNumber(any, any).Return(block, nil).AnyTimes()
	backend.EXPECT().ChainConfig(any).Return(chainConfig).AnyTimes()
	backend.EXPECT().RPCGasCap().Return(uint64(10_000_000)).AnyTimes()
	backend.EXPECT().GetNetworkRules(any, any).Return(&opera.Rules{Upgrades: upgrades}, nil).AnyTimes()
	backend.EXPECT().GetEVM(any, any, any, any, any).DoAndReturn(
		func(ctx context.Context, _ state.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockCtx *vm.BlockContext) (*vm.EVM, func() error, error) {
			if blockCtx == nil {
				bctx := GetBlockContext(ctx, backend, header)
				blockCtx = &bctx
			}
			return vm.NewEVM(*blockCtx, stateDb, chainConfig, *vmConfig), func() error { return nil }, nil
		}).AnyTimes()

	// The sender must afford the gas at the highest price the tests use.
	stateDb.EXPECT().GetBalance(any).Return(uint256.NewInt(1e18)).AnyTimes()
	stateDb.EXPECT().GetCode(contract).Return(code).AnyTimes()
	stateDb.EXPECT().GetCodeHash(contract).Return(crypto.Keccak256Hash(code)).AnyTimes()
	stateDb.EXPECT().AddressInAccessList(any).Return(false).AnyTimes()
	stateDb.EXPECT().AddAddressToAccessList(any).AnyTimes()
	stateDb.EXPECT().AddRefund(any).AnyTimes()
	stateDb.EXPECT().SubRefund(any).AnyTimes()
	setExpectedStateCalls(stateDb)
	return backend
}
