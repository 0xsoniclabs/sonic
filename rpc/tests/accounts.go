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
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
)

// Account is a key with the address it controls.
type Account struct {
	Key  *ecdsa.PrivateKey
	Addr common.Address
}

// NewAccount creates an account and gives it the balance in the state.
func NewAccount(t testing.TB, cs *CarmenState, balance *big.Int) Account {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	a := Account{Key: key, Addr: crypto.PubkeyToAddress(key.PublicKey)}
	if cs != nil && balance != nil {
		cs.SetBalance(a.Addr, balance)
	}
	return a
}

// Signer is the signer of the fake chain.
func Signer() types.Signer {
	return types.LatestSignerForChainID(new(big.Int).SetUint64(opera.FakeNetworkID))
}

// Transfer signs a transfer of the value to the recipient.
func (a Account) Transfer(t testing.TB, nonce uint64, to common.Address, value int64) *types.Transaction {
	t.Helper()
	return a.Call(t, nonce, to, value, nil, 100_000)
}

// Call signs a transaction with data.
func (a Account) Call(t testing.TB, nonce uint64, to common.Address, value int64, data []byte, gas uint64) *types.Transaction {
	t.Helper()
	tx, err := types.SignNewTx(a.Key, Signer(), &types.DynamicFeeTx{
		ChainID:   new(big.Int).SetUint64(opera.FakeNetworkID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(1000),
		Gas:       gas,
		To:        &to,
		Value:     big.NewInt(value),
		Data:      data,
	})
	require.NoError(t, err)
	return tx
}
