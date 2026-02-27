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
	"iter"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_NonBundledTransaction_Works(t *testing.T) {
	rules := opera.GetBrioUpgrades()
	rules.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(rules),
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	sender0 := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	_, counterAbi, counterAddress := prepareContract(t, net, counter.CounterMetaData.GetAbi, counter.DeployCounter)
	input := generateCallData(t, counterAbi, "incrementCounter")

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)

	gasLimit, err := client.EstimateGas(t.Context(), ethereum.CallMsg{
		From:     sender0.Address(),
		To:       &counterAddress,
		Data:     input,
		GasPrice: gasPrice,
		AccessList: types.AccessList{
			// add one entry to the estimation, to allocate gas for the bundle-only marker
			{StorageKeys: []common.Hash{{}}},
		},
	})
	require.NoError(t, err, "failed to estimate gas")
	fmt.Printf("gasLimit: %d (%x)\n", gasLimit, gasLimit)

	donation := big.NewInt(1e16)
	gas_subsidies.Fund(t, net, sender0.Address(), donation)
	tx0 := gas_subsidies.MakeSponsorRequestTransaction(t,
		tests.SetTransactionDefaults(t, net,
			&types.AccessListTx{
				To:       &counterAddress,
				Gas:      gasLimit,
				Data:     input,
				GasPrice: big.NewInt(0),
			},
			sender0,
		),
		net.GetChainId(),
		sender0,
	)

	err = client.SendTransaction(t.Context(), tx0)
	require.NoError(t, err)

	receipt, err := net.GetReceipt(tx0.Hash())
	require.NoError(t, err, "failed to get payment tx receipt; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "payment transaction failed")

	// Check all transactions have been executed and the order is correct
	expectedTranactionHashes := []common.Hash{tx0.Hash()}
	transactionHashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
	require.Equal(t, expectedTranactionHashes, transactionHashes)

	// Check the transaction status
	receipt, err = net.GetReceipt(tx0.Hash())
	require.NoError(t, err, "failed to get transaction tx 0 receipt; %v", err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx0 failed")

	// Check the counter value from the contract
	counterInstance, err := counter.NewCounter(counterAddress, client)
	require.NoError(t, err, "failed to create counter instance; %v", err)
	count, err := counterInstance.GetCount(nil)
	require.NoError(t, err, "failed to get counter value; %v", err)
	require.Equal(t, count.Int64(), int64(1))
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

const (
	successfulNormalTx    = 0
	successfulSponsoredTx = 1
	successfulBundleTx    = 2
	failedTx              = 3
	invalidTx             = 4
)

func makeUnsignedBundleTxs(t *testing.T, net *tests.IntegrationTestNet, client *tests.PooledEhtClient, txTypes []int, counterAddress *common.Address) ([]*types.Transaction, []*tests.Account, common.Address) {
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
		From:     senders[0].Address(),
		To:       counterAddress,
		Data:     counterInput,
		GasPrice: gasPrice,
		AccessList: types.AccessList{
			// add one entry to the estimation, to allocate gas for the bundle-only marker
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{}}},
		},
	})
	require.NoError(t, err, "failed to estimate gas")

	revertGasLimit := uint64(1000000)

	txs := make([]*types.Transaction, len(txTypes))
	for i, txType := range txTypes {
		tx := types.AccessListTx{}
		switch txType {
		case invalidTx:
			tx = types.AccessListTx{
				To:       counterAddress,
				Gas:      1, // invalid
				Data:     counterInput,
				GasPrice: gasPrice,
			}
			txs[i] = types.NewTx(tests.SetTransactionDefaults(t, net, &tx, senders[i]))
		case failedTx:
			tx = types.AccessListTx{
				To:       &revertAddress,
				Gas:      revertGasLimit,
				Data:     revertInput,
				GasPrice: gasPrice,
			}
			txs[i] = types.NewTx(tests.SetTransactionDefaults(t, net, &tx, senders[i]))
		case successfulNormalTx:
			tx = types.AccessListTx{
				To:       counterAddress,
				Gas:      counterGasLimit,
				Data:     counterInput,
				GasPrice: gasPrice,
			}
			txs[i] = types.NewTx(&tx)
		case successfulSponsoredTx:
			tx = types.AccessListTx{
				To:       counterAddress,
				Gas:      counterGasLimit,
				Data:     counterInput,
				GasPrice: big.NewInt(0),
			}
			txs[i] = types.NewTx(&tx)
		case successfulBundleTx:
			flags := bundle.ExecutionFlag(0)
			btxs, bsenders, _ := makeUnsignedBundleTxs(t, net, client, []int{ /*invalidTx, failedTx,*/ successfulNormalTx, successfulNormalTx}, counterAddress)

			signer := types.NewCancunSigner(net.GetChainId())

			// steps := []bundle.ExecutionStep{}
			// if flags.TolerateInvalid() {
			// 	steps = append(steps, bundle.ExecutionStep{From: senders[0].Address(), Hash: signer.Hash(txs[0])})
			// }
			// if flags.TolerateFailed() {
			// 	steps = append(steps, bundle.ExecutionStep{From: senders[1].Address(), Hash: signer.Hash(txs[1])})
			// }
			// steps = append(steps, bundle.ExecutionStep{From: senders[2].Address(), Hash: signer.Hash(txs[2])})
			// steps = append(steps, bundle.ExecutionStep{From: senders[3].Address(), Hash: signer.Hash(txs[3])})
			steps := []bundle.ExecutionStep{
				{From: bsenders[0].Address(), Hash: signer.Hash(btxs[0])},
				{From: bsenders[1].Address(), Hash: signer.Hash(btxs[1])},
			}
			plan := bundle.ExecutionPlan{Flags: flags, Steps: steps}

			signBundleTxs(t, net, btxs, bsenders, plan)

			// submittedTxs := types.Transactions{}
			// if flags.TolerateInvalid() {
			// 	submittedTxs = append(submittedTxs, txs[0])
			// }
			// if flags.TolerateFailed() {
			// 	submittedTxs = append(submittedTxs, txs[1])
			// }
			// submittedTxs = append(submittedTxs, txs[2], txs[3])
			submittedTxs := btxs

			// bundler := net.GetSessionSponsor()
			bundler := senders[i]
			bundleTx, paymentTxHash := makeBundleTransaction(t, net, submittedTxs, plan, bundler)
			require.NotNil(t, bundleTx)
			require.NotZero(t, paymentTxHash)
			txs[i] = bundleTx
		}
	}

	return txs, senders, *counterAddress
}

func signBundleTxs(t *testing.T, net *tests.IntegrationTestNet, txs []*types.Transaction, senders []*tests.Account, plan bundle.ExecutionPlan) {
	for i, tx := range txs {
		txx := &types.AccessListTx{
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
		if tx.GasPrice().Cmp(big.NewInt(0)) == 0 {
			donation := big.NewInt(1e16)
			gas_subsidies.Fund(t, net, senders[i].Address(), donation)
			txs[i] = gas_subsidies.MakeSponsorRequestTransaction(t, txx, net.GetChainId(), senders[i])
		} else {
			txs[i] = tests.SignTransaction(t, net.GetChainId(), txx, senders[i])
		}
	}
}

func checkHashesEqAndStatus(t *testing.T, net *tests.IntegrationTestNet, expectedHash common.Hash, expectedStatus uint64, txHash func() (common.Hash, bool)) {
	t.Helper()
	hash, ok := txHash()
	if !ok {
		require.Fail(t, "iterator exhausted")
	}
	require.Equal(t, expectedHash, hash, "transaction hash does not match expected hash")
	checkStatus(t, net, expectedStatus, func() (common.Hash, bool) { return hash, true })
}

func checkStatus(t *testing.T, net *tests.IntegrationTestNet, status uint64, txHash func() (common.Hash, bool)) {
	t.Helper()
	hash, ok := txHash()
	if !ok {
		require.Fail(t, "iterator exhausted")
	}
	receipt, err := net.GetReceipt(hash)
	require.NoError(t, err, "failed to get transaction receipt; %v", err)
	require.Equal(t, status, receipt.Status)
}

func Test_Bundle_Ignores_And_AtMostOne_Work(t *testing.T) {
	// transactions = [
	//     if TolerateInvalid: invalidTx,
	//     if TolerateFailed: failedTx,
	//     validTx,
	//     validTx,
	// ]
	// This test ensures that:
	// - if TolerateInvalid is set, invalidTx will be ignored and the rest of the bundle will be executed
	// - if TolerateFailed is set, failedTx will be ignored and the rest of the bundle will be executed
	// - if TryUntil is set, only the first transaction (after ignoring invalid/failed transactions) will be executed
	runWithAllFlags(t, func(
		name string,
		net *tests.IntegrationTestNet,
		client *tests.PooledEhtClient,
		flags bundle.ExecutionFlag,
	) {
		for _, successfulTxType := range []int{successfulNormalTx, successfulSponsoredTx, successfulBundleTx} {
			name := fmt.Sprintf("%s/successfulTxType=%v", name, successfulTxType)
			t.Run(name, func(t *testing.T) {

				txs, senders, counterAddress := makeUnsignedBundleTxs(t, net, client, []int{invalidTx, failedTx, successfulTxType, successfulTxType}, nil)

				signer := types.NewCancunSigner(net.GetChainId())

				steps := []bundle.ExecutionStep{}
				if flags.TolerateInvalid() {
					steps = append(steps, bundle.ExecutionStep{From: senders[0].Address(), Hash: signer.Hash(txs[0])})
				}
				if flags.TolerateFailed() {
					steps = append(steps, bundle.ExecutionStep{From: senders[1].Address(), Hash: signer.Hash(txs[1])})
				}
				steps = append(steps, bundle.ExecutionStep{From: senders[2].Address(), Hash: signer.Hash(txs[2])})
				steps = append(steps, bundle.ExecutionStep{From: senders[3].Address(), Hash: signer.Hash(txs[3])})
				plan := bundle.ExecutionPlan{Flags: flags, Steps: steps}

				signBundleTxs(t, net, txs, senders, plan)

				submittedTxs := types.Transactions{}
				if flags.TolerateInvalid() {
					submittedTxs = append(submittedTxs, txs[0])
				}
				if flags.TolerateFailed() {
					submittedTxs = append(submittedTxs, txs[1])
				}
				submittedTxs = append(submittedTxs, txs[2], txs[3])

				bundler := net.GetSessionSponsor()
				bundleTx, paymentTxHash := makeBundleTransaction(t, net, submittedTxs, plan, bundler)
				require.NotNil(t, bundleTx)
				require.NotZero(t, paymentTxHash)

				err := client.SendTransaction(t.Context(), bundleTx)
				require.NoError(t, err)

				receipt, err := net.GetReceipt(paymentTxHash)
				require.NoError(t, err, "failed to get payment tx receipt; %v", err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "payment transaction failed")

				// Check all transactions have been executed and the order is correct
				transactionHashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
				nextTxHash, stop := iter.Pull(slices.Values(transactionHashes))
				defer stop()
				if successfulTxType == successfulNormalTx {
					if flags.TryUntil() {
						require.Len(t, transactionHashes, 2)
					} else if flags.TolerateFailed() {
						require.Len(t, transactionHashes, 4)
					} else {
						require.Len(t, transactionHashes, 3)
					}

					checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx

					if flags.TolerateFailed() {
						checkHashesEqAndStatus(t, net, txs[1].Hash(), types.ReceiptStatusFailed, nextTxHash) // failedTx
					}

					if !flags.TolerateFailed() || !flags.TryUntil() {
						checkHashesEqAndStatus(t, net, txs[2].Hash(), types.ReceiptStatusSuccessful, nextTxHash) // successfulNormalTx
					}

					if !flags.TryUntil() {
						checkHashesEqAndStatus(t, net, txs[3].Hash(), types.ReceiptStatusSuccessful, nextTxHash) // successfulNormalTx
					}
				} else if successfulTxType == successfulSponsoredTx {
					if flags.TryUntil() {
						if flags.TolerateFailed() {
							require.Len(t, transactionHashes, 2) // paymentTx, failedTx
						} else {
							require.Len(t, transactionHashes, 3) // paymentTx, successfulSponsoredTx, payment for successfulSponsoredTx
						}
					} else if flags.TolerateFailed() {
						require.Len(t, transactionHashes, 6)
					} else {
						require.Len(t, transactionHashes, 5)
					}

					checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx

					if flags.TolerateFailed() {
						checkHashesEqAndStatus(t, net, txs[1].Hash(), types.ReceiptStatusFailed, nextTxHash) // failedTx
					}

					if !flags.TolerateFailed() || !flags.TryUntil() {
						checkHashesEqAndStatus(t, net, txs[2].Hash(), types.ReceiptStatusSuccessful, nextTxHash) // successfulSponsoredTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash)                           // txHash payment for successfulSponsoredTx
					}

					if !flags.TryUntil() {
						checkHashesEqAndStatus(t, net, txs[3].Hash(), types.ReceiptStatusSuccessful, nextTxHash) // successfulSponsoredTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash)                           // txHash payment for successfulSponsoredTx
					}
				} else { // successfulTxType == successfulBundleTx
					if flags.TryUntil() {
						if flags.TolerateFailed() {
							require.Len(t, transactionHashes, 2) // paymentTx, failedTx
						} else {
							require.Len(t, transactionHashes, 4) // paymentTx, inner paymentTx, inner successfulNormalTx, inner successfulNormalTx
						}
					} else if flags.TolerateFailed() {
						require.Len(t, transactionHashes, 8)
					} else {
						require.Len(t, transactionHashes, 7)
					}

					checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx

					if flags.TolerateFailed() {
						checkHashesEqAndStatus(t, net, txs[1].Hash(), types.ReceiptStatusFailed, nextTxHash) // failedTx
					}

					if !flags.TolerateFailed() || !flags.TryUntil() {
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash) // txHash inner paymentTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash) // txHash inner successfulNormalTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash) // txHash inner successfulNormalTx
					}

					if !flags.TryUntil() {
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash) // txHash inner paymentTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash) // txHash inner successfulNormalTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash) // txHash inner successfulNormalTx
					}
				}

				// Check the counter value from the contract
				count := getCounterValue(t, client, counterAddress)
				if flags.TryUntil() {
					if flags.TolerateFailed() {
						require.Equal(t, count, int64(0))
					} else {
						if successfulTxType == successfulBundleTx {
							require.Equal(t, count, int64(2))
						} else {
							require.Equal(t, count, int64(1))
						}
					}
				} else {
					if successfulTxType == successfulBundleTx {
						require.Equal(t, count, int64(4))
					} else {
						require.Equal(t, count, int64(2))
					}
				}
			})
		}
	})
}

func Test_Bundle_ResetIfFailed_Works(t *testing.T) {
	// transactions = [
	//     validTx,
	//     if !TolerateInvalid: invalidTx,
	//     if !TolerateFailed: failedTx,
	// ]
	// This test ensures that:
	// - if TryUntil is set, only the first transaction is executed and the rest of the bundle is ignored
	// - otherwise the successful transaction gets skipped if there is another transaction after it that skips or reverts and this is not ignored
	runWithAllFlags(t, func(
		name string,
		net *tests.IntegrationTestNet,
		client *tests.PooledEhtClient,
		flags bundle.ExecutionFlag,
	) {
		for _, successfulTxType := range []int{successfulNormalTx, successfulSponsoredTx, successfulBundleTx} {
			name := fmt.Sprintf("%s/successfulTxType=%v", name, successfulTxType)
			t.Run(name, func(t *testing.T) {
				txs, senders, counterAddress := makeUnsignedBundleTxs(t, net, client, []int{successfulTxType, invalidTx, failedTx}, nil)

				signer := types.NewCancunSigner(net.GetChainId())

				steps := []bundle.ExecutionStep{}
				steps = append(steps, bundle.ExecutionStep{From: senders[0].Address(), Hash: signer.Hash(txs[0])})
				if !flags.TolerateInvalid() {
					steps = append(steps, bundle.ExecutionStep{From: senders[1].Address(), Hash: signer.Hash(txs[1])})
				}
				if !flags.TolerateFailed() {
					steps = append(steps, bundle.ExecutionStep{From: senders[2].Address(), Hash: signer.Hash(txs[2])})
				}
				plan := bundle.ExecutionPlan{Flags: flags, Steps: steps}

				signBundleTxs(t, net, txs, senders, plan)

				submittedTxs := types.Transactions{}
				submittedTxs = append(submittedTxs, txs[0])
				if !flags.TolerateInvalid() {
					submittedTxs = append(submittedTxs, txs[1])
				}
				if !flags.TolerateFailed() {
					submittedTxs = append(submittedTxs, txs[2])
				}

				bundler := net.GetSessionSponsor()
				bundleTx, paymentTxHash := makeBundleTransaction(t, net, submittedTxs, plan, bundler)
				require.NotNil(t, bundleTx)
				require.NotZero(t, paymentTxHash)

				err := client.SendTransaction(t.Context(), bundleTx)
				require.NoError(t, err)

				receipt, err := net.GetReceipt(paymentTxHash)
				require.NoError(t, err, "failed to get payment tx receipt; %v", err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "payment transaction failed")

				// Check all transactions have been executed and the order is correct
				transactionHashes := getTransactionsInBlock(t, net, receipt.BlockNumber)
				nextTxHash, stop := iter.Pull(slices.Values(transactionHashes))
				defer stop()
				if flags.TryUntil() || flags.TolerateInvalid() && flags.TolerateFailed() {
					if successfulTxType == successfulNormalTx {
						require.Len(t, transactionHashes, 2)
						checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx
						checkHashesEqAndStatus(t, net, txs[0].Hash(), types.ReceiptStatusSuccessful, nextTxHash) // successfulNormalTx
					} else if successfulTxType == successfulSponsoredTx {
						require.Len(t, transactionHashes, 3)
						checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx
						checkHashesEqAndStatus(t, net, txs[0].Hash(), types.ReceiptStatusSuccessful, nextTxHash) // successfulSponsoredTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash)                           // payment for successfulSponsoredTx
					} else { // successfulTxType == successfulBundleTx
						require.Len(t, transactionHashes, 4)
						checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash)                           // inner paymentTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash)                           // inner successfulNormalTx
						checkStatus(t, net, types.ReceiptStatusSuccessful, nextTxHash)                           // inner successfulNormalTx
					}
				} else {
					require.Len(t, transactionHashes, 1)
					checkHashesEqAndStatus(t, net, paymentTxHash, types.ReceiptStatusSuccessful, nextTxHash) // paymentTx
				}

				// Check the transaction status
				if successfulTxType != successfulBundleTx {
					if flags.TryUntil() || (flags.TolerateInvalid() && flags.TolerateFailed()) {
						receipt, err = net.GetReceipt(txs[0].Hash())
						require.NoError(t, err, "failed to get transaction tx 0 receipt; %v", err)
						require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx0 failed")
					}
				}

				// Check the counter value from the contract
				count := getCounterValue(t, client, counterAddress)
				if flags.TryUntil() || (flags.TolerateInvalid() && flags.TolerateFailed()) {
					if successfulTxType == successfulBundleTx {
						require.Equal(t, count, int64(2))
					} else {
						require.Equal(t, count, int64(1))
					}
				} else {
					require.Equal(t, count, int64(0))
				}
			})
		}
	})
}

func runWithAllFlags(t *testing.T, f func(string, *tests.IntegrationTestNet, *tests.PooledEhtClient, bundle.ExecutionFlag)) {
	updates := opera.GetBrioUpgrades()
	updates.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(updates),
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	for _, ignoreInvalid := range []bool{true, false} {
		for _, ignoreFailed := range []bool{true, false} {
			for _, atMostOne := range []bool{true, false} {
				name := fmt.Sprintf("ignoreInvalid=%v/ignoreFailed=%v/atMostOne=%v", ignoreInvalid, ignoreFailed, atMostOne)
				t.Run(name, func(t *testing.T) {
					flags := bundle.ExecutionFlag(0)
					flags.SetTolerateInvalid(ignoreInvalid)
					flags.SetTolerateFailed(ignoreFailed)
					flags.SetTryUntil(atMostOne)
					f(name, net, client, flags)
				})
			}
		}
	}
}

// makeBundleTransaction creates a bundle transaction with the given transactions and execution plan
// This function will create the corresponding payment transaction. Both payment and the bundle transaction
// are signed by the bundler account.
// It returns the bundle transaction and the hash of the payment transaction, the later is used
// for waiting on the completion of the bundle execution, as the bundle transaction will not be included
// in a block.
func makeBundleTransaction(t *testing.T,
	net *tests.IntegrationTestNet,
	transactions types.Transactions,
	plan bundle.ExecutionPlan,
	bundler *tests.Account) (*types.Transaction, common.Hash) {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	sameNonceForBundleAndPayment, err := client.PendingNonceAt(t.Context(), bundler.Address())
	require.NoError(t, err, "failed to get nonce for bundler; %v", err)

	cost := big.NewInt(0)
	for _, tx := range transactions {
		txCost := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice())
		cost = new(big.Int).Add(cost, txCost)
	}

	// make payment transaction
	paymentTx := tests.CreateTransaction(t, net,
		&types.AccessListTx{Nonce: sameNonceForBundleAndPayment,
			To:    &common.Address{0x01},
			Value: cost,
			AccessList: types.AccessList{
				{Address: bundle.BundleOnly, StorageKeys: []common.Hash{plan.Hash()}},
			}}, bundler)

	var gas uint64
	for _, tx := range append(transactions, paymentTx) {
		gas += tx.Gas()
	}

	bundlePayload := bundle.TransactionBundle{
		Version: bundle.BundleV1,
		Bundle:  transactions,
		Payment: paymentTx,
		Flags:   plan.Flags,
	}

	// create the bundle transaction with the same nonce as the payment transaction
	bundleTx := tests.CreateTransaction(t, net,
		&types.LegacyTx{Nonce: sameNonceForBundleAndPayment,
			To:   &bundle.BundleAddress,
			Gas:  gas,
			Data: bundle.Encode(bundlePayload),
		}, bundler)

	// Sanity check the bundle before sending it to the mempool, if fails to validate before making
	// a bundle transaction, it will fail to be included in a block and waiting for payment receipt will timeout
	upgrades := net.GetUpgrades()
	signer := types.NewCancunSigner(net.GetChainId())
	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err, "failed to suggest gas price; %v", err)
	require.NoError(t, bundle.ValidateTransactionBundle(bundleTx, bundlePayload, signer, gasPrice, upgrades))

	return bundleTx, paymentTx.Hash()
}

func prepareContract[T any](
	t testing.TB, net *tests.IntegrationTestNet,
	getABI func() (*abi.ABI, error),
	deployFunc tests.ContractDeployer[T],
) (*T, *abi.ABI, common.Address) {
	t.Helper()
	abi, err := getABI()
	require.NoError(t, err, "failed to get counter abi; %v", err)

	contract, receipt, err := tests.DeployContract(net, deployFunc)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
	return contract, abi, receipt.ContractAddress
}

func generateCallData(t testing.TB, abi *abi.ABI, methodName string, params ...any) []byte {
	t.Helper()
	input, err := abi.Pack(methodName, params...)
	require.NoError(t, err, "failed to pack input for method %s; %v", methodName, err)
	return input
}

func getTransactionsInBlock(t *testing.T, net *tests.IntegrationTestNet, blockNumber *big.Int) []common.Hash {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	block, err := client.BlockByNumber(t.Context(), blockNumber)
	require.NoError(t, err, "failed to get block by number")

	hashes := make([]common.Hash, 0, len(block.Transactions()))
	for _, btx := range block.Transactions() {
		hashes = append(hashes, btx.Hash())
	}
	return hashes
}
