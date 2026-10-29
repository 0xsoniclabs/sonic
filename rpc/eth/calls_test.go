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
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	reverter      = common.HexToAddress("0xc2")
	looper        = common.HexToAddress("0xc3")
	blockHasher   = common.HexToAddress("0xc4")
	balanceReader = common.HexToAddress("0xc5")
	needsData     = common.HexToAddress("0xc6")
	readAccount   = common.HexToAddress("0xbb")
	callAccounts  = map[common.Address]tests.Account{
		holder:        {Balance: new(big.Int).Lsh(big.NewInt(1), 100)},
		reverter:      {Code: []byte{0x63, 0xde, 0xad, 0xbe, 0xef, 0x60, 0x00, 0x52, 0x60, 0x04, 0x60, 0x1c, 0xfd}},
		looper:        {Code: []byte{0x5b, 0x60, 0x00, 0x56}},
		blockHasher:   {Code: []byte{0x60, 0x00, 0x35, 0x40, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}},
		balanceReader: {Code: append(append([]byte{0x73}, readAccount.Bytes()...), 0x31, 0x50, 0x00)},
		needsData:     {Code: []byte{0x36, 0x60, 0x08, 0x57, 0x60, 0x00, 0x80, 0xfd, 0x5b, 0x00}},
	}
)

func callTo(recipient common.Address, fields ...any) map[string]any {
	call := map[string]any{"from": holder, "to": recipient}
	for index := 0; index+1 < len(fields); index += 2 {
		call[fields[index].(string)] = fields[index+1]
	}
	return call
}

func TestCall_RunsTheMessageOnTheStateOfTheBlock(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000000000000000000000000008"`, string(chain.harness.Result("eth_call", callTo(contract))))
	require.JSONEq(t, `"0x0000000000000000000000000000000000000000000000000000000000000008"`, string(chain.harness.Result("eth_call", callTo(contract), "0x1")))

	overrides := map[string]any{contract.Hex(): map[string]any{"stateDiff": map[string]any{common.Hash{}.Hex(): common.BigToHash(big.NewInt(41)).Hex()}}}
	require.JSONEq(t, `"0x000000000000000000000000000000000000000000000000000000000000002a"`, string(chain.harness.Result("eth_call", callTo(contract), "latest", overrides)))

	both := map[string]any{contract.Hex(): map[string]any{"state": map[string]any{}, "stateDiff": map[string]any{}}}
	require.Contains(t, chain.harness.Failure("eth_call", callTo(contract), "latest", both).Message, "has both 'state' and 'stateDiff'")
}

func TestCall_AnswersARevertWithItsData(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	failure := chain.harness.Failure("eth_call", callTo(reverter))
	require.Equal(t, &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"}, failure)
}

// TestCall_ChecksInTheLegacyOrder holds F2, F3, F4 and F5.
func TestCall_ChecksInTheLegacyOrder(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	badFees := callTo(contract, "gasPrice", "0x1", "maxFeePerGas", "0x1")
	require.Equal(t, "header not found", chain.harness.Failure("eth_call", badFees, "0x9").Message, "the block before the fees")
	both := map[string]any{contract.Hex(): map[string]any{"state": map[string]any{}, "stateDiff": map[string]any{}}}
	require.Contains(t, chain.harness.Failure("eth_call", badFees, "latest", both).Message, "has both", "the overrides before the fees")
	failure := chain.harness.Failure("eth_call", badFees)
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"}, failure)
	require.Equal(t, "err: intrinsic gas too low: have 1000, want 21000 (supplied gas 1000)", chain.harness.Failure("eth_call", callTo(contract, "gas", "0x3e8")).Message)
	require.Equal(t, "unknown header selector", chain.harness.Failure("eth_call", callTo(contract), map[string]any{}).Message)
}

// TestCall_AnswersBlockHashByOneLookup holds V11: an override that moves the
// block far above the head changes what BLOCKHASH may ask for, not what it
// costs.
func TestCall_AnswersBlockHashByOneLookup(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	far := new(big.Int).Lsh(big.NewInt(1), 40)
	overrides := map[string]any{"number": "0x" + far.Text(16)}
	below := common.BigToHash(new(big.Int).Sub(far, big.NewInt(1))).Hex()

	before := chain.core.Calls("Header")
	answer := chain.harness.Result("eth_call", callTo(blockHasher, "input", below), "0x1", nil, overrides)
	require.JSONEq(t, `"`+common.Hash{}.Hex()+`"`, string(answer))
	require.Equal(t, before+1, chain.core.Calls("Header"), "the header of the block only")

	answer = chain.harness.Result("eth_call", callTo(blockHasher, "input", common.BigToHash(big.NewInt(1)).Hex()), "0x2")
	require.JSONEq(t, `"`+chain.blocks[0].Hex()+`"`, string(answer), "a block below is looked up")
}

// TestCall_IsBoundedByTheGasCap holds T6 for eth_call: Tosca's interpreter
// does not poll the EVM timeout, so a loop runs until RPCGasCap is spent.
func TestCall_IsBoundedByTheGasCap(t *testing.T) {
	chain := newStateChain(t, callAccounts, func(configs *tests.Configs) {
		configs.Exec.RPCGasCap = 1_000_000
	})
	require.Equal(t, "execution unsuccessful", chain.harness.Failure("eth_call", callTo(looper)).Message)
}

// TestCreateAccessList_IsBoundedByTheRequestDeadline holds T6 for the
// methods that run go-ethereum's interpreter, which a tracer selects: the
// request deadline (RPCTimeout) cancels it, and the method answers the error
// of its context, as the reference did.
func TestCreateAccessList_IsBoundedByTheRequestDeadline(t *testing.T) {
	chain := newStateChain(t, callAccounts, func(configs *tests.Configs) {
		configs.Dispatch.RequestTimeout = 50 * time.Millisecond
		configs.Exec.RPCGasCap = 1 << 40
	})
	start := time.Now()
	failure := chain.harness.Failure("eth_createAccessList", callTo(looper, "gas", "0x10000000000"))
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "context deadline exceeded"}, failure)
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestEstimateGas_AnswersTheLeastGas(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	require.JSONEq(t, `"0x5208"`, string(chain.harness.Result("eth_estimateGas", callTo(readAccount))))
	require.JSONEq(t, `"0x5208"`, string(chain.harness.Result("eth_estimateGas", callTo(readAccount), "0x1")))
	require.Equal(t, &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"}, chain.harness.Failure("eth_estimateGas", callTo(reverter)))
	require.Equal(t, "block does not exists", chain.harness.Failure("eth_estimateGas", callTo(readAccount), "0x9").Message)
	unknown := common.Hash{0xee}
	require.Equal(t, "block with hash "+unknown.Hex()+" not found", chain.harness.Failure("eth_estimateGas", callTo(readAccount), unknown).Message)
	require.Equal(t, "invalid block number or hash", chain.harness.Failure("eth_estimateGas", callTo(readAccount), map[string]any{}).Message)

	code := map[string]any{readAccount.Hex(): map[string]any{"code": "0x63deadbeef6000526004601cfd"}}
	require.Equal(t, "0xdeadbeef", chain.harness.Failure("eth_estimateGas", callTo(readAccount), "latest", code).Data, "the state overrides apply")
}

func TestCreateAccessList_CollectsWhatTheTransactionTouches(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	answer := decodeMap(t, chain.harness.Result("eth_createAccessList", callTo(balanceReader)))
	require.Equal(t, []any{map[string]any{"address": strings.ToLower(readAccount.Hex()), "storageKeys": []any{}}}, lowerAddresses(answer["accessList"]))
	require.NotEqual(t, "0x0", answer["gasUsed"])
	require.NotContains(t, answer, "error")

	answer = decodeMap(t, chain.harness.Result("eth_createAccessList", callTo(reverter, "gas", "0x186a0")))
	require.Equal(t, "execution reverted", answer["error"], "a failed transaction is reported in the answer")
	require.Equal(t, "header not found", chain.harness.Failure("eth_createAccessList", callTo(balanceReader), "0x9").Message)
}

// TestCreateAccessList_ReportsAFailedApplicationWithSenderAndNonce holds F5.
func TestCreateAccessList_ReportsAFailedApplicationWithSenderAndNonce(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	chain.core.SetPoolNonce(holder, 3)
	failure := chain.harness.Failure("eth_createAccessList", callTo(balanceReader, "gas", "0x3e8"))
	require.Equal(t, "failed to apply transaction from sender "+holder.Hex()+" and nonce 3: intrinsic gas too low: have 1000, want 21000", failure.Message)
}

// TestCreateAccessList_EstimatesWithoutInput reproduces the reference's bug 2
// of section 6 (F8): without gas, a transaction that sends only input is
// estimated without call data.
func TestCreateAccessList_EstimatesWithoutInput(t *testing.T) {
	chain := newStateChain(t, callAccounts)
	chain.harness.Result("eth_createAccessList", callTo(needsData, "data", "0x01"))
	require.Equal(t, "execution reverted", chain.harness.Failure("eth_createAccessList", callTo(needsData, "input", "0x01")).Message)
	chain.harness.Result("eth_createAccessList", callTo(needsData, "input", "0x01", "gas", "0x186a0"))
}

func lowerAddresses(list any) []any {
	entries := list.([]any)
	for _, entry := range entries {
		fields := entry.(map[string]any)
		fields["address"] = strings.ToLower(fields["address"].(string))
	}
	return entries
}
