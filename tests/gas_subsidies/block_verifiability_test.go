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

package gas_subsidies

import (
	"encoding/binary"
	"math/big"
	"sync"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBlockVerifiability(t *testing.T) {
	tests := map[string]opera.Upgrades{
		"sonic":   opera.GetSonicUpgrades(),
		"allegro": opera.GetAllegroUpgrades(),
		"brio":    opera.GetBrioUpgrades(),
	}

	for name, upgrades := range tests {
		t.Run(name, func(t *testing.T) {
			single := upgrades
			single.SingleProposerBlockFormation = true
			distributed := upgrades
			distributed.SingleProposerBlockFormation = false
			t.Run("single", func(t *testing.T) {
				testBlockVerifiability(t, single)
			})
			t.Run("distributed", func(t *testing.T) {
				testBlockVerifiability(t, distributed)
			})
		})
	}
}

func testBlockVerifiability(t *testing.T, upgrades opera.Upgrades) {
	const N = 250
	require := require.New(t)
	updates := opera.GetSonicUpgrades()
	updates.GasSubsidies = true

	net := tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
		Upgrades: &updates,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Add some sponsorship funds to the registry.
	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	_, id, err := reg.GlobalSponsorshipFundId(nil)
	require.NoError(err)

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(1e18)
		return reg.Sponsor(opts, id)
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	tests.WaitForProofOf(t, client, int(receipt.BlockNumber.Int64()))

	// Check sponsorship balance.
	state, err := reg.Sponsorships(nil, id)
	require.NoError(err)
	require.Equal(big.NewInt(1e18), state.Funds)

	// Run some sponsored transactions interacting with the registry.
	account := tests.NewAccount()
	signer := types.LatestSignerForChainID(net.GetChainId())
	txs := []*types.Transaction{}
	for i := range N {
		// We create calls to the registries "chooseFund" method, since this is
		// the function which is called when checking if a transaction is
		// sponsored. If warm slots are leaking from those sponsorship calls
		// into the actual transaction execution, it should affect the gas
		// used by these transactions.
		//
		// Unfortunately, there seems to be no way to get those transactions
		// from the generated registry binding, since this is an unintended
		// use-case, so we create the transactions manually.
		receiver := registry.GetAddress()
		data := make([]byte, 4+6*32+2*32)

		// Add the function selector.
		binary.BigEndian.PutUint32(data, registry.ChooseFundFunctionSelector)

		// Add the offset for the call data parameter.
		data[4+4*32+31] = 32 * 6

		// Make sure that the to address (parameter 2) is not zero.
		data[4+32+31] = 1

		// Request some fees (parameter 5).
		data[4+5*32+31] = 123

		tx := types.MustSignNewTx(account.PrivateKey, signer, &types.LegacyTx{
			To:       &receiver,
			Nonce:    uint64(i),
			Data:     data,
			Gas:      70_000,
			GasPrice: big.NewInt(0),
		})
		require.True(subsidies.IsSponsorshipRequest(tx))

		txs = append(txs, tx)
	}

	// Advance epochs while running sponsored transactions to introduce additional
	// internal transactions. The aim is to verify proper use of nonces in internal
	// transactions.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 2 {
			net.AdvanceEpoch(t, 1)
		}
	}()

	receipts, err := net.RunAll(txs)
	require.NoError(err)
	require.Equal(N, len(receipts))

	for i, receipt := range receipts {
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "tx %d failed", i)
	}

	wg.Wait()

	// Check that all sponsored transactions were successful and for free.
	lastBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Download all blocks of this chain.
	blocks := []*types.Block{}
	for i := range lastBlock + 1 {
		b, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(err)
		blocks = append(blocks, b)
	}

	// Check proper use of nonces in internal transactions.
	next := uint64(0)
	for _, b := range blocks {
		for _, tx := range b.Transactions() {
			if internaltx.IsInternal(tx) {
				require.Equal(next, tx.Nonce())
				next++
			}
		}
	}

	// Make sure that the history can be verified using the genesis and block
	// data only. This facilitates compatibility with downstream tools.
	genesis := net.GetJsonGenesis()
	require.NotNil(genesis, "network must be started with JSON genesis")
	tests.VerifyBlocks(t, genesis, blocks)
}
