package rpctest

import (
	"context"
	"errors"
	"iter"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/result"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// AccountManager implements [ethapi.Backend].
func (b *backend) AccountManager() *accounts.Manager {
	panic("unimplemented")
}

// BlockByHash implements [ethapi.Backend].
func (b *backend) BlockByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmBlock, error) {
	panic("unimplemented")
}

// BlockByNumber implements [ethapi.Backend].
func (b *backend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmBlock, error) {
	panic("unimplemented")
}

// CalcBlockExtApi implements [ethapi.Backend].
func (b *backend) CalcBlockExtApi() bool {
	panic("unimplemented")
}

// ChainConfig implements [ethapi.Backend].
func (b *backend) ChainConfig(blockHeight idx.Block) *params.ChainConfig {
	panic("unimplemented")
}

// ChainID implements [ethapi.Backend].
func (b *backend) ChainID() *big.Int {
	return big.NewInt(int64(b.chainId))
}

// CurrentBlock implements [ethapi.Backend].
func (b *backend) CurrentBlock() *evmcore.EvmBlock {
	lastblock := b.blockHistory[len(b.blockHistory)-1]
	return &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(int64(lastblock.Number)),
		},
	}

}

// CurrentEpoch implements [ethapi.Backend].
func (b *backend) CurrentEpoch(ctx context.Context) idx.Epoch {
	panic("unimplemented")
}

// EnumerateBlockCertificates implements [ethapi.Backend].
func (b *backend) EnumerateBlockCertificates(first idx.Block) iter.Seq[result.T[cert.BlockCertificate]] {
	panic("unimplemented")
}

// EnumerateCommitteeCertificates implements [ethapi.Backend].
func (b *backend) EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[cert.CommitteeCertificate]] {
	panic("unimplemented")
}

// ExtRPCEnabled implements [ethapi.Backend].
func (b *backend) ExtRPCEnabled() bool {
	panic("unimplemented")
}

// FetchReceiptsForBlock implements [ethapi.Backend].
func (b *backend) FetchReceiptsForBlock(block *evmcore.EvmBlock) types.Receipts {
	panic("unimplemented")
}

// GetDowntime implements [ethapi.Backend].
func (b *backend) GetDowntime(ctx context.Context, vid idx.ValidatorID) (idx.Block, inter.Timestamp, error) {
	panic("unimplemented")
}

// GetEVM implements [ethapi.Backend].
func (b *backend) GetEVM(ctx context.Context, state vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {
	panic("unimplemented")
}

// GetEpochBlockState implements [ethapi.Backend].
func (b *backend) GetEpochBlockState(ctx context.Context, epoch rpc.BlockNumber) (*iblockproc.BlockState, *iblockproc.EpochState, error) {
	panic("unimplemented")
}

// GetEvent implements [ethapi.Backend].
func (b *backend) GetEvent(ctx context.Context, shortEventID string) (*inter.Event, error) {
	panic("unimplemented")
}

// GetEventPayload implements [ethapi.Backend].
func (b *backend) GetEventPayload(ctx context.Context, shortEventID string) (*inter.EventPayload, error) {
	panic("unimplemented")
}

// GetGenesisID implements [ethapi.Backend].
func (b *backend) GetGenesisID() common.Hash {
	panic("unimplemented")
}

// GetHeads implements [ethapi.Backend].
func (b *backend) GetHeads(ctx context.Context, epoch rpc.BlockNumber) (hash.Events, error) {
	panic("unimplemented")
}

// GetLatestBlockCertificate implements [ethapi.Backend].
func (b *backend) GetLatestBlockCertificate() (cert.BlockCertificate, error) {
	panic("unimplemented")
}

// GetLatestCommitteeCertificate implements [ethapi.Backend].
func (b *backend) GetLatestCommitteeCertificate() (cert.CommitteeCertificate, error) {
	panic("unimplemented")
}

// GetNetworkRules implements [ethapi.Backend].
func (b *backend) GetNetworkRules(ctx context.Context, blockHeight idx.Block) (*opera.Rules, error) {
	panic("unimplemented")
}

// GetOriginatedFee implements [ethapi.Backend].
func (b *backend) GetOriginatedFee(ctx context.Context, vid idx.ValidatorID) (*big.Int, error) {
	panic("unimplemented")
}

// GetPoolNonce implements [ethapi.Backend].
func (b *backend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	panic("unimplemented")
}

// GetPoolTransaction implements [ethapi.Backend].
func (b *backend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	panic("unimplemented")
}

// GetPoolTransactions implements [ethapi.Backend].
func (b *backend) GetPoolTransactions() (types.Transactions, error) {
	panic("unimplemented")
}

// GetReceiptsByNumber implements [ethapi.Backend].
func (b *backend) GetReceiptsByNumber(ctx context.Context, number rpc.BlockNumber) (types.Receipts, error) {
	panic("unimplemented")
}

// GetTransaction implements [ethapi.Backend].
func (b *backend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, uint64, uint64, error) {
	panic("unimplemented")
}

// GetUpgradeHeights implements [ethapi.Backend].
func (b *backend) GetUpgradeHeights() []opera.UpgradeHeight {
	panic("unimplemented")
}

// GetUptime implements [ethapi.Backend].
func (b *backend) GetUptime(ctx context.Context, vid idx.ValidatorID) (*big.Int, error) {
	panic("unimplemented")
}

// HeaderByHash implements [ethapi.Backend].
func (b *backend) HeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeader, error) {
	panic("unimplemented")
}

// HeaderByNumber implements [ethapi.Backend].
func (b *backend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error) {
	panic("unimplemented")
}

// HistoryPruningCutoff implements [ethapi.Backend].
func (b *backend) HistoryPruningCutoff() uint64 {
	panic("unimplemented")
}

// MaxGasLimit implements [ethapi.Backend].
func (b *backend) MaxGasLimit() uint64 {
	panic("unimplemented")
}

// MinGasPrice implements [ethapi.Backend].
func (b *backend) MinGasPrice() *big.Int {
	panic("unimplemented")
}

// Progress implements [ethapi.Backend].
func (b *backend) Progress() ethapi.PeerProgress {
	panic("unimplemented")
}

// RPCEVMTimeout implements [ethapi.Backend].
func (b *backend) RPCEVMTimeout() time.Duration {
	panic("unimplemented")
}

// RPCGasCap implements [ethapi.Backend].
func (b *backend) RPCGasCap() uint64 {
	panic("unimplemented")
}

// RPCTxFeeCap implements [ethapi.Backend].
func (b *backend) RPCTxFeeCap() float64 {
	panic("unimplemented")
}

// ResolveRpcBlockNumberOrHash implements [ethapi.Backend].
func (b *backend) ResolveRpcBlockNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (idx.Block, error) {
	panic("unimplemented")
}

// SealedEpochTiming implements [ethapi.Backend].
func (b *backend) SealedEpochTiming(ctx context.Context) (start inter.Timestamp, end inter.Timestamp) {
	panic("unimplemented")
}

// SendTx implements [ethapi.Backend].
func (b *backend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	if b.pool == nil {
		return errors.New("tx pool not initialized")
	}
	return b.pool.AddLocal(signedTx)
}

// StateAndBlockByNumberOrHash implements [ethapi.Backend].
func (b *backend) StateAndBlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (state.StateDB, *evmcore.EvmBlock, error) {
	// TODO: look for the right block

	return b.state.Copy(), &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(int64(b.CurrentBlock().NumberU64())),
		},
	}, nil
}

// Stats implements [ethapi.Backend].
func (b *backend) Stats() (pending int, queued int) {
	panic("unimplemented")
}

// SubscribeNewTxsNotify implements [ethapi.Backend].
func (b *backend) SubscribeNewTxsNotify(chan<- evmcore.NewTxsNotify) event.Subscription {
	panic("unimplemented")
}

// SuggestGasTipCap implements [ethapi.Backend].
func (b *backend) SuggestGasTipCap(ctx context.Context, certainty uint64) *big.Int {
	panic("unimplemented")
}

// TxPoolContent implements [ethapi.Backend].
func (b *backend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	panic("unimplemented")
}

// TxPoolContentFrom implements [ethapi.Backend].
func (b *backend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	panic("unimplemented")
}

// UnprotectedAllowed implements [ethapi.Backend].
func (b *backend) UnprotectedAllowed() bool {
	panic("unimplemented")
}
