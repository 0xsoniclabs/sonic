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

package rpctest

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func (a *Account) Address() *common.Address {
	addr := crypto.PubkeyToAddress(a.PrivateKey.PublicKey)
	return &addr
}

func ToHexUint64(i uint64) *hexutil.Uint64 {
	hu := hexutil.Uint64(i)
	return &hu
}

func ToHexUint(i uint) *hexutil.Uint {
	hu := hexutil.Uint(i)
	return &hu
}

func ToHexBig(i big.Int) *hexutil.Big {
	return (*hexutil.Big)(&i)
}

func ToHexBigInt(i int64) *hexutil.Big {
	hu := hexutil.Big(*big.NewInt(i))
	return &hu
}

func ToHexBytes(b []byte) *hexutil.Bytes {
	hb := hexutil.Bytes(b)
	return &hb
}

func ToEvmHeader(block TestBlock) *evmcore.EvmHeader {
	return &evmcore.EvmHeader{
		Number:     big.NewInt(int64(block.Number)),
		Hash:       block.Hash,
		ParentHash: block.ParentHash,
	}
}
