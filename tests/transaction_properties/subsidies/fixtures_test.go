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

package subsidies

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// fakeSpec is a transaction of the plainest kind there is, for the checks that need a spec but no
// generator. It is assembled from the same capabilities the real types are, so what it answers is
// what a real one would.
type fakeSpec struct {
	core.Envelope
	core.Payload
	core.OptionalRecipient
	core.WideValue
	core.SinglePrice
}

func (fakeSpec) TxType() uint8 { return types.LegacyTxType }

func (s *fakeSpec) TxData(nonce uint64, ctx core.BuildContext) (types.TxData, error) {
	return &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: s.GasPrice,
		Gas:      s.GasLimit,
		To:       ctx.RecipientAddress(s),
		Value:    s.Amount(),
		Data:     s.Data(),
	}, nil
}

// request is a sponsorship request from the given sender, asking for the given gas.
func request(sender int, gas uint64) *fakeSpec {
	return &fakeSpec{
		Envelope:    core.Envelope{SenderIdx: sender, GasLimit: gas},
		WideValue:   core.WideValue{Value: new(big.Int)},
		SinglePrice: core.SinglePrice{GasPrice: new(big.Int)},
	}
}

// paying is a transaction that pays its own way.
func paying(sender int, gas uint64) *fakeSpec {
	spec := request(sender, gas)
	spec.GasPrice = new(big.Int).Set(core.MinimumViableFeeCap)
	return spec
}

// registryCall builds the internal transaction a sponsorship's follow-up is, so a check can be given
// a well-formed one and every way of spoiling it.
func registryCall(selector uint32, id [32]byte, amount *big.Int) *types.Transaction {
	input := make([]byte, 0, 4+2*32)
	input = append(input, byte(selector>>24), byte(selector>>16), byte(selector>>8), byte(selector))
	input = append(input, id[:]...)
	word := uint256.MustFromBig(amount).Bytes32()
	input = append(input, word[:]...)

	to := registry.GetAddress()
	return types.NewTx(&types.LegacyTx{
		Nonce:    0,
		GasPrice: new(big.Int),
		Gas:      100_000,
		To:       &to,
		Value:    new(big.Int),
		Data:     input,
	})
}

// feeCharge is the deductFees transaction a fund-backed sponsorship is followed by.
func feeCharge(amount *big.Int) *types.Transaction {
	return registryCall(registry.DeductFeesFunctionSelector, [32]byte{0x01}, amount)
}

// track is the transaction a network sponsorship with tracking is followed by.
func track(amount *big.Int) *types.Transaction {
	return registryCall(registry.TrackFunctionSelector, [32]byte{0x02}, amount)
}

// plainTx is an ordinary transaction, which no check may mistake for a follow-up.
func plainTx(nonce uint64) *types.Transaction {
	to := common.Address{0xaa}
	return types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1),
		Gas:      21_000,
		To:       &to,
		Value:    new(big.Int),
	})
}
