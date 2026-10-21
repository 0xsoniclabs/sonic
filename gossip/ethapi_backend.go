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

package gossip

import (
	"context"
	stderrors "errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"

	"github.com/0xsoniclabs/sonic/api/legacy"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

//go:generate go tool mockgen -source=ethapi_backend.go -destination=ethapi_backend_mock.go -package=gossip

// EthAPIBackend is what the node's own handler of the legacy RPC methods asks
// of the node: it implements legacy.Backend and the backend of the test API. The
// RPC module reaches the node through rpcCore instead.
type EthAPIBackend struct {
	extRPCEnabled       bool
	svc                 *Service
	state               StateReader
	signer              types.Signer
	allowUnprotectedTxs bool

	// tipSuggester and gasEstimator answer what the node's legacy wallet
	// methods default their fees and gas to. The gas price oracle and the EVM
	// run in the RPC module, which sets them.
	tipSuggester atomic.Pointer[func(certainty uint64) *big.Int]
	gasEstimator atomic.Pointer[func(ctx context.Context, args legacy.TransactionArgs) (hexutil.Uint64, error)]
}

var _ legacy.Backend = (*EthAPIBackend)(nil)

// SetGasEstimator sets the source of the gas estimates.
func (b *EthAPIBackend) SetGasEstimator(estimate func(ctx context.Context, args legacy.TransactionArgs) (hexutil.Uint64, error)) {
	b.gasEstimator.Store(&estimate)
}

// EstimateGas estimates the gas of a transaction at the head.
func (b *EthAPIBackend) EstimateGas(ctx context.Context, args legacy.TransactionArgs) (hexutil.Uint64, error) {
	if estimate := b.gasEstimator.Load(); estimate != nil {
		return (*estimate)(ctx, args)
	}
	return 0, stderrors.New("gas estimation is not available before the RPC module runs")
}

// isLatestBlockNumber returns true if the block number is latest, pending, finalized or safe
func isLatestBlockNumber(number rpc.BlockNumber) bool {
	return number == rpc.LatestBlockNumber ||
		number == rpc.PendingBlockNumber ||
		number == rpc.FinalizedBlockNumber ||
		number == rpc.SafeBlockNumber
}

// decodeShortEventID decodes ShortID
// example of a ShortID: "5:26:a2395846", where 5 is epoch, 26 is lamport, a2395846 are first bytes of the hash
// s is a string splitted by ":" separator
func decodeShortEventID(s []string) (idx.Epoch, idx.Lamport, []byte, error) {
	if len(s) != 3 {
		return 0, 0, nil, errors.New("incorrect format of short event ID (need Epoch:Lamport:Hash")
	}
	epoch, err := strconv.ParseUint(s[0], 10, 32)
	if err != nil {
		return 0, 0, nil, errors.Wrap(err, "short hash parsing error (lamport)")
	}
	lamport, err := strconv.ParseUint(s[1], 10, 32)
	if err != nil {
		return 0, 0, nil, errors.Wrap(err, "short hash parsing error (lamport)")
	}
	return idx.Epoch(epoch), idx.Lamport(lamport), common.FromHex(s[2]), nil
}

// SetTipSuggester sets the source of the suggested gas tip.
func (b *EthAPIBackend) SetTipSuggester(suggest func(certainty uint64) *big.Int) {
	b.tipSuggester.Store(&suggest)
}

// SetExtRPCEnabled updates extRPCEnabled
func (b *EthAPIBackend) SetExtRPCEnabled(v bool) {
	b.extRPCEnabled = v
}

// ChainConfig returns the active chain configuration.
func (b *EthAPIBackend) ChainConfig(blockHeight idx.Block) *params.ChainConfig {
	return b.svc.store.GetEvmChainConfig(blockHeight)
}

func (b *EthAPIBackend) CurrentBlock() *evmcore.EvmBlock {
	return b.state.CurrentBlock()
}

// HistoryPruningCutoff returns the block height at which pruning was done.
func (b *EthAPIBackend) HistoryPruningCutoff() uint64 {
	return 0
}

// GetFullEventID "converts" ShortID to full event's hash, by searching in events DB.
func (b *EthAPIBackend) GetFullEventID(shortEventID string) (hash.Event, error) {
	s := strings.Split(shortEventID, ":")
	if len(s) == 1 {
		// it's a full hash
		eventHash, err := hexutil.Decode(shortEventID)
		if err != nil {
			return hash.Event{}, errors.Wrap(err, "full hash parsing error")
		}
		return hash.Event(hash.BytesToHash(eventHash)), nil
	}
	// short hash
	epoch, lamport, prefix, err := decodeShortEventID(s)
	if err != nil {
		return hash.Event{}, err
	}

	options := b.svc.store.FindEventHashes(epoch, lamport, prefix)
	if len(options) == 0 {
		return hash.Event{}, errors.New("event not found by short ID")
	}
	if len(options) > 1 {
		return hash.Event{}, errors.New("there're multiple events with the same short ID, please use full ID")
	}
	return options[0], nil
}

// GetEventPayload returns Lachesis event by hash or short ID.
func (b *EthAPIBackend) GetEventPayload(ctx context.Context, shortEventID string) (*inter.EventPayload, error) {
	id, err := b.GetFullEventID(shortEventID)
	if err != nil {
		return nil, err
	}
	return b.svc.store.GetEventPayload(id), nil
}

// GetEvent returns the Lachesis event header by hash or short ID.
func (b *EthAPIBackend) GetEvent(ctx context.Context, shortEventID string) (*inter.Event, error) {
	id, err := b.GetFullEventID(shortEventID)
	if err != nil {
		return nil, err
	}
	return b.svc.store.GetEvent(id), nil
}

// GetHeads returns IDs of all the epoch events with no descendants.
// Heads are only kept for the open epoch, selected as pending or by its
// number. Sealed epochs, including latest (the latest sealed one), fail
// with errHeadsUnavailable.
func (b *EthAPIBackend) GetHeads(ctx context.Context, epoch rpc.BlockNumber) (hash.Events, error) {
	requested, err := b.epochWithDefault(ctx, epoch)
	if err != nil {
		return nil, err
	}

	// The epoch may seal between resolving it and reading its heads, so the
	// store is the only authority on their availability.
	heads := b.svc.store.GetHeads(requested)
	if heads == nil {
		return nil, fmt.Errorf("epoch %d: %w", requested, errHeadsUnavailable)
	}
	heads.RLock()
	defer heads.RUnlock()
	return heads.Val.Slice(), nil
}

func (b *EthAPIBackend) epochWithDefault(ctx context.Context, epoch rpc.BlockNumber) (requested idx.Epoch, err error) {
	current := b.svc.store.GetEpoch()

	switch {
	case epoch == rpc.EarliestBlockNumber:
		requested = idx.Epoch(b.HistoryPruningCutoff())
	case epoch == rpc.PendingBlockNumber:
		requested = current
	case isLatestBlockNumber(epoch):
		requested = current - 1
	case epoch >= 0 && epoch <= rpc.BlockNumber(current):
		requested = idx.Epoch(epoch)
	default:
		err = errors.New("epoch is not in range")
		return
	}
	return requested, nil
}

func (b *EthAPIBackend) FetchReceiptsForBlock(block *evmcore.EvmBlock) types.Receipts {
	time := uint64(block.Time.Unix())
	baseFee := block.BaseFee
	blobGasPrice := new(big.Int) // TODO issue #147
	receipts := b.svc.store.evm.GetReceipts(idx.Block(block.NumberU64()),
		b.ChainConfig(idx.Block(block.NumberU64())),
		block.Hash,
		time,
		baseFee,
		blobGasPrice,
		block.Transactions,
	)
	return receipts
}

func (b *EthAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.svc.txpool.AddLocal(signedTx)
}

func (b *EthAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.svc.txpool.Pending(false)
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

// GetTxPosition returns the position of the given transaction in the chain, or
// nil if the transaction is unknown. Transactions of blocks that are not
// published yet are reported as unknown, so that they cannot be observed
// before the state of their block can be.
func (b *EthAPIBackend) GetTxPosition(txHash common.Hash) *evmstore.TxPosition {
	position := b.svc.store.evm.GetTxPosition(txHash)
	if position == nil || position.Block > b.svc.store.GetLatestBlockIndex() {
		return nil
	}
	return position
}

func (b *EthAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.svc.txpool.Nonce(addr), nil
}

// Progress returns current synchronization status of this node
func (b *EthAPIBackend) Progress() legacy.PeerProgress {
	p2pProgress := b.svc.handler.myProgress()
	highestP2pProgress := b.svc.handler.highestPeerProgress()
	lastBlock := b.svc.store.GetBlock(p2pProgress.LastBlockIdx)

	return legacy.PeerProgress{
		CurrentEpoch:     p2pProgress.Epoch,
		CurrentBlock:     p2pProgress.LastBlockIdx,
		CurrentBlockHash: p2pProgress.LastBlockAtropos,
		CurrentBlockTime: lastBlock.Time,
		HighestBlock:     highestP2pProgress.LastBlockIdx,
		HighestEpoch:     highestP2pProgress.Epoch,
	}
}

func (b *EthAPIBackend) SuggestGasTipCap(ctx context.Context, certainty uint64) *big.Int {
	if suggest := b.tipSuggester.Load(); suggest != nil {
		return (*suggest)(certainty)
	}
	// Until the RPC module runs, the pool's own floor is the best answer.
	return new(big.Int).Set(b.svc.txpool.MinTip())
}

func (b *EthAPIBackend) AccountManager() *accounts.Manager {
	return b.svc.AccountManager()
}

func (b *EthAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *EthAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *EthAPIBackend) RPCTxFeeCap() float64 {
	return b.svc.config.RPCTxFeeCap
}

// CurrentEpoch returns current epoch number.
func (b *EthAPIBackend) CurrentEpoch(ctx context.Context) idx.Epoch {
	return b.svc.store.GetEpoch()
}

func (b *EthAPIBackend) MinGasPrice() *big.Int {
	current := b.state.CurrentBaseFee()
	return gaspricelimits.GetSuggestedGasPriceForNewTransactions(current)
}

func (b *EthAPIBackend) GetUptime(ctx context.Context, vid idx.ValidatorID) (*big.Int, error) {
	// Note: loads bs and es atomically to avoid a race condition
	bs, es := b.svc.store.GetBlockEpochState()
	if !es.Validators.Exists(vid) {
		return nil, nil
	}
	return new(big.Int).SetUint64(uint64(bs.GetValidatorState(vid, es.Validators).Uptime)), nil
}

func (b *EthAPIBackend) GetOriginatedFee(ctx context.Context, vid idx.ValidatorID) (*big.Int, error) {
	// Note: loads bs and es atomically to avoid a race condition
	bs, es := b.svc.store.GetBlockEpochState()
	if !es.Validators.Exists(vid) {
		return nil, nil
	}
	return bs.GetValidatorState(vid, es.Validators).Originated, nil
}

func (b *EthAPIBackend) GetDowntime(ctx context.Context, vid idx.ValidatorID) (idx.Block, inter.Timestamp, error) {
	// Note: loads bs and es atomically to avoid a race condition
	bs, es := b.svc.store.GetBlockEpochState()
	if !es.Validators.Exists(vid) {
		return 0, 0, nil
	}
	vs := bs.GetValidatorState(vid, es.Validators)
	missedBlocks := idx.Block(0)
	if bs.LastBlock.Idx > vs.LastBlock {
		missedBlocks = bs.LastBlock.Idx - vs.LastBlock
	}
	missedTime := inter.Timestamp(0)
	if bs.LastBlock.Time > vs.LastOnlineTime {
		missedTime = bs.LastBlock.Time - vs.LastOnlineTime
	}
	if missedBlocks < es.Rules.Economy.BlockMissedSlack {
		return 0, 0, nil
	}
	return missedBlocks, missedTime, nil
}

func (b *EthAPIBackend) GetEpochBlockState(ctx context.Context, epoch rpc.BlockNumber) (*iblockproc.BlockState, *iblockproc.EpochState, error) {
	switch epoch {
	case rpc.PendingBlockNumber:
		bs, es := b.svc.store.GetBlockState(), b.svc.store.GetEpochState()
		return &bs, &es, nil
	case rpc.LatestBlockNumber, rpc.FinalizedBlockNumber, rpc.SafeBlockNumber:
		epoch = rpc.BlockNumber(b.svc.store.GetEpoch())
	case rpc.EarliestBlockNumber:
		epoch = rpc.BlockNumber(b.HistoryPruningCutoff())
	}
	bs, es := b.svc.store.GetHistoryBlockEpochState(idx.Epoch(epoch))
	return bs, es, nil
}

func (b *EthAPIBackend) ChainID() *big.Int {
	return new(big.Int).SetUint64(b.svc.store.GetRules().NetworkID)
}

func (b *EthAPIBackend) SealedEpochTiming(ctx context.Context) (start inter.Timestamp, end inter.Timestamp) {
	es := b.svc.store.GetEpochState()
	return es.PrevEpochStart, es.EpochStart
}

func (b *EthAPIBackend) IsTestOnlyApiEnabled() bool {
	return b.svc.config.EnableTestOnlyApi
}

func (b *EthAPIBackend) ProposeTransactions(
	txs types.Transactions,
) error {
	if !b.IsTestOnlyApiEnabled() {
		return fmt.Errorf("this feature is disabled")
	}
	if len(b.svc.emitters) == 0 {
		return fmt.Errorf("no emitters available to propose transactions")
	}
	return b.proposeTransactionsInternal(txs, b.svc.emitters[0])
}

func (b *EthAPIBackend) AddTransactions(
	txs types.Transactions,
) error {
	if !b.IsTestOnlyApiEnabled() {
		return fmt.Errorf("this feature is disabled")
	}
	if b.svc.txpool == nil {
		return fmt.Errorf("no transaction pool available to add transactions")
	}
	return stderrors.Join(b.svc.txpool.AddLocals(txs)...)
}

func (b *EthAPIBackend) proposeTransactionsInternal(
	txs types.Transactions,
	emitter forceableEmitter,
) error {
	return emitter.ForceEventEmissionForTesting(txs)
}

var errHeadsUnavailable = stderrors.New("heads are not available")

type forceableEmitter interface {
	ForceEventEmissionForTesting(txs []*types.Transaction) error
}
