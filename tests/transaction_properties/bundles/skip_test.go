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

package bundles

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// contentSpec is the least a transaction has to be for the content filter to walk it: a sender, a
// recipient, a value and a price. Only what the balance arithmetic reads is filled in.
type contentSpec struct {
	core.Envelope
	core.Payload
	core.OptionalRecipient
	core.WideValue
	core.SinglePrice
}

func (contentSpec) TxType() uint8 { return types.LegacyTxType }

func (contentSpec) TxData(uint64, core.BuildContext) (types.TxData, error) { return nil, nil }

func content(to core.ToChoice, value *big.Int) *contentSpec {
	return &contentSpec{
		Envelope:          core.Envelope{GasLimit: 100_000},
		OptionalRecipient: core.OptionalRecipient{To: to},
		WideValue:         core.WideValue{Value: value},
		SinglePrice:       core.SinglePrice{GasPrice: new(big.Int).Set(core.MinimumViableFeeCap)},
	}
}

// TestDropOffending_JudgesEachContentAgainstWhatTheOnesAheadOfItLeave is the same property the
// injected batch has, one layer down: a bundle is a batch, and a creation carried behind transfers
// that hand the balance away meets what they leave rather than what the sender opened with. The order
// is the one the bundle is written in, since the execution plan runs its contents as it references
// them and nothing scrambles the inside of a bundle.
func TestDropOffending_JudgesEachContentAgainstWhatTheOnesAheadOfItLeave(t *testing.T) {
	baseFee := big.NewInt(1)
	half := new(big.Int).Div(core.AccountBalance, big.NewInt(2))
	quarter := new(big.Int).Div(core.AccountBalance, big.NewInt(4))

	creation := content(core.ToCreate, half)
	contents := []core.TxSpec{
		content(core.ToOther, half),
		content(core.ToOther, quarter),
		creation,
	}

	domain := &Domain{
		Inner:   regular.Domain{},
		inner:   []core.SenderState{{Balance: new(big.Int).Set(core.AccountBalance)}},
		skipped: map[string]int{},
	}

	kept := domain.dropOffending(contents, baseFee)
	require.Equal(t, contents[:2], kept,
		"the creation cannot hand over its value once the transfers ahead of it have run")
	require.Equal(t, 1, domain.skipped[regular.Domain{}.Notes()[0]],
		"the reason must be tallied for the report")

	// The creation on its own has the whole balance behind it, so nothing keeps it out.
	kept = domain.dropOffending([]core.TxSpec{creation}, baseFee)
	require.Equal(t, []core.TxSpec{creation}, kept)
}
