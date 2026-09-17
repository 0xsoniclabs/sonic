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
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/nonce_priority_registry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// NoncePriorityRegistry keys a priority on (sender, nonce) rather than on the sender alone, so one
// sender's transactions can be prioritized over one stretch of nonces and ordinary over the next.
// This is the registry Faust drives; the test checks the contract answers what was registered and
// that the network schedules by those answers.
func TestPriorities_NonceRegistry_PrioritizesTheRegisteredNonceSlotsOnly(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	reg := installNonceRegistry(t, net, client)

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	other := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Register the sender's first nonces in one transaction, each slot with its own weight and id.
	const numTxs = 5
	window := make([]nonce_priority_registry.NoncePriorityRegistryPriority, numTxs)
	for i := range window {
		window[i] = nonce_priority_registry.NoncePriorityRegistryPriority{
			Level: 1, Weight: uint64(i + 1), Id: big.NewInt(int64(0xaa + i)),
		}
	}
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetPriorities(opts, sender.Address(), big.NewInt(0), window)
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// The registry answers slot by slot: what was registered inside the window, nothing right
	// after it, and nothing for a sender that was never registered.
	for i, want := range window {
		got := noncePriority(t, reg, sender.Address(), uint64(i))
		require.Equal(want.Level, got.Level, "nonce %d", i)
		require.Equal(want.Weight, got.Weight, "nonce %d", i)
		require.Equal(want.Id, got.Id, "nonce %d", i)
	}
	require.Zero(noncePriority(t, reg, sender.Address(), numTxs).Level)
	require.Zero(noncePriority(t, reg, other.Address(), 0).Level)

	config, err := reg.GetPriorityConfig(nil)
	require.NoError(err)
	require.Equal(uint64(math.MaxUint64), config.MaxGasPerEntityPerBlock.Uint64())
	require.Equal(uint64(math.MaxUint64), config.MaxPiggybackTxsPerEntityPerEvent.Uint64())

	buildSenderTxs := func() []*types.Transaction {
		nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
		require.NoError(err)
		txs := make([]*types.Transaction, numTxs)
		for i := range txs {
			txs[i] = newSignedTx(t, net, sender, nonce+uint64(i), 1, 21_000, nil)
		}
		return txs
	}
	txHasSender := func(tx *types.Transaction) bool {
		from, err := types.Sender(signer, tx)
		require.NoError(err)
		return from == sender.Address()
	}

	// The registered stretch of nonces goes ahead of ordinary traffic, and the same sender's
	// next, unregistered stretch does not.
	requirePriorityHasEffect(t, net, buildSenderTxs(), true, txHasSender)
	requirePriorityHasEffect(t, net, buildSenderTxs(), false, txHasSender)
}

// installNonceRegistry points the priority registry proxy at a fresh NoncePriorityRegistry and
// returns it bound through the proxy: the proxy delegates, so its storage is what registrations
// write and getPriority reads, and a call sent to the implementation itself would land where the
// network never looks.
func installNonceRegistry(
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
) *nonce_priority_registry.NoncePriorityRegistry {
	t.Helper()
	require := require.New(t)

	_, deployReceipt, err := tests.DeployContract(net, nonce_priority_registry.DeployNoncePriorityRegistry)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, deployReceipt.Status)
	switchPriorityRegistry(t, net, deployReceipt.ContractAddress)

	reg, err := nonce_priority_registry.NewNoncePriorityRegistry(registry.GetAddress(), client)
	require.NoError(err)

	// The limits live in slots of this implementation's own, so the ones set at start are set again.
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetConfig(opts,
			new(big.Int).SetUint64(math.MaxUint64), new(big.Int).SetUint64(math.MaxUint64))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	return reg
}

func noncePriority(
	t *testing.T,
	reg *nonce_priority_registry.NoncePriorityRegistry,
	from common.Address,
	nonce uint64,
) nonce_priority_registry.NoncePriorityRegistryPriority {
	t.Helper()
	got, err := reg.GetPriority(nil,
		from, common.Address{}, big.NewInt(1), new(big.Int).SetUint64(nonce), nil, big.NewInt(21_000))
	require.NoError(t, err)
	return nonce_priority_registry.NoncePriorityRegistryPriority(got)
}
