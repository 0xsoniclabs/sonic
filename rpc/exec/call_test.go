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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// counter increments slot 0 and returns its new value.
var counter = []byte{0x60, 0x00, 0x54, 0x60, 0x01, 0x01, 0x80, 0x60, 0x00, 0x55, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}

// looper loops until it runs out of gas.
var looper = []byte{0x5b, 0x60, 0x00, 0x56}

// reverter reverts with the data 0xdeadbeef.
var reverter = []byte{0x63, 0xde, 0xad, 0xbe, 0xef, 0x60, 0x00, 0x52, 0x60, 0x04, 0x60, 0x1c, 0xfd}

var (
	counterAddress  = common.HexToAddress("0xc1")
	reverterAddress = common.HexToAddress("0xc2")
	looperAddress   = common.HexToAddress("0xc3")
)

// newCallChain returns an environment over twenty archived blocks whose
// state holds the counter and the reverter.
func newCallChain(t *testing.T) (*exec.Environment, *tests.CarmenCore, exec.Head) {
	t.Helper()
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		counterAddress:  {Code: counter},
		reverterAddress: {Code: reverter},
		looperAddress:   {Code: looper},
	})
	for range 19 {
		carmenCore.AppendBlock(tests.BlockSpec{})
	}
	carmenCore.PublishHead(19)
	harness := tests.NewHarness(t, carmenCore, func(*dispatch.Dispatcher, *exec.Environment) {})
	harness.WaitForHead(19)
	head, err := harness.Environment.Latest(context.Background())
	require.NoError(t, err)
	return harness.Environment, carmenCore, head
}

func prepare(t *testing.T, environment *exec.Environment, head exec.Head, number uint64, overrides *exec.BlockOverrides) *exec.Call {
	t.Helper()
	header, err := environment.HeaderAt(head, number)
	require.NoError(t, err)
	call, err := environment.PrepareCall(head, header, overrides)
	require.NoError(t, err)
	return call
}

func message(t *testing.T, recipient common.Address, gas uint64) *core.Message {
	t.Helper()
	arguments := exec.CallArguments{Recipient: &recipient, Gas: (*hexutil.Uint64)(&gas)}
	converted, err := arguments.ToMessage(50_000_000, big.NewInt(1))
	require.NoError(t, err)
	return converted
}

// TestBlockContext_AnswersBlockHashByOneLookupBelowTheBlock holds V11: a
// number below the block is one lookup by number; at and above it, whatever
// number an override gives the block, the answer is zero without a lookup.
func TestBlockContext_AnswersBlockHashByOneLookupBelowTheBlock(t *testing.T) {
	environment, carmenCore, head := newCallChain(t)
	header, err := environment.HeaderAt(head, 10)
	require.NoError(t, err)
	five, err := environment.HeaderAt(head, 5)
	require.NoError(t, err)

	context := environment.BlockContext(head, header)
	(&exec.BlockOverrides{Number: (*hexutil.Big)(big.NewInt(1 << 40))}).Apply(&context)
	before := carmenCore.Calls("Header")
	require.Equal(t, five.Hash, context.GetHash(5))
	require.Equal(t, before+1, carmenCore.Calls("Header"), "one lookup by number")
	for _, number := range []uint64{10, 11, 19, 1<<40 - 1} {
		require.Equal(t, common.Hash{}, context.GetHash(number), "block %d", number)
	}
	require.Equal(t, before+1, carmenCore.Calls("Header"), "no lookup at or above the block of the call")
}

func TestPrepareCall_AppliesTheBlockOverrides(t *testing.T) {
	environment, _, head := newCallChain(t)
	coinbase := common.HexToAddress("0xc0ffee")
	call := prepare(t, environment, head, 10, &exec.BlockOverrides{Number: (*hexutil.Big)(big.NewInt(12345)), Coinbase: &coinbase})
	require.Equal(t, uint64(10), call.Header.Number.Uint64(), "the header stays the block's")

	carmenCore := tests.NewFakeCore(t)
	carmenCore.SetMode("Rules", tests.Fail)
	failing := tests.NewHarness(t, carmenCore, func(*dispatch.Dispatcher, *exec.Environment) {})
	failingHead, err := failing.Environment.Latest(context.Background())
	require.NoError(t, err)
	header, err := failing.Environment.HeaderAt(failingHead, 0)
	require.NoError(t, err)
	_, err = failing.Environment.PrepareCall(failingHead, header, nil)
	require.ErrorIs(t, err, tests.ErrInjected, "the rules of the block are read")
}

// TestCallRun_AppliesThePreviousMessagesFirst holds I8: previous messages
// run on the same state, each in its transaction, and one that cannot be
// applied is skipped.
func TestCallRun_AppliesThePreviousMessagesFirst(t *testing.T) {
	environment, _, head := newCallChain(t)
	call := prepare(t, environment, head, 19, nil)
	statedb, err := environment.OpenState(head, 19)
	require.NoError(t, err)
	defer statedb.Release()

	previous := []*core.Message{message(t, counterAddress, 100_000), message(t, counterAddress, 1_000), message(t, counterAddress, 100_000)}
	result, err := call.Run(context.Background(), statedb, previous, message(t, counterAddress, 100_000), 0)
	require.NoError(t, err)
	require.Equal(t, common.BigToHash(big.NewInt(3)).Bytes(), result.Return(), "two previous increments and this one")
}

// TestCallRun_WrapsAMessageThatCannotBeApplied holds F5.
func TestCallRun_WrapsAMessageThatCannotBeApplied(t *testing.T) {
	environment, _, head := newCallChain(t)
	call := prepare(t, environment, head, 19, nil)
	statedb, err := environment.OpenState(head, 19)
	require.NoError(t, err)
	defer statedb.Release()
	_, err = call.Run(context.Background(), statedb, nil, message(t, counterAddress, 1_000), 0)
	require.ErrorIs(t, err, core.ErrIntrinsicGas)
	require.Equal(t, "err: intrinsic gas too low: have 1000, want 21000 (supplied gas 1000)", err.Error())
}

func TestCallRun_ReportsACancelledRunAsAborted(t *testing.T) {
	environment, _, head := newCallChain(t)
	call := prepare(t, environment, head, 19, nil)
	statedb, err := environment.OpenState(head, 19)
	require.NoError(t, err)
	defer statedb.Release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// The cancel reaches the EVM from a goroutine of its own; a run long
	// enough for it to arrive is reported as aborted.
	_, err = call.Run(ctx, statedb, nil, message(t, looperAddress, 50_000_000), 0)
	require.EqualError(t, err, "execution aborted (timeout = 0s)")
}

func TestCallTrace_ReportsTheBareError(t *testing.T) {
	environment, _, head := newCallChain(t)
	call := prepare(t, environment, head, 19, nil)
	statedb, err := environment.OpenState(head, 19)
	require.NoError(t, err)
	defer statedb.Release()
	_, err = call.Trace(context.Background(), statedb, message(t, counterAddress, 1_000), nil)
	require.EqualError(t, err, "intrinsic gas too low: have 1000, want 21000")
}

func TestRevertError_CarriesCodeThreeAndTheData(t *testing.T) {
	environment, _, head := newCallChain(t)
	call := prepare(t, environment, head, 19, nil)
	statedb, err := environment.OpenState(head, 19)
	require.NoError(t, err)
	defer statedb.Release()
	result, err := call.Run(context.Background(), statedb, nil, message(t, reverterAddress, 100_000), 0)
	require.NoError(t, err)
	require.ErrorIs(t, result.Err, vm.ErrExecutionReverted)

	converted := dispatch.ToError(exec.RevertError(result))
	require.Equal(t, &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"}, converted)

	reason := &core.ExecutionResult{Err: vm.ErrExecutionReverted, ReturnData: common.FromHex(
		"0x08c379a0" + "0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000004" +
			"626f6f6d00000000000000000000000000000000000000000000000000000000")}
	require.Equal(t, "execution reverted: boom", exec.RevertError(reason).Error())
}

func TestStateOverride_WritesTheAccountsAndEndsATransaction(t *testing.T) {
	environment, _, head := newCallChain(t)
	statedb, err := environment.OpenState(head, 19)
	require.NoError(t, err)
	defer statedb.Release()

	address := common.HexToAddress("0xa1")
	nonce, code := hexutil.Uint64(7), hexutil.Bytes{0x00}
	balance := (*hexutil.U256)(uint256.NewInt(9))
	storage := map[common.Hash]common.Hash{{1}: {2}}
	diff := map[common.Hash]common.Hash{{3}: {4}}
	overrides := exec.StateOverride{
		address:        {Nonce: &nonce, Code: &code, Balance: &balance, State: &storage},
		counterAddress: {StateDiff: &diff},
	}
	require.NoError(t, overrides.Apply(statedb))
	require.Equal(t, uint64(7), statedb.GetNonce(address))
	require.Equal(t, []byte{0x00}, statedb.GetCode(address))
	require.Equal(t, uint64(9), statedb.GetBalance(address).Uint64())
	require.Equal(t, common.Hash{2}, statedb.GetState(address, common.Hash{1}))
	require.Equal(t, common.Hash{4}, statedb.GetState(counterAddress, common.Hash{3}))
	require.Equal(t, counter, statedb.GetCode(counterAddress), "a diff keeps the rest of the account")

	both := exec.StateOverride{address: {State: &storage, StateDiff: &diff}}
	require.EqualError(t, both.Apply(statedb), "account 0x00000000000000000000000000000000000000A1 has both 'state' and 'stateDiff'")
	require.NoError(t, exec.StateOverride(nil).Apply(statedb))
}

func TestBlockOverrides_ChangeTheContext(t *testing.T) {
	environment, _, head := newCallChain(t)
	header, err := environment.HeaderAt(head, 3)
	require.NoError(t, err)
	context := environment.BlockContext(head, header)
	random, coinbase := common.Hash{9}, common.HexToAddress("0xcb")
	time, gasLimit := hexutil.Uint64(77), hexutil.Uint64(88)
	(&exec.BlockOverrides{
		Number: (*hexutil.Big)(big.NewInt(1)), Difficulty: (*hexutil.Big)(big.NewInt(2)), Time: &time, GasLimit: &gasLimit,
		Coinbase: &coinbase, Random: &random, BaseFee: (*hexutil.Big)(big.NewInt(3)), BlobBaseFee: (*hexutil.Big)(big.NewInt(4)),
	}).Apply(&context)
	require.Equal(t, big.NewInt(1), context.BlockNumber)
	require.Equal(t, big.NewInt(2), context.Difficulty)
	require.Equal(t, uint64(77), context.Time)
	require.Equal(t, uint64(88), context.GasLimit)
	require.Equal(t, coinbase, context.Coinbase)
	require.Equal(t, &random, context.Random)
	require.Equal(t, big.NewInt(3), context.BaseFee)
	require.Equal(t, big.NewInt(4), context.BlobBaseFee)

	unchanged := environment.BlockContext(head, header)
	(*exec.BlockOverrides)(nil).Apply(&unchanged)
	require.Equal(t, header.Number, unchanged.BlockNumber)
}
