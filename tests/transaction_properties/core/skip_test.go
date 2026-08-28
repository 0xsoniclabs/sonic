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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// spendingSpec is the least a transaction has to be for dropOffending to walk it: a sender, a value
// and a price. Only what the balance arithmetic reads is filled in.
type spendingSpec struct {
	Envelope
	Payload
	OptionalRecipient
	WideValue
	SinglePrice
}

func (spendingSpec) TxType() uint8 { return types.LegacyTxType }

func (spendingSpec) TxData(uint64, BuildContext) (types.TxData, error) {
	return nil, nil
}

func spending(sender int, value *big.Int) spendingSpec {
	return spendingSpec{
		Envelope:    Envelope{SenderIdx: sender, GasLimit: 21_000},
		WideValue:   WideValue{Value: value},
		SinglePrice: SinglePrice{GasPrice: big.NewInt(0)},
	}
}

// balanceReportingDomain answers every Skip with what the sender was said to hold, so a test can read
// back the balance each transaction was judged against.
type balanceReportingDomain struct {
	seen []*big.Int
}

func (d *balanceReportingDomain) Skip(_ TxSpec, sender SenderState, _ *big.Int) string {
	d.seen = append(d.seen, new(big.Int).Set(sender.Balance))
	return ""
}

func (*balanceReportingDomain) Batch() *rapid.Generator[[]TxSpec] { return nil }
func (*balanceReportingDomain) Pricing() PricingRules             { return nil }
func (*balanceReportingDomain) Notes() []string                   { return nil }

func (*balanceReportingDomain) Prepare(*rapid.T, context.Context, []PooledAccount, []TxSpec) error {
	return nil
}

func (*balanceReportingDomain) Extra(*rapid.T, context.Context) ([]ExtraExecuted, error) {
	return nil, nil
}

func (*balanceReportingDomain) Check(*rapid.T, context.Context, Observation) error { return nil }

// TestDropOffending_JudgesEachTransactionAgainstWhatIsLeftOfTheBalance is the property the running
// balance exists for: a policy asked about the third transaction of a sender must be told what the
// first two leave, not what the sender opened the batch with. Judging every transaction against the
// opening balance is how a creation that does provoke defect 1 was injected anyway.
func TestDropOffending_JudgesEachTransactionAgainstWhatIsLeftOfTheBalance(t *testing.T) {
	domain := &balanceReportingDomain{}
	runner := &Runner{Domain: domain, skipped: map[string]int{}}

	// One sender, spending a quarter of its balance twice. Gas is free at a zero price, so what each
	// transaction takes is exactly its value.
	quarter := new(big.Int).Div(AccountBalance, big.NewInt(4))
	specs := []TxSpec{
		spending(0, quarter),
		spending(0, quarter),
		spending(0, quarter),
	}
	senders := []SenderState{{Nonce: 0, Balance: new(big.Int).Set(AccountBalance)}}

	kept := runner.dropOffending(specs, senders, big.NewInt(0))
	require.Len(t, kept, 3, "the domain skipped nothing, so nothing may be dropped")

	require.Equal(t, []*big.Int{
		AccountBalance,
		new(big.Int).Mul(quarter, big.NewInt(3)),
		new(big.Int).Mul(quarter, big.NewInt(2)),
	}, domain.seen)

	require.Equal(t, AccountBalance, senders[0].Balance,
		"the state the caller passed in must not be spent by the walk")
}

// TestDropOffending_KeepsSendersApart checks the running balance is per sender: what one sender
// spends says nothing about what another can cover.
func TestDropOffending_KeepsSendersApart(t *testing.T) {
	domain := &balanceReportingDomain{}
	runner := &Runner{Domain: domain, skipped: map[string]int{}}

	half := new(big.Int).Div(AccountBalance, big.NewInt(2))
	specs := []TxSpec{
		spending(0, half),
		spending(1, half),
		spending(0, big.NewInt(0)),
		spending(1, big.NewInt(0)),
	}
	senders := []SenderState{
		{Balance: new(big.Int).Set(AccountBalance)},
		{Balance: new(big.Int).Set(AccountBalance)},
	}

	runner.dropOffending(specs, senders, big.NewInt(0))

	require.Equal(t, []*big.Int{AccountBalance, AccountBalance, half, half}, domain.seen)
}

// TestDropOffending_ABalanceCannotGoNegative covers a sender drawn to hand away more than it holds,
// which is a case the generator draws deliberately: the next transaction is judged against nothing
// left rather than against a negative balance a policy would read as enormous.
func TestDropOffending_ABalanceCannotGoNegative(t *testing.T) {
	domain := &balanceReportingDomain{}
	runner := &Runner{Domain: domain, skipped: map[string]int{}}

	beyond := new(big.Int).Mul(AccountBalance, big.NewInt(10))
	specs := []TxSpec{spending(0, beyond), spending(0, big.NewInt(1))}
	senders := []SenderState{{Balance: new(big.Int).Set(AccountBalance)}}

	runner.dropOffending(specs, senders, big.NewInt(0))

	require.Equal(t, []*big.Int{AccountBalance, big.NewInt(0)}, domain.seen)
}

// TestDropOffending_ASkippedTransactionSpendsNothing checks that only what is kept is charged against
// the balance: a transaction the domain refuses never reaches a node, so it cannot take anything from
// the sender the transactions behind it are judged against.
func TestDropOffending_ASkippedTransactionSpendsNothing(t *testing.T) {
	half := new(big.Int).Div(AccountBalance, big.NewInt(2))

	var seen []*big.Int
	skipFirst := &fixedSkipDomain{
		skip: func(spec TxSpec) string {
			if spec.Amount().Cmp(half) == 0 {
				return "refused"
			}
			return ""
		},
		report: func(balance *big.Int) { seen = append(seen, new(big.Int).Set(balance)) },
	}
	runner := &Runner{Domain: skipFirst, skipped: map[string]int{}}

	specs := []TxSpec{spending(0, half), spending(0, big.NewInt(0))}
	senders := []SenderState{{Balance: new(big.Int).Set(AccountBalance)}}

	kept := runner.dropOffending(specs, senders, big.NewInt(0))
	require.Len(t, kept, 1)
	require.Equal(t, 1, runner.skipped["refused"], "the reason must be tallied")
	require.Equal(t, []*big.Int{AccountBalance, AccountBalance}, seen,
		"the refused transaction takes nothing, so the one behind it sees the full balance")
}

// fixedSkipDomain skips whatever its predicate names, reporting the balance it was told about.
type fixedSkipDomain struct {
	balanceReportingDomain
	skip   func(TxSpec) string
	report func(*big.Int)
}

func (d *fixedSkipDomain) Skip(spec TxSpec, sender SenderState, _ *big.Int) string {
	d.report(sender.Balance)
	return d.skip(spec)
}
