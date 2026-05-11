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

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestApi provides a set of API methods only to be used in Sonic integration
// tests. It is not intended to be used in production and may be modified at
// any time in the future.
type TestApi struct {
	backend Backend
}

func NewTestApi(backend Backend) *TestApi {
	return &TestApi{backend: backend}
}

// ProposeTransactions allows to propose a batch of transactions to be included
// in a block. If called on a validator node, the node will prepare a new event
// proposing the given transactions, without any prior validation.
//
// This method is intended to be used in integration tests to simulate the
// malicious behavior of a validator proposing invalid transactions.
func (a *TestApi) ProposeTransactions(
	_ context.Context,
	transactions [][]byte,
) error {
	if !a.backend.IsTestOnlyApiEnabled() {
		return fmt.Errorf("test-only API is not enabled")
	}
	txs := make(types.Transactions, len(transactions))
	for i, cur := range transactions {
		var tx types.Transaction
		if err := rlp.DecodeBytes(cur, &tx); err != nil {
			return err
		}
		txs[i] = &tx
	}
	return a.backend.ProposeTransactions(txs)
}
