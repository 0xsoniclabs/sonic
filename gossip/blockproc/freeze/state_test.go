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

package freeze

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestBalanceChanges_CountTransfersAndSelfdestructsOfFrozenAccounts(t *testing.T) {
	fake := newFakeState(frozenAccount)
	db := Wrap(fake)
	one := uint256.NewInt(1)
	call := db.Snapshot()

	db.AddBalance(frozenAccount, one, tracing.BalanceIncreaseGasReturn)
	db.SubBalance(frozenAccount, one, tracing.BalanceDecreaseGasBuy)
	db.AddBalance(otherAccount, one, tracing.BalanceChangeTransfer)
	db.AddBalance(otherAccount, one, tracing.BalanceIncreaseSelfdestruct)
	db.SubBalance(frozenAccount, new(uint256.Int), tracing.BalanceChangeTransfer)
	require.False(t, db.movedSince(call), "gas, unfrozen or zero value")
	require.Equal(t, 2, fake.moved, "balance changes reach the state")

	for _, reason := range []tracing.BalanceChangeReason{tracing.BalanceChangeTransfer, tracing.BalanceIncreaseSelfdestruct, tracing.BalanceDecreaseSelfdestruct, tracing.BalanceDecreaseSelfdestructBurn} {
		inner := db.Snapshot()
		db.AddBalance(frozenAccount, one, reason)
		require.True(t, db.movedSince(inner), "reason %v", reason)
	}
}
