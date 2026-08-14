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
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// fakeChain serves receipts and transactions from memory, standing in for the client.
type fakeChain struct {
	Receipts map[common.Hash]*types.Receipt
	present  map[common.Hash]bool
}

func (c fakeChain) TransactionReceipt(_ context.Context, hash common.Hash) (*types.Receipt, error) {
	if receipt, ok := c.Receipts[hash]; ok {
		return receipt, nil
	}
	return nil, ethereum.NotFound
}

func (c fakeChain) TransactionByHash(_ context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	if c.present[hash] {
		return types.NewTx(&types.LegacyTx{}), false, nil
	}
	return nil, false, ethereum.NotFound
}

// blockOf builds a block carrying the given transactions and gas, with receipts to match unless a test
// spoils one.
func blockOf(number uint64, gasUsed uint64, txs []*types.Transaction) *types.Block {
	return types.NewBlock(
		&types.Header{
			Number:   new(big.Int).SetUint64(number),
			GasLimit: 10_000_000,
			GasUsed:  gasUsed,
			Time:     number * 1000,
		},
		&types.Body{Transactions: txs},
		nil,
		trie.NewStackTrie(nil),
	)
}

func transactionsOf(count int) []*types.Transaction {
	txs := make([]*types.Transaction, count)
	for i := range txs {
		txs[i] = types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 100_000})
	}
	return txs
}

// receiptsFor builds the receipts a block's transactions should have: index in order, cumulative gas
// as the running sum.
func receiptsFor(block *types.Block, gasUsed uint64) map[common.Hash]*types.Receipt {
	receipts := map[common.Hash]*types.Receipt{}
	cumulative := uint64(0)
	for i, tx := range block.Transactions() {
		cumulative += gasUsed
		receipts[tx.Hash()] = &types.Receipt{
			TransactionIndex:  uint(i),
			BlockHash:         block.Hash(),
			BlockNumber:       block.Number(),
			GasUsed:           gasUsed,
			CumulativeGasUsed: cumulative,
			EffectiveGasPrice: big.NewInt(1),
		}
	}
	return receipts
}

func TestCheckOneBlock_AcceptsABlockWhoseReceiptsAgreeWithIt(t *testing.T) {
	block := blockOf(1, 3*21_000, transactionsOf(3))
	chain := fakeChain{Receipts: receiptsFor(block, 21_000)}

	require.NoError(t, CheckOneBlock(t.Context(), chain, block))
}

func TestCheckOneBlock_RejectsBrokenBookkeeping(t *testing.T) {
	tests := map[string]struct {
		spoil func(receipts map[common.Hash]*types.Receipt, block *types.Block)
		want  string
	}{
		"a transaction without a receipt": {
			spoil: func(receipts map[common.Hash]*types.Receipt, block *types.Block) {
				delete(receipts, block.Transactions()[1].Hash())
			},
			want: "has no receipt",
		},
		"a receipt index out of step, as a dropped transaction would leave": {
			spoil: func(receipts map[common.Hash]*types.Receipt, block *types.Block) {
				receipts[block.Transactions()[1].Hash()].TransactionIndex = 2
			},
			want: "has receipt index 2",
		},
		"a receipt pointing at another block": {
			spoil: func(receipts map[common.Hash]*types.Receipt, block *types.Block) {
				receipts[block.Transactions()[0].Hash()].BlockHash = common.Hash{0xff}
			},
			want: "receipt block hash",
		},
		"a receipt pointing at another block number": {
			spoil: func(receipts map[common.Hash]*types.Receipt, block *types.Block) {
				receipts[block.Transactions()[0].Hash()].BlockNumber = big.NewInt(99)
			},
			want: "receipt block number",
		},
		"gas used above the transaction's own limit": {
			spoil: func(receipts map[common.Hash]*types.Receipt, block *types.Block) {
				receipts[block.Transactions()[0].Hash()].GasUsed = 200_000
			},
			want: "more than its limit",
		},
		"cumulative gas that is not the running sum": {
			spoil: func(receipts map[common.Hash]*types.Receipt, block *types.Block) {
				receipts[block.Transactions()[1].Hash()].CumulativeGasUsed = 1
			},
			want: "reports cumulative gas 1",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			block := blockOf(1, 3*21_000, transactionsOf(3))
			receipts := receiptsFor(block, 21_000)
			test.spoil(receipts, block)

			err := CheckOneBlock(t.Context(), fakeChain{Receipts: receipts}, block)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestCheckOneBlock_RejectsABlockDisagreeingWithItsOwnReceipts(t *testing.T) {
	// The block claims more gas than its receipts account for, which is what keeping a skipped
	// transaction in a block would produce.
	block := blockOf(1, 99_999, transactionsOf(2))
	err := CheckOneBlock(t.Context(), fakeChain{Receipts: receiptsFor(block, 21_000)}, block)
	require.ErrorContains(t, err, "block reports 99999 gas used, but its receipts sum to 42000")
}

func TestCheckBlockInvariants_RequiresAContiguousChain(t *testing.T) {
	first := blockOf(1, 0, nil)

	tests := map[string]struct {
		second *types.Block
		want   string
	}{
		"a gap in the numbering": {
			second: blockOf(3, 0, nil),
			want:   "not contiguous",
		},
		"a parent that is not the previous block": {
			second: types.NewBlock(
				&types.Header{Number: big.NewInt(2), ParentHash: common.Hash{0xaa}, GasLimit: 10_000_000, Time: 2000},
				&types.Body{}, nil, trie.NewStackTrie(nil),
			),
			want: "does not chain to block 1",
		},
		"a timestamp going backwards": {
			second: types.NewBlock(
				&types.Header{Number: big.NewInt(2), ParentHash: first.Hash(), GasLimit: 10_000_000, Time: 0},
				&types.Body{}, nil, trie.NewStackTrie(nil),
			),
			want: "before block 1's",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := CheckBlockInvariants(t.Context(), fakeChain{}, []*types.Block{first, test.second})
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestCheckBlockInvariants_AcceptsAWellFormedChain(t *testing.T) {
	first := blockOf(1, 0, nil)
	second := types.NewBlock(
		&types.Header{
			Number: big.NewInt(2), ParentHash: first.Hash(), GasLimit: 10_000_000, Time: 2000,
		},
		&types.Body{}, nil, trie.NewStackTrie(nil),
	)

	require.NoError(t, CheckBlockInvariants(t.Context(), fakeChain{}, []*types.Block{first, second}))
}

func TestCheckAbsent_OnlyAcceptsATransactionTheChainDoesNotHave(t *testing.T) {
	hash := common.Hash{0x01}

	require.NoError(t, CheckAbsent(t.Context(), fakeChain{}, hash))
	require.ErrorContains(t,
		CheckAbsent(t.Context(), fakeChain{present: map[common.Hash]bool{hash: true}}, hash),
		"produced no receipt but is still retrievable",
	)
}

// executedFor is one executed transaction costing the given fee, with no value transfer.
func executedFor(gasUsed uint64, price int64) ExecutedTx {
	return ExecutedTx{
		Receipt:        &types.Receipt{GasUsed: gasUsed, EffectiveGasPrice: big.NewInt(price)},
		Value:          big.NewInt(0),
		TransfersValue: false,
		BlobFee:        new(big.Int),
	}
}

func TestCheckAccounting_RequiresExactMovementWhenNoDefectApplies(t *testing.T) {
	balance := big.NewInt(1_000_000)
	fee := int64(21_000 * 2)

	observed, err := CheckAccounting(common.Address{0x01}, StateChange{
		nonceBefore:   4,
		nonceAfter:    5,
		balanceBefore: balance,
		balanceAfter:  new(big.Int).Sub(balance, big.NewInt(fee)),
	}, AccountExpectation{executed: []ExecutedTx{executedFor(21_000, 2)}})

	require.NoError(t, err)
	require.Zero(t, observed.unreceiptedCharge.Sign(), "nothing may be charged without a receipt")
}

func TestCheckAccounting_CountsValueOnlyWhenItLeavesTheAccount(t *testing.T) {
	balance := big.NewInt(1_000_000)
	transferred := executedFor(21_000, 1)
	transferred.Value = big.NewInt(500)
	transferred.TransfersValue = true

	_, err := CheckAccounting(common.Address{0x01}, StateChange{
		nonceBefore:   0,
		nonceAfter:    1,
		balanceBefore: balance,
		balanceAfter:  new(big.Int).Sub(balance, big.NewInt(21_000+500)),
	}, AccountExpectation{executed: []ExecutedTx{transferred}})

	require.NoError(t, err)
}

func TestCheckAccounting_RejectsWeiAppearingOrVanishing(t *testing.T) {
	balance := big.NewInt(1_000_000)
	expectation := AccountExpectation{executed: []ExecutedTx{executedFor(21_000, 1)}}

	tests := map[string]*big.Int{
		"more wei than the fees account for": new(big.Int).Sub(balance, big.NewInt(21_001)),
		"wei appearing from nothing":         new(big.Int).Add(balance, big.NewInt(1)),
		"no charge at all":                   balance,
	}

	for name, after := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := CheckAccounting(common.Address{0x01}, StateChange{
				nonceBefore: 0, nonceAfter: 1, balanceBefore: balance, balanceAfter: after,
			}, expectation)
			require.ErrorContains(t, err, "so it should hold")
		})
	}
}

func TestCheckAccounting_RejectsANonceThatDidNotFollowItsTransactions(t *testing.T) {
	balance := big.NewInt(1_000_000)

	_, err := CheckAccounting(common.Address{0x01}, StateChange{
		nonceBefore:   0,
		nonceAfter:    0, // one transaction executed, so it should be 1
		balanceBefore: balance,
		balanceAfter:  new(big.Int).Sub(balance, big.NewInt(21_000)),
	}, AccountExpectation{executed: []ExecutedTx{executedFor(21_000, 1)}})

	require.ErrorContains(t, err, "its nonce should be 1, but it is 0")
}

func TestCheckAccounting_ToleratesAnUnreceiptedChargeWithinItsAllowance(t *testing.T) {
	balance := big.NewInt(1_000_000)

	// Known defect 2: pre-Allegro, a transaction that reached no block still paid for its gas.
	observed, err := CheckAccounting(common.Address{0x01}, StateChange{
		nonceBefore:   0,
		nonceAfter:    0,
		balanceBefore: balance,
		balanceAfter:  new(big.Int).Sub(balance, big.NewInt(900)),
	}, AccountExpectation{maxUnreceiptedCharge: big.NewInt(1_000)})

	require.NoError(t, err)
	require.Equal(t, big.NewInt(900), observed.unreceiptedCharge)

	// Beyond the allowance it is a failure again.
	_, err = CheckAccounting(common.Address{0x01}, StateChange{
		nonceBefore: 0, nonceAfter: 0, balanceBefore: balance,
		balanceAfter: new(big.Int).Sub(balance, big.NewInt(1_001)),
	}, AccountExpectation{maxUnreceiptedCharge: big.NewInt(1_000)})
	require.ErrorContains(t, err, "should hold between")
}

func TestOutcomeOf_IsDecidedByTheReceipt(t *testing.T) {
	executed := common.Hash{0x01}
	result := Outcomes{Receipts: map[common.Hash]*types.Receipt{executed: {}}}

	require.Equal(t, OutcomeExecuted, result.OutcomeOf(executed))
	require.Equal(t, OutcomeDropped, result.OutcomeOf(common.Hash{0x02}))
}
