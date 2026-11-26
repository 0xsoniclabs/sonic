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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/txtrace"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

// newHashChainBackend returns a mock Backend serving the headers of a chain
// of blocks 0 to head, where block i has hash chainHash(i).
func newHashChainBackend(t *testing.T, head uint64) (*MockBackend, *evmcore.EvmHeader) {
	t.Helper()
	backend := NewMockBackend(gomock.NewController(t))
	headers := make([]*evmcore.EvmHeader, head+1)
	for i := range headers {
		headers[i] = &evmcore.EvmHeader{
			Number:     new(big.Int).SetUint64(uint64(i)),
			Hash:       chainHash(uint64(i)),
			ParentHash: chainHash(uint64(i) - 1),
		}
	}
	backend.EXPECT().HeaderByNumber(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, n rpc.BlockNumber) (*evmcore.EvmHeader, error) {
			if n < 0 || uint64(n) > head {
				return nil, nil
			}
			return headers[n], nil
		}).AnyTimes()
	return backend, headers[head]
}

func chainHash(n uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(n + 1))
}

func TestOverriddenBlockContext_BaseBlockHashIsKnownOnlyToNextBlock(t *testing.T) {
	const baseNumber = 10
	number := func(n int64) *BlockOverrides {
		return &BlockOverrides{Number: (*hexutil.Big)(big.NewInt(n))}
	}
	time := hexutil.Uint64(1234)

	tests := map[string]struct {
		overrides  *BlockOverrides
		wantNumber uint64
		// wantHashes maps block numbers to the hash BLOCKHASH resolves them to.
		wantHashes map[uint64]common.Hash
	}{
		"no overrides": {
			overrides:  nil,
			wantNumber: baseNumber,
			wantHashes: map[uint64]common.Hash{baseNumber: {}, baseNumber - 1: chainHash(baseNumber - 1)},
		},
		"number not overridden": {
			overrides:  &BlockOverrides{Time: &time},
			wantNumber: baseNumber,
			wantHashes: map[uint64]common.Hash{baseNumber: {}, baseNumber - 1: chainHash(baseNumber - 1)},
		},
		"earlier block": {
			overrides:  number(baseNumber - 1),
			wantNumber: baseNumber - 1,
			wantHashes: map[uint64]common.Hash{baseNumber: {}, baseNumber - 1: chainHash(baseNumber - 1)},
		},
		"same block": {
			overrides:  number(baseNumber),
			wantNumber: baseNumber,
			wantHashes: map[uint64]common.Hash{baseNumber: {}, baseNumber - 1: chainHash(baseNumber - 1)},
		},
		"next block": {
			overrides:  number(baseNumber + 1),
			wantNumber: baseNumber + 1,
			wantHashes: map[uint64]common.Hash{
				baseNumber:     chainHash(baseNumber),
				baseNumber - 1: chainHash(baseNumber - 1),
				baseNumber - 5: chainHash(baseNumber - 5),
			},
		},
		"two blocks ahead": {
			overrides:  number(baseNumber + 2),
			wantNumber: baseNumber + 2,
			wantHashes: map[uint64]common.Hash{
				baseNumber + 1: {},
				baseNumber:     {},
				baseNumber - 1: chainHash(baseNumber - 1),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			backend, base := newHashChainBackend(t, baseNumber)

			blockCtx := overriddenBlockContext(t.Context(), backend, base, test.overrides)

			require.Equal(t, test.wantNumber, blockCtx.BlockNumber.Uint64())
			for n, want := range test.wantHashes {
				require.Equal(t, want, blockCtx.GetHash(n), "hash of block %d", n)
			}
		})
	}
}

func TestOverriddenBlockContext_EarlierBlockLooksUpOnlyTheAskedHeader(t *testing.T) {
	const baseNumber = 1_000_000
	base := &evmcore.EvmHeader{Number: big.NewInt(baseNumber), Hash: chainHash(baseNumber)}
	backend := NewMockBackend(gomock.NewController(t))
	backend.EXPECT().HeaderByNumber(gomock.Any(), rpc.BlockNumber(100)).
		Return(&evmcore.EvmHeader{Number: big.NewInt(100), Hash: chainHash(100)}, nil).
		Times(1)
	overrides := &BlockOverrides{Number: (*hexutil.Big)(big.NewInt(300))}

	blockCtx := overriddenBlockContext(t.Context(), backend, base, overrides)

	require.Equal(t, chainHash(100), blockCtx.GetHash(100))
	require.Equal(t, chainHash(100), blockCtx.GetHash(100), "second lookup")
}

func TestOverriddenBlockContext_AppliesOtherOverrides(t *testing.T) {
	backend, base := newHashChainBackend(t, 3)
	time := hexutil.Uint64(1234)
	overrides := &BlockOverrides{
		Number:  (*hexutil.Big)(big.NewInt(4)),
		Time:    &time,
		BaseFee: (*hexutil.Big)(big.NewInt(77)),
	}

	blockCtx := overriddenBlockContext(t.Context(), backend, base, overrides)

	require.Equal(t, uint64(4), blockCtx.BlockNumber.Uint64())
	require.Equal(t, uint64(1234), blockCtx.Time)
	require.Equal(t, big.NewInt(77), blockCtx.BaseFee)
}

func TestApplyNextBlockSystemCalls_NotNextBlock_DoesNothing(t *testing.T) {
	base := &evmcore.EvmHeader{Number: big.NewInt(10), Hash: common.Hash{10}}
	for _, number := range []int64{9, 10, 12} {
		t.Run(big.NewInt(number).String(), func(t *testing.T) {
			// Neither mock expects a call: building an EVM or touching the
			// state would fail the test.
			backend := NewMockBackend(gomock.NewController(t))
			stateDb := state.NewMockStateDB(gomock.NewController(t))
			blockCtx := &vm.BlockContext{BlockNumber: big.NewInt(number)}

			err := applyNextBlockSystemCalls(t.Context(), backend, stateDb, base, blockCtx)
			require.NoError(t, err)
		})
	}
}

func TestApplyNextBlockSystemCalls_NextBlockBeforePrague_DoesNotTouchState(t *testing.T) {
	base := &evmcore.EvmHeader{Number: big.NewInt(10), Hash: common.Hash{10}}
	// The state mock expects no call.
	stateDb := state.NewMockStateDB(gomock.NewController(t))
	backend := newSystemCallTestBackend(t, stateDb, opera.GetSonicUpgrades())
	blockCtx := &vm.BlockContext{BlockNumber: big.NewInt(11)}

	err := applyNextBlockSystemCalls(t.Context(), backend, stateDb, base, blockCtx)
	require.NoError(t, err)
}

func TestApplyNextBlockSystemCalls_NextBlockAfterPrague_StoresBaseBlockHash(t *testing.T) {
	const baseNumber = 10
	base := &evmcore.EvmHeader{Number: big.NewInt(baseNumber), Hash: common.Hash{0xab}}
	stateDb := state.NewMockStateDB(gomock.NewController(t))
	slot := common.BigToHash(big.NewInt(baseNumber % params.HistoryServeWindow))
	stateDb.EXPECT().SetState(params.HistoryStorageAddress, slot, base.Hash).Times(1)
	expectHistoryContractCall(stateDb)
	backend := newSystemCallTestBackend(t, stateDb, opera.GetAllegroUpgrades())
	blockCtx := &vm.BlockContext{
		BlockNumber: big.NewInt(baseNumber + 1),
		Random:      &common.Hash{}, // post-merge rules, needed for PUSH0
		CanTransfer: evmcore.CanTransfer,
		Transfer:    evmcore.Transfer,
	}

	err := applyNextBlockSystemCalls(t.Context(), backend, stateDb, base, blockCtx)
	require.NoError(t, err)
}

func TestApplyNextBlockSystemCalls_PropagatesErrors(t *testing.T) {
	base := &evmcore.EvmHeader{Number: big.NewInt(10)}
	blockCtx := &vm.BlockContext{BlockNumber: big.NewInt(11)}
	stateDb := state.NewMockStateDB(gomock.NewController(t))
	injected := errors.New("injected")

	t.Run("vm config", func(t *testing.T) {
		backend := NewMockBackend(gomock.NewController(t))
		backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).Return(nil, injected)
		err := applyNextBlockSystemCalls(t.Context(), backend, stateDb, base, blockCtx)
		require.ErrorIs(t, err, injected)
	})
	t.Run("evm", func(t *testing.T) {
		backend := NewMockBackend(gomock.NewController(t))
		backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).
			Return(&opera.Rules{Upgrades: opera.GetAllegroUpgrades()}, nil)
		backend.EXPECT().GetEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, nil, injected)
		err := applyNextBlockSystemCalls(t.Context(), backend, stateDb, base, blockCtx)
		require.ErrorIs(t, err, injected)
	})
}

// newSystemCallTestBackend returns a mock Backend building EVMs with the
// given upgrades on the given state, using the block context it is given.
func newSystemCallTestBackend(t *testing.T, stateDb state.StateDB, upgrades opera.Upgrades) *MockBackend {
	t.Helper()
	backend := NewMockBackend(gomock.NewController(t))
	chainConfig := makeChainConfig(upgrades)
	backend.EXPECT().GetNetworkRules(gomock.Any(), gomock.Any()).
		Return(&opera.Rules{Upgrades: upgrades}, nil)
	backend.EXPECT().GetEVM(gomock.Any(), stateDb, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ vm.StateDB, _ *evmcore.EvmHeader, vmConfig *vm.Config, blockCtx *vm.BlockContext) (*vm.EVM, func() error, error) {
			require.True(t, vmConfig.NoBaseFee)
			return vm.NewEVM(*blockCtx, stateDb, chainConfig, *vmConfig), func() error { return nil }, nil
		})
	return backend
}

// expectHistoryContractCall sets the state expectations of a system call to
// the EIP-2935 history contract, apart from the storage write.
func expectHistoryContractCall(stateDb *state.MockStateDB) {
	any := gomock.Any()
	code := params.HistoryStorageCode
	stateDb.EXPECT().GetCode(params.HistoryStorageAddress).Return(code).AnyTimes()
	stateDb.EXPECT().GetCodeHash(params.HistoryStorageAddress).Return(crypto.Keccak256Hash(code)).AnyTimes()
	stateDb.EXPECT().AddressInAccessList(any).Return(true).AnyTimes()
	stateDb.EXPECT().SlotInAccessList(any, any).Return(true, true).AnyTimes()
	stateDb.EXPECT().AddAddressToAccessList(any).AnyTimes()
	stateDb.EXPECT().AddSlotToAccessList(any, any).AnyTimes()
	stateDb.EXPECT().GetState(any, any).Return(common.Hash{}).AnyTimes()
	stateDb.EXPECT().GetStateAndCommittedState(any, any).Return(common.Hash{}, common.Hash{}).AnyTimes()
	stateDb.EXPECT().AddRefund(any).AnyTimes()
	stateDb.EXPECT().SubRefund(any).AnyTimes()
	stateDb.EXPECT().Finalise(any).AnyTimes()
	setExpectedStateCalls(stateDb)
}

func TestIsNextBlock(t *testing.T) {
	base := &evmcore.EvmHeader{Number: big.NewInt(10)}
	tests := map[string]struct {
		blockCtx *vm.BlockContext
		base     *evmcore.EvmHeader
		want     bool
	}{
		"next block":       {&vm.BlockContext{BlockNumber: big.NewInt(11)}, base, true},
		"same block":       {&vm.BlockContext{BlockNumber: big.NewInt(10)}, base, false},
		"two blocks ahead": {&vm.BlockContext{BlockNumber: big.NewInt(12)}, base, false},
		"earlier block":    {&vm.BlockContext{BlockNumber: big.NewInt(9)}, base, false},
		"no block number":  {&vm.BlockContext{}, base, false},
		"no base number":   {&vm.BlockContext{BlockNumber: big.NewInt(11)}, &evmcore.EvmHeader{}, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, isNextBlock(test.blockCtx, test.base))
		})
	}
}

func TestApplyNextBlockSystemCalls_StopsWhenTheContextEnds(t *testing.T) {
	const baseNumber = 10
	base := &evmcore.EvmHeader{Number: big.NewInt(baseNumber), Hash: common.Hash{0xab}}
	blockCtx := &vm.BlockContext{
		BlockNumber: big.NewInt(baseNumber + 1),
		Random:      &common.Hash{}, // post-merge rules, needed for PUSH0
		CanTransfer: evmcore.CanTransfer,
		Transfer:    evmcore.Transfer,
	}
	// A history contract replaced by a loop reading storage:
	// JUMPDEST, PUSH0, SLOAD, POP, PUSH0, JUMP. Without being stopped, it
	// reads storage about 270,000 times before its gas runs out.
	loop := []byte{0x5b, 0x5f, 0x54, 0x50, 0x5f, 0x56}

	t.Run("ended before", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		// The state mock expects no call.
		stateDb := state.NewMockStateDB(gomock.NewController(t))
		backend := newSystemCallTestBackend(t, stateDb, opera.GetAllegroUpgrades())

		err := applyNextBlockSystemCalls(ctx, backend, stateDb, base, blockCtx)
		require.ErrorIs(t, err, context.Canceled)
	})
	t.Run("ends during", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		stateDb := state.NewMockStateDB(gomock.NewController(t))
		stateDb.EXPECT().GetCode(params.HistoryStorageAddress).DoAndReturn(func(common.Address) []byte {
			cancel()
			return loop
		}).AnyTimes()
		stateDb.EXPECT().GetCodeHash(params.HistoryStorageAddress).Return(crypto.Keccak256Hash(loop)).AnyTimes()
		reads := 0
		stateDb.EXPECT().GetState(params.HistoryStorageAddress, gomock.Any()).DoAndReturn(func(common.Address, common.Hash) common.Hash {
			reads++
			return common.Hash{}
		}).AnyTimes()
		stateDb.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()
		expectHistoryContractCall(stateDb)
		backend := newSystemCallTestBackend(t, stateDb, opera.GetAllegroUpgrades())

		err := applyNextBlockSystemCalls(ctx, backend, stateDb, base, blockCtx)
		require.ErrorIs(t, err, context.Canceled)
		require.Less(t, reads, 100_000, "storage reads after the context ended")
	})
}

func TestOverriddenBlockContext_LaterBlocksAreUnknownWithoutHeaderLookups(t *testing.T) {
	const baseNumber = 10_000
	base := &evmcore.EvmHeader{Number: big.NewInt(baseNumber), Hash: chainHash(baseNumber)}
	backend := NewMockBackend(gomock.NewController(t)) // expects no header lookup
	overrides := &BlockOverrides{Number: (*hexutil.Big)(big.NewInt(baseNumber + 256))}

	blockCtx := overriddenBlockContext(t.Context(), backend, base, overrides)

	for _, n := range []uint64{baseNumber, baseNumber + 1, baseNumber + 255} {
		require.Equal(t, common.Hash{}, blockCtx.GetHash(n), "hash of block %d", n)
	}
}
