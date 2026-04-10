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

// Package rpctest provides tools for testing RPC handlers.
//
// The core component is a fake backend that implements the ethapi.Backend interface,
// allowing developers to create controlled environments for testing various RPC
// methods without needing a full node. NewBackendBuilder is the main entry point
// for constructing a fake backend, and therefore using the tools in this package.
//
// The features implemented in the fake backend are introduced on demand as tests
// require more of them, if a new test uses a backend method which is not yet implemented,
// the fake backend will panic.
package rpctest

import (
	"context"
	"errors"
	"math/big"
	"sort"
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

// AccountState represents the state of an account
// in the fake backend's state database.
type AccountState struct {
	Nonce   uint64
	Balance *big.Int
	Code    []byte
	Store   map[common.Hash]common.Hash
}

// Block represents a block in the fake backend's block history.
type Block struct {
	Number     uint64
	Hash       common.Hash
	ParentHash common.Hash
}

type backendBuilder struct {
	be fakeBackend
}

// NewBackendBuilder creates a new backendBuilder with default values for all fields of the fakeBackend.
//
// An example of using the builder in a test:
//
//		func TestMyRPCMethod(t *testing.T) {
//		    backend := rpctest.NewBackendBuilder(t).
//
//		        // preload some account state
//		        WithAccount(someAddress, rpctest.AccountState{Balance: big.NewInt(1e18)}).
//
//		        // Define a custom block history
//		        WithBlockHistory([]Block{
//		            {Number: 1, Hash: common.HexToHash("0xabc"), ParentHash: common.HexToHash("0xdef")},
//		            {Number: 2, Hash: common.HexToHash("0xdef"), ParentHash: common.HexToHash("0xabc")},
//		        }).
//
//		        // Build the instance
//		        Build()
//
//		    // Use backend in the test...
//		    api := SomeRPCHandler{backend: backend}
//
//		    // Call the RPC method
//		    result, err := api.MyRPCMethod(t.Context(), ...)
//
//	     	// Assert results...
//		}
func NewBackendBuilder(t *testing.T) backendBuilder {
	return backendBuilder{
		be: fakeBackend{
			chainID:      opera.FakeNetworkID,
			state:        NewTestState(t),
			blockHistory: defaultBlockHistory(),
		},
	}
}

// WithChainID sets the chain ID for the fake backend.
//
// By default, the chain ID is set to the FakeNetworkID defined in the opera package.
func (b backendBuilder) WithChainID(chainID uint64) backendBuilder {
	b.be.chainID = chainID
	return b
}

// WithBlockHistory sets the block history for the fake backend.
//
// By default, the block history contains a single block with number 1 and hash "0x1".
func (b backendBuilder) WithBlockHistory(blocks []Block) backendBuilder {
	if len(blocks) == 0 {
		b.be.blockHistory = defaultBlockHistory()
	} else {
		b.be.blockHistory = blocks
	}
	return b
}

// WithPool sets the transaction pool for the fake backend. This method is
// compatible with a mocks of the transaction pool, using the TxPool interface
// defined at the end of this file.
//
// By default, the transaction pool is nil, any RPC method that tries to access
// the transaction pool without it being set will cause the fake backend to panic.
func (b backendBuilder) WithPool(pool TxPool) backendBuilder {
	b.be.pool = pool
	return b
}

// WithAccount preloads the state of a given account in the fake backend's state database.
// This allows tests to set up specific account conditions (e.g., balance, nonce, code)
// before executing RPC methods that depend on that state.
//
// By default, the fake backend's state database is empty
func (b backendBuilder) WithAccount(addr common.Address, account AccountState) backendBuilder {
	b.be.state.setAccount(addr, account)
	return b
}

// WithUpgrade sets the network upgrades for the fake backend's rules.
//
// By default, the fake backend uses the Brio upgrades defined in the opera package.
func (b backendBuilder) WithUpgrade(blockHeight idx.Block, upgrades opera.Upgrades) backendBuilder {
	if b.be.upgrades == nil {
		b.be.upgrades = make(map[idx.Block]opera.Upgrades)
	}
	b.be.upgrades[blockHeight] = upgrades
	return b
}

// Build constructs the fakeBackend instance with the configured parameters.
func (b backendBuilder) Build() *fakeBackend {
	if b.be.upgrades == nil {
		b.be.upgrades = make(map[idx.Block]opera.Upgrades)
		b.be.upgrades[0] = opera.GetBrioUpgrades()
	}
	return &b.be
}

// defaultBlockHistory returns a default block history with a single block (number 1, hash "0x1").
func defaultBlockHistory() []Block {
	return []Block{
		{
			Number: 1,
			Hash:   common.HexToHash("0x1"),
		},
	}
}

// fakeBackend is a simple implementation of the ethapi.Backend interface,
// with only the methods needed for testing implemented.
//
// This type is not exported as it is meant to be used via the builder constructed
// by NewBackendBuilder.
//
// When using the backend in new tests, if a method is called which is not yet
// implemented, the test will panic. ethapi.Backend methods are to be implemented
// on demand as the tests require more of them.
type fakeBackend struct {
	ethapi.Backend

	chainID      uint64
	state        *testState
	pool         TxPool
	blockHistory []Block
	upgrades     map[idx.Block]opera.Upgrades
}

// GetSigner is a helper method outside of the ethapi.Backend interface,
// to facilitate the use of the correct signer in tests.
func (b *fakeBackend) GetSigner() types.Signer {
	return types.LatestSignerForChainID(b.ChainID())
}

func (b *fakeBackend) ChainID() *big.Int {
	return big.NewInt(int64(b.chainID))
}

func (b *fakeBackend) CurrentBlock() *evmcore.EvmBlock {
	lastblock := b.blockHistory[len(b.blockHistory)-1]
	return &evmcore.EvmBlock{EvmHeader: *ToEvmHeader(lastblock)}
}

func (b *fakeBackend) GetEVM(
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
	chainConfig := b.ChainConfig(idx.Block(blockContext.BlockNumber.Uint64()))
	if vmConfig == nil {
		defaultCfg := vm.Config{}
		vmConfig = &defaultCfg
	}
	return vm.NewEVM(*blockContext, state, chainConfig, *vmConfig), func() error { return nil }, nil
}

func (b *fakeBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error) {

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

func (b *fakeBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeader, error) {
	for _, block := range b.blockHistory {
		if block.Hash == hash {
			return ToEvmHeader(block), nil
		}
	}
	return nil, errors.New("block header not found")
}

func (b *fakeBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmBlock, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return &evmcore.EvmBlock{
		EvmHeader: *header,
	}, nil
}

func (b *fakeBackend) BlockByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmBlock, error) {
	header, err := b.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return &evmcore.EvmBlock{
		EvmHeader: *header,
	}, nil
}

func (b *fakeBackend) MaxGasLimit() uint64 {
	rules, err := b.GetNetworkRules(context.Background(), 0)
	if err != nil {
		return 0
	}
	return rules.Economy.Gas.MaxEventGas
}

func (b *fakeBackend) MinGasPrice() *big.Int {
	return big.NewInt(1)
}

func (b *fakeBackend) RPCGasCap() uint64 {
	return 50_000_000
}

func (b *fakeBackend) RPCTxFeeCap() float64 {
	return 1.0
}

func (b *fakeBackend) RPCEVMTimeout() time.Duration {
	return 5 * time.Second
}

func (b *fakeBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	if b.pool == nil {
		// This error indicates that the test is trying to send a transaction
		// without having set a transaction pool in the fake backend.
		panic("transaction pool not set in fake backend; use WithPool to set it")
	}
	return b.pool.AddLocal(signedTx)
}

func (b *fakeBackend) StateAndBlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (state.StateDB, *evmcore.EvmBlock, error) {

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

// GetNetworkRules returns the network rules applicable at a given block height.
func (b *fakeBackend) GetNetworkRules(ctx context.Context, blockHeight idx.Block) (*opera.Rules, error) {

	heights := make([]idx.Block, 0, len(b.upgrades))
	for k := range b.upgrades {
		heights = append(heights, k)
	}

	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })

	for i := len(heights) - 1; i >= 0; i-- {
		k := heights[i]
		if k <= blockHeight {
			rules := opera.FakeNetRules(b.upgrades[k])
			return &rules, nil
		}
	}

	return nil, errors.New("no network rules found for the given block height")
}

// ChainConfig returns the chain configuration applicable at a given block height,
// based on the defined network upgrades.
func (b *fakeBackend) ChainConfig(blockHeight idx.Block) *params.ChainConfig {
	heights := make([]opera.UpgradeHeight, 0, len(b.upgrades))
	for height, upgrades := range b.upgrades {

		heights = append(heights, opera.MakeUpgradeHeight(upgrades, height))
	}
	return opera.CreateTransientEvmChainConfig(b.chainID, heights, blockHeight)
}

// TxPool is a minimal interface for the transaction pool, only including the methods needed for testing.
type TxPool interface {
	AddLocal(*types.Transaction) error
}
