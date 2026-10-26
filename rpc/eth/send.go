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

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// EIP-7966 error codes of eth_sendRawTransactionSync.
const (
	codeSyncTimeout  = 4 // in the pool, not processed within the timeout
	codeSyncRejected = 5 // not added to the pool
	codeSyncNonceGap = 6 // not added to the pool: the nonce leaves a gap
)

type rawTxArg struct {
	Tx hexutil.Bytes `rpc:"tx"`
}

type rawTxSyncArgs struct {
	Tx        hexutil.Bytes   `rpc:"tx"`
	TimeoutMs *hexutil.Uint64 `rpc:"timeout"`
}

func decodeTx(encoded hexutil.Bytes) (*types.Transaction, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encoded); err != nil {
		return nil, err
	}
	return tx, nil
}

func (a *api) sendRawTransaction(ctx context.Context, p rawTxArg) (any, error) {
	tx, err := decodeTx(p.Tx)
	if err != nil {
		return nil, err
	}
	return a.env.Submit(ctx, tx)
}

// sendRawTransactionSync submits a transaction and waits for its receipt. It
// waits on the head feed, not by polling: every new head is a chance the
// transaction was included.
func (a *api) sendRawTransactionSync(ctx context.Context, p rawTxSyncArgs) (any, error) {
	tx, err := decodeTx(p.Tx)
	if err != nil {
		return nil, err
	}
	hash := tx.Hash()

	cfg := a.env.Config()
	timeout := cfg.TxSyncDefaultTimeout
	if p.TimeoutMs != nil {
		if *p.TimeoutMs == 0 {
			return nil, errors.New("timeout must be greater than zero")
		}
		requested := time.Duration(math.MaxInt64)
		if *p.TimeoutMs < hexutil.Uint64(requested/time.Millisecond) {
			requested = time.Duration(*p.TimeoutMs) * time.Millisecond
		}
		timeout = requested
		if cfg.TxSyncMaxTimeout > 0 && timeout > cfg.TxSyncMaxTimeout {
			timeout = cfg.TxSyncMaxTimeout
		}
	}

	from, err := types.Sender(a.env.Signer(), tx)
	if err != nil {
		return nil, err
	}
	if expected := a.pool.PoolNonce(from); tx.Nonce() > expected {
		return nil, &dispatch.Error{Code: codeSyncNonceGap, Message: fmt.Sprintf("nonce gap: expected nonce %d", expected), Data: hexutil.Uint64(expected)}
	}

	// Subscribe before submitting, so that no block including the
	// transaction is missed.
	heads := a.env.Chain().Heads().Subscribe()
	defer func() { heads.Close() }() // the consumer in use at the end

	if _, err := a.env.Submit(ctx, tx); err != nil {
		return nil, &dispatch.Error{Code: codeSyncRejected, Message: fmt.Sprintf("transaction was rejected: %v", err), Data: hash}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		receipt, found, err := a.syncReceipt(ctx, hash)
		if err != nil {
			return nil, err
		}
		if found {
			return receipt, nil
		}
		select {
		case <-heads.Events():
		case <-heads.Dropped():
			// The consumer fell behind; the next look at the chain catches
			// up, and a new consumer follows from here.
			heads.Close()
			heads = a.env.Chain().Heads().Subscribe()
		case <-ctx.Done():
			return nil, &dispatch.Error{Code: codeSyncTimeout, Message: "transaction not confirmed within timeout", Data: hash}
		}
	}
}

// syncReceipt looks the transaction up at the head served now.
func (a *api) syncReceipt(ctx context.Context, hash common.Hash) (map[string]any, bool, error) {
	head := a.env.Latest(ctx)
	tx, number, index, err := a.env.TxByHash(head, hash)
	if exec.IsMissing(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	block, err := a.env.BlockAt(head, number)
	if err != nil {
		return nil, false, err
	}
	receipts, err := a.env.ReceiptsAt(head, number)
	if err != nil && !exec.IsMissing(err) {
		return nil, false, err
	}
	if uint64(len(receipts)) <= index {
		return nil, false, fmt.Errorf("confirmed transaction %s found in block %d but receipt is unavailable or transaction index %d is out of range", hash, number, index)
	}
	return format.Receipt(block.Header(), tx, index, receipts[index], a.env.Signer()), true, nil
}

// pendingTransactions answers the pending transactions of the pool. The
// reference answered only those of accounts its account manager held, which
// the module does not have.
func (a *api) pendingTransactions(ctx context.Context, _ dispatch.None) (any, error) {
	pending, err := a.pool.PoolTransactions()
	if err != nil {
		return nil, err
	}
	head := a.env.Latest(ctx)
	baseFee, chainID := a.env.SuggestedGasPrice(head), a.env.ChainID()
	result := make([]*format.Transaction, 0, len(pending))
	for _, tx := range pending {
		result = append(result, format.NewPendingTransaction(tx, baseFee, chainID))
	}
	return result, nil
}
