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

package gas_subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Fund creates a sponsorship fund for the given sponsee account and
// donates the given amount of wei to it from the sponsor account. It returns
// the registry instance for further queries.
func Fund(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	sponsee common.Address,
	donation *big.Int,
) *registry.Registry {
	t.Helper()
	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	ok, fundId, err := registry.AccountSponsorshipFundId(nil, sponsee)
	require.NoError(t, err)

	latestBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(t, err)

	tests.WaitForProofOf(t, client, int(latestBlock.NumberU64()))

	sponsorshipBefore, err := registry.Sponsorships(nil, fundId)
	require.NoError(t, err)

	receipt, err := session.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = donation
		require.NoError(t, err)
		require.True(t, ok, "registry should have a fund ID")
		return registry.Sponsor(opts, fundId)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	tests.WaitForProofOf(t, client, int(receipt.BlockNumber.Int64()))

	// check that the sponsorshipAfter funds got deposited
	sponsorshipAfter, err := registry.Sponsorships(nil, fundId)
	require.NoError(t, err)
	require.Equal(t, sponsorshipBefore.Funds.Uint64()+donation.Uint64(), sponsorshipAfter.Funds.Uint64())

	return registry
}

// validateSponsoredTxInBlock checks that the sponsored transaction with the
// given hash is included in a block and that it is immediately followed by a
// successful internal transaction that pays for its gas fees.
func validateSponsoredTxInBlock(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	txHash common.Hash) {
	t.Helper()

	require := require.New(t)

	client, err := session.GetClient()
	require.NoError(err)
	defer client.Close()

	receipt, err := session.GetReceipt(txHash)
	require.NoError(err)

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err)

	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(err)

	config, err := registry.GetGasConfig(nil)
	require.NoError(err)

	// Check that the payment transaction is included right after the sponsored
	// transaction and that it was successful and has a non-zero value.
	found := false
	for i, tx := range block.Transactions() {
		if tx.Hash() == receipt.TxHash {
			// check that the next transaction is an internal payment transaction
			require.Less(i+1, len(block.Transactions()), "sponsored transaction must not be last, it is last")
			payment := block.Transactions()[i+1]
			require.True(internaltx.IsInternal(payment), "payment transaction must be internal, but it is not")
			substractReceipt, err := session.GetReceipt(payment.Hash())
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, substractReceipt.Status)

			// check that the deduced amount matches the (gas used + overhead) * market price
			feeCharged := (receipt.GasUsed + config.OverheadCharge.Uint64()) * block.BaseFee().Uint64()
			require.Len(substractReceipt.Logs, 1, "no logs found in the payment transaction receipt")
			log := substractReceipt.Logs[0]
			reportedCharge := new(big.Int).SetBytes(log.Data)
			require.EqualValues(feeCharged, reportedCharge.Uint64(),
				"the fee charged does not match the expected value",
			)

			found = true
			break
		}
	}
	require.True(found, "sponsored transaction not found in the block")
}

// makeSponsoredTransactionWithNonce creates a sponsored transaction (with
// gas price zero) from the given sender to the given receiver with the given
// nonce.
func makeSponsorRequestTransaction(t *testing.T, tx types.TxData, chainId *big.Int, sender *tests.Account) *types.Transaction {
	t.Helper()
	signer := types.LatestSignerForChainID(chainId)
	switch tx := tx.(type) {
	case *types.LegacyTx:
		tx.GasPrice = big.NewInt(0)
	case *types.AccessListTx:
		tx.GasPrice = big.NewInt(0)
	case *types.DynamicFeeTx:
		tx.GasFeeCap = big.NewInt(0)
		tx.GasTipCap = big.NewInt(0)
	case *types.BlobTx:
		tx.GasFeeCap = uint256.NewInt(0)
		tx.GasTipCap = uint256.NewInt(0)
	case *types.SetCodeTx:
		tx.GasFeeCap = uint256.NewInt(0)
		tx.GasTipCap = uint256.NewInt(0)
	default:
		t.Fatalf("unexpected transaction type: %T", tx)
	}
	sponsoredTx, err := types.SignNewTx(sender.PrivateKey, signer, tx)
	require.NoError(t, err)

	// This function checks that the final transaction can indeed be a sponsorship request
	// to early detect issues while creating tests.
	// This function sets signature and gas price to 0, the only reason for it to fail
	// is if the tx.To() is nil (contract creation cannot be sponsored).
	require.True(t,
		subsidies.IsSponsorshipRequest(sponsoredTx),
		"transaction cannot be sponsor, To:%v",
		sponsoredTx.To(),
	)
	return sponsoredTx
}

// getTransactionIndexInBlock returns the index of the given transaction
// receipt in its block, along with the block itself.
func getTransactionIndexInBlock(
	t *testing.T,
	client *tests.PooledEhtClient,
	receipt *types.Receipt,
) (int, *types.Block) {
	t.Helper()
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
