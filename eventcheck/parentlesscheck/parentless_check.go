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

package parentlesscheck

import (
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
)

type Checker struct {
	HeavyCheck HeavyCheck
	LightCheck LightCheck
}

type LightCheck func(dag.Event) error

type HeavyCheck interface {
	Enqueue(e dag.Event, checked func(error)) error
}

// Enqueue tries to fill gaps the fetcher's future import queue.
func (c *Checker) Enqueue(e dag.Event, checked func(error)) {
	// Run light checks right away
	err := c.LightCheck(e)
	if err != nil {
		checked(err)
		return
	}

	// Run heavy check in parallel
	_ = c.HeavyCheck.Enqueue(e, checked)
}
