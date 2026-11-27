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

package account_freeze

import (
	"math/big"
	"testing"

	freeze_registry "github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc1155"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc20"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc721"
	"github.com/0xsoniclabs/sonic/tests/contracts/payer"
	"github.com/0xsoniclabs/sonic/tests/contracts/selfdestruct"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAccountFreeze_FrozenSendersAndFundMoves(t *testing.T) {
	upgrades := opera.GetCantoUpgrades()
	upgrades.AccountFreeze = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	reg, err := freeze_registry.NewRegistry(freeze_registry.GetAddress(), client)
	require.NoError(t, err)

	oneS := big.NewInt(1e18)
	frozen := tests.MakeAccountWithBalance(t, net, oneS)
	freeze(t, net, reg, frozen.Address(), true)

	t.Run("pool rejects frozen sender", func(t *testing.T) {
		tx := valueTransfer(t, net, frozen, common.Address{0x1})
		_, err := net.Send(tx)
		require.ErrorContains(t, err, "account is frozen")
	})

	t.Run("frozen sender is skipped in a block", func(t *testing.T) {
		tx := valueTransfer(t, net, frozen, common.Address{0x1})
		nonce, err := client.NonceAt(t.Context(), frozen.Address(), nil)
		require.NoError(t, err)

		_, err = net.ForceEmit(t.Context(), tx)
		require.NoError(t, err)

		waitBlocks(t, net, 3)

		_, err = client.TransactionReceipt(t.Context(), tx.Hash())
		require.ErrorIs(t, err, ethereum.NotFound)
		after, err := client.NonceAt(t.Context(), frozen.Address(), nil)
		require.NoError(t, err)
		require.Equal(t, nonce, after)
	})

	t.Run("pool drops pending transactions of a newly frozen sender", func(t *testing.T) {
		account := tests.MakeAccountWithBalance(t, net, oneS)
		nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		to := common.Address{0x1}
		gapped := tests.CreateTransaction(t, net, &types.LegacyTx{Nonce: nonce + 1, To: &to, Value: big.NewInt(1), Gas: 21_000}, account)
		_, err = net.Send(gapped)
		require.NoError(t, err)
		require.Len(t, poolContentFrom(t, net, account.Address())["queued"], 1)

		freeze(t, net, reg, account.Address(), true)
		content := poolContentFrom(t, net, account.Address())
		require.Empty(t, content["queued"])
		require.Empty(t, content["pending"])
	})

	t.Run("native S sent to frozen account fails", func(t *testing.T) {
		sender := tests.MakeAccountWithBalance(t, net, oneS)
		before := balanceOf(t, client, frozen.Address())

		receipt, err := net.Run(valueTransfer(t, net, sender, frozen.Address()))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
		fee := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), receipt.EffectiveGasPrice)
		require.Equal(t, new(big.Int).Sub(oneS, fee), balanceOf(t, client, sender.Address()))

		require.Equal(t, before, balanceOf(t, client, frozen.Address()))
		nonce, err := client.NonceAt(t.Context(), sender.Address(), nil)
		require.NoError(t, err)
		require.Equal(t, uint64(1), nonce)
	})

	t.Run("transaction moving frozen funds drops its EIP-7702 authorizations", func(t *testing.T) {
		account := tests.MakeAccountWithBalance(t, net, oneS)
		_, receipt, err := tests.DeployContract(net, payer.DeployPayer)
		require.NoError(t, err)
		payerABI, err := payer.PayerMetaData.GetAbi()
		require.NoError(t, err)
		input, err := payerABI.Pack("pay", frozen.Address(), big.NewInt(1))
		require.NoError(t, err)
		// The account delegates to the payer and, in the same transaction,
		// pays the frozen account through the delegated code.
		auth, err := types.SignSetCode(account.PrivateKey, types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(net.GetChainId()),
			Address: receipt.ContractAddress,
			Nonce:   1,
		})
		require.NoError(t, err)

		receipt, err = net.Run(tests.CreateTransaction(t, net, &types.SetCodeTx{
			To: account.Address(), Gas: 200_000, Data: input, AuthList: []types.SetCodeAuthorization{auth},
		}, account))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		code, err := client.CodeAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		require.Empty(t, code, "delegation rolled back")
		nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		require.Equal(t, uint64(1), nonce, "only the transaction's nonce is used")
	})

	t.Run("contract paying frozen account fails", func(t *testing.T) {
		cases := map[string]struct {
			pay func(*payer.Payer, *bind.TransactOpts, common.Address, *big.Int) (*types.Transaction, error)
		}{
			"checking the result": {pay: (*payer.Payer).Pay},
			"ignoring the result": {pay: (*payer.Payer).PayIgnoringResult},
		}
		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				contract, _ := deployFundedPayer(t, net, 10)
				before := balanceOf(t, client, frozen.Address())

				tx, err := c.pay(contract, unsentOpts(t, net, net.GetSessionSponsor()), frozen.Address(), big.NewInt(1))
				require.NoError(t, err)
				receipt, err := net.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

				require.Equal(t, before, balanceOf(t, client, frozen.Address()))
			})
		}
	})

	t.Run("contract paying zero to frozen account succeeds", func(t *testing.T) {
		contract, _, err := tests.DeployContract(net, payer.DeployPayer)
		require.NoError(t, err)

		tx, err := contract.Pay(unsentOpts(t, net, net.GetSessionSponsor()), frozen.Address(), big.NewInt(0))
		require.NoError(t, err)
		receipt, err := net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	})

	t.Run("payment undone by a reverted call does not count", func(t *testing.T) {
		contract, address := deployFundedPayer(t, net, 10)
		// The token has no receive function, so a call paying it reverts
		// after the payment.
		_, rejecting := deployToken(t, net)
		freeze(t, net, reg, rejecting, true)

		tx, err := contract.PayIgnoringResult(unsentOpts(t, net, net.GetSessionSponsor()), rejecting, big.NewInt(1))
		require.NoError(t, err)
		receipt, err := net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		require.Equal(t, int64(10), balanceOf(t, client, address).Int64())
	})

	t.Run("selfdestruct cannot pay frozen account", func(t *testing.T) {
		factory, receipt, err := tests.DeployContract(net, selfdestruct.DeploySelfDestructFactory)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		before := balanceOf(t, client, frozen.Address())

		opts := unsentOpts(t, net, net.GetSessionSponsor())
		opts.Value = big.NewInt(1)
		tx, err := factory.CreateAndDestroy(opts, frozen.Address())
		require.NoError(t, err)
		receipt, err = net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		require.Equal(t, before, balanceOf(t, client, frozen.Address()))
	})

	t.Run("contract creation paying frozen account fails", func(t *testing.T) {
		before := balanceOf(t, client, frozen.Address())

		_, receipt, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
			opts.Value = big.NewInt(1)
			return selfdestruct.DeploySelfDestruct(opts, backend, true, false, frozen.Address())
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		require.Equal(t, before, balanceOf(t, client, frozen.Address()))
	})

	t.Run("frozen contract cannot pay out", func(t *testing.T) {
		contract, address := deployFundedPayer(t, net, 10)
		freeze(t, net, reg, address, true)

		tx, err := contract.Pay(unsentOpts(t, net, net.GetSessionSponsor()), common.Address{0x1}, big.NewInt(1))
		require.NoError(t, err)
		receipt, err := net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		require.Equal(t, int64(10), balanceOf(t, client, address).Int64())
	})

	t.Run("frozen contract cannot selfdestruct its funds away", func(t *testing.T) {
		contract, receipt, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
			opts.Value = big.NewInt(1)
			return selfdestruct.DeploySelfDestruct(opts, backend, false, false, common.Address{})
		})
		require.NoError(t, err)
		address := receipt.ContractAddress
		freeze(t, net, reg, address, true)

		tx, err := contract.DestroyContract(unsentOpts(t, net, net.GetSessionSponsor()), false, common.Address{0x1})
		require.NoError(t, err)
		receipt, err = net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		require.Equal(t, int64(1), balanceOf(t, client, address).Int64())
	})

	t.Run("EIP-7702 delegated frozen account cannot move its funds", func(t *testing.T) {
		account := tests.MakeAccountWithBalance(t, net, oneS)
		_, receipt, err := tests.DeployContract(net, payer.DeployPayer)
		require.NoError(t, err)
		auth, err := types.SignSetCode(account.PrivateKey, types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(net.GetChainId()),
			Address: receipt.ContractAddress,
			Nonce:   1,
		})
		require.NoError(t, err)
		receipt, err = net.Run(tests.CreateTransaction(t, net, &types.SetCodeTx{
			To: account.Address(), Gas: 100_000, AuthList: []types.SetCodeAuthorization{auth},
		}, account))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		freeze(t, net, reg, account.Address(), true)
		before := balanceOf(t, client, account.Address())
		delegated, err := payer.NewPayer(account.Address(), client)
		require.NoError(t, err)

		tx, err := delegated.Pay(unsentOpts(t, net, net.GetSessionSponsor()), common.Address{0x1}, big.NewInt(1))
		require.NoError(t, err)
		receipt, err = net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		require.Equal(t, before, balanceOf(t, client, account.Address()))
	})

	t.Run("tokens cannot move to frozen account", func(t *testing.T) {
		sponsor := net.GetSessionSponsor().Address()
		token, _ := deployToken(t, net)
		nft, _, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *erc721.Erc721, error) {
			return erc721.DeployErc721(opts, backend, big.NewInt(1))
		})
		require.NoError(t, err)
		multi, _, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *erc1155.Erc1155, error) {
			return erc1155.DeployErc1155(opts, backend, []*big.Int{big.NewInt(1)}, []*big.Int{big.NewInt(10)})
		})
		require.NoError(t, err)
		one := []*big.Int{big.NewInt(1)}

		cases := map[string]struct {
			transfer func(*bind.TransactOpts) (*types.Transaction, error)
		}{
			"erc-20": {transfer: func(opts *bind.TransactOpts) (*types.Transaction, error) {
				return token.Transfer(opts, frozen.Address(), big.NewInt(1))
			}},
			"erc-721": {transfer: func(opts *bind.TransactOpts) (*types.Transaction, error) {
				return nft.TransferFrom(opts, sponsor, frozen.Address(), big.NewInt(1))
			}},
			"erc-1155 single": {transfer: func(opts *bind.TransactOpts) (*types.Transaction, error) {
				return multi.SafeTransferFrom(opts, sponsor, frozen.Address(), big.NewInt(1), big.NewInt(1), nil)
			}},
			"erc-1155 batch": {transfer: func(opts *bind.TransactOpts) (*types.Transaction, error) {
				return multi.SafeBatchTransferFrom(opts, sponsor, frozen.Address(), one, one, nil)
			}},
		}
		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				tx, err := c.transfer(unsentOpts(t, net, net.GetSessionSponsor()))
				require.NoError(t, err)
				receipt, err := net.Run(tx)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
			})
		}
	})

	t.Run("old allowance cannot move tokens out of frozen account", func(t *testing.T) {
		holder := tests.MakeAccountWithBalance(t, net, oneS)
		spender := tests.MakeAccountWithBalance(t, net, oneS)
		token, tokenAddr := deployToken(t, net)
		tokenABI, err := erc20.Erc20MetaData.GetAbi()
		require.NoError(t, err)
		input, err := tokenABI.Pack("transferFrom", holder.Address(), spender.Address(), big.NewInt(10))
		require.NoError(t, err)
		// Before the freeze the holder receives tokens and grants allowances,
		// to the spender and to a router contract.
		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return token.Transfer(opts, holder.Address(), big.NewInt(100))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		router, receipt, err := tests.DeployContract(net, payer.DeployPayer)
		require.NoError(t, err)
		for _, approved := range []common.Address{spender.Address(), receipt.ContractAddress} {
			tx, err := token.Approve(unsentOpts(t, net, holder), approved, big.NewInt(100))
			require.NoError(t, err)
			receipt, err = net.Run(tx)
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		}
		freeze(t, net, reg, holder.Address(), true)

		// An old allowance cannot move tokens out of the account.
		tx, err := token.TransferFrom(unsentOpts(t, net, spender), holder.Address(), spender.Address(), big.NewInt(10))
		require.NoError(t, err)
		receipt, err = net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		// A router moving the tokens fails as well.
		tx, err = router.Forward(unsentOpts(t, net, spender), tokenAddr, input)
		require.NoError(t, err)
		receipt, err = net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

		balance, err := token.BalanceOf(nil, holder.Address())
		require.NoError(t, err)
		require.Equal(t, int64(100), balance.Int64())
	})

	t.Run("opted-out token is not checked", func(t *testing.T) {
		token, tokenAddr := deployToken(t, net)
		optOut(t, net, reg, tokenAddr)

		tx, err := token.Transfer(unsentOpts(t, net, net.GetSessionSponsor()), frozen.Address(), big.NewInt(1))
		require.NoError(t, err)
		receipt, err := net.Run(tx)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	})

	t.Run("unfreezing restores normal behaviour", func(t *testing.T) {
		sender := tests.MakeAccountWithBalance(t, net, oneS)
		freeze(t, net, reg, frozen.Address(), false)

		receipt, err := net.Run(valueTransfer(t, net, frozen, common.Address{0x1}))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		receipt, err = net.Run(valueTransfer(t, net, sender, frozen.Address()))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	})
}

// deployToken deploys an ERC-20 whose supply is held by the session sponsor.
func deployToken(t *testing.T, net *tests.IntegrationTestNet) (*erc20.Erc20, common.Address) {
	t.Helper()
	token, receipt, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *erc20.Erc20, error) {
		return erc20.DeployErc20(opts, backend, big.NewInt(1_000_000))
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return token, receipt.ContractAddress
}

// deployFundedPayer deploys a Payer holding amount wei of native S.
func deployFundedPayer(t *testing.T, net *tests.IntegrationTestNet, amount int64) (*payer.Payer, common.Address) {
	t.Helper()
	contract, receipt, err := tests.DeployContract(net, payer.DeployPayer)
	require.NoError(t, err)
	address := receipt.ContractAddress
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(amount)
		return contract.Receive(opts)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return contract, address
}

// poolContentFrom returns the pending and queued pool transactions of account.
func poolContentFrom(t *testing.T, net *tests.IntegrationTestNet, account common.Address) map[string]map[string]any {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	var content map[string]map[string]any
	require.NoError(t, client.Client().Call(&content, "txpool_contentFrom", account))
	return content
}

// freeze sets the freeze status of account and waits until the pool has seen it.
func freeze(t *testing.T, net *tests.IntegrationTestNet, reg *freeze_registry.Registry, account common.Address, value bool) *types.Receipt {
	t.Helper()
	var tx *types.Transaction
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		var err error
		tx, err = reg.SetFrozen(opts, account, value)
		return tx, err
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	require.NoError(t, tests.WaitUntilTransactionIsRetiredFromPool(t, client, tx))
	return receipt
}

// optOut exempts the token contract at account from the freeze.
func optOut(t *testing.T, net *tests.IntegrationTestNet, reg *freeze_registry.Registry, account common.Address) {
	t.Helper()
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetOptedOut(opts, account, true)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

// unsentOpts returns options that make a contract binding sign but not send.
func unsentOpts(t *testing.T, net *tests.IntegrationTestNet, account *tests.Account) *bind.TransactOpts {
	t.Helper()
	opts, err := net.GetTransactOptions(account)
	require.NoError(t, err)
	opts.NoSend = true
	return opts
}

// valueTransfer signs a transfer of 1 wei from `from` to `to`.
func valueTransfer(t *testing.T, net *tests.IntegrationTestNet, from *tests.Account, to common.Address) *types.Transaction {
	t.Helper()
	return tests.CreateTransaction(t, net, &types.LegacyTx{To: &to, Value: big.NewInt(1), Gas: 21_000}, from)
}

// waitBlocks lets n blocks pass by sending a transaction for each.
func waitBlocks(t *testing.T, net *tests.IntegrationTestNet, n int) {
	t.Helper()
	for range n {
		_, err := net.EndowAccount(common.Address{0x2}, big.NewInt(1))
		require.NoError(t, err)
	}
}

// balanceOf returns the native S balance of account.
func balanceOf(t *testing.T, client *tests.PooledEthClient, account common.Address) *big.Int {
	t.Helper()
	balance, err := client.BalanceAt(t.Context(), account, nil)
	require.NoError(t, err)
	return balance
}
