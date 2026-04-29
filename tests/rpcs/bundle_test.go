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

package rpcs

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/bundles"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// bundleTestFixture holds shared state for bundle RPC tests.
type bundleTestFixture struct {
	client                 *tests.PooledEhtClient
	sender                 *tests.Account
	chainID                *big.Int
	counterContractAddress common.Address
	incrementData          []byte
	nonce                  uint64
}

// TestBundleRPCFunctions_UsingTypes tests the sonic_prepareBundle →
// sonic_submitBundle → sonic_getBundleInfo flow using typed sonicapi/ethapi
// structs to build the proposal and decode the response.
func TestBundleRPCFunctions_UsingTypes(t *testing.T) {
	const incrementCount = 3
	f := newBundleTestFixture(t)
	require := require.New(t)

	steps := make([]any, incrementCount)
	for i := range incrementCount {
		n := hexutil.Uint64(f.nonce + uint64(i))
		data := hexutil.Bytes(f.incrementData)
		from := f.sender.Address()
		to := f.counterContractAddress
		steps[i] = sonicapi.RPCExecutionStepProposal{
			TransactionArgs: ethapi.TransactionArgs{
				From:    &from,
				To:      &to,
				Nonce:   &n,
				Data:    &data,
				ChainID: (*hexutil.Big)(f.chainID),
			},
		}
	}

	proposal := sonicapi.RPCExecutionProposal{
		RPCExecutionPlanGroup: sonicapi.RPCExecutionPlanGroup{
			Steps: steps,
		},
	}

	var prepared sonicapi.RPCPreparedBundle
	err := f.client.Client().CallContext(t.Context(), &prepared, "sonic_prepareBundle", proposal)
	require.NoError(err)
	require.Len(prepared.Transactions, incrementCount)

	signer := types.LatestSignerForChainID(f.chainID)
	signedTxs := make([]hexutil.Bytes, incrementCount)
	for i := range incrementCount {
		tx := prepared.Transactions[i].ToTransaction()
		signedTx, err := types.SignTx(tx, signer, f.sender.PrivateKey)
		require.NoError(err)
		encoded, err := signedTx.MarshalBinary()
		require.NoError(err)
		signedTxs[i] = encoded
	}

	var planHash common.Hash
	err = f.client.Client().CallContext(
		t.Context(),
		&planHash,
		"sonic_submitBundle",
		sonicapi.SubmitBundleArgs{
			SignedTransactions: signedTxs,
			ExecutionPlan:      prepared.ExecutionPlan,
		},
	)
	require.NoError(err)

	info, err := bundles.WaitForBundleExecution(t.Context(), f.client.Client(), planHash)
	require.NoError(err)
	require.NotNil(info)
	require.EqualValues(incrementCount, info.Count)
	require.Greater(uint64(info.Block), uint64(0))

	f.verifyCounterValue(t, incrementCount)
}

// TestBundleRPCFunctionsGeneric exercises the same flow as
// TestBundleRPCFunctions_UsingTypes but communicates with the node exclusively
// through generic map/interface{} values, with no dependency on the sonicapi
// or ethapi packages.
func TestBundleRPCFunctionsGeneric(t *testing.T) {
	const incrementCount = 3
	f := newBundleTestFixture(t)
	require := require.New(t)

	steps := make([]interface{}, incrementCount)
	for i := range incrementCount {
		steps[i] = map[string]interface{}{
			"from":    f.sender.Address(),
			"to":      f.counterContractAddress,
			"nonce":   hexutil.Uint64(f.nonce + uint64(i)),
			"data":    hexutil.Bytes(f.incrementData),
			"chainId": (*hexutil.Big)(f.chainID),
		}
	}
	proposal := map[string]interface{}{"steps": steps}

	// Keep execution plan as raw JSON so it can be forwarded unchanged.
	var prepared struct {
		Transactions  []json.RawMessage `json:"transactions"`
		ExecutionPlan json.RawMessage   `json:"executionPlan"`
	}
	err := f.client.Client().CallContext(t.Context(), &prepared, "sonic_prepareBundle", proposal)
	require.NoError(err)
	require.Len(prepared.Transactions, incrementCount)

	signer := types.LatestSignerForChainID(f.chainID)
	signedTxs := make([]hexutil.Bytes, incrementCount)
	for i, rawTx := range prepared.Transactions {
		var fields struct {
			To         string `json:"to"`
			Nonce      string `json:"nonce"`
			Gas        string `json:"gas"`
			GasPrice   string `json:"gasPrice"`
			Data       string `json:"data"`
			ChainID    string `json:"chainId"`
			AccessList []struct {
				Address     string   `json:"address"`
				StorageKeys []string `json:"storageKeys"`
			} `json:"accessList"`
		}
		require.NoError(json.Unmarshal(rawTx, &fields))

		txNonce, err := hexutil.DecodeUint64(fields.Nonce)
		require.NoError(err)
		txGas, err := hexutil.DecodeUint64(fields.Gas)
		require.NoError(err)
		txGasPrice, err := hexutil.DecodeBig(fields.GasPrice)
		require.NoError(err)
		txData, err := hexutil.Decode(fields.Data)
		require.NoError(err)
		txTo := common.HexToAddress(fields.To)
		txChainID, err := hexutil.DecodeBig(fields.ChainID)
		require.NoError(err)

		var accessList types.AccessList
		for _, entry := range fields.AccessList {
			var storageKeys []common.Hash
			for _, k := range entry.StorageKeys {
				storageKeys = append(storageKeys, common.HexToHash(k))
			}
			accessList = append(accessList, types.AccessTuple{
				Address:     common.HexToAddress(entry.Address),
				StorageKeys: storageKeys,
			})
		}

		tx := types.NewTx(&types.AccessListTx{
			ChainID:    txChainID,
			Nonce:      txNonce,
			To:         &txTo,
			Gas:        txGas,
			GasPrice:   txGasPrice,
			Data:       txData,
			AccessList: accessList,
		})
		signedTx, err := types.SignTx(tx, signer, f.sender.PrivateKey)
		require.NoError(err)
		encoded, err := signedTx.MarshalBinary()
		require.NoError(err)
		signedTxs[i] = encoded
	}

	var planHash common.Hash
	err = f.client.Client().CallContext(
		t.Context(),
		&planHash,
		"sonic_submitBundle",
		map[string]interface{}{
			"signedTransactions": signedTxs,
			"executionPlan":      prepared.ExecutionPlan,
		},
	)
	require.NoError(err)

	var bundleInfo map[string]interface{}
	require.NoError(tests.WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
		bundleInfo = nil
		err := f.client.Client().CallContext(ctx, &bundleInfo, "sonic_getBundleInfo", planHash)
		if err != nil {
			return false, err
		}
		return bundleInfo != nil, nil
	}))

	countStr, ok := bundleInfo["count"].(string)
	require.True(ok, "count field missing from bundle info")
	bundleCount, err := hexutil.DecodeUint64(countStr)
	require.NoError(err)
	require.Equal(uint64(incrementCount), bundleCount)

	blockStr, ok := bundleInfo["block"].(string)
	require.True(ok, "block field missing from bundle info")
	bundleBlock, err := hexutil.DecodeUint64(blockStr)
	require.NoError(err)
	require.Greater(bundleBlock, uint64(0))

	f.verifyCounterValue(t, incrementCount)
}

// newBundleTestFixture starts an integration network with bundles enabled,
// deploys a Counter contract, funds a sender account, and returns the shared
// state needed by all bundle RPC tests. Client cleanup is registered with t.
func newBundleTestFixture(t *testing.T) *bundleTestFixture {
	t.Helper()
	require := require.New(t)
	net := bundles.GetIntegrationTestNetWithBundlesEnabled(t)

	_, receipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(err)

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	client, err := net.GetClient()
	require.NoError(err)
	t.Cleanup(func() { client.Close() })

	counterAbi, err := counter.CounterMetaData.GetAbi()
	require.NoError(err)
	incrementData, err := counterAbi.Pack("incrementCounter")
	require.NoError(err)

	nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
	require.NoError(err)

	return &bundleTestFixture{
		client:                 client,
		sender:                 sender,
		chainID:                net.GetChainId(),
		counterContractAddress: receipt.ContractAddress,
		incrementData:          incrementData,
		nonce:                  nonce,
	}
}

// verifyCounterValue asserts the on-chain counter equals expected.
func (f *bundleTestFixture) verifyCounterValue(t *testing.T, expected int64) {
	t.Helper()
	require := require.New(t)
	instance, err := counter.NewCounter(f.counterContractAddress, f.client)
	require.NoError(err)
	count, err := instance.GetCount(nil)
	require.NoError(err)
	require.Equal(expected, count.Int64())
}
