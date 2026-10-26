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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

func decodeTransaction(encoded hexutil.Bytes) (*types.Transaction, error) {
	transaction := new(types.Transaction)
	if err := transaction.UnmarshalBinary(encoded); err != nil {
		return nil, err
	}
	return transaction, nil
}

func (handler *handlers) sendRawTransaction(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	transaction, err := decodeTransaction(dispatch.Argument[hexutil.Bytes](arguments, 0))
	if err != nil {
		return nil, err
	}
	return handler.environment.Send(ctx, transaction)
}

// The errors of eth_sendRawTransactionSync, with the codes of EIP-7966.
const (
	codeSyncTimeout  = 4
	codeSyncRejected = 5
	codeSyncNonceGap = 6
)

var errZeroTimeout = errors.New("timeout must be greater than zero")

func syncTimeoutError(hash common.Hash) error {
	return &dispatch.Error{Code: codeSyncTimeout, Message: "transaction not confirmed within timeout", Data: hash}
}

func syncRejectedError(hash common.Hash, err error) error {
	return &dispatch.Error{Code: codeSyncRejected, Message: fmt.Sprintf("transaction was rejected: %v", err), Data: hash}
}

func syncNonceGapError(expected uint64) error {
	return &dispatch.Error{Code: codeSyncNonceGap, Message: fmt.Sprintf("nonce gap: expected nonce %d", expected), Data: hexutil.Uint64(expected)}
}

// sendRawTransactionSync submits a transaction and waits for its receipt,
// as EIP-7966 and the reference's handler say: a nonce above the pool's is
// refused before the pool sees it. The wait is on the head feed, not a poll
// (I9): the feed is taken before the transaction is submitted, so no block
// that holds it is missed.
func (handler *handlers) sendRawTransactionSync(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	transaction, err := decodeTransaction(dispatch.Argument[hexutil.Bytes](arguments, 0))
	if err != nil {
		return nil, err
	}
	hash := transaction.Hash()
	timeout, err := handler.syncTimeout(dispatch.Argument[*hexutil.Uint64](arguments, 1))
	if err != nil {
		return nil, err
	}
	sender, err := types.Sender(handler.environment.Signer(), transaction)
	if err != nil {
		return nil, err
	}
	if expected := handler.node.PoolNonce(sender); transaction.Nonce() > expected {
		return nil, syncNonceGapError(expected)
	}
	heads := handler.environment.Heads().Subscribe()
	defer heads.Close()
	if _, err := handler.environment.Send(ctx, transaction); err != nil {
		return nil, syncRejectedError(hash, err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return handler.awaitReceipt(ctx, heads, hash)
}

// syncTimeout is the wait the client asked for, bounded by TxSyncMaxTimeout,
// or the default.
func (handler *handlers) syncTimeout(requested *hexutil.Uint64) (time.Duration, error) {
	limits := handler.environment.Config()
	if requested == nil {
		return limits.TxSyncDefaultTimeout, nil
	}
	if *requested == 0 {
		return 0, errZeroTimeout
	}
	timeout := time.Duration(math.MaxInt64)
	if *requested < hexutil.Uint64(timeout/time.Millisecond) {
		timeout = time.Duration(*requested) * time.Millisecond
	}
	if limits.TxSyncMaxTimeout > 0 && timeout > limits.TxSyncMaxTimeout {
		timeout = limits.TxSyncMaxTimeout
	}
	return timeout, nil
}

// awaitReceipt waits for a block that holds the transaction. A feed that
// dropped its consumer is taken again, and the head it missed is searched
// once.
func (handler *handlers) awaitReceipt(ctx context.Context, heads *chain.Consumer[core.HeadEvent], hash common.Hash) (any, error) {
	current := heads
	defer func() {
		if current != heads {
			current.Close()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil, syncTimeoutError(hash)
		case event, open := <-current.Events():
			if !open {
				if current != heads {
					current.Close()
				}
				current = handler.environment.Heads().Subscribe()
				if receipt, err := handler.receiptAtLatest(ctx, hash); receipt != nil || err != nil {
					return receipt, err
				}
				continue
			}
			if receipt, err := handler.receiptIn(ctx, event.Block, hash); receipt != nil || err != nil {
				return receipt, err
			}
		}
	}
}

func (handler *handlers) receiptAtLatest(ctx context.Context, hash common.Hash) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	located, err := exec.NullIfAbsent(handler.environment.TransactionByHash(head, hash))
	if located == nil || err != nil {
		return nil, err
	}
	block, err := handler.environment.BlockAt(head, located.Block)
	if err != nil {
		return nil, err
	}
	return handler.receiptIn(ctx, block, hash)
}

// receiptIn answers the receipt of the transaction if the block holds it,
// nil if it does not.
func (handler *handlers) receiptIn(ctx context.Context, block *evmcore.EvmBlock, hash common.Hash) (any, error) {
	index := -1
	for position, transaction := range block.Transactions {
		if transaction.Hash() == hash {
			index = position
			break
		}
	}
	if index < 0 {
		return nil, nil
	}
	number := block.NumberU64()
	head := handler.environment.View(ctx, chain.Head{Number: number, Block: block})
	receipts, err := handler.environment.ReceiptsAt(head, number)
	if err != nil && !exec.Absent(err) {
		return nil, err
	}
	if len(receipts) <= index {
		return nil, fmt.Errorf("confirmed transaction %s found in block %d but receipt is unavailable or transaction index %d is out of range", hash, number, index)
	}
	return format.Receipt(block.Header(), block.Transactions[index], uint64(index), receipts[index], handler.environment.ChainID()), nil
}
