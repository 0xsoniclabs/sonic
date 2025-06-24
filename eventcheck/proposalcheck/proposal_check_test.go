// Copyright 2025 Sonic Operations Ltd
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

package proposalcheck

import (
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestProposalCheck_Validate_NonVersion3WithoutProposal_Passes(t *testing.T) {
	for version := range uint8(10) {
		if version == 3 {
			continue
		}
		ctrl := gomock.NewController(t)
		event := inter.NewMockEventPayloadI(ctrl)
		event.EXPECT().Version().Return(version)
		event.EXPECT().Payload().Return(&inter.Payload{})

		checker := New(nil)
		require.NoError(t, checker.Validate(event))
	}
}

func TestProposalCheck_Validate_NonVersion3WithProposal_Fails(t *testing.T) {
	for version := range uint8(10) {
		if version == 3 {
			continue
		}
		ctrl := gomock.NewController(t)
		event := inter.NewMockEventPayloadI(ctrl)
		event.EXPECT().Version().Return(version)
		event.EXPECT().Payload().Return(&inter.Payload{
			Proposal: &inter.Proposal{},
		})

		checker := New(nil)
		require.ErrorIs(t, checker.Validate(event), ErrProposalInInvalidEventVersion)
	}
}

func TestProposalCheck_Validate_ValidGenesisEventWithoutProposalPasses(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)
	event := NewMockEventPassingVersion3PropertyTests(ctrl)

	// The event to be tested is a genesis event - there are no parents.
	event.EXPECT().Parents().Return([]hash.Event{})
	event.EXPECT().Payload().Return(&inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  0,
			LastSeenProposalFrame: 0,
		},
	}).AnyTimes()

	checker := New(reader)
	require.NoError(t, checker.Validate(event))
}

func TestProposalCheck_Validate_ValidGenesisEventWithProposalPasses(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)
	event := NewMockEventPassingVersion3PropertyTests(ctrl)

	validator := idx.ValidatorID(1)
	validators := pos.EqualWeightValidators([]idx.ValidatorID{validator}, 1)
	reader.EXPECT().GetEpochValidators().Return(validators)

	event.EXPECT().Creator().Return(validator)
	event.EXPECT().Epoch().Return(idx.Epoch(4))
	event.EXPECT().Frame().Return(idx.Frame(1))
	event.EXPECT().Parents().Return([]hash.Event{})
	event.EXPECT().Payload().Return(&inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  1,
			LastSeenProposalFrame: 1,
		},
		Proposal: &inter.Proposal{},
	}).AnyTimes()

	checker := New(reader)
	require.NoError(t, checker.Validate(event))
}

func TestProposalCheck_Validate_ValidEventWithoutProposalPasses(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)
	event := NewMockEventPassingVersion3PropertyTests(ctrl)

	parent1 := hash.Event{1}
	parent2 := hash.Event{2}

	syncState1 := inter.ProposalSyncState{
		LastSeenProposalTurn:  12,
		LastSeenProposalFrame: 14,
	}
	syncState2 := inter.ProposalSyncState{
		LastSeenProposalTurn:  16,
		LastSeenProposalFrame: 10,
	}
	joinedState := inter.JoinProposalSyncStates(syncState1, syncState2)

	reader.EXPECT().GetEventPayload(parent1).Return(inter.Payload{
		ProposalSyncState: syncState1,
	})
	reader.EXPECT().GetEventPayload(parent2).Return(inter.Payload{
		ProposalSyncState: syncState2,
	})

	event.EXPECT().Parents().Return([]hash.Event{parent1, parent2})
	event.EXPECT().Payload().Return(&inter.Payload{
		ProposalSyncState: joinedState,
	}).AnyTimes()

	checker := New(reader)
	require.NoError(t, checker.Validate(event))
}

func TestProposalCheck_Validate_ValidEventWithProposalPasses(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)
	event := NewMockEventPassingVersion3PropertyTests(ctrl)

	validator := idx.ValidatorID(1)
	validators := pos.EqualWeightValidators([]idx.ValidatorID{validator}, 1)
	reader.EXPECT().GetEpochValidators().Return(validators)

	parent1 := hash.Event{1}
	parent2 := hash.Event{2}

	syncState1 := inter.ProposalSyncState{
		LastSeenProposalTurn:  12,
		LastSeenProposalFrame: 14,
	}
	syncState2 := inter.ProposalSyncState{
		LastSeenProposalTurn:  16,
		LastSeenProposalFrame: 10,
	}
	joinedState := inter.JoinProposalSyncStates(syncState1, syncState2)

	reader.EXPECT().GetEventPayload(parent1).Return(inter.Payload{
		ProposalSyncState: syncState1,
	})
	reader.EXPECT().GetEventPayload(parent2).Return(inter.Payload{
		ProposalSyncState: syncState2,
	})

	event.EXPECT().Creator().Return(validator)
	event.EXPECT().Epoch().Return(idx.Epoch(4))
	event.EXPECT().Frame().Return(idx.Frame(16))
	event.EXPECT().Parents().Return([]hash.Event{parent1, parent2})
	event.EXPECT().Payload().Return(&inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  joinedState.LastSeenProposalTurn + 1,
			LastSeenProposalFrame: 16,
		},
		Proposal: &inter.Proposal{},
	}).AnyTimes()

	checker := New(reader)
	require.NoError(t, checker.Validate(event))
}

func TestChecker_Validate_DetectsInvalidEvent(t *testing.T) {
	tests := map[string]struct {
		corrupt  func(event *inter.MockEventPayloadI)
		expected error
	}{
		"invalid version 3 content": {
			corrupt: func(event *inter.MockEventPayloadI) {
				// just one example of invalid content
				event.EXPECT().AnyTxs().Return(true)
			},
			expected: ErrVersion3MustNotContainIndividualTransactions,
		},
		"nil payload": {
			corrupt: func(event *inter.MockEventPayloadI) {
				event.EXPECT().Payload().Return(nil)
			},
			expected: ErrVersion3MustHaveANonNilPayload,
		},
		"sudden nil payload": {
			corrupt: func(event *inter.MockEventPayloadI) {
				// This is called by the version-3 checker.
				event.EXPECT().Payload().Return(&inter.Payload{})
				// This is called by Validate before running payload checks.
				event.EXPECT().Payload().Return(nil)
			},
			expected: ErrVersion3MustHaveANonNilPayload,
		},
		"proposal without sync state progression": {
			corrupt: func(event *inter.MockEventPayloadI) {
				event.EXPECT().Payload().Return(&inter.Payload{
					ProposalSyncState: inter.ProposalSyncState{
						LastSeenProposalTurn:  0, // no progression
						LastSeenProposalFrame: 0,
					},
					Proposal: &inter.Proposal{},
				}).AnyTimes()
			},
			expected: ErrProposalWithoutSyncStateProgression,
		},
		"invalid sync state progression": {
			corrupt: func(event *inter.MockEventPayloadI) {
				event.EXPECT().Payload().Return(&inter.Payload{
					ProposalSyncState: inter.ProposalSyncState{
						LastSeenProposalTurn:  1, // invalid turn, should be 0
						LastSeenProposalFrame: 0,
					},
					Proposal: &inter.Proposal{},
				}).AnyTimes()
			},
			expected: ErrInvalidTurnProgression,
		},
		"sync state progression without proposal": {
			corrupt: func(event *inter.MockEventPayloadI) {
				event.EXPECT().Payload().Return(&inter.Payload{
					ProposalSyncState: inter.ProposalSyncState{
						LastSeenProposalTurn:  1,
						LastSeenProposalFrame: 1,
					},
					Proposal: nil, // no proposal
				}).AnyTimes()
			},
			expected: ErrSyncStateProgressionWithoutProposal,
		},
		"proposal made by proposer without permission": {
			corrupt: func(event *inter.MockEventPayloadI) {
				event.EXPECT().Payload().Return(&inter.Payload{
					ProposalSyncState: inter.ProposalSyncState{
						LastSeenProposalTurn:  1,
						LastSeenProposalFrame: 1,
					},
					Proposal: &inter.Proposal{},
				}).AnyTimes()
				event.EXPECT().Creator().Return(idx.ValidatorID(12))
			},
			expected: ErrProposalMadeByProposerWithoutPermission,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			reader := NewMockReader(ctrl)
			event := inter.NewMockEventPayloadI(ctrl)

			creator := idx.ValidatorID(1)
			validators := pos.EqualWeightValidators([]idx.ValidatorID{creator}, 1)
			reader.EXPECT().GetEpochValidators().Return(validators).AnyTimes()

			test.corrupt(event)

			event.EXPECT().Version().Return(uint8(3)).AnyTimes()
			event.EXPECT().AnyTxs().AnyTimes()
			event.EXPECT().AnyBlockVotes().AnyTimes()
			event.EXPECT().AnyEpochVote().AnyTimes()
			event.EXPECT().AnyMisbehaviourProofs().AnyTimes()

			event.EXPECT().Creator().Return(creator).AnyTimes()
			event.EXPECT().Epoch().Return(idx.Epoch(0)).AnyTimes()
			event.EXPECT().Frame().Return(idx.Frame(1)).AnyTimes()
			event.EXPECT().Parents().Return([]hash.Event{}).AnyTimes()
			event.EXPECT().Payload().Return(&inter.Payload{
				ProposalSyncState: inter.ProposalSyncState{
					LastSeenProposalTurn:  0,
					LastSeenProposalFrame: 0,
				},
			}).AnyTimes()

			checker := New(reader)
			require.ErrorIs(t, checker.Validate(event), test.expected)
		})
	}
}

func TestProposalCheck_Validate_ReportsInvalidValidatorSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)
	event := NewMockEventPassingVersion3PropertyTests(ctrl)

	// An empty validator set is invalid.
	validator := idx.ValidatorID(1)
	validators := pos.EqualWeightValidators([]idx.ValidatorID{}, 1)
	reader.EXPECT().GetEpochValidators().Return(validators)

	event.EXPECT().Creator().Return(validator)
	event.EXPECT().Epoch().Return(idx.Epoch(4))
	event.EXPECT().Frame().Return(idx.Frame(1))
	event.EXPECT().Parents().Return([]hash.Event{})
	event.EXPECT().Payload().Return(&inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  1,
			LastSeenProposalFrame: 1,
		},
		Proposal: &inter.Proposal{},
	}).AnyTimes()

	checker := New(reader)
	require.ErrorContains(t, checker.Validate(event), "no validators")
}

func TestCheckVersion3EventProperties_AcceptsValidEvent(t *testing.T) {
	builder := inter.MutableEventPayload{}
	builder.SetVersion(3)
	event := builder.Build()
	require.NoError(t, checkVersion3EventProperties(event))
}

func TestCheckVersion3EventProperties_DetectsInvalidContent(t *testing.T) {

	tests := map[string]struct {
		taint    func(event *inter.MockEventPayloadI)
		expected error
	}{
		"with transactions": {
			taint: func(event *inter.MockEventPayloadI) {
				event.EXPECT().AnyTxs().Return(true)
			},
			expected: ErrVersion3MustNotContainIndividualTransactions,
		},
		"with block votes": {
			taint: func(event *inter.MockEventPayloadI) {
				event.EXPECT().AnyBlockVotes().Return(true)
			},
			expected: ErrVersion3MustNotContainBlockVotes,
		},
		"with epoch votes": {
			taint: func(event *inter.MockEventPayloadI) {
				event.EXPECT().AnyEpochVote().Return(true)
			},
			expected: ErrVersion3MustNotContainEpochVotes,
		},
		"with misbehavior proofs": {
			taint: func(event *inter.MockEventPayloadI) {
				event.EXPECT().AnyMisbehaviourProofs().Return(true)
			},
			expected: ErrVersion3MustNotContainMisbehaviorProofs,
		},
		"with nil payload": {
			taint: func(event *inter.MockEventPayloadI) {
				event.EXPECT().Payload().Return(nil)
			},
			expected: ErrVersion3MustHaveANonNilPayload,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			event := inter.NewMockEventPayloadI(ctrl)
			test.taint(event)
			event.EXPECT().AnyTxs().AnyTimes()
			event.EXPECT().AnyBlockVotes().AnyTimes()
			event.EXPECT().AnyEpochVote().AnyTimes()
			event.EXPECT().AnyMisbehaviourProofs().AnyTimes()
			got := checkVersion3EventProperties(event)
			require.ErrorIs(t, got, test.expected)
		})
	}
}

func TestCheckProposal_AcceptsValidProposal(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"empty transactions": {},
		"single transaction": {
			types.NewTx(&types.LegacyTx{Nonce: 1}),
		},
		"multiple transaction": {
			types.NewTx(&types.LegacyTx{Nonce: 1}),
			types.NewTx(&types.LegacyTx{Nonce: 2}),
			types.NewTx(&types.LegacyTx{Nonce: 3}),
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, checkProposal(inter.Proposal{
				Transactions: transactions,
			}))
		})
	}
}

func TestCheckProposal_DetectsInvalidProposals(t *testing.T) {
	tests := map[string]struct {
		corrupt  func(proposal *inter.Proposal)
		expected error
	}{
		"nil transaction": {
			corrupt: func(proposal *inter.Proposal) {
				proposal.Transactions = []*types.Transaction{
					nil, // nil transaction
				}
			},
			expected: ErrProposalContainsNilTransaction,
		},
		"transactions exceed size limit": {
			corrupt: func(proposal *inter.Proposal) {
				// add transactions to exceed the size limit
				big := types.NewTx(&types.LegacyTx{
					Nonce: 1,
					Data:  make([]byte, MaxSizeOfProposedTransactions/2),
				})
				proposal.Transactions = append(
					proposal.Transactions,
					[]*types.Transaction{big, big, big}...,
				)
			},
			expected: ErrTransactionsExceedSizeLimit,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			proposal := &inter.Proposal{}
			test.corrupt(proposal)
			require.ErrorIs(t, checkProposal(*proposal), test.expected)
		})
	}
}

func NewMockEventPassingVersion3PropertyTests(ctrl *gomock.Controller) *inter.MockEventPayloadI {
	event := inter.NewMockEventPayloadI(ctrl)
	event.EXPECT().Version().Return(uint8(3)).AnyTimes()
	event.EXPECT().AnyTxs().AnyTimes()
	event.EXPECT().AnyBlockVotes().AnyTimes()
	event.EXPECT().AnyEpochVote().AnyTimes()
	event.EXPECT().AnyMisbehaviourProofs().AnyTimes()
	return event
}
