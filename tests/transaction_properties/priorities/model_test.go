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

// The two senders every case draws from, and a shorthand for one transaction of one of them. The
// hashes are given explicitly, because a hash is what breaks a tie the level and the weight leave.
var (
	alice = common.Address{0xa1}
	bob   = common.Address{0xb0}
)

func tx(hash byte, sender common.Address, nonce uint64, gas uint64, p Priority) BlockTx {
	return BlockTx{
		Hash:     common.Hash{hash},
		Sender:   sender,
		Nonce:    nonce,
		Gas:      gas,
		Priority: p,
	}
}

func prio(level, weight, entity uint64) Priority {
	return Priority{Level: level, Weight: weight, Entity: entity}
}

var none = Priority{}

func TestPrioritizedPrefix_OrdersAndDemotesAsTheRulesSay(t *testing.T) {
	tests := map[string]struct {
		txs    []BlockTx
		starts map[common.Address]uint64
		budget uint64
		want   []byte // the hashes of the expected prefix, in order
	}{
		"nothing prioritized is hoisted": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, none),
				tx(2, bob, 0, 21_000, none),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: perEntityBudget,
			want:   nil,
		},
		"a prioritized transaction goes ahead of an ordinary one": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, none),
				tx(2, bob, 0, 21_000, prio(1, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: perEntityBudget,
			want:   []byte{2},
		},
		"the higher level goes first": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, prio(1, 9, 0)),
				tx(2, bob, 0, 21_000, prio(2, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: perEntityBudget,
			want:   []byte{2, 1},
		},
		"the higher weight breaks a level": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, prio(1, 1, 0)),
				tx(2, bob, 0, 21_000, prio(1, 2, 0)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: perEntityBudget,
			want:   []byte{2, 1},
		},
		"the smaller hash breaks a weight": {
			txs: []BlockTx{
				tx(2, alice, 0, 21_000, prio(1, 1, 0)),
				tx(1, bob, 0, 21_000, prio(1, 1, 0)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: perEntityBudget,
			want:   []byte{1, 2},
		},
		"the maximum level and weight are ordinary values": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, prio(math.MaxUint64, 0, 0)),
				tx(2, bob, 0, 21_000, prio(math.MaxUint64, math.MaxUint64, 0)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: perEntityBudget,
			want:   []byte{2, 1},
		},
		"one sender's run keeps its nonce order whatever its priorities say": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, prio(1, 0, 0)),
				tx(2, alice, 1, 21_000, prio(2, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 0},
			budget: perEntityBudget,
			want:   []byte{1, 2},
		},
		"a prioritized nonce behind an ordinary one is not hoisted": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, none),
				tx(2, alice, 1, 21_000, prio(2, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 0},
			budget: perEntityBudget,
			want:   nil,
		},
		"a hole in the run ends it": {
			txs: []BlockTx{
				tx(1, alice, 0, 21_000, prio(1, 0, 0)),
				tx(2, alice, 1, 21_000, none),
				tx(3, alice, 2, 21_000, prio(2, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 0},
			budget: perEntityBudget,
			want:   []byte{1},
		},
		"a nonce the sender has spent is passed over": {
			txs: []BlockTx{
				tx(1, alice, 4, 21_000, prio(1, 0, 0)),
				tx(2, alice, 5, 21_000, prio(1, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 5},
			budget: perEntityBudget,
			want:   []byte{2},
		},
		"the run starts at the nonce the block started with": {
			txs: []BlockTx{
				tx(1, alice, 6, 21_000, prio(1, 0, 0)),
			},
			starts: map[common.Address]uint64{alice: 5},
			budget: perEntityBudget,
			want:   nil, // nonce 5 is missing, so nothing reaches this one
		},
		"an entity spends its budget and no more": {
			txs: []BlockTx{
				tx(1, alice, 0, 40_000, prio(1, 0, 7)),
				tx(2, alice, 1, 40_000, prio(1, 0, 7)),
				tx(3, alice, 2, 40_000, prio(1, 0, 7)),
			},
			starts: map[common.Address]uint64{alice: 0},
			budget: 100_000,
			want:   []byte{1, 2},
		},
		"the later nonces of a sender over budget go with it": {
			txs: []BlockTx{
				tx(1, alice, 0, 90_000, prio(2, 0, 7)),
				tx(2, alice, 1, 10_000, prio(2, 0, 7)),
				tx(3, alice, 2, 10_000, prio(2, 0, 7)),
			},
			starts: map[common.Address]uint64{alice: 0},
			budget: 50_000,
			want:   nil,
		},
		"another sender of the same entity keeps packing": {
			txs: []BlockTx{
				tx(1, alice, 0, 90_000, prio(2, 0, 7)),
				tx(2, bob, 0, 10_000, prio(1, 0, 7)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: 50_000,
			want:   []byte{2},
		},
		"a budget is spent per entity": {
			txs: []BlockTx{
				tx(1, alice, 0, 40_000, prio(1, 0, 7)),
				tx(2, bob, 0, 40_000, prio(1, 0, 8)),
			},
			starts: map[common.Address]uint64{alice: 0, bob: 0},
			budget: 40_000,
			want:   []byte{1, 2},
		},
		"a transaction of exactly the budget still fits": {
			txs: []BlockTx{
				tx(1, alice, 0, 50_000, prio(1, 0, 7)),
			},
			starts: map[common.Address]uint64{alice: 0},
			budget: 50_000,
			want:   []byte{1},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			prefix := PrioritizedPrefix(test.txs, test.starts, test.budget)

			hashes := make([]byte, 0, len(prefix))
			for _, tx := range prefix {
				hashes = append(hashes, tx.Hash[0])
			}
			require.Equal(t, test.want, nonEmpty(hashes))
		})
	}
}

// nonEmpty renders an empty prefix as nil, so a case expecting nothing can say so.
func nonEmpty(hashes []byte) []byte {
	if len(hashes) == 0 {
		return nil
	}
	return hashes
}
