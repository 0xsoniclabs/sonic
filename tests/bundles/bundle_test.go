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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type txType interface {
	makeTx(txMakeOptions) *types.Transaction
}

type txIndex int

const (
	uncheckedTxIndex txIndex = -1
)

type txStatus uint64

const (
	successStatus txStatus = txStatus(types.ReceiptStatusSuccessful)
	failedStatus  txStatus = txStatus(types.ReceiptStatusFailed)
)

type Case struct {
	tryUntil         bool
	tolerateFailed   bool
	tolerateInvalid  bool
	submittedTxTypes []txType
	blockTxIndices   []txIndex
	blockTxStatuses  []txStatus
	counter          int64
}

type NamedCase struct {
	name  string
	case_ Case
}

type SubCaseVariant struct {
	submittedTxTypes txType
	blockTxIndices   []txIndex
	blockTxStatuses  []txStatus
	counter          int64
}

type SubCase struct {
	success SubCaseVariant
	failed  SubCaseVariant
	invalid SubCaseVariant
}

func getSubcases() map[string]SubCase {
	return map[string]SubCase{
		"normal": {
			success: SubCaseVariant{
				successfulNormalTx{},
				[]txIndex{uncheckedTxIndex}, // relative 0
				[]txStatus{successStatus},
				1,
			},
			failed: SubCaseVariant{
				failedNormalTx{},
				[]txIndex{uncheckedTxIndex}, // relative 0
				[]txStatus{failedStatus},
				0,
			},
			invalid: SubCaseVariant{
				invalidNormalTx{},
				[]txIndex{},
				[]txStatus{},
				0,
			},
		},
		"sponsored": {
			success: SubCaseVariant{
				successfulSponsoredTx{},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex}, // relative 0, uncheckedTxIndex
				[]txStatus{successStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				failedSponsoredTx{},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex}, // relative 0, uncheckedTxIndex
				[]txStatus{failedStatus, successStatus},
				0,
			},
			invalid: SubCaseVariant{
				invalidSponsoredTx{},
				[]txIndex{},
				[]txStatus{},
				0,
			},
		},
		"bundled": {
			success: SubCaseVariant{
				subBundleTx{txTypes: []txType{successfulNormalTx{}, successfulNormalTx{}}},
				[]txIndex{uncheckedTxIndex, uncheckedTxIndex},
				[]txStatus{successStatus, successStatus},
				2,
			},
			failed: SubCaseVariant{
				subBundleTx{txTypes: []txType{successfulNormalTx{}, failedNormalTx{}}},
				[]txIndex{},
				[]txStatus{},
				0,
			},
			// skipped bundles are no longer necessary, and all **/bundled/invalid tests are skipped
		},
	}
}

func Test_RunAllUnlessNotTolerated_Works(t *testing.T) {
	cases := []NamedCase{}
	for name, subcase := range getSubcases() {
		cases = append(cases, []NamedCase{
			{
				name + "/success",
				Case{false, false, false,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, false, false,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			{
				name + "/invalid",
				Case{false, false, false,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			// TolerateInvalid
			{
				name + "/success",
				Case{false, false, true,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, false, true,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			{
				name + "/invalid",
				Case{false, false, true,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), txIndex(2)),
					Merge[txStatus](successStatus, successStatus),
					1 + 1,
				},
			},
			// TolerateFailed
			{
				name + "/success",
				Case{false, true, false,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, true, false,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.failed.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.failed.blockTxStatuses, successStatus),
					1 + subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{false, true, false,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			// TolerateFailed & TolerateInvalid
			{
				name + "/success",
				Case{false, true, true,
					Merge[txType](successfulNormalTx{}, subcase.success.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.success.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, true, true,
					Merge[txType](successfulNormalTx{}, subcase.failed.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), subcase.failed.blockTxIndices, txIndex(2)),
					Merge[txStatus](successStatus, subcase.failed.blockTxStatuses, successStatus),
					1 + subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{false, true, true,
					Merge[txType](successfulNormalTx{}, subcase.invalid.submittedTxTypes, successfulNormalTx{}),
					Merge[txIndex](txIndex(0), txIndex(2)),
					Merge[txStatus](successStatus, successStatus),
					1 + 1,
				},
			},
		}...)
	}
	net, client := startTestnet(t)
	defer client.Close()
	for _, c := range cases {
		if c.name != "bundled/invalid" {
			checkCase(t, net, client, c)
		}
	}
}

func Test_RunUntilTolerated_Works(t *testing.T) {
	cases := []NamedCase{}
	for name, subcase := range getSubcases() {
		cases = append(cases, []NamedCase{
			{
				name + "/success",
				Case{true, false, false,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, false, false,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices, txIndex(1)),
					Merge[txStatus](subcase.failed.blockTxStatuses, successStatus),
					subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{true, false, false,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](txIndex(1)),
					Merge[txStatus](successStatus),
					1,
				},
			},
			// TolerateInvalid
			{
				name + "/success",
				Case{true, false, true,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, false, true,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices, txIndex(1)),
					Merge[txStatus](subcase.failed.blockTxStatuses, successStatus),
					subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{true, false, true,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
			// TolerateFailed
			{
				name + "/success",
				Case{true, true, false,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, true, false,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices),
					Merge[txStatus](subcase.failed.blockTxStatuses),
					subcase.failed.counter,
				},
			},
			{
				name + "/invalid",
				Case{true, true, false,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](txIndex(1)),
					Merge[txStatus](successStatus),
					1,
				},
			},
			// TolerateFailed & TolerateInvalid
			{
				name + "/success",
				Case{true, true, true,
					Merge[txType](subcase.success.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.success.blockTxIndices),
					Merge[txStatus](subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, true, true,
					Merge[txType](subcase.failed.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](subcase.failed.blockTxIndices),
					Merge[txStatus](subcase.failed.blockTxStatuses),
					subcase.failed.counter,
				},
			},
			{
				name + "/invalid",
				Case{true, true, true,
					Merge[txType](subcase.invalid.submittedTxTypes, successfulNormalTx{}, successfulNormalTx{}),
					Merge[txIndex](),
					Merge[txStatus](),
					0,
				},
			},
		}...)
	}
	net, client := startTestnet(t)
	defer client.Close()
	for _, c := range cases {
		if c.name != "bundled/invalid" {
			checkCase(t, net, client, c)
		}
	}
}

func Merge[T any](items ...any) []T {
	var result []T
	if len(items) == 0 {
		return result
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		switch v := item.(type) {
		case T:
			result = append(result, v)
		case []T:
			result = append(result, v...)
		default:
			panic(fmt.Sprintf("unexpected type %T in Merge", v))
		}
	}

	return result
}

func checkCase(t *testing.T, net *tests.IntegrationTestNet, client *tests.PooledEhtClient, namedCase NamedCase) {
	c := namedCase.case_
	name := fmt.Sprintf("TryUntil=%v/TolerateFailed=%v/TolerateInvalid=%v/%s", c.tryUntil, c.tolerateFailed, c.tolerateInvalid, namedCase.name)
	t.Run(name, func(t *testing.T) {
		flags := bundle.ExecutionFlag(0)
		flags.SetTolerateInvalid(c.tolerateInvalid)
		flags.SetTolerateFailed(c.tolerateFailed)
		flags.SetTryUntil(c.tryUntil)

		txs, plan, counterAddress := makeSignedBundleOnlyTxsAndPlan(t, net, client, c.submittedTxTypes, nil, flags)

		bundleTx := makeBundleTransaction(t, net, txs, plan)
		require.NotNil(t, bundleTx)

		err := client.SendTransaction(t.Context(), bundleTx)
		require.NoError(t, err)

		// Wait for the bundle to be processed.
		info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
		require.NoError(t, err)
		require.NotNil(t, info.Block)

		// Check transactions hashes and statuses
		transactionHashes := getTransactionsInBlock(t, net, big.NewInt(int64(*info.Block)))

		// Truncate potential internal transactions at the beginning of the
		// block. The rest should only be transactions from the bundle.
		require.LessOrEqual(t, int(*info.Position), len(transactionHashes))
		transactionHashes = transactionHashes[*info.Position:]

		require.Len(t, transactionHashes, len(c.blockTxIndices))
		for i := range c.blockTxIndices {
			switch c.blockTxIndices[i] {
			case uncheckedTxIndex:
				checkStatus(t, net, c.blockTxStatuses[i], transactionHashes[i])
			default:
				checkHashesEqAndStatus(t, net, txs[c.blockTxIndices[i]].Hash(), c.blockTxStatuses[i], transactionHashes[i])
			}
		}

		// Check the final state is correct
		require.Equal(t, c.counter, getCounterValue(t, client, counterAddress))
	})
}

func startTestnet(t *testing.T) (*tests.IntegrationTestNet, *tests.PooledEhtClient) {
	updates := opera.GetBrioUpgrades()
	updates.GasSubsidies = true
	updates.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(updates),
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	return net, client
}

func counterAddressAndInput(t *testing.T, net *tests.IntegrationTestNet) (common.Address, []byte) {
	_, counterAbi, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
	counterInput := generateCallData(t, counterAbi, "incrementCounter")
	return counterAddress, counterInput
}

func revertAddressAndInput(t *testing.T, net *tests.IntegrationTestNet) (common.Address, []byte) {
	_, revertABI, revertAddress := prepareContract(t, net, revert.RevertMetaData.GetAbi, revert.DeployRevert)
	revertInput := generateCallData(t, revertABI, "doCrash")
	return revertAddress, revertInput
}

func getCounterValue(t *testing.T, client *tests.PooledEhtClient, counterAddress common.Address) int64 {
	counterInstance, err := counter.NewCounter(counterAddress, client)
	require.NoError(t, err, "failed to create counter instance; %v", err)
	count, err := counterInstance.GetCount(nil)
	require.NoError(t, err, "failed to get counter value; %v", err)
	return count.Int64()
}

type txMakeOptions struct {
	t      *testing.T
	net    *tests.IntegrationTestNet
	client *tests.PooledEhtClient

	counterAddress  *common.Address
	counterGasLimit uint64
	counterInput    []byte

	revertAddress  common.Address
	revertGasLimit uint64
	revertInput    []byte

	gasPrice *big.Int

	sender *tests.Account
}

type successfulNormalTx struct{}

func (t successfulNormalTx) makeTx(opts txMakeOptions) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		To:       opts.counterAddress,
		Gas:      opts.counterGasLimit,
		Data:     opts.counterInput,
		GasPrice: opts.gasPrice,
	})
}

type failedNormalTx struct{}

func (t failedNormalTx) makeTx(opts txMakeOptions) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		To:       &opts.revertAddress,
		Gas:      opts.revertGasLimit,
		Data:     opts.revertInput,
		GasPrice: opts.gasPrice,
	})
}

type invalidNormalTx struct{}

func (t invalidNormalTx) makeTx(opts txMakeOptions) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		To:       opts.counterAddress,
		Gas:      1, // invalid
		Data:     opts.counterInput,
		GasPrice: opts.gasPrice,
	})
}

type successfulSponsoredTx struct{}

func (t successfulSponsoredTx) makeTx(opts txMakeOptions) *types.Transaction {
	donation := big.NewInt(1e16)
	gas_subsidies.Fund(opts.t, opts.net, opts.sender.Address(), donation)
	return types.NewTx(&types.AccessListTx{
		To:       opts.counterAddress,
		Gas:      opts.counterGasLimit,
		Data:     opts.counterInput,
		GasPrice: big.NewInt(0),
	})
}

type failedSponsoredTx struct{}

func (t failedSponsoredTx) makeTx(opts txMakeOptions) *types.Transaction {
	donation := big.NewInt(1e16)
	gas_subsidies.Fund(opts.t, opts.net, opts.sender.Address(), donation)
	return types.NewTx(&types.AccessListTx{
		To:       &opts.revertAddress,
		Gas:      opts.revertGasLimit,
		Data:     opts.revertInput,
		GasPrice: big.NewInt(0),
	})
}

type invalidSponsoredTx struct{}

func (t invalidSponsoredTx) makeTx(opts txMakeOptions) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		To:       opts.counterAddress,
		Gas:      opts.counterGasLimit,
		Data:     opts.counterInput,
		GasPrice: big.NewInt(0),
	})
}

type subBundleTx struct {
	txTypes []txType
	flags   bundle.ExecutionFlag
}

func (t subBundleTx) makeTx(opts txMakeOptions) *types.Transaction {
	bundleTxs, bundlePlan, _ := makeSignedBundleOnlyTxsAndPlan(opts.t, opts.net, opts.client, t.txTypes, opts.counterAddress, t.flags)

	bundleTx := makeBundleTransaction(opts.t, opts.net, bundleTxs, bundlePlan)
	// remove signature
	bundleTx = types.NewTx(&types.AccessListTx{
		Nonce:      bundleTx.Nonce(),
		GasPrice:   bundleTx.GasPrice(),
		Gas:        bundleTx.Gas(),
		To:         bundleTx.To(),
		Value:      bundleTx.Value(),
		Data:       bundleTx.Data(),
		AccessList: bundleTx.AccessList(),
	})

	require.NotNil(opts.t, bundleTx)
	return bundleTx
}

func makeUnsignedBundleTxs(
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
	txTypes []txType,
	counterAddress *common.Address,
) ([]*types.Transaction, []*tests.Account, common.Address) {
	senders := make([]*tests.Account, len(txTypes))
	for i := range txTypes {
		senders[i] = tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	}

	counterAddr, counterInput := counterAddressAndInput(t, net)
	if counterAddress == nil {
		counterAddress = &counterAddr
	}
	revertAddress, revertInput := revertAddressAndInput(t, net)

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)

	counterGasLimit, err := client.EstimateGas(t.Context(), ethereum.CallMsg{
		From:     tests.MakeAccountWithBalance(t, net, big.NewInt(1e18)).Address(),
		To:       counterAddress,
		Data:     counterInput,
		GasPrice: gasPrice,
		AccessList: types.AccessList{
			// add one entry to the estimation, to allocate gas for the bundle-only marker
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{}}},
		},
	})
	require.NoError(t, err, "failed to estimate gas")

	revertGasLimit := counterGasLimit

	txs := make([]*types.Transaction, len(txTypes))
	for i, tType := range txTypes {
		txs[i] = tType.makeTx(txMakeOptions{
			t, net, client,
			counterAddress,
			counterGasLimit,
			counterInput,
			revertAddress,
			revertGasLimit,
			revertInput,
			gasPrice,
			senders[i],
		})
	}

	return txs, senders, *counterAddress
}

func signBundleOnlyTxs(
	t *testing.T,
	net *tests.IntegrationTestNet,
	txs []*types.Transaction,
	senders []*tests.Account,
	plan bundle.ExecutionPlan,
) {
	for i, tx := range txs {
		bundleOnlyTx := &types.AccessListTx{
			Nonce:    tx.Nonce(),
			GasPrice: tx.GasPrice(),
			Gas:      tx.Gas(),
			To:       tx.To(),
			Value:    tx.Value(),
			Data:     tx.Data(),
			AccessList: append(tx.AccessList(),
				types.AccessTuple{Address: bundle.BundleOnly, StorageKeys: []common.Hash{plan.Hash()}},
			),
		}
		txs[i] = tests.SignTransaction(t, net.GetChainId(), bundleOnlyTx, senders[i])
	}
}

func makeSignedBundleOnlyTxsAndPlan(
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
	txTypes []txType,
	counterAddressPtr *common.Address,
	flags bundle.ExecutionFlag,
) ([]*types.Transaction, bundle.ExecutionPlan, common.Address) {
	txs, senders, counterAddress := makeUnsignedBundleTxs(t, net, client, txTypes, counterAddressPtr)

	signer := types.NewCancunSigner(net.GetChainId())
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err, "failed to get block number; %v", err)

	steps := make([]bundle.ExecutionStep, len(txs))
	for i, tx := range txs {
		steps[i] = bundle.ExecutionStep{From: senders[i].Address(), Hash: signer.Hash(tx)}
	}
	plan := bundle.ExecutionPlan{
		Flags:    flags,
		Steps:    steps,
		Earliest: blockNumber,
		Latest:   blockNumber + 100,
	}

	signBundleOnlyTxs(t, net, txs, senders, plan)

	return txs, plan, counterAddress
}

func checkHashesEqAndStatus(
	t *testing.T,
	net *tests.IntegrationTestNet,
	expectedHash common.Hash,
	expectedStatus txStatus,
	txHash common.Hash,
) {
	t.Helper()
	require.Equal(t, expectedHash, txHash)
	checkStatus(t, net, expectedStatus, txHash)
}

func checkStatus(
	t *testing.T,
	net *tests.IntegrationTestNet,
	status txStatus,
	txHash common.Hash,
) {
	t.Helper()
	receipt, err := net.GetReceipt(txHash)
	require.NoError(t, err, "failed to get transaction receipt; %v", err)
	require.Equal(t, status, txStatus(receipt.Status))
}
