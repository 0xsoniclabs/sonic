package proposalcheck

import (
	"errors"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

// TODO: document and make const
var (
	ErrMissingProposalEnvelope = errors.New("event has no proposal envelope")
	ErrInvalidProposer         = errors.New("event contains proposal of invalid proposer")
	ErrInvalidProposalTime     = errors.New("event contains proposal with invalid time")

	ErrVersion3MustNotContainTransactions      = errors.New("event with version 3 must not contain transactions")
	ErrVersion3MustNotContainBlockVotes        = errors.New("event with version 3 must not contain block votes")
	ErrVersion3MustNotContainEpochVotes        = errors.New("event with version 3 must not contain epoch votes")
	ErrVersion3MustNotContainMisbehaviorProofs = errors.New("event with version 3 must not contain misbehavior proofs")
)

// Checker that block proposal and envelope information in events is valid.
type Checker struct {
	reader Reader
}

type Reader interface {
	GetEpochValidators() *pos.Validators
}

// New creates a new Checker validating the content of an event's proposal
// envelope.
func New(reader Reader) *Checker {
	return &Checker{
		reader: reader,
	}
}

// Validate event
func (v *Checker) Validate(e inter.EventPayloadI) error {

	// TODO: check the following properties
	// - last-seen fields are correctly filled
	// - a new proposal is not older than the last seen proposal
	// - the proposer has the right to propose
	// - if it is a re-proposal, sufficient time has passed
	// - the proposed block is valid, thus
	//   - the block number is correct
	//   - the attempt number is correct
	//   - the base-fee is correct
	//   - all transactions can be executed
	//   - the total gas used is within the limit of the accepted network throughput

	// This check only concerns events of protocol version 3.
	if e.Version() != 3 {
		return nil // all fine with other events
	}

	// None of the version 1 or 2 payload fields must be present.
	if e.AnyTxs() {
		return ErrVersion3MustNotContainTransactions
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

	// -- Envelope Checks --

	// Check that there is an envelope.
	envelope := e.ProposalEnvelope()
	if envelope == nil {
		return ErrMissingProposalEnvelope
	}

	// -- Proposal Checks --

	proposal := envelope.Proposal
	if proposal == nil {
		return nil
	}

	// Check that the creator of the event is allowed to make the present proposal.
	proposer, err := inter.GetProposer(
		v.reader.GetEpochValidators(),
		proposal.Number,
		proposal.Attempt,
	)
	if err != nil {
		return err
	}
	if proposer != e.Creator() {
		return ErrInvalidProposer
	}

	// Check that the proposed block time is equal to the median time.
	if proposal.Time != e.MedianTime() {
		return ErrInvalidProposalTime
	}

	// For these checks access to the preceding block is required:
	// - the right hash of the preceding block
	// - the use of the correct base-fee price
	// - the use of the correct randao value

	// For these checks access to the preceding block's state is required:
	// - all transactions are processable
	// - the total gas used is within the limit of the accepted network throughput

	// all fine
	return nil
}
