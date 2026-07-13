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

// Package priorities implements the transaction-priorities feature: querying an
// on-chain registry contract to determine, per transaction, a priority level, a
// weight, and an entity id, and using those to order transactions during block
// formation. See transaction_priorities.md for the design.
package priorities

import "github.com/holiman/uint256"

// Priority is the result of a getPriority query for a single transaction.
//
// Level zero means the transaction is not prioritized. A higher level forms an
// earlier partition (scheduled before lower levels). Weight breaks ties within a
// level (higher first). Id identifies the entity the transaction belongs to and
// is used for per-entity rate limiting. The semantics of Id are opaque to this
// code and only interpreted by the registry.
type Priority struct {
	Level  uint256.Int
	Weight uint256.Int
	Id     [32]byte
}

// IsPrioritized reports whether the transaction has a non-zero priority level.
func (p Priority) IsPrioritized() bool {
	return p.Level.Sign() > 0
}

// Cmp compares two priorities by (level, weight), returning -1, 0, or +1.
// A higher level takes precedence; weight breaks ties within the same level.
func (p Priority) Cmp(other Priority) int {
	if c := p.Level.Cmp(&other.Level); c != 0 {
		return c
	}
	return p.Weight.Cmp(&other.Weight)
}

// zeroPriority returns a normalized non-prioritized Priority.
func zeroPriority() Priority {
	return Priority{}
}

// Config holds the per-entity rate limits returned by the registry's
// getPriorityConfig function.
type Config struct {
	// MaxGasPerEntityPerBlock bounds the total gas of prioritized transactions
	// of one entity in a single block (authoritative, enforced at block
	// formation). Transactions are packed in (level desc, weight desc, hash asc)
	// order until the next transaction would exceed the budget; the remainder
	// is demoted.
	MaxGasPerEntityPerBlock uint64
	// MaxTxsPerEntityPerEvent bounds how many transactions of one entity a
	// validator eagerly includes in a single emitted event (best-effort hint).
	MaxTxsPerEntityPerEvent uint64
}
