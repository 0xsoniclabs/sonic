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
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestJudged_ARivalTheOrderingStepTookAndExecutionDroppedLeavesItsSurvivorUnjudged is the case a CI
// run of the property found: bob draws two transactions for his nonce 0, which the registry answers
// for once, so both carry the maximum level. The ordering step is given both, takes the one with the
// lower hash into bob's sequence and hoists it ahead of alice's, and execution then drops it for what
// it costs -- leaving a block that begins with alice's transaction and bob's survivor behind it,
// which the model called a violation as long as it judged the survivors alone.
func TestJudged_ARivalTheOrderingStepTookAndExecutionDroppedLeavesItsSurvivorUnjudged(t *testing.T) {
	taken := tx(1, bob, 0, 110_592, prio(math.MaxUint64, 0, 1))
	survivor := tx(2, bob, 0, 51_060, prio(math.MaxUint64, 0, 1))
	alices := tx(3, alice, 0, 105_448, prio(2, 1, 0))

	starts := map[common.Address]uint64{alice: 0, bob: 0}

	blind := PrioritizedPrefix([]BlockTx{alices, survivor}, starts, perEntityBudget)
	require.Equal(t, []common.Hash{survivor.Hash, alices.Hash}, hashesInOrder(blind),
		"judging what a block kept alone is what expected a hoist the ordering step never made")

	candidates := []BlockTx{alices, survivor, taken}
	expected := PrioritizedPrefix(candidates, starts, perEntityBudget)
	require.Equal(t, []common.Hash{taken.Hash, alices.Hash}, hashesInOrder(expected),
		"the rival with the lower hash is the one bob's sequence was built from")

	contested := contestedSlots(candidates)
	require.Equal(t, map[slot]bool{{bob, 0}: true}, contested,
		"a slot two of the batch's transactions claim is not the model's to decide")

	// The block as the failing run reported it: alice's transaction, and bob's survivor behind it.
	mine := []BlockTx{alices, survivor}
	want, got, at := judged(mine, []int{0, 1}, expected, hashesOf(mine), contested)

	require.Equal(t, []common.Hash{alices.Hash}, hashesInOrder(want))
	require.Equal(t, []common.Hash{alices.Hash}, hashesInOrder(got))
	require.Equal(t, []int{0}, at)
	require.Equal(t, hashesInOrder(want), hashesInOrder(got[:len(want)]),
		"what is left to judge is in the order the rules call for")
}

func hashesInOrder(txs []BlockTx) []common.Hash {
	out := make([]common.Hash, 0, len(txs))
	for _, tx := range txs {
		out = append(out, tx.Hash)
	}
	return out
}
