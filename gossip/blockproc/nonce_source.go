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

package blockproc

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter/state"
)

// NewNonceSource returns a NonceSource backed by the given state database. It
// exposes only the zero-address nonce, hiding the rest of the state from the
// internal-transaction builders.
func NewNonceSource(statedb state.StateDB) NonceSource {
	return stateDBNonceSource{statedb: statedb}
}

// stateDBNonceSource adapts a state.StateDB to a NonceSource.
type stateDBNonceSource struct {
	statedb state.StateDB
}

func (s stateDBNonceSource) ZeroAddressNonce() uint64 {
	return s.statedb.GetNonce(common.Address{})
}
