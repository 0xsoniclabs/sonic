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

package priority

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/even_value_priority"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPriority_UpdateContract_PrioritizationChanges verifies that swapping the
// on-chain priority-registry implementation via the proxy actually changes the
// classifier consulted by the network. It exercises two phases sequentially:
//
//   - default (sender-based) registry: only senders registered via
//     `setSenderPriority` are prioritized; the tx `value` is ignored.
//   - `EvenValuePriority` (value-based) registry: the previously registered
//     sender is no longer prioritized; instead any tx with an even `value`
//     is, regardless of sender.
//
// Each phase is checked via direct RPC probes against the registry's
// read-only `getPriority` entry point and additionally on the consensus
// path via requireClassifierAppliedOnConsensusPath, which submits a mixed
// batch and asserts that in blocks containing both classes the prioritized
// txs form a prefix.
func TestPriority_UpdateContract_PrioritizationChanges(t *testing.T) {
	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	// Register one sender in the default registry; the other is a funded
	// account that stays unregistered throughout the test.
	registered := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	setPrioritized(t, net, registered.Address(), 1, 0, common.Hash{0xaa})
	unregistered := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Phase 1: default registry - prioritization depends only on the sender.
	defaultRegistryProbes := map[string]struct {
		from              common.Address
		value             int64
		expectPrioritized bool
	}{
		"default: registered/odd value":    {registered.Address(), 1, true},
		"default: registered/even value":   {registered.Address(), 2, true},
		"default: unregistered/odd value":  {unregistered.Address(), 1, false},
		"default: unregistered/even value": {unregistered.Address(), 2, false},
	}
	for name, tc := range defaultRegistryProbes {
		require.Equal(tc.expectPrioritized, checkPriority(t, reg, tc.from, tc.value), name)
	}

	// Consensus-path proof for the default registry: the registered sender's
	// txs (with alternating odd/even values, to additionally exercise that
	// value parity is irrelevant here) must be scheduled ahead of ordinary
	// txs in blocks that contain both.
	requireClassifierAppliedOnConsensusPath(t, net, signer,
		signTxs(t, client, signer, registered, []int64{1, 2, 3, 4}))

	// Swap the priority-registry implementation to EvenValuePriority.
	switchContract(t, net, client, even_value_priority.DeployEvenValuePriority)

	// Phase 2: EvenValuePriority - prioritization depends only on `value` parity.
	evenValueRegistryProbes := map[string]struct {
		from              common.Address
		value             int64
		expectPrioritized bool
	}{
		"even-value: prev-registered/even value": {registered.Address(), 2, true},
		"even-value: prev-registered/odd value":  {registered.Address(), 1, false},
		"even-value: unregistered/even value":    {unregistered.Address(), 2, true},
		"even-value: unregistered/odd value":     {unregistered.Address(), 1, false},
	}
	for name, tc := range evenValueRegistryProbes {
		require.Equal(tc.expectPrioritized, checkPriority(t, reg, tc.from, tc.value), name)
	}

	// Consensus-path proof for EvenValuePriority: the previously registered
	// sender (now classified purely by value under EvenValuePriority) sends
	// only even-value txs, which must be scheduled ahead of the ordinary
	// odd-value background traffic in blocks that contain both.
	requireClassifierAppliedOnConsensusPath(t, net, signer,
		signTxs(t, client, signer, registered, []int64{2, 4, 6, 8}))
}

// checkPriority invokes the registry's `getPriority` entry point for a
// transaction with the given `from` and `value`, and returns whether
// the returned level marks the tx as prioritized.
func checkPriority(
	t *testing.T,
	reg *registry.Registry,
	from common.Address,
	value int64,
) bool {
	t.Helper()
	priority, err := reg.GetPriority(nil,
		from, common.Address{}, big.NewInt(value), big.NewInt(0), nil, big.NewInt(21_000))
	require.NoError(t, err)
	return priority.Level.Sign() > 0
}

// switchContract deploys the priority-registry replacement returned by
// `deploy` and points the proxy at it, asserting the proxy's implementation
// slot indeed holds the new address afterwards.
func switchContract[T any](
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
	deploy tests.ContractDeployer[T],
) {
	t.Helper()
	require := require.New(t)

	_, deployReceipt, err := tests.DeployContract(net, deploy)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, deployReceipt.Status)
	newImpl := deployReceipt.ContractAddress

	pxy, err := proxy.NewProxy(registry.GetAddress(), client)
	require.NoError(err)
	updateReceipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return pxy.Update(opts, newImpl)
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, updateReceipt.Status)

	slotValue, err := client.StorageAt(t.Context(), registry.GetAddress(),
		proxy.GetSlotForImplementation(), nil)
	require.NoError(err)
	var storedAddr common.Address
	copy(storedAddr[:], slotValue[12:])
	require.Equal(newImpl, storedAddr)
}

// requireClassifierAppliedOnConsensusPath proves the currently installed
// priority classifier is consulted on the block-formation path. It submits
// `prioritizedTxs` (transactions the current classifier is expected to
// prioritize) together with a burst of ordinary background traffic from
// fresh, unregistered senders in a single batch through the tx pool, and
// asserts that in every block containing both classes the prioritized txs form
// a prefix. The senders of `prioritizedTxs` must be disjoint from the
// background-traffic senders which is always the case.
func requireClassifierAppliedOnConsensusPath(
	t *testing.T,
	net *tests.IntegrationTestNet,
	signer types.Signer,
	prioritizedTxs []*types.Transaction,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	prioritizedSenders := map[common.Address]struct{}{}
	for _, tx := range prioritizedTxs {
		from, err := types.Sender(signer, tx)
		require.NoError(err)
		prioritizedSenders[from] = struct{}{}
	}

	// Pre-build ordinary background traffic (fresh accounts are funded now
	// so their funding txs don't compete for block space with the actual
	// test batch).
	ordinaryTxs := buildOrdinaryTraffic(t, net, signer, 10, 10)

	firstBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Submit prioritized + ordinary txs in one batch so they reach the pool
	// almost simultaneously, maximizing the chance that a single block
	// contains both classes.
	batch := append([]*types.Transaction{}, ordinaryTxs...)
	batch = append(batch, prioritizedTxs...)
	hashes, err := net.SendAll(batch)
	require.NoError(err)
	waitForReceipts(t, net, hashes)

	requirePriorityAppliedSince(t, net, signer, firstBlock, true,
		func(a common.Address) bool {
			_, ok := prioritizedSenders[a]
			return ok
		},
	)
}

// signTxs returns one signed transfer from `sender` per entry in `values`,
// using consecutive nonces starting at `sender`'s current pending nonce.
// The transfers are to a fixed sink address and carry the given value.
func signTxs(
	t *testing.T,
	client *tests.PooledEhtClient,
	signer types.Signer,
	sender *tests.Account,
	values []int64,
) []*types.Transaction {
	t.Helper()
	require := require.New(t)

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
	require.NoError(err)
	sink := common.Address{0x99}

	txs := make([]*types.Transaction, len(values))
	for i, v := range values {
		tx, err := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    nonce + uint64(i),
			To:       &sink,
			Value:    big.NewInt(v),
			Gas:      21_000,
			GasPrice: gasPrice,
		}), signer, sender.PrivateKey)
		require.NoError(err)
		txs[i] = tx
	}
	return txs
}
