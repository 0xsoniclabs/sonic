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

package testonly_test

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/testonly"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func newHarness(t *testing.T, enabled bool) *tests.Harness {
	t.Helper()
	return tests.NewHarness(t, tests.NewFakeCore(), func(h *tests.Harness) {
		testonly.Register(h.Dispatcher, h.Environment, h.Core)
	}, func(settings *tests.Settings) {
		settings.Exec.EnableTestOnlyApi = enabled
	})
}

// encoded is a batch of two transactions, RLP-encoded, which JSON encodes in
// base64.
func encoded(t *testing.T) [][]byte {
	t.Helper()
	account := tests.NewAccount()
	recipient := common.Address{0x42}
	batch := [][]byte{}
	for nonce := range uint64(2) {
		binary, err := rlp.EncodeToBytes(account.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), Nonce: nonce,
			To: &recipient, Gas: 21000, GasFeeCap: big.NewInt(1e9)}))
		require.NoError(t, err)
		batch = append(batch, binary)
	}
	return batch
}

var methods = map[string]string{
	"test_addTransactions":     "SendTransactions",
	"test_proposeTransactions": "EmitTransactions",
}

func TestTestOnly_HandsTheTransactionsToTheNodeWhenEnabled(t *testing.T) {
	h := newHarness(t, true)
	for method, call := range methods {
		t.Run(method, func(t *testing.T) {
			require.Equal(t, "null", h.Result(t, method, encoded(t)))
			require.Equal(t, 1, h.Core.Calls(call))

			require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "typed transaction too short"},
				h.Fail(t, method, [][]byte{{0x01}}))
			h.Core.SetMode(call, tests.Fail)
			require.Equal(t, tests.ErrInjected.Error(), h.Fail(t, method, encoded(t)).Message)
			require.Equal(t, 2, h.Core.Calls(call))
		})
	}
}

func TestTestOnly_AsksNothingOfTheNodeWhenDisabled(t *testing.T) {
	h := newHarness(t, false)
	for method, call := range methods {
		t.Run(method, func(t *testing.T) {
			require.Equal(t, &dispatch.Error{Code: dispatch.CodeDefault, Message: "test-only API is not enabled"}, h.Fail(t, method, encoded(t)))
			require.Zero(t, h.Core.Calls(call))
		})
	}
}
