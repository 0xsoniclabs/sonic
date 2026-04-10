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

package bundles

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// PrepareBundle is a wrapper around the rpc method sonic_prepareBundle, which
// prepares a bundle for execution by filling in all necessary fields and
// encoding them properly.
//
// It accepts transactions in the form of CallMsg to keep compatibility with
// standard go-ethereum client methods like EstimateGas.
// CallMsg is a more convenient type to prepare transactions,
// it does not encode fields into hex and is compatible with standard
// go-ethereum client methods like EstimateGas.
// Unfortunately, it does not include nonce, therefore this function needs
// to assign a fitting value.
//
// This function also estimates gas for each transaction and fills in the Gas field,
// as it is required by sonic_prepareBundle.
// if earliest and latest block numbers are not provided, it will set earliest to the next block after submission
// and latest to 1024 blocks after earliest.
//
// This function should be part of the go-ethereum client object, being the entry
// point to the api from go programs.
func PrepareBundle(
	t *testing.T, client *tests.PooledEhtClient,
	txs []ethereum.CallMsg,
	earliest, latest *int64,
) (ethapi.RPCPreparedBundle, error) {

	nonces := make(map[common.Address]uint64)
	for _, tx := range txs {
		if _, ok := nonces[tx.From]; !ok {
			nonce, err := client.PendingNonceAt(t.Context(), tx.From)
			require.NoError(t, err, "failed to get pending nonce")
			nonces[tx.From] = nonce
		}
	}

	// Convert CallMsg without nonce into TransactionArgs with nonce and hex encoding of fields
	txsArgs := make([]ethapi.TransactionArgs, len(txs))
	for i, tx := range txs {
		nonce := nonces[tx.From]
		nonces[tx.From] = nonce + 1
		txArgs := ethapi.TransactionArgs{
			From:     &tx.From,
			To:       tx.To,
			Nonce:    (*hexutil.Uint64)(&nonce),
			GasPrice: (*hexutil.Big)(tx.GasPrice),
			Value:    (*hexutil.Big)(tx.Value),
			Data:     (*hexutil.Bytes)(&tx.Data),
		}
		txsArgs[i] = txArgs
	}

	var earliestBlock, latestBlock *rpc.BlockNumber
	if earliest != nil {
		earliestBlock = (*rpc.BlockNumber)(earliest)
	}
	if latest != nil {
		latestBlock = (*rpc.BlockNumber)(latest)
	}

	// Call sonic_prepareBundle to get a bundle with all fields properly filled in and encoded
	var preparedBundle ethapi.RPCPreparedBundle
	err := client.Client().Call(&preparedBundle, "sonic_prepareBundle",
		ethapi.PrepareBundleArgs{
			Transactions:  txsArgs,
			EarliestBlock: earliestBlock,
			LatestBlock:   latestBlock,
		})
	require.NoError(t, err, "failed to call sonic_prepareBundle")
	return preparedBundle, nil
}

// SubmitBundle is a wrapper around the rpc method sonic_submitBundle, which
// submits a prepared bundle for execution.
// It uses types.Transaction just like the method SendTransaction.
// This function should be part of the go-ethereum client object, being the entry
// point to the api from go programs.
func SubmitBundle(client *tests.PooledEhtClient,
	txs []*types.Transaction,
	plan ethapi.RPCExecutionPlan,
) (common.Hash, error) {
	encodedTransactions := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		data, err := tx.MarshalBinary()
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to marshal transaction: %w", err)
		}
		encodedTransactions[i] = hexutil.Bytes(data)
	}

	var bundleHash common.Hash
	err := client.Client().Call(&bundleHash, "sonic_submitBundle",
		ethapi.SubmitBundleArgs{
			SignedTransactions: encodedTransactions,
			ExecutionPlan:      plan,
		})
	return bundleHash, err
}

func waitForBundleExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*ethapi.RPCBundleInfo, error) {
	infos, err := WaitForBundlesExecution(
		ctxt, client,
		[]common.Hash{executionPlanHash},
	)
	return infos[0], err
}
func GetBundleInfo(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*ethapi.RPCBundleInfo, error) {
	var info *ethapi.RPCBundleInfo
	err := client.CallContext(
		ctxt,
		&info,
		"sonic_getBundleInfo",
		executionPlanHash,
	)
	if err == nil && info == nil {
		return nil, ethereum.NotFound
	}
	return info, err
}

func GetProcessedBundleHistoryHash(
	ctxt context.Context,
	client *rpc.Client,
) (*bundle.HistoryHash, error) {
	var historyHash *bundle.HistoryHash
	err := client.CallContext(
		ctxt,
		&historyHash,
		"sonic_getProcessedBundleHistoryHash",
	)
	return historyHash, err
}

func WaitForBundlesExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHashes []common.Hash,
) ([]*ethapi.RPCBundleInfo, error) {

	infos := make([]*ethapi.RPCBundleInfo, len(executionPlanHashes))
	err := tests.WaitFor(ctxt, func(innerCtx context.Context) (bool, error) {
		for i, plan := range executionPlanHashes {
			if infos[i] != nil {
				continue
			}

			info, err := GetBundleInfo(innerCtx, client, plan)
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				return false, err
			}

			if info != nil {
				infos[i] = info
			}
		}
		return !slices.Contains(infos, nil), nil
	})
	return infos, err
}
