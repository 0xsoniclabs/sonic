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
	"encoding/binary"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestPriorities_PerEntityGasLimitCapsThePrioritizedPrefixOfEveryBlock(t *testing.T) {
	const (
		gasPerTx    = 21_000
		txsInBudget = 5

		numEntities  = 5
		txsPerEntity = txsInBudget + 1
	)

	require := require.New(t)

	net, client, signer := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	setPriorityConfig(t, net, txsInBudget*gasPerTx, math.MaxUint64)

	// One sender per (entity, weight) with a single transaction each, so the
	// selection is decided by the gas budget alone and is fully determined: of
	// an entity's transactions in a block, the txsInBudget ones with the highest
	// weight fit its budget, the rest is demoted.
	prioBySender := map[common.Address]priorities.Priority{}
	prioritized := make(types.Transactions, 0, numEntities*txsPerEntity)
	for entity := uint64(1); entity <= numEntities; entity++ {
		for weight := uint64(1); weight <= txsPerEntity; weight++ {
			account := newFundedPrioritizedAccount(t, net, 1, weight, entity)
			prio := priorities.Priority{Level: 1, Weight: weight}
			binary.BigEndian.PutUint64(prio.ID[8:], entity)
			prioBySender[account.Address()] = prio
			prioritized = append(prioritized, newSignedTx(t, net, account, 0, 1, gasPerTx, nil))
		}
	}

	ordinary := buildOrdinaryTraffic(t, net, 20, 5)

	hashes := sendShuffledToPool(t, net, slices.Concat(ordinary, prioritized))
	first, last := blockRange(requireSuccessfulReceipts(t, net, hashes))

	demotedAfterOrdinary := false
	for number := first; number <= last; number++ {
		block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(number))
		require.NoError(err)

		seenOfEntity := map[priorities.PriorityID]uint64{}
		lastFittingWeightOfEntity := map[priorities.PriorityID]uint64{}
		seenOrdinary := false
		for _, tx := range block.Transactions() {
			if internaltx.IsInternal(tx) {
				continue
			}
			from, err := types.Sender(signer, tx)
			require.NoError(err)
			prio := prioBySender[from]
			if !prio.IsPrioritized() {
				seenOrdinary = true
				continue
			}
			if lastFitting, ok := lastFittingWeightOfEntity[prio.ID]; ok {
				require.Less(prio.Weight, lastFitting)
			}
			if seenOfEntity[prio.ID] < txsInBudget {
				require.False(seenOrdinary)
				lastFittingWeightOfEntity[prio.ID] = prio.Weight
			} else {
				demotedAfterOrdinary = demotedAfterOrdinary || seenOrdinary
			}
			seenOfEntity[prio.ID]++
		}
	}

	require.True(demotedAfterOrdinary)
}
