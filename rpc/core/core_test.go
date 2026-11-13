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

package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The interface grows only when a method cannot be served from what is
// there, and never by a call whose answer depends on "now" beyond the pool,
// liveness and the live consensus calls. A change here is that decision.
func TestCore_MethodSetIsFrozen(t *testing.T) {
	want := []string{
		"Block(uint64) (*evmcore.EvmBlock, error)",
		"BlockNumberByHash(common.Hash) (uint64, error)",
		"BundleInfo(common.Hash) (core.BundleInfo, error)",
		"ChainInfo() core.ChainInfo",
		"EmitTransactions(types.Transactions) error",
		"EpochOf(uint64) (uint64, error)",
		"EpochState(uint64) (core.EpochSnapshot, error)",
		"Event(common.Hash) (*inter.Event, error)",
		"EventPayload(common.Hash) (*inter.EventPayload, error)",
		"FindEvents(uint64, uint32, []uint8) []common.Hash",
		"Head() (*evmcore.EvmBlock, error)",
		"Header(uint64) (*evmcore.EvmHeader, error)",
		"LiveEpochHeads(uint64) ([]common.Hash, error)",
		"LiveEpochState() core.EpochSnapshot",
		"Logs(context.Context, core.LogQuery) ([]*types.Log, error)",
		"OracleInputs() core.OracleInputs",
		"PeerCount() int",
		"PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)",
		"PoolContentFrom(common.Address) (types.Transactions, types.Transactions)",
		"PoolNextTransactions() types.Transactions",
		"PoolNonce(common.Address) uint64",
		"PoolStats() (int, int)",
		"PoolTransaction(common.Hash) (*types.Transaction, error)",
		"Progress() core.Progress",
		"Receipts(uint64) (types.Receipts, error)",
		"Rules(uint64) (*opera.Rules, error)",
		"SendTransaction(context.Context, *types.Transaction) error",
		"SendTransactions(types.Transactions) error",
		"State(uint64) (state.StateDB, error)",
		"SubscribeHead(chan<- core.HeadNotify) core.Subscription",
		"SubscribeTransactions(chan<- evmcore.NewTxsNotify) core.Subscription",
		"TransactionByHash(common.Hash) (*types.Transaction, uint64, uint64, error)",
	}
	var got []string
	for method := range reflect.TypeFor[Core]().Methods() {
		got = append(got, method.Name+strings.TrimPrefix(method.Type.String(), "func"))
	}
	require.Equal(t, want, got)
}
