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

package testapi

import "github.com/ethereum/go-ethereum/core/types"

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=testapi

type Backend interface {
	// IsTestOnlyApiEnabled shall return true if in the current node access to
	// the test-only API shall be granted. This is an extra safety measure to
	// prevent accidental usage of the test-only API in production environments.
	IsTestOnlyApiEnabled() bool

	// ProposeTransactions allows to propose a batch of transactions to be
	// included in a block. If called on a validator node, the node will prepare
	// a new event proposing the given transactions, without any prior validation.
	//
	// This method is intended to be used in integration tests to simulate the
	// malicious behavior of a validator proposing invalid transactions.
	ProposeTransactions(txs types.Transactions) error
}
