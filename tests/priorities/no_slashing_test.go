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

package priorities

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/contract/sfc100"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriorities_ValidatorEmittingTxWhenNotItsTurnIsNotSlashed(t *testing.T) {
	const numNodes = 3

	cases := map[string]struct {
		enablePriorities bool
		prioritizeTx     bool
	}{
		"priorities disabled": {
			enablePriorities: false,
			prioritizeTx:     false,
		},
		"priorities enabled, tx not prioritized": {
			enablePriorities: true,
			prioritizeTx:     false,
		},
		"priorities enabled, tx prioritized": {
			enablePriorities: true,
			prioritizeTx:     true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			upgrades := opera.GetBrioUpgrades()
			upgrades.TransactionPriorities = tc.enablePriorities

			net := tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrades,
				NumNodes: numNodes,
			})

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			sfcContract, err := sfc100.NewContract(sfc.ContractAddress, client)
			require.NoError(err)
			epochBig, err := sfcContract.CurrentEpoch(nil)
			require.NoError(err)
			epoch := idx.Epoch(epochBig.Uint64())

			const node0Index = 0
			sender := pickSenderNotInTurnOfValidatorIndex(t, epoch, numNodes, node0Index)
			receipt, err := net.EndowAccount(sender.Address(), big.NewInt(1e18))
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

			if tc.enablePriorities && tc.prioritizeTx {
				setPrioritized(t, net, sender.Address(), 1, 1, 0xaa)
			}

			tx := newSignedTx(t, net, sender, 0, 1, 21000, nil)

			// Force node 0 to emit the transaction directly, bypassing the pool
			// and the isMyTxTurn check. By construction above, it is not node
			// 0's turn in round 0 for this tx, so this exercises the
			// "validator emits a transaction when it is not its turn" path.
			hash, err := net.ForceEmit(t.Context(), tx)
			require.NoError(err)
			require.Equal(tx.Hash(), hash)

			receipt, err = net.GetReceipt(tx.Hash())
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

			// Advance a couple of epochs so any slashing decision has been finalized.
			net.AdvanceEpoch(t, 2)

			// No validator must have been slashed and all must remain active.
			for i := 1; i <= numNodes; i++ {
				vid := big.NewInt(int64(i))

				slashed, err := sfcContract.IsSlashed(nil, vid)
				require.NoError(err)
				require.False(slashed)

				info, err := sfcContract.GetValidator(nil, vid)
				require.NoError(err)
				require.Zero(info.Status.Uint64())
			}
		})
	}
}

// pickSenderNotInTurnOfValidatorIndex returns a fresh account whose nonce-0
// transaction is not the turn of the validator at turnIndex in the sorted
// validator list, according to the emitter's isMyTxTurn rotation (see
// gossip/emitter/txs.go). The fake genesis assigns validator IDs 1..numNodes
// with equal stakes, so SortedIDs is [1..numNodes] and the actual weight value
// does not affect the resulting permutation.
func pickSenderNotInTurnOfValidatorIndex(
	t *testing.T,
	epoch idx.Epoch,
	numNodes int,
	turnIndex int,
) *tests.Account {
	t.Helper()
	ids := make([]idx.ValidatorID, numNodes)
	for i := range ids {
		ids[i] = idx.ValidatorID(i + 1)
	}
	validators := pos.EqualWeightValidators(ids, 1)
	weights := validators.SortedWeights()
	nonceBytes := bigendian.Uint64ToBytes(0)
	epochBytes := epoch.Bytes()
	for i := 0; i < 1024; i++ {
		candidate := tests.NewAccount()
		addr := candidate.Address()
		roundsHash := hash.Of(addr.Bytes(), nonceBytes, epochBytes)
		perm := utils.WeightedPermutation(int(validators.Len()), weights, roundsHash)
		if perm[0] != turnIndex {
			return candidate
		}
	}
	t.Fatalf("failed to find sender not in turn (turnIndex=%d, numNodes=%d, epoch=%d)", turnIndex, numNodes, epoch)
	return nil
}
