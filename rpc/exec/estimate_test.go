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

package exec_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestEstimateGas_AnswersTheSearchOrWhyItFails(t *testing.T) {
	h := newStateEnvironment(t)
	recipient, unknown := common.Address{0x42}, common.Hash{0x01}
	price := (*hexutil.Big)(big.NewInt(1e9))
	poor := common.Address{0x99}
	latest := chain.ReferenceTo(chain.LatestBlockNumber)
	cases := map[string]struct {
		args       exec.CallArguments
		estimation exec.Estimation
		gas        hexutil.Uint64
		err        string
	}{
		"transfer":            {args: exec.CallArguments{From: &funded, To: &recipient}, gas: 21000},
		"priced by its funds": {args: exec.CallArguments{From: &funded, To: &recipient, GasPrice: price}, gas: 21000},
		"capped by its funds": {args: exec.CallArguments{From: &funded, To: &spinner,
			MaxFeePerGas: (*hexutil.Big)(big.NewInt(1e12)), MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1))},
			err: "execution unsuccessful"},
		"funds for the value": {args: exec.CallArguments{From: &poor, To: &recipient, GasPrice: price, Value: price},
			err: "insufficient funds for transfer"},
		"both pricings": {args: exec.CallArguments{To: &recipient, GasPrice: price, MaxFeePerGas: price},
			err: "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"},
		"unknown hash": {estimation: exec.Estimation{Block: chain.BlockReference{Hash: &unknown}},
			err: "block with hash " + unknown.Hex() + " not found"},
		"block above the head": {estimation: exec.Estimation{Block: chain.ReferenceTo(5)}, err: "block does not exists"},
		"no selector":          {estimation: exec.Estimation{}, err: "invalid block number or hash"},
		"overrides of the state": {args: exec.CallArguments{To: &recipient},
			estimation: exec.Estimation{Block: latest,
				StateOverrides: &exec.StateOverride{recipient: {State: &map[common.Hash]common.Hash{}, StateDiff: &map[common.Hash]common.Hash{}}}},
			err: "account " + recipient.Hex() + " has both 'state' and 'stateDiff'"},
		"with a block override": {args: exec.CallArguments{From: &funded, To: &recipient},
			estimation: exec.Estimation{Block: latest, BlockOverrides: &exec.BlockOverrides{Number: (*hexutil.Big)(big.NewInt(9)), Time: new(hexutil.Uint64)}},
			gas:        21000},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if test.estimation == (exec.Estimation{}) && name != "no selector" {
				test.estimation.Block = latest
			}
			gas, err := h.Environment.EstimateGas(context.Background(), h.Environment.Head(), test.estimation, test.args)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.gas, gas)
			}
			require.Zero(t, h.Core.OpenStates())
		})
	}
}

func TestEstimateGas_PassesOnAFailingState(t *testing.T) {
	h := newStateEnvironment(t)
	h.Core.SetMode("State", tests.Fail)
	recipient := common.Address{0x42}
	_, err := h.Environment.EstimateGas(context.Background(), h.Environment.Head(),
		exec.Estimation{Block: chain.ReferenceTo(chain.LatestBlockNumber)}, exec.CallArguments{To: &recipient})
	require.ErrorIs(t, err, tests.ErrInjected)
}

// Each estimation runs the calls before it with their estimates, on one
// state for all of them.
func TestEstimateGasInOrder_RunsTheCallsBeforeEach(t *testing.T) {
	h := newStateEnvironment(t)
	recipient := common.Address{0x42}
	estimation := exec.Estimation{Block: chain.ReferenceTo(chain.LatestBlockNumber)}
	write := exec.CallArguments{From: &funded, To: &writer}
	states := h.Core.Calls("State")
	estimates, err := h.Environment.EstimateGasInOrder(context.Background(), h.Environment.Head(), estimation,
		[]exec.CallArguments{write, write, {From: &funded, To: &recipient}})
	require.NoError(t, err)
	require.Equal(t, states+1, h.Core.Calls("State"))
	require.Zero(t, h.Core.OpenStates())
	require.Len(t, estimates, 3)
	require.Greater(t, estimates[0], estimates[1], "the first write sets the slot, the second finds it set")
	require.Equal(t, hexutil.Uint64(21000), estimates[2])
}

func TestEstimateGasInOrder_NamesTheCallThatFails(t *testing.T) {
	h := newStateEnvironment(t)
	recipient, reverter := common.Address{0x42}, common.Address{0xc6}
	_, err := h.Environment.EstimateGasInOrder(context.Background(), h.Environment.Head(), exec.Estimation{
		Block:          chain.ReferenceTo(chain.LatestBlockNumber),
		StateOverrides: &exec.StateOverride{reverter: {Code: (*hexutil.Bytes)(&tests.RevertsWithData)}},
	}, []exec.CallArguments{{To: &recipient}, {To: &reverter}})
	require.EqualError(t, err, "failed to estimate gas for transaction 1: execution reverted")
	require.Zero(t, h.Core.OpenStates())

	estimates, err := h.Environment.EstimateGasInOrder(context.Background(), h.Environment.Head(), exec.Estimation{}, nil)
	require.NoError(t, err, "no call reads no block")
	require.Empty(t, estimates)
}

func TestNewEVM_TakesTheBlockOverrides(t *testing.T) {
	h := newStateEnvironment(t)
	stateDB, header, err := h.Environment.StateAt(h.Environment.Head(), chain.ReferenceTo(1))
	require.NoError(t, err)
	defer stateDB.Release()
	coinbase, random := common.Address{0x07}, common.Hash{0x08}
	time, gasLimit := hexutil.Uint64(11), hexutil.Uint64(12)
	evm := h.Environment.NewEVM(stateDB, header, &exec.BlockOverrides{
		Number: (*hexutil.Big)(big.NewInt(9)), Difficulty: (*hexutil.Big)(big.NewInt(10)), Time: &time, GasLimit: &gasLimit,
		Coinbase: &coinbase, Random: &random, BaseFee: (*hexutil.Big)(big.NewInt(13)), BlobBaseFee: (*hexutil.Big)(big.NewInt(14)),
	}, vm.Config{})
	context := evm.Context
	require.Equal(t, int64(9), context.BlockNumber.Int64())
	require.Equal(t, int64(10), context.Difficulty.Int64())
	require.Equal(t, uint64(11), context.Time)
	require.Equal(t, uint64(12), context.GasLimit)
	require.Equal(t, coinbase, context.Coinbase)
	require.Equal(t, &random, context.Random)
	require.Equal(t, int64(13), context.BaseFee.Int64())
	require.Equal(t, int64(14), context.BlobBaseFee.Int64())
}

// BLOCKHASH reads one header below the block, none at or above it, and
// answers the zero hash for a header it cannot read.
func TestBlockContext_AnswersBlockHashesByLookup(t *testing.T) {
	h := newStateEnvironment(t)
	header, err := h.Core.Header(1)
	require.NoError(t, err)
	genesis, err := h.Core.Header(0)
	require.NoError(t, err)
	getHash := h.Environment.BlockContext(header).GetHash
	headers := h.Core.Calls("Header")
	require.Equal(t, genesis.Hash, getHash(0))
	require.Equal(t, common.Hash{}, getHash(1))
	require.Equal(t, common.Hash{}, getHash(1<<63))
	require.Equal(t, headers+1, h.Core.Calls("Header"))
	h.Core.SetMode("Header", tests.Fail)
	require.Equal(t, common.Hash{}, getHash(0))
}

func TestRevertError_CarriesTheReasonAndTheData(t *testing.T) {
	reason := hexutil.MustDecode("0x08c379a0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"6e6f706500000000000000000000000000000000000000000000000000000000")
	cases := map[string]struct {
		revert  []byte
		message string
	}{
		"with a reason":    {reason, "execution reverted: nope"},
		"without a reason": {[]byte{0xde, 0xad}, "execution reverted"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			err := exec.NewRevertError(&core.ExecutionResult{Err: vm.ErrExecutionReverted, ReturnData: test.revert})
			require.EqualError(t, err, test.message)
			require.Equal(t, 3, err.ErrorCode())
			require.Equal(t, hexutil.Encode(test.revert), err.ErrorData())
		})
	}
}

func TestEnvironment_OffersItsConfigurationAndOracle(t *testing.T) {
	h := newStateEnvironment(t)
	require.Equal(t, exec.DefaultConfig(), h.Environment.Config())
	require.NotNil(t, h.Environment.Oracle())
	require.Equal(t, big.NewInt(1.1e9), h.Environment.SuggestedGasPrice(h.Environment.Head()))
}
