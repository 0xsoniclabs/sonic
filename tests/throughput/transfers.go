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

// Package throughput provides the transfer loads of the throughput benchmarks.
package throughput

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/erc20"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const (
	Senders  = 2000
	GasPrice = 100e9
)

type Load struct {
	Name           string
	TxsPerTransfer int
	// setup returns the transactions the first sender issues before the
	// measurement, starting with the deployment of the token.
	setup func(token common.Address, senders []common.Address) []*types.LegacyTx
	// transfer returns the transactions moving one unit from sender to recipient.
	transfer func(token, sender, recipient common.Address) []*types.LegacyTx
}

var Loads = []Load{
	{
		Name:           "native",
		TxsPerTransfer: 1,
		setup:          func(common.Address, []common.Address) []*types.LegacyTx { return nil },
		transfer: func(_, _, recipient common.Address) []*types.LegacyTx {
			return []*types.LegacyTx{{To: &recipient, Value: big.NewInt(1), Gas: 21_000}}
		},
	},
	{
		Name:           "erc20",
		TxsPerTransfer: 2,
		setup: func(token common.Address, senders []common.Address) []*types.LegacyTx {
			abi, err := erc20.ERC20MetaData.GetAbi()
			if err != nil {
				panic(err)
			}
			args, err := abi.Pack("", "Token", "TKN")
			if err != nil {
				panic(err)
			}
			res := []*types.LegacyTx{{Gas: 10_000_000, Data: append(common.FromHex(erc20.ERC20MetaData.Bin), args...)}}
			for chunk := range sliceChunks(senders, 500) {
				res = append(res, &types.LegacyTx{To: &token, Gas: 20_000_000, Data: erc20Call("mintForAll", chunk, big.NewInt(1e18))})
			}
			return res
		},
		// The sender approves itself, so that its nonce orders the allowance before the withdrawal.
		transfer: func(token, sender, recipient common.Address) []*types.LegacyTx {
			return []*types.LegacyTx{
				{To: &token, Gas: 70_000, Data: erc20Call("approve", sender, big.NewInt(1))},
				{To: &token, Gas: 70_000, Data: erc20Call("transferFrom", sender, recipient, big.NewInt(1))},
			}
		},
	},
}

func erc20Call(method string, args ...any) []byte {
	abi, err := erc20.ERC20MetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	data, err := abi.Pack(method, args...)
	if err != nil {
		panic(err)
	}
	return data
}

func sliceChunks[T any](s []T, size int) func(func([]T) bool) {
	return func(yield func([]T) bool) {
		for len(s) > 0 {
			n := min(size, len(s))
			if !yield(s[:n]) {
				return
			}
			s = s[n:]
		}
	}
}

// NewKeys returns the keys of Senders new accounts.
func NewKeys() []*ecdsa.PrivateKey {
	res := make([]*ecdsa.PrivateKey, Senders)
	for i := range res {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		res[i] = key
	}
	return res
}

// Issuer signs the transactions of the senders with the given keys, keeping
// track of their nonces.
type Issuer struct {
	Signer  types.Signer
	Senders []common.Address
	keys    []*ecdsa.PrivateKey
	nonces  []uint64
}

func NewIssuer(chainID *big.Int, keys []*ecdsa.PrivateKey) *Issuer {
	senders := make([]common.Address, len(keys))
	for i, key := range keys {
		senders[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	return &Issuer{
		Signer:  types.LatestSignerForChainID(chainID),
		Senders: senders,
		keys:    keys,
		nonces:  make([]uint64, len(keys)),
	}
}

// Nonce returns the nonce of the next transaction of the given sender.
func (s *Issuer) Nonce(sender int) uint64 {
	return s.nonces[sender]
}

func (s *Issuer) sign(tb testing.TB, sender int, txs []*types.LegacyTx) []*types.Transaction {
	res := make([]*types.Transaction, len(txs))
	for i, tx := range txs {
		tx.Nonce = s.nonces[sender]
		tx.GasPrice = big.NewInt(GasPrice)
		s.nonces[sender]++
		signed, err := types.SignNewTx(s.keys[sender], s.Signer, tx)
		require.NoError(tb, err)
		res[i] = signed
	}
	return res
}

func (s *Issuer) Token() common.Address {
	return crypto.CreateAddress(s.Senders[0], 0)
}

func (s *Issuer) Setup(tb testing.TB, load Load) []*types.Transaction {
	return s.sign(tb, 0, load.setup(s.Token(), s.Senders))
}

// Transfers returns the transactions of one transfer of the given sender to the next sender.
func (s *Issuer) Transfers(tb testing.TB, load Load, sender int) []*types.Transaction {
	recipient := s.Senders[(sender+1)%len(s.Senders)]
	return s.sign(tb, sender, load.transfer(s.Token(), s.Senders[sender], recipient))
}
