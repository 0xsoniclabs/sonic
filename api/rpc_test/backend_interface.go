package rpctest

import (
	"context"
	"errors"
	"math/big"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func (b *backend) ChainConfig(blockHeight idx.Block) *params.ChainConfig {
	return &params.ChainConfig{}
}

func (b *backend) ChainID() *big.Int {
	return big.NewInt(int64(b.chainId))
}

func (b *backend) CurrentBlock() *evmcore.EvmBlock {
	lastblock := b.blockHistory[len(b.blockHistory)-1]
	return &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(int64(lastblock.Number)),
		},
	}
}

func (b *backend) GetEVM(
	ctx context.Context,
	state vm.StateDB,
	header *evmcore.EvmHeader,
	vmConfig *vm.Config,
	blockContext *vm.BlockContext,
) (*vm.EVM, func() error, error) {

	chainConfig := &params.ChainConfig{}
	if blockContext == nil {
		chainCtx := ethapi.ChainContext{
			Be:  b,
			Ctx: ctx,
		}
		newCtx := evmcore.NewEVMBlockContext(header, &chainCtx, nil)
		blockContext = &newCtx
	}
	return vm.NewEVM(*blockContext, state, chainConfig, *vmConfig), func() error { return nil }, nil
}

func (b *backend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error) {
	// TODO: look for the right block
	block := b.blockHistory[len(b.blockHistory)-1]
	return &evmcore.EvmHeader{
		Number: big.NewInt(int64(block.Number)),
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

func (b *backend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	if b.pool == nil {
		return errors.New("tx pool not initialized")
	}
	return b.pool.AddLocal(signedTx)
}

func (b *backend) StateAndBlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (state.StateDB, *evmcore.EvmBlock, error) {
	// TODO: look for the right block
	return b.state.Copy(), &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(int64(b.CurrentBlock().NumberU64())),
		},
	}, nil
}

func (b *backend) GetNetworkRules(ctx context.Context, blockHeight idx.Block) (*opera.Rules, error) {
	return &b.rules, nil
}
