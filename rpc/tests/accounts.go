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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer is a key that signs transactions on the chain of a FakeCore.
type Signer struct {
	Address common.Address
	key     *ecdsa.PrivateKey
}

// NewSigner returns a signer with a new key.
func NewSigner(t testing.TB) *Signer {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate a key: %v", err)
	}
	return &Signer{Address: crypto.PubkeyToAddress(key.PublicKey), key: key}
}

// Sign signs a transaction for the chain of a FakeCore.
func (signer *Signer) Sign(t testing.TB, data types.TxData) *types.Transaction {
	t.Helper()
	signed, err := types.SignNewTx(signer.key, types.LatestSignerForChainID(big.NewInt(FakeChainID)), data)
	if err != nil {
		t.Fatalf("failed to sign a transaction: %v", err)
	}
	return signed
}

// Encode returns the binary encoding of a transaction, as a client sends it.
func Encode(t testing.TB, transaction *types.Transaction) hexutil.Bytes {
	t.Helper()
	encoded, err := transaction.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to encode a transaction: %v", err)
	}
	return encoded
}
