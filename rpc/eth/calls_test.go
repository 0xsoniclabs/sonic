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

package eth

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

var (
	contract = common.HexToAddress("0x2000000000000000000000000000000000000002")

	// PUSH1 0x2a PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN: returns 42 as a word.
	returns42 = []byte{0x60, 0x2a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
	// PUSH1 0xff PUSH1 0 MSTORE8 PUSH1 1 PUSH1 0 REVERT: reverts with one byte.
	reverts = []byte{0x60, 0xff, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xfd}
	// PUSH1 1 PUSH1 0 SSTORE STOP: writes one storage slot.
	stores = []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00}
)

func TestCall_ReturnsTheResult(t *testing.T) {
	h, _, cs := newStateHarness(t, 2)
	cs.SetCode(contract, returns42)

	var out hexutil.Bytes
	h.Into(&out, "eth_call", map[string]any{"to": contract}, "latest")
	require.Equal(t, common.BigToHash(big.NewInt(42)).Bytes(), []byte(out))

	// Both the block and the overrides are optional.
	h.Into(&out, "eth_call", map[string]any{"to": contract})
	require.Equal(t, common.BigToHash(big.NewInt(42)).Bytes(), []byte(out))
	h.Into(&out, "eth_call", map[string]any{"to": contract, "input": "0x"}, nil, nil, nil)
	require.Len(t, out, 32)
}

func TestCall_StateOverrideReplacesCode(t *testing.T) {
	h, _, _ := newStateHarness(t, 1)
	var out hexutil.Bytes
	h.Into(&out, "eth_call", map[string]any{"to": contract}, "latest",
		map[string]any{contract.Hex(): map[string]any{"code": hexutil.Encode(returns42)}})
	require.Equal(t, common.BigToHash(big.NewInt(42)).Bytes(), []byte(out))

	err := h.Fails("eth_call", map[string]any{"to": contract}, "latest",
		map[string]any{contract.Hex(): map[string]any{"state": map[string]string{}, "stateDiff": map[string]string{}}})
	require.Equal(t, dispatch.CodeDefault, err.Code)
	require.Contains(t, err.Message, "has both 'state' and 'stateDiff'")
}

func TestCall_RevertCarriesCodeAndData(t *testing.T) {
	h, _, cs := newStateHarness(t, 1)
	cs.SetCode(contract, reverts)
	err := h.Fails("eth_call", map[string]any{"to": contract}, "latest")
	require.Equal(t, 3, err.Code)
	require.Equal(t, "execution reverted", err.Message)
}

func TestCall_BlockIsCheckedBeforeTheCallObject(t *testing.T) {
	h, _, _ := newStateHarness(t, 1)
	// Both gasPrice and maxFeePerGas: invalid, but the missing block is reported.
	args := map[string]any{"to": contract, "gasPrice": "0x1", "maxFeePerGas": "0x1"}
	err := h.Fails("eth_call", args, "0x99")
	require.Equal(t, "header not found", err.Message)

	// Without the missing block, the call object is rejected by the execution
	// with the wrapped message.
	err = h.Fails("eth_call", args, "latest")
	require.Contains(t, err.Message, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
}

func TestEstimateGas(t *testing.T) {
	h, _, cs := newStateHarness(t, 2)
	cs.SetCode(contract, stores)

	var gas hexutil.Uint64
	h.Into(&gas, "eth_estimateGas", map[string]any{"to": account})
	require.EqualValues(t, 21000, gas)

	h.Into(&gas, "eth_estimateGas", map[string]any{"to": contract}, "latest")
	require.Greater(t, uint64(gas), uint64(21000))
	first := gas

	// The result does not depend on the way the block is named.
	h.Into(&gas, "eth_estimateGas", map[string]any{"to": contract}, "0x1")
	require.Equal(t, first, gas)
}

func TestEstimateGas_MissingBlockWordings(t *testing.T) {
	h, _, _ := newStateHarness(t, 1)
	// A missing block is "block does not exists" for a number, and
	// "block with hash ... not found" for a hash.
	require.Equal(t, "block does not exists", h.Fails("eth_estimateGas", map[string]any{"to": contract}, "0x99").Message)
	hash := common.Hash{9}
	require.Equal(t, "block with hash "+hash.String()+" not found",
		h.Fails("eth_estimateGas", map[string]any{"to": contract}, hash).Message)
	// The fee pair is checked after the block.
	args := map[string]any{"to": contract, "gasPrice": "0x1", "maxFeePerGas": "0x1"}
	require.Equal(t, "block does not exists", h.Fails("eth_estimateGas", args, "0x99").Message)
	require.Equal(t, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified",
		h.Fails("eth_estimateGas", args, "latest").Message)
}

func TestEstimateGas_FailsWhenNothingHelps(t *testing.T) {
	h, _, cs := newStateHarness(t, 1)
	cs.SetCode(contract, reverts)
	err := h.Fails("eth_estimateGas", map[string]any{"to": contract})
	require.Equal(t, 3, err.Code)
}

func TestCreateAccessList(t *testing.T) {
	h, _, cs := newStateHarness(t, 1)
	cs.SetCode(contract, stores).SetBalance(account, big.NewInt(1e18))
	var res struct {
		AccessList []struct {
			Address     common.Address `json:"address"`
			StorageKeys []common.Hash  `json:"storageKeys"`
		} `json:"accessList"`
		GasUsed hexutil.Uint64 `json:"gasUsed"`
		Error   string         `json:"error"`
	}
	h.Into(&res, "eth_createAccessList", map[string]any{"from": account, "to": contract, "gas": "0x100000"})
	require.Empty(t, res.Error)
	require.Len(t, res.AccessList, 1)
	require.Equal(t, contract, res.AccessList[0].Address)
	require.Equal(t, []common.Hash{{}}, res.AccessList[0].StorageKeys)
	require.NotZero(t, res.GasUsed)
}

func TestCreateAccessList_FailedExecutionIsReportedInTheResult(t *testing.T) {
	h, _, cs := newStateHarness(t, 1)
	cs.SetCode(contract, reverts).SetBalance(account, big.NewInt(1e18))
	var res struct {
		Error string `json:"error"`
	}
	h.Into(&res, "eth_createAccessList", map[string]any{"from": account, "to": contract, "gas": "0x100000"})
	require.Equal(t, "execution reverted", res.Error)
}

func TestFeeHistory_And_Prices(t *testing.T) {
	h, _, _ := newStateHarness(t, 5)

	var price hexutil.Big
	h.Into(&price, "eth_gasPrice")
	require.Equal(t, int64(1100), price.ToInt().Int64()) // base fee 1000 + 10%

	// With all gas power unused the oracle asks for a small premium (2.5%) of
	// the minimal gas price of the rules.
	h.Into(&price, "eth_maxPriorityFeePerGas")
	require.Positive(t, price.ToInt().Sign())
	require.Less(t, price.ToInt().Int64(), int64(100_000_000))

	h.Into(&price, "eth_blobBaseFee")
	require.Equal(t, int64(1), price.ToInt().Int64())

	var history FeeHistoryResult
	h.Into(&history, "eth_feeHistory", "0x3", "latest", []float64{10, 90})
	require.Equal(t, int64(3), history.OldestBlock.ToInt().Int64())
	require.Len(t, history.GasUsedRatio, 3)
	require.Len(t, history.BaseFee, 4)
	require.Len(t, history.Reward, 3)
	require.Len(t, history.Reward[0], 2)

	// The count is clamped, not refused.
	h.Into(&history, "eth_feeHistory", "0x1000", "latest", []float64{})
	require.Len(t, history.GasUsedRatio, 6)
	require.Equal(t, int64(0), history.OldestBlock.ToInt().Int64())

	err := h.Fails("eth_feeHistory", "0x1", "0x10", []float64{})
	require.Equal(t, "block 16 not found; latest block is 5", err.Message)
	err = h.Fails("eth_feeHistory", "0x1", "latest", []float64{50, 10})
	require.Contains(t, err.Message, "invalid reward percentile")
	err = h.Fails("eth_feeHistory", "0x1", "latest", []float64{101})
	require.Contains(t, err.Message, "invalid reward percentile")

	var empty FeeHistoryResult
	h.Into(&empty, "eth_feeHistory", "0x0", "latest", nil)
	require.Nil(t, empty.GasUsedRatio)
}

func TestGetAccountAndProof(t *testing.T) {
	// The in-memory state cannot create witness proofs: the answer is an error,
	// not a crash. The values are checked by the integration tests, on a real
	// archive.
	h, _, cs := newStateHarness(t, 1)
	cs.SetBalance(account, big.NewInt(5))
	err := h.Fails("eth_getAccount", account, "latest")
	require.Contains(t, err.Message, "proofs are not available")
	err = h.Fails("eth_getProof", account, []string{"0x1"}, "latest")
	require.Contains(t, err.Message, "failed to generate proof")

	// The block selector is optional for both.
	_, e := h.Call("eth_getAccount", account)
	require.NotNil(t, e)
	_, e = h.Call("eth_getProof", account, []string{})
	require.NotNil(t, e)

	// Missing blocks and bad keys are refused before any proof is made.
	require.Equal(t, "header not found", h.Fails("eth_getAccount", account, "0x99").Message)
	require.Equal(t, "header not found", h.Fails("eth_getProof", account, []string{"0x1"}, "0x99").Message)
	e2 := h.Fails("eth_getProof", account, []string{"0xzz"}, "0x99")
	require.Equal(t, dispatch.CodeInvalidParams, e2.Code)
	e2 = h.Fails("eth_getProof", account, make([]string, 1025), "latest")
	require.Equal(t, dispatch.CodeInvalidParams, e2.Code)
	require.Contains(t, e2.Message, "too many storage keys requested (max 1024, got 1025)")
}

func TestSimulateV1(t *testing.T) {
	h, _, cs := newStateHarness(t, 2)
	cs.SetBalance(account, big.NewInt(1e18))
	var res []struct {
		Number string `json:"number"`
		Calls  []struct {
			Status  string `json:"status"`
			GasUsed string `json:"gasUsed"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"calls"`
	}
	h.Into(&res, "eth_simulateV1", map[string]any{
		"blockStateCalls": []any{map[string]any{
			"calls": []any{
				map[string]any{"from": account, "to": contract, "value": "0x1"},
				map[string]any{"from": account, "to": contract, "value": "0x2"},
			},
		}},
	})
	require.Len(t, res, 1)
	require.Equal(t, "0x3", res[0].Number)
	require.Len(t, res[0].Calls, 2)
	for _, c := range res[0].Calls {
		require.Equal(t, "0x1", c.Status)
		require.Equal(t, "0x5208", c.GasUsed)
	}

	// A gap in the block numbers is filled with empty blocks.
	h.Into(&res, "eth_simulateV1", map[string]any{
		"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x6"}}},
	}, "latest")
	require.Len(t, res, 4) // blocks 3, 4, 5, 6

	err := h.Fails("eth_simulateV1", map[string]any{"blockStateCalls": []any{}})
	require.Equal(t, dispatch.CodeInvalidParams, err.Code)
	require.Equal(t, "empty input", err.Message)
	err = h.Fails("eth_simulateV1", map[string]any{"blockStateCalls": make([]map[string]any, 300)})
	require.Equal(t, -38026, err.Code)
	err = h.Fails("eth_simulateV1", map[string]any{"blockStateCalls": []any{map[string]any{}}}, "0x99")
	require.Equal(t, "header not found", err.Message)
	err = h.Fails("eth_simulateV1", map[string]any{"blockStateCalls": []any{
		map[string]any{"blockOverrides": map[string]any{"blockNumber": "0x1"}}}})
	require.Equal(t, -38020, err.Code)
}
