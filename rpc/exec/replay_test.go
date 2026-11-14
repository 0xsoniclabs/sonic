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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// replayChain serves block 1 with two transfers of a funded account.
func replayChain(t *testing.T) (*tests.Harness, *tests.Account, types.Transactions) {
	t.Helper()
	account := tests.NewAccount()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{account.Address(): {Balance: big.NewInt(1e18)}})
	recipient := common.Address{0x42}
	transfers := types.Transactions{}
	for nonce := range uint64(2) {
		transfers = append(transfers, account.Sign(&types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), Nonce: nonce, To: &recipient,
			Gas: 21000, GasFeeCap: big.NewInt(2e9), GasTipCap: big.NewInt(1), Value: big.NewInt(5)}))
	}
	fake.AddBlock(transfers...)
	fake.SetHead(1)
	return tests.NewHarness(t, fake, func(*tests.Harness) {}), account, transfers
}

// The state of a transaction holds what the transactions before it did.
func TestReplayState_AppliesTheTransactionsBefore(t *testing.T) {
	h, account, transfers := replayChain(t)
	head := h.Environment.Head()
	block, err := h.Environment.BlockAt(head, 1)
	require.NoError(t, err)
	for index := range transfers {
		message, stateDB, err := h.Environment.ReplayState(head, block, index)
		require.NoError(t, err)
		require.Equal(t, uint64(index), message.Nonce)
		require.Equal(t, uint64(index), stateDB.GetNonce(account.Address()), "the transactions before ran")
		stateDB.Release()
	}
	require.Zero(t, h.Core.OpenStates())
}

func TestReplayState_RefusesWhatItCannotReplay(t *testing.T) {
	h, _, transfers := replayChain(t)
	head := h.Environment.Head()
	genesis, err := h.Environment.BlockAt(head, 0)
	require.NoError(t, err)
	block, err := h.Environment.BlockAt(head, 1)
	require.NoError(t, err)
	_, _, err = h.Environment.ReplayState(head, genesis, 0)
	require.EqualError(t, err, "no transaction in genesis")
	for _, index := range []int{len(transfers), len(transfers) + 1} {
		_, _, err = h.Environment.ReplayState(head, block, index)
		require.ErrorContains(t, err, "out of range for block")
	}
	h.Core.SetMode("State", tests.Fail)
	_, _, err = h.Environment.ReplayState(head, block, 0)
	require.ErrorIs(t, err, tests.ErrInjected)
	require.Zero(t, h.Core.OpenStates())
}

// A block without transactions replays to the state after its system calls.
func TestReplayState_OfABlockWithoutTransactions(t *testing.T) {
	h := newStateEnvironment(t)
	head := h.Environment.Head()
	message, stateDB, err := h.Environment.ReplayState(head, head, 0)
	require.NoError(t, err)
	require.Nil(t, message)
	stateDB.Release()
	h.Core.SetMode("Rules", tests.Fail)
	_, _, err = h.Environment.ReplayState(head, head, 0)
	require.ErrorContains(t, err, "failed to get vm config")
}

// The hooked state reports every change and log to the hooks, and the
// burning of what reached a self-destructed account.
func TestWithHooks_ReportsTheChangesOfTheState(t *testing.T) {
	h := newStateEnvironment(t)
	stateDB, _, err := h.Environment.StateAt(h.Environment.Head(), chain.ReferenceTo(1))
	require.NoError(t, err)
	defer stateDB.Release()
	var reported []string
	hooked := exec.WithHooks(stateDB, &tracing.Hooks{
		OnBalanceChange: func(common.Address, *big.Int, *big.Int, tracing.BalanceChangeReason) {
			reported = append(reported, "balance")
		},
		OnNonceChange:   func(common.Address, uint64, uint64) { reported = append(reported, "nonce") },
		OnCodeChange:    func(common.Address, common.Hash, []byte, common.Hash, []byte) { reported = append(reported, "code") },
		OnStorageChange: func(common.Address, common.Hash, common.Hash, common.Hash) { reported = append(reported, "storage") },
		OnLog:           func(*types.Log) { reported = append(reported, "log") },
	})
	account := common.Address{0x77}
	hooked.AddBalance(account, uint256.NewInt(3), tracing.BalanceChangeUnspecified)
	hooked.AddBalance(account, uint256.NewInt(0), tracing.BalanceChangeUnspecified)
	hooked.SubBalance(account, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	hooked.SetNonce(account, 1, tracing.NonceChangeUnspecified)
	hooked.SetCode(account, []byte{0x00}, tracing.CodeChangeUnspecified)
	hooked.SetState(account, common.Hash{}, common.Hash{0x01})
	hooked.AddLog(&types.Log{})
	hooked.SelfDestruct(account)
	hooked.AddBalance(account, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	hooked.EndTransaction()
	require.Equal(t, []string{"balance", "balance", "nonce", "code", "storage", "log", "balance", "balance", "balance"}, reported)

	silent := exec.WithHooks(stateDB, &tracing.Hooks{})
	silent.SelfDestruct(account)
	silent.EndTransaction()
}
