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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
)

// ReplayChain is a chain over real state for the methods that execute: a
// funded wallet, a contract that writes storage and one that returns a word,
// and block 1 with a transfer and a call of the storage-writing contract,
// published as the head.
type ReplayChain struct {
	Core      *CarmenCore
	Wallet    *Wallet
	Recipient common.Address
	Writer    common.Address // runs StorageWritingCode
	Returner  common.Address // runs ReturningCode
	Block     *evmcore.EvmBlock
	Transfer  *types.Transaction
	Call      *types.Transaction
	Log       *types.Log // emitted by the call, as the receipts record it
}

// NewReplayChain creates the chain.
func NewReplayChain(t testing.TB) *ReplayChain {
	t.Helper()
	w := NewWallet(t)
	r := &ReplayChain{
		Wallet:    w,
		Recipient: common.HexToAddress("0x1111"),
		Writer:    common.HexToAddress("0x2222"),
		Returner:  common.HexToAddress("0x3333"),
	}
	r.Core = NewCarmenCore(t, map[common.Address]Account{
		w.Address:  {Balance: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6))},
		r.Writer:   {Code: StorageWritingCode, Balance: big.NewInt(1)},
		r.Returner: {Code: ReturningCode},
	})
	r.Transfer = w.Transfer(t, 0, r.Recipient, big.NewInt(1000))
	r.Call = w.Sign(t, &types.DynamicFeeTx{
		ChainID:   big.NewInt(ChainID),
		Nonce:     1,
		To:        &r.Writer,
		Gas:       100_000,
		GasFeeCap: big.NewInt(1e12),
		GasTipCap: big.NewInt(1),
	})
	r.Log = &types.Log{Address: r.Writer, Topics: []common.Hash{common.HexToHash("0x10")}, Data: []byte{1}}
	r.Block = r.Core.AddAndPublish(types.Transactions{r.Transfer, r.Call}, nil, []*types.Log{r.Log})
	return r
}
