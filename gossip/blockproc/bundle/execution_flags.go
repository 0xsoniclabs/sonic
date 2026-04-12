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

package bundle

import (
	"strings"
)

// ExecutionFlags represents the execution flags that specify the behavior of
// the bundle execution. Zero value means the default behavior, which is to
// revert the entire bundle if any of the transactions is invalid or fails.
type ExecutionFlags uint8

const (
	// EF_Default represents the default execution behavior, where invalid or
	// failed transactions are treated as failed.
	EF_Default ExecutionFlags = 0
	// EF_TolerateInvalid accepts invalid transactions as successfully executed.
	EF_TolerateInvalid ExecutionFlags = 0b01
	// EF_TolerateFailed accepts failed transactions as successfully executed.
	EF_TolerateFailed ExecutionFlags = 0b10

	// numUsedBits of supported execution flags.
	numUsedBits = 2
)

// Valid checks whether there are no unknown flags set in the execution flags.
func (e ExecutionFlags) Valid() bool {
	return e < 1<<numUsedBits
}

// TolerateInvalid checks whether the execution flags allow invalid transactions
// to be treated as successful.
func (e ExecutionFlags) TolerateInvalid() bool {
	return e.getFlag(EF_TolerateInvalid)
}

// TolerateFailed checks whether the execution flags allow failed transactions
// to be treated as successful.
func (e ExecutionFlags) TolerateFailed() bool {
	return e.getFlag(EF_TolerateFailed)
}

func (e ExecutionFlags) getFlag(flag ExecutionFlags) bool {
	return e&flag != 0
}

func (e ExecutionFlags) String() string {
	var flags []string
	if e.TolerateInvalid() {
		flags = append(flags, "TolerateInvalid")
	}
	if e.TolerateFailed() {
		flags = append(flags, "TolerateFailed")
	}
	if len(flags) == 0 {
		return "Default"
	}
	return strings.Join(flags, "|")
}
