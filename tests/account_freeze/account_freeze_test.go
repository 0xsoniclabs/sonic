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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc20"
	"github.com/0xsoniclabs/sonic/tests/contracts/payer"
	"github.com/0xsoniclabs/sonic/tests/contracts/selfdestruct"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAccountFreeze_EndToEnd(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.AccountFreeze = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	oneS := big.NewInt(1e18)
	frozen := tests.MakeAccountWithBalance(t, net, oneS)
	frozenAt := freeze(t, net, reg, frozen.Address(), true).BlockNumber

	t.Run("pool rejects frozen sender", func(t *testing.T) {
		tx := valueTransfer(t, net, frozen, common.Address{0x1})
		_, err := net.Send(tx)
		require.ErrorContains(t, err, "account is frozen")
	})

	t.Run("frozen sender is skipped in a block", func(t *testing.T) {
		tx := valueTransfer(t, net, frozen, common.Address{0x1})
		requireSkipped(t, net, tx)
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
		_, err = net.EndowAccount(common.Address{0x2}, big.NewInt(1))
		require.NoError(t, err)
		content := poolContentFrom(t, net, account.Address())
		require.Empty(t, content["queued"])
		require.Empty(t, content["pending"])
	})

	t.Run("native S sent to frozen account fails", func(t *testing.T) {
		sender := tests.MakeAccountWithBalance(t, net, oneS)
		before := balanceOf(t, net, frozen.Address())

		receipt := requireFrozenRevert(t, net, valueTransfer(t, net, sender, frozen.Address()))
		require.Equal(t, before, balanceOf(t, net, frozen.Address()))

		// The failed transaction uses the nonce and pays for its gas.
		nonce, err := client.NonceAt(t.Context(), sender.Address(), nil)
		require.NoError(t, err)
		require.Equal(t, uint64(1), nonce)
		fee := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), receipt.EffectiveGasPrice)
		require.Equal(t, new(big.Int).Sub(oneS, fee), balanceOf(t, net, sender.Address()))
	})

	t.Run("failed transaction drops its EIP-7702 authorizations", func(t *testing.T) {
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
		to := account.Address()
		receipt = requireFrozenRevert(t, net, tests.CreateTransaction(t, net, &types.SetCodeTx{
			To: to, Gas: 200_000, Data: input, AuthList: []types.SetCodeAuthorization{auth},
		}, account))

		code, err := client.CodeAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		require.Empty(t, code, "delegation rolled back")
		nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		require.Equal(t, uint64(1), nonce, "only the transaction's nonce is used")
	})

	t.Run("RPC applies the freeze of the queried block", func(t *testing.T) {
		to := frozen.Address()
		msg := ethereum.CallMsg{From: net.GetSessionSponsor().Address(), To: &to, Value: big.NewInt(1)}

		_, err := client.CallContract(t.Context(), msg, nil)
		require.ErrorContains(t, err, "account is frozen")
		_, err = client.EstimateGas(t.Context(), msg)
		require.ErrorContains(t, err, "account is frozen")

		// Before the account was frozen, the same call succeeds.
		_, err = client.CallContract(t.Context(), msg, new(big.Int).Sub(frozenAt, big.NewInt(1)))
		require.NoError(t, err)

		var blocks []struct {
			Calls []struct {
				Status hexutil.Uint64 `json:"status"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"calls"`
		}
		require.NoError(t, client.Client().Call(&blocks, "eth_simulateV1", map[string]any{
			"blockStateCalls": []any{map[string]any{"calls": []any{map[string]any{
				"from": msg.From, "to": to, "value": "0x1",
			}}}},
		}, "latest"))
		require.Len(t, blocks, 1)
		require.Len(t, blocks[0].Calls, 1)
		require.Zero(t, blocks[0].Calls[0].Status)
		require.NotNil(t, blocks[0].Calls[0].Error)
		require.Contains(t, blocks[0].Calls[0].Error.Message, "account is frozen")
	})

	t.Run("contract paying frozen account fails even if it ignores the result", func(t *testing.T) {
		contract := deployFundedPayer(t, net, 10)
		before := balanceOf(t, net, frozen.Address())

		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return contract.Pay(opts, frozen.Address(), big.NewInt(1))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
		require.Equal(t, before, balanceOf(t, net, frozen.Address()))

		// The payment itself succeeds inside the transaction; the freeze
		// reverts the transaction's outermost call afterwards.
		trace := requireTracesMatchReceipt(t, net, receipt)
		require.Equal(t, "account is frozen", trace.RevertReason)
		require.Len(t, trace.Calls, 1)
		require.Empty(t, trace.Calls[0].Error)
	})

	t.Run("frozen contract cannot pay out", func(t *testing.T) {
		contract := deployFundedPayer(t, net, 10)
		freeze(t, net, reg, contract.address, true)

		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return contract.Pay(opts, common.Address{0x1}, big.NewInt(1))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
		require.Equal(t, int64(10), balanceOf(t, net, contract.address).Int64())
	})

	t.Run("selfdestruct cannot pay frozen account", func(t *testing.T) {
		factory, receipt, err := tests.DeployContract(net, selfdestruct.DeploySelfDestructFactory)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		before := balanceOf(t, net, frozen.Address())

		sender := tests.MakeAccountWithBalance(t, net, oneS)
		opts := unsentOpts(t, net, sender)
		opts.Value = big.NewInt(1)
		tx, err := factory.CreateAndDestroy(opts, frozen.Address())
		require.NoError(t, err)
		requireFrozenRevert(t, net, tx)
		require.Equal(t, before, balanceOf(t, net, frozen.Address()))
	})

	t.Run("frozen contract cannot selfdestruct its funds away", func(t *testing.T) {
		contract, receipt, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *selfdestruct.SelfDestruct, error) {
			opts.Value = big.NewInt(1)
			return selfdestruct.DeploySelfDestruct(opts, backend, false, false, common.Address{})
		})
		require.NoError(t, err)
		address := receipt.ContractAddress
		freeze(t, net, reg, address, true)

		tx, err := contract.DestroyContract(unsentOpts(t, net, tests.MakeAccountWithBalance(t, net, oneS)), false, common.Address{0x1})
		require.NoError(t, err)
		requireFrozenRevert(t, net, tx)
		require.Equal(t, int64(1), balanceOf(t, net, address).Int64())
	})

	t.Run("tokens cannot move from or to frozen account", func(t *testing.T) {
		holder := tests.MakeAccountWithBalance(t, net, oneS)
		spender := tests.MakeAccountWithBalance(t, net, oneS)
		token, tokenAddr := deployToken(t, net)

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
			opts, err := net.GetTransactOptions(holder)
			require.NoError(t, err)
			tx, err := token.Approve(opts, approved, big.NewInt(100))
			require.NoError(t, err)
			receipt, err = net.GetReceipt(tx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		}

		freeze(t, net, reg, holder.Address(), true)

		// The frozen holder cannot receive tokens.
		tx, err := token.Transfer(unsentOpts(t, net, net.GetSessionSponsor()), holder.Address(), big.NewInt(1))
		require.NoError(t, err)
		requireFrozenRevert(t, net, tx)

		// An old allowance cannot move tokens out of the account.
		tokenABI, err := erc20.Erc20MetaData.GetAbi()
		require.NoError(t, err)
		input, err := tokenABI.Pack("transferFrom", holder.Address(), spender.Address(), big.NewInt(10))
		require.NoError(t, err)
		_, err = client.CallContract(t.Context(), ethereum.CallMsg{From: spender.Address(), To: &tokenAddr, Data: input}, nil)
		require.ErrorContains(t, err, "account is frozen")

		tx, err = token.TransferFrom(unsentOpts(t, net, spender), holder.Address(), spender.Address(), big.NewInt(10))
		require.NoError(t, err)
		requireFrozenRevert(t, net, tx)

		// A router moving the tokens fails as well: the token call succeeds
		// inside it, and the transaction reverts afterwards.
		tx, err = router.Forward(unsentOpts(t, net, spender), tokenAddr, input)
		require.NoError(t, err)
		requireFrozenRevert(t, net, tx)

		balance, err := token.BalanceOf(nil, holder.Address())
		require.NoError(t, err)
		require.Equal(t, int64(100), balance.Int64())
	})

	t.Run("EIP-7702 delegated frozen account cannot move its funds", func(t *testing.T) {
		account := tests.MakeAccountWithBalance(t, net, oneS)
		_, receipt, err := tests.DeployContract(net, payer.DeployPayer)
		require.NoError(t, err)

		nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
		require.NoError(t, err)
		auth, err := types.SignSetCode(account.PrivateKey, types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(net.GetChainId()),
			Address: receipt.ContractAddress,
			Nonce:   nonce + 1,
		})
		require.NoError(t, err)
		to := account.Address()
		receipt, err = net.Run(tests.CreateTransaction(t, net, &types.SetCodeTx{
			To: to, Gas: 100_000, AuthList: []types.SetCodeAuthorization{auth},
		}, account))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		freeze(t, net, reg, account.Address(), true)
		before := balanceOf(t, net, account.Address())
		delegated, err := payer.NewPayer(account.Address(), client)
		require.NoError(t, err)
		tx, err := delegated.Pay(unsentOpts(t, net, tests.MakeAccountWithBalance(t, net, oneS)), common.Address{0x1}, big.NewInt(1))
		require.NoError(t, err)
		requireFrozenRevert(t, net, tx)
		require.Equal(t, before, balanceOf(t, net, account.Address()))
	})

	t.Run("opted-out token and operator are not checked", func(t *testing.T) {
		token, tokenAddr := deployToken(t, net)
		optOut(t, net, reg, tokenAddr, true)
		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return token.Transfer(opts, frozen.Address(), big.NewInt(1))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		optOut(t, net, reg, tokenAddr, false)

		operator := tests.MakeAccountWithBalance(t, net, oneS)
		optOut(t, net, reg, operator.Address(), true)
		receipt, err = net.Run(valueTransfer(t, net, operator, frozen.Address()))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	})

	t.Run("unfreezing restores normal behaviour", func(t *testing.T) {
		freeze(t, net, reg, frozen.Address(), false)
		receipt, err := net.Run(valueTransfer(t, net, frozen, common.Address{0x1}))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		sender := tests.MakeAccountWithBalance(t, net, oneS)
		receipt, err = net.Run(valueTransfer(t, net, sender, frozen.Address()))
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	})
}

// deployToken deploys an ERC-20 whose supply is held by the network's
// default account.
func deployToken(t *testing.T, net *tests.IntegrationTestNet) (*erc20.Erc20, common.Address) {
	t.Helper()
	token, receipt, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *erc20.Erc20, error) {
		return erc20.DeployErc20(opts, backend, big.NewInt(1_000_000))
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return token, receipt.ContractAddress
}

func optOut(t *testing.T, net *tests.IntegrationTestNet, reg *registry.Registry, account common.Address, value bool) {
	t.Helper()
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetOptedOut(opts, account, value)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

func poolContentFrom(t *testing.T, net *tests.IntegrationTestNet, account common.Address) map[string]map[string]any {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	var content map[string]map[string]any
	require.NoError(t, client.Client().Call(&content, "txpool_contentFrom", account))
	return content
}

type fundedPayer struct {
	*payer.Payer
	address common.Address
}

// deployFundedPayer deploys a Payer holding amount wei of native S.
func deployFundedPayer(t *testing.T, net *tests.IntegrationTestNet, amount int64) fundedPayer {
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
	require.Equal(t, amount, balanceOf(t, net, address).Int64())
	return fundedPayer{contract, address}
}

// requireFrozenRevert runs tx and checks that it fails with the freeze's
// revert, and that tracing it reproduces its receipt.
func requireFrozenRevert(t *testing.T, net *tests.IntegrationTestNet, tx *types.Transaction) *types.Receipt {
	t.Helper()
	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, receipt.Status)

	trace := requireTracesMatchReceipt(t, net, receipt)
	require.Equal(t, "execution reverted", trace.Error)
	require.Equal(t, "account is frozen", trace.RevertReason)
	return receipt
}

// callFrame is a frame of the callTracer's result.
type callFrame struct {
	Error        string         `json:"error"`
	RevertReason string         `json:"revertReason"`
	GasUsed      hexutil.Uint64 `json:"gasUsed"`
	Calls        []callFrame    `json:"calls"`
}

// frames lists f and its nested frames in the order of the trace namespace.
func (f callFrame) frames() []callFrame {
	all := []callFrame{f}
	for _, c := range f.Calls {
		all = append(all, c.frames()...)
	}
	return all
}

// requireTracesMatchReceipt checks that the debug and trace namespaces
// replay the transaction of receipt as the block processor ran it, and
// returns its call trace.
func requireTracesMatchReceipt(t *testing.T, net *tests.IntegrationTestNet, receipt *types.Receipt) callFrame {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	failed := receipt.Status == types.ReceiptStatusFailed
	callTracer := map[string]any{"tracer": "callTracer"}

	var trace callFrame
	require.NoError(t, client.Client().Call(&trace, "debug_traceTransaction", receipt.TxHash, callTracer))
	require.Equal(t, failed, trace.Error != "")
	require.Equal(t, receipt.GasUsed, uint64(trace.GasUsed))

	var block []struct {
		TxHash common.Hash `json:"txHash"`
		Result callFrame   `json:"result"`
	}
	require.NoError(t, client.Client().Call(&block, "debug_traceBlockByNumber", hexutil.EncodeBig(receipt.BlockNumber), callTracer))
	require.Len(t, block, int(receipt.TransactionIndex)+1)
	require.Equal(t, receipt.TxHash, block[receipt.TransactionIndex].TxHash)
	require.Equal(t, trace, block[receipt.TransactionIndex].Result)

	var opcodes struct {
		Failed bool   `json:"failed"`
		Gas    uint64 `json:"gas"`
	}
	require.NoError(t, client.Client().Call(&opcodes, "debug_traceTransaction", receipt.TxHash, map[string]any{}))
	require.Equal(t, failed, opcodes.Failed)
	require.Equal(t, receipt.GasUsed, opcodes.Gas)

	var actions []struct {
		Error string `json:"error"`
	}
	require.NoError(t, client.Client().Call(&actions, "trace_transaction", receipt.TxHash))
	frames := trace.frames()
	require.Len(t, actions, len(frames))
	for i, action := range actions {
		require.Equal(t, frames[i].Error != "", action.Error != "", "frame %d", i)
	}
	return trace
}

func balanceOf(t *testing.T, net *tests.IntegrationTestNet, account common.Address) *big.Int {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	balance, err := client.BalanceAt(t.Context(), account, nil)
	require.NoError(t, err)
	return balance
}

func freeze(t *testing.T, net *tests.IntegrationTestNet, reg *registry.Registry, account common.Address, value bool) *types.Receipt {
	t.Helper()
	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetFrozen(opts, account, value)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	return receipt
}

func valueTransfer(t *testing.T, net *tests.IntegrationTestNet, from *tests.Account, to common.Address) *types.Transaction {
	t.Helper()
	return tests.CreateTransaction(t, net, &types.LegacyTx{To: &to, Value: big.NewInt(1), Gas: 21_000}, from)
}

// unsentOpts returns options that make a contract binding sign a transaction
// without sending it, so it can be force-emitted past the pool.
func unsentOpts(t *testing.T, net *tests.IntegrationTestNet, account *tests.Account) *bind.TransactOpts {
	t.Helper()
	opts, err := net.GetTransactOptions(account)
	require.NoError(t, err)
	opts.NoSend = true
	opts.GasLimit = 1_000_000
	return opts
}

// requireSkipped proposes tx directly to a validator, bypassing the pool, and
// checks that the block processor left it out: no receipt, no nonce bump.
func requireSkipped(t *testing.T, net *tests.IntegrationTestNet, tx *types.Transaction) {
	t.Helper()
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender, err := types.Sender(types.LatestSignerForChainID(net.GetChainId()), tx)
	require.NoError(t, err)
	nonce, err := client.NonceAt(t.Context(), sender, nil)
	require.NoError(t, err)

	_, err = net.ForceEmit(t.Context(), tx)
	require.NoError(t, err)

	// Let a few blocks pass after the proposal.
	for range 3 {
		_, err = net.EndowAccount(common.Address{0x2}, big.NewInt(1))
		require.NoError(t, err)
	}

	_, err = client.TransactionReceipt(t.Context(), tx.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)
	after, err := client.NonceAt(t.Context(), sender, nil)
	require.NoError(t, err)
	require.Equal(t, nonce, after)
}
