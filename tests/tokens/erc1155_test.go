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
	"github.com/0xsoniclabs/sonic/tests/contracts/erc1155"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTokenStandards_ERC1155(t *testing.T) {
	operatorBob := []erc1155Step{{alice, erc1155SetApprovalForAll(bob, true)}}
	operatorOfCarol := []erc1155Step{{carol, erc1155SetApprovalForAll(bob, true)}}
	revokedOperator := append(operatorBob, erc1155Step{alice, erc1155SetApprovalForAll(bob, false)})
	initial := erc1155Balances{{10, 5}, {0, 0}, {0, 0}}

	// Alice starts with 10 tokens of id 1 and 5 tokens of id 2.
	cases := map[string]tokenTest[erc1155.Erc1155, erc1155State]{
		"safeTransferFrom by the holder succeeds": {
			sender:   alice,
			action:   erc1155SafeTransferFrom(alice, carol, 1, 3),
			wantLogs: emits("TransferSingle", alice, alice, carol, 1, 3),
			want:     erc1155Held(erc1155Balances{{7, 5}, {0, 0}, {3, 0}}, false),
		},
		"safeTransferFrom by an operator succeeds": {
			setup:    operatorBob,
			sender:   bob,
			action:   erc1155SafeTransferFrom(alice, carol, 1, 3),
			wantLogs: emits("TransferSingle", bob, alice, carol, 1, 3),
			want:     erc1155Held(erc1155Balances{{7, 5}, {0, 0}, {3, 0}}, true),
		},
		"safeTransferFrom of the entire balance succeeds": {
			sender:   alice,
			action:   erc1155SafeTransferFrom(alice, carol, 2, 5),
			wantLogs: emits("TransferSingle", alice, alice, carol, 2, 5),
			want:     erc1155Held(erc1155Balances{{10, 0}, {0, 0}, {0, 5}}, false),
		},
		"safeTransferFrom of zero tokens succeeds": {
			sender:   alice,
			action:   erc1155SafeTransferFrom(alice, carol, 1, 0),
			wantLogs: emits("TransferSingle", alice, alice, carol, 1, 0),
			want:     erc1155Held(initial, false),
		},
		"safeTransferFrom to oneself succeeds": {
			sender:   alice,
			action:   erc1155SafeTransferFrom(alice, alice, 1, 3),
			wantLogs: emits("TransferSingle", alice, alice, alice, 1, 3),
			want:     erc1155Held(initial, false),
		},
		"safeTransferFrom to an accepting receiver succeeds": {
			sender: alice,
			action: erc1155SafeTransferFrom(alice, acceptingReceiver, 1, 3),
			wantLogs: emits("TransferSingle", alice, alice, acceptingReceiver, 1, 3).
				then("ERC1155Received", alice, alice, 1, 3, erc1155HookData),
			want: erc1155Held(erc1155Balances{{7, 5}, {0, 0}, {0, 0}}, false),
		},
		"safeTransferFrom by an operator to an accepting receiver succeeds": {
			setup:  operatorBob,
			sender: bob,
			action: erc1155SafeTransferFrom(alice, acceptingReceiver, 1, 3),
			wantLogs: emits("TransferSingle", bob, alice, acceptingReceiver, 1, 3).
				then("ERC1155Received", bob, alice, 1, 3, erc1155HookData),
			want: erc1155Held(erc1155Balances{{7, 5}, {0, 0}, {0, 0}}, true),
		},
		"safeTransferFrom without approval fails": {
			sender: bob,
			action: erc1155SafeTransferFrom(alice, carol, 1, 3),
		},
		"safeTransferFrom by an operator of another holder fails": {
			setup:  operatorOfCarol,
			sender: bob,
			action: erc1155SafeTransferFrom(alice, carol, 1, 3),
		},
		"safeTransferFrom by a revoked operator fails": {
			setup:  revokedOperator,
			sender: bob,
			action: erc1155SafeTransferFrom(alice, carol, 1, 3),
		},
		"safeTransferFrom with insufficient funds fails": {
			sender: alice,
			action: erc1155SafeTransferFrom(alice, carol, 2, 6),
		},
		"safeTransferFrom of an unminted id fails": {
			sender: alice,
			action: erc1155SafeTransferFrom(alice, carol, 3, 1),
		},
		"safeTransferFrom to the zero address fails": {
			sender: alice,
			action: erc1155SafeTransferFrom(alice, zeroAddress, 1, 3),
		},
		"safeTransferFrom to a contract without receiver hook fails": {
			sender: alice,
			action: erc1155SafeTransferFrom(alice, tokenContract, 1, 3),
		},
		"safeTransferFrom to a rejecting receiver fails": {
			sender: alice,
			action: erc1155SafeTransferFrom(alice, rejectingReceiver, 1, 3),
		},

		"safeBatchTransferFrom by the holder succeeds": {
			sender:   alice,
			action:   erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3, 5}),
			wantLogs: emits("TransferBatch", alice, alice, carol, []int{1, 2}, []int{3, 5}),
			want:     erc1155Held(erc1155Balances{{7, 0}, {0, 0}, {3, 5}}, false),
		},
		"safeBatchTransferFrom by an operator succeeds": {
			setup:    operatorBob,
			sender:   bob,
			action:   erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3, 5}),
			wantLogs: emits("TransferBatch", bob, alice, carol, []int{1, 2}, []int{3, 5}),
			want:     erc1155Held(erc1155Balances{{7, 0}, {0, 0}, {3, 5}}, true),
		},
		"safeBatchTransferFrom of an empty batch succeeds": {
			sender:   alice,
			action:   erc1155SafeBatchTransferFrom(alice, carol, nil, nil),
			wantLogs: emits("TransferBatch", alice, alice, carol, []int{}, []int{}),
			want:     erc1155Held(initial, false),
		},
		"safeBatchTransferFrom with repeated ids within the balance succeeds": {
			sender:   alice,
			action:   erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 1}, []int64{6, 4}),
			wantLogs: emits("TransferBatch", alice, alice, carol, []int{1, 1}, []int{6, 4}),
			want:     erc1155Held(erc1155Balances{{0, 5}, {0, 0}, {10, 0}}, false),
		},
		"safeBatchTransferFrom to an accepting receiver succeeds": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, acceptingReceiver, []int64{1, 2}, []int64{3, 5}),
			wantLogs: emits("TransferBatch", alice, alice, acceptingReceiver, []int{1, 2}, []int{3, 5}).
				then("ERC1155BatchReceived", alice, alice, []int{1, 2}, []int{3, 5}, erc1155HookData),
			want: erc1155Held(erc1155Balances{{7, 0}, {0, 0}, {0, 0}}, false),
		},
		"safeBatchTransferFrom without approval fails": {
			sender: bob,
			action: erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3, 5}),
		},
		"safeBatchTransferFrom by an operator of another holder fails": {
			setup:  operatorOfCarol,
			sender: bob,
			action: erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3, 5}),
		},
		"safeBatchTransferFrom by a revoked operator fails": {
			setup:  revokedOperator,
			sender: bob,
			action: erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3, 5}),
		},
		"safeBatchTransferFrom with insufficient funds for one id fails entirely": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3, 6}),
		},
		"safeBatchTransferFrom with repeated ids exceeding the balance fails": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 1}, []int64{6, 5}),
		},
		"safeBatchTransferFrom with mismatched lengths fails": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, carol, []int64{1, 2}, []int64{3}),
		},
		"safeBatchTransferFrom to the zero address fails": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, zeroAddress, []int64{1, 2}, []int64{3, 5}),
		},
		"safeBatchTransferFrom to a contract without receiver hook fails": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, tokenContract, []int64{1, 2}, []int64{3, 5}),
		},
		"safeBatchTransferFrom to a rejecting receiver fails": {
			sender: alice,
			action: erc1155SafeBatchTransferFrom(alice, rejectingReceiver, []int64{1, 2}, []int64{3, 5}),
		},

		"setApprovalForAll grants an operator": {
			sender:   alice,
			action:   erc1155SetApprovalForAll(bob, true),
			wantLogs: emits("ApprovalForAll", alice, bob, true),
			want:     erc1155Held(initial, true),
		},
		"setApprovalForAll revokes an operator": {
			setup:    operatorBob,
			sender:   alice,
			action:   erc1155SetApprovalForAll(bob, false),
			wantLogs: emits("ApprovalForAll", alice, bob, false),
			want:     erc1155Held(initial, false),
		},
	}

	t.Parallel()
	session := tests.StartIntegrationTestNet(t).SpawnSession(t)
	n := len(cases)
	token, deployment, shared := deployToken(t, session, deployErc1155([]int64{10 * int64(n), 5 * int64(n)}))
	shared = addTokenReceivers(t, session, shared)

	t.Run("contract is correctly deployed for defined cases", func(t *testing.T) {
		require.Equal(t,
			emits("TransferBatch", deployer, zeroAddress, deployer, []int{1, 2}, []int{10 * n, 5 * n}),
			decodeTokenLogs(t, erc1155Events, deployment, shared, nil),
			"creating the contract must emit a TransferBatch of all ids from the zero address to the deployer")

		checkSupportedInterfaces(t, token.SupportsInterface, map[[4]byte]bool{
			erc165InterfaceId:  true,
			erc721InterfaceId:  false,
			erc1155InterfaceId: true,
		})
		_, err := token.BalanceOfBatch(nil, []common.Address{shared[deployer]}, toBigInts(erc1155Ids))
		require.Error(t, err, "balanceOfBatch with mismatched lengths must revert")
	})

	names := slices.Sorted(maps.Keys(cases))
	parties := newTokenParties(t, session, shared, len(names))
	fundTokenParties(t, session, token, parties, erc1155SafeBatchTransferFrom(deployer, alice, erc1155Ids, []int64{10, 5}))

	for i, name := range names {
		test, p := cases[name], parties[i]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, step := range test.setup {
				applyAs(t, session, p.accounts[step.sender], true, issue(token, p, step.action))
			}
			before := observeErc1155(t, token, p.actors)
			receipt := applyAs(t, session, p.accounts[test.sender], test.want != nil, issue(token, p, test.action))

			after := observeErc1155(t, token, p.actors)
			require.Equal(t,
				test.wantLogs,
				decodeTokenLogs(t, erc1155Events, receipt, p.actors, nil),
				"the events emitted by the transaction; a reverted transaction emits none")
			if test.want == nil {
				require.Equal(t,
					before, after,
					"a reverted transaction must not change any balance or the operator")
			} else {
				require.Equal(t,
					*test.want, after,
					"the balances of Alice, Bob and Carol for ids 1 and 2, and whether Bob is Alice's operator after the transaction")
			}
		})
	}
}

// erc1155Events are the events as EIP-1155 specifies them.
var erc1155Events = parseEvents(`[
	{"type": "event", "name": "TransferSingle", "inputs": [
		{"name": "operator", "type": "address", "indexed": true},
		{"name": "from", "type": "address", "indexed": true},
		{"name": "to", "type": "address", "indexed": true},
		{"name": "id", "type": "uint256"},
		{"name": "value", "type": "uint256"}]},
	{"type": "event", "name": "TransferBatch", "inputs": [
		{"name": "operator", "type": "address", "indexed": true},
		{"name": "from", "type": "address", "indexed": true},
		{"name": "to", "type": "address", "indexed": true},
		{"name": "ids", "type": "uint256[]"},
		{"name": "values", "type": "uint256[]"}]},
	{"type": "event", "name": "ApprovalForAll", "inputs": [
		{"name": "owner", "type": "address", "indexed": true},
		{"name": "operator", "type": "address", "indexed": true},
		{"name": "approved", "type": "bool"}]},
	{"type": "event", "name": "URI", "inputs": [
		{"name": "value", "type": "string"},
		{"name": "id", "type": "uint256", "indexed": true}]}
]`)

type erc1155Step = tokenStep[erc1155.Erc1155]

// erc1155Balances are those of Alice, Bob and Carol for ids 1 and 2.
type erc1155Balances = [3][2]uint64

type erc1155State struct {
	balances erc1155Balances
	operator bool // whether Bob is an operator for Alice
}

func erc1155Held(b erc1155Balances, operator bool) *erc1155State {
	return &erc1155State{b, operator}
}

// erc1155Ids are the ids created for the deployer, of which Alice of each case
// receives 10 and 5 tokens.
var erc1155Ids = []int64{1, 2}

// erc1155HookData is passed to every transfer, so that receivers log it.
var erc1155HookData = []byte{0x01, 0x02}

// observeErc1155 also checks that balanceOfBatch agrees with balanceOf.
func observeErc1155(t *testing.T, token *erc1155.Erc1155, a tokenActors) erc1155State {
	t.Helper()
	var s erc1155State
	var owners []common.Address
	var ids []*big.Int
	for holder := range s.balances {
		for i, id := range toBigInts(erc1155Ids) {
			balance, err := token.BalanceOf(nil, a[holder], id)
			require.NoError(t, err, "balanceOf %v for id %v", tokenActor(holder), id)
			s.balances[holder][i] = balance.Uint64()
			owners, ids = append(owners, a[holder]), append(ids, id)
		}
	}
	batch, err := token.BalanceOfBatch(nil, owners, ids)
	require.NoError(t, err, "balanceOfBatch of Alice, Bob and Carol")
	require.Len(t, batch, len(owners), "balanceOfBatch must return one balance per owner and id")
	for i, balance := range batch {
		holder, id := i/len(erc1155Ids), i%len(erc1155Ids)
		require.Equal(t,
			s.balances[holder][id], balance.Uint64(),
			"balanceOfBatch must agree with balanceOf for %v and id %d", tokenActor(holder), erc1155Ids[id])
	}
	operator, err := token.IsApprovedForAll(nil, a[alice], a[bob])
	require.NoError(t, err, "isApprovedForAll(Alice, Bob)")
	s.operator = operator
	return s
}

func deployErc1155(values []int64) tests.ContractDeployer[erc1155.Erc1155] {
	return func(o *bind.TransactOpts, b bind.ContractBackend) (common.Address, *types.Transaction, *erc1155.Erc1155, error) {
		return erc1155.DeployErc1155(o, b, toBigInts(erc1155Ids), toBigInts(values))
	}
}

func erc1155SafeTransferFrom(from, to tokenActor, id, value int64) tokenAction[erc1155.Erc1155] {
	return func(o *bind.TransactOpts, c *erc1155.Erc1155, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.SafeTransferFrom(o, a[from], a[to], big.NewInt(id), big.NewInt(value), erc1155HookData)
	}
}

func erc1155SafeBatchTransferFrom(from, to tokenActor, ids, values []int64) tokenAction[erc1155.Erc1155] {
	return func(o *bind.TransactOpts, c *erc1155.Erc1155, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.SafeBatchTransferFrom(o, a[from], a[to], toBigInts(ids), toBigInts(values), erc1155HookData)
	}
}

func erc1155SetApprovalForAll(operator tokenActor, approved bool) tokenAction[erc1155.Erc1155] {
	return func(o *bind.TransactOpts, c *erc1155.Erc1155, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.SetApprovalForAll(o, a[operator], approved)
	}
}

func toBigInts(values []int64) []*big.Int {
	res := make([]*big.Int, len(values))
	for i, v := range values {
		res[i] = big.NewInt(v)
	}
	return res
}
