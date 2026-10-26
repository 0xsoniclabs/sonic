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

package sonic_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/sonic"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	sender  = common.HexToAddress("0xa0")
	counter = common.HexToAddress("0xc1")
	// reverter reverts without data.
	reverter = common.HexToAddress("0xc2")
)

func newSonic(t *testing.T) *tests.Harness {
	t.Helper()
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		sender:   {Balance: new(big.Int).Lsh(big.NewInt(1), 100)},
		counter:  {Code: []byte{0x60, 0x00, 0x54, 0x60, 0x01, 0x01, 0x80, 0x60, 0x00, 0x55, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}},
		reverter: {Code: []byte{0x60, 0x00, 0x80, 0xfd}},
	})
	carmenCore.AppendAndPublish(tests.BlockSpec{})
	harness := tests.NewHarness(t, carmenCore.FakeCore, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		sonic.Register(dispatcher, environment)
	})
	harness.WaitForHead(1)
	return harness
}

func call(recipient common.Address) map[string]any {
	return map[string]any{"from": sender, "to": recipient}
}

// TestEstimateGasForTransactions_EstimatesEachAfterTheOnesBefore holds I8:
// the second increment of the counter is estimated on the state the first
// left, so it is cheaper.
func TestEstimateGasForTransactions_EstimatesEachAfterTheOnesBefore(t *testing.T) {
	harness := newSonic(t)
	alone := tests.Decode[map[string][]string](t, harness.Result("sonic_estimateGasForTransactions", []any{call(counter)}))
	both := tests.Decode[map[string][]string](t, harness.Result("sonic_estimateGasForTransactions", []any{call(counter), call(counter)}, "latest"))
	require.Equal(t, alone["gasLimits"][0], both["gasLimits"][0])
	first, second := parse(t, both["gasLimits"][0]), parse(t, both["gasLimits"][1])
	require.Less(t, second, first, "a slot the first transaction set is cheaper to change")

	transfer := tests.Decode[map[string][]string](t, harness.Result("sonic_estimateGasForTransactions", []any{call(sender)}))
	require.Equal(t, []string{"0x62d4"}, transfer["gasLimits"], "21000 and the gas of an address and a key in the access list")
}

func TestEstimateGasForTransactions_RefusesTooManyAndReportsTheFailingOne(t *testing.T) {
	harness := newSonic(t)
	tooMany := make([]any, 17)
	for index := range tooMany {
		tooMany[index] = call(sender)
	}
	require.Equal(t, "too many transactions to estimate gas for: got 17, max is 16", harness.Failure("sonic_estimateGasForTransactions", tooMany).Message)

	failure := harness.Failure("sonic_estimateGasForTransactions", []any{call(counter), call(reverter)})
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "failed to estimate gas for transaction 1: execution reverted"}, failure,
		"a wrapped revert loses its code and data, as in the reference")
	require.Equal(t, "failed to estimate gas for transaction 0: block does not exists", harness.Failure("sonic_estimateGasForTransactions", []any{call(sender)}, "0x7").Message)
	require.JSONEq(t, `{"gasLimits":[]}`, string(harness.Result("sonic_estimateGasForTransactions", []any{})))
}

func parse(t *testing.T, quantity string) uint64 {
	t.Helper()
	value, ok := new(big.Int).SetString(quantity[2:], 16)
	require.True(t, ok)
	return value.Uint64()
}
