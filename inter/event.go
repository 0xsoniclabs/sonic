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

package inter

import (
	"crypto/sha256"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/utils/byteutils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

//go:generate mockgen -source=event.go -destination=event_mock.go -package=inter

type EventI interface {
	consensus.Event
	Version() uint8
	NetForkID() uint16
	CreationTime() Timestamp
	MedianTime() Timestamp
	PrevEpochHash() *consensus.Hash
	Extra() []byte
	GasPowerLeft() GasPowerLeft
	GasPowerUsed() uint64

	HashToSign() consensus.Hash
	Locator() EventLocator

	// Payload-related fields

	AnyTxs() bool
	AnyBlockVotes() bool
	AnyEpochVote() bool
	AnyMisbehaviourProofs() bool
	HasProposal() bool
	PayloadHash() consensus.Hash
}

type EventLocator struct {
	BaseHash    consensus.Hash
	NetForkID   uint16
	Epoch       consensus.Epoch
	Seq         consensus.Seq
	Lamport     consensus.Lamport
	Creator     consensus.ValidatorID
	PayloadHash consensus.Hash
}

type SignedEventLocator struct {
	Locator EventLocator
	Sig     Signature
}

type EventPayloadI interface {
	EventI
	Sig() Signature

	// Transactions list the transactions included in this event. These may be
	// transactions included directly in the payload (version 2 events) or
	// transactions included in a block proposal (version 3 events).
	Transactions() types.Transactions

	// TransactionsToMeter returns the transactions that should be used for
	// metering purposes. These only include transactions that are directly
	// included in the event payload (version 2 events). Transactions included
	// in a block proposal (version 3 events) are not charged against a
	// validator's gas power. Their emission rate is controlled through turns.
	TransactionsToMeter() types.Transactions

	//Txs() types.Transactions
	EpochVote() LlrEpochVote
	BlockVotes() LlrBlockVotes
	MisbehaviourProofs() []MisbehaviourProof
	Payload() *Payload
}

var emptyPayloadHash1 = CalcPayloadHash(&MutableEventPayload{extEventData: extEventData{version: 1}})
var emptyPayloadHash3 = CalcPayloadHash(&MutableEventPayload{extEventData: extEventData{version: 3}})

func EmptyPayloadHash(version uint8) consensus.Hash {
	switch version {
	case 1:
		return emptyPayloadHash1
	case 3:
		return emptyPayloadHash3
	default:
		return consensus.Hash(types.EmptyRootHash)
	}
}

type baseEvent struct {
	consensus.BaseEvent
}

type mutableBaseEvent struct {
	consensus.MutableBaseEvent
}

type extEventData struct {
	version       uint8
	netForkID     uint16
	creationTime  Timestamp
	medianTime    Timestamp
	prevEpochHash *consensus.Hash
	gasPowerLeft  GasPowerLeft
	gasPowerUsed  uint64
	extra         []byte

	anyTxs                bool
	anyBlockVotes         bool
	anyEpochVote          bool
	anyMisbehaviourProofs bool
	hasProposal           bool
	payloadHash           consensus.Hash
}

type sigData struct {
	sig Signature
}

type payloadData struct {
	txs                types.Transactions
	misbehaviourProofs []MisbehaviourProof

	epochVote  LlrEpochVote
	blockVotes LlrBlockVotes

	payload Payload
}

type Event struct {
	baseEvent
	extEventData

	// cache
	_baseHash    *consensus.Hash
	_locatorHash *consensus.Hash
}

type SignedEvent struct {
	Event
	sigData
}

type EventPayload struct {
	SignedEvent
	payloadData

	// cache
	_size int
}

type MutableEventPayload struct {
	mutableBaseEvent
	extEventData
	sigData
	payloadData
}

func (e *Event) HashToSign() consensus.Hash {
	return *e._locatorHash
}

func asLocator(basehash consensus.Hash, e EventI) EventLocator {
	return EventLocator{
		BaseHash:    basehash,
		NetForkID:   e.NetForkID(),
		Epoch:       e.Epoch(),
		Seq:         e.Seq(),
		Lamport:     e.Lamport(),
		Creator:     e.Creator(),
		PayloadHash: e.PayloadHash(),
	}
}

func (e *Event) Locator() EventLocator {
	return asLocator(*e._baseHash, e)
}

func (e *EventPayload) Size() int {
	return e._size
}

func (e *extEventData) Version() uint8 { return e.version }

func (e *extEventData) NetForkID() uint16 { return e.netForkID }

func (e *extEventData) CreationTime() Timestamp { return e.creationTime }

func (e *extEventData) CreationTimePortable() uint64 { return uint64(e.creationTime) }

func (e *extEventData) MedianTime() Timestamp { return e.medianTime }

func (e *extEventData) PrevEpochHash() *consensus.Hash { return e.prevEpochHash }

func (e *extEventData) Extra() []byte { return e.extra }

func (e *extEventData) PayloadHash() consensus.Hash { return e.payloadHash }

func (e *extEventData) AnyTxs() bool { return e.anyTxs }

func (e *extEventData) AnyMisbehaviourProofs() bool { return e.anyMisbehaviourProofs }

func (e *extEventData) AnyEpochVote() bool { return e.anyEpochVote }

func (e *extEventData) AnyBlockVotes() bool { return e.anyBlockVotes }

func (e *extEventData) HasProposal() bool { return e.hasProposal }

func (e *extEventData) GasPowerLeft() GasPowerLeft { return e.gasPowerLeft }

func (e *extEventData) GasPowerUsed() uint64 { return e.gasPowerUsed }

func (e *sigData) Sig() Signature { return e.sig }

func (e *payloadData) Transactions() types.Transactions {
	if proposal := e.payload.Proposal; proposal != nil {
		return proposal.Transactions
	}
	return e.txs
}

func (e *payloadData) TransactionsToMeter() types.Transactions {
	return e.txs
}

func (e *payloadData) MisbehaviourProofs() []MisbehaviourProof { return e.misbehaviourProofs }

func (e *payloadData) BlockVotes() LlrBlockVotes { return e.blockVotes }

func (e *payloadData) EpochVote() LlrEpochVote { return e.epochVote }

func (e *payloadData) Payload() *Payload {
	return &e.payload
}

func CalcTxHash(txs types.Transactions) consensus.Hash {
	return consensus.Hash(types.DeriveSha(txs, trie.NewStackTrie(nil)))
}

func CalcMisbehaviourProofsHash(mps []MisbehaviourProof) consensus.Hash {
	hasher := sha256.New()
	_ = rlp.Encode(hasher, mps)
	return consensus.BytesToHash(hasher.Sum(nil))
}

func CalcPayloadHash(e EventPayloadI) consensus.Hash {
	if e.Version() == 1 {
		return consensus.EventHashFromBytes(consensus.EventHashFromBytes(CalcTxHash(e.Transactions()).Bytes(), CalcMisbehaviourProofsHash(e.MisbehaviourProofs()).Bytes()).Bytes(), consensus.EventHashFromBytes(e.EpochVote().Hash().Bytes(), e.BlockVotes().Hash().Bytes()).Bytes())
	}
	if e.Version() == 3 {
		return e.Payload().Hash()
	}
	return CalcTxHash(e.Transactions())
}

func (e *MutableEventPayload) SetVersion(v uint8) { e.version = v }

func (e *MutableEventPayload) SetNetForkID(v uint16) { e.netForkID = v }

func (e *MutableEventPayload) SetCreationTime(v Timestamp) { e.creationTime = v }

func (e *MutableEventPayload) SetMedianTime(v Timestamp) { e.medianTime = v }

func (e *MutableEventPayload) SetPrevEpochHash(v *consensus.Hash) { e.prevEpochHash = v }

func (e *MutableEventPayload) SetExtra(v []byte) { e.extra = v }

func (e *MutableEventPayload) SetPayloadHash(v consensus.Hash) { e.payloadHash = v }

func (e *MutableEventPayload) SetGasPowerLeft(v GasPowerLeft) { e.gasPowerLeft = v }

func (e *MutableEventPayload) SetGasPowerUsed(v uint64) { e.gasPowerUsed = v }

func (e *MutableEventPayload) SetSig(v Signature) { e.sig = v }

func (e *MutableEventPayload) SetTxs(v types.Transactions) {
	e.txs = v
	e.anyTxs = len(v) != 0
}

func (e *MutableEventPayload) SetMisbehaviourProofs(v []MisbehaviourProof) {
	e.misbehaviourProofs = v
	e.anyMisbehaviourProofs = len(v) != 0
}

func (e *MutableEventPayload) SetBlockVotes(v LlrBlockVotes) {
	e.blockVotes = v
	e.anyBlockVotes = len(v.Votes) != 0
}

func (e *MutableEventPayload) SetEpochVote(v LlrEpochVote) {
	e.epochVote = v
	e.anyEpochVote = v.Epoch != 0 && v.Vote != consensus.Zero
}

func (e *MutableEventPayload) SetPayload(payload Payload) {
	e.payload = payload
	e.hasProposal = payload.Proposal != nil
	e.payloadHash = payload.Hash()
}

func calcEventID(h consensus.Hash) (id [24]byte) {
	copy(id[:], h[:24])
	return id
}

func calcEventHashes(ser []byte, e EventI) (locator consensus.Hash, base consensus.Hash) {
	base = consensus.EventHashFromBytes(ser)
	if e.Version() < 1 {
		return base, base
	}
	return asLocator(base, e).HashToSign(), base
}

func (e *MutableEventPayload) calcHashes() (locator consensus.Hash, base consensus.Hash) {
	b, _ := e.immutable().Event.MarshalBinary()
	return calcEventHashes(b, e)
}

func (e *MutableEventPayload) size() int {
	b, err := e.immutable().MarshalBinary()
	if err != nil {
		panic("can't encode: " + err.Error())
	}
	return len(b)
}

func (e *MutableEventPayload) HashToSign() consensus.Hash {
	h, _ := e.calcHashes()
	return h
}

func (e *MutableEventPayload) Locator() EventLocator {
	_, baseHash := e.calcHashes()
	return asLocator(baseHash, e)
}

func (e *MutableEventPayload) Size() int {
	return e.size()
}

func (e *MutableEventPayload) build(locatorHash consensus.Hash, baseHash consensus.Hash, size int) *EventPayload {
	return &EventPayload{
		SignedEvent: SignedEvent{
			Event: Event{
				baseEvent:    baseEvent{*e.MutableBaseEvent.Build(calcEventID(locatorHash))},
				extEventData: e.extEventData,
				_baseHash:    &baseHash,
				_locatorHash: &locatorHash,
			},
			sigData: e.sigData,
		},
		payloadData: e.payloadData,
		_size:       size,
	}
}

func (e *MutableEventPayload) immutable() *EventPayload {
	return e.build(consensus.Hash{}, consensus.Hash{}, 0)
}

func (e *MutableEventPayload) Build() *EventPayload {
	locatorHash, baseHash := e.calcHashes()
	payloadSer, _ := e.immutable().MarshalBinary()
	return e.build(locatorHash, baseHash, len(payloadSer))
}

func (l EventLocator) HashToSign() consensus.Hash {
	return consensus.EventHashFromBytes(l.BaseHash.Bytes(), byteutils.Uint16ToBigEndian(l.NetForkID), l.Epoch.Bytes(), l.Seq.Bytes(), l.Lamport.Bytes(), l.Creator.Bytes(), l.PayloadHash.Bytes())
}

func (l EventLocator) ID() consensus.EventHash {
	h := l.HashToSign()
	copy(h[0:4], l.Epoch.Bytes())
	copy(h[4:8], l.Lamport.Bytes())
	return consensus.EventHash(h)
}
