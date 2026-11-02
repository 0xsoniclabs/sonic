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

package evmstore

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestLoggingStateDB_SelfDestruct_ReportsTheBurnOnlyBeforeAmsterdam(t *testing.T) {
	type balanceChange struct {
		prev, new uint64
		reason    tracing.BalanceChangeReason
	}
	contract := common.Address{1}
	tests := map[string]struct {
		rules params.Rules
		want  []balanceChange
	}{
		"before Amsterdam": {
			rules: params.Rules{},
			want: []balanceChange{
				{0, 12, tracing.BalanceChangeTransfer},
				{12, 17, tracing.BalanceChangeTransfer},
				{17, 0, tracing.BalanceDecreaseSelfdestructBurn},
			},
		},
		"from Amsterdam on": {
			rules: params.Rules{IsAmsterdam: true},
			want: []balanceChange{
				{0, 12, tracing.BalanceChangeTransfer},
				{12, 17, tracing.BalanceChangeTransfer},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var got []balanceChange
			logger := &tracing.Hooks{
				OnBalanceChange: func(_ common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
					got = append(got, balanceChange{prev.Uint64(), new.Uint64(), reason})
				},
			}
			state := WrapStateDbWithLogger(newTestCarmenStateDb(t), logger)

			state.BeginBlock(1)
			state.CreateAccount(contract)
			state.CreateContract(contract)
			state.AddBalance(contract, uint256.NewInt(12), tracing.BalanceChangeTransfer)
			state.SelfDestruct(contract)
			state.AddBalance(contract, uint256.NewInt(5), tracing.BalanceChangeTransfer)
			state.EndTransaction(test.rules)

			require.Equal(t, test.want, got)
		})
	}
}
