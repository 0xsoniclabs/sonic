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
	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/0xsoniclabs/sonic/tests/contracts/selfdestruct"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBundles_ContractExecutionsAreRevertedInCaseOfBundleRevert(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())

	senders := tests.MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))
	creator := senders[0]
	fallbackSender := senders[1]

	// Determine the nonce the contract creation tx will use so we can
	// pre-compute the address that would be created.
	creatorNonce, err := client.PendingNonceAt(t.Context(), creator.Address())
	require.NoError(t, err)
	expectedContractAddr := crypto.CreateAddress(creator.Address(), creatorNonce)

	// Use the counter contract bytecode for the creation transaction.
	creationData := common.FromHex(counter.CounterBin)

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Build a bundle using OneOf:
	//   OneOf(
	//     AllOf(contract_creation, invalid_tx),  <-- this group fails
	//     fallback_tx,                           <-- this succeeds
	//   )
	// The AllOf group reverts because the invalid tx fails, rolling
	// back the contract creation. The fallback ensures the bundle itself
	// lands on-chain so we can inspect state at that block.
	envelope, _, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		OneOf(
			bundle.AllOf(
				Step(t, net, creator, &types.AccessListTx{
					To:   nil, // contract creation
					Data: creationData,
					Gas:  1_000_000,
				}),
				Step(t, net, creator, &types.AccessListTx{
					Nonce: 0, // invalid nonce to cause tx failure
					Gas:   1_000_000,
				}),
			),
			Step(t, net, fallbackSender, &types.AccessListTx{}),
		).
		BuildEnvelopeBundleAndPlan()

	// Submit the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), envelope))

	// Wait for the bundle to be executed.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// Verify the contract does not exist at the block where the bundle landed.
	code, err := client.CodeAt(
		t.Context(),
		expectedContractAddr,
		big.NewInt(info.Block.Int64()),
	)
	require.NoError(t, err)
	require.Empty(t, code, "contract should not exist after bundle revert")
}

func TestBundles_TransactionsInBundleAreSeparateTransactions(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Determine the address of the contract that will be created.
	nonce, err := client.PendingNonceAt(t.Context(), sender.Address())
	require.NoError(t, err)
	contractAddr := crypto.CreateAddress(sender.Address(), nonce)

	// Use the selfdestruct contract bytecode for the creation transaction.
	beneficiary := common.Address{0x42}
	parsedABI, err := selfdestruct.SelfDestructMetaData.GetAbi()
	require.NoError(t, err)
	constructorArgs, err := parsedABI.Pack("", false, false, beneficiary)
	require.NoError(t, err)
	deployData := append(common.FromHex(selfdestruct.SelfDestructBin), constructorArgs...)

	// Prepare input for destroyContract call.
	destroyInput := tests.MustGetMethodParameters(t,
		selfdestruct.SelfDestructMetaData,
		"destroyContract",
		false, beneficiary,
	)

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Build the two transactions manually so we can assign consecutive
	// nonces for the same sender. The Step helper would query
	// PendingNonceAt for each call and get the same nonce for both.
	deployTxData := tests.SetTransactionDefaults(t, net, &types.AccessListTx{
		To:   nil, // contract creation
		Data: deployData,
		Gas:  1_000_000,
	}, sender)

	destroyTxData := *deployTxData
	destroyTxData.Nonce = deployTxData.Nonce + 1
	destroyTxData.To = &contractAddr
	destroyTxData.Data = destroyInput

	// Bundle: deploy contract, then call destroyContract on it.
	envelope, _, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		AllOf(
			bundle.Step(sender.PrivateKey, deployTxData),
			bundle.Step(sender.PrivateKey, &destroyTxData),
		).
		BuildEnvelopeBundleAndPlan()

	require.NoError(t, client.SendTransaction(t.Context(), envelope))

	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// Selfdestruct in a different transaction than creation does not destroy the contract.
	atBlock := big.NewInt(info.Block.Int64())
	code, err := client.CodeAt(t.Context(), contractAddr, atBlock)
	require.NoError(t, err)
	require.NotEmpty(t, code,
		"contract code should NOT be deleted after selfdestruct in a different transaction")
}
