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

package exec_test

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func noMethods(*dispatch.Dispatcher, *exec.Environment) {}

// newEnvironment returns the environment of a harness over a core holding
// three blocks, the first two archived, and the head as a request sees it.
func newEnvironment(t *testing.T) (*exec.Environment, *tests.CarmenCore, exec.Head) {
	t.Helper()
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		common.HexToAddress("0x01"): {Balance: big.NewInt(5)},
	})
	transaction := types.NewTx(&types.LegacyTx{Nonce: 1, Gas: 21000})
	carmenCore.AppendBlock(tests.BlockSpec{Transactions: types.Transactions{transaction}})
	carmenCore.PublishHead(1)
	carmenCore.AppendBlock(tests.BlockSpec{})
	harness := tests.NewHarness(t, carmenCore, noMethods)
	head, err := harness.Environment.Latest(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(1), head.Number)
	return harness.Environment, carmenCore, head
}

func TestEnvironment_LookupsAnswerAtTheHead(t *testing.T) {
	environment, carmenCore, head := newEnvironment(t)

	header, err := environment.HeaderAt(head, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), header.Number.Uint64())
	block, err := environment.BlockAt(head, 1)
	require.NoError(t, err)
	require.Len(t, block.Transactions, 1)
	receipts, err := environment.ReceiptsAt(head, 1)
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	located, err := environment.TransactionByHash(head, block.Transactions[0].Hash())
	require.NoError(t, err)
	require.Equal(t, exec.Located{Transaction: located.Transaction, Block: 1, Index: 0}, *located)

	before := carmenCore.States()
	state, stateHeader, err := environment.StateAt(head, chain.Latest())
	require.NoError(t, err)
	require.Equal(t, uint64(1), stateHeader.Number.Uint64())
	require.Equal(t, uint64(5), state.GetBalance(common.HexToAddress("0x01")).Uint64())
	state.Release()
	require.Equal(t, tests.StateCount{Opened: before.Opened + 1, Released: before.Released + 1}, carmenCore.States())
}

func TestEnvironment_NothingAboveTheHeadIsServed(t *testing.T) {
	environment, carmenCore, head := newEnvironment(t)
	unpublished, err := carmenCore.Block(2)
	require.NoError(t, err)
	before := carmenCore.Calls("Block")

	_, err = environment.HeaderAt(head, 2)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.BlockAt(head, 2)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.ReceiptsAt(head, 2)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.OpenState(head, 2)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.FindBlock(head, chain.ReferenceToHash(unpublished.Hash))
	require.ErrorIs(t, err, chain.ErrUnknownHash)
	_, err = environment.RulesAt(head, 3)
	require.ErrorIs(t, err, core.ErrNotFound)
	require.Equal(t, before, carmenCore.Calls("Block"), "a block above the head is refused without asking the node")

	rules, err := environment.RulesAt(head, 2)
	require.NoError(t, err, "the block after the head has the rules it is processed with")
	require.Equal(t, uint64(tests.FakeChainID), rules.NetworkID)
}

func TestEnvironment_ATransactionAboveTheHeadIsNotFound(t *testing.T) {
	fake := tests.NewFakeCore(t)
	transaction := types.NewTx(&types.LegacyTx{Nonce: 3})
	fake.AppendBlock(tests.BlockSpec{Transactions: types.Transactions{transaction}})
	harness := tests.NewHarness(t, fake, noMethods)
	head, err := harness.Environment.Latest(context.Background())
	require.NoError(t, err)
	_, err = harness.Environment.TransactionByHash(head, transaction.Hash())
	require.ErrorIs(t, err, core.ErrNotFound)
}

// TestEnvironment_LookupsNeverAnswerNothingWithoutAnError holds the rule of
// one way to say "not found": a core that answers nil without an error is
// read as not found.
func TestEnvironment_LookupsNeverAnswerNothingWithoutAnError(t *testing.T) {
	environment, carmenCore, head := newEnvironment(t)
	carmenCore.SetAllModes(tests.Nil)

	_, err := environment.HeaderAt(head, 1)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.BlockAt(head, 1)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.ReceiptsAt(head, 1)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.OpenState(head, 1)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = environment.TransactionByHash(head, common.Hash{})
	require.ErrorIs(t, err, core.ErrNotFound)
}

// TestEnvironment_ACancelledRequestMakesNoMoreCalls holds ADR16: a lookup
// checks the context before it asks the node.
func TestEnvironment_ACancelledRequestMakesNoMoreCalls(t *testing.T) {
	environment, carmenCore, head := newEnvironment(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := environment.View(ctx, head.Head)
	before := map[string]int{}
	for _, method := range tests.CoreMethods {
		before[method] = carmenCore.Calls(method)
	}

	lookups := map[string]func() error{
		"HeaderAt":          func() error { _, err := environment.HeaderAt(cancelled, 1); return err },
		"BlockAt":           func() error { _, err := environment.BlockAt(cancelled, 1); return err },
		"ReceiptsAt":        func() error { _, err := environment.ReceiptsAt(cancelled, 1); return err },
		"RulesAt":           func() error { _, err := environment.RulesAt(cancelled, 1); return err },
		"TransactionByHash": func() error { _, err := environment.TransactionByHash(cancelled, common.Hash{}); return err },
		"OpenState":         func() error { _, err := environment.OpenState(cancelled, 1); return err },
		"Resolve by hash": func() error {
			_, err := environment.Resolve(cancelled, chain.ReferenceToHash(common.Hash{}))
			return err
		},
	}
	for name, lookup := range lookups {
		require.ErrorIs(t, lookup(), context.Canceled, name)
	}
	for _, method := range tests.CoreMethods {
		if slices.Contains(tests.LoopMethods, method) {
			continue
		}
		require.Equal(t, before[method], carmenCore.Calls(method), "a cancelled request called %s", method)
	}
}

// TestEnvironment_AStateOpenedWhileTheRequestWasCancelledIsReleased holds
// ADR16 and I7 for the call that is already under way when the cancel comes.
func TestEnvironment_AStateOpenedWhileTheRequestWasCancelledIsReleased(t *testing.T) {
	environment, carmenCore, head := newEnvironment(t)
	ctx, cancel := context.WithCancel(context.Background())
	release := carmenCore.Hold("State")
	opened := make(chan error, 1)
	go func() {
		_, err := environment.OpenState(environment.View(ctx, head.Head), 1)
		opened <- err
	}()
	tests.Eventually(t, func() bool { return carmenCore.Calls("State") == 1 }, "the state to be opened")
	cancel()
	release()
	require.ErrorIs(t, <-opened, context.Canceled)
	count := carmenCore.States()
	require.Equal(t, count.Opened, count.Released)
}

func TestEnvironment_HandsOnTheFailuresOfTheNode(t *testing.T) {
	environment, carmenCore, head := newEnvironment(t)
	carmenCore.SetAllModes(tests.Fail)
	_, err := environment.HeaderAt(head, 1)
	require.ErrorIs(t, err, tests.ErrInjected)
	_, _, err = environment.StateAt(head, chain.Latest())
	require.ErrorIs(t, err, tests.ErrInjected)
	_, err = environment.FindHeader(head, chain.ReferenceToHash(common.Hash{}))
	require.ErrorIs(t, err, tests.ErrInjected)
}

func TestEnvironment_ChainFacts(t *testing.T) {
	environment, _, head := newEnvironment(t)
	require.Equal(t, big.NewInt(tests.FakeChainID), environment.ChainID())
	require.Equal(t, big.NewInt(tests.FakeChainID), environment.ChainConfig(1).ChainID)
	require.Equal(t, environment.ChainID(), environment.Signer().ChainID())
	require.Equal(t, head.Context(), context.Background())
	require.Equal(t, exec.DefaultConfig(), environment.Config())
	require.Equal(t, uint64(tests.FakeChainID), environment.ChainInfo().ChainID)
	require.Equal(t, 0, exec.SuggestedGasPrice(head).Cmp(big.NewInt(1_100_000_000)), "the base fee of 1 gwei plus 10%")
}

func TestNullIfAbsent_AnswersNullOnlyForWhatDoesNotExist(t *testing.T) {
	value, err := exec.NullIfAbsent(5, core.ErrNotFound)
	require.NoError(t, err)
	require.Zero(t, value)
	value, err = exec.NullIfAbsent(5, chain.ErrUnknownHash)
	require.NoError(t, err)
	require.Zero(t, value)
	_, err = exec.NullIfAbsent(5, chain.ErrInvalidTag)
	require.ErrorIs(t, err, chain.ErrInvalidTag, "an invalid tag stays an error")
	failure := errors.New("failure")
	_, err = exec.NullIfAbsent(5, failure)
	require.Equal(t, failure, err)
	value, err = exec.NullIfAbsent(5, nil)
	require.NoError(t, err)
	require.Equal(t, 5, value)
}

func TestUnknownHashError_IsWordedAsTheReference(t *testing.T) {
	require.EqualError(t, exec.UnknownHashError(common.HexToHash("0x01")),
		"block with hash 0x0000000000000000000000000000000000000000000000000000000000000001 not found")
}
