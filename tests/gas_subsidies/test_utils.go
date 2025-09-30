package gas_subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Copyright 2025 Sonic Operations Ltd
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

func Fund(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	sponsor, sponsee, receiver *tests.Account,
	donation *big.Int,
) *registry.Registry {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	receipt, err := session.EndowAccount(sponsor.Address(), big.NewInt(1e18))
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	receipt, err = session.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(1e16)
		return registry.SponsorUser(opts, sponsee.Address(), receiver.Address())
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// check that the sponsorship funds got deposited
	sponsorship, err := registry.UserSponsorships(nil, sponsee.Address(), receiver.Address())
	require.NoError(t, err)
	require.Equal(t, donation, sponsorship.Funds)

	return registry
}

func validateSponsoredTxInBlock(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	txHash common.Hash) {

	require := require.New(t)

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	receipt, err := session.GetReceipt(txHash)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err)

	// Check that the payment transaction is included right after the sponsored
	// transaction and that it was successful and has a non-zero value.
	found := false
	for i, tx := range block.Transactions() {
		if tx.Hash() == receipt.TxHash {
			require.Less(i, len(block.Transactions()))
			payment := block.Transactions()[i+1]
			require.True(internaltx.IsInternal(payment), "payment transaction should be internal")
			receipt, err := session.GetReceipt(payment.Hash())
			require.NoError(err)
			require.Less(receipt.GasUsed, uint64(100_000))
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
			found = true
			break
		}
	}
	require.True(found, "sponsored transaction not found in the block")

}

func makeSponsoredTransactionWithNonce(t *testing.T,
	session tests.IntegrationTestNetSession, receiver common.Address,
	sender *tests.Account, nonce uint64) *types.Transaction {
	require := require.New(t)

	signer := types.LatestSignerForChainID(session.GetChainId())
	sponsoredTx, err := types.SignNewTx(sender.PrivateKey, signer, &types.LegacyTx{
		To:       &receiver,
		Gas:      21000,
		GasPrice: big.NewInt(0),
		Nonce:    nonce,
	})
	require.NoError(err)
	return sponsoredTx
}

func getTransactionIndexInBlock(
	t *testing.T,
	client *tests.PooledEhtClient,
	receipt *types.Receipt,
) (int, *types.Block) {
	require := require.New(t)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err)

	for i, tx := range block.Transactions() {
		if tx.Hash() == receipt.TxHash {
			return i, block
		}
	}
	require.Fail("transaction not found in block")
	return -1, nil
}
