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

package coretypes

import (
	"github.com/ethereum/go-ethereum/common"
)

//go:generate mockgen -source=interface.go -destination=interface_mock.go -package=coretypes

// DummyChain supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type DummyChain interface {
	// Header returns the header of the block with the given number.
	// If the block is not found, nil is returned.
	// If the hash provided is not zero and does not match, nil is returned.
	Header(hash common.Hash, number uint64) *EvmHeader
}
