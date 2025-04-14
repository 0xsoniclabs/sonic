package inter

import (
	"crypto/sha256"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/utils/byteutils"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

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

	Txs() types.Transactions
	EpochVote() LlrEpochVote
	BlockVotes() LlrBlockVotes
	MisbehaviourProofs() []MisbehaviourProof
}

var emptyPayloadHash1 = CalcPayloadHash(&MutableEventPayload{extEventData: extEventData{version: 1}})

func EmptyPayloadHash(version uint8) consensus.Hash {
	if version == 1 {
		return emptyPayloadHash1
	} else {
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

func (e *extEventData) GasPowerLeft() GasPowerLeft { return e.gasPowerLeft }

func (e *extEventData) GasPowerUsed() uint64 { return e.gasPowerUsed }

func (e *sigData) Sig() Signature { return e.sig }

func (e *payloadData) Txs() types.Transactions { return e.txs }

func (e *payloadData) MisbehaviourProofs() []MisbehaviourProof { return e.misbehaviourProofs }

func (e *payloadData) BlockVotes() LlrBlockVotes { return e.blockVotes }

func (e *payloadData) EpochVote() LlrEpochVote { return e.epochVote }

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
		return consensus.EventHashFromBytes(consensus.EventHashFromBytes(CalcTxHash(e.Txs()).Bytes(), CalcMisbehaviourProofsHash(e.MisbehaviourProofs()).Bytes()).Bytes(), consensus.EventHashFromBytes(e.EpochVote().Hash().Bytes(), e.BlockVotes().Hash().Bytes()).Bytes())
	} else {
		return CalcTxHash(e.Txs())
	}
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
