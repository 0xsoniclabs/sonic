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

package subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// The numbers one sponsored transaction of the fixtures below is charged.
const (
	sponsoredGasUsed = 40_000
	testBaseFee      = 1_000
)

// observationOf builds what an iteration would have observed for a block holding the given
// transactions, with the sponsored one at index 0.
func observationOf(t *testing.T, sponsored core.TxSpec, txs ...*types.Transaction) core.Observation {
	t.Helper()

	block := types.NewBlock(
		&types.Header{Number: big.NewInt(7), BaseFee: big.NewInt(testBaseFee)},
		&types.Body{Transactions: txs}, nil, trie.NewStackTrie(nil),
	)

	key, err := core.DeriveKey(0)
	require.NoError(t, err)

	return core.Observation{
		Blocks: []*types.Block{block},
		Txs: []core.TxObservation{{
			Spec:    sponsored,
			Hash:    txs[0].Hash(),
			Outcome: core.OutcomeExecuted,
			Receipt: &types.Receipt{
				BlockNumber:      big.NewInt(7),
				TransactionIndex: 0,
				GasUsed:          sponsoredGasUsed,
			},
			Sender: core.PooledAccount{Account: &tests.Account{PrivateKey: key}},
		}},
	}
}

// chargeFor is what the follow-up of a mode must report for the fixtures above.
func chargeFor(mode Mode) *big.Int {
	return domainWith(mode, new(big.Int), 0).Charge(sponsoredGasUsed, big.NewInt(testBaseFee))
}

func TestCheckPostTransactions_AcceptsTheFollowUpEachModeCallsFor(t *testing.T) {
	sponsored := request(0, 100_000)

	tests := map[Mode][]*types.Transaction{
		ModeFundBacked:     {plainTx(0), feeCharge(chargeFor(ModeFundBacked))},
		ModeNetwork:        {plainTx(0)},
		ModeNetworkTracked: {plainTx(0), track(chargeFor(ModeNetworkTracked))},
	}

	for mode, txs := range tests {
		t.Run(mode.String(), func(t *testing.T) {
			domain := domainWith(mode, new(big.Int), 0)
			charged, err := domain.CheckPostTransactions(observationOf(t, sponsored, txs...))
			require.NoError(t, err)

			if mode == ModeNetwork {
				require.Empty(t, charged, "nothing is charged where the network pays")
				return
			}
			require.Len(t, charged, 1)
			for _, amount := range charged {
				require.Equal(t, chargeFor(mode), amount)
			}
		})
	}
}

func TestCheckPostTransactions_RejectsASpoiledFollowUp(t *testing.T) {
	sponsored := request(0, 100_000)
	charge := chargeFor(ModeFundBacked)

	tests := map[string]struct {
		mode  Mode
		txs   []*types.Transaction
		issue string
	}{
		"a missing follow-up": {
			mode:  ModeFundBacked,
			txs:   []*types.Transaction{plainTx(0)},
			issue: "follow-up is missing",
		},
		"an ordinary transaction where the follow-up belongs": {
			mode:  ModeFundBacked,
			txs:   []*types.Transaction{plainTx(0), plainTx(1)},
			issue: "not a deductFees transaction",
		},
		"a charge that is one wei off": {
			mode:  ModeFundBacked,
			txs:   []*types.Transaction{plainTx(0), feeCharge(new(big.Int).Add(charge, big.NewInt(1)))},
			issue: "charges",
		},
		"a track transaction charging a fund-backed sponsorship": {
			mode:  ModeFundBacked,
			txs:   []*types.Transaction{plainTx(0), track(charge)},
			issue: "not a deductFees transaction",
		},
		"a fee charged where the network was supposed to pay": {
			mode:  ModeNetwork,
			txs:   []*types.Transaction{plainTx(0), feeCharge(charge)},
			issue: "append none",
		},
		"a tracked sponsorship charging a fund instead": {
			mode:  ModeNetworkTracked,
			txs:   []*types.Transaction{plainTx(0), feeCharge(charge)},
			issue: "not a track transaction",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			domain := domainWith(test.mode, new(big.Int), 0)
			_, err := domain.CheckPostTransactions(observationOf(t, sponsored, test.txs...))
			require.ErrorContains(t, err, test.issue)
		})
	}
}

func TestCheckPostTransactions_RejectsAFundChargedForATransactionPayingItsOwnWay(t *testing.T) {
	domain := domainWith(ModeFundBacked, new(big.Int), 0)

	obs := observationOf(t, paying(0, 100_000),
		plainTx(0), feeCharge(chargeFor(ModeFundBacked)))

	_, err := domain.CheckPostTransactions(obs)
	require.ErrorContains(t, err, "pays its own way but is followed by the registry call")
}

func TestCheckPostTransactions_RejectsAReceiptPointingAtTheWrongTransaction(t *testing.T) {
	domain := domainWith(ModeFundBacked, new(big.Int), 0)

	obs := observationOf(t, request(0, 100_000), plainTx(0), feeCharge(chargeFor(ModeFundBacked)))
	obs.Txs[0].Receipt.TransactionIndex = 1

	_, err := domain.CheckPostTransactions(obs)
	require.ErrorContains(t, err, "is not at index 1 of block 7")
}

func TestCheckPostTransactions_RejectsABlockItNeverSaw(t *testing.T) {
	domain := domainWith(ModeFundBacked, new(big.Int), 0)

	obs := observationOf(t, request(0, 100_000), plainTx(0), feeCharge(chargeFor(ModeFundBacked)))
	obs.Txs[0].Receipt.BlockNumber = big.NewInt(99)

	_, err := domain.CheckPostTransactions(obs)
	require.ErrorContains(t, err, "not among the blocks of this iteration")
}
