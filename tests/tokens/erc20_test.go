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

package tokens

import (
	"maps"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc20"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTokenStandards_ERC20(t *testing.T) {
	// Alice starts with 100 tokens.
	cases := map[string]tokenTest[erc20.Erc20, erc20State]{
		"transfer with sufficient funds succeeds": {
			sender:   alice,
			action:   erc20Transfer(carol, 30),
			wantLogs: emits("Transfer", alice, carol, 30),
			want:     erc20Balances(70, 0, 30, 0),
		},
		"transfer of the entire balance succeeds": {
			sender:   alice,
			action:   erc20Transfer(carol, 100),
			wantLogs: emits("Transfer", alice, carol, 100),
			want:     erc20Balances(0, 0, 100, 0),
		},
		"transfer of zero tokens succeeds": {
			sender:   alice,
			action:   erc20Transfer(carol, 0),
			wantLogs: emits("Transfer", alice, carol, 0),
			want:     erc20Balances(100, 0, 0, 0),
		},
		"transfer to oneself succeeds": {
			sender:   alice,
			action:   erc20Transfer(alice, 30),
			wantLogs: emits("Transfer", alice, alice, 30),
			want:     erc20Balances(100, 0, 0, 0),
		},
		"transfer with insufficient funds fails": {
			sender: alice,
			action: erc20Transfer(carol, 101),
		},
		"transfer without funds fails": {
			sender: bob,
			action: erc20Transfer(carol, 1),
		},

		"transferFrom with approval and funds succeeds": {
			setup:    erc20Allowance(50),
			sender:   bob,
			action:   erc20TransferFrom(alice, carol, 40),
			wantLogs: emits("Transfer", alice, carol, 40),
			want:     erc20Balances(60, 0, 40, 10),
		},
		"transferFrom of the entire allowance succeeds": {
			setup:    erc20Allowance(50),
			sender:   bob,
			action:   erc20TransferFrom(alice, carol, 50),
			wantLogs: emits("Transfer", alice, carol, 50),
			want:     erc20Balances(50, 0, 50, 0),
		},
		"transferFrom to the spender succeeds": {
			setup:    erc20Allowance(50),
			sender:   bob,
			action:   erc20TransferFrom(alice, bob, 50),
			wantLogs: emits("Transfer", alice, bob, 50),
			want:     erc20Balances(50, 50, 0, 0),
		},
		"transferFrom without approval fails": {
			sender: bob,
			action: erc20TransferFrom(alice, carol, 1),
		},
		"transferFrom exceeding the allowance fails": {
			setup:  erc20Allowance(50),
			sender: bob,
			action: erc20TransferFrom(alice, carol, 51),
		},
		"transferFrom with approval but insufficient funds fails": {
			setup:  erc20Allowance(200),
			sender: bob,
			action: erc20TransferFrom(alice, carol, 101),
		},
		"transferFrom by a spender approved for someone else fails": {
			setup:  erc20Allowance(50),
			sender: carol,
			action: erc20TransferFrom(alice, carol, 1),
		},
		"transferFrom by a spender approved by another holder fails": {
			setup:  []erc20Step{{carol, erc20Approve(bob, 50)}},
			sender: bob,
			action: erc20TransferFrom(alice, carol, 1),
		},
		"transferFrom after the allowance was revoked fails": {
			setup:  append(erc20Allowance(50), erc20Allowance(0)...),
			sender: bob,
			action: erc20TransferFrom(alice, carol, 1),
		},
		"transferFrom by the holder without self-approval fails": {
			sender: alice,
			action: erc20TransferFrom(alice, carol, 1),
		},

		"approve grants an allowance": {
			sender:   alice,
			action:   erc20Approve(bob, 20),
			wantLogs: emits("Approval", alice, bob, 20),
			want:     erc20Balances(100, 0, 0, 20),
		},
		"approve overwrites a previous allowance": {
			setup:    erc20Allowance(50),
			sender:   alice,
			action:   erc20Approve(bob, 20),
			wantLogs: emits("Approval", alice, bob, 20),
			want:     erc20Balances(100, 0, 0, 20),
		},
		"approve of zero revokes the allowance": {
			setup:    erc20Allowance(50),
			sender:   alice,
			action:   erc20Approve(bob, 0),
			wantLogs: emits("Approval", alice, bob, 0),
			want:     erc20Balances(100, 0, 0, 0),
		},
		"approve beyond the balance succeeds": {
			sender:   alice,
			action:   erc20Approve(bob, 1000),
			wantLogs: emits("Approval", alice, bob, 1000),
			want:     erc20Balances(100, 0, 0, 1000),
		},
	}

	t.Parallel()
	session := tests.StartIntegrationTestNet(t).SpawnSession(t)
	supply := 100 * len(cases)
	token, deployment, shared := deployToken(t, session, deployErc20(supply))

	t.Run("contract is correctly deployed for defined cases", func(t *testing.T) {
		require.Equal(t,
			emits("Transfer", zeroAddress, deployer, supply),
			decodeTokenLogs(t, erc20.Erc20MetaData, deployment, shared, nil),
			"creating the contract must emit a Transfer of the entire supply from the zero address to the deployer")
	})

	names := slices.Sorted(maps.Keys(cases))
	parties := newTokenParties(t, session, shared, len(names))
	fundTokenParties(t, session, token, parties, erc20Transfer(alice, 100))

	for i, name := range names {
		test, p := cases[name], parties[i]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, step := range test.setup {
				applyAs(t, session, p.accounts[step.sender], true, issue(token, p, step.action))
			}
			before := observeErc20(t, token, p.actors)
			receipt := applyAs(t, session, p.accounts[test.sender], test.want != nil, issue(token, p, test.action))

			after := observeErc20(t, token, p.actors)
			require.Equal(t,
				test.wantLogs,
				decodeTokenLogs(t, erc20.Erc20MetaData, receipt, p.actors, nil),
				"the events emitted by the transaction; a reverted transaction emits none")
			if test.want == nil {
				require.Equal(t,
					before, after,
					"a reverted transaction must not change any balance or the allowance")
			} else {
				require.Equal(t,
					*test.want, after,
					"the balances of Alice, Bob and Carol, and Bob's allowance over Alice's tokens after the transaction")
			}
		})
	}
}

type erc20Step = tokenStep[erc20.Erc20]

type erc20State struct {
	balances  [3]uint64 // of Alice, Bob and Carol
	allowance uint64    // of Bob over Alice's tokens
}

func erc20Balances(a, b, c, allowance uint64) *erc20State {
	return &erc20State{[3]uint64{a, b, c}, allowance}
}

func observeErc20(t *testing.T, token *erc20.Erc20, a tokenActors) erc20State {
	var s erc20State
	for i := range s.balances {
		balance, err := token.BalanceOf(nil, a[i])
		require.NoError(t, err, "balanceOf %v", tokenActor(i))
		s.balances[i] = balance.Uint64()
	}
	allowed, err := token.Allowance(nil, a[alice], a[bob])
	require.NoError(t, err, "allowance of Bob over Alice's tokens")
	s.allowance = allowed.Uint64()
	return s
}

func deployErc20(supply int) tests.ContractDeployer[erc20.Erc20] {
	return func(o *bind.TransactOpts, b bind.ContractBackend) (common.Address, *types.Transaction, *erc20.Erc20, error) {
		return erc20.DeployErc20(o, b, big.NewInt(int64(supply)))
	}
}

func erc20Transfer(to tokenActor, value int64) tokenAction[erc20.Erc20] {
	return func(o *bind.TransactOpts, c *erc20.Erc20, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.Transfer(o, a[to], big.NewInt(value))
	}
}

func erc20TransferFrom(from, to tokenActor, value int64) tokenAction[erc20.Erc20] {
	return func(o *bind.TransactOpts, c *erc20.Erc20, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.TransferFrom(o, a[from], a[to], big.NewInt(value))
	}
}

func erc20Approve(spender tokenActor, value int64) tokenAction[erc20.Erc20] {
	return func(o *bind.TransactOpts, c *erc20.Erc20, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.Approve(o, a[spender], big.NewInt(value))
	}
}

// erc20Allowance is the setup of Alice granting Bob the given allowance.
func erc20Allowance(value int64) []erc20Step {
	return []erc20Step{{alice, erc20Approve(bob, value)}}
}
