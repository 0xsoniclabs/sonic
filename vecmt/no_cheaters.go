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

package vecmt

import (
	"errors"

	"github.com/Fantom-foundation/lachesis-base/hash"
)

// NoCheaters excludes events which are observed by selfParents as cheaters.
// Called by emitter to exclude cheater's events from potential parents list.
func (vi *Index) NoCheaters(selfParent *hash.Event, options hash.Events) hash.Events {
	if selfParent == nil {
		return options
	}
	vi.InitBranchesInfo()

	if !vi.AtLeastOneFork() {
		return options
	}

	// no need to merge, because every branch is marked by IsForkDetected if fork is observed
	highest := vi.Base.GetHighestBefore(*selfParent)
	filtered := make(hash.Events, 0, len(options))
	for _, id := range options {
		e := vi.getEvent(id)
		if e == nil {
			vi.crit(errors.New("event not found"))
		}
		if !highest.Get(vi.validatorIdxs[e.Creator()]).IsForkDetected() {
			filtered.Add(id)
		}
	}
	return filtered
}
