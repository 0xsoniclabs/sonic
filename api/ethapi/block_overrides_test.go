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

func TestDoCall_GasPriceFollowsOverriddenBaseFee(t *testing.T) {
	headerBaseFee := big.NewInt(100)
	overriddenBaseFee := big.NewInt(1000)

	tests := map[string]struct {
		overrides    *BlockOverrides
		wantBaseFee  *big.Int
		wantGasPrice *big.Int
	}{
		"no overrides": {
			overrides:    nil,
			wantBaseFee:  headerBaseFee,
			wantGasPrice: headerBaseFee,
		},
		"base fee overridden": {
			overrides:    &BlockOverrides{BaseFee: (*hexutil.Big)(overriddenBaseFee)},
			wantBaseFee:  overriddenBaseFee,
			wantGasPrice: overriddenBaseFee,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// The contract returns GASPRICE and BASEFEE as two 32-byte words.
			code := []byte{
				byte(vm.GASPRICE), byte(vm.PUSH0), byte(vm.MSTORE),
				byte(vm.BASEFEE), byte(vm.PUSH1), 0x20, byte(vm.MSTORE),
				byte(vm.PUSH1), 0x40, byte(vm.PUSH0), byte(vm.RETURN),
			}
			contract := common.Address{0x42}
			backend := newDoCallTestBackend(t, headerBaseFee, contract, code)

			// A dynamic-fee call without a tip pays exactly the base fee.
			maxFee := (*hexutil.Big)(big.NewInt(1_000_000))
			tip := (*hexutil.Big)(big.NewInt(0))
			gas := hexutil.Uint64(100_000)
			args := TransactionArgs{
				From:                 &common.Address{0x1},
				To:                   &contract,
				Gas:                  &gas,
				MaxFeePerGas:         maxFee,
				MaxPriorityFeePerGas: tip,
			}

			result, err := DoCall(t.Context(), backend, args,
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, test.overrides, time.Second, 10_000_000, nil)
			require.NoError(t, err)
			require.False(t, result.Failed(), "call failed: %v", result.Err)
			require.Len(t, result.ReturnData, 64)

			gasPrice := new(big.Int).SetBytes(result.ReturnData[:32])
			baseFee := new(big.Int).SetBytes(result.ReturnData[32:])
			require.Equal(t, test.wantBaseFee, baseFee, "BASEFEE")
			require.Equal(t, test.wantGasPrice, gasPrice, "GASPRICE")
		})
	}
}

// newDoCallTestBackend builds a mock Backend able to run DoCall against a
// single contract with the given code, on a block with the given base fee.
// The EVM uses the block context DoCall passes, so block overrides apply.
func newDoCallTestBackend(t *testing.T, baseFee *big.Int, contract common.Address, code []byte) *MockBackend {
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
