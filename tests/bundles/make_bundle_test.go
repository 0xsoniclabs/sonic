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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// makeBundleTransaction creates a bundle transaction with the given
// transactions and execution plan. The bundle transaction is signed by the
// bundler account.
func makeBundleTransaction(
	t *testing.T,
	net *tests.IntegrationTestNet,
	transactions types.Transactions,
	plan bundle.ExecutionPlan,
	nested bool,
) *types.Transaction {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client; %v", err)
	defer client.Close()

	// Create a dedicated coordinator for every bundle.
	coordinator := tests.NewAccount()

	cost := big.NewInt(0)
	for _, tx := range transactions {
		txCost := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice())
		cost = new(big.Int).Add(cost, txCost)
	}

	var gas uint64
	for _, tx := range transactions {
		gas += tx.Gas()
	}

	bundlePayload := bundle.TransactionBundle{
		Version:  bundle.BundleV1,
		Bundle:   transactions,
		Flags:    plan.Flags,
		Earliest: plan.Earliest,
		Latest:   plan.Latest,
	}

	data := bundle.Encode(bundlePayload)

	// If this envelope is nested, meaning it is part of a super/parent bundle,
	// its gas cost shall include enough intrinsics gas for one extra access list entry
	// (the bundleOnly marker) because the parent bundle will add the marker
	// to the access list of the bundle transaction.
	accessList := []types.AccessTuple{}
	if nested {
		accessList = []types.AccessTuple{
			{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{}}},
		}
	}
	intrGas, err := core.IntrinsicGas(
		data,
		accessList,
		nil,   // code auth is not used in the bundle transaction
		false, // bundle transaction is not a contract creation
		true,
		true,
		true,
	)
	require.NoError(t, err, "failed to calculate intrinsic gas; %v", err)

	// EIP-7623 part of Prague revision: Floor data gas
	// see: https://eips.ethereum.org/EIPS/eip-7623
	floorDataGas, err := core.FloorDataGas(data)
	require.NoError(t, err, "failed to calculate floor data gas; %v", err)

	// create the bundle transaction with the same nonce as the payment transaction
	signer := types.LatestSignerForChainID(net.GetChainId())
	bundleTx := types.MustSignNewTx(coordinator.PrivateKey, signer,
		&types.AccessListTx{
			Nonce: 0,
			To:    &bundle.BundleAddress,
			Gas:   max(gas, intrGas, floorDataGas),
			Data:  data,
		},
	)

	// If the bundle transaction is nested, it does not pass validation at this point
	// because its gas already accounts for the presence of the bundleOnly marker,
	// but the marker is not yet added to the access list.
	// This is because we need the transaction without the marker to compute the
	// execution plan of the super/parent bundle.
	if !nested {
		// Sanity check the bundle before sending it to the mempool, if fails to validate before making
		// a bundle transaction, it will fail to be included in a block and waiting for payment receipt will timeout
		_, _, err = bundle.ValidateTransactionBundle(bundleTx, signer)
		require.NoError(t, err, "failed to validate transaction bundle; %v", err)
	}

	return bundleTx
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
