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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Send checks a signed transaction as the legacy send path does, its fee
// within the cap and its replay protection, and hands it to the pool, whose
// verdict it answers as it is.
func (e *Environment) Send(ctx context.Context, transaction *types.Transaction) error {
	if err := checkFee(transaction.GasPrice(), transaction.Gas(), e.config.RPCTxFeeCap); err != nil {
		return err
	}
	if !e.config.AllowUnprotectedTxs && !transaction.Protected() {
		return errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := e.reader.SendTransaction(ctx, transaction); err != nil {
		return err
	}
	sender, err := types.Sender(types.LatestSignerForChainID(e.chainID), transaction)
	if err != nil {
		return err
	}
	if transaction.To() == nil {
		log.Debug("Submitted contract creation", "hash", transaction.Hash().Hex(), "from", sender, "nonce", transaction.Nonce(),
			"contract", crypto.CreateAddress(sender, transaction.Nonce()).Hex(), "value", transaction.Value())
	} else {
		log.Debug("Submitted transaction", "hash", transaction.Hash().Hex(), "from", sender, "nonce", transaction.Nonce(),
			"recipient", transaction.To(), "value", transaction.Value())
	}
	return nil
}

// checkFee refuses a fee, in FTM, above the cap unless the cap is zero.
func checkFee(gasPrice *big.Int, gas uint64, feeCap float64) error {
	if feeCap == 0 {
		return nil
	}
	fee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))
	inFTM, _ := new(big.Float).Quo(new(big.Float).SetInt(fee), new(big.Float).SetInt(big.NewInt(params.Ether))).Float64()
	if inFTM > feeCap {
		return fmt.Errorf("tx fee (%.2f FTM) exceeds the configured cap (%.2f FTM)", inFTM, feeCap)
	}
	return nil
}
