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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// TestPendingTransactions_ListsThePool records the deviation of section 6:
// the reference listed only the transactions of the node's accounts.
func TestPendingTransactions_ListsThePool(t *testing.T) {
	fake := tests.NewFakeCore(t)
	harness := newEth(t, fake)
	signer := tests.NewSigner(t)
	pending := transfer(t, signer, 0, big.NewInt(1_000_000_000))
	fake.SetPool(map[common.Address]types.Transactions{signer.Address: {pending}}, nil)
	listed := tests.Decode[[]map[string]any](t, harness.Result("eth_pendingTransactions"))
	require.Len(t, listed, 1)
	require.Equal(t, pending.Hash().Hex(), listed[0]["hash"])
	require.Nil(t, listed[0]["blockHash"])
}
