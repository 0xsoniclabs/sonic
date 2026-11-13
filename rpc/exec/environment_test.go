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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// newEnvironment serves a chain of three blocks, the first with one
// transaction, up to block 1; block 2 is stored but not served.
func newEnvironment(t *testing.T) (*tests.Harness, *types.Transaction) {
	t.Helper()
	account := tests.NewAccount()
	transaction := account.Sign(&types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(1)})
	fake := tests.NewFakeCore()
	fake.AddBlock(transaction)
	fake.AddBlock(account.Sign(&types.LegacyTx{Nonce: 1, Gas: 21000, GasPrice: big.NewInt(1)}))
	fake.SetHead(1)
	return tests.NewHarness(t, fake, func(*tests.Harness) {}), transaction
}

func TestEnvironment_FindsNothingAboveTheHead(t *testing.T) {
	h, _ := newEnvironment(t)
	head := h.Environment.Head()
	require.Equal(t, uint64(1), head.NumberU64())
	cases := map[string]func(number uint64) error{
		"header": func(number uint64) error {
			_, err := h.Environment.HeaderAt(head, number)
			return err
		},
		"block": func(number uint64) error {
			_, err := h.Environment.BlockAt(head, number)
			return err
		},
		"receipts": func(number uint64) error {
			_, err := h.Environment.ReceiptsAt(head, number)
			return err
		},
	}
	for name, lookup := range cases {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, lookup(1))
			require.ErrorIs(t, lookup(2), core.ErrNotFound)
		})
	}
}

func TestEnvironment_FindsBlocksByReference(t *testing.T) {
	h, _ := newEnvironment(t)
	head := h.Environment.Head()
	stored, err := h.Core.Block(2)
	require.NoError(t, err)
	unknown := common.Hash{0x01}
	cases := map[string]struct {
		reference chain.BlockReference
		want      uint64
		err       error
	}{
		"latest":                {chain.ReferenceTo(chain.LatestBlockNumber), 1, nil},
		"number":                {chain.ReferenceTo(0), 0, nil},
		"number above the head": {chain.ReferenceTo(2), 0, core.ErrNotFound},
		"hash":                  {chain.BlockReference{Hash: &head.Hash}, 1, nil},
		"hash above the head":   {chain.BlockReference{Hash: &stored.Hash}, 0, chain.ErrUnknownHash},
		"unknown hash":          {chain.BlockReference{Hash: &unknown}, 0, chain.ErrUnknownHash},
		"missing selector":      {chain.BlockReference{}, 0, chain.ErrMissingSelector},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			block, err := h.Environment.FindBlock(head, test.reference)
			require.ErrorIs(t, err, test.err)
			header, headerErr := h.Environment.FindHeader(head, test.reference)
			require.ErrorIs(t, headerErr, test.err)
			if test.err == nil {
				require.Equal(t, test.want, block.NumberU64())
				require.Equal(t, test.want, header.Number.Uint64())
			}
		})
	}
}

func TestEnvironment_FindsTransactionsUpToTheHead(t *testing.T) {
	h, transaction := newEnvironment(t)
	head := h.Environment.Head()
	found, block, index, err := h.Environment.TransactionByHash(head, transaction.Hash())
	require.NoError(t, err)
	require.Equal(t, transaction.Hash(), found.Hash())
	require.Equal(t, uint64(1), block)
	require.Zero(t, index)

	stored, err := h.Core.Block(2)
	require.NoError(t, err)
	h.SetHead(t, 2)
	_, _, _, err = h.Environment.TransactionByHash(head, stored.Transactions[0].Hash())
	require.ErrorIs(t, err, core.ErrNotFound, "a transaction above the head the request read is not there")

	h.Core.SetMode("TransactionByHash", tests.Fail)
	_, _, _, err = h.Environment.TransactionByHash(head, transaction.Hash())
	require.ErrorIs(t, err, tests.ErrInjected)
}

func TestEnvironment_DescribesTheChain(t *testing.T) {
	h, _ := newEnvironment(t)
	require.Equal(t, big.NewInt(tests.ChainID), h.Environment.ChainID())
	require.Equal(t, uint64(tests.ChainID), h.Environment.ChainInfo().ChainID)

	h.Core.SetUpgradeHeights([]opera.UpgradeHeight{
		opera.MakeUpgradeHeight(opera.GetSonicUpgrades(), 0),
		opera.MakeUpgradeHeight(opera.GetAllegroUpgrades(), 5),
	})
	require.Nil(t, h.Environment.ChainConfig(4).PragueTime)
	require.NotNil(t, h.Environment.ChainConfig(5).PragueTime)
}

func TestNoneIfMissing_AnswersAMissingItemAsNoValue(t *testing.T) {
	cases := map[string]struct {
		err     error
		value   *int
		wantErr error
	}{
		"found":        {nil, pointer(1), nil},
		"not found":    {core.ErrNotFound, nil, nil},
		"unknown hash": {chain.ErrUnknownHash, nil, nil},
		"failure":      {tests.ErrInjected, pointer(1), tests.ErrInjected},
		"no selector":  {chain.ErrMissingSelector, pointer(1), chain.ErrMissingSelector},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			value, err := exec.NoneIfMissing(pointer(1), test.err)
			require.ErrorIs(t, err, test.wantErr)
			require.Equal(t, test.value, value)
		})
	}
}

func pointer[Value any](value Value) *Value {
	return &value
}
