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

// Package legacy serves the legacy methods the RPC module does not port
// (X3): eth_accounts and the signing and sending methods of the account
// manager, personal_*, the chain database methods of debug, dag_*, abft_*
// and the epoch methods of eth, with net_listening and net_peerCount. The
// node's handler serves them, on IPC only, until they are removed. The code
// is cut mechanically out of api/ethapi; the wallet methods take the tip and
// the gas of what they send from the module (Defaults).
package legacy

import (
	"context"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
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

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
//
//go:generate go tool mockgen -source=backend.go -destination=backend_mock.go -package=legacy
type Backend interface {
	// General Ethereum API
	Progress() PeerProgress

	AccountManager() *accounts.Manager
	ExtRPCEnabled() bool
	RPCTxFeeCap() float64     // global tx fee cap for all transaction related APIs
	UnprotectedAllowed() bool // allows only for EIP155 transactions.

	// Blockchain API

	MinGasPrice() *big.Int

	// Transaction pool API
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
