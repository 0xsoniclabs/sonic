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

var errUnprotected = errors.New("only replay-protected (EIP-155) transactions allowed over RPC")

// Send submits a signed transaction to the pool, as the reference's
// SubmitTransaction did: a fee within RPCTxFeeCap, replay protection unless
// unprotected transactions are allowed, then the pool's verdict verbatim
// (V7). The one path to the pool, which the sonic namespace's bundles take
// too.
func (environment *Environment) Send(ctx context.Context, transaction *types.Transaction) (common.Hash, error) {
	if err := checkFee(transaction.GasPrice(), transaction.Gas(), environment.config.RPCTxFeeCap); err != nil {
		return common.Hash{}, err
	}
	if !environment.config.AllowUnprotectedTxs && !transaction.Protected() {
		return common.Hash{}, errUnprotected
	}
	if err := environment.nodeCore.SendTransaction(ctx, transaction); err != nil {
		return common.Hash{}, err
	}
	sender, err := types.Sender(environment.Signer(), transaction)
	if err != nil {
		return common.Hash{}, err
	}
	if transaction.To() == nil {
		created := crypto.CreateAddress(sender, transaction.Nonce())
		log.Debug("Submitted contract creation", "hash", transaction.Hash().Hex(), "from", sender, "nonce", transaction.Nonce(), "contract", created.Hex(), "value", transaction.Value())
	} else {
		log.Debug("Submitted transaction", "hash", transaction.Hash().Hex(), "from", sender, "nonce", transaction.Nonce(), "recipient", transaction.To(), "value", transaction.Value())
	}
	return transaction.Hash(), nil
}

// checkFee refuses a transaction whose fee, at its gas price and gas,
// exceeds the cap in FTM; a cap of zero refuses none.
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
