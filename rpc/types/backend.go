package rpctypes

import (
	"context"
	"iter"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
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

//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=rpctypes

// PeerProgress is synchronization status of a peer
type PeerProgress struct {
	CurrentEpoch     idx.Epoch
	CurrentBlock     idx.Block
	CurrentBlockHash hash.Event
	CurrentBlockTime inter.Timestamp
	HighestBlock     idx.Block
	HighestEpoch     idx.Epoch
}

type Backend interface {
	EthereunAPIBackend
	RPCLimitsBackend
	BlockchainApiBackend
	TxPoolSenderBackend
	TxPoolGetterBackend
	BundlesBackend
	LachesisDAGApiBackend
	LachesisaBFTApiBackend
	SccApiBackend

	SharedBackend
}

type SharedBackend interface {
	ChainID() *big.Int
}

type EthereunAPIBackend interface {
	Progress() PeerProgress
	SuggestGasTipCap(ctx context.Context, certainty uint64) *big.Int
	AccountManager() *accounts.Manager
	ExtRPCEnabled() bool
	CalcBlockExtApi() bool
	HistoryPruningCutoff() uint64 // block height at which pruning was done
	SharedBackend
}

type RPCLimitsBackend interface {
	RPCGasCap() uint64            // global gas cap for eth_call over rpc: DoS protection
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	RPCTxFeeCap() float64         // global tx fee cap for all transaction related APIs
}

type BlockchainApiBackend interface {
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmHeader, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmHeader, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*evmcore.EvmBlock, error)
	StateAndBlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (state.StateDB, *evmcore.EvmBlock, error)
	ResolveRpcBlockNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (idx.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*evmcore.EvmBlock, error)
	GetReceiptsByNumber(ctx context.Context, number rpc.BlockNumber) (types.Receipts, error)
	FetchReceiptsForBlock(block *evmcore.EvmBlock) types.Receipts
	GetEVM(ctx context.Context, state vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error)
	MinGasPrice() *big.Int
	MaxGasLimit() uint64

	ChainConfig(blockHeight idx.Block) *params.ChainConfig
	CurrentBlock() *evmcore.EvmBlock
	GetUpgradeHeights() []opera.UpgradeHeight
	GetGenesisID() common.Hash

	GetNetworkRules(ctx context.Context, blockHeight idx.Block) (*opera.Rules, error)
	SharedBackend
}

type TxPoolSenderBackend interface {
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	UnprotectedAllowed() bool // allows only for EIP155 transactions.
	SharedBackend
}

type TxPoolGetterBackend interface {
	GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, uint64, uint64, error)
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash common.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error)
	Stats() (pending int, queued int)
	TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)
	TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions)
	SubscribeNewTxsNotify(chan<- evmcore.NewTxsNotify) event.Subscription
	SharedBackend
}

type LachesisDAGApiBackend interface {
	GetEventPayload(ctx context.Context, shortEventID string) (*inter.EventPayload, error)
	GetEvent(ctx context.Context, shortEventID string) (*inter.Event, error)
	GetHeads(ctx context.Context, epoch rpc.BlockNumber) (hash.Events, error)
	CurrentEpoch(ctx context.Context) idx.Epoch
	SealedEpochTiming(ctx context.Context) (start inter.Timestamp, end inter.Timestamp)
	SharedBackend
}

type BundlesBackend interface {
	GetBundleExecutionInfo(common.Hash) *bundle.ExecutionInfo
	SharedBackend
}

type LachesisaBFTApiBackend interface {
	GetEpochBlockState(ctx context.Context, epoch rpc.BlockNumber) (*iblockproc.BlockState, *iblockproc.EpochState, error)
	GetDowntime(ctx context.Context, vid idx.ValidatorID) (idx.Block, inter.Timestamp, error)
	GetUptime(ctx context.Context, vid idx.ValidatorID) (*big.Int, error)
	GetOriginatedFee(ctx context.Context, vid idx.ValidatorID) (*big.Int, error)
	SharedBackend
}

// SccApiBackend is the backend interface for the Sonic Certification Chain API.
// An implementation thereof provides access to the Sonic Certification Chain.
type SccApiBackend interface {
	GetLatestCommitteeCertificate() (cert.CommitteeCertificate, error)
	EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[cert.CommitteeCertificate]]

	GetLatestBlockCertificate() (cert.BlockCertificate, error)
	EnumerateBlockCertificates(first idx.Block) iter.Seq[result.T[cert.BlockCertificate]]
}
