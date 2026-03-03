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

package bundle

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

const (
	successfulNormalTx    = 0
	failedNormalTx        = 1
	invalidNormalTx       = 2
	successfulSponsoredTx = 3
	failedSponsoredTx     = 4
	invalidSponsoredTx    = 5
)

const (
	paymentTxIndex   = -1
	uncheckedTxIndex = -2
)

const (
	successStatus = types.ReceiptStatusSuccessful
	failedStatus  = types.ReceiptStatusFailed
)

type Case struct {
	tryUntil         bool
	tolerateFailed   bool
	tolerateInvalid  bool
	submittedTxTypes []any
	blockTxs         []int // index of tx hash of submitted transactions, -1 for paymentTx, -2 for unchecked transactions
	blockTxStatuses  []uint64
	counter          int64
}

type NamedCase struct {
	name  string
	case_ Case
}

type SubCaseVariant struct {
	submittedTxTypes any
	blockTxs         []int
	blockTxStatuses  []uint64
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
				successfulNormalTx,
				[]int{uncheckedTxIndex}, // relative 0
				[]uint64{successStatus},
				1,
			},
			failed: SubCaseVariant{
				failedNormalTx,
				[]int{uncheckedTxIndex}, // relative 0
				[]uint64{failedStatus},
				0,
			},
			invalid: SubCaseVariant{
				invalidNormalTx,
				[]int{},
				[]uint64{},
				0,
			},
		},
		"sponsored": {
			success: SubCaseVariant{
				successfulSponsoredTx,
				[]int{uncheckedTxIndex, uncheckedTxIndex}, // relative 0, uncheckedTxIndex
				[]uint64{successStatus, successStatus},
				1,
			},
			failed: SubCaseVariant{
				failedSponsoredTx,
				[]int{uncheckedTxIndex, uncheckedTxIndex}, // relative 0, uncheckedTxIndex
				[]uint64{failedStatus, successStatus},
				0,
			},
			invalid: SubCaseVariant{
				invalidSponsoredTx,
				[]int{},
				[]uint64{},
				0,
			},
		},
		"bundled": {
			success: SubCaseVariant{
				[]any{successfulNormalTx, successfulNormalTx},
				[]int{uncheckedTxIndex, uncheckedTxIndex, uncheckedTxIndex},
				[]uint64{successStatus, successStatus, successStatus},
				2,
			},
			failed: SubCaseVariant{
				[]any{successfulNormalTx, failedNormalTx},
				[]int{uncheckedTxIndex},
				[]uint64{successStatus},
				0,
			},
			invalid: SubCaseVariant{
				[]any{}, // empty bundle will be converted to bundle with invalid payment transaction
				[]int{uncheckedTxIndex},
				[]uint64{failedStatus},
				0,
			},
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
					Merge[any](successfulNormalTx, subcase.success.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, subcase.success.blockTxs, 2),
					Merge[uint64](successStatus, successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, false, false,
					Merge[any](successfulNormalTx, subcase.failed.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex),
					Merge[uint64](successStatus),
					0,
				},
			},
			{
				name + "/invalid",
				Case{false, false, false,
					Merge[any](successfulNormalTx, subcase.invalid.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex),
					Merge[uint64](successStatus),
					0,
				},
			},
			// TolerateInvalid
			{
				name + "/success",
				Case{false, false, true,
					Merge[any](successfulNormalTx, subcase.success.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, subcase.success.blockTxs, 2),
					Merge[uint64](successStatus, successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, false, true,
					Merge[any](successfulNormalTx, subcase.failed.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex),
					Merge[uint64](successStatus),
					0,
				},
			},
			{
				name + "/invalid",
				Case{false, false, true,
					Merge[any](successfulNormalTx, subcase.invalid.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, 2),
					Merge[uint64](successStatus, successStatus, successStatus),
					1 + 1,
				},
			},
			// TolerateFailed
			{
				name + "/success",
				Case{false, true, false,
					Merge[any](successfulNormalTx, subcase.success.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, subcase.success.blockTxs, 2),
					Merge[uint64](successStatus, successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, true, false,
					Merge[any](successfulNormalTx, subcase.failed.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, subcase.failed.blockTxs, 2),
					Merge[uint64](successStatus, successStatus, subcase.failed.blockTxStatuses, successStatus),
					1 + subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{false, true, false,
					Merge[any](successfulNormalTx, subcase.invalid.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex),
					Merge[uint64](successStatus),
					0,
				},
			},
			// TolerateFailed & TolerateInvalid
			{
				name + "/success",
				Case{false, true, true,
					Merge[any](successfulNormalTx, subcase.success.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, subcase.success.blockTxs, 2),
					Merge[uint64](successStatus, successStatus, subcase.success.blockTxStatuses, successStatus),
					1 + subcase.success.counter + 1,
				},
			},
			{
				name + "/failed",
				Case{false, true, true,
					Merge[any](successfulNormalTx, subcase.failed.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, subcase.failed.blockTxs, 2),
					Merge[uint64](successStatus, successStatus, subcase.failed.blockTxStatuses, successStatus),
					1 + subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{false, true, true,
					Merge[any](successfulNormalTx, subcase.invalid.submittedTxTypes, successfulNormalTx),
					Merge[int](paymentTxIndex, 0, 2),
					Merge[uint64](successStatus, successStatus, successStatus),
					1 + 1,
				},
			},
		}...)
	}
	net, client := startTestnet(t)
	defer client.Close()
	for _, c := range cases {
		checkCase(t, net, client, c)
	}
}

func Test_RunUntilTolerated_Works(t *testing.T) {
	cases := []NamedCase{}
	for name, subcase := range getSubcases() {
		cases = append(cases, []NamedCase{
			{
				name + "/success",
				Case{true, false, false,
					Merge[any](subcase.success.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.success.blockTxs),
					Merge[uint64](successStatus, subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, false, false,
					Merge[any](subcase.failed.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.failed.blockTxs, 1),
					Merge[uint64](successStatus, subcase.failed.blockTxStatuses, successStatus),
					subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{true, false, false,
					Merge[any](subcase.invalid.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, 1),
					Merge[uint64](successStatus, successStatus),
					1,
				},
			},
			// TolerateInvalid
			{
				name + "/success",
				Case{true, false, true,
					Merge[any](subcase.success.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.success.blockTxs),
					Merge[uint64](successStatus, subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, false, true,
					Merge[any](subcase.failed.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.failed.blockTxs, 1),
					Merge[uint64](successStatus, subcase.failed.blockTxStatuses, successStatus),
					subcase.failed.counter + 1,
				},
			},
			{
				name + "/invalid",
				Case{true, false, true,
					Merge[any](subcase.invalid.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex),
					Merge[uint64](successStatus),
					0,
				},
			},
			// TolerateFailed
			{
				name + "/success",
				Case{true, true, false,
					Merge[any](subcase.success.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.success.blockTxs),
					Merge[uint64](successStatus, subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, true, false,
					Merge[any](subcase.failed.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.failed.blockTxs),
					Merge[uint64](successStatus, subcase.failed.blockTxStatuses),
					subcase.failed.counter,
				},
			},
			{
				name + "/invalid",
				Case{true, true, false,
					Merge[any](subcase.invalid.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, 1),
					Merge[uint64](successStatus, successStatus),
					1,
				},
			},
			// TolerateFailed & TolerateInvalid
			{
				name + "/success",
				Case{true, true, true,
					Merge[any](subcase.success.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.success.blockTxs),
					Merge[uint64](successStatus, subcase.success.blockTxStatuses),
					subcase.success.counter,
				},
			},
			{
				name + "/failed",
				Case{true, true, true,
					Merge[any](subcase.failed.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex, subcase.failed.blockTxs),
					Merge[uint64](successStatus, subcase.failed.blockTxStatuses),
					subcase.failed.counter,
				},
			},
			{
				name + "/invalid",
				Case{true, true, true,
					Merge[any](subcase.invalid.submittedTxTypes, successfulNormalTx, successfulNormalTx),
					Merge[int](paymentTxIndex),
					Merge[uint64](successStatus),
					0,
				},
			},
		}...)
	}
	net, client := startTestnet(t)
	defer client.Close()
	for _, c := range cases {
		checkCase(t, net, client, c)
	}
}

func Merge[T any](items ...any) []T {
	var result []T

	for _, item := range items {
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

		bundler := net.GetSessionSponsor()
		bundleTx, paymentTxHash := makeBundleTransaction(t, net, txs, plan, bundler)
		require.NotNil(t, bundleTx)
		require.NotZero(t, paymentTxHash)

		err := client.SendTransaction(t.Context(), bundleTx)
		require.NoError(t, err)

		receipt, err := net.GetReceipt(paymentTxHash)
		require.NoError(t, err, "failed to get payment tx receipt; %v", err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "payment transaction failed")

		// Check transactions hashes and statuses
		transactionHashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
		require.Len(t, transactionHashes, len(c.blockTxs))
		for i := range c.blockTxs {
			switch c.blockTxs[i] {
			case paymentTxIndex:
				checkHashesEqAndStatus(t, net, paymentTxHash, c.blockTxStatuses[i], transactionHashes[i])
			case uncheckedTxIndex:
				checkStatus(t, net, c.blockTxStatuses[i], transactionHashes[i])
			default:
				checkHashesEqAndStatus(t, net, txs[c.blockTxs[i]].Hash(), c.blockTxStatuses[i], transactionHashes[i])
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

func makeUnsignedBundleTxs(
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
	txTypes []any,
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
	for i, txType := range txTypes {
		tx := types.AccessListTx{}
		switch v := txType.(type) {
		case int:
			switch txType {
			case successfulNormalTx:
				tx = types.AccessListTx{
					To:       counterAddress,
					Gas:      counterGasLimit,
					Data:     counterInput,
					GasPrice: gasPrice,
				}
				txs[i] = types.NewTx(&tx)
			case failedNormalTx:
				tx = types.AccessListTx{
					To:       &revertAddress,
					Gas:      revertGasLimit,
					Data:     revertInput,
					GasPrice: gasPrice,
				}
				txs[i] = types.NewTx(tests.SetTransactionDefaults(t, net, &tx, senders[i]))
			case invalidNormalTx:
				tx = types.AccessListTx{
					To:       counterAddress,
					Gas:      1, // invalid
					Data:     counterInput,
					GasPrice: gasPrice,
				}
				txs[i] = types.NewTx(tests.SetTransactionDefaults(t, net, &tx, senders[i]))
			case successfulSponsoredTx:
				donation := big.NewInt(1e16)
				gas_subsidies.Fund(t, net, senders[i].Address(), donation)
				tx = types.AccessListTx{
					To:       counterAddress,
					Gas:      counterGasLimit,
					Data:     counterInput,
					GasPrice: big.NewInt(0),
				}
				txs[i] = types.NewTx(&tx)
			case failedSponsoredTx:
				donation := big.NewInt(1e16)
				gas_subsidies.Fund(t, net, senders[i].Address(), donation)
				tx = types.AccessListTx{
					To:       &revertAddress,
					Gas:      revertGasLimit,
					Data:     revertInput,
					GasPrice: big.NewInt(0),
				}
				txs[i] = types.NewTx(&tx)
			case invalidSponsoredTx:
				tx = types.AccessListTx{
					To:       counterAddress,
					Gas:      counterGasLimit,
					Data:     counterInput,
					GasPrice: big.NewInt(0),
				}
				txs[i] = types.NewTx(&tx)
			}
		case []any:
			flags := bundle.ExecutionFlag(0)
			btxs, bplan, _ := makeSignedBundleOnlyTxsAndPlan(t, net, client, txType.([]any), counterAddress, flags)

			bundler := senders[i]
			if len(txType.([]any)) == 0 {
				// make invalid paymentTx
				bundler = net.GetSessionSponsor()
			}
			bundleTx, paymentTxHash := makeBundleTransaction(t, net, btxs, bplan, bundler)
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

			require.NotNil(t, bundleTx)
			require.NotZero(t, paymentTxHash)
			txs[i] = bundleTx
		default:
			panic(fmt.Sprintf("unexpected type %T in makeUnsignedBundleTxs", v))
		}
	}

	return txs, senders, *counterAddress
}

func signBundleOnlyTxs(t *testing.T, net *tests.IntegrationTestNet, txs []*types.Transaction, senders []*tests.Account, plan bundle.ExecutionPlan) {
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
	txTypes []any,
	counterAddressPtr *common.Address,
	flags bundle.ExecutionFlag,
) ([]*types.Transaction, bundle.ExecutionPlan, common.Address) {
	txs, senders, counterAddress := makeUnsignedBundleTxs(t, net, client, txTypes, counterAddressPtr)

	signer := types.NewCancunSigner(net.GetChainId())

	steps := make([]bundle.ExecutionStep, len(txs))
	for i, tx := range txs {
		steps[i] = bundle.ExecutionStep{From: senders[i].Address(), Hash: signer.Hash(tx)}
	}
	plan := bundle.ExecutionPlan{Flags: flags, Steps: steps}

	signBundleOnlyTxs(t, net, txs, senders, plan)

	return txs, plan, counterAddress
}

func checkHashesEqAndStatus(t *testing.T, net *tests.IntegrationTestNet, expectedHash common.Hash, expectedStatus uint64, txHash common.Hash) {
	t.Helper()
	require.Equal(t, expectedHash, txHash)
	checkStatus(t, net, expectedStatus, txHash)
}

func checkStatus(t *testing.T, net *tests.IntegrationTestNet, status uint64, txHash common.Hash) {
	t.Helper()
	receipt, err := net.GetReceipt(txHash)
	require.NoError(t, err, "failed to get transaction receipt; %v", err)
	require.Equal(t, status, receipt.Status)
}
