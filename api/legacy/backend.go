// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package legacy holds the RPC methods that the node still serves on its IPC
// endpoint only, and that the RPC module does not port: the wallet methods
// (X1), the consensus internals (X2) and the database maintenance methods of
// the debug namespace. They are kept as they were, until they are removed.
package legacy

import (
	"context"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

// PeerProgress is synchronization status of a peer
type PeerProgress struct {
	CurrentEpoch     idx.Epoch
	CurrentBlock     idx.Block
	CurrentBlockHash hash.Event
	CurrentBlockTime inter.Timestamp
	HighestBlock     idx.Block
	HighestEpoch     idx.Epoch
}

//go:generate go tool mockgen -source=backend.go -destination=backend_mock.go -package=legacy

// Backend is what the legacy methods ask of the node.
type Backend interface {
	Progress() PeerProgress
	AccountManager() *accounts.Manager
	ExtRPCEnabled() bool
	RPCTxFeeCap() float64     // global tx fee cap for all transaction related APIs
	UnprotectedAllowed() bool // allows only for EIP155 transactions.

	// SuggestGasTipCap and EstimateGas are answered by the RPC module, whose
	// gas price oracle and EVM replace the node's own.
	SuggestGasTipCap(ctx context.Context, certainty uint64) *big.Int
	EstimateGas(ctx context.Context, args TransactionArgs) (hexutil.Uint64, error)
	MinGasPrice() *big.Int

	// Transaction pool
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetPoolTransactions() (types.Transactions, error)
	GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error)

	ChainConfig(blockHeight idx.Block) *params.ChainConfig
	ChainID() *big.Int
	CurrentBlock() *evmcore.EvmBlock

	// Lachesis DAG API
	GetEventPayload(ctx context.Context, shortEventID string) (*inter.EventPayload, error)
	GetEvent(ctx context.Context, shortEventID string) (*inter.Event, error)
	GetHeads(ctx context.Context, epoch rpc.BlockNumber) (hash.Events, error)
	CurrentEpoch(ctx context.Context) idx.Epoch
	SealedEpochTiming(ctx context.Context) (start inter.Timestamp, end inter.Timestamp)

	// Lachesis aBFT API
	GetEpochBlockState(ctx context.Context, epoch rpc.BlockNumber) (*iblockproc.BlockState, *iblockproc.EpochState, error)
	GetDowntime(ctx context.Context, vid idx.ValidatorID) (idx.Block, inter.Timestamp, error)
	GetUptime(ctx context.Context, vid idx.ValidatorID) (*big.Int, error)
	GetOriginatedFee(ctx context.Context, vid idx.ValidatorID) (*big.Int, error)
}
