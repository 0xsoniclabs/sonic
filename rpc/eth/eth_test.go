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

	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

// testChain is a fake chain of three blocks: the genesis, a block with a
// legacy, an access list and a dynamic fee transaction, and an empty block,
// which is the head; a fourth block is stored but not yet served.
type testChain struct {
	fake         *tests.FakeCore
	account      *tests.Account
	transactions types.Transactions
	unserved     common.Hash
}

func newTestChain(t *testing.T) *testChain {
	t.Helper()
	account := tests.NewAccount()
	recipient := common.Address{0x42}
	transactions := types.Transactions{
		account.Sign(&types.LegacyTx{Nonce: 0, To: &recipient, Gas: 21000, GasPrice: big.NewInt(2e9), Value: big.NewInt(1)}),
		account.Sign(&types.AccessListTx{ChainID: big.NewInt(tests.ChainID), Nonce: 1, To: &recipient, Gas: 21000, GasPrice: big.NewInt(2e9)}),
		account.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), Nonce: 2, Gas: 50000, GasFeeCap: big.NewInt(3e9), GasTipCap: big.NewInt(1e9), Data: []byte{0x60}}),
	}
	fake := tests.NewFakeCore()
	fake.AddBlock(transactions...)
	fake.AddBlock()
	fake.SetHead(2)
	fake.AddBlock()
	unserved, err := fake.Block(3)
	if err != nil {
		t.Fatal(err)
	}
	return &testChain{fake: fake, account: account, transactions: transactions, unserved: unserved.Hash}
}

func (c *testChain) hashOf(t *testing.T, number uint64) common.Hash {
	t.Helper()
	block, err := c.fake.Block(number)
	if err != nil {
		t.Fatal(err)
	}
	return block.Hash
}

func newHarness(t *testing.T, fake *tests.FakeCore) *tests.Harness {
	t.Helper()
	return tests.NewHarness(t, fake, func(h *tests.Harness) {
		eth.Register(h.Dispatcher, h.Environment, h.Core)
	})
}

func trieHasher() *trie.StackTrie {
	return trie.NewStackTrie(nil)
}
