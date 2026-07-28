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
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakePriorityLookup resolves priorities from a fixed map, standing in for the
// priority cache. Unknown transactions are non-prioritized.
type fakePriorityLookup struct {
	byHash map[common.Hash]priorities.Priority
}

func (l fakePriorityLookup) Priority(hash common.Hash) priorities.Priority {
	return l.byHash[hash]
}

func prioritized(id byte) priorities.Priority {
	return priorityWith(id, 1, 1)
}

// priorityWith builds a prioritized Priority with an explicit level and weight
// so tests can assert (level, weight) ordering.
func priorityWith(id byte, level, weight uint64) priorities.Priority {
	return priorities.Priority{Level: level, Weight: weight, ID: [16]byte{id}}
}

func TestEmitter_NewPriorityContext_IsNilWithoutClassifiableHeadState(t *testing.T) {
	enabled := opera.Rules{Upgrades: opera.Upgrades{TransactionPriorities: true}}
	tests := map[string]func(*MockExternal){
		"priorities disabled": func(world *MockExternal) {
			world.EXPECT().GetRules().Return(opera.Rules{})
		},
		"no head block": func(world *MockExternal) {
			world.EXPECT().GetRules().Return(enabled)
			world.EXPECT().GetLatestBlock().Return(nil)
		},
		"no head header": func(world *MockExternal) {
			world.EXPECT().GetRules().Return(enabled)
			world.EXPECT().GetLatestBlock().Return(&inter.Block{})
			world.EXPECT().Header(gomock.Any(), gomock.Any()).Return(nil)
		},
		"no head state": func(world *MockExternal) {
			world.EXPECT().GetRules().Return(enabled)
			world.EXPECT().GetLatestBlock().Return(&inter.Block{})
			world.EXPECT().Header(gomock.Any(), gomock.Any()).Return(&evmcore.EvmHeader{})
			world.EXPECT().StateDB().Return(nil)
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			world := NewMockExternal(gomock.NewController(t))
			setup(world)
			em := &Emitter{world: World{External: world}}
			require.Nil(t, em.newPriorityContext())
		})
	}
}

func TestEmitter_RefreshPriorityContext_KeepsTheContextOfTheHeadBlock(t *testing.T) {
	world := NewMockExternal(gomock.NewController(t))
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(7))
	em := &Emitter{world: World{External: world}}

	context := &priorityContext{block: 7}
	require.Same(t, context, em.refreshPriorityContext(context))
}

func TestEmitter_RefreshPriorityContext_ReleasesTheContextOfAnOutdatedBlock(t *testing.T) {
	world := NewMockExternal(gomock.NewController(t))
	world.EXPECT().GetLatestBlockIndex().Return(idx.Block(8))
	world.EXPECT().GetRules().Return(opera.Rules{}) // priorities disabled: no new context
	em := &Emitter{world: World{External: world}}

	released := 0
	context := &priorityContext{block: 7, release: func() { released++ }}
	require.Nil(t, em.refreshPriorityContext(context))
	require.Equal(t, 1, released)
}

func TestPriorityHinter_Nil_IsNeverEligible(t *testing.T) {
	var h *priorityHinter
	ok, _ := h.eligible(prioritized(1))
	require.False(t, ok)
}

func TestPriorityHinter_NonPrioritized_IsNotEligible(t *testing.T) {
	h := &priorityHinter{
		config: priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 5},
		counts: map[[16]byte]uint64{},
	}
	ok, _ := h.eligible(priorities.Priority{})
	require.False(t, ok)
}

func TestPriorityHinter_EnforcesPerEntityPerEventCap(t *testing.T) {
	h := &priorityHinter{
		config: priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 2},
		counts: map[[16]byte]uint64{},
	}

	ok, id := h.eligible(prioritized(7))
	require.True(t, ok)
	h.record(id)

	ok, id = h.eligible(prioritized(7)) // same entity
	require.True(t, ok)
	h.record(id)

	// Third transaction of the same entity exceeds the cap.
	ok, _ = h.eligible(prioritized(7))
	require.False(t, ok)
}
