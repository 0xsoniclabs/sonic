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
	"errors"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ErrFrozenAccount is returned if a message is sent by an account listed as
// frozen in the freeze registry.
var ErrFrozenAccount = errors.New("account is frozen")

// ApplyMessage runs msg like core.ApplyMessage. If checker is enabled, a frozen
// sender fails with ErrFrozenAccount.
func ApplyMessage(evm *vm.EVM, msg *core.Message, gp *core.GasPool, checker Checker) (*core.ExecutionResult, error) {
	// before the upgrade, run msg exactly as go-ethereum does
	if !checker.IsEnabled() {
		return core.ApplyMessage(evm, msg, gp)
	}

	// the sender is frozen, so fail immediately
	if checker.IsFrozen(msg.From) {
		return nil, ErrFrozenAccount
	}

	return core.ApplyMessage(evm, msg, gp)
}
