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
// weight, and an entity id, and using those to eagerly emit prioritized
// transactions and to order them first during block formation.
package priorities

import "cmp"

// Priority is the result of a getPriority query for a single transaction.
//
// Level zero means the transaction is not prioritized. A higher level forms an
// earlier partition (scheduled before lower levels). Weight breaks ties within a
// level (higher first). ID identifies the entity the transaction belongs to and
// is used for per-entity rate limiting. The semantics of ID are opaque to this
// code and only interpreted by the registry.
type Priority struct {
	Level  uint64
	Weight uint64
	ID     [16]byte
}

// IsPrioritized reports whether the transaction has a non-zero priority level.
func (p Priority) IsPrioritized() bool {
	return p.Level != 0
}

// Cmp compares two priorities by (level, weight), returning -1, 0, or +1.
// A higher level takes precedence; weight breaks ties within the same non-zero
// level. Two non-prioritized entries (level zero) always compare equal
// regardless of weight.
func (p Priority) Cmp(other Priority) int {
	if c := cmp.Compare(p.Level, other.Level); c != 0 {
		return c
	}
	if p.Level == 0 {
		return 0
	}
	return cmp.Compare(p.Weight, other.Weight)
}

// Config holds the per-entity rate limits returned by the registry's
// getPriorityConfig function.
type Config struct {
	// MaxGasPerEntityPerBlock bounds the total gas limit of prioritized
	// transactions of one entity in a single block (authoritative, enforced at
	// block formation).
	// Transactions are packed in (level desc, weight desc, hash asc) order
	// until the next transaction would exceed the budget; the remainder is
	// demoted.
	MaxGasPerEntityPerBlock uint64
	// MaxPiggybackTxsPerEntityPerEvent bounds how many prioritized transactions
	// of one entity a validator eagerly includes in a single emitted event
	// (best-effort hint). It applies only to prioritized transactions admitted
	// while it is not this validator's turn; transactions admitted on the
	// validator's own turn are not bounded by this limit.
	MaxPiggybackTxsPerEntityPerEvent uint64
}
