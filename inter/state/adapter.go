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

package state

import (
	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source adapter.go -destination adapter_mock.go -package state

const (
	// InvalidSnapshotID is a special snapshot ID that can be used to trigger an
	// invalid revert in the StateDB. This can be used to keep track of invalid
	// reverts, such that they can be handled when checking for errors in the
	// StateDB after the block processing.
	InvalidSnapshotID = int(-1)
)

// StagedBlock is a block that has been applied to the live state but is not yet
// part of the archive. It lets a caller execute several blocks ahead of a decision
// it has not taken yet -- as a consensus certifying blocks before finalizing them
// must -- and then keep or discard each of them.
//
// Exactly one of Commit or Rollback must be called. Both invalidate the block, and
// a second call on it reports an error rather than acting twice.
//
// Ordering is enforced: Commit only accepts the oldest staged block, because the
// archive is append-only and must receive blocks in order; Rollback only accepts
// the newest, because taking a block back rests on every later block already being
// gone.
type StagedBlock interface {
	// StateHash returns the root of the live state as of this block. It stays the
	// root this block produced even once later blocks have been staged on top.
	StateHash() common.Hash

	// Commit promotes this block into the archive. It returns as soon as the write
	// is under way; use Wait to await its completion.
	Commit() error

	// Wait blocks until the archive write triggered by Commit has completed and
	// reports its outcome. It must be called after Commit, and returns immediately
	// when no archive is maintained.
	Wait() error

	// Rollback reverts this block from the live state, restoring the root its
	// predecessor left behind.
	Rollback() error
}

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

	// EndBlock collects the changes made since BeginBlock into a block and applies
	// them to the live state, returning the block as staged: its content is live,
	// but it is not yet part of the archive. The caller decides its fate through
	// the returned StagedBlock; until it does, the block can still be taken back.
	EndBlock(number uint64) (StagedBlock, error)

	EndTransaction()
	Release()
	InterTxSnapshot() int
	RevertToInterTxSnapshot(id int)

	// -- Sonic Extensions --

	// AddProcessedBundle marks the given execution plan as being processed.
	// Marks are subject to inter-Tx-snapshots. Thus, rolling back the state
	// to a previous snapshot will also undo the marking plans.
	AddProcessedBundle(execPlanHash common.Hash, positionInBlock bundle.PositionInBlock)

	// HasBeenProcessed checks whether the given execution plan has been
	// processed in the past. The processing may have happened as part of the
	// current block (by being marked as processed using [AddProcessedBundles])
	// or in a previous block, tracked in the state and not subject to rollbacks.
	HasBundleRecentlyBeenProcessed(execPlanHash common.Hash) bool
}
