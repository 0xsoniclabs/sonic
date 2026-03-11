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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Account struct {
	PrivateKey *ecdsa.PrivateKey
}

func NewAccount() *Account {
	key, _ := crypto.GenerateKey()
	return &Account{
		PrivateKey: key,
	}
}

func (a *Account) Address() common.Address {
	return crypto.PubkeyToAddress(a.PrivateKey.PublicKey)
}

func NewAccounts(num int) []*Account {
	res := make([]*Account, num)
	for i := range res {
		res[i] = NewAccount()
	}
	return res
}

func (a *Account) Into() bundle.Account {
	return bundle.Account{PrivateKey: a.PrivateKey}
}

func From(account *bundle.Account) *Account {
	return &Account{
		PrivateKey: account.PrivateKey,
	}
}
