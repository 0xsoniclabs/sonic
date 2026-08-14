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

// Package core is the harness the transaction property tests are built on: what a transaction is and
// what it can carry, how one is assembled and signed, the model predicting what the network will do
// with it, the invariants every block and account must satisfy, the pool of funded accounts, the
// barrier confirming absence, and the runner tying them together.
//
// It knows nothing about any particular kind of transaction. What one family of transactions is, how
// it is priced, what it needs arranged on chain and what only it can check, is a Domain -- see the
// regular and subsidies packages, and ../README.md for the design.
package core

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// TxSpec is one transaction to inject, described as plain data so that rapid can shrink it and a
// failing case can be printed verbatim. Every method comes from an embedded capability, so a
// transaction type declares what it can carry by embedding that capability and nothing else, and
// what it cannot carry is unrepresentable rather than merely unused.
type TxSpec interface {
	TxType() uint8

	// from Envelope
	Sender() int
	NonceChoice() NonceChoice
	GapSize() uint64
	Gas() uint64
	SigningMode() SigningChoice

	// from Payload
	Data() []byte

	// from a recipient capability
	Recipient() ToChoice
	IsCreate() bool

	// from a value capability
	Amount() *big.Int

	// from a price capability
	FeeCap() *big.Int
	TipCap() *big.Int
	SeparateTipCap() bool

	// txData assembles the payload of a transaction of this type, unsigned.
	TxData(nonce uint64, ctx BuildContext) (types.TxData, error)
}

// The capabilities a transaction type can carry. Each owns the fields it contributes and defines
// what they mean, which is what keeps the builder and the model from disagreeing about what a
// transaction really holds.

// Envelope is carried by every transaction: who sends it, where it sits in its sender's sequence,
// how much gas it asks for and how it is signed.
type Envelope struct {
	SenderIdx    int
	Nonce        NonceChoice
	NonceGapSize uint64 // size of the hole, used only when Nonce is NonceGap
	GasLimit     uint64
	Signing      SigningChoice
}

// SetSigning overrides the drawn signing choice, for a transaction whose signature is not this
// harness's to choose: a bundle signs its own contents, so a defect drawn for one of them never
// reaches the transaction that is built. A spec has to say what its transaction really is.
func (e *Envelope) SetSigning(choice SigningChoice) { e.Signing = choice }

func (e Envelope) Sender() int                { return e.SenderIdx }
func (e Envelope) NonceChoice() NonceChoice   { return e.Nonce }
func (e Envelope) GapSize() uint64            { return e.NonceGapSize }
func (e Envelope) Gas() uint64                { return e.GasLimit }
func (e Envelope) SigningMode() SigningChoice { return e.Signing }

// Payload is the call data. Its content matters only in that zero and non-zero bytes are priced
// differently, so it is drawn as a length and a choice between the two.
type Payload struct {
	DataLen     int
	DataNonZero bool
}

func (p Payload) Data() []byte {
	if p.DataLen <= 0 {
		return nil
	}
	data := make([]byte, p.DataLen)
	if p.DataNonZero {
		for i := range data {
			data[i] = 0x01
		}
	}
	return data
}

// OptionalRecipient holds its recipient as a pointer, so leaving it empty makes the transaction a
// contract creation.
type OptionalRecipient struct {
	To ToChoice
}

func (r OptionalRecipient) Recipient() ToChoice { return r.To }
func (r OptionalRecipient) IsCreate() bool {
	return r.To == ToCreate
}

// RequiredRecipient holds its recipient by value, so a contract creation is unrepresentable.
type RequiredRecipient struct {
	To ToChoice
}

func (r RequiredRecipient) Recipient() ToChoice { return r.To }
func (r RequiredRecipient) IsCreate() bool      { return false }

// WideValue is a value field held as a big.Int, which can carry more than 256 bits.
type WideValue struct {
	Value *big.Int
}

func (v WideValue) Amount() *big.Int { return v.Value }

// NarrowValue is a value field held as a uint256, which saturates anything wider.
type NarrowValue struct {
	Value *big.Int
}

func (v NarrowValue) Amount() *big.Int { return ToUint256(v.Value).ToBig() }

// SinglePrice is one gas price field standing for both caps, so a tip above the fee cap is
// unrepresentable rather than merely unlikely.
type SinglePrice struct {
	GasPrice *big.Int
}

func (p SinglePrice) FeeCap() *big.Int     { return p.GasPrice }
func (p SinglePrice) TipCap() *big.Int     { return p.GasPrice }
func (p SinglePrice) SeparateTipCap() bool { return false }
func (p *SinglePrice) ZeroPrices()         { p.GasPrice = big.NewInt(0) }

// WidePrices are separate fee and tip caps held as big.Int.
type WidePrices struct {
	GasFeeCap *big.Int
	GasTipCap *big.Int
}

func (p WidePrices) FeeCap() *big.Int     { return p.GasFeeCap }
func (p WidePrices) TipCap() *big.Int     { return p.GasTipCap }
func (p WidePrices) SeparateTipCap() bool { return true }
func (p *WidePrices) ZeroPrices()         { p.GasFeeCap, p.GasTipCap = big.NewInt(0), big.NewInt(0) }

// NarrowPrices are separate fee and tip caps held as uint256, which saturate anything wider.
type NarrowPrices struct {
	GasFeeCap *big.Int
	GasTipCap *big.Int
}

func (p NarrowPrices) FeeCap() *big.Int     { return ToUint256(p.GasFeeCap).ToBig() }
func (p NarrowPrices) TipCap() *big.Int     { return ToUint256(p.GasTipCap).ToBig() }
func (p NarrowPrices) SeparateTipCap() bool { return true }
func (p *NarrowPrices) ZeroPrices()         { p.GasFeeCap, p.GasTipCap = big.NewInt(0), big.NewInt(0) }

// AccessListEntries is an access list of one storage key per entry.
type AccessListEntries struct {
	AccessListLen int
}

func (a AccessListEntries) AccessList() types.AccessList {
	list := make(types.AccessList, max(a.AccessListLen, 0))
	for i := range list {
		list[i] = types.AccessTuple{
			Address:     common.Address{byte(i + 1)},
			StorageKeys: []common.Hash{{byte(i + 1)}},
		}
	}
	return list
}

// BlobHashVersion is the leading byte a blob hash needs to be structurally valid, per
// kzg4844.IsValidVersionedHash.
const BlobHashVersion = 0x01

// BlobHashList is a list of versioned blob hashes. Sonic accepts blob transactions only without any,
// so a non-empty list is malformed.
type BlobHashList struct {
	BlobHashLen int
}

func (b BlobHashList) BlobHashes() []common.Hash {
	hashes := make([]common.Hash, max(b.BlobHashLen, 0))
	for i := range hashes {
		hashes[i] = common.Hash{BlobHashVersion, byte(i + 1)}
	}
	return hashes
}

// AuthList is a set-code transaction's authorization list, which is malformed empty.
type AuthList struct {
	Entries []AuthChoice
}

func (a AuthList) Auths() []AuthChoice { return a.Entries }

// SignedElsewhere is a transaction whose signature is decided by something other than its drawn
// signing choice. Every transaction type embeds the envelope capability, so every spec can be asked.
type SignedElsewhere interface {
	SetSigning(SigningChoice)
}

// ZeroablePrice is a price capability whose fields can be set to zero, which every one of them is:
// a price is a number, and zero is a number. Every transaction type embeds exactly one, so a spec
// can always be asked for it.
//
// The fields are replaced rather than modified, because a drawn *big.Int is shared between
// iterations -- see the note on the shared generators in generators.go.
type ZeroablePrice interface {
	ZeroPrices()
}

// The capabilities a transaction type may lack. A spec is asked for one through these, so that a
// type simply does not embed what it cannot carry.
type (
	WithAccessList interface{ AccessList() types.AccessList }
	WithBlobHashes interface{ BlobHashes() []common.Hash }
	WithAuths      interface{ Auths() []AuthChoice }
)

func AccessListOf(spec TxSpec) types.AccessList {
	if with, ok := spec.(WithAccessList); ok {
		return with.AccessList()
	}
	return nil
}

func BlobHashesOf(spec TxSpec) []common.Hash {
	if with, ok := spec.(WithBlobHashes); ok {
		return with.BlobHashes()
	}
	return nil
}

func AuthsOf(spec TxSpec) []AuthChoice {
	if with, ok := spec.(WithAuths); ok {
		return with.Auths()
	}
	return nil
}

// Format renders a spec compactly enough to paste into a bug report.
func Format(spec TxSpec) string {
	var out strings.Builder
	fmt.Fprintf(&out,
		"type=%d sender=%d nonce=%s(+%d) gas=%d feeCap=%s tipCap=%s value=%s data=%dB to=%s signing=%s",
		spec.TxType(), spec.Sender(), spec.NonceChoice(), spec.GapSize(), spec.Gas(),
		spec.FeeCap(), spec.TipCap(), spec.Amount(), len(spec.Data()), spec.Recipient(), spec.SigningMode(),
	)
	if list := AccessListOf(spec); len(list) > 0 {
		fmt.Fprintf(&out, " accessList=%d", len(list))
	}
	if hashes := BlobHashesOf(spec); len(hashes) > 0 {
		fmt.Fprintf(&out, " blobHashes=%d", len(hashes))
	}
	if auths := AuthsOf(spec); auths != nil {
		fmt.Fprintf(&out, " auths=%v", auths)
	}
	// A transaction type that carries more than the capabilities above says so itself, so that a
	// failure names what it really was.
	if described, ok := spec.(fmt.Stringer); ok {
		fmt.Fprintf(&out, " %s", described)
	}
	return out.String()
}

// ToUint256 converts saturating at the maximum, because a value wider than 256 bits is worth
// generating but a uint256-backed transaction type cannot carry one.
func ToUint256(value *big.Int) *uint256.Int {
	converted, overflow := uint256.FromBig(value)
	if overflow {
		return new(uint256.Int).SetAllOne()
	}
	return converted
}

// Outcome is the observable fate of an injected transaction. Only three results can be told apart,
// because the block formation filter and execution both drop a transaction without a trace.
//
//go:generate go tool stringer -type=Outcome -trimprefix=Outcome
type Outcome int

const (
	// OutcomeEventRejected means the injection RPC refused the batch, so no transaction in it may
	// have had any effect.
	OutcomeEventRejected Outcome = iota
	// OutcomeDropped means the transaction produced no receipt, so it must appear in no block and
	// leave its sender untouched.
	OutcomeDropped
	// OutcomeExecuted means a receipt was produced, successful or failed.
	OutcomeExecuted
)

// NonceChoice selects how a transaction's nonce relates to its sender's own.
//
//go:generate go tool stringer -type=NonceChoice -trimprefix=Nonce
type NonceChoice uint8

const (
	// NonceCorrect is the sender's next expected nonce.
	NonceCorrect NonceChoice = iota
	// NonceTooLow is a nonce the sender has already spent.
	NonceTooLow
	// NonceGap leaves a hole above the expected nonce, which consensus cannot queue.
	NonceGap
	// NonceMax is math.MaxUint64, which ValidateTxStatic rejects outright.
	NonceMax
)

// ToChoice selects a transaction's recipient.
//
//go:generate go tool stringer -type=ToChoice -trimprefix=To
type ToChoice uint8

const (
	// ToSelf sends to the sender's own address.
	ToSelf ToChoice = iota
	// ToOther sends to a bystander address, never an account under observation.
	ToOther
	// ToCreate leaves the recipient empty, making this a contract creation.
	ToCreate
	// ToPrecompile targets a precompile address.
	ToPrecompile
)

// SigningChoice selects how a transaction is signed. Everything but SignCorrect is a deliberate
// defect, of one of two kinds: a signature that cannot be recovered at all, or one that recovers to
// an address holding nothing.
//
//go:generate go tool stringer -type=SigningChoice -trimprefix=Sign
type SigningChoice uint8

const (
	// SignCorrect signs with the sender's key for the network's chain ID.
	SignCorrect SigningChoice = iota
	// SignWrongChainID signs for a different chain, so recovery fails.
	SignWrongChainID
	// SignUnfundedKey signs with a throwaway key that holds no balance.
	SignUnfundedKey
	// SignZeroR zeroes the signature's R, so recovery fails.
	SignZeroR
	// SignZeroS zeroes the signature's S, so recovery fails.
	SignZeroS
	// SignForeign transplants a signature made over an unrelated payload.
	SignForeign
)

// RecoversToStranger reports which of the two kinds of signing defect this is: one that recovers to
// an address holding nothing, rather than one that cannot be recovered at all. The difference is
// invisible while the sender pays for its own gas -- either way the transaction is dropped -- and
// decisive once somebody else pays, because a stranger with a sponsor behind it can execute.
func (c SigningChoice) RecoversToStranger() bool {
	return c == SignUnfundedKey || c == SignForeign
}

// Recovers reports whether a sender can be derived from the signature at all, whoever it turns out to
// be. It decides more than whether the transaction can pay: the scrambler that orders a block builds
// its entries from the recovered sender and silently leaves out whatever it cannot recover
// (gossip/scrambler/tx_scrambler.go:66), so such a transaction never reaches the block processor --
// which matters for a transaction that would otherwise have had an effect without being charged.
func (c SigningChoice) Recovers() bool {
	return c == SignCorrect || c.RecoversToStranger()
}

// AuthChoice selects an entry of a set-code transaction's authorization list.
//
//go:generate go tool stringer -type=AuthChoice -trimprefix=Auth
type AuthChoice uint8

const (
	// AuthSelf delegates the sender's own account.
	AuthSelf AuthChoice = iota
	// AuthOther delegates a bystander address.
	AuthOther
	// AuthWrongChainID signs the authorization for a different chain.
	AuthWrongChainID
)
