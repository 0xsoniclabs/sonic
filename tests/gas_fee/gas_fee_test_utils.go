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

package gasfee

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func createTransactionMix(
	t testing.TB,
	session tests.IntegrationTestNetSession,
	numTransactions int,
) []*types.Transaction {

	perPart := numTransactions / 3
	firstPart := numTransactions - 2*perPart

	regularTxs := createRegularTransactions(t, session, firstPart)
	sponsoredTxs := createSponsoredTransactions(t, session, perPart)
	bundledTxs := createBundledTransactions(t, session, perPart)

	res := make([]*types.Transaction, 0, numTransactions)
	res = append(res, regularTxs...)
	res = append(res, sponsoredTxs...)
	res = append(res, bundledTxs...)
	require.Len(t, res, numTransactions)

	rand.Shuffle(len(res), func(i, j int) {
		res[i], res[j] = res[j], res[i]
	})

	return res
}

func createRegularTransactions(
	t testing.TB,
	session tests.IntegrationTestNetSession,
	numTransactions int,
) []*types.Transaction {

	accounts := tests.MakeAccountsWithBalance(t, session, numTransactions, big.NewInt(1e18))
	gasPrice := getGasPrice(t, session)
	signer := types.LatestSignerForChainID(session.GetChainId())

	res := make([]*types.Transaction, 0, numTransactions)

	parts := numTransactions / 3
	firstPart := numTransactions - 2*parts

	// do a few legacy transactions
	for range firstPart {
		tx := types.MustSignNewTx(
			accounts[0].PrivateKey, signer, &types.LegacyTx{
				To:       &common.Address{}, // send to the zero address
				Gas:      21_000,
				GasPrice: gasPrice,
			},
		)
		res = append(res, tx)
		accounts = accounts[1:]
	}

	// followed by a few access list transactions
	for range parts {
		tx := types.MustSignNewTx(
			accounts[0].PrivateKey, signer, &types.AccessListTx{
				To:       &common.Address{}, // send to the zero address
				Gas:      21_000,
				GasPrice: gasPrice,
			},
		)
		res = append(res, tx)
		accounts = accounts[1:]
	}

	// followed by a few dynamic fee transactions
	for range parts {
		tx := types.MustSignNewTx(
			accounts[0].PrivateKey, signer, &types.DynamicFeeTx{
				To:        &common.Address{}, // send to the zero address
				Gas:       21_000,
				GasFeeCap: new(big.Int).Mul(gasPrice, big.NewInt(2)),
				GasTipCap: big.NewInt(1000),
			},
		)
		res = append(res, tx)
		accounts = accounts[1:]
	}

	return res
}

func createSponsoredTransactions(
	t testing.TB,
	session tests.IntegrationTestNetSession,
	numTransactions int,
) []*types.Transaction {

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Install a target contract for sponsorships.
	counterContract, receipt, err := tests.DeployContract(session, counter.DeployCounter)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Fund all calls to the counter contract.
	registry, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	ok, fundId, err := registry.ContractSponsorshipFundId(nil, receipt.ContractAddress)
	require.NoError(t, err)
	require.True(t, ok, "counter contract is not eligible for sponsorship")

	donation := big.NewInt(1e18)
	receipt, err = session.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = donation
		require.NoError(t, err)
		return registry.Sponsor(opts, fundId)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Create sponsored transactions calling the counter contract.
	accounts := tests.NewAccounts(numTransactions) // no need for endowment
	signer := types.LatestSignerForChainID(session.GetChainId())
	res := make([]*types.Transaction, 0, numTransactions)
	for _, a := range accounts {
		tx, err := counterContract.IncrementCounter(&bind.TransactOpts{
			From: a.Address(),
			Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
				require.Equal(t, a.Address(), address)
				return types.SignTx(tx, signer, a.PrivateKey)
			},
			GasPrice: big.NewInt(0), // sponsored: zero gas price
			NoSend:   true,
		})
		require.NoError(t, err)
		require.True(t, subsidies.IsSponsorshipRequest(tx))
		require.Zero(t, tx.GasPrice().Sign())
		res = append(res, tx)
	}

	return res
}

func createBundledTransactions(
	t testing.TB,
	session tests.IntegrationTestNetSession,
	numTransactions int,
) []*types.Transaction {

	// We create three types of bundled transactions:
	// - bundles where everything is accepted
	// - bundles where the some transactions are rolled back
	// - bundles with nested bundles
	// For all bundles created here, the last referenced transaction is to be
	// accepted.

	parts := numTransactions / 3
	firstPart := numTransactions - 2*parts

	accounts := tests.MakeAccountsWithBalance(t, session, 3*numTransactions, big.NewInt(1e18))
	gasPrice := getGasPrice(t, session)
	signer := types.LatestSignerForChainID(session.GetChainId())

	success := &types.AccessListTx{
		To:       &common.Address{}, // send to the zero address
		GasPrice: gasPrice,
		Gas:      21_000,
	}

	fail := &types.AccessListTx{
		To:       &common.Address{}, // send to the zero address
		GasPrice: gasPrice,
		Gas:      10_000, // below minimum gas
	}

	// Create bundles with all-succeeding transactions.
	// Each of those bundles is a AllOf(A, B, C) with three steps, each signed
	// by a different account.
	res := make([]*types.Transaction, 0, numTransactions)
	for range firstPart {
		res = append(res,
			bundle.NewBuilder().
				AllOf(
					bundle.Step(accounts[0].PrivateKey, success),
					bundle.Step(accounts[1].PrivateKey, success),
					bundle.Step(accounts[2].PrivateKey, success),
				).
				WithSigner(signer).
				Build(),
		)
		accounts = accounts[3:]
	}

	// Create bundles with some rolled back transactions.
	// Each of those bundles is a OneOf(AllOf(success, fail), success) with
	// three different signers for each step.
	for range parts {
		res = append(res,
			bundle.NewBuilder().
				OneOf(
					bundle.AllOf(
						bundle.Step(accounts[0].PrivateKey, success),
						bundle.Step(accounts[1].PrivateKey, fail),
					),
					bundle.Step(accounts[2].PrivateKey, success),
				).
				WithSigner(signer).
				Build(),
		)
		accounts = accounts[3:]
	}

	// Create nested bundles of the form
	//            AllOf(Envelope(AllOf(success)), success)
	// where the both success steps and the envelope are signed by different
	// accounts.
	for range parts {
		inner := bundle.NewBuilder().
			AllOf(bundle.Step(accounts[0].PrivateKey, success)).
			WithSigner(signer).
			Build()

		res = append(res,
			bundle.NewBuilder().
				AllOf(
					bundle.Step(accounts[2].PrivateKey, inner),
					bundle.Step(accounts[1].PrivateKey, success),
				).
				WithSigner(signer).
				Build(),
		)
		accounts = accounts[3:]
	}

	return res
}

func getGasPrice(
	t testing.TB,
	session tests.IntegrationTestNetSession,
) *big.Int {
	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(5))
	return gasPrice
}

func waitForTransactionMixToBeComplete(
	t testing.TB,
	session tests.IntegrationTestNetSession,
	txs []*types.Transaction,
) {
	// In the end, wait for all transactions to be included and successful.
	signer := types.LatestSignerForChainID(session.GetChainId())
	for _, tx := range txs {
		represent := tx
		if bundle.IsEnvelope(tx) {
			txBundle, err := bundle.OpenEnvelope(signer, tx)
			require.NoError(t, err)
			txs := txBundle.GetTransactionsInReferencedOrder()
			represent = txs[len(txs)-1]
		}
		receipt, err := session.GetReceipt(represent.Hash())
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	}
}
