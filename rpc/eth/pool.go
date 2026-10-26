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

package eth

import (
	"context"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// pendingTransactions lists the pending transactions of the pool, priced at
// the suggested gas price, as the reference priced them. The reference
// listed only those of the accounts the node held, so none (section 6).
func (handler *handlers) pendingTransactions(ctx context.Context, _ dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	pending, err := handler.node.PoolTransactions()
	if err != nil {
		return nil, err
	}
	price, chainID := exec.SuggestedGasPrice(head), handler.environment.ChainID()
	transactions := make([]*format.Transaction, 0, len(pending))
	for _, transaction := range pending {
		transactions = append(transactions, format.PendingTransaction(transaction, price, chainID))
	}
	return transactions, nil
}
