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
package tests

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Wallet is an account with its key, signing on the fake chain id.
type Wallet struct {
	Key     *ecdsa.PrivateKey
	Address common.Address
}

// NewWallet creates an account with a fresh key.
func NewWallet(t testing.TB) *Wallet {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generating a key: %v", err)
	}
	return &Wallet{Key: key, Address: crypto.PubkeyToAddress(key.PublicKey)}
}

// Signer is the signer of the fake chain.
func Signer() types.Signer {
	return types.LatestSignerForChainID(big.NewInt(ChainID))
}

// Sign signs a transaction on the fake chain.
func (w *Wallet) Sign(t testing.TB, data types.TxData) *types.Transaction {
	t.Helper()
	tx, err := types.SignNewTx(w.Key, Signer(), data)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}
	return tx
}

// Transfer signs a dynamic-fee transfer of value to a recipient.
func (w *Wallet) Transfer(t testing.TB, nonce uint64, to common.Address, value *big.Int) *types.Transaction {
	t.Helper()
	return w.Sign(t, &types.DynamicFeeTx{
		ChainID:   big.NewInt(ChainID),
		Nonce:     nonce,
		To:        &to,
		Value:     value,
		Gas:       21000,
		GasFeeCap: big.NewInt(1e12),
		GasTipCap: big.NewInt(1),
	})
}

// Encode is the binary encoding of a transaction, as eth_sendRawTransaction
// takes it.
func Encode(t testing.TB, tx *types.Transaction) []byte {
	t.Helper()
	enc, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	return enc
}
