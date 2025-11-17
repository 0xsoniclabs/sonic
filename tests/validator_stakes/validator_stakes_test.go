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

package many

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/contract/sfc100"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestValidatorsStakes_AllNodesProduceBlocks_WhenStakeDistributionChanges(t *testing.T) {

	// Start a network with many nodes where two nodes dominate the stake
	initialStake := []uint64{
		750, 750, // 75% of stake
		125, 125, 125, 125, // 25% of stake
	}
	net := tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
		ValidatorsStake: initialStake,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sfcContract, err := sfc100.NewContract(sfc.ContractAddress, client)
	require.NoError(t, err)

	validators, firstEpoch := getValidatorsInCurrentEpoch(t, sfcContract)
	t.Run("first epoch is dominated by two validators",
		func(t *testing.T) {
			requireValidatorStakesEqualTo(t, validators, []uint64{750, 750, 125, 125, 125, 125})
			requireAllNodesReachSameBlockHeight(t, net)
		})

	t.Run("second epoch has equal stake for all validators",
		func(t *testing.T) {
			// Prepare new stake distribution
			for _, id := range []idx.ValidatorID{3, 4, 5, 6} {
				increaseValidatorStake(t, net, sfcContract, id, utils.ToFtm(625))
			}
			net.AdvanceEpoch(t, 1)

			// Test new stake distribution
			validators, secondEpoch := getValidatorsInCurrentEpoch(t, sfcContract)
			require.Equal(t, firstEpoch.Uint64()+1, secondEpoch.Uint64(),
				"epoch did not advance as expected")
			requireValidatorStakesEqualTo(t, validators, []uint64{750, 750, 750, 750, 750, 750})
			requireAllNodesReachSameBlockHeight(t, net)
		})

	t.Run("third epoch is dominated by two validators again",
		func(t *testing.T) {
			// Prepare new stake distribution
			for _, id := range []idx.ValidatorID{5, 6} {
				increaseValidatorStake(t, net, sfcContract, id, utils.ToFtm(1_000_000))
			}
			net.AdvanceEpoch(t, 1)

			// Test new stake distribution
			validators, secondEpoch := getValidatorsInCurrentEpoch(t, sfcContract)

			require.Equal(t, firstEpoch.Uint64()+2, secondEpoch.Uint64(),
				"epoch did not advance as expected")
			requireValidatorStakesEqualTo(t, validators, []uint64{750, 750, 750, 750, 1000750, 1000750})
			requireAllNodesReachSameBlockHeight(t, net)
		})
}

func increaseValidatorStake(t *testing.T, net *tests.IntegrationTestNet, sfcContract *sfc100.Contract, id idx.ValidatorID, amount *big.Int) {
	t.Helper()

	// First endow the validator account with enough FTM to stake.
	// If this step is not done, the account calling sfc.Delegate will be
	// delegating into the stake. The interaction with the sfc is simpler
	// if there is only one delegator, which is the validator account itself.
	account := tests.Account{
		PrivateKey: makefakegenesis.FakeKey(id),
	}
	receipt, err := net.EndowAccount(account.Address(), amount)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Then the valdator account delegates stake to its validatorID.
	opts, err := net.GetTransactOptions(&account)
	require.NoError(t, err)
	opts.Value = amount
	tx, err := sfcContract.Delegate(opts, big.NewInt(int64(id)))
	require.NoError(t, err)

	receipt, err = net.GetReceipt(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

// requireValidatorStakesEqualTo checks that the stakes of the validators
// in the 'validators' map match the expected stakes provided in 'expectedStakes' slice.
// The index of the expectedStakes slice corresponds to the ValidatorID - 1.
func requireValidatorStakesEqualTo(t *testing.T, validators map[idx.ValidatorID]*big.Int, expectedStakes []uint64) {
	t.Helper()

	for id, expectedStake := range expectedStakes {
		stake, ok := validators[idx.ValidatorID(id+1)]
		require.True(t, ok, "validator %d not found", id+1)
		expectedStakeInTokens := utils.ToFtm(expectedStake)
		require.Equal(t, expectedStakeInTokens.Uint64(), stake.Uint64(), "validator %d has incorrect stake", id+1)
	}
}

// getValidatorsInCurrentEpoch retrieves the current epoch number and a map of validator IDs to their stakes
// from the provided sfcContract.
func getValidatorsInCurrentEpoch(t *testing.T, sfcContract *sfc100.Contract) (map[idx.ValidatorID]*big.Int, *big.Int) {
	t.Helper()

	epoch, err := sfcContract.CurrentEpoch(nil)
	require.NoError(t, err)

	ids, err := sfcContract.GetEpochValidatorIDs(nil, epoch)
	require.NoError(t, err)
	validators := make(map[idx.ValidatorID]*big.Int)
	for _, bigId := range ids {

		id := idx.ValidatorID(bigId.Uint64())
		key := makefakegenesis.FakeKey(id)
		delegator := crypto.PubkeyToAddress(key.PublicKey)
		stake, err := sfcContract.GetStake(nil, delegator, bigId)
		require.NoError(t, err)

		validators[id] = stake
	}

	return validators, epoch
}

// requireAllNodesReachSameBlockHeight checks that all nodes in the provided
// IntegrationTestNet have reached the same block height.
// In combination with epoch advancement, this ensures that all nodes are producing blocks.
func requireAllNodesReachSameBlockHeight(t *testing.T, net *tests.IntegrationTestNet) {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// retrieve last produced block number from the first node
	number, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	for i := 1; i < net.NumNodes(); i++ {
		nodeClient, err := net.GetClientConnectedToNode(i)
		require.NoError(t, err)
		defer nodeClient.Close()

		// all other nodes should reach the same block number
		tests.WaitForProofOf(t, nodeClient, int(number))
	}
}
