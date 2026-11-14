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
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// The error codes of eth_sendRawTransactionSync, as EIP-7966 names them.
const (
	codeSyncTimeout  = 4
	codeSyncRejected = 5
	codeSyncNonceGap = 6
)

type rawTransactionParams struct {
	Transaction hexutil.Bytes
}

// sendRawTransaction hands a signed transaction to the pool and answers its
// hash.
func (h *handlers) sendRawTransaction(ctx context.Context, params rawTransactionParams) (common.Hash, error) {
	transaction := new(types.Transaction)
	if err := transaction.UnmarshalBinary(params.Transaction); err != nil {
		return common.Hash{}, err
	}
	if err := h.environment.Send(ctx, transaction); err != nil {
		return common.Hash{}, err
	}
	return transaction.Hash(), nil
}

// pendingTransactions answers none: the pending transactions of the node's
// own accounts, of which the module knows none.
func pendingTransactions(context.Context, struct{}) ([]*format.Transaction, error) {
	return []*format.Transaction{}, nil
}

type rawTransactionSyncParams struct {
	Transaction hexutil.Bytes
	// TimeoutMs is the wait in milliseconds.
	TimeoutMs *hexutil.Uint64
}

// sendRawTransactionSync hands a signed transaction to the pool and answers
// its receipt once its block is served, as EIP-7966 specifies. A transaction
// whose nonce leaves a gap is not sent.
func (h *handlers) sendRawTransactionSync(ctx context.Context, params rawTransactionSyncParams) (map[string]any, error) {
	transaction := new(types.Transaction)
	if err := transaction.UnmarshalBinary(params.Transaction); err != nil {
		return nil, err
	}
	timeout, err := h.syncTimeout(params.TimeoutMs)
	if err != nil {
		return nil, err
	}
	sender, err := types.Sender(types.LatestSignerForChainID(h.environment.ChainID()), transaction)
	if err != nil {
		return nil, err
	}
	if expected := h.node.PoolNonce(sender); transaction.Nonce() > expected {
		return nil, &dispatch.Error{Code: codeSyncNonceGap, Message: fmt.Sprintf("nonce gap: expected nonce %d", expected), Data: hexutil.Uint64(expected)}
	}
	hash := transaction.Hash()
	if err := h.environment.Send(ctx, transaction); err != nil {
		return nil, &dispatch.Error{Code: codeSyncRejected, Message: fmt.Sprintf("transaction was rejected: %v", err), Data: hash}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		next := h.environment.NextHead()
		receipt, err := h.confirmedReceipt(hash)
		if receipt != nil || err != nil {
			return receipt, err
		}
		select {
		case <-ctx.Done():
			return nil, &dispatch.Error{Code: codeSyncTimeout, Message: "transaction not confirmed within timeout", Data: hash}
		case <-next:
		}
	}
}

// syncTimeout is the wait a request names, capped, or the default one.
func (h *handlers) syncTimeout(requested *hexutil.Uint64) (time.Duration, error) {
	config := h.environment.Config()
	if requested == nil {
		return config.TxSyncDefaultTimeout, nil
	}
	if *requested == 0 {
		return 0, errors.New("timeout must be greater than zero")
	}
	timeout := time.Duration(math.MaxInt64)
	if *requested < hexutil.Uint64(timeout/time.Millisecond) {
		timeout = time.Duration(*requested) * time.Millisecond
	}
	if config.TxSyncMaxTimeout > 0 && timeout > config.TxSyncMaxTimeout {
		timeout = config.TxSyncMaxTimeout
	}
	return timeout, nil
}

// confirmedReceipt is the receipt of a transaction of a served block, nil
// while no served block holds it.
func (h *handlers) confirmedReceipt(hash common.Hash) (map[string]any, error) {
	head := h.environment.Head()
	transaction, number, index, err := h.environment.TransactionByHash(head, hash)
	if exec.IsMissing(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	header, err := exec.NoneIfMissing(h.environment.HeaderAt(head, number))
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, fmt.Errorf("confirmed transaction %s found in block %d but the block is unavailable", hash, number)
	}
	receipts, err := exec.NoneIfMissing(h.environment.ReceiptsAt(head, number))
	if err != nil {
		return nil, err
	}
	if uint64(len(receipts)) <= index {
		return nil, fmt.Errorf("confirmed transaction %s found in block %d but receipt is unavailable or transaction index %d is out of range", hash, number, index)
	}
	return format.Receipt(header, transaction, index, receipts[index], h.environment.ChainID()), nil
}
