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

package eth_test

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// word42 is the answer of ReturningCode.
var word42 = hexutil.Bytes(common.BigToHash(common.Big2.SetInt64(42)).Bytes())

// blockHashCode returns BLOCKHASH(n).
func blockHashCode(n uint64) hexutil.Bytes {
	code := []byte{byte(vm.PUSH8)}
	code = binary.BigEndian.AppendUint64(code, n)
	return append(code, byte(vm.BLOCKHASH), byte(vm.PUSH1), 0, byte(vm.MSTORE), byte(vm.PUSH1), 32, byte(vm.PUSH1), 0, byte(vm.RETURN))
}

// extCodeSizeCode touches another account and stops.
func extCodeSizeCode(of common.Address) hexutil.Bytes {
	code := append([]byte{byte(vm.PUSH20)}, of.Bytes()...)
	return append(code, byte(vm.EXTCODESIZE), byte(vm.POP), byte(vm.STOP))
}

var probe = common.HexToAddress("0x9999")

func withCode(code []byte) map[string]any {
	return map[string]any{probe.Hex(): map[string]any{"code": hexutil.Bytes(code)}}
}

func TestCall_ReturnsTheOutput(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	var out hexutil.Bytes
	h.MustCall(&out, "eth_call", map[string]any{"to": r.Returner}, "latest")
	require.Equal(t, word42, out)
	require.Equal(t, `"0x"`, string(h.MustCall(nil, "eth_call", map[string]any{"from": r.Wallet.Address, "to": r.Writer}, "0x1")))
}

func TestCall_ARevertIsCode3WithItsData(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	e := h.Fails("eth_call", map[string]any{"to": probe}, "latest", withCode(tests.RevertingCode))
	require.Equal(t, 3, e.Code)
	require.Equal(t, "execution reverted: boom", e.Message)
	require.Contains(t, e.Data, "0x08c379a0")

	e = h.Fails("eth_estimateGas", map[string]any{"to": probe}, "latest", withCode(tests.RevertingCode))
	require.Equal(t, 3, e.Code)
	require.Equal(t, "execution reverted: boom", e.Message)
}

// TestCall_TheBlockIsCheckedFirst is F2 and F3: a request naming a missing
// block and carrying a bad fee pair reports the block, in each method's
// wording.
func TestCall_TheBlockIsCheckedFirst(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	bad := map[string]any{"to": r.Returner, "gasPrice": "0x1", "maxFeePerGas": "0x1"}
	require.Equal(t, "header not found", h.Fails("eth_call", bad, "0x9").Message)
	require.Equal(t, "header not found", h.Fails("eth_createAccessList", bad, "0x9").Message)
	require.Equal(t, "block does not exists", h.Fails("eth_estimateGas", bad, "0x9").Message)
	require.Equal(t, "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found",
		h.Fails("eth_estimateGas", bad, common.Hash{1}).Message)
	require.Equal(t, "invalid block number or hash", h.Fails("eth_estimateGas", bad, map[string]any{}).Message)

	msg := "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"
	require.Equal(t, msg, h.Fails("eth_call", bad, "latest").Message)
	require.Equal(t, msg, h.Fails("eth_estimateGas", bad, "latest").Message)
	require.Equal(t, -32000, h.Fails("eth_call", bad, "latest").Code, "a handler's own check is a plain error (F4)")
}

// TestCall_AFailedMessageNamesItsGas is F5: eth_call and eth_estimateGas wrap
// the error with the supplied gas; eth_createAccessList reports it bare.
func TestCall_AFailedMessageNamesItsGas(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	poor := common.HexToAddress("0x7777")
	call := map[string]any{"from": poor, "to": r.Recipient, "value": "0x1", "gas": "0x5208", "gasPrice": "0x3b9aca00"}
	e := h.Fails("eth_call", call, "latest")
	require.Equal(t, "err: insufficient funds for gas * price + value: address 0x0000000000000000000000000000000000007777 have 0 want 21000000000000 (supplied gas 21000)", e.Message)

	e = h.Fails("eth_createAccessList", call, "latest")
	require.Equal(t, fmt.Sprintf("failed to apply transaction from sender %v and nonce 0: insufficient funds for gas * price + value: address %v have 0 want 21000000000000", poor, poor), e.Message)
}

func TestCall_OverridesOfStateAndBlock(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	var out hexutil.Bytes
	h.MustCall(&out, "eth_call", map[string]any{"to": probe}, "latest", withCode(tests.ReturningCode))
	require.Equal(t, word42, out)

	both := map[string]any{probe.Hex(): map[string]any{"state": map[string]any{}, "stateDiff": map[string]any{}}}
	require.Equal(t, "account 0x0000000000000000000000000000000000009999 has both 'state' and 'stateDiff'",
		h.Fails("eth_call", map[string]any{"to": probe}, "latest", both).Message)

	block0, err := r.Core.Header(0)
	require.NoError(t, err)
	h.MustCall(&out, "eth_call", map[string]any{"to": probe}, "latest", withCode(blockHashCode(0)), map[string]any{"number": "0x5"})
	require.Equal(t, hexutil.Bytes(block0.Hash.Bytes()), out, "a block of the chain below the overridden number")
	h.MustCall(&out, "eth_call", map[string]any{"to": probe}, "latest", withCode(blockHashCode(1)), map[string]any{"number": "0x5"})
	require.Equal(t, make(hexutil.Bytes, 32), out, "no block at or above the block of the call")
}

// TestCall_BlockHashIsOneLookupWhateverTheOverride is V11: BLOCKHASH reads
// at most one header, whatever block number an override sets.
func TestCall_BlockHashIsOneLookupWhateverTheOverride(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	block0, err := r.Core.Header(0)
	require.NoError(t, err)
	for _, number := range []uint64{1, 2, 1_000_000, 1 << 62} {
		for _, asked := range []uint64{0, number - 1} {
			before := r.Core.Calls("Header")
			var out hexutil.Bytes
			h.MustCall(&out, "eth_call", map[string]any{"to": probe}, "latest", withCode(blockHashCode(asked)),
				map[string]any{"number": hexutil.EncodeUint64(number)})
			lookups := r.Core.Calls("Header") - before
			require.LessOrEqual(t, lookups, 2, "number %d, BLOCKHASH(%d)", number, asked)
			if asked == 0 && number <= 256 {
				require.Equal(t, hexutil.Bytes(block0.Hash.Bytes()), out, "number %d", number)
			}
		}
		before := r.Core.Calls("Header")
		h.MustCall(nil, "eth_estimateGas", map[string]any{"to": probe}, "latest", withCode(blockHashCode(number-1)),
			map[string]any{"number": hexutil.EncodeUint64(number)})
		steps := r.Core.Calls("Header") - before
		require.LessOrEqual(t, steps, 64, "eth_estimateGas reads one header per step at most")
	}
}

func TestEstimateGas_ATransferAndItsLimits(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	var gas hexutil.Uint64
	h.MustCall(&gas, "eth_estimateGas", map[string]any{"from": r.Wallet.Address, "to": r.Recipient, "value": "0x1"})
	require.Equal(t, hexutil.Uint64(21000), gas)

	e := h.Fails("eth_estimateGas", map[string]any{"from": probe, "to": r.Recipient, "value": "0x1", "gasPrice": "0x1"})
	require.Equal(t, "insufficient funds for transfer", e.Message)

	// The chain's interpreter reports running out of gas as an unsuccessful
	// execution, which the reference answered as it is.
	e = h.Fails("eth_estimateGas", map[string]any{"to": probe, "gas": "0x10000"}, "latest", withCode(tests.SpinningCode))
	require.Equal(t, "execution unsuccessful", e.Message)
}

// TestCall_TheGasCapBoundsASpinningCall is the timeout test of O2 and section
// 8: the interpreters of the chain do not poll for the deadline, so a call
// that never ends is bounded by the gas cap.
func TestCall_TheGasCapBoundsASpinningCall(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core, func(c *tests.Configs) { c.Exec.GasCap = 1_000_000 })
	start := time.Now()
	e := h.Fails("eth_call", map[string]any{"to": probe, "gas": "0xffffffff"}, "latest", withCode(tests.SpinningCode))
	require.Equal(t, "execution unsuccessful", e.Message)
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestCreateAccessList_ListsTheTouchedAccounts(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	r.Core.SetAccount(probe, tests.Account{Code: extCodeSizeCode(r.Writer)})
	var res struct {
		AccessList []struct {
			Address     common.Address `json:"address"`
			StorageKeys []common.Hash  `json:"storageKeys"`
		} `json:"accessList"`
		GasUsed hexutil.Uint64 `json:"gasUsed"`
		Error   string         `json:"error"`
	}
	h.MustCall(&res, "eth_createAccessList", map[string]any{"from": r.Wallet.Address, "to": probe})
	require.Len(t, res.AccessList, 1)
	require.Equal(t, r.Writer, res.AccessList[0].Address)
	require.Positive(t, res.GasUsed)
	require.Empty(t, res.Error)

	// Without gas, the gas is estimated first, which fails on a revert; with
	// gas, the revert is the transaction's own failure.
	r.Core.SetAccount(probe, tests.Account{Code: tests.RevertingCode})
	require.Equal(t, "execution reverted: boom", h.Fails("eth_createAccessList", map[string]any{"from": r.Wallet.Address, "to": probe}).Message)
	h.MustCall(&res, "eth_createAccessList", map[string]any{"from": r.Wallet.Address, "to": probe, "gas": "0x10000"}, "latest")
	require.Equal(t, "execution reverted", res.Error)
}

func TestSimulate_RunsCallsInSimulatedBlocks(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	var blocks []struct {
		Number hexutil.Uint64 `json:"number"`
		Calls  []struct {
			ReturnData hexutil.Bytes  `json:"returnData"`
			Status     hexutil.Uint64 `json:"status"`
			GasUsed    hexutil.Uint64 `json:"gasUsed"`
		} `json:"calls"`
	}
	h.MustCall(&blocks, "eth_simulateV1", map[string]any{
		"blockStateCalls": []any{
			map[string]any{"calls": []any{map[string]any{"from": r.Wallet.Address, "to": r.Returner}}},
			map[string]any{"calls": []any{map[string]any{"from": r.Wallet.Address, "to": r.Writer}}},
		},
	}, "latest")
	require.Len(t, blocks, 2)
	require.Equal(t, hexutil.Uint64(2), blocks[0].Number)
	require.Equal(t, word42, blocks[0].Calls[0].ReturnData)
	require.Equal(t, hexutil.Uint64(1), blocks[1].Calls[0].Status)

	calls := make([]any, 5001)
	for i := range calls {
		calls[i] = map[string]any{"to": r.Returner}
	}
	e := h.Fails("eth_simulateV1", map[string]any{"blockStateCalls": []any{map[string]any{"calls": calls}}}, "latest")
	require.Equal(t, -38026, e.Code)

	e = h.Fails("eth_simulateV1", map[string]any{"blockStateCalls": []any{
		map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x5"}},
		map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x4"}},
	}}, "latest")
	require.Equal(t, -38020, e.Code)
	require.Equal(t, "block numbers must be in order: 4 <= 5", e.Message)
}

// TestCall_ReadsHeadersNotBlocks is I13: the methods that run a call read the
// header of their block, never the block.
func TestCall_ReadsHeadersNotBlocks(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := newEth(t, r.Core)
	blocks := r.Core.Calls("Block")
	h.MustCall(nil, "eth_call", map[string]any{"to": r.Returner}, "latest")
	h.MustCall(nil, "eth_estimateGas", map[string]any{"from": r.Wallet.Address, "to": r.Recipient})
	h.MustCall(nil, "eth_createAccessList", map[string]any{"from": r.Wallet.Address, "to": r.Writer})
	h.MustCall(nil, "eth_getBalance", r.Writer, "0x1")
	h.MustCall(nil, "eth_getProof", r.Writer, []string{}, "0x1")
	require.Equal(t, blocks, r.Core.Calls("Block"))
}
