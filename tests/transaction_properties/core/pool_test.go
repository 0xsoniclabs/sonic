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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAccountPool_DerivesTheSameKeysEveryTime(t *testing.T) {
	first, err := NewAccountPool(8)
	require.NoError(t, err)
	second, err := NewAccountPool(8)
	require.NoError(t, err)

	// The genesis written at startup and the keys the test signs with are derived separately, so they
	// have to agree without anything being persisted between them.
	for i := range first.Accounts {
		require.Equal(t, first.Accounts[i].Address(), second.Accounts[i].Address())
	}

	addresses := map[common.Address]bool{}
	for _, account := range first.Accounts {
		addresses[account.Address()] = true
	}
	require.Len(t, addresses, 8, "every account must be distinct")
}

func TestAccountPool_FundsEveryAccountInGenesis(t *testing.T) {
	pool, err := NewAccountPool(4)
	require.NoError(t, err)

	genesis := pool.GenesisAccounts()
	require.Len(t, genesis, 4)
	for i, account := range genesis {
		require.Equal(t, pool.Accounts[i].Address(), account.Address)
		require.Equal(t, uint256.MustFromBig(AccountBalance), account.Balance)
	}
}

func TestAccountPool_HandsOutUntouchedAccountsFirst(t *testing.T) {
	pool, err := NewAccountPool(3)
	require.NoError(t, err)

	claimed, err := pool.Claim(2)
	require.NoError(t, err)
	require.Len(t, claimed, 2)

	// One executed a transaction, one did not.
	pool.Release(claimed[0], true)
	pool.Release(claimed[1], false)

	handedOut, dirtied, untouched := pool.Stats()
	require.Equal(t, 2, handedOut)
	require.Equal(t, 1, dirtied)
	require.Equal(t, 2, untouched, "the clean account rejoins the untouched list")

	// While an untouched account remains, the dirty one is not handed out again: a fresh account makes
	// a shrink replay more likely to see what the original run saw.
	next, err := pool.Claim(2)
	require.NoError(t, err)
	for _, account := range next {
		require.NotEqual(t, claimed[0].Index, account.Index)
	}
}

func TestAccountPool_ReusesDirtiedAccountsOnceTheUntouchedRunOut(t *testing.T) {
	pool, err := NewAccountPool(2)
	require.NoError(t, err)

	first, err := pool.Claim(2)
	require.NoError(t, err)
	for _, account := range first {
		pool.Release(account, true)
	}

	_, _, untouched := pool.Stats()
	require.Zero(t, untouched)

	// Only now can a sender start mid-sequence, which is what makes a too-low nonce meaningful.
	second, err := pool.Claim(2)
	require.NoError(t, err)
	require.Len(t, second, 2)
}

func TestAccountPool_ClaimFailsRatherThanHandingOutAnAccountTwice(t *testing.T) {
	pool, err := NewAccountPool(2)
	require.NoError(t, err)

	_, err = pool.Claim(3)
	require.ErrorContains(t, err, "3 accounts requested but only 2 of 2 are free")

	claimed, err := pool.Claim(2)
	require.NoError(t, err)
	_, err = pool.Claim(1)
	require.ErrorContains(t, err, "only 0 of 2 are free")

	for _, account := range claimed {
		pool.Release(account, false)
	}
	_, err = pool.Claim(1)
	require.NoError(t, err)
}

func TestDeriveKey_IsDeterministicAndDistinctPerIndex(t *testing.T) {
	first, err := DeriveKey(7)
	require.NoError(t, err)
	again, err := DeriveKey(7)
	require.NoError(t, err)
	other, err := DeriveKey(8)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(first.PublicKey), crypto.PubkeyToAddress(again.PublicKey))
	require.NotEqual(t, crypto.PubkeyToAddress(first.PublicKey), crypto.PubkeyToAddress(other.PublicKey))
}
