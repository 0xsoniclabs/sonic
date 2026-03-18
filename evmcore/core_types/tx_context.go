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

package core_types

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:generate mockgen -source=tx_context.go -destination=tx_context_mock.go -package=core_types

// ProcessedTransaction represents a transaction that was considered for
// inclusion in a block by the state processor. It contains the transaction
// itself and the receipt either confirming its execution, or nil if the
// transaction was skipped.
type ProcessedTransaction struct {
	Transaction *types.Transaction
	Receipt     *types.Receipt
}

// ProcessedBundle summarizes the result of a processed bundle.
type ProcessedBundle struct {
	ExecutionPlanHash common.Hash
	Position          uint32 // < position in the block transaction list
	Count             uint32 // < number of transactions from this bundle in the block transaction list
}
type TransactionRunner interface {
	RunRegularTransaction(ctxt *RunContext, tx *types.Transaction, txIndex int) (ProcessedTransaction, TransactionResult)
	RunSponsoredTransaction(ctxt *RunContext, tx *types.Transaction, txIndex int) ([]ProcessedTransaction, TransactionResult)
	RunTransactionBundle(ctxt *RunContext, tx *types.Transaction, legacyTxIndex int, trueTxIndex int) ([]ProcessedTransaction, *ProcessedBundle, TransactionResult)
}

// RunContext bundles the parameters required for processing transactions in a
// block. It is used as input to the runTransactions helper function and passed
// along the processing layers to make the parameters available where needed.
type RunContext struct {
	Signer      types.Signer
	BaseFee     *big.Int
	StateDB     state.StateDB
	GasPool     *core.GasPool
	BlockNumber *big.Int
	UsedGas     *uint64
	OnNewLog    func(*types.Log)
	Upgrades    opera.Upgrades
	Runner      TransactionRunner
}

// NewRunContext creates a new RunContext instance bundling the given parameters
// required for processing transactions in a block. In productive code this
// function should be used instead of directly creating a RunContext instance to
// ensure that all required parameters are provided.
func NewRunContext(
	signer types.Signer,
	baseFee *big.Int,
	statedb state.StateDB,
	gasPool *core.GasPool,
	blockNumber *big.Int,
	usedGas *uint64,
	onNewLog func(*types.Log),
	upgrades opera.Upgrades,
	runner TransactionRunner,
) *RunContext {
	return &RunContext{
		Signer:      signer,
		BaseFee:     baseFee,
		StateDB:     statedb,
		GasPool:     gasPool,
		BlockNumber: blockNumber,
		UsedGas:     usedGas,
		OnNewLog:    onNewLog,
		Upgrades:    upgrades,
		Runner:      runner,
	}
}
