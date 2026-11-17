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

package tokens

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc1155"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc20"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc721"
	"github.com/0xsoniclabs/sonic/tests/contracts/payer"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPayer shows how the Payer moves funds on behalf of others: it pays
// native S it holds, and, as a router does, forwards token calls in which it
// is the caller, spending its own tokens or an approval granted to it.
func TestPayer(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	contract, deployment, err := tests.DeployContract(net, payer.DeployPayer)
	require.NoError(t, err, "deploying the payer")
	require.Equal(t, types.ReceiptStatusSuccessful, deployment.Status, "deploying the payer must succeed")
	payerAddress := deployment.ContractAddress

	// Alice holds the tokens, Bob calls the payer, Carol receives.
	accounts := tests.MakeAccountsWithBalance(t, net, 3, big.NewInt(1e17))
	alice, bob, carol := accounts[0], accounts[1], accounts[2]
	sponsor := net.GetSessionSponsor()

	t.Run("pay sends native S held by the payer", func(t *testing.T) {
		c := payerCase{session: net, payer: contract}
		check := txCheck{t: t, session: net}

		funding := c.txOptsFrom(t, sponsor)
		funding.Value = big.NewInt(10)
		check.succeeds(contract.Receive(funding))
		recipient := common.Address{0x42}
		check.succeeds(contract.Pay(c.txOptsFrom(t, bob), recipient, big.NewInt(4)))
		require.Equal(t, int64(6), nativeBalance(t, net, payerAddress))
		require.Equal(t, int64(4), nativeBalance(t, net, recipient))

		require.ErrorContains(t, check.reverts(contract.Pay(c.txOptsFrom(t, bob), recipient, big.NewInt(7))), "payment failed")
		require.Equal(t, int64(6), nativeBalance(t, net, payerAddress), "a failed payment must not move funds")
	})

	t.Run("ERC-20", func(t *testing.T) {
		token, deployment, _ := deployToken(t, net, deployErc20(1_000))
		tokenAbi, err := erc20.Erc20MetaData.GetAbi()
		require.NoError(t, err, "ABI of the token")
		c := payerCase{session: net, payer: contract, token: deployment.ContractAddress, tokenAbi: tokenAbi}
		check := txCheck{t: t, session: net}
		requireTransfer := func(receipt *types.Receipt, from, to common.Address, value int64) {
			t.Helper()
			require.Len(t, receipt.Logs, 1, "the forwarded call must emit one log")
			transfer, err := token.ParseTransfer(*receipt.Logs[0])
			require.NoError(t, err, "parsing the Transfer log")
			require.Equal(t, []any{from, to, value}, []any{transfer.From, transfer.To, transfer.Value.Int64()})
		}
		for _, holder := range []common.Address{alice.Address(), payerAddress} {
			check.succeeds(token.Transfer(c.txOptsFrom(t, sponsor), holder, big.NewInt(100)))
		}

		// The payer pays its own tokens.
		receipt := check.succeeds(c.callMethod(t, c.txOptsFrom(t, bob), "transfer", carol.Address(), big.NewInt(10)))
		requireTransfer(receipt, payerAddress, carol.Address(), 10)
		require.Equal(t, int64(90), c.balanceOf(t, payerAddress))

		// The payer spends Alice's allowance.
		check.succeeds(token.Approve(c.txOptsFrom(t, alice), payerAddress, big.NewInt(50)))
		receipt = check.succeeds(c.callMethod(t, c.txOptsFrom(t, bob), "transferFrom", alice.Address(), carol.Address(), big.NewInt(40)))
		requireTransfer(receipt, alice.Address(), carol.Address(), 40)
		require.Equal(t, int64(60), c.balanceOf(t, alice.Address()))
		require.Equal(t, int64(50), c.balanceOf(t, carol.Address()))
		allowance, err := token.Allowance(nil, alice.Address(), payerAddress)
		require.NoError(t, err, "allowance of the payer")
		require.Equal(t, int64(10), allowance.Int64())

		// Exceeding the allowance makes the token, and so forward, revert.
		err = check.reverts(c.callMethod(t, c.txOptsFrom(t, bob), "transferFrom", alice.Address(), carol.Address(), big.NewInt(11)))
		require.ErrorContains(t, err, "forward failed")
		require.Equal(t, int64(60), c.balanceOf(t, alice.Address()), "a reverted forward must not move tokens")
	})

	t.Run("ERC-721", func(t *testing.T) {
		token, deployment, _ := deployToken(t, net, deployErc721(3))
		tokenAbi, err := erc721.Erc721MetaData.GetAbi()
		require.NoError(t, err, "ABI of the token")
		c := payerCase{session: net, payer: contract, token: deployment.ContractAddress, tokenAbi: tokenAbi}
		check := txCheck{t: t, session: net}
		requireMoved := func(receipt *types.Receipt, from, to common.Address, id int64) {
			t.Helper()
			require.Len(t, receipt.Logs, 1, "the forwarded call must emit one log")
			transfer, err := token.ParseTransfer(*receipt.Logs[0])
			require.NoError(t, err, "parsing the Transfer log")
			require.Equal(t, []any{from, to, id}, []any{transfer.From, transfer.To, transfer.TokenId.Int64()})
			owner, err := token.OwnerOf(nil, big.NewInt(id))
			require.NoError(t, err, "owner of token %d", id)
			require.Equal(t, to, owner, "owner of token %d", id)
		}
		// Alice owns tokens 1 and 2, the payer token 3.
		for id, owner := range []common.Address{alice.Address(), alice.Address(), payerAddress} {
			check.succeeds(token.TransferFrom(c.txOptsFrom(t, sponsor), sponsor.Address(), owner, big.NewInt(int64(id+1))))
		}

		// The payer pays its own token.
		receipt := check.succeeds(c.callMethod(t, c.txOptsFrom(t, bob), "transferFrom", payerAddress, carol.Address(), big.NewInt(3)))
		requireMoved(receipt, payerAddress, carol.Address(), 3)

		// The payer moves a token Alice approved it for.
		check.succeeds(token.Approve(c.txOptsFrom(t, alice), payerAddress, big.NewInt(1)))
		receipt = check.succeeds(c.callMethod(t, c.txOptsFrom(t, bob), "transferFrom", alice.Address(), carol.Address(), big.NewInt(1)))
		requireMoved(receipt, alice.Address(), carol.Address(), 1)

		// Without an approval the token, and so forward, reverts.
		err = check.reverts(c.callMethod(t, c.txOptsFrom(t, bob), "transferFrom", alice.Address(), carol.Address(), big.NewInt(2)))
		require.ErrorContains(t, err, "forward failed")

		// As Alice's operator, the payer moves any of her tokens.
		check.succeeds(token.SetApprovalForAll(c.txOptsFrom(t, alice), payerAddress, true))
		receipt = check.succeeds(c.callMethod(t, c.txOptsFrom(t, bob), "transferFrom", alice.Address(), carol.Address(), big.NewInt(2)))
		requireMoved(receipt, alice.Address(), carol.Address(), 2)
	})

	t.Run("ERC-1155", func(t *testing.T) {
		token, deployment, _ := deployToken(t, net, deployErc1155([]int64{100, 100}))
		tokenAbi, err := erc1155.Erc1155MetaData.GetAbi()
		require.NoError(t, err, "ABI of the token")
		c := payerCase{session: net, payer: contract, token: deployment.ContractAddress, tokenAbi: tokenAbi}
		check := txCheck{t: t, session: net}
		id := big.NewInt(1)
		check.succeeds(token.SafeTransferFrom(c.txOptsFrom(t, sponsor), sponsor.Address(), alice.Address(), id, big.NewInt(100), nil))

		// The payer is not Alice's operator: the token, and so forward, reverts.
		err = check.reverts(c.callMethod(t, c.txOptsFrom(t, bob), "safeTransferFrom", alice.Address(), carol.Address(), id, big.NewInt(30), []byte{}))
		require.ErrorContains(t, err, "forward failed")

		// As Alice's operator, the payer moves her tokens.
		check.succeeds(token.SetApprovalForAll(c.txOptsFrom(t, alice), payerAddress, true))
		receipt := check.succeeds(c.callMethod(t, c.txOptsFrom(t, bob), "safeTransferFrom", alice.Address(), carol.Address(), id, big.NewInt(30), []byte{}))
		require.Len(t, receipt.Logs, 1, "the forwarded call must emit one log")
		transfer, err := token.ParseTransferSingle(*receipt.Logs[0])
		require.NoError(t, err, "parsing the TransferSingle log")
		require.Equal(t,
			[]any{payerAddress, alice.Address(), carol.Address(), int64(1), int64(30)},
			[]any{transfer.Operator, transfer.From, transfer.To, transfer.Id.Int64(), transfer.Value.Int64()})
		require.Equal(t, int64(70), c.balanceOf(t, alice.Address(), id))
		require.Equal(t, int64(30), c.balanceOf(t, carol.Address(), id))
	})
}

// payerCase builds the transactions of one subtest of TestPayer, forwarding
// calls through the payer to the token of the subtest.
type payerCase struct {
	session  tests.IntegrationTestNetSession
	payer    *payer.Payer
	token    common.Address
	tokenAbi *abi.ABI
}

// txOptsFrom returns the options for a transaction sent by the account. They
// use the fixed gas limit of GetTransactOptions, so no gas estimation fails on
// transactions that revert.
func (c payerCase) txOptsFrom(t testing.TB, account *tests.Account) *bind.TransactOpts {
	t.Helper()
	opts, err := c.session.GetTransactOptions(account)
	require.NoError(t, err, "transaction options of %v", account.Address())
	return opts
}

// callMethod has the payer call method of the token.
func (c payerCase) callMethod(t testing.TB, o *bind.TransactOpts, method string, args ...any) (*types.Transaction, error) {
	t.Helper()
	data, err := c.tokenAbi.Pack(method, args...)
	require.NoError(t, err, "packing %s", method)
	return c.payer.Forward(o, c.token, data)
}

// balanceOf returns the token balance of the account. ERC-1155 tokens take
// the token id as a second argument.
func (c payerCase) balanceOf(t testing.TB, account common.Address, id ...any) int64 {
	t.Helper()
	client, err := c.session.GetClient()
	require.NoError(t, err, "client")
	defer client.Close()
	var out []any
	err = bind.NewBoundContract(c.token, *c.tokenAbi, client, nil, nil).
		Call(&bind.CallOpts{Context: t.Context()}, &out, "balanceOf", append([]any{account}, id...)...)
	require.NoError(t, err, "token balance of %v", account)
	return out[0].(*big.Int).Int64()
}

// txCheck waits for the transactions of one subtest and checks their outcome.
// Its checks take the (tx, err) pair returned by a contract binding, so a call
// can be passed to them directly.
type txCheck struct {
	t       testing.TB
	session tests.IntegrationTestNetSession
}

func (c txCheck) succeeds(tx *types.Transaction, err error) *types.Receipt {
	c.t.Helper()
	receipt := c.receipt(tx, err)
	require.Equal(c.t, types.ReceiptStatusSuccessful, receipt.Status, "the transaction must succeed")
	return receipt
}

// reverts checks that the transaction reverted and returns the error of
// replaying it as a call on the state before its block, which carries the
// revert reason.
func (c txCheck) reverts(tx *types.Transaction, err error) error {
	c.t.Helper()
	receipt := c.receipt(tx, err)
	require.Equal(c.t, types.ReceiptStatusFailed, receipt.Status, "the transaction must revert")
	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	require.NoError(c.t, err, "sender of the transaction")
	client, err := c.session.GetClient()
	require.NoError(c.t, err, "client")
	defer client.Close()
	call := ethereum.CallMsg{From: sender, To: tx.To(), Gas: tx.Gas(), Value: tx.Value(), Data: tx.Data()}
	_, err = client.CallContract(c.t.Context(), call, new(big.Int).Sub(receipt.BlockNumber, big.NewInt(1)))
	require.Error(c.t, err, "replaying a reverted transaction must fail")
	return err
}

func (c txCheck) receipt(tx *types.Transaction, err error) *types.Receipt {
	c.t.Helper()
	require.NoError(c.t, err, "issuing the transaction")
	receipt, err := c.session.GetReceipt(tx.Hash())
	require.NoError(c.t, err, "receipt of the transaction")
	return receipt
}

// nativeBalance returns the native S balance of the account.
func nativeBalance(t testing.TB, session tests.IntegrationTestNetSession, account common.Address) int64 {
	t.Helper()
	client, err := session.GetClient()
	require.NoError(t, err, "client")
	defer client.Close()
	balance, err := client.BalanceAt(t.Context(), account, nil)
	require.NoError(t, err, "balance of %v", account)
	return balance.Int64()
}
