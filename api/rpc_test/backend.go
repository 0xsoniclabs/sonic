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
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=rpctest

type backend struct {
	ethapi.Backend

	chainId      uint64
	rules        opera.Rules
	state        *testState
	pool         txPool
	blockHistory []TestBlock
}

type TestAccount struct {
	Nonce   uint64
	Balance *big.Int
	Code    []byte
	Store   map[common.Hash]common.Hash
}

type TestBlock struct {
	Number     uint64
	Hash       common.Hash
	ParentHash common.Hash
}

type backendBuilder struct {
	be backend
}

type txPool interface {
	AddLocal(*types.Transaction) error
}

func NewBackendBuilder(t *testing.T) backendBuilder {
	return backendBuilder{
		be: backend{
			chainId:      opera.FakeNetworkID,
			rules:        opera.FakeNetRules(opera.GetBrioUpgrades()),
			state:        newTestState(t),
			blockHistory: DefaultBlockHistory(),
		},
	}
}

func (b backendBuilder) WithChainId(chainId uint64) backendBuilder {
	b.be.chainId = chainId
	return b
}

func (b backendBuilder) WithBlockHistory(blocks []TestBlock) backendBuilder {
	b.be.blockHistory = blocks
	return b
}

func (b backendBuilder) WithPool(pool txPool) backendBuilder {
	b.be.pool = pool
	return b
}

func (b backendBuilder) WithAccount(addr common.Address, account TestAccount) backendBuilder {
	b.be.state.setAccount(addr, account)
	return b
}

func (b backendBuilder) WithUpgrade(upgrades opera.Upgrades) backendBuilder {
	b.be.rules = opera.FakeNetRules(upgrades)
	return b
}

func (b backendBuilder) Build() *backend {
	return &b.be
}

func DefaultBlockHistory() []TestBlock {
	return []TestBlock{
		{
			Number: 1,
			Hash:   common.HexToHash("0x1"),
		},
	}
}

func (b *backend) ChainID() *big.Int {
	return big.NewInt(int64(b.chainId))
}

func (b *backend) CurrentBlock() *evmcore.EvmBlock {
	lastblock := b.blockHistory[len(b.blockHistory)-1]
	return &evmcore.EvmBlock{EvmHeader: *ToEvmHeader(lastblock)}
}

func (b *backend) GetEVM(
	ctx context.Context,
	state vm.StateDB,
	header *evmcore.EvmHeader,
	vmConfig *vm.Config,
	blockContext *vm.BlockContext,
) (*vm.EVM, func() error, error) {
	if blockContext == nil {
		blkctx := ethapi.GetBlockContext(ctx, b, header)
		blockContext = &blkctx
	}
	chainConfig := b.ChainConfig(idx.Block(header.Number.Int64()))
	return vm.NewEVM(*blockContext, state, chainConfig, *vmConfig), func() error { return nil }, nil
}

func (b *backend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error) {

	if number == rpc.LatestBlockNumber ||
		number == rpc.PendingBlockNumber ||
		number == rpc.FinalizedBlockNumber ||
		number == rpc.SafeBlockNumber {

		block := b.blockHistory[len(b.blockHistory)-1]
		return ToEvmHeader(block), nil
	}
	n := number.Int64()
	for _, block := range b.blockHistory {
		if int64(block.Number) == n {
			return ToEvmHeader(block), nil
		}
	}
	return nil, errors.New("block header not found")
}

func (b *backend) HeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeader, error) {
	for _, block := range b.blockHistory {
		if block.Hash == hash {
			return ToEvmHeader(block), nil
		}
	}
	return nil, errors.New("block header not found")
}

func (b *backend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmBlock, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return &evmcore.EvmBlock{
		EvmHeader: *header,
	}, nil
}

func (b *backend) BlockByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmBlock, error) {
	header, err := b.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return &evmcore.EvmBlock{
		EvmHeader: *header,
	}, nil
}

func (b *backend) MaxGasLimit() uint64 {
	return b.rules.Economy.Gas.MaxEventGas
}

func (b *backend) MinGasPrice() *big.Int {
	return big.NewInt(1)
}

func (b *backend) RPCGasCap() uint64 {
	return 50_000_000
}

func (b *backend) RPCTxFeeCap() float64 {
	return 1.0
}

func (b *backend) RPCEVMTimeout() time.Duration {
	return 5 * time.Second
}

func (b *backend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	if b.pool == nil {
		return errors.New("tx pool not initialized")
	}
	return b.pool.AddLocal(signedTx)
}

func (b *backend) StateAndBlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (state.StateDB, *evmcore.EvmBlock, error) {

	var (
		block *evmcore.EvmBlock
		err   error
	)
	if blockNrOrHash.BlockNumber != nil {
		block, err = b.BlockByNumber(ctx, *blockNrOrHash.BlockNumber)
		if err != nil {
			return nil, nil, err
		}
	} else if blockNrOrHash.BlockHash != nil {
		block, err = b.BlockByHash(ctx, *blockNrOrHash.BlockHash)
		if err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, errors.New("invalid block number or hash")
	}

	if block.Number == nil {
		return nil, nil, errors.New("block number is nil")
	}

	return b.state.Copy(), block, nil
}

func (b *backend) GetNetworkRules(ctx context.Context, blockHeight idx.Block) (*opera.Rules, error) {
	return &b.rules, nil
}

func (b *backend) ChainConfig(blockHeight idx.Block) *params.ChainConfig {
	heights := opera.MakeUpgradeHeight(b.rules.Upgrades, 0)
	return opera.CreateTransientEvmChainConfig(b.chainId, []opera.UpgradeHeight{heights}, blockHeight)
}

func (b *backend) GetSigner() types.Signer {
	return types.LatestSignerForChainID(b.ChainID())
}
