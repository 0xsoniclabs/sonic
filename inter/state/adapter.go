// Copyright 2025 Sonic Operations Ltd
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

package state

import (
	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source adapter.go -destination adapter_mock.go -package state

type StateDB interface {
	vm.StateDB

	Error() error
	GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log
	SetTxContext(thash common.Hash, ti int)
	TxIndex() int
	GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error)
	SetBalance(addr common.Address, amount *uint256.Int)
	SetStorage(addr common.Address, storage map[common.Hash]common.Hash)
	Copy() StateDB
	GetStateHash() common.Hash

	BeginBlock(number uint64)
	EndBlock(number uint64)
	EndTransaction()
	Release()
}
