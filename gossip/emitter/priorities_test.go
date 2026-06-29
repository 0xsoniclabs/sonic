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

package emitter

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type fakePriorityClassifier struct {
	byHash map[common.Hash]priorities.Priority
}

func (c fakePriorityClassifier) Priority(tx *types.Transaction) (priorities.Priority, error) {
	if p, ok := c.byHash[tx.Hash()]; ok {
		return p, nil
	}
	return priorities.Priority{Level: big.NewInt(0), Weight: big.NewInt(0)}, nil
}

func tx(nonce uint64) *types.Transaction {
	return types.NewTransaction(nonce, common.Address{0x1}, big.NewInt(0), 21000, big.NewInt(1), nil)
}

func prioritized(id byte) priorities.Priority {
	return priorities.Priority{Level: big.NewInt(1), Weight: big.NewInt(1), Id: [32]byte{id}}
}

func TestPriorityHinter_Nil_IsNeverEligible(t *testing.T) {
	var h *priorityHinter
	ok, _ := h.eligible(true, tx(0))
	require.False(t, ok)
}

func TestPriorityHinter_RequiresNonEmptyEvent(t *testing.T) {
	a := tx(0)
	h := &priorityHinter{
		classifier: fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{a.Hash(): prioritized(1)}},
		config:     priorities.Config{MaxTxsPerEntityPerEvent: 5},
		counts:     map[[32]byte]uint64{},
	}
	// Empty event: not eligible (avoids priority-only events).
	ok, _ := h.eligible(false, a)
	require.False(t, ok)
	// Non-empty event: eligible.
	ok, _ = h.eligible(true, a)
	require.True(t, ok)
}

func TestPriorityHinter_NonPrioritized_IsNotEligible(t *testing.T) {
	a := tx(0) // not in classifier map -> level 0
	h := &priorityHinter{
		classifier: fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{}},
		config:     priorities.Config{MaxTxsPerEntityPerEvent: 5},
		counts:     map[[32]byte]uint64{},
	}
	ok, _ := h.eligible(true, a)
	require.False(t, ok)
}

func TestPriorityHinter_EnforcesPerEntityPerEventCap(t *testing.T) {
	a, b, c := tx(0), tx(1), tx(2)
	h := &priorityHinter{
		classifier: fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
			a.Hash(): prioritized(7),
			b.Hash(): prioritized(7), // same entity
			c.Hash(): prioritized(7),
		}},
		config: priorities.Config{MaxTxsPerEntityPerEvent: 2},
		counts: map[[32]byte]uint64{},
	}

	ok, id := h.eligible(true, a)
	require.True(t, ok)
	h.record(id)

	ok, id = h.eligible(true, b)
	require.True(t, ok)
	h.record(id)

	// Third transaction of the same entity exceeds the cap.
	ok, _ = h.eligible(true, c)
	require.False(t, ok)
}
