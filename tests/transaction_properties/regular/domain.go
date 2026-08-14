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

package regular

import (
	"context"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"pgregory.net/rapid"
)

// Domain draws ordinary transactions and prices them from their sender's balance. It needs nothing
// arranged on chain beyond the genesis funding the account pool already has, and adds no checks of
// its own: everything an ordinary transaction must satisfy is either a rule of the model or an
// invariant that holds for every transaction.
type Domain struct {
	Cfg core.GenConfig
}

func (d Domain) Batch() *rapid.Generator[[]core.TxSpec] {
	return rapid.Custom(func(t *rapid.T) []core.TxSpec {
		return GenBatch(t, d.Cfg)
	})
}

func (d Domain) Pricing() core.PricingRules {
	return PricingRules
}

func (d Domain) Prepare(*rapid.T, context.Context, []core.PooledAccount, []core.TxSpec) error {
	return nil
}

func (d Domain) Extra(*rapid.T, context.Context) ([]core.ExtraExecuted, error) {
	return nil, nil // an ordinary transaction is injected, or it does not execute
}

func (d Domain) Check(*rapid.T, context.Context, core.Observation) error {
	return nil
}
