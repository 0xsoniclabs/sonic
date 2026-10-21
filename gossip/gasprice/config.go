// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gasprice

import (
	"math"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/params"
)

// DefaultMaxGasPrice provides the recommended default maximum gas price
// for the gas price oracle. It always returns a new *big.Int instance to ensure
// that each caller receives an independent value. This avoids accidental sharing
// of mutable state between different Oracle configurations or usages.
// Use this function when initializing Oracle configs or as a safe default
// in gas price logic.
func DefaultMaxGasPrice() *big.Int {
	return big.NewInt(10000000 * params.GWei)
}

const (
	AsDefaultCertainty = math.MaxUint64
	DecimalUnit        = piecefunc.DecimalUnit
)

// Config is the "GPO" section of the configuration file. The gas price oracle
// itself runs in the RPC module, which reads these settings.
type Config struct {
	MaxGasPrice      *big.Int `toml:",omitempty"`
	MinGasPrice      *big.Int `toml:",omitempty"`
	DefaultCertainty uint64   `toml:",omitempty"`
}
