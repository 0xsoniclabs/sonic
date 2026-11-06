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

package gossip

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
)

// The methods below complete the LedgerStore implementation of Store by
// forwarding to its EVM store and its EVM state reader.

var _ LedgerStore = (*Store)(nil)

func (s *Store) Header(verificationHash common.Hash, number uint64) *evmcore.EvmHeader {
	return (&EvmStateReader{store: s}).Header(verificationHash, number)
}

func (s *Store) GetLiveStateDb(stateRoot hash.Hash) (state.StateDB, error) {
	return s.evm.GetLiveStateDb(stateRoot)
}

func (s *Store) SetTx(txid common.Hash, tx *types.Transaction) {
	s.evm.SetTx(txid, tx)
}

func (s *Store) SetTxPosition(txid common.Hash, position evmstore.TxPosition) {
	s.evm.SetTxPosition(txid, position)
}

func (s *Store) SetReceipts(n idx.Block, receipts types.Receipts) {
	s.evm.SetReceipts(n, receipts)
}

func (s *Store) IndexLogs(logs ...*types.Log) {
	s.evm.IndexLogs(logs...)
}

func (s *Store) SetCachedEvmBlock(n idx.Block, b *evmcore.EvmBlock) {
	s.evm.SetCachedEvmBlock(n, b)
}
