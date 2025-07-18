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
	"errors"
	"io"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/utils/cser"
)

var (
	ErrSerMalformedEvent = errors.New("serialization of malformed event")
	ErrTooLowEpoch       = errors.New("serialization of events with epoch<256 and version=0 is unsupported")
	ErrUnknownVersion    = errors.New("unknown serialization version")
)

const MaxSerializationVersion = 3

const ProtocolMaxMsgSize = 10 * 1024 * 1024

func (e *Event) MarshalCSER(w *cser.Writer) error {
	// version
	if e.Version() > 0 {
		w.BitsW.Write(2, 0)
		w.U8(e.Version())
	} else {
		if e.Epoch() < 256 {
			return ErrTooLowEpoch
		}
	}
	// base fields
	if e.Version() > 0 {
		w.U16(e.NetForkID())
	}
	w.U32(uint32(e.Epoch()))
	w.U32(uint32(e.Lamport()))
	w.U32(uint32(e.Creator()))
	w.U32(uint32(e.Seq()))
	w.U32(uint32(e.Frame()))
	w.U64(uint64(e.creationTime))
	medianTimeDiff := int64(e.creationTime) - int64(e.medianTime)
	w.I64(medianTimeDiff)
	// gas power
	w.U64(e.gasPowerUsed)
	w.U64(e.gasPowerLeft.Gas[0])
	w.U64(e.gasPowerLeft.Gas[1])
	// parents
	w.U32(uint32(len(e.Parents())))
	for _, p := range e.Parents() {
		if e.Lamport() < p.Lamport() {
			return ErrSerMalformedEvent
		}
		// lamport difference
		w.U32(uint32(e.Lamport() - p.Lamport()))
		// without epoch and lamport
		w.FixedBytes(p.Bytes()[8:])
	}
	// prev epoch hash
	w.Bool(e.prevEpochHash != nil)
	if e.prevEpochHash != nil {
		w.FixedBytes(e.prevEpochHash.Bytes())
	}
	// tx hash
	w.Bool(e.AnyTxs())
	if e.Version() == 1 {
		w.Bool(e.AnyMisbehaviourProofs())
		w.Bool(e.AnyEpochVote())
		w.Bool(e.AnyBlockVotes())
	}
	if e.Version() == 3 {
		w.Bool(e.HasProposal())
	}
	if e.AnyTxs() || e.AnyMisbehaviourProofs() || e.AnyBlockVotes() || e.AnyEpochVote() || e.Version() == 3 {
		w.FixedBytes(e.PayloadHash().Bytes())
	}
	// extra
	w.SliceBytes(e.Extra())
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaller interface.
func (e *Event) MarshalBinary() ([]byte, error) {
	return cser.MarshalBinaryAdapter(e.MarshalCSER)
}

func eventUnmarshalCSER(r *cser.Reader, e *MutableEventPayload) (err error) {
	// version
	var version uint8
	if r.BitsR.View(2) == 0 {
		r.BitsR.Read(2)
		version = r.U8()
		if version == 0 {
			return cser.ErrNonCanonicalEncoding
		}
	}
	if version > MaxSerializationVersion {
		return ErrUnknownVersion
	}

	// base fields
	var netForkID uint16
	if version > 0 {
		netForkID = r.U16()
	}
	epoch := r.U32()
	lamport := r.U32()
	creator := r.U32()
	seq := r.U32()
	frame := r.U32()
	creationTime := r.U64()
	medianTimeDiff := r.I64()
	// gas power
	gasPowerUsed := r.U64()
	gasPowerLeft0 := r.U64()
	gasPowerLeft1 := r.U64()
	// parents
	parentsNum := r.U32()
	if parentsNum > ProtocolMaxMsgSize/24 {
		return cser.ErrTooLargeAlloc
	}
	parents := make(hash.Events, 0, parentsNum)
	for i := uint32(0); i < parentsNum; i++ {
		// lamport difference
		lamportDiff := r.U32()
		// hash
		h := [24]byte{}
		r.FixedBytes(h[:])
		eID := dag.MutableBaseEvent{}
		eID.SetEpoch(idx.Epoch(epoch))
		eID.SetLamport(idx.Lamport(lamport - lamportDiff))
		eID.SetID(h)
		parents.Add(eID.ID())
	}
	// prev epoch hash
	var prevEpochHash *hash.Hash
	prevEpochHashExists := r.Bool()
	if prevEpochHashExists {
		prevEpochHash_ := hash.Hash{}
		r.FixedBytes(prevEpochHash_[:])
		prevEpochHash = &prevEpochHash_
	}
	// tx hash
	anyTxs := r.Bool()
	anyMisbehaviourProofs := version == 1 && r.Bool()
	anyEpochVote := version == 1 && r.Bool()
	anyBlockVotes := version == 1 && r.Bool()
	hasProposal := version == 3 && r.Bool()
	payloadHash := EmptyPayloadHash(version)
	if anyTxs || anyMisbehaviourProofs || anyEpochVote || anyBlockVotes || version == 3 {
		r.FixedBytes(payloadHash[:])
		if version != 3 && payloadHash == EmptyPayloadHash(version) {
			return cser.ErrNonCanonicalEncoding
		}
	}
	// extra
	extra := r.SliceBytes(ProtocolMaxMsgSize)

	if version == 0 && epoch < 256 {
		return ErrTooLowEpoch
	}

	e.SetVersion(version)
	e.SetNetForkID(netForkID)
	e.SetEpoch(idx.Epoch(epoch))
	e.SetLamport(idx.Lamport(lamport))
	e.SetCreator(idx.ValidatorID(creator))
	e.SetSeq(idx.Event(seq))
	e.SetFrame(idx.Frame(frame))
	e.SetCreationTime(Timestamp(creationTime))
	e.SetMedianTime(Timestamp(int64(creationTime) - medianTimeDiff))
	e.SetGasPowerUsed(gasPowerUsed)
	e.SetGasPowerLeft(GasPowerLeft{[2]uint64{gasPowerLeft0, gasPowerLeft1}})
	e.SetParents(parents)
	e.SetPrevEpochHash(prevEpochHash)
	e.anyTxs = anyTxs
	e.anyBlockVotes = anyBlockVotes
	e.anyEpochVote = anyEpochVote
	e.anyMisbehaviourProofs = anyMisbehaviourProofs
	e.hasProposal = hasProposal
	e.SetPayloadHash(payloadHash)
	e.SetExtra(extra)
	return nil
}

func MarshalTxsCSER(txs types.Transactions, w *cser.Writer) error {
	// txs size
	w.U56(uint64(txs.Len()))
	// txs
	for _, tx := range txs {
		err := TransactionMarshalCSER(w, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bvs LlrBlockVotes) MarshalCSER(w *cser.Writer) error {
	w.U64(uint64(bvs.Start))
	w.U32(uint32(bvs.Epoch))
	w.U32(uint32(len(bvs.Votes)))
	for _, r := range bvs.Votes {
		w.FixedBytes(r[:])
	}
	return nil
}

func (bvs *LlrBlockVotes) UnmarshalCSER(r *cser.Reader) error {
	start := r.U64()
	epoch := r.U32()
	num := r.U32()
	if num > ProtocolMaxMsgSize/32 {
		return cser.ErrTooLargeAlloc
	}
	records := make([]hash.Hash, num)
	for i := range records {
		r.FixedBytes(records[i][:])
	}
	bvs.Start = idx.Block(start)
	bvs.Epoch = idx.Epoch(epoch)
	bvs.Votes = records
	return nil
}

func (ers LlrEpochVote) MarshalCSER(w *cser.Writer) error {
	w.U32(uint32(ers.Epoch))
	w.FixedBytes(ers.Vote[:])
	return nil
}

func (ers *LlrEpochVote) UnmarshalCSER(r *cser.Reader) error {
	epoch := r.U32()
	record := hash.Hash{}
	r.FixedBytes(record[:])
	ers.Epoch = idx.Epoch(epoch)
	ers.Vote = record
	return nil
}

func (e *EventPayload) MarshalCSER(w *cser.Writer) error {
	if e.AnyTxs() != (e.txs.Len() != 0) {
		return ErrSerMalformedEvent
	}
	if e.AnyMisbehaviourProofs() != (len(e.misbehaviourProofs) != 0) {
		return ErrSerMalformedEvent
	}
	if e.AnyEpochVote() != (e.epochVote.Epoch != 0) {
		return ErrSerMalformedEvent
	}
	if e.AnyBlockVotes() != (len(e.blockVotes.Votes) != 0) {
		return ErrSerMalformedEvent
	}
	if e.Version() == 3 {
		if e.AnyBlockVotes() || e.AnyEpochVote() || e.AnyMisbehaviourProofs() || e.AnyTxs() {
			return ErrSerMalformedEvent
		}
		if e.HasProposal() != (e.payload.Proposal != nil) {
			return ErrSerMalformedEvent
		}
	}
	err := e.Event.MarshalCSER(w)
	if err != nil {
		return err
	}
	w.FixedBytes(e.sig.Bytes())
	if e.AnyTxs() {
		if e.Version() == 0 {
			// Txs are serialized with CSER for legacy events
			err = MarshalTxsCSER(e.txs, w)
			if err != nil {
				return err
			}
		} else {
			b, err := rlp.EncodeToBytes(e.txs)
			if err != nil {
				return err
			}
			w.SliceBytes(b)
		}
	}
	if e.AnyMisbehaviourProofs() {
		b, err := rlp.EncodeToBytes(e.misbehaviourProofs)
		if err != nil {
			return err
		}
		w.SliceBytes(b)
	}
	if e.AnyEpochVote() {
		err = e.EpochVote().MarshalCSER(w)
		if err != nil {
			return err
		}
	}
	if e.AnyBlockVotes() {
		err = e.BlockVotes().MarshalCSER(w)
		if err != nil {
			return err
		}
	}
	if e.Version() == 3 {
		b, err := e.Payload().Serialize()
		if err != nil {
			return err
		}
		w.SliceBytes(b)
	}
	return nil
}

func (e *MutableEventPayload) UnmarshalCSER(r *cser.Reader) error {
	err := eventUnmarshalCSER(r, e)
	if err != nil {
		return err
	}
	r.FixedBytes(e.sig[:])
	// txs
	txs := make(types.Transactions, 0, 4)
	if e.AnyTxs() {
		if e.version == 0 {
			// txs size
			size := r.U56()
			if size == 0 {
				return cser.ErrNonCanonicalEncoding
			}
			if size > ProtocolMaxMsgSize/64 {
				return cser.ErrTooLargeAlloc
			}
			for i := uint64(0); i < size; i++ {
				tx, err := TransactionUnmarshalCSER(r)
				if err != nil {
					return err
				}
				txs = append(txs, tx)
			}
		} else {
			b := r.SliceBytes(ProtocolMaxMsgSize)
			err := rlp.DecodeBytes(b, &txs)
			if err != nil {
				return err
			}
		}
	}
	e.txs = txs
	// mps
	mps := make([]MisbehaviourProof, 0)
	if e.AnyMisbehaviourProofs() {
		b := r.SliceBytes(ProtocolMaxMsgSize)
		err := rlp.DecodeBytes(b, &mps)
		if err != nil {
			return err
		}
	}
	e.misbehaviourProofs = mps
	// ev
	ev := LlrEpochVote{}
	if e.AnyEpochVote() {
		err := ev.UnmarshalCSER(r)
		if err != nil {
			return err
		}
		if ev.Epoch == 0 {
			return cser.ErrNonCanonicalEncoding
		}
	}
	e.epochVote = ev
	// bvs
	bvs := LlrBlockVotes{Votes: make([]hash.Hash, 0, 2)}
	if e.AnyBlockVotes() {
		err := bvs.UnmarshalCSER(r)
		if err != nil {
			return err
		}
		if len(bvs.Votes) == 0 || bvs.Start == 0 || bvs.Epoch == 0 {
			return cser.ErrNonCanonicalEncoding
		}
	}
	e.blockVotes = bvs
	// generic payload
	if e.Version() == 3 {
		b := r.SliceBytes(ProtocolMaxMsgSize)
		if err := e.payload.Deserialize(b); err != nil {
			return err
		}
	}
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaller interface.
func (e *EventPayload) MarshalBinary() ([]byte, error) {
	return cser.MarshalBinaryAdapter(e.MarshalCSER)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaller interface.
func (e *MutableEventPayload) UnmarshalBinary(raw []byte) (err error) {
	return cser.UnmarshalBinaryAdapter(raw, e.UnmarshalCSER)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaller interface.
func (e *EventPayload) UnmarshalBinary(raw []byte) (err error) {
	mutE := MutableEventPayload{}
	err = mutE.UnmarshalBinary(raw)
	if err != nil {
		return err
	}
	eventSer, _ := mutE.immutable().Event.MarshalBinary()
	locatorHash, baseHash := calcEventHashes(eventSer, &mutE)
	*e = *mutE.build(locatorHash, baseHash, len(raw))
	return nil
}

// EncodeRLP implements rlp.Encoder interface.
func (e *EventPayload) EncodeRLP(w io.Writer) error {
	bytes, err := e.MarshalBinary()
	if err != nil {
		return err
	}

	err = rlp.Encode(w, &bytes)

	return err
}

// DecodeRLP implements rlp.Decoder interface.
func (e *EventPayload) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	return e.UnmarshalBinary(bytes)
}

// DecodeRLP implements rlp.Decoder interface.
func (e *MutableEventPayload) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	return e.UnmarshalBinary(bytes)
}

// RPCMarshalEvent converts the given event to the RPC output .
func RPCMarshalEvent(e EventI) map[string]interface{} {
	return map[string]interface{}{
		"version":        hexutil.Uint64(e.Version()),
		"networkVersion": hexutil.Uint64(e.NetForkID()),
		"epoch":          hexutil.Uint64(e.Epoch()),
		"seq":            hexutil.Uint64(e.Seq()),
		"id":             hexutil.Bytes(e.ID().Bytes()),
		"frame":          hexutil.Uint64(e.Frame()),
		"creator":        hexutil.Uint64(e.Creator()),
		"prevEpochHash":  e.PrevEpochHash(),
		"parents":        EventIDsToHex(e.Parents()),
		"lamport":        hexutil.Uint64(e.Lamport()),
		"creationTime":   hexutil.Uint64(e.CreationTime()),
		"medianTime":     hexutil.Uint64(e.MedianTime()),
		"extraData":      hexutil.Bytes(e.Extra()),
		"payloadHash":    hexutil.Bytes(e.PayloadHash().Bytes()),
		"gasPowerLeft": map[string]interface{}{
			"shortTerm": hexutil.Uint64(e.GasPowerLeft().Gas[ShortTermGas]),
			"longTerm":  hexutil.Uint64(e.GasPowerLeft().Gas[LongTermGas]),
		},
		"gasPowerUsed": hexutil.Uint64(e.GasPowerUsed()),
		"anyTxs":       e.AnyTxs(),
	}
}

// RPCMarshalEventPayload converts the given event to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalEventPayload(event EventPayloadI, inclTx bool) (map[string]interface{}, error) {
	fields := RPCMarshalEvent(event)
	fields["size"] = hexutil.Uint64(event.Size())

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}
		txs := event.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}

		fields["transactions"] = transactions
	}

	return fields, nil
}

func EventIDsToHex(ids hash.Events) []hexutil.Bytes {
	res := make([]hexutil.Bytes, len(ids))
	for i, id := range ids {
		res[i] = id.Bytes()
	}
	return res
}
