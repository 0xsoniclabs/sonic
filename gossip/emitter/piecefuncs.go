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

package emitter

import (
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
)

var (
	// confirmingEmitIntervalF is a piecewise function for validator confirming internal depending on a stake amount before him
	confirmingEmitIntervalF = piecefunc.NewFunc([]piecefunc.Dot{
		{
			X: 0,
			Y: 1.0 * piecefunc.DecimalUnit,
		},
		{
			X: 0.78 * piecefunc.DecimalUnit,
			Y: 1.1 * piecefunc.DecimalUnit,
		},
		{
			X: 0.8 * piecefunc.DecimalUnit,
			Y: 10.0 * piecefunc.DecimalUnit,
		},
		{ // validators >0.8 emit confirming events very rarely
			X: 0.81 * piecefunc.DecimalUnit,
			Y: 50.0 * piecefunc.DecimalUnit,
		},
		{ // validators >0.8 emit confirming events very rarely
			X: 1.0 * piecefunc.DecimalUnit,
			Y: 60.0 * piecefunc.DecimalUnit,
		},
	})
	// scalarUpdMetricF is a piecewise function for validator's event metric diff depending on a number of newly observed events
	scalarUpdMetricF = piecefunc.NewFunc([]piecefunc.Dot{
		{
			X: 0,
			Y: 0,
		},
		{ // first observed event gives a major metric diff
			X: 1.0 * piecefunc.DecimalUnit,
			Y: 0.66 * piecefunc.DecimalUnit,
		},
		{ // second observed event gives a minor diff
			X: 2.0 * piecefunc.DecimalUnit,
			Y: 0.8 * piecefunc.DecimalUnit,
		},
		{ // other observed event give only a subtle diff
			X: 8.0 * piecefunc.DecimalUnit,
			Y: 0.99 * piecefunc.DecimalUnit,
		},
		{
			X: 100.0 * piecefunc.DecimalUnit,
			Y: 0.999 * piecefunc.DecimalUnit,
		},
		{
			X: 10000.0 * piecefunc.DecimalUnit,
			Y: 0.9999 * piecefunc.DecimalUnit,
		},
	})
)
