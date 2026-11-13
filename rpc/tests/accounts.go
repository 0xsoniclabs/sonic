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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Account is a key that signs transactions on the chain of a FakeCore.
type Account struct {
	key *ecdsa.PrivateKey
}

// NewAccount creates an account with a new key.
func NewAccount() *Account {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return &Account{key: key}
}

// Address is the address of the account.
func (a *Account) Address() common.Address {
	return crypto.PubkeyToAddress(a.key.PublicKey)
}

// Sign signs a transaction for the chain of a FakeCore.
func (a *Account) Sign(transaction types.TxData) *types.Transaction {
	signer := types.LatestSignerForChainID(big.NewInt(ChainID))
	return types.MustSignNewTx(a.key, signer, transaction)
}
