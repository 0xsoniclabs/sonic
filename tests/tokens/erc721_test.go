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
	"fmt"
	"maps"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/erc721"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTokenStandards_ERC721(t *testing.T) {
	approvedBob := []erc721Step{{alice, erc721Approve(bob)}}
	operatorBob := []erc721Step{{alice, erc721SetApprovalForAll(bob, true)}}

	// Alice starts owning the case's token.
	cases := map[string]tokenTest[erc721.Erc721, erc721State]{
		"transferFrom by the owner succeeds": {
			sender:   alice,
			action:   erc721TransferFrom(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, false),
		},
		"transferFrom by the approved address succeeds and clears the approval": {
			setup:    approvedBob,
			sender:   bob,
			action:   erc721TransferFrom(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, false),
		},
		"transferFrom by an operator succeeds": {
			setup:    operatorBob,
			sender:   bob,
			action:   erc721TransferFrom(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, true),
		},
		"transferFrom to the approved address succeeds": {
			setup:    approvedBob,
			sender:   bob,
			action:   erc721TransferFrom(alice, bob),
			wantLogs: emits("Transfer", alice, bob, caseToken),
			want:     erc721Owned(bob, zeroAddress, false),
		},
		"transferFrom to oneself succeeds and clears the approval": {
			setup:    approvedBob,
			sender:   alice,
			action:   erc721TransferFrom(alice, alice),
			wantLogs: emits("Transfer", alice, alice, caseToken),
			want:     erc721Owned(alice, zeroAddress, false),
		},
		"transferFrom to a contract without receiver hook succeeds": {
			sender:   alice,
			action:   erc721TransferFrom(alice, tokenContract),
			wantLogs: emits("Transfer", alice, tokenContract, caseToken),
			want:     erc721Owned(tokenContract, zeroAddress, false),
		},
		"transferFrom to a rejecting receiver succeeds without calling it": {
			sender:   alice,
			action:   erc721TransferFrom(alice, rejectingReceiver),
			wantLogs: emits("Transfer", alice, rejectingReceiver, caseToken),
			want:     erc721Owned(rejectingReceiver, zeroAddress, false),
		},
		"transferFrom without approval fails": {
			sender: bob,
			action: erc721TransferFrom(alice, carol),
		},
		"transferFrom by an address other than the approved one fails": {
			setup:  approvedBob,
			sender: carol,
			action: erc721TransferFrom(alice, carol),
		},
		"transferFrom by a formerly approved address fails": {
			setup:  append(approvedBob, erc721Step{alice, erc721Approve(carol)}),
			sender: bob,
			action: erc721TransferFrom(alice, carol),
		},
		"transferFrom by an operator of another owner fails": {
			setup:  []erc721Step{{carol, erc721SetApprovalForAll(bob, true)}},
			sender: bob,
			action: erc721TransferFrom(alice, carol),
		},
		"transferFrom by a revoked operator fails": {
			setup:  append(operatorBob, erc721Step{alice, erc721SetApprovalForAll(bob, false)}),
			sender: bob,
			action: erc721TransferFrom(alice, carol),
		},
		"transferFrom from someone other than the owner fails": {
			sender: alice,
			action: erc721TransferFrom(bob, carol),
		},
		"transferFrom to the zero address fails": {
			sender: alice,
			action: erc721TransferFrom(alice, zeroAddress),
		},
		"transferFrom of an unminted token fails": {
			sender: alice,
			action: erc721OnUnmintedToken(erc721TransferFrom(alice, carol)),
		},

		"safeTransferFrom to an account succeeds": {
			sender:   alice,
			action:   erc721SafeTransferFrom(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, false),
		},
		"safeTransferFrom with data to an account succeeds": {
			sender:   alice,
			action:   erc721SafeTransferFromWithData(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, false),
		},
		"safeTransferFrom by the approved address succeeds": {
			setup:    approvedBob,
			sender:   bob,
			action:   erc721SafeTransferFrom(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, false),
		},
		"safeTransferFrom by an operator succeeds": {
			setup:    operatorBob,
			sender:   bob,
			action:   erc721SafeTransferFrom(alice, carol),
			wantLogs: emits("Transfer", alice, carol, caseToken),
			want:     erc721Owned(carol, zeroAddress, true),
		},
		"safeTransferFrom to an accepting receiver succeeds": {
			sender: alice,
			action: erc721SafeTransferFrom(alice, acceptingReceiver),
			wantLogs: emits("Transfer", alice, acceptingReceiver, caseToken).
				then("ERC721Received", alice, alice, caseToken, []byte{}),
			want: erc721Owned(acceptingReceiver, zeroAddress, false),
		},
		"safeTransferFrom with data to an accepting receiver succeeds": {
			sender: alice,
			action: erc721SafeTransferFromWithData(alice, acceptingReceiver),
			wantLogs: emits("Transfer", alice, acceptingReceiver, caseToken).
				then("ERC721Received", alice, alice, caseToken, []byte{0x01, 0x02}),
			want: erc721Owned(acceptingReceiver, zeroAddress, false),
		},
		"safeTransferFrom by an operator to an accepting receiver succeeds": {
			setup:  operatorBob,
			sender: bob,
			action: erc721SafeTransferFrom(alice, acceptingReceiver),
			wantLogs: emits("Transfer", alice, acceptingReceiver, caseToken).
				then("ERC721Received", bob, alice, caseToken, []byte{}),
			want: erc721Owned(acceptingReceiver, zeroAddress, true),
		},
		"safeTransferFrom without approval fails": {
			sender: bob,
			action: erc721SafeTransferFrom(alice, carol),
		},
		"safeTransferFrom to a contract without receiver hook fails": {
			sender: alice,
			action: erc721SafeTransferFrom(alice, tokenContract),
		},
		"safeTransferFrom with data to a contract without receiver hook fails": {
			sender: alice,
			action: erc721SafeTransferFromWithData(alice, tokenContract),
		},
		"safeTransferFrom to a rejecting receiver fails": {
			sender: alice,
			action: erc721SafeTransferFrom(alice, rejectingReceiver),
		},
		"safeTransferFrom with data to a rejecting receiver fails": {
			sender: alice,
			action: erc721SafeTransferFromWithData(alice, rejectingReceiver),
		},

		"approve by the owner succeeds": {
			sender:   alice,
			action:   erc721Approve(bob),
			wantLogs: emits("Approval", alice, bob, caseToken),
			want:     erc721Owned(alice, bob, false),
		},
		"approve by an operator succeeds": {
			setup:    operatorBob,
			sender:   bob,
			action:   erc721Approve(carol),
			wantLogs: emits("Approval", alice, carol, caseToken),
			want:     erc721Owned(alice, carol, true),
		},
		"approve replaces the previously approved address": {
			setup:    approvedBob,
			sender:   alice,
			action:   erc721Approve(carol),
			wantLogs: emits("Approval", alice, carol, caseToken),
			want:     erc721Owned(alice, carol, false),
		},
		"approve of the zero address clears the approval": {
			setup:    approvedBob,
			sender:   alice,
			action:   erc721Approve(zeroAddress),
			wantLogs: emits("Approval", alice, zeroAddress, caseToken),
			want:     erc721Owned(alice, zeroAddress, false),
		},
		"approve by a non-owner fails": {
			sender: bob,
			action: erc721Approve(carol),
		},
		"approve by the approved address fails": {
			setup:  approvedBob,
			sender: bob,
			action: erc721Approve(carol),
		},
		"approve by an operator of another owner fails": {
			setup:  []erc721Step{{carol, erc721SetApprovalForAll(bob, true)}},
			sender: bob,
			action: erc721Approve(carol),
		},
		"approve of an unminted token fails": {
			sender: alice,
			action: erc721OnUnmintedToken(erc721Approve(bob)),
		},

		"setApprovalForAll grants an operator": {
			sender:   alice,
			action:   erc721SetApprovalForAll(bob, true),
			wantLogs: emits("ApprovalForAll", alice, bob, true),
			want:     erc721Owned(alice, zeroAddress, true),
		},
		"setApprovalForAll revokes an operator": {
			setup:    operatorBob,
			sender:   alice,
			action:   erc721SetApprovalForAll(bob, false),
			wantLogs: emits("ApprovalForAll", alice, bob, false),
			want:     erc721Owned(alice, zeroAddress, false),
		},
	}

	t.Parallel()
	session := tests.StartIntegrationTestNet(t).SpawnSession(t)
	token, deployment, shared := deployToken(t, session, deployErc721(len(cases)))

	t.Run("contract is correctly deployed for defined cases", func(t *testing.T) {
		var want tokenLogs
		for id := 1; id <= len(cases); id++ {
			want = want.then("Transfer", zeroAddress, deployer, id)
		}
		require.Equal(t,
			want, decodeTokenLogs(t, erc721.Erc721MetaData, deployment, shared, nil),
			"creating the contract must emit a Transfer from the zero address to the deployer for each token")
	})

	names := slices.Sorted(maps.Keys(cases))
	parties := newTokenParties(t, session, shared, len(names))
	fundTokenParties(t, session, token, parties, erc721TransferFrom(deployer, alice))

	for i, name := range names {
		test, p := cases[name], parties[i]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, step := range test.setup {
				applyAs(t, session, p.accounts[step.sender], true, issue(token, p, step.action))
			}
			before := observeErc721(t, token, p.actors, p.tokenId)
			receipt := applyAs(t, session, p.accounts[test.sender], test.want != nil, issue(token, p, test.action))

			after := observeErc721(t, token, p.actors, p.tokenId)
			require.Equal(t,
				test.wantLogs,
				decodeTokenLogs(t, erc721.Erc721MetaData, receipt, p.actors, p.tokenId),
				"the events emitted by the transaction; a reverted transaction emits none")
			if test.want == nil {
				require.Equal(t,
					before, after,
					"a reverted transaction must not change the owner, the approved address or the operator")
			} else {
				require.Equal(t,
					*test.want, after,
					"the owner and approved address of the case's token, and whether Bob is Alice's operator after the transaction")
			}
		})
	}
}

type erc721Step = tokenStep[erc721.Erc721]

type erc721State struct {
	owner, approved tokenActor // of the case's token
	operator        bool       // whether Bob is an operator for Alice
}

// GoString makes failure messages show actors by name.
func (s erc721State) GoString() string {
	return fmt.Sprintf("{owner: %v, approved: %v, operator: %v}", s.owner, s.approved, s.operator)
}

func erc721Owned(owner, approved tokenActor, operator bool) *erc721State {
	return &erc721State{owner, approved, operator}
}

func observeErc721(t *testing.T, token *erc721.Erc721, a tokenActors, id *big.Int) erc721State {
	owner, err := token.OwnerOf(nil, id)
	require.NoError(t, err, "ownerOf the case's token")
	approved, err := token.GetApproved(nil, id)
	require.NoError(t, err, "getApproved of the case's token")
	operator, err := token.IsApprovedForAll(nil, a[alice], a[bob])
	require.NoError(t, err, "isApprovedForAll(Alice, Bob)")
	return erc721State{owner: a.of(t, owner), approved: a.of(t, approved), operator: operator}
}

func deployErc721(numTokens int) tests.ContractDeployer[erc721.Erc721] {
	return func(o *bind.TransactOpts, b bind.ContractBackend) (common.Address, *types.Transaction, *erc721.Erc721, error) {
		return erc721.DeployErc721(o, b, big.NewInt(int64(numTokens)))
	}
}

var erc721Unminted = new(big.Int).Lsh(big.NewInt(1), 128) // no case owns this id

// erc721OnUnmintedToken runs the action on a token that does not exist instead
// of the case's token.
func erc721OnUnmintedToken(action tokenAction[erc721.Erc721]) tokenAction[erc721.Erc721] {
	return func(o *bind.TransactOpts, c *erc721.Erc721, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return action(o, c, a, erc721Unminted)
	}
}

func erc721TransferFrom(from, to tokenActor) tokenAction[erc721.Erc721] {
	return func(o *bind.TransactOpts, c *erc721.Erc721, a tokenActors, id *big.Int) (*types.Transaction, error) {
		return c.TransferFrom(o, a[from], a[to], id)
	}
}

func erc721SafeTransferFrom(from, to tokenActor) tokenAction[erc721.Erc721] {
	return func(o *bind.TransactOpts, c *erc721.Erc721, a tokenActors, id *big.Int) (*types.Transaction, error) {
		return c.SafeTransferFrom(o, a[from], a[to], id)
	}
}

func erc721SafeTransferFromWithData(from, to tokenActor) tokenAction[erc721.Erc721] {
	return func(o *bind.TransactOpts, c *erc721.Erc721, a tokenActors, id *big.Int) (*types.Transaction, error) {
		return c.SafeTransferFrom0(o, a[from], a[to], id, []byte{0x01, 0x02})
	}
}

func erc721Approve(approved tokenActor) tokenAction[erc721.Erc721] {
	return func(o *bind.TransactOpts, c *erc721.Erc721, a tokenActors, id *big.Int) (*types.Transaction, error) {
		return c.Approve(o, a[approved], id)
	}
}

func erc721SetApprovalForAll(operator tokenActor, approved bool) tokenAction[erc721.Erc721] {
	return func(o *bind.TransactOpts, c *erc721.Erc721, a tokenActors, _ *big.Int) (*types.Transaction, error) {
		return c.SetApprovalForAll(o, a[operator], approved)
	}
}
