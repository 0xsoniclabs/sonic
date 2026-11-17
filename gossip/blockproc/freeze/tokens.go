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

package freeze

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	transferTopic = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	approvalTopic = common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
)

// isFrozenTokenLog reports whether log is an ERC-20 Transfer or Approval
// naming a frozen account as either of its two addresses, emitted by a
// contract that is not opted out. The events are standard, so this covers
// every token without listing them.
func isFrozenTokenLog(db StateReader, log *types.Log) bool {
	if len(log.Topics) != 3 {
		return false // ERC-20 Transfer and Approval have exactly 3 topics
	}
	if log.Topics[0] != transferTopic && log.Topics[0] != approvalTopic {
		return false
	}
	return (IsFrozen(db, common.BytesToAddress(log.Topics[1].Bytes())) ||
		IsFrozen(db, common.BytesToAddress(log.Topics[2].Bytes()))) &&
		!IsOptedOut(db, log.Address)
}

func (s *StateDB) AddLog(log *types.Log) {
	s.StateDB.AddLog(log)
	if isFrozenTokenLog(s.StateDB, log) {
		s.frozenMoves++
	}
}
