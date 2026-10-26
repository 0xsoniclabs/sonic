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

package txpool_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/txpool"
)

func newPool(t *testing.T) (*tests.FakeCore, *tests.Harness, *types.Transaction, *types.Transaction, common.Address) {
	t.Helper()
	fake := tests.NewFakeCore(t)
	signer := tests.NewSigner(t)
	recipient := common.HexToAddress("0xb0")
	pending := signer.Sign(t, &types.LegacyTx{Nonce: 0, To: &recipient, Gas: 21000, GasPrice: big.NewInt(2_000_000_000), Value: big.NewInt(5)})
	queued := signer.Sign(t, &types.LegacyTx{Nonce: 3, Gas: 60000, GasPrice: big.NewInt(3), Data: []byte{0x01}})
	fake.SetPool(map[common.Address]types.Transactions{signer.Address: {pending}}, map[common.Address]types.Transactions{signer.Address: {queued}})
	harness := tests.NewHarness(t, fake, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		txpool.Register(dispatcher, environment, fake)
	})
	return fake, harness, pending, queued, signer.Address
}

func TestContent_ListsThePoolByAccountAndNonce(t *testing.T) {
	_, harness, pending, queued, sender := newPool(t)
	content := tests.Decode[map[string]map[string]map[string]map[string]any](t, harness.Result("txpool_content"))
	require.Equal(t, pending.Hash().Hex(), content["pending"][sender.Hex()]["0"]["hash"])
	require.Equal(t, queued.Hash().Hex(), content["queued"][sender.Hex()]["3"]["hash"])
	require.Nil(t, content["pending"][sender.Hex()]["0"]["blockHash"])

	from := tests.Decode[map[string]map[string]map[string]any](t, harness.Result("txpool_contentFrom", sender))
	require.Equal(t, pending.Hash().Hex(), from["pending"]["0"]["hash"])
	require.Equal(t, queued.Hash().Hex(), from["queued"]["3"]["hash"])
	require.JSONEq(t, `{"pending":{},"queued":{}}`, string(harness.Result("txpool_contentFrom", common.HexToAddress("0x99"))))
}

func TestStatusAndInspect_SummarizeThePool(t *testing.T) {
	_, harness, _, _, sender := newPool(t)
	require.JSONEq(t, `{"pending":"0x1","queued":"0x1"}`, string(harness.Result("txpool_status")))
	inspected := tests.Decode[map[string]map[string]map[string]string](t, harness.Result("txpool_inspect"))
	require.Equal(t, "0x00000000000000000000000000000000000000B0: 5 wei + 21000 gas × 2000000000 wei", inspected["pending"][sender.Hex()]["0"])
	require.Equal(t, "contract creation: 0 wei + 60000 gas × 3 wei", inspected["queued"][sender.Hex()]["3"])
}

func TestContent_OfAnEmptyPool(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := tests.NewHarness(t, fake, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		txpool.Register(dispatcher, environment, fake)
	})
	require.JSONEq(t, `{"pending":{},"queued":{}}`, string(harness.Result("txpool_content")))
	require.JSONEq(t, `{"pending":{},"queued":{}}`, string(harness.Result("txpool_inspect")))
}
