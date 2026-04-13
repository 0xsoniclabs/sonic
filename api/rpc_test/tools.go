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
	"github.com/ethereum/go-ethereum/rpc"
)

// This file contains utility functions and types for the RPC tests.

// Wallet is a simple wrapper around an ECDSA private key,
// providing a method to get the corresponding address.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
}

// NewWallet creates a new wallet with a random private key.
func NewWallet() (*Wallet, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return &Wallet{
		PrivateKey: key,
	}, nil
}

// Address returns the public address corresponding to the private key.
func (a *Wallet) Address() *common.Address {
	addr := crypto.PubkeyToAddress(a.PrivateKey.PublicKey)
	return &addr
}

// ToHexUint64 converts a uint64 to a hexutil.Uint64.
func ToHexUint64(i uint64) *hexutil.Uint64 {
	hu := hexutil.Uint64(i)
	return &hu
}

// ToHexUint converts a uint to a hexutil.Uint.
func ToHexUint(i uint) *hexutil.Uint {
	hu := hexutil.Uint(i)
	return &hu
}

// ToHexBigInt converts a big.Int to a hexutil.Big.
func ToHexBigInt(i *big.Int) *hexutil.Big {
	if i == nil {
		return nil
	}
	hu := hexutil.Big(*i)
	return &hu
}

// ToHexBytes converts a byte slice to a hexutil.Bytes.
func ToHexBytes(b []byte) *hexutil.Bytes {
	hb := hexutil.Bytes(b)
	return &hb
}

// ToEvmHeader converts a Block to an evmcore.EvmHeader.
func ToEvmHeader(block Block) *evmcore.EvmHeader {
	return &evmcore.EvmHeader{
		Number:     big.NewInt(int64(block.Number)),
		Hash:       block.Hash,
		ParentHash: block.ParentHash,
		PrevRandao: block.PrevRandao,
	}
}

// ToBlockNum converts a uint64 to a *rpc.BlockNumber.
func ToBlockNum(i uint64) *rpc.BlockNumber {
	bn := rpc.BlockNumber(i)
	return &bn
}
