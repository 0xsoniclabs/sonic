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

package gossip

import (
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestApplyTransactionPriorities_FeatureDisabled_IsNoOp verifies that, with the
// feature off, the transaction list is returned unchanged and no EVM/state is
// touched (nil arguments would panic if they were used).
func TestApplyTransactionPriorities_FeatureDisabled_IsNoOp(t *testing.T) {
	txs := types.Transactions{
		types.NewTransaction(0, common.Address{0x1}, big.NewInt(0), 21000, big.NewInt(1), nil),
		types.NewTransaction(1, common.Address{0x2}, big.NewInt(0), 21000, big.NewInt(1), nil),
	}
	rules := opera.FakeNetRules(opera.GetBrioUpgrades()) // TransactionPriorities == false

	got := applyTransactionPriorities(
		txs, rules,
		nil, // chainCfg
		nil, // statedb
		nil, // reader
		nil, // signer
		0, 0, common.Hash{}, nil,
	)
	require.Equal(t, txs, got)
}

func TestApplyTransactionPriorities_EmptyInput_IsNoOp(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true
	rules := opera.FakeNetRules(upgrades)

	got := applyTransactionPriorities(
		types.Transactions{}, rules,
		nil, nil, nil, nil, 0, 0, common.Hash{}, nil,
	)
	require.Empty(t, got)
}

func TestPriorityQueryHeader_UsesConsensusInputs(t *testing.T) {
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	randao := common.Hash{0xab}
	parent := &evmcore.EvmHeader{
		Hash:     common.Hash{0xae},
		BaseFee:  big.NewInt(1_000),
		Duration: time.Second,
		GasUsed:  10_000,
	}

	header := priorityQueryHeader(rules, 42, inter.Timestamp(1234), randao, parent)

	require.Equal(t, uint64(42), header.Number.Uint64())
	require.Equal(t, inter.Timestamp(1234), header.Time)
	require.Equal(t, randao, header.PrevRandao)
	require.Equal(t, parent.Hash, header.ParentHash)
	require.Equal(t, rules.Blocks.MaxBlockGas, header.GasLimit)
	require.Equal(t, evmcore.GetCoinbase(), header.Coinbase)
	require.NotNil(t, header.BaseFee)
}
