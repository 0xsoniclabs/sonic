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

// Package bundles layers transaction bundles over another domain's transactions: it takes that
// domain's generator and replaces part of a batch with an envelope carrying a batch of its own,
// builds that bundle with the production builder, and checks that the bundle either executed whole or
// not at all. It is the first domain that is a container rather than a property of a transaction, and
// the only one whose transactions reach a block without having been injected -- which is what
// core.Domain.Extra exists for.
package bundles

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Envelope is a transaction carrying a bundle: a call to the bundle processor whose payload is the
// encoded contents and execution plan. It is a spec like any other, so the builder signs it with the
// drawn signing choice and the model judges it with the same rule table -- but the capabilities it
// carries are only the ones an envelope really has. It has no value, no access list, no blob hashes
// and no authorizations, and its payload and gas limit are not drawn at all: both follow from the
// bundle, which can only be built once the accounts and the block number are known.
//
// Contents and the choices below are what the generator draws; Built is what Prepare fills in.
type Envelope struct {
	core.Envelope
	core.SinglePrice

	// Contents is the batch this envelope carries. Its senders are indexed from zero like any batch
	// and are resolved against the second window of the claimed accounts -- see Domain.Offset.
	Contents []core.TxSpec

	// Range says where the execution plan's block range sits relative to the block the bundle would
	// land in, and Root whether the plan is a group or a bare step.
	Range RangeChoice
	Root  RootChoice

	// Declared says whether the envelope declares the gas limit validation demands, or one either
	// side of it, which ValidateEnvelope refuses.
	Declared GasChoice

	// Built is what Prepare made of all that. Until then the envelope has no payload, and nothing
	// reads one: Prepare runs before the model and the builder both.
	Built *Built
}

// Built is the bundle as the production builder made it.
type Built struct {
	// Payload and GasLimit are what the envelope transaction carries. GasLimit is what the Gas choice
	// asked for, which is CalculateEnvelopeGas only when that choice was Exact.
	Payload  []byte
	GasLimit uint64

	// PlanHash identifies the bundle to sonic_getBundleInfo, and is what replay protection keys on.
	PlanHash common.Hash

	// Plan is the execution plan the builder made, kept so a failure can show the tree that produced
	// it. Which shape the root has decides whether a failing transaction is rolled back, so a report
	// without it names a defect the reader cannot see.
	Plan bundle.ExecutionPlan

	// Inner are the contents as signed transactions, in the order the plan references them, which is
	// the order they must appear in a block.
	Inner []*types.Transaction

	// Refused says the model expects the whole set to be refused before any of it is opened, which is
	// what an unsupported transaction type among the contents means.
	Refused bool

	// Plans are what the model made of the contents, planned as the batch they are: the nonce each one
	// carries and what would become of it on its own.
	Plans []core.TxPlan
}

// RangeChoice places the execution plan's block range relative to the current block.
//
//go:generate go tool stringer -type=RangeChoice -trimprefix=Range
type RangeChoice uint8

const (
	// RangeCovering starts at the current block and runs for the longest range allowed, so it covers
	// whichever of the next blocks the bundle lands in.
	RangeCovering RangeChoice = iota
	// RangeBefore ends before the current block, so the bundle is too late.
	RangeBefore
	// RangeAfter starts well after the current block, so the bundle is too early.
	RangeAfter
)

// RootChoice is the shape of the execution plan's root step.
//
//go:generate go tool stringer -type=RootChoice -trimprefix=Root
type RootChoice uint8

const (
	// RootAllOf wraps the contents in a group that must succeed as a whole.
	RootAllOf RootChoice = iota
	// RootSingle carries one transaction as a bare step, with no group around it. Nothing else
	// generates this shape, and the block processor takes no snapshot of its own for it.
	RootSingle
)

// GasChoice is what the envelope declares as its gas limit.
//
//go:generate go tool stringer -type=GasChoice -trimprefix=Gas
type GasChoice uint8

const (
	// GasExact declares what CalculateEnvelopeGas requires, which is the only accepted value.
	GasExact GasChoice = iota
	// GasTooLow declares one less.
	GasTooLow
	// GasTooHigh declares one more.
	GasTooHigh
)

// The capabilities an envelope does not carry, answered here so that what it holds stays explicit.

// Data is the encoded bundle, which is also what the model prices as the payload.
func (e *Envelope) Data() []byte {
	if e.Built == nil {
		return nil
	}
	return e.Built.Payload
}

// Gas returns the declared gas limit. It shadows the drawn one from core.Envelope, since an
// envelope's is dictated by its contents rather than drawn.
func (e *Envelope) Gas() uint64 {
	if e.Built == nil {
		return 0
	}
	return e.Built.GasLimit
}

// Recipient reports a bystander, because the real recipient is the bundle processor and no drawn
// choice names it. TxData below is what actually addresses the transaction.
func (e *Envelope) Recipient() core.ToChoice { return core.ToOther }

func (e *Envelope) IsCreate() bool { return false }

// Amount is zero: an envelope transfers nothing, and nothing is charged to it.
func (e *Envelope) Amount() *big.Int { return new(big.Int) }

func (e *Envelope) TxType() uint8 { return types.LegacyTxType }

// TxData assembles the envelope transaction. It is a plain call to the bundle processor, so
// core.SignSpec signs it with the drawn signing choice like any other transaction -- which is
// deliberate: an envelope whose signature recovers to nobody still runs its bundle, because nothing
// on that path needs its sender.
func (e *Envelope) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	if e.Built == nil {
		return nil, fmt.Errorf("the bundle of this envelope has not been built")
	}
	to := bundle.BundleProcessor
	return &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: e.GasPrice,
		Gas:      e.Built.GasLimit,
		To:       &to,
		Value:    new(big.Int),
		Data:     e.Built.Payload,
	}, nil
}

// reference is the name the plan's own rendering gives the transaction at one index: it letters them
// in the order the plan references them, which is the order the contents are built in.
func reference(index int) string {
	return string([]byte{byte('A' + index)})
}

// String renders the envelope, the plan tree, and what it carries, since core.Format only knows what
// every spec has and a bundle that went wrong is unreadable without its plan and contents.
func (e *Envelope) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "bundle[root=%v range=%v declaredGas=%v", e.Root, e.Range, e.Declared)
	if e.Built != nil {
		fmt.Fprintf(&out, " gas=%d hash=%v payload=%dB\n      plan %s over blocks %v",
			e.Built.GasLimit, e.Built.PlanHash.TerminalString(), len(e.Built.Payload),
			e.Built.Plan.Root.String(), e.Built.Plan.Range)
	}
	for i, spec := range e.Contents {
		nonce := "unassigned"
		if e.Built != nil && i < len(e.Built.Plans) {
			nonce = fmt.Sprint(e.Built.Plans[i].Nonce)
		}
		fmt.Fprintf(&out, "\n      carrying %s nonce=%s %s", reference(i), nonce, core.Format(spec))
	}
	out.WriteString("]")
	return out.String()
}
