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
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strings"
	"testing"

	freeze_registry "github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc20"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestAccountFreeze_FrozenSenders(t *testing.T) {
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
}

func TestAccountFreeze_RPCReportsTheFreezeAsReason(t *testing.T) {
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
	sender := tests.MakeAccountWithBalance(t, net, oneS)
	frozen := tests.MakeAccountWithBalance(t, net, oneS)
	freeze(t, net, reg, frozen.Address(), true)
	second := tests.MakeAccountWithBalance(t, net, oneS)

	token, deployed, err := tests.DeployContract(net, func(opts *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *erc20.Erc20, error) {
		return erc20.DeployErc20(opts, backend, big.NewInt(2))
	})
	require.NoError(t, err)
	tokenAddress := deployed.ContractAddress
	for _, holder := range []*tests.Account{sender, second} {
		receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return token.Transfer(opts, holder.Address(), big.NewInt(1))
		})
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	}
	tokenAbi, err := erc20.Erc20MetaData.GetAbi()
	require.NoError(t, err)
	tokenToFrozen, err := tokenAbi.Pack("transfer", frozen.Address(), big.NewInt(1))
	require.NoError(t, err)

	stringType, err := abi.NewType("string", "", nil)
	require.NoError(t, err)
	reason, err := abi.Arguments{{Type: stringType}}.Pack("account is frozen")
	require.NoError(t, err)
	frozenRevertData := hexutil.Encode(append(crypto.Keccak256([]byte("Error(string)"))[:4], reason...))

	callTracer := map[string]any{"tracer": "callTracer"}
	callTracerReason := map[string]any{"error": "execution reverted", "revertReason": "account is frozen"}
	structLoggerReason := map[string]any{"failed": true, "returnValue": frozenRevertData}
	parityReason := map[string]any{"error": "account is frozen"}
	simulate := func(call map[string]any) map[string]any {
		return map[string]any{"blockStateCalls": []any{map[string]any{"calls": []any{call}}}}
	}

	// replayPath and replayWant locate the frozen account's balance in the
	// prestate of a transaction, which the replay of an earlier move must not
	// have changed
	moves := map[string]struct {
		to         common.Address
		value      *big.Int
		data       []byte
		replayPath []any
		replayWant map[string]any
	}{
		"native transfer": {
			to:         frozen.Address(),
			value:      big.NewInt(1),
			replayPath: []any{strings.ToLower(frozen.Address().Hex())},
			replayWant: map[string]any{"balance": hexutil.EncodeBig(oneS)},
		},
		"ERC-20 transfer": {
			to:         tokenAddress,
			data:       tokenToFrozen,
			replayPath: []any{strings.ToLower(tokenAddress.Hex()), "storage"},
			replayWant: map[string]any{crypto.Keccak256Hash(common.LeftPadBytes(frozen.Address().Bytes(), 32), common.Hash{}.Bytes()).Hex(): common.Hash{}.Hex()},
		},
	}
	for name, move := range moves {
		t.Run(name, func(t *testing.T) {
			// tracing the later of two frozen moves in a block replays the earlier one
			moveFrom := func(from *tests.Account) *types.Transaction {
				return tests.CreateTransaction(t, net, &types.LegacyTx{To: &move.to, Value: move.value, Data: move.data, Gas: 200_000}, from)
			}
			hashes, err := net.ForceEmitAll(t.Context(), []*types.Transaction{moveFrom(sender), moveFrom(second)})
			require.NoError(t, err)
			receipts, err := net.GetReceipts(hashes)
			require.NoError(t, err)
			require.Equal(t, receipts[0].BlockNumber, receipts[1].BlockNumber)
			require.Equal(t, []uint64{types.ReceiptStatusFailed, types.ReceiptStatusFailed}, []uint64{receipts[0].Status, receipts[1].Status})
			receipt := slices.MaxFunc(receipts, func(a, b *types.Receipt) int { return cmp.Compare(a.TransactionIndex, b.TransactionIndex) })

			toFrozen := map[string]any{"from": sender.Address(), "to": move.to, "value": (*hexutil.Big)(move.value), "data": hexutil.Bytes(move.data)}
			fromFrozen := maps.Clone(toFrozen)
			fromFrozen["from"] = frozen.Address()
			withGas := maps.Clone(toFrozen)
			withGas["gas"] = hexutil.Uint64(200_000)
			fromFrozenWithGas := maps.Clone(withGas)
			fromFrozenWithGas["from"] = frozen.Address()
			tx, number := receipt.TxHash, hexutil.EncodeBig(receipt.BlockNumber)

			t.Run("results carry the reason", func(t *testing.T) {
				cases := map[string]struct {
					method string
					params []any
					path   []any
					want   map[string]any
				}{
					"eth_simulateV1": {
						method: "eth_simulateV1",
						params: []any{simulate(toFrozen), "latest"},
						path:   []any{0, "calls", 0, "error"},
						want:   map[string]any{"code": -32000.0, "message": "execution reverted: account is frozen", "data": frozenRevertData},
					},
					"eth_createAccessList with gas": {
						method: "eth_createAccessList",
						params: []any{withGas, "latest"},
						want:   map[string]any{"error": "execution reverted: account is frozen"},
					},
					"debug_traceCall with callTracer": {
						method: "debug_traceCall",
						params: []any{toFrozen, "latest", callTracer},
						want:   callTracerReason,
					},
					"debug_traceCall with struct logger": {
						method: "debug_traceCall",
						params: []any{toFrozen, "latest", map[string]any{}},
						want:   structLoggerReason,
					},
					"trace_call": {
						method: "trace_call",
						params: []any{toFrozen, []string{"trace"}, "latest"},
						path:   []any{"trace", 0},
						want:   parityReason,
					},
					"trace_callMany": {
						method: "trace_callMany",
						params: []any{[]any{[]any{toFrozen, []string{"trace"}}}, "latest"},
						path:   []any{0, "trace", 0},
						want:   parityReason,
					},
					"debug_traceTransaction with callTracer": {
						method: "debug_traceTransaction",
						params: []any{tx, callTracer},
						want:   callTracerReason,
					},
					"debug_traceTransaction with struct logger": {
						method: "debug_traceTransaction",
						params: []any{tx, map[string]any{}},
						want:   structLoggerReason,
					},
					"debug_traceTransaction after the replay of an earlier frozen move": {
						method: "debug_traceTransaction",
						params: []any{tx, map[string]any{"tracer": "prestateTracer"}},
						path:   move.replayPath,
						want:   move.replayWant,
					},
					"debug_traceBlockByNumber with callTracer": {
						method: "debug_traceBlockByNumber",
						params: []any{number, callTracer},
						path:   []any{map[string]any{"txHash": tx.Hex()}, "result"},
						want:   callTracerReason,
					},
					"debug_traceBlockByNumber with struct logger": {
						method: "debug_traceBlockByNumber",
						params: []any{number, map[string]any{}},
						path:   []any{map[string]any{"txHash": tx.Hex()}, "result"},
						want:   structLoggerReason,
					},
					"debug_traceBlockByHash with callTracer": {
						method: "debug_traceBlockByHash",
						params: []any{receipt.BlockHash, callTracer},
						path:   []any{map[string]any{"txHash": tx.Hex()}, "result"},
						want:   callTracerReason,
					},
					"debug_traceBlockByHash with struct logger": {
						method: "debug_traceBlockByHash",
						params: []any{receipt.BlockHash, map[string]any{}},
						path:   []any{map[string]any{"txHash": tx.Hex()}, "result"},
						want:   structLoggerReason,
					},
					"trace_transaction": {
						method: "trace_transaction",
						params: []any{tx},
						path:   []any{0},
						want:   parityReason,
					},
					"trace_block": {
						method: "trace_block",
						params: []any{number},
						path:   []any{map[string]any{"transactionHash": tx.Hex()}},
						want:   parityReason,
					},
				}
				for name, test := range cases {
					t.Run(name, func(t *testing.T) {
						var result any
						require.NoError(t, client.Client().Call(&result, test.method, test.params...))
						for _, step := range test.path {
							switch step := step.(type) {
							case string:
								result = result.(map[string]any)[step]
							case int:
								result = result.([]any)[step]
							case map[string]any:
								i := slices.IndexFunc(result.([]any), func(e any) bool {
									return maps.Equal(step, fieldsOf(e.(map[string]any), step))
								})
								require.NotEqual(t, -1, i)
								result = result.([]any)[i]
							}
						}
						require.Equal(t, test.want, fieldsOf(result.(map[string]any), test.want))
					})
				}
			})

			t.Run("errors carry the reason", func(t *testing.T) {
				cases := map[string]struct {
					method   string
					params   []any
					wantCode int
					wantErr  string
					wantData any
				}{
					"eth_call moving frozen funds":        {method: "eth_call", params: []any{toFrozen, "latest"}, wantCode: 3, wantErr: "execution reverted: account is frozen", wantData: frozenRevertData},
					"eth_estimateGas moving frozen funds": {method: "eth_estimateGas", params: []any{toFrozen, "latest"}, wantCode: 3, wantErr: "execution reverted: account is frozen", wantData: frozenRevertData},
					"eth_createAccessList without gas":    {method: "eth_createAccessList", params: []any{toFrozen, "latest"}, wantCode: 3, wantErr: "execution reverted: account is frozen", wantData: frozenRevertData},
					"eth_call from frozen":                {method: "eth_call", params: []any{fromFrozen, "latest"}, wantCode: -32000, wantErr: "account is frozen"},
					"eth_estimateGas from frozen":         {method: "eth_estimateGas", params: []any{fromFrozen, "latest"}, wantCode: -32000, wantErr: "account is frozen"},
					"eth_simulateV1 from frozen":          {method: "eth_simulateV1", params: []any{simulate(fromFrozen), "latest"}, wantCode: -32602, wantErr: "account is frozen"},
					"eth_createAccessList from frozen":    {method: "eth_createAccessList", params: []any{fromFrozenWithGas, "latest"}, wantCode: -32000, wantErr: "account is frozen"},
					"debug_traceCall from frozen":         {method: "debug_traceCall", params: []any{fromFrozen, "latest", callTracer}, wantCode: -32000, wantErr: "account is frozen"},
					"trace_call from frozen":              {method: "trace_call", params: []any{fromFrozen, []string{"trace"}, "latest"}, wantCode: -32000, wantErr: "account is frozen"},
					"trace_callMany from frozen":          {method: "trace_callMany", params: []any{[]any{[]any{fromFrozen, []string{"trace"}}}, "latest"}, wantCode: -32000, wantErr: "account is frozen"},
				}
				for name, test := range cases {
					t.Run(name, func(t *testing.T) {
						var result any
						err := client.Client().Call(&result, test.method, test.params...)
						require.ErrorContains(t, err, test.wantErr)
						var rpcErr interface {
							rpc.Error
							rpc.DataError
						}
						require.ErrorAs(t, err, &rpcErr)
						require.Equal(t, test.wantCode, rpcErr.ErrorCode())
						require.Equal(t, test.wantData, rpcErr.ErrorData())
					})
				}
			})
		})
	}

	t.Run("eth_createAccessList reports the reason of other reverts", func(t *testing.T) {
		overdraw, err := tokenAbi.Pack("transfer", common.Address{0x1}, big.NewInt(2))
		require.NoError(t, err)
		var result map[string]any
		require.NoError(t, client.Client().Call(&result, "eth_createAccessList", map[string]any{"from": sender.Address(), "to": tokenAddress, "data": hexutil.Bytes(overdraw), "gas": hexutil.Uint64(200_000)}, "latest"))
		require.Equal(t, "execution reverted: arithmetic underflow or overflow", result["error"])
	})
}

func TestAccountFreeze_RPCAppliesTheFreezeOnlyWhileEnabled(t *testing.T) {
	sender, frozen := tests.NewAccount().Address(), tests.NewAccount().Address()
	oneS := hexutil.EncodeBig(big.NewInt(1e18))
	overrides := map[string]any{
		freeze_registry.GetAddress().Hex(): map[string]any{
			"nonce":     "0x1", // keeps the registry from being empty while the feature is disabled
			"stateDiff": map[string]any{freeze_registry.GetFrozenSlot(frozen).Hex(): common.BytesToHash([]byte{1}).Hex()},
		},
		sender.Hex(): map[string]any{"balance": oneS},
		frozen.Hex(): map[string]any{"balance": oneS},
	}
	toFrozen := map[string]any{"from": sender, "to": frozen, "value": "0x1"}
	fromFrozen := map[string]any{"from": frozen, "to": common.Address{0x1}, "value": "0x1"}
	cases := map[string]struct {
		method string
		params []any
	}{
		"eth_call moving frozen funds":        {method: "eth_call", params: []any{toFrozen, "latest", overrides}},
		"eth_call from frozen":                {method: "eth_call", params: []any{fromFrozen, "latest", overrides}},
		"eth_estimateGas moving frozen funds": {method: "eth_estimateGas", params: []any{toFrozen, "latest", overrides}},
		"eth_estimateGas from frozen":         {method: "eth_estimateGas", params: []any{fromFrozen, "latest", overrides}},
		"eth_simulateV1":                      {method: "eth_simulateV1", params: []any{map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": overrides, "calls": []any{toFrozen}}}}, "latest"}},
		"eth_simulateV1 from frozen":          {method: "eth_simulateV1", params: []any{map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": overrides, "calls": []any{fromFrozen}}}}, "latest"}},
		"debug_traceCall":                     {method: "debug_traceCall", params: []any{toFrozen, "latest", map[string]any{"tracer": "callTracer", "stateOverrides": overrides}}},
		"trace_call":                          {method: "trace_call", params: []any{toFrozen, []string{"trace"}, "latest", map[string]any{"StateOverrides": overrides}}},
		"trace_callMany":                      {method: "trace_callMany", params: []any{[]any{[]any{toFrozen, []string{"trace"}}}, "latest", map[string]any{"StateOverrides": overrides}}},
	}
	for name, enabled := range map[string]bool{"enabled": true, "disabled": false} {
		t.Run(name, func(t *testing.T) {
			upgrades := opera.GetCantoUpgrades()
			upgrades.AccountFreeze = enabled
			net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrades,
			})
			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			for name, test := range cases {
				t.Run(name, func(t *testing.T) {
					var result json.RawMessage
					err := client.Client().Call(&result, test.method, test.params...)
					if !enabled {
						require.NoError(t, err)
					}
					require.Equal(t, enabled, strings.Contains(fmt.Sprint(err)+string(result), "account is frozen"))
				})
			}
		})
	}
}

// fieldsOf returns the fields of m named by the keys of names.
func fieldsOf(m map[string]any, names map[string]any) map[string]any {
	fields := map[string]any{}
	for name := range names {
		fields[name] = m[name]
	}
	return fields
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

func valueTransfer(t *testing.T, net *tests.IntegrationTestNet, from *tests.Account, to common.Address) *types.Transaction {
	t.Helper()
	return tests.CreateTransaction(t, net, &types.LegacyTx{To: &to, Value: big.NewInt(1), Gas: 21_000}, from)
}

func waitBlocks(t *testing.T, net *tests.IntegrationTestNet, n int) {
	t.Helper()
	for range n {
		_, err := net.EndowAccount(common.Address{0x2}, big.NewInt(1))
		require.NoError(t, err)
	}
}
