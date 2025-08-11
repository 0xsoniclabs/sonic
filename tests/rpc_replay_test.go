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

package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestRpcReplay_IsConsistentWithUpgradesAtBlockHeight(t *testing.T) {
	t.Parallel()

	// This test checks the behavior of the RPC methods after an upgrade
	// when using the block number before and after the upgrade.
	//
	// This test exploits the semantic change on gas computation in Allegro
	// to identify different behavior as the result of computing the wrong ChainConfig:
	// floor data cost (EIP-7623) increases the minimum gas cost of a transaction
	// with large input buffers.

	net := StartIntegrationTestNetWithJsonGenesis(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))

	tx := SignTransaction(t, net.GetChainId(),
		SetTransactionDefaults(
			t, net,
			&types.LegacyTx{
				To:    &common.Address{0x42},
				Value: big.NewInt(1),
				// large data buffer, starting with an STOP opcode
				Data: []byte{0x0, 40_000: 0xff},
			},
			sender),
		sender)

	receiptBeforeUpgrade, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receiptBeforeUpgrade.Status)

	blockBeforeUpdate := receiptBeforeUpgrade.BlockNumber

	type rulesType struct {
		Upgrades struct{ Allegro bool }
	}
	rulesDiff := rulesType{
		Upgrades: struct{ Allegro bool }{Allegro: true},
	}
	UpdateNetworkRules(t, net, rulesDiff)
	net.AdvanceEpoch(t, 1)
	AdvanceEpochAndWaitForBlocks(t, net)

	tx2 := SignTransaction(t, net.GetChainId(),
		SetTransactionDefaults(
			t, net,
			&types.LegacyTx{
				To:    &common.Address{0x42},
				Value: big.NewInt(1),
				Nonce: 1,
				// large data buffer, starting with an STOP opcode
				Data: []byte{0x0, 40_000: 0xff},
			},
			sender),
		sender)
	receiptAfterUpgrade, err := net.Run(tx2)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receiptAfterUpgrade.Status)
	require.Greater(t, receiptAfterUpgrade.GasUsed, receiptBeforeUpgrade.GasUsed)

	lastBlockNumber := receiptAfterUpgrade.BlockNumber

	rpcTx := ethapi.TransactionArgs{
		From:     (*common.Address)(AsPointer(sender.Address())),
		To:       (*common.Address)(tx.To()),
		Gas:      AsPointer(hexutil.Uint64(tx2.Gas())),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Value:    (*hexutil.Big)(tx.Value()),
		Data:     AsPointer(hexutil.Bytes(tx2.Data())),
	}
	t.Run("eth_createAccessList", func(t *testing.T) {

		rpcTx := ethapi.TransactionArgs{
			From:     (*common.Address)(AsPointer(sender.Address())),
			To:       (*common.Address)(tx.To()),
			Gas:      AsPointer(hexutil.Uint64(tx2.Gas())),
			GasPrice: (*hexutil.Big)(tx.GasPrice()),
			Value:    (*hexutil.Big)(tx.Value()),
			Data:     AsPointer(hexutil.Bytes(tx2.Data())),
		}

		type accessListResult struct {
			AccessList *types.AccessList `json:"accessList"`
			Error      string            `json:"error,omitempty"`
			GasUsed    hexutil.Uint64    `json:"gasUsed"`
		}
		var result accessListResult
		err = client.Client().Call(&result, "eth_createAccessList", rpcTx, (*hexutil.Big)(lastBlockNumber))
		require.NoError(t, err)
		require.Equal(t, receiptAfterUpgrade.GasUsed, (uint64)(result.GasUsed),
			"access list must use the same gas as the transaction running with the correct rules")

		err = client.Client().Call(&result, "eth_createAccessList", rpcTx, (*hexutil.Big)(blockBeforeUpdate))
		require.NoError(t, err)

		require.Less(t, (uint64)(result.GasUsed), receiptAfterUpgrade.GasUsed)
		// because sonic charges a percentage of the unused gas, the gas usage from
		// the block before the upgrade cannot be reproduced after the upgrade
		require.GreaterOrEqual(t, (uint64)(result.GasUsed), receiptBeforeUpgrade.GasUsed)
	})

	t.Run("debug_traceCall", func(t *testing.T) {
		var res map[string]any

		config := ethapi.TraceCallConfig{}
		err = client.Client().Call(&res, "debug_traceCall", rpcTx, (*hexutil.Big)(lastBlockNumber), config)
		require.NoError(t, err)

		gasUsed, ok := res["gas"].(float64)
		require.True(t, ok)
		require.Equal(t, receiptAfterUpgrade.GasUsed, (uint64)(gasUsed))

		err = client.Client().Call(&res, "debug_traceCall", rpcTx, (*hexutil.Big)(blockBeforeUpdate), config)
		require.NoError(t, err)

		gasUsed, ok = res["gas"].(float64)
		require.True(t, ok)
		require.Less(t, (uint64)(gasUsed), receiptAfterUpgrade.GasUsed)
		// because sonic charges a percentage of the unused gas, the gas usage from
		// the block before the upgrade cannot be reproduced after the upgrade
		require.GreaterOrEqual(t, (uint64)(gasUsed), receiptBeforeUpgrade.GasUsed)
	})

	t.Run("trace_block", func(t *testing.T) {
		type traceResult struct {
			Hash   common.Hash `json:"transactionHash"`
			Result struct {
				GasUsed hexutil.Uint64 `json:"gasUsed"`
			}
		}
		var result []traceResult

		targetBlock := hexutil.EncodeUint64(lastBlockNumber.Uint64())
		err := client.Client().Call(&result, "trace_block", targetBlock)
		require.NoError(t, err)

		idx := slices.IndexFunc(result, func(item traceResult) bool {
			return item.Hash == receiptAfterUpgrade.TxHash
		})
		require.Greater(t, idx, -1, "transaction not found in trace results")
		require.Equal(t, receiptAfterUpgrade.GasUsed, (uint64)(result[idx].Result.GasUsed))
	})

	t.Run("debug_traceBlockBy*", func(t *testing.T) {
		type traceResult struct {
			Hash   common.Hash `json:"txHash"`           // transaction hash
			Result any         `json:"result,omitempty"` // Trace results produced by the tracer
			Error  error       `json:"error,omitempty"`  // Trace failure produced by the tracer
		}
		var result []traceResult

		tests := map[string]struct {
			method              string
			blockByNumberOrHash any
			transactionHash     common.Hash
		}{
			"hash before upgrade": {
				method:              "debug_traceBlockByHash",
				blockByNumberOrHash: receiptBeforeUpgrade.BlockHash,
				transactionHash:     receiptBeforeUpgrade.TxHash,
			},
			"hash after upgrade": {
				method:              "debug_traceBlockByHash",
				blockByNumberOrHash: receiptAfterUpgrade.BlockHash,
				transactionHash:     receiptAfterUpgrade.TxHash,
			},
			"number before upgrade": {
				method:              "debug_traceBlockByNumber",
				blockByNumberOrHash: (*hexutil.Big)(blockBeforeUpdate),
				transactionHash:     receiptBeforeUpgrade.TxHash,
			},
			"number after upgrade": {
				method:              "debug_traceBlockByNumber",
				blockByNumberOrHash: (*hexutil.Big)(lastBlockNumber),
				transactionHash:     receiptAfterUpgrade.TxHash,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {

				err := client.Client().Call(&result, test.method, test.blockByNumberOrHash)
				require.NoError(t, err)

				idx := slices.IndexFunc(result, func(item traceResult) bool {
					return item.Hash == test.transactionHash
				})
				require.Greater(t, idx, -1, "transaction not found in trace results")
				// This API call does not return gas used, best we can do is
				// to verify that the transaction was traced without errors.
				require.NoError(t, result[idx].Error)
			})
		}
	})
}
