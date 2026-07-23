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

	priorityregistry "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// netClientSignerWithPriorities starts a fresh integration test network with
// the Brio hardfork plus TransactionPriorities enabled, applying any
// additional upgrade flags requested by `configure`. It configures generous
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

	client, err := net.GetClient()
	require.NoError(err)

	configureHighPriorityLimits(t, net, client)

	signer := types.LatestSignerForChainID(net.GetChainId())
	return net, client, signer
}

// configureHighPriorityLimits sets a generous per-entity per-block gas budget and
// per-event tx cap in the priority registry so that rate limiting never trims
// a prioritized transaction during a test. It is meant to be called once per
// test, before registering any prioritized senders.
func configureHighPriorityLimits(
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
) {
	t.Helper()
	require := require.New(t)

	reg, err := priorityregistry.NewRegistry(priorityregistry.GetAddress(), client)
	require.NoError(err)

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetConfig(opts, big.NewInt(100_000_000), big.NewInt(100))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// setPrioritized registers `sender` in the priority registry with
// (level, weight) under the given identifier. Callers are expected to have
// configured generous rate-limit values once via configureHighPriorityLimits.
func setPrioritized(
	t *testing.T,
	net *tests.IntegrationTestNet,
	sender common.Address,
	level, weight uint64,
	id uint64,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	reg, err := priorityregistry.NewRegistry(priorityregistry.GetAddress(), client)
	require.NoError(err)

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetSenderPriority(opts, sender, level, weight, new(big.Int).SetUint64(id))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// waitForReceipts waits for the receipt of every transaction hash in `hashes`.
func waitForReceipts(t *testing.T, net *tests.IntegrationTestNet, hashes []common.Hash) {
	t.Helper()
	for _, h := range hashes {
		_, err := net.GetReceipt(h)
		require.NoError(t, err)
	}
}

// buildOrdinaryTraffic constructs a set of signed, unremarkable transfers from
// freshly created, unregistered accounts. The caller is responsible for
// submitting them (typically batched together with a prioritized transaction so
// that they all reach the pool at nearly the same time, maximising the chance
// that a single block contains both).
func buildOrdinaryTraffic(
	t *testing.T,
	net *tests.IntegrationTestNet,
	signer types.Signer,
	numAccounts int,
	txsPerAccount uint64,
) []*types.Transaction {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	sink := common.Address{0x99}

	txs := make([]*types.Transaction, 0, numAccounts*int(txsPerAccount))
	for i := 0; i < numAccounts; i++ {
		acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		for n := uint64(0); n < txsPerAccount; n++ {
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    n,
				To:       &sink,
				Value:    big.NewInt(1),
				Gas:      21000,
				GasPrice: gasPrice,
			})
			signed, err := types.SignTx(tx, signer, acc.PrivateKey)
			require.NoError(err)
			txs = append(txs, signed)
		}
	}
	return txs
}

// requirePriorityAppliedSince scans the user transactions of every block from
// `firstBlock` onward, in block-then-index order, and asserts on the global
// ordering of prioritized (per `isPrioritized`) vs ordinary transactions. Both
// classes must be present for the check to be meaningful.
//
//   - expectPrioritized == true: all prioritized txs come before all ordinary
//     ones (no ordinary tx precedes a prioritized one).
//   - expectPrioritized == false: at least one ordinary tx precedes a
//     prioritized one (proof that no reordering ran).
//
// Callers should submit the whole batch via SendAllToPool, so this global
// ordering is deterministic and no per-block bookkeeping is required.
func requirePriorityAppliedSince(
	t *testing.T,
	net *tests.IntegrationTestNet,
	signer types.Signer,
	firstBlock uint64,
	expectPrioritized bool,
	isPrioritized func(common.Address) bool,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	latest, err := client.BlockNumber(t.Context())
	require.NoError(err)

	sawPriority := false
	sawOrdinary := false
	ordinaryBeforePriority := false
	for n := firstBlock; n <= latest; n++ {
		block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(n))
		require.NoError(err)
		for _, tx := range block.Transactions() {
			if internaltx.IsInternal(tx) {
				continue
			}
			sender, err := types.Sender(signer, tx)
			require.NoError(err)
			if isPrioritized(sender) {
				sawPriority = true
				if sawOrdinary {
					ordinaryBeforePriority = true
				}
			} else {
				sawOrdinary = true
			}
		}
	}

	allPrioritizedFirst := !ordinaryBeforePriority
	require.True(sawPriority && sawOrdinary)
	require.Equal(expectPrioritized, allPrioritizedFirst)
}

// requireAllNodesAgreeOnHead waits for every node in `net` to catch up to the
// current head as observed on node 0, and asserts that they all report the
// same block hash at that height.
func requireAllNodesAgreeOnHead(t *testing.T, net *tests.IntegrationTestNet) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	head, err := client.HeaderByNumber(t.Context(), nil)
	require.NoError(err)
	referenceNumber := head.Number.Uint64()
	referenceHash := head.Hash()

	for i := 1; i < net.NumNodes(); i++ {
		nc, err := net.GetClientConnectedToNode(i)
		require.NoError(err)
		defer nc.Close()
		tests.WaitForProofOf(t, nc, int(referenceNumber))
		header, err := nc.HeaderByNumber(t.Context(), new(big.Int).SetUint64(referenceNumber))
		require.NoError(err)
		require.Equal(referenceHash, header.Hash(),
			"node %d disagrees with node 0 on block %d hash", i, referenceNumber)
	}
}
