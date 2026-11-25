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
	"github.com/0xsoniclabs/sonic/txtrace"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// gasPriceAndBaseFeeCode returns GASPRICE and BASEFEE as two 32-byte words.
var gasPriceAndBaseFeeCode = []byte{
	byte(vm.GASPRICE), byte(vm.PUSH0), byte(vm.MSTORE),
	byte(vm.BASEFEE), byte(vm.PUSH1), 0x20, byte(vm.MSTORE),
	byte(vm.PUSH1), 0x40, byte(vm.PUSH0), byte(vm.RETURN),
}

// revertIfGasPriceBelowBaseFeeCode reverts if GASPRICE < BASEFEE and stops
// otherwise.
var revertIfGasPriceBelowBaseFeeCode = []byte{
	byte(vm.BASEFEE), byte(vm.GASPRICE), byte(vm.LT), // GASPRICE < BASEFEE
	byte(vm.PUSH1), 0x07, byte(vm.JUMPI),
	byte(vm.STOP),
	byte(vm.JUMPDEST), byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.REVERT),
}

// gasPriceTestCase is a call on a block with the given header base fee and
// block overrides, with the GASPRICE and BASEFEE the call must see. A nil
// gasPrice makes it a dynamic-fee call without a tip, which pays exactly the
// base fee; otherwise it is a legacy call with that gas price.
type gasPriceTestCase struct {
	headerBaseFee *big.Int
	overrides     *BlockOverrides
	gasPrice      *big.Int
	wantBaseFee   *big.Int
	wantGasPrice  *big.Int
}

var gasPriceTestCases = map[string]gasPriceTestCase{
	"no overrides": {
		headerBaseFee: big.NewInt(100),
		wantBaseFee:   big.NewInt(100),
		wantGasPrice:  big.NewInt(100),
	},
	"base fee overridden": {
		headerBaseFee: big.NewInt(100),
		overrides:     &BlockOverrides{BaseFee: (*hexutil.Big)(big.NewInt(1000))},
		wantBaseFee:   big.NewInt(1000),
		wantGasPrice:  big.NewInt(1000),
	},
	"base fee overridden on block without base fee": {
		headerBaseFee: nil,
		overrides:     &BlockOverrides{BaseFee: (*hexutil.Big)(big.NewInt(1000))},
		wantBaseFee:   big.NewInt(1000),
		wantGasPrice:  big.NewInt(1000),
	},
	"legacy gas price with base fee overridden": {
		headerBaseFee: big.NewInt(100),
		overrides:     &BlockOverrides{BaseFee: (*hexutil.Big)(big.NewInt(1000))},
		gasPrice:      big.NewInt(2000),
		wantBaseFee:   big.NewInt(1000),
		wantGasPrice:  big.NewInt(2000),
	},
}

const gasPriceTestGas = 100_000

func (test gasPriceTestCase) args(from, contract common.Address) TransactionArgs {
	gas := hexutil.Uint64(gasPriceTestGas)
	args := TransactionArgs{From: &from, To: &contract, Gas: &gas}
	if test.gasPrice != nil {
		args.GasPrice = (*hexutil.Big)(test.gasPrice)
	} else {
		args.MaxFeePerGas = (*hexutil.Big)(big.NewInt(1_000_000))
		args.MaxPriorityFeePerGas = (*hexutil.Big)(big.NewInt(0))
	}
	return args
}

func requireGasPriceAndBaseFee(t *testing.T, test gasPriceTestCase, returnData []byte) {
	t.Helper()
	require.Len(t, returnData, 64)
	gasPrice := new(big.Int).SetBytes(returnData[:32])
	baseFee := new(big.Int).SetBytes(returnData[32:])
	require.Equal(t, test.wantBaseFee, baseFee, "BASEFEE")
	require.Equal(t, test.wantGasPrice, gasPrice, "GASPRICE")
}

var (
	gasPriceTestSender   = common.Address{0x1}
	gasPriceTestContract = common.Address{0x42}
)

func TestDoCall_GasPriceFollowsOverriddenBaseFee(t *testing.T) {
	for name, test := range gasPriceTestCases {
		t.Run(name, func(t *testing.T) {
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			backend := newCallTestBackend(t, stateDb, test.headerBaseFee, gasPriceTestContract, gasPriceAndBaseFeeCode)

			result, err := DoCall(t.Context(), backend, test.args(gasPriceTestSender, gasPriceTestContract),
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, test.overrides, time.Second, 10_000_000, nil)
			require.NoError(t, err)
			require.False(t, result.Failed(), "call failed: %v", result.Err)
			requireGasPriceAndBaseFee(t, test, result.ReturnData)
		})
	}
}

func TestDoCall_PreArgsGasPriceFollowsOverriddenBaseFee(t *testing.T) {
	for name, test := range gasPriceTestCases {
		t.Run(name, func(t *testing.T) {
			preSender := common.Address{0x2}
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			// Buying gas charges the sender gas limit times gas price, which
			// reveals the gas price the previous transaction is priced with.
			charge := new(big.Int).Mul(big.NewInt(gasPriceTestGas), test.wantGasPrice)
			stateDb.EXPECT().SubBalance(preSender, uint256.MustFromBig(charge), gomock.Any())
			backend := newCallTestBackend(t, stateDb, test.headerBaseFee, gasPriceTestContract, gasPriceAndBaseFeeCode)

			preArgs := []TransactionArgs{test.args(preSender, gasPriceTestContract)}
			result, err := DoCall(t.Context(), backend, test.args(gasPriceTestSender, gasPriceTestContract),
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, test.overrides, time.Second, 10_000_000, preArgs)
			require.NoError(t, err)
			require.False(t, result.Failed(), "call failed: %v", result.Err)
		})
	}
}

func TestDoEstimateGas_GasPriceFollowsOverriddenBaseFee(t *testing.T) {
	for name, test := range gasPriceTestCases {
		t.Run(name, func(t *testing.T) {
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			backend := newCallTestBackend(t, stateDb, test.headerBaseFee, gasPriceTestContract, revertIfGasPriceBelowBaseFeeCode)

			// The contract reverts if GASPRICE < BASEFEE, which fails the estimation.
			_, err := DoEstimateGas(t.Context(), backend, test.args(gasPriceTestSender, gasPriceTestContract),
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, test.overrides, 10_000_000, nil)
			require.NoError(t, err)
		})
	}
}

func TestTraceCall_GasPriceFollowsOverriddenBaseFee(t *testing.T) {
	for name, test := range gasPriceTestCases {
		t.Run(name, func(t *testing.T) {
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			backend := newCallTestBackend(t, stateDb, test.headerBaseFee, gasPriceTestContract, gasPriceAndBaseFeeCode)
			api := NewPublicDebugAPI(backend, 10000, 10000, false)

			tracer := "callTracer"
			config := &TraceCallConfig{BlockOverrides: test.overrides}
			config.Tracer = &tracer
			res, err := api.TraceCall(t.Context(), test.args(gasPriceTestSender, gasPriceTestContract),
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

// trace_call and trace_callMany take no block overrides, so the call keeps
// the block's base fee even if the config carries block overrides.
func TestTxTraceCalls_IgnoreBlockOverridesAndUseBlockBaseFee(t *testing.T) {
	headerBaseFee := big.NewInt(100)
	test := gasPriceTestCase{wantBaseFee: headerBaseFee, wantGasPrice: headerBaseFee}
	config := &TraceCallConfig{BlockOverrides: &BlockOverrides{BaseFee: (*hexutil.Big)(big.NewInt(1000))}}
	args := test.args(gasPriceTestSender, gasPriceTestContract)

	calls := map[string]func(*PublicTxTraceAPI) (*txtrace.TraceCallResult, error){
		"trace_call": func(api *PublicTxTraceAPI) (*txtrace.TraceCallResult, error) {
			return api.Call(t.Context(), args, []string{TraceTypeTrace}, rpc.BlockNumberOrHashWithNumber(1), config)
		},
		"trace_callMany": func(api *PublicTxTraceAPI) (*txtrace.TraceCallResult, error) {
			results, err := api.CallMany(t.Context(),
				[]CallRequest{{Args: args, TraceTypes: []string{TraceTypeTrace}}},
				rpc.BlockNumberOrHashWithNumber(1), config)
			if err != nil {
				return nil, err
			}
			require.Len(t, results, 1)
			return results[0], nil
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			backend := newCallTestBackend(t, stateDb, headerBaseFee, gasPriceTestContract, gasPriceAndBaseFeeCode)
			api := NewPublicTxTraceAPI(backend, 10000, 10000)

			result, err := call(api)
			require.NoError(t, err)
			requireGasPriceAndBaseFee(t, test, result.Output)
		})
	}
}

// newCallTestBackend builds a mock Backend able to run calls and traces
// against a single contract with the given code, on a block with the given
// base fee. The EVM uses the block context the caller passes, so block
// overrides apply. Expectations already set on stateDb take precedence over
// the defaults added here.
func newCallTestBackend(t *testing.T, stateDb *state.MockStateDB, baseFee *big.Int, contract common.Address, code []byte) *MockBackend {
	t.Helper()
	backend := NewMockBackend(gomock.NewController(t))

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
	backend.EXPECT().HeaderByNumber(any, any).Return(header, nil).AnyTimes()
	backend.EXPECT().MaxGasLimit().Return(uint64(10_000_000)).AnyTimes()
	backend.EXPECT().RPCEVMTimeout().Return(time.Duration(0)).AnyTimes()
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
	stateDb.EXPECT().RevertToSnapshot(any).AnyTimes()
	setExpectedStateCalls(stateDb)
	return backend
}

// logBaseFeeCode emits BASEFEE as the data of a LOG0.
var logBaseFeeCode = []byte{
	byte(vm.BASEFEE), byte(vm.PUSH0), byte(vm.MSTORE),
	byte(vm.PUSH1), 0x20, byte(vm.PUSH0), byte(vm.LOG0),
	byte(vm.STOP),
}

func TestDoCall_PreArgsRunWithTheFeesOfTheirOwnPricing(t *testing.T) {
	headerBaseFee := big.NewInt(100)
	// A dynamic-fee call without a tip pays exactly the base fee.
	pricedArgs := gasPriceTestCase{}.args
	unpriced := func(from, to common.Address) TransactionArgs {
		gas := hexutil.Uint64(gasPriceTestGas)
		return TransactionArgs{From: &from, To: &to, Gas: &gas}
	}

	tests := map[string]struct {
		preArgs, call func(from, to common.Address) TransactionArgs
		wantBaseFees  []string
	}{
		"a priced previous transaction before an unpriced call": {
			preArgs:      pricedArgs,
			call:         unpriced,
			wantBaseFees: []string{"100", "0"},
		},
		"an unpriced previous transaction before a priced call": {
			preArgs:      unpriced,
			call:         pricedArgs,
			wantBaseFees: []string{"0", "100"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			var gotBaseFees []string
			stateDb.EXPECT().AddLog(gomock.Any()).Do(func(l *types.Log) {
				gotBaseFees = append(gotBaseFees, new(big.Int).SetBytes(l.Data).String())
			}).AnyTimes()
			backend := newCallTestBackend(t, stateDb, headerBaseFee, gasPriceTestContract, logBaseFeeCode)

			preArgs := []TransactionArgs{test.preArgs(common.Address{0x2}, gasPriceTestContract)}
			result, err := DoCall(t.Context(), backend, test.call(gasPriceTestSender, gasPriceTestContract),
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, nil, time.Second, 10_000_000, preArgs)
			require.NoError(t, err)
			require.False(t, result.Failed(), "call failed: %v", result.Err)
			require.Equal(t, test.wantBaseFees, gotBaseFees, "BASEFEE of the previous transaction and the call")
		})
	}
}

// Block processing runs with a difficulty of 1 unless the header carries a
// PREVRANDAO. Calls are not part of a block and must instead see the
// difficulty the header reports, so they must not rely on the default
// context of GetEVM.
func TestCalls_RunInTheBlockContextOfTheHeader(t *testing.T) {
	calls := map[string]func(Backend) error{
		"eth_call": func(b Backend) error {
			_, err := DoCall(t.Context(), b, getTxArgs(t),
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, nil, 0, 10_000_000, nil)
			return err
		},
		"eth_createAccessList": func(b Backend) error {
			nonce := hexutil.Uint64(0)
			gas := hexutil.Uint64(100_000)
			args := getTxArgs(t)
			args.Nonce = &nonce
			args.Gas = &gas
			// A missing gas price would be filled in with a suggested one.
			args.GasPrice = (*hexutil.Big)(big.NewInt(0))
			_, _, _, err := AccessList(t.Context(), b, rpc.BlockNumberOrHashWithNumber(1), args)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			stateDb := state.NewMockStateDB(ctrl)

			// No PREVRANDAO, so block processing would use a difficulty of 1.
			block := &evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{
				Number:     big.NewInt(1),
				BaseFee:    big.NewInt(100),
				Difficulty: big.NewInt(0),
			}}
			chainConfig := &params.ChainConfig{ChainID: big.NewInt(1), BerlinBlock: big.NewInt(0), LondonBlock: big.NewInt(0)}

			any := gomock.Any()
			backend.EXPECT().StateAndBlockByNumberOrHash(any, any).Return(stateDb, block, nil).AnyTimes()
			backend.EXPECT().ChainConfig(any).Return(chainConfig).AnyTimes()
			backend.EXPECT().GetNetworkRules(any, any).Return(&opera.Rules{}, nil).AnyTimes()
			backend.EXPECT().RPCGasCap().Return(uint64(10_000_000)).AnyTimes()
			backend.EXPECT().MaxGasLimit().Return(uint64(10_000_000)).AnyTimes()
			backend.EXPECT().ChainID().Return(big.NewInt(1)).AnyTimes()
			backend.EXPECT().CurrentBlock().Return(block).AnyTimes()
			stateDb.EXPECT().Copy().Return(stateDb).AnyTimes()
			stateDb.EXPECT().GetBalance(any).Return(uint256.NewInt(1e18)).AnyTimes()
			setExpectedStateCalls(stateDb)

			var contexts []*vm.BlockContext
			backend.EXPECT().GetEVM(any, any, any, any, any).DoAndReturn(
				func(_ context.Context, _ state.StateDB, _ *evmcore.EvmHeader, vmConfig *vm.Config, blockCtx *vm.BlockContext) (*vm.EVM, func() error, error) {
					contexts = append(contexts, blockCtx)
					require.NotNil(t, blockCtx, "the call must pass its block context")
					return vm.NewEVM(*blockCtx, stateDb, chainConfig, *vmConfig), func() error { return nil }, nil
				}).AnyTimes()

			require.NoError(t, call(backend))
			require.NotEmpty(t, contexts, "the call must have created an EVM")
			for _, blockCtx := range contexts {
				require.Zero(t, blockCtx.Difficulty.Sign(), "DIFFICULTY")
			}
		})
	}
}

// A block without a base fee prices calls as legacy calls, whose tip is not
// checked against a fee cap. Block overrides that leave the base fee alone
// must keep it that way.
func TestDoCall_BlockWithoutBaseFeePricesLegacyCallsUnlessBaseFeeIsOverridden(t *testing.T) {
	blockTime := hexutil.Uint64(1234)
	tests := map[string]*BlockOverrides{
		"no overrides":               nil,
		"overrides without base fee": {Time: &blockTime},
	}
	for name, overrides := range tests {
		t.Run(name, func(t *testing.T) {
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			backend := newCallTestBackend(t, stateDb, nil, gasPriceTestContract, gasPriceAndBaseFeeCode)

			// A tip without a fee cap is rejected as a dynamic-fee call.
			from, to := gasPriceTestSender, gasPriceTestContract
			gas := hexutil.Uint64(gasPriceTestGas)
			args := TransactionArgs{From: &from, To: &to, Gas: &gas,
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(7))}

			result, err := DoCall(t.Context(), backend, args,
				rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
				nil, overrides, time.Second, 10_000_000, nil)
			require.NoError(t, err)
			require.False(t, result.Failed(), "call failed: %v", result.Err)
			require.Len(t, result.ReturnData, 64)
			require.Zero(t, new(big.Int).SetBytes(result.ReturnData[:32]).Sign(), "GASPRICE")
			require.Zero(t, new(big.Int).SetBytes(result.ReturnData[32:]).Sign(), "BASEFEE")
		})
	}
}
