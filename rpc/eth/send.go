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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

func registerSend(d *dispatch.Dispatcher, a *API, m func(string) string) {
	dispatch.Add1(d, m("sendRawTransaction"), a.sendRawTransaction, "input")
	dispatch.Add2(d, m("sendRawTransactionSync"), a.sendRawTransactionSync, "input", "timeoutMs?")
	dispatch.Add0(d, m("pendingTransactions"), a.pendingTransactions)
}

// SubmitTransaction submits a transaction to the pool and logs a message. The
// fee is checked against the configured cap, and unprotected transactions are
// refused unless the node allows them. The verdict of the pool is returned
// verbatim.
func SubmitTransaction(ctx context.Context, c core.Core, cfg dispatch.Config, tx *types.Transaction) (common.Hash, error) {
	// If the transaction fee cap is already specified, ensure the fee of the
	// given transaction is _reasonable_.
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), cfg.TxFeeCap); err != nil {
		return common.Hash{}, err
	}
	if !cfg.UnprotectedAllowed && !tx.Protected() {
		// Ensure only eip155 signed transactions are submitted if EIP155Required is set.
		return common.Hash{}, errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := c.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	// Print a log with full tx details for manual investigations and interventions.
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(c.ChainInfo().ChainID))
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if tx.To() == nil {
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Debug("Submitted contract creation", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "contract", addr.Hex(), "value", tx.Value())
	} else {
		log.Debug("Submitted transaction", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "recipient", tx.To(), "value", tx.Value())
	}
	return tx.Hash(), nil
}

// checkTxFee checks whether the fee of the given transaction is _reasonable_
// (under the cap).
func checkTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	feeEth := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))), new(big.Float).SetInt(big.NewInt(params.Ether)))
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f FTM) exceeds the configured cap (%.2f FTM)", feeFloat, cap)
	}
	return nil
}

// sendRawTransaction adds the signed transaction to the pool. The sender is
// responsible for signing the transaction and using the correct nonce.
func (a *API) sendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encodedTx); err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, a.core, a.cfg, tx)
}

// sendRawTransactionSync submits a signed transaction and waits synchronously
// for it to be included in a block, returning the receipt. Implements EIP-7966.
//
// Parameters:
//   - encodedTx: hex-encoded RLP signed transaction
//   - timeoutMs: optional max wait time for tx processing in milliseconds
//
// Returns the transaction receipt on success, in the same format as
// eth_getTransactionReceipt. On failure, returns a typed JSON-RPC error:
//   - Code 4: tx was added to the pool but not processed within the timeout (data: tx hash)
//   - Code 5: tx was NOT added to the pool, rejected for a reason other than a nonce gap (data: tx hash)
//   - Code 6: tx was NOT added to the pool, nonce gap detected (data: expected nonce)
//
// The wait follows the head of the module instead of polling the node.
func (a *API) sendRawTransactionSync(ctx context.Context, encodedTx hexutil.Bytes, timeoutMs *hexutil.Uint64) (map[string]any, error) {
	// 1. Decode the transaction.
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encodedTx); err != nil {
		return nil, err
	}
	txHash := tx.Hash()

	// 2. Determine timeout.
	timeout := a.cfg.TxSyncDefaultTimeout
	if timeoutMs != nil {
		if *timeoutMs == 0 {
			return nil, errors.New("timeout must be greater than zero")
		}
		// Convert without overflowing time.Duration (int64 nanoseconds).
		requested := time.Duration(math.MaxInt64)
		if *timeoutMs < hexutil.Uint64(requested/time.Millisecond) {
			requested = time.Duration(*timeoutMs) * time.Millisecond
		}
		timeout = requested
		if maxTimeout := a.cfg.TxSyncMaxTimeout; maxTimeout > 0 && timeout > maxTimeout {
			timeout = maxTimeout
		}
	}

	// 3. Nonce gap check (EIP-7966 Code 6): reject before pool submission.
	signer := types.LatestSignerForChainID(a.bigChainID())
	from, err := types.Sender(signer, tx)
	if err != nil {
		return nil, err
	}
	expectedNonce := a.core.PoolNonce(from)
	if tx.Nonce() > expectedNonce {
		return nil, errSendRawSyncNonceGap(expectedNonce)
	}

	// 4. Follow the head from here on, so that no block is missed between the
	// submission and the wait.
	heads := a.chain.SubscribeHeads(headQueueSize)
	defer func() { heads.Close() }()

	// 5. Submit transaction (validates fee, EIP-155, adds to pool).
	if _, err := SubmitTransaction(ctx, a.core, a.cfg, tx); err != nil {
		return nil, errSendRawSyncRejected(txHash, err)
	}

	// 6. Wait for confirmation with a deadline.
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// latest is the newest block the transaction may be in: a block is served
	// once it is archived, which the head subscription announces.
	latest := a.chain.Latest()
	for {
		// Check for confirmation: the fast path, which also handles an
		// already confirmed transaction, and the check after each new head.
		confirmed, blockNumber, index, err := a.core.TxByHash(txHash)
		if err != nil && !errors.Is(err, core.ErrNotFound) {
			return nil, err
		}
		if err == nil && confirmed != nil && blockNumber <= latest {
			return a.confirmedReceipt(txHash, confirmed, blockNumber, index)
		}

		select {
		case <-deadlineCtx.Done():
			return nil, errSendRawSyncTimeout(txHash)
		case ev, ok := <-heads.Events():
			if !ok {
				// The consumer fell behind and was dropped: take a new one.
				heads.Close()
				heads = a.chain.SubscribeHeads(headQueueSize)
				latest = max(latest, a.chain.Latest())
				continue
			}
			if ev.Block != nil {
				latest = max(latest, ev.Block.NumberU64())
			}
		}
	}
}

// headQueueSize is the queue of a consumer of the head waiting for a transaction.
const headQueueSize = 64

// confirmedReceipt builds the receipt of a transaction found in a block.
func (a *API) confirmedReceipt(txHash common.Hash, tx *types.Transaction, blockNumber, index uint64) (map[string]any, error) {
	header, err := a.core.Header(blockNumber)
	if errors.Is(err, core.ErrNotFound) {
		return nil, fmt.Errorf("confirmed transaction %s found in block %d but the block is unavailable", txHash, blockNumber)
	}
	if err != nil {
		return nil, err
	}
	receipts, err := a.receiptsAt(blockNumber)
	if err != nil {
		return nil, err
	}
	if receipts == nil || receipts.Len() <= int(index) {
		return nil, fmt.Errorf("confirmed transaction %s found in block %d but receipt is unavailable or transaction index %d is out of range", txHash, blockNumber, index)
	}
	return formatTxReceipt(a.core.ChainInfo().ChainID, header, tx, index, receipts[index]), nil
}

// pendingTransactions returns the transactions that are pending in the pool.
func (a *API) pendingTransactions(context.Context) ([]*RPCTransaction, error) {
	pending, err := a.core.PoolTransactions()
	if err != nil {
		return nil, err
	}
	head := a.chain.Head()
	transactions := make([]*RPCTransaction, 0, len(pending))
	minGasPrice := a.minGasPrice(head)
	chainID := a.bigChainID()
	for _, tx := range pending {
		transactions = append(transactions, newRPCPendingTransaction(tx, minGasPrice, chainID))
	}
	return transactions, nil
}
