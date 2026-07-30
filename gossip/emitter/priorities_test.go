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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// PrioEnabledRules are rules with the transaction priorities feature turned on.
var PrioEnabledRules = opera.Rules{Upgrades: opera.Upgrades{TransactionPriorities: true}}

func TestEmitter_NewPriorityContext_IsNilExactlyWithoutClassifiableHeadState(t *testing.T) {
	tests := map[string]struct {
		setup     func(*gomock.Controller, *MockExternal)
		expectNil bool
	}{
		"priorities disabled return nil": {
			setup: func(_ *gomock.Controller, world *MockExternal) {
				world.EXPECT().GetRules().Return(opera.Rules{})
			},
			expectNil: true,
		},
		"no head block returns nil": {
			setup: func(_ *gomock.Controller, world *MockExternal) {
				world.EXPECT().GetRules().Return(PrioEnabledRules)
				world.EXPECT().GetLatestBlock().Return(nil)
			},
			expectNil: true,
		},
		"no header returns nil": {
			setup: func(_ *gomock.Controller, world *MockExternal) {
				world.EXPECT().GetRules().Return(PrioEnabledRules)
				world.EXPECT().GetLatestBlock().Return(&inter.Block{})
				world.EXPECT().Header(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectNil: true,
		},
		"no head state returns nil": {
			setup: func(_ *gomock.Controller, world *MockExternal) {
				world.EXPECT().GetRules().Return(PrioEnabledRules)
				world.EXPECT().GetLatestBlock().Return(&inter.Block{})
				world.EXPECT().Header(gomock.Any(), gomock.Any()).Return(&evmcore.EvmHeader{})
				world.EXPECT().StateDB().Return(nil)
			},
			expectNil: true,
		},
		"classifiable head state returns context": {
			setup: func(ctrl *gomock.Controller, world *MockExternal) {
				world.EXPECT().GetRules().Return(PrioEnabledRules)
				world.EXPECT().GetLatestBlock().Return(&inter.Block{})
				world.EXPECT().Header(gomock.Any(), gomock.Any()).Return(&evmcore.EvmHeader{Number: big.NewInt(0)})
				// An empty registry account is enough to build the context; the
				// config read fails and falls back.
				db := state.NewMockStateDB(ctrl)
				// config query uses inter tx snapshots
				gomock.InOrder(
					db.EXPECT().InterTxSnapshot().Return(42),
					db.EXPECT().Snapshot(),
					db.EXPECT().Exist(gomock.Any()).Return(false),
					db.EXPECT().RevertToInterTxSnapshot(42),
				)
				world.EXPECT().StateDB().Return(db)
				world.EXPECT().GetUpgradeHeights().Return(nil)
				world.EXPECT().GetLatestBlockIndex().Return(idx.Block(0))
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			world := NewMockExternal(ctrl)
			test.setup(ctrl, world)
			em := &Emitter{world: World{External: world}}
			context := em.newPriorityContext()
			require.Equal(t, test.expectNil, context == nil)
		})
	}
}

func TestPriorityContext_Release_CallsReleaseDBFunctionIfContextNotNil(t *testing.T) {
	var context *priorityContext
	context.release() // no-op (does not panic)

	called := false
	context = &priorityContext{releaseDB: func() { called = true }}
	context.release()
	require.True(t, called)
}

func TestPriorityContext_PriorityOf_TakesTheClassificationUnlessItFails(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})

	tests := map[string]struct {
		priority priorities.Priority
		err      error
		expected priorities.Priority
	}{
		"prioritized": {
			priority: prio(1, 1, 1),
			err:      nil,
			expected: prio(1, 1, 1),
		},
		"ordinary": {
			priority: priorities.Priority{},
			err:      nil,
			expected: priorities.Priority{},
		},
		"failed classification": {
			priority: prio(1, 1, 2),
			err:      fmt.Errorf("injected error"),
			expected: priorities.Priority{},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			classifier := priorities.NewMockClassifier(gomock.NewController(t))
			classifier.EXPECT().Priority(tx).Return(test.priority, test.err)
			context := &priorityContext{
				cache:      evmcore.NewPriorityCache(evmcore.DefaultTxPoolConfig),
				classifier: classifier,
			}
			require.Equal(t, test.expected, context.priorityOf(tx))
		})
	}
}

func TestPriorityContext_PriorityOf_ReturnsNotPrioritizedWhenContextIsNil(t *testing.T) {
	var context *priorityContext
	require.Equal(t, priorities.Priority{}, context.priorityOf(types.NewTx(&types.LegacyTx{Nonce: 1})))
}

func TestPriorityContext_GetConfig_ReturnsConfigIfContextIsNotNil(t *testing.T) {
	var context *priorityContext
	require.Equal(t, priorities.Config{}, context.getConfig())

	config := priorities.Config{MaxPiggybackTxsPerEntityPerEvent: 3}
	require.Equal(t, config, (&priorityContext{config: config}).getConfig())
}

func prio(level uint64, weight uint64, id byte) priorities.Priority {
	return priorities.Priority{Level: level, Weight: weight, ID: priorities.PriorityID{id}}
}
