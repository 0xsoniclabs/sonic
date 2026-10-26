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

package exec

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Submit checks a signed transaction against the fee cap and the replay
// protection the node requires, and hands it to the pool, whose verdict it
// returns verbatim.
func (e *Env) Submit(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	if err := CheckTxFee(tx.GasPrice(), tx.Gas(), e.cfg.TxFeeCap); err != nil {
		return common.Hash{}, err
	}
	if !e.cfg.AllowUnprotectedTxs && !tx.Protected() {
		return common.Hash{}, errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := e.core.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	from, err := types.Sender(e.Signer(), tx)
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

// CheckTxFee refuses a fee above the cap, in FTM; a zero cap is no cap.
func CheckTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	if cap == 0 {
		return nil
	}
	fee := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))), new(big.Float).SetInt(big.NewInt(params.Ether)))
	feeFloat, _ := fee.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f FTM) exceeds the configured cap (%.2f FTM)", feeFloat, cap)
	}
	return nil
}
