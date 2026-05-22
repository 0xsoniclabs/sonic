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
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/bundles"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestBundleRPCFunctions exercises sonic_prepareBundle, sonic_submitBundle
// and sonic_getBundleInfo RPC methods in a single test. It prepares a bundle
// using a plain JSON string as the proposal. The proposal covers blockRange,
// tolerateInvalid, tolerateFailed, and oneOf flags to verify that the RPC
// layer correctly unmarshals all fields. The test then submits the bundle and waits for it to get executed,
// finally verifying the execution by checking the counter value and the bundle info.
func TestBundleRPCFunctions(t *testing.T) {
	require := require.New(t)
	net := bundles.GetIntegrationTestNetWithBundlesEnabled(t)

	counterContract, receipt, err := tests.DeployContract(net, counter.DeployCounter)
	require.NoError(err)

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	client, err := net.GetClient()
	require.NoError(err)
	t.Cleanup(func() { client.Close() })

	counterAbi, err := counter.CounterMetaData.GetAbi()
	require.NoError(err)
	incrementData, err := counterAbi.Pack("incrementCounter")
	require.NoError(err)

	blockNum, err := client.BlockNumber(t.Context())
	require.NoError(err)
	nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
	require.NoError(err)
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)

	blockFirst := hexutil.EncodeUint64(blockNum)
	senderAddress := sender.Address().Hex()
	contract := receipt.ContractAddress.Hex()
	gasP := hexutil.EncodeBig(gasPrice)
	data := hexutil.Encode(incrementData)
	nonce0 := hexutil.EncodeUint64(nonce)
	nonce1 := hexutil.EncodeUint64(nonce + 1)
	nonceInv := hexutil.EncodeUint64(uint64(1_000_000))

	// The proposal is written as a plain JSON string so the wire format is readable.
	// Structure:
	//   blockRange  — explicit first/length to verify hex uint64 unmarshalling
	//   step 1      — valid increment (nonce N)
	//   step 2      — tolerateInvalid with a far-future nonce; skipped at runtime
	//   oneOf group — two branches, both tolerateFailed:
	//                   branch A: valid nonce (N+1), executes first
	//                   branch B: far-future nonce, unreachable once A succeeds
	proposalJSON := fmt.Sprintf(`{
    "blockRange": {
        "first": %q,
        "length": "0xa"
    },
    "steps": [
        {
            "from": %q, "to": %q,
            "gas": "0xc350", "gasPrice": %q,
            "nonce": %q, "data": %q
        },
        {
            "tolerateInvalid": true,
            "from": %q, "to": %q,
            "gas": "0xc350", "gasPrice": %q,
            "nonce": %q, "data": %q
        },
        {
            "oneOf": true,
            "steps": [
                {
                    "tolerateFailed": true,
                    "from": %q, "to": %q,
                    "gas": "0xc350", "gasPrice": %q,
                    "nonce": %q, "data": %q
                },
                {
                    "tolerateFailed": true,
                    "from": %q, "to": %q,
                    "gas": "0xc350", "gasPrice": %q,
                    "nonce": %q, "data": %q
                }
            ]
        }
    ]
}`,
		blockFirst,
		senderAddress, contract, gasP, nonce0, data, // step 1: valid
		senderAddress, contract, gasP, nonceInv, data, // step 2: tolerateInvalid
		senderAddress, contract, gasP, nonce1, data, // oneOf branchA: tolerateFailed (valid)
		senderAddress, contract, gasP, nonceInv, data, // oneOf branchB: tolerateFailed (invalid, unreachable)
	)

	var prepared struct {
		Transactions  []json.RawMessage `json:"transactions"`
		ExecutionPlan json.RawMessage   `json:"executionPlan"`
	}
	err = client.Client().CallContext(
		t.Context(), &prepared, "sonic_prepareBundle", json.RawMessage(proposalJSON),
	)
	require.NoError(err)
	require.NotEmpty(prepared.Transactions)

	signer := types.LatestSignerForChainID(net.GetChainId())
	signedTxs := make([]hexutil.Bytes, len(prepared.Transactions))
	for i, rawTx := range prepared.Transactions {
		signedTxs[i] = signPreparedTx(t, rawTx, signer, sender.PrivateKey)
	}

	var planHash common.Hash
	err = client.Client().CallContext(
		t.Context(), &planHash, "sonic_submitBundle",
		map[string]interface{}{
			"signedTransactions": signedTxs,
			"executionPlan":      prepared.ExecutionPlan,
		},
	)
	require.NoError(err)
	require.NotEqual(common.Hash{}, planHash)

	var bundleInfo map[string]interface{}
	require.NoError(tests.WaitFor(t.Context(), func(ctx context.Context) (bool, error) {
		bundleInfo = nil
		err := client.Client().CallContext(ctx, &bundleInfo, "sonic_getBundleInfo", planHash)
		if err != nil {
			return false, err
		}
		return bundleInfo != nil, nil
	}))

	countStr, ok := bundleInfo["count"].(string)
	require.True(ok, "count field missing from bundle info")
	bundleCount, err := hexutil.DecodeUint64(countStr)
	require.NoError(err)
	require.Equal(uint64(2), bundleCount, "unexpected transaction count in bundle info")

	blockStr, ok := bundleInfo["block"].(string)
	require.True(ok, "block field missing from bundle info")
	bundleBlock, err := hexutil.DecodeUint64(blockStr)
	require.NoError(err)
	require.Greater(bundleBlock, uint64(0))

	// Verify that the bundle's transactions got executed by checking the counter value.
	count, err := counterContract.GetCount(nil)
	require.NoError(err)
	require.Equal(int64(2), count.Int64())

}

// signPreparedTx signs a single transaction returned by sonic_prepareBundle.
func signPreparedTx(t *testing.T, rawTx json.RawMessage, signer types.Signer, key *ecdsa.PrivateKey) hexutil.Bytes {
	t.Helper()
	require := require.New(t)

	var fields struct {
		To         string `json:"to"`
		Nonce      string `json:"nonce"`
		Gas        string `json:"gas"`
		GasPrice   string `json:"gasPrice"`
		Data       string `json:"data"`
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
	txData, err := hexutil.Decode(fields.Data)
	require.NoError(err)
	txTo := common.HexToAddress(fields.To)

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

	txGasPrice, err := hexutil.DecodeBig(fields.GasPrice)
	require.NoError(err)
	tx := types.NewTx(&types.AccessListTx{
		Nonce:      txNonce,
		To:         &txTo,
		Gas:        txGas,
		GasPrice:   txGasPrice,
		Data:       txData,
		AccessList: accessList,
	})

	signedTx, err := types.SignTx(tx, signer, key)
	require.NoError(err)
	encoded, err := signedTx.MarshalBinary()
	require.NoError(err)
	return hexutil.Bytes(encoded)
}
