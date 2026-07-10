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

package priority

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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestPriority_ValidatorEmittingNonOwnedTxIsNotSlashed verifies that a
// validator which force-emits a transaction it does not own (per the
// emitter's isMyTxTurn round-owner rotation) is never slashed by the SFC,
// whether or not TransactionPriorities is active and whether or not the
// transaction's sender is registered as prioritized.
//
// The sender is chosen deterministically so that node 0 is not the natural
// round-0 owner of the transaction, and the transaction is then delivered via
// Session.ForceEmit so it bypasses the transaction pool and the isMyTxTurn
// check itself, mirroring how a misbehaving proposer might inject a
// transaction outside its own turn.
func TestPriority_ValidatorEmittingNonOwnedTxIsNotSlashed(t *testing.T) {
	require := require.New(t)
	const numNodes = 3

	testCases := map[string]struct {
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			upgrades := opera.GetBrioUpgrades()
			upgrades.TransactionPriorities = tc.enablePriorities

			net := tests.StartIntegrationTestNetWithJsonGenesis(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrades,
				NumNodes: numNodes,
			})

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			signer := types.LatestSignerForChainID(net.GetChainId())

			sfcContract, err := sfc100.NewContract(sfc.ContractAddress, client)
			require.NoError(err)
			epochBig, err := sfcContract.CurrentEpoch(nil)
			require.NoError(err)
			epoch := idx.Epoch(epochBig.Uint64())

			// Pick a fresh sender such that, under the emitter's isMyTxTurn rotation,
			// node 0 is not the round-0 owner of its nonce-0 transaction. Node 0 uses
			// FakeKey(1) and therefore has validator ID 1, which is index 0 in the
			// equal-weight, ID-sorted validator list produced by the fake genesis.
			// Guaranteeing perm[0] != 0 mirrors "another validator would naturally
			// originate this tx", making the force-emit exercise the non-owned path.
			const node0Index = 0
			sender := pickSenderNotOwnedByValidatorIndex(t, epoch, numNodes, node0Index)
			receipt, err := net.EndowAccount(sender.Address(), big.NewInt(1e18))
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

			// If priorities are enabled and the case wants the transaction prioritized, register
			// its sender in the on-chain priority registry.
			if tc.enablePriorities && tc.prioritizeTx {
				setPrioritized(t, net, sender.Address(), 1, 1, common.Hash{0xaa})
			}

			// Create a signed transaction from the sender.
			sink := common.Address{0x99}
			gasPrice, err := client.SuggestGasPrice(t.Context())
			require.NoError(err)
			tx := types.MustSignNewTx(sender.PrivateKey, signer, &types.LegacyTx{
				Nonce:    0,
				To:       &sink,
				Value:    big.NewInt(1),
				Gas:      21000,
				GasPrice: gasPrice,
			})

			// Force node 0 to emit the transaction directly, bypassing the pool and
			// the isMyTxTurn round-owner check. By construction (see
			// pickSenderNotOwnedByValidatorIndex above), node 0 is not the natural
			// round-0 owner of this tx, so this exercises the "validator emits a
			// non-owned transaction" path.
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
				require.False(slashed, "validator %d must not be slashed", i)

				info, err := sfcContract.GetValidator(nil, vid)
				require.NoError(err)
				require.EqualValues(0, info.Status.Uint64(),
					"validator %d must remain active (status=0), got %d", i, info.Status.Uint64())
			}

			// All nodes must agree on the head block.
			requireAllNodesAgreeOnHead(t, net)
		})
	}
}

// pickSenderNotOwnedByValidatorIndex returns a fresh account whose nonce-0
// transaction is not "owned" by the validator at ownerIndex in the sorted
// validator list, according to the emitter's isMyTxTurn rotation (see
// gossip/emitter/txs.go). The fake genesis assigns validator IDs 1..numNodes
// with equal stakes, so SortedIDs is [1..numNodes] and the actual weight value
// does not affect the resulting permutation.
func pickSenderNotOwnedByValidatorIndex(
	t *testing.T,
	epoch idx.Epoch,
	numNodes int,
	ownerIndex int,
) *tests.Account {
	t.Helper()
	ids := make([]idx.ValidatorID, numNodes)
	for i := range ids {
		ids[i] = idx.ValidatorID(i + 1)
	}
	validators := pos.EqualWeightValidators(ids, 1)
	weights := validators.SortedWeights()
	// Nonce is 0, so accountNonce/txTurnNonces is 0 regardless of the
	// (package-private) txTurnNonces constant.
	nonceBytes := bigendian.Uint64ToBytes(0)
	epochBytes := epoch.Bytes()
	for {
		candidate := tests.NewAccount()
		addr := candidate.Address()
		roundsHash := hash.Of(addr.Bytes(), nonceBytes, epochBytes)
		perm := utils.WeightedPermutation(int(validators.Len()), weights, roundsHash)
		if perm[0] != ownerIndex {
			return candidate
		}
	}
}
