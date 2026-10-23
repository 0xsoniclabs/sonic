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

package txpool

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestTxPool_Methods(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := tests.NewHarness(t, fc, func(d *dispatch.Dispatcher, chain *dispatch.Chain) { Register(d, chain) })
	alice := tests.NewAccount(t, nil, nil)
	bob := tests.NewAccount(t, nil, nil)
	tx0 := alice.Transfer(t, 0, bob.Addr, 5)
	tx1 := alice.Transfer(t, 1, bob.Addr, 6)
	other := bob.Call(t, 0, common.Address{}, 0, []byte{1}, 50_000) // deploys nothing: to is zero address
	fc.AddPoolTransaction(tx0)
	fc.AddPoolTransaction(tx1)
	fc.AddPoolTransaction(other)

	var status map[string]string
	h.Into(&status, "txpool_status")
	require.Equal(t, map[string]string{"pending": "0x3", "queued": "0x0"}, status)

	var content map[string]map[string]map[string]map[string]any
	h.Into(&content, "txpool_content")
	require.Len(t, content["pending"], 2)
	require.Empty(t, content["queued"])
	byNonce := content["pending"][alice.Addr.Hex()]
	require.Equal(t, tx0.Hash().Hex(), byNonce["0"]["hash"])
	require.Equal(t, tx1.Hash().Hex(), byNonce["1"]["hash"])
	// Transactions in the pool have no block.
	require.Nil(t, byNonce["0"]["blockHash"])

	var from map[string]map[string]map[string]any
	h.Into(&from, "txpool_contentFrom", alice.Addr)
	require.Len(t, from["pending"], 2)
	require.Empty(t, from["queued"])
	h.Into(&from, "txpool_contentFrom", common.Address{0x77})
	require.Empty(t, from["pending"])
	require.NotNil(t, from["pending"])

	var inspect map[string]map[string]map[string]string
	h.Into(&inspect, "txpool_inspect")
	want := bob.Addr.Hex() + ": " + big.NewInt(5).String() + " wei + 100000 gas × 1000 wei"
	require.Equal(t, want, inspect["pending"][alice.Addr.Hex()]["0"])
	require.Contains(t, inspect["pending"][bob.Addr.Hex()]["0"], "0x0000000000000000000000000000000000000000: 0 wei + 50000 gas")
}

func TestTxPool_CoreFailuresAreNotPanics(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(1))
	h := tests.NewHarness(t, fc, func(d *dispatch.Dispatcher, chain *dispatch.Chain) { Register(d, chain) })
	for _, mode := range []tests.Mode{tests.Fail, tests.Nil, tests.NotFound} {
		fc.Inject("*", mode)
		for _, method := range []string{"txpool_content", "txpool_status", "txpool_inspect"} {
			h.Result(method)
		}
		h.Result("txpool_contentFrom", common.Address{})
	}
	require.Zero(t, h.Panics())
}
