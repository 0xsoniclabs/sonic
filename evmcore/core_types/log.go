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

package core_types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Log is a reduced version of types.Log that only includes the fields covered
// by the consensus. In particular, it does not include the transaction index,
// block hash, block number, or log index.
type Log struct {
	// address of the contract that generated the event
	Address common.Address
	// list of topics provided by the contract.
	Topics []common.Hash
	// supplied by the contract, usually ABI-encoded
	Data []byte
}

// CoreLogFromGethLog converts a types.Log from the go-ethereum library to a
// core_types.Log used internally in Sonic.
func CoreLogFromGethLog(l *types.Log) *Log {
	if l == nil {
		return nil
	}
	return &Log{
		Address: l.Address,
		Topics:  l.Topics,
		Data:    l.Data,
	}
}
