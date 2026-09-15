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

package core

import (
	"math/big"
	"strconv"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core/contracts"
	"pgregory.net/rapid"
)

const (
	// MaxDataLen is deliberately well below params.MaxInitCodeSize, because an oversized init code
	// behaves inconsistently across forks and the model cannot predict one outcome for it. Short
	// payloads still reach the intrinsic and floor data gas boundaries, which is the point of varying
	// the length.
	MaxDataLen = 2048

	// MaxAccessListLen at 4700 intrinsic gas per entry is enough to push a transaction over a modest
	// gas limit without inflating a batch's total.
	MaxAccessListLen = 8
)

// GenConfig parameterises generation for one network. The gas budget stays clearly below MaxEventGas,
// since a batch rejected for exceeding it starves every other case of coverage.
type GenConfig struct {
	Network             NetworkConfig
	MaxTxsPerBatch      int
	MaxAccountsPerBatch int
	GasBudget           uint64

	// Contracts are the applications deployed on this network before the run, which a transaction
	// may be drawn to call. Empty leaves every payload meaningless to whoever receives it.
	Contracts []contracts.Deployed
}

// GenEnv is the state a draw needs but does not draw: the network being generated for, how many
// senders the batch shares, what this transaction may spend, and which transaction of the batch it is.
// Every capability is handed the same one, and takes from it only what its own fields need.
type GenEnv struct {
	Network     NetworkConfig
	Senders     int
	GasCeiling  uint64
	ClaimAllGas bool // ask for the whole event allowance rather than a share of the budget
	Index       int
	Contracts   []contracts.Deployed
}

// label names a draw after the transaction it belongs to, so a failure report says where a value came
// from.
func (e GenEnv) Label(name string) string {
	return name + "_" + strconv.Itoa(e.Index)
}

// Drawable is a spec that fills its own fields. Every transaction type implements it by drawing each
// capability it embeds, in an order it chooses.
type Drawable interface {
	TxSpec
	Draw(t *rapid.T, env GenEnv)
}

// Each capability draws its own fields. The draws are weighted towards values that let a transaction
// execute, because a generator producing uniform garbage never reaches past the first check that
// rejects it. A drawn *big.Int is shared between iterations, so nothing may modify one: types.NewTx
// copies what it is given, and the model only ever reads.

func (e *Envelope) Draw(t *rapid.T, env GenEnv) {
	e.SenderIdx = rapid.IntRange(0, env.Senders-1).Draw(t, env.Label("sender"))
	e.Nonce = Nonces.Draw(t, env.Label("nonce"))
	e.NonceGapSize = rapid.Uint64Range(1, 4).Draw(t, env.Label("nonceGap"))
	e.Signing = Signatures.Draw(t, env.Label("signing"))
}

// DrawFor draws the payload for a transaction already known to be going somewhere, because what the
// payload should be depends on it: call data for a deployed contract, and bytes nobody will read for
// anything else. It is drawn after the recipient for that reason alone.
func (p *Payload) DrawFor(t *rapid.T, env GenEnv, to ToChoice) {
	if to == ToContract && len(env.Contracts) > 0 {
		p.Call = contracts.DrawCall(t, env.Label("call"), env.Contracts)
		return
	}
	p.DataLen = DataLengths.Draw(t, env.Label("dataLen"))
	p.DataNonZero = rapid.Bool().Draw(t, env.Label("dataNonZero"))
}

func (r *OptionalRecipient) Draw(t *rapid.T, env GenEnv) {
	r.To = rapid.SampledFrom(RecipientChoices(env, ToCreate)).Draw(t, env.Label("to"))
}

func (r *RequiredRecipient) Draw(t *rapid.T, env GenEnv) {
	r.To = rapid.SampledFrom(RecipientChoices(env)).Draw(t, env.Label("to"))
}

// RecipientChoices are the recipients a transaction can be drawn towards: the ones every transaction
// has, whatever else the capability admits, and a deployed contract where the network has any.
func RecipientChoices(env GenEnv, also ...ToChoice) []ToChoice {
	choices := append([]ToChoice{ToSelf, ToOther, ToPrecompile}, also...)
	if len(env.Contracts) > 0 {
		choices = append(choices, ToContract)
	}
	return choices
}

func (v *WideValue) Draw(t *rapid.T, env GenEnv) {
	v.Value = Values.Draw(t, env.Label("value"))
}

func (v *NarrowValue) Draw(t *rapid.T, env GenEnv) {
	v.Value = Values.Draw(t, env.Label("value"))
}

func (p *SinglePrice) Draw(t *rapid.T, env GenEnv) {
	p.GasPrice = FeeCaps.Draw(t, env.Label("gasPrice"))
}

func (p *WidePrices) Draw(t *rapid.T, env GenEnv) {
	p.GasFeeCap = FeeCaps.Draw(t, env.Label("feeCap"))
	p.GasTipCap = TipCapsFor(p.GasFeeCap).Draw(t, env.Label("tipCap"))
}

func (p *NarrowPrices) Draw(t *rapid.T, env GenEnv) {
	p.GasFeeCap = FeeCaps.Draw(t, env.Label("feeCap"))
	p.GasTipCap = TipCapsFor(p.GasFeeCap).Draw(t, env.Label("tipCap"))
}

func (a *AccessListEntries) Draw(t *rapid.T, env GenEnv) {
	a.AccessListLen = rapid.IntRange(0, MaxAccessListLen).Draw(t, env.Label("accessList"))
}

func (b *BlobHashList) Draw(t *rapid.T, env GenEnv) {
	b.BlobHashLen = rapid.IntRange(0, 2).Draw(t, env.Label("blobHashes"))
}

func (a *AuthList) Draw(t *rapid.T, env GenEnv) {
	a.Entries = Authorizations.Draw(t, env.Label("auths"))
}

// The shared generators every capability draws from.
//
// A value appearing more than once is how these are weighted: SampledFrom and OneOf pick an entry by
// index, so multiplicity raises the odds of that entry. Only approximately, though -- the index comes
// from rapid's biased draw, which favours the front of the list, so an earlier entry gets somewhat
// more than its share. Measured over 20k draws: the eight SignCorrect entries of thirteen below come
// out at 71% rather than 62%, the three correct nonces of six at 57% rather than 50%, and the single
// entry past a fork's maximum transaction type comes out rarer than its ninth.
//
// So the order is part of the intent, not decoration: what is worth drawing most goes first, which is
// also the direction rapid shrinks a failing case towards.
var (
	// Nonces are weighted towards correct ones.
	Nonces = rapid.SampledFrom([]NonceChoice{
		NonceCorrect, NonceCorrect, NonceCorrect,
		NonceTooLow, NonceGap, NonceMax,
	})

	// Signatures are almost always correct, because a bad one takes its whole batch down and frequent
	// ones would leave everything downstream unreached.
	Signatures = rapid.SampledFrom([]SigningChoice{
		SignCorrect, SignCorrect, SignCorrect, SignCorrect, SignCorrect,
		SignCorrect, SignCorrect, SignCorrect,
		SignWrongChainID, SignUnfundedKey, SignZeroR, SignZeroS, SignForeign,
	})

	// Authorizations include the empty list that the block formation filter must catch.
	Authorizations = rapid.SliceOfN(
		rapid.SampledFrom([]AuthChoice{AuthSelf, AuthOther, AuthWrongChainID}), 0, 3,
	)

	// DataLengths concentrate on the small lengths where the gas boundaries live.
	DataLengths = rapid.OneOf(
		rapid.Just(0),
		rapid.IntRange(1, 64),
		rapid.IntRange(0, MaxDataLen),
	)

	// Values straddle the pooled accounts' balance, so the insufficient-funds boundary is hit from
	// both sides, and reach past 256 bits.
	Values = rapid.OneOf(
		rapid.Just(big.NewInt(0)),
		rapid.Just(big.NewInt(1)),
		rapid.Just(new(big.Int).Div(AccountBalance, big.NewInt(2))),
		rapid.Just(new(big.Int).Sub(AccountBalance, big.NewInt(1))),
		rapid.Just(new(big.Int).Set(AccountBalance)),
		rapid.Just(new(big.Int).Add(AccountBalance, big.NewInt(1))),
		rapid.Just(new(big.Int).Mul(AccountBalance, big.NewInt(2))),
		WideValues,
	)

	// FeeCaps are either clearly unusable or clearly sufficient and never in the narrow band around
	// the base fee, which moves as blocks are produced and would make the outcome depend on timing
	// rather than on the transaction.
	FeeCaps = rapid.OneOf(
		rapid.Just(new(big.Int).Set(MinimumViableFeeCap)),
		rapid.Just(new(big.Int).Set(MinimumViableFeeCap)),
		rapid.Just(new(big.Int).Mul(MinimumViableFeeCap, big.NewInt(10))),
		rapid.Just(new(big.Int).Mul(MinimumViableFeeCap, big.NewInt(10))),
		rapid.Just(big.NewInt(0)),
		rapid.Just(big.NewInt(1)),
		WideValues,
	)

	// WideValues draw up to 33 bytes, one more than a uint256 holds, so that the range checks in
	// ValidateTxStatic are reachable. The leading 0xff makes a 33-byte draw reliably exceed 256 bits
	// rather than merely possibly.
	WideValues = rapid.Custom(func(t *rapid.T) *big.Int {
		size := rapid.IntRange(30, 33).Draw(t, "wideValueBytes")
		bytes := make([]byte, size)
		bytes[0] = 0xff
		for i := 1; i < size; i++ {
			bytes[i] = rapid.Byte().Draw(t, "wideValueByte")
		}
		return new(big.Int).SetBytes(bytes)
	})

	// OversizedBatch fires rarely, and only then can a batch reach the event gas allowance, which is
	// otherwise unreachable from Allegro onwards where no representable transaction exceeds the
	// maximum type.
	OversizedBatch = rapid.SampledFrom([]bool{false, false, false, false, false, true})
)

// TipCapsFor draws a tip cap at or below the fee cap, and occasionally above it, which execution must
// reject.
func TipCapsFor(feeCap *big.Int) *rapid.Generator[*big.Int] {
	return rapid.OneOf(
		rapid.Just(big.NewInt(0)),
		rapid.Just(big.NewInt(0)),
		rapid.Just(feeCap),
		rapid.Just(new(big.Int).Add(feeCap, big.NewInt(1))),
	)
}
