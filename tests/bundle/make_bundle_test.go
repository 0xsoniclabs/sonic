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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type Case struct {
	tryUntil         bool
	tolerateFailed   bool
	tolerateInvalid  bool
	submittedTxTypes []int
	blockTxs         []int // index of tx hash of submitted transactions, -1 for paymentTx, -2 for unchecked transactions
	blockTxStatuses  []uint64
	counter          int64
}

const (
	successfulNormalTx    = 0
	successfulSponsoredTx = 1
	successfulBundleTx    = 2
	failedTx              = 3
	invalidTx             = 4
)

const (
	paymentTxIndex   = -1
	uncheckedTxIndex = -2
)

const (
	successStatus = types.ReceiptStatusSuccessful
	failedStatus  = types.ReceiptStatusFailed
)

func Test_RunAllUnlessNotTolerated_Works(t *testing.T) {
	cases := []Case{
		{false, false, false,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1, 2},
			[]uint64{successStatus, successStatus, successStatus, successStatus},
			3,
		},
		{false, false, false,
			[]int{successfulNormalTx, failedTx, successfulNormalTx},
			[]int{paymentTxIndex},
			[]uint64{successStatus},
			0,
		},
		{false, false, false,
			[]int{successfulNormalTx, invalidTx, successfulNormalTx},
			[]int{paymentTxIndex},
			[]uint64{successStatus},
			0,
		},
		// TolerateInvalid
		{false, false, true,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1, 2},
			[]uint64{successStatus, successStatus, successStatus, successStatus},
			3,
		},
		{false, false, true,
			[]int{successfulNormalTx, failedTx, successfulNormalTx},
			[]int{paymentTxIndex},
			[]uint64{successStatus},
			0,
		},
		{false, false, true,
			[]int{successfulNormalTx, invalidTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 2},
			[]uint64{successStatus, successStatus, successStatus},
			2,
		},
		// TolerateFailed
		{false, true, false,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1, 2},
			[]uint64{successStatus, successStatus, successStatus, successStatus},
			3,
		},
		{false, true, false,
			[]int{successfulNormalTx, failedTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1, 2},
			[]uint64{successStatus, successStatus, failedStatus, successStatus},
			2,
		},
		{false, true, false,
			[]int{successfulNormalTx, invalidTx, successfulNormalTx},
			[]int{paymentTxIndex},
			[]uint64{successStatus},
			0,
		},
		// TolerateFailed & TolerateInvalid
		{false, true, true,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1, 2},
			[]uint64{successStatus, successStatus, successStatus, successStatus},
			3,
		},
		{false, true, true,
			[]int{successfulNormalTx, failedTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1, 2},
			[]uint64{successStatus, successStatus, failedStatus, successStatus},
			2,
		},
		{false, true, true,
			[]int{successfulNormalTx, invalidTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 2},
			[]uint64{successStatus, successStatus, successStatus},
			2,
		},
	}
	net, client := startTestnet(t)
	defer client.Close()
	for _, c := range cases {
		checkCase(t, net, client, c)
	}
}

func Test_RunUntilTolerated_Works(t *testing.T) {
	cases := []Case{
		{true, false, false,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0},
			[]uint64{successStatus, successStatus},
			1,
		},
		{true, false, false,
			[]int{failedTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1},
			[]uint64{successStatus, failedStatus, successStatus},
			1,
		},
		{true, false, false,
			[]int{invalidTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 1},
			[]uint64{successStatus, successStatus},
			1,
		},
		// TolerateInvalid
		{true, false, true,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0},
			[]uint64{successStatus, successStatus},
			1,
		},
		{true, false, true,
			[]int{failedTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0, 1},
			[]uint64{successStatus, failedStatus, successStatus},
			1,
		},
		{true, false, true,
			[]int{invalidTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex},
			[]uint64{successStatus},
			0,
		},
		// TolerateFailed
		{true, true, false,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0},
			[]uint64{successStatus, successStatus},
			1,
		},
		{true, true, false,
			[]int{failedTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0},
			[]uint64{successStatus, failedStatus},
			0,
		},
		{true, true, false,
			[]int{invalidTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 1},
			[]uint64{successStatus, successStatus},
			1,
		},
		// TolerateFailed & TolerateInvalid
		{true, true, true,
			[]int{successfulNormalTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0},
			[]uint64{successStatus, successStatus},
			1,
		},
		{true, true, true,
			[]int{failedTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex, 0},
			[]uint64{successStatus, failedStatus},
			0,
		},
		{true, true, true,
			[]int{invalidTx, successfulNormalTx, successfulNormalTx},
			[]int{paymentTxIndex},
			[]uint64{successStatus},
			0,
		},
	}
	net, client := startTestnet(t)
	defer client.Close()
	for _, c := range cases {
		checkCase(t, net, client, c)
	}
}

func checkCase(t *testing.T, net *tests.IntegrationTestNet, client *tests.PooledEhtClient, c Case) {
	name := fmt.Sprintf("TryUntil=%v/TolerateFailed=%v/TolerateInvalid=%v", c.tryUntil, c.tolerateFailed, c.tolerateInvalid)
	t.Run(name, func(t *testing.T) {
		flags := bundle.ExecutionFlag(0)
		flags.SetTolerateInvalid(c.tolerateInvalid)
		flags.SetTolerateFailed(c.tolerateFailed)
		flags.SetTryUntil(c.tryUntil)

		txs, senders, counterAddress := makeUnsignedBundleTxs(t, net, client, c.submittedTxTypes, nil)

		signer := types.NewCancunSigner(net.GetChainId())

		steps := make([]bundle.ExecutionStep, len(txs))
		for i, tx := range txs {
			steps[i] = bundle.ExecutionStep{From: senders[i].Address(), Hash: signer.Hash(tx)}
		}
		plan := bundle.ExecutionPlan{Flags: flags, Steps: steps}

		signBundleTxs(t, net, txs, senders, plan)

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
		for i, _ := range c.blockTxs {
			if c.blockTxs[i] == paymentTxIndex {
				checkHashesEqAndStatus(t, net, paymentTxHash, c.blockTxStatuses[i], transactionHashes[i])
			} else if c.blockTxs[i] == uncheckedTxIndex {
				checkStatus(t, net, c.blockTxStatuses[i], transactionHashes[i])
			} else {
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
		if tx.GasPrice().Cmp(big.NewInt(0)) == 0 {
			donation := big.NewInt(1e16)
			gas_subsidies.Fund(t, net, senders[i].Address(), donation)
			txs[i] = gas_subsidies.MakeSponsorRequestTransaction(t, bundleOnlyTx, net.GetChainId(), senders[i])
		} else {
			txs[i] = tests.SignTransaction(t, net.GetChainId(), bundleOnlyTx, senders[i])
		}
	}
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
