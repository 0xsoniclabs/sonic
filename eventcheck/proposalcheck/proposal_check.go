package proposalcheck

import (
	"fmt"

	"github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

//go:generate mockgen -source=proposal_check.go -destination=proposal_check_mock.go -package=proposalcheck

const (
	// MaxSizeOfProposedTransactions is the maximum size of the transactions
	// being allowed to being included in a single proposal.
	MaxSizeOfProposedTransactions = 8 * 1024 * 1024 // 8 MiB

	ErrProposalInInvalidEventVersion = common.ConstError("proposal in event with invalid version")

	ErrVersion3MustNotContainIndividualTransactions = common.ConstError("version 3 events must not contain individual transactions")
	ErrVersion3MustNotContainBlockVotes             = common.ConstError("version 3 events must not contain block votes")
	ErrVersion3MustNotContainEpochVotes             = common.ConstError("version 3 events must not contain epoch votes")
	ErrVersion3MustNotContainMisbehaviorProofs      = common.ConstError("version 3 events must not contain misbehavior proofs")
	ErrVersion3MustHaveANonNilPayload               = common.ConstError("version 3 events must have a non-nil payload")

	ErrSyncStateProgressionWithoutProposal     = common.ConstError("sync state progression without proposal in event payload")
	ErrProposalWithoutSyncStateProgression     = common.ConstError("proposal without sync state progression in event payload")
	ErrInvalidTurnProgression                  = common.ConstError("invalid turn progression in proposal sync state")
	ErrProposalMadeByProposerWithoutPermission = common.ConstError("proposal made by proposer without permission")

	ErrProposalContainsNilTransaction = common.ConstError("nil transaction in proposal")
	ErrTransactionsExceedSizeLimit    = common.ConstError("total size of transactions in proposal exceeds the allowed limit")
)

// Checker verifies that block proposal and proposal sync-state information in
// events is valid. This check must only be performed after all parent events
// are available, as the check requires to retrieve the parent's payload data.
type Checker struct {
	reader Reader
}

// Reader is an abstraction of a data source required by the proposal checker
// to validate an event.
type Reader interface {
	// GetEpochValidators returns the validators for the epoch in which the
	// event was created. This is used to check whether the proposer is allowed
	// to make a proposal in the event.
	GetEpochValidators() *pos.Validators

	// GetEventPayload returns the payload of the event with the given hash.
	// This is used to obtain the payload of the parent events of the event to
	// be checked.
	GetEventPayload(hash.Event) inter.Payload
}

// New creates a new Checker validating the content of an event's proposal
// envelope.
func New(reader Reader) *Checker {
	return &Checker{
		reader: reader,
	}
}

// Validate checks whether the event payload is correctly tracking the proposer
// state and the validity of a potentially included proposal.
func (v *Checker) Validate(e inter.EventPayloadI) error {
	// Only version 3 events are allowed to contain proposals.
	if e.Version() != 3 {
		if payload := e.Payload(); payload != nil {
			if proposal := payload.Proposal; proposal != nil {
				return ErrProposalInInvalidEventVersion
			}
		}
		// All remaining checks are only applicable to version 3 events.
		return nil
	}

	// Check version 3 properties.
	if err := checkVersion3EventProperties(e); err != nil {
		return err
	}

	// Check that the payload is not nil.
	payload := e.Payload()
	if payload == nil {
		return ErrVersion3MustHaveANonNilPayload
	}

	// Check valid progression of the proposal sync state.
	incoming := inter.CalculateIncomingProposalSyncState(v.reader, e)
	present := payload.ProposalSyncState
	if incoming == present {
		if payload.Proposal != nil {
			return ErrProposalWithoutSyncStateProgression
		}
		return nil
	}

	// Since the sync state progressed, there must be a proposal.
	proposal := payload.Proposal
	if proposal == nil {
		return ErrSyncStateProgressionWithoutProposal
	}

	// Check that the progression was valid.
	before := inter.ProposalSummary{
		Turn:  incoming.LastSeenProposalTurn,
		Frame: incoming.LastSeenProposalFrame,
	}
	after := inter.ProposalSummary{
		Turn:  present.LastSeenProposalTurn,
		Frame: present.LastSeenProposalFrame,
	}
	if !inter.IsValidTurnProgression(before, after) {
		return ErrInvalidTurnProgression
	}

	// Check that the proposer was allowed to make this proposal.
	allowed, _, err := inter.IsAllowedToPropose(
		e.Creator(),
		v.reader.GetEpochValidators(),
		incoming,
		e.Epoch(),
		e.Frame(),
	)
	if err != nil {
		return fmt.Errorf("failed to check whether proposer is allowed to propose: %w", err)
	}
	if !allowed {
		return ErrProposalMadeByProposerWithoutPermission
	}

	// --- check the content of the proposal ---
	return checkProposal(e, *proposal)
}

// checkVersion3EventProperties checks key properties of the event payload for
// version 3 events.
func checkVersion3EventProperties(e inter.EventPayloadI) error {
	// None of the version 1 or 2 payload fields must be present.
	if e.AnyTxs() {
		return ErrVersion3MustNotContainIndividualTransactions
	}
	if e.AnyBlockVotes() {
		return ErrVersion3MustNotContainBlockVotes
	}
	if e.AnyEpochVote() {
		return ErrVersion3MustNotContainEpochVotes
	}
	if e.AnyMisbehaviourProofs() {
		return ErrVersion3MustNotContainMisbehaviorProofs
	}
	if e.Payload() == nil {
		return ErrVersion3MustHaveANonNilPayload
	}
	return nil
}

// checkProposal checks the proposal in the event payload for validity.
func checkProposal(
	event inter.EventPayloadI,
	proposal inter.Proposal,
) error {
	// --- check the present transactions ---

	// Check that there are no nil-transactions in the proposal.
	for _, tx := range proposal.Transactions {
		if tx == nil {
			return ErrProposalContainsNilTransaction
		}
	}

	// Check that the total size of the transactions is in allowed limits.
	totalSize := uint64(0)
	for _, tx := range proposal.Transactions {
		totalSize += tx.Size()
	}
	if totalSize > MaxSizeOfProposedTransactions {
		return ErrTransactionsExceedSizeLimit
	}

	return nil
}
