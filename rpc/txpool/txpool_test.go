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
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/0xsoniclabs/sonic/rpc/txpool"
)

func TestTxpool_ContentStatusAndInspect(t *testing.T) {
	fc := tests.NewFakeCore()
	w := tests.NewWallet(t)
	to := common.HexToAddress("0x42")
	pending := w.Transfer(t, 0, to, big.NewInt(5))
	queued := w.Sign(t, &types.LegacyTx{Nonce: 7, Gas: 50_000, GasPrice: big.NewInt(3), Data: []byte{1}})
	fc.SetPool(map[common.Address]types.Transactions{w.Address: {pending}}, map[common.Address]types.Transactions{w.Address: {queued}})
	h := tests.NewHarness(t, fc, func(h *tests.Harness) { txpool.Register(h.D, h.Env, fc) })

	var content map[string]map[string]map[string]format.Transaction
	h.MustCall(&content, "txpool_content")
	require.Equal(t, pending.Hash(), content["pending"][w.Address.Hex()]["0"].Hash)
	require.Equal(t, queued.Hash(), content["queued"][w.Address.Hex()]["7"].Hash)
	require.Nil(t, content["pending"][w.Address.Hex()]["0"].BlockHash)

	var from map[string]map[string]format.Transaction
	h.MustCall(&from, "txpool_contentFrom", w.Address)
	require.Equal(t, pending.Hash(), from["pending"]["0"].Hash)
	h.MustCall(&from, "txpool_contentFrom", common.Address{1})
	require.Empty(t, from["pending"])
	require.NotNil(t, from["pending"])

	require.JSONEq(t, `{"pending":"0x1","queued":"0x1"}`, string(h.MustCall(nil, "txpool_status")))

	var inspect map[string]map[string]map[string]string
	h.MustCall(&inspect, "txpool_inspect")
	require.Equal(t, fmt.Sprintf("%s: 5 wei + 21000 gas × 1000000000000 wei", to.Hex()), inspect["pending"][w.Address.Hex()]["0"])
	require.Equal(t, "contract creation: 0 wei + 50000 gas × 3 wei", inspect["queued"][w.Address.Hex()]["7"])
}
