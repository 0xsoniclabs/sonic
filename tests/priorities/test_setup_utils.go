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

package priorities

import (
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// netClientSignerWithPriorities starts a fresh integration test network with
// the Brio hard-fork and TransactionPriorities enabled by default but
// overridable with `configure`. It configures effectively unlimited
// priority-registry limits and returns the network together with an open
// client and a signer bound to the network's chain ID. The caller owns the
// returned client and is expected to Close it.
func netClientSignerWithPriorities(
	t *testing.T,
	configure func(*opera.Upgrades),
) (*tests.IntegrationTestNet, *tests.PooledEhtClient, types.Signer) {
	t.Helper()
	require := require.New(t)

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true
	if configure != nil {
		configure(&upgrades)
	}
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	setPriorityConfig(t, net, math.MaxUint64, math.MaxUint64)

	client, err := net.GetClient()
	require.NoError(err)

	return net, client, types.LatestSignerForChainID(net.GetChainId())
}

// setPriorityConfig sets the per-entity rate limits of the priority registry.
func setPriorityConfig(
	t *testing.T,
	net *tests.IntegrationTestNet,
	maxGasPerEntityPerBlock, maxPiggybackTxsPerEntityPerEvent uint64,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetConfig(opts,
			new(big.Int).SetUint64(maxGasPerEntityPerBlock),
			new(big.Int).SetUint64(maxPiggybackTxsPerEntityPerEvent))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// newFundedPrioritizedAccount creates a funded account and registers it in the
// priority registry with (level, weight, id).
func newFundedPrioritizedAccount(
	t *testing.T,
	net *tests.IntegrationTestNet,
	level, weight, id uint64,
) *tests.Account {
	t.Helper()
	acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	setPrioritized(t, net, acc.Address(), level, weight, id)
	return acc
}

// setPrioritized registers `sender` in the priority registry with
// (level, weight, id).
func setPrioritized(
	t *testing.T,
	net *tests.IntegrationTestNet,
	sender common.Address,
	level, weight, id uint64,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetSenderPriority(opts, sender, level, weight, new(big.Int).SetUint64(id))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// switchPriorityRegistry points the priority-registry proxy at `newImpl`,
// asserting the proxy's implementation slot changes to that address.
func switchPriorityRegistry(
	t *testing.T,
	net *tests.IntegrationTestNet,
	newImpl common.Address,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	oldSlotValue, err := client.StorageAt(t.Context(), registry.GetAddress(),
		proxy.GetSlotForImplementation(), nil)
	require.NoError(err)
	require.NotEqual(newImpl, common.BytesToAddress(oldSlotValue))

	pxy, err := proxy.NewProxy(registry.GetAddress(), client)
	require.NoError(err)
	updateReceipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return pxy.Update(opts, newImpl)
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, updateReceipt.Status)

	newSlotValue, err := client.StorageAt(t.Context(), registry.GetAddress(),
		proxy.GetSlotForImplementation(), nil)
	require.NoError(err)
	require.Equal(newImpl, common.BytesToAddress(newSlotValue))
}

// newSignedTx builds and signs transfer from `account`. If `gasPrice` is nil,
// the network's currently suggested gas price is used.
func newSignedTx(
	t *testing.T,
	net *tests.IntegrationTestNet,
	account *tests.Account,
	nonce uint64,
	value uint64,
	gasLimit uint64,
	gasPrice *big.Int,
) *types.Transaction {
	t.Helper()
	require := require.New(t)

	if gasPrice == nil {
		client, err := net.GetClient()
		require.NoError(err)
		defer client.Close()

		gasPrice, err = client.SuggestGasPrice(t.Context())
		require.NoError(err)
	}

	signer := types.LatestSignerForChainID(net.GetChainId())
	return types.MustSignNewTx(account.PrivateKey, signer, &types.LegacyTx{
		Nonce:    nonce,
		To:       &common.Address{0x99},
		Value:    new(big.Int).SetUint64(value),
		Gas:      gasLimit,
		GasPrice: gasPrice,
	})
}
