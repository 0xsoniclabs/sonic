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

// Package priority contains end-to-end integration tests for the transaction
// priorities feature.
package priority

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPriorities_PrioritizedTransactionsAreScheduledFirst demonstrates the
// end-to-end behavior of the transaction priorities feature: a configurable
// on-chain registry designates some senders as prioritized, and the resulting
// blocks schedule those transactions ahead of ordinary ones, ordered by
// (level, weight), regardless of submission order.
//
// It runs in both block-formation modes. In single-proposer mode this also
// exercises the authoritative override: even though the proposer schedules the
// transactions, block formation re-derives and enforces the priority order.
func TestPriorities_PrioritizedTransactionsAreScheduledFirst(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		testPrioritiesScheduledFirst(t, false)
	})
	t.Run("single-proposer", func(t *testing.T) {
		testPrioritiesScheduledFirst(t, true)
	})
}

func testPrioritiesScheduledFirst(t *testing.T, singleProposer bool) {
	require := require.New(t)

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true
	upgrades.SingleProposerBlockFormation = singleProposer
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// The registry must have been deployed in genesis.
	code, err := client.CodeAt(t.Context(), registry.GetAddress(), nil)
	require.NoError(err)
	require.NotEmpty(code, "priority registry must be deployed")

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	// Configure generous per-entity limits so rate limiting does not interfere.
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetConfig(opts, big.NewInt(100), big.NewInt(100))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Define prioritized accounts, each with a distinct (level, weight). They are
	// declared in the order we expect them to appear within a block.
	type prioritized struct {
		account *tests.Account
		level   int64
		weight  int64
	}
	prios := []prioritized{
		{level: 2, weight: 50}, // highest level -> first
		{level: 1, weight: 90},
		{level: 1, weight: 10},
	}
	prioByAddr := map[common.Address]prioritized{}
	for i := range prios {
		acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
		prios[i].account = acc
		prioByAddr[acc.Address()] = prios[i]

		id := common.Hash{byte(i + 1)}
		r, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return reg.SetSenderPriority(opts, acc.Address(),
				big.NewInt(prios[i].level), big.NewInt(prios[i].weight), id)
		})
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, r.Status)
	}

	// Define ordinary accounts.
	const numNormal = 4
	const txsPerNormal = 3
	normals := make([]*tests.Account, numNormal)
	for i := range normals {
		normals[i] = tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	}

	chainID := net.GetChainId()
	signer := types.LatestSignerForChainID(chainID)
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	sink := common.Address{0x99}

	sign := func(acc *tests.Account, nonce uint64) *types.Transaction {
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			To:       &sink,
			Value:    big.NewInt(1),
			Gas:      21000,
			GasPrice: gasPrice,
		})
		signed, err := types.SignTx(tx, signer, acc.PrivateKey)
		require.NoError(err)
		return signed
	}

	// Build a burst: ordinary transactions first, prioritized ones last (so a
	// naive ordering would not place them first). Each prioritized account sends
	// a single transaction to avoid intra-sender nonce reordering.
	burst := make(types.Transactions, 0, numNormal*txsPerNormal+len(prios))
	for _, acc := range normals {
		for n := uint64(0); n < txsPerNormal; n++ {
			burst = append(burst, sign(acc, n))
		}
	}
	for i := range prios {
		burst = append(burst, sign(prios[i].account, 0))
	}

	hashes, err := net.SendAll(burst)
	require.NoError(err)
	for _, h := range hashes {
		_, err := net.GetReceipt(h)
		require.NoError(err)
	}

	// Inspect all blocks: within each block, prioritized user transactions must
	// form a prefix (appear before any ordinary user transaction) and be ordered
	// by (level desc, weight desc). At least one block must mix both classes to
	// prove that prioritized transactions actually jumped ahead.
	latest, err := client.BlockNumber(t.Context())
	require.NoError(err)

	mixSeen := false
	for n := uint64(0); n <= latest; n++ {
		block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(n))
		require.NoError(err)

		sawNormal := false
		prioInBlock := make([]prioritized, 0)
		for _, tx := range block.Transactions() {
			if internaltx.IsInternal(tx) {
				continue
			}
			sender, err := types.Sender(signer, tx)
			if err != nil {
				continue
			}
			if p, ok := prioByAddr[sender]; ok {
				require.False(sawNormal,
					"block %d: prioritized tx scheduled after an ordinary tx", n)
				prioInBlock = append(prioInBlock, p)
			} else {
				sawNormal = true
			}
		}

		if len(prioInBlock) > 0 && sawNormal {
			mixSeen = true
		}
		for i := 1; i < len(prioInBlock); i++ {
			prev, cur := prioInBlock[i-1], prioInBlock[i]
			ordered := prev.level > cur.level ||
				(prev.level == cur.level && prev.weight >= cur.weight)
			require.True(ordered,
				"block %d: prioritized txs not ordered by (level, weight): %+v before %+v",
				n, prev, cur)
		}
	}

	require.True(mixSeen,
		"expected at least one block containing both prioritized and ordinary transactions")
}
