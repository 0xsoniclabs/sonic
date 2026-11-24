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

package account_freeze

import (
	"math/big"
	"testing"

	freeze_registry "github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestAccountFreeze_FrozenSenders(t *testing.T) {
	upgrades := opera.GetCantoUpgrades()
	upgrades.AccountFreeze = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	reg, err := freeze_registry.NewRegistry(freeze_registry.GetAddress(), client)
	require.NoError(t, err)

	oneS := big.NewInt(1e18)
	frozen := tests.MakeAccountWithBalance(t, net, oneS)
	freeze(t, net, reg, frozen.Address(), true)

	t.Run("pool rejects frozen sender", func(t *testing.T) {
		tx := valueTransfer(t, net, frozen, common.Address{0x1})
		_, err := net.Send(tx)
		require.ErrorContains(t, err, "account is frozen")
	})

	t.Run("frozen sender is skipped in a block", func(t *testing.T) {
		tx := valueTransfer(t, net, frozen, common.Address{0x1})
		nonce, err := client.NonceAt(t.Context(), frozen.Address(), nil)
		require.NoError(t, err)

		_, err = net.ForceEmit(t.Context(), tx)
		require.NoError(t, err)

		waitBlocks(t, net, 3)

		_, err = client.TransactionReceipt(t.Context(), tx.Hash())
		require.ErrorIs(t, err, ethereum.NotFound)
		after, err := client.NonceAt(t.Context(), frozen.Address(), nil)
		require.NoError(t, err)
		require.Equal(t, nonce, after)
	})

	t.Run("pool drops pending transactions of a newly frozen sender", func(t *testing.T) {
		account := tests.MakeAccountWithBalance(t, net, oneS)
		nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		to := common.Address{0x1}
		gapped := tests.CreateTransaction(t, net, &types.LegacyTx{Nonce: nonce + 1, To: &to, Value: big.NewInt(1), Gas: 21_000}, account)
		_, err = net.Send(gapped)
		require.NoError(t, err)
		require.Len(t, poolContentFrom(t, net, account.Address())["queued"], 1)

		freeze(t, net, reg, account.Address(), true)
		content := poolContentFrom(t, net, account.Address())
		require.Empty(t, content["queued"])
		require.Empty(t, content["pending"])
	})
}

func poolContentFrom(t *testing.T, net *tests.IntegrationTestNet, account common.Address) map[string]map[string]any {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	var content map[string]map[string]any
	require.NoError(t, client.Client().Call(&content, "txpool_contentFrom", account))
	return content
}

// freeze sets the freeze status of account and waits until the pool has seen it.
func freeze(t *testing.T, net *tests.IntegrationTestNet, reg *freeze_registry.Registry, account common.Address, value bool) *types.Receipt {
	t.Helper()
	var tx *types.Transaction
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		var err error
		tx, err = reg.SetFrozen(opts, account, value)
		return tx, err
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	require.NoError(t, tests.WaitUntilTransactionIsRetiredFromPool(t, client, tx))
	return receipt
}

func valueTransfer(t *testing.T, net *tests.IntegrationTestNet, from *tests.Account, to common.Address) *types.Transaction {
	t.Helper()
	return tests.CreateTransaction(t, net, &types.LegacyTx{To: &to, Value: big.NewInt(1), Gas: 21_000}, from)
}

func waitBlocks(t *testing.T, net *tests.IntegrationTestNet, n int) {
	t.Helper()
	for range n {
		_, err := net.EndowAccount(common.Address{0x2}, big.NewInt(1))
		require.NoError(t, err)
	}
}
