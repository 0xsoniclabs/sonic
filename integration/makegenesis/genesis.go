package makegenesis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/core/tracing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/drivermodule"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/eventmodule"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/evmmodule"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/sealmodule"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/utils"

	mptIo "github.com/0xsoniclabs/carmen/go/database/mpt/io"
	carmen "github.com/0xsoniclabs/carmen/go/state"
)

type GenesisBuilder struct {
	tmpStateDB    state.StateDB
	carmenDir     string
	carmenStateDb carmen.StateDB

	totalSupply *big.Int

	blocks       []ibr.LlrIdxFullBlockRecord
	epochs       []ier.LlrIdxFullEpochRecord
	currentEpoch ier.LlrIdxFullEpochRecord
}

type BlockProc struct {
	SealerModule     blockproc.SealerModule
	TxListenerModule blockproc.TxListenerModule
	PreTxTransactor  blockproc.TxTransactor
	PostTxTransactor blockproc.TxTransactor
	EventsModule     blockproc.ConfirmedEventsModule
	EVMModule        blockproc.EVM
}

func DefaultBlockProc() BlockProc {
	return BlockProc{
		SealerModule:     sealmodule.New(),
		TxListenerModule: drivermodule.NewDriverTxListenerModule(),
		PreTxTransactor:  drivermodule.NewDriverTxPreTransactor(),
		PostTxTransactor: drivermodule.NewDriverTxTransactor(),
		EventsModule:     eventmodule.New(),
		EVMModule:        evmmodule.New(),
	}
}

func (b *GenesisBuilder) AddBalance(acc common.Address, balance *big.Int) {
	if len(b.blocks) > 0 {
		panic("cannot add balance after block zero is finalized")
	}
	b.tmpStateDB.AddBalance(acc, utils.BigIntToUint256(balance), tracing.BalanceIncreaseGenesisBalance)
	b.totalSupply.Add(b.totalSupply, balance)
}

func (b *GenesisBuilder) SetCode(acc common.Address, code []byte) {
	if len(b.blocks) > 0 {
		panic("cannot add code after block zero is finalized")
	}
	b.tmpStateDB.SetCode(acc, code)
}

func (b *GenesisBuilder) SetNonce(acc common.Address, nonce uint64) {
	if len(b.blocks) > 0 {
		panic("cannot add nonce after block zero is finalized")
	}
	b.tmpStateDB.SetNonce(acc, nonce, tracing.NonceChangeGenesis)
}

func (b *GenesisBuilder) SetStorage(acc common.Address, key, val common.Hash) {
	if len(b.blocks) > 0 {
		panic("cannot set storage after block zero is finalized")
	}
	b.tmpStateDB.SetState(acc, key, val)
}

func (b *GenesisBuilder) SetCurrentEpoch(er ier.LlrIdxFullEpochRecord) {
	b.currentEpoch = er
}

func (b *GenesisBuilder) TotalSupply() *big.Int {
	return b.totalSupply
}

func (b *GenesisBuilder) CurrentHash() consensus.Hash {
	er := b.epochs[len(b.epochs)-1]
	return er.Hash()
}

func NewGenesisBuilder() *GenesisBuilder {
	carmenDir, err := os.MkdirTemp("", "opera-tmp-genesis")
	if err != nil {
		panic(fmt.Errorf("failed to create temporary dir for GenesisBuilder: %v", err))
	}
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:   "go-file",
		Schema:    carmen.Schema(5),
		Archive:   carmen.S5Archive,
		Directory: carmenDir,
		LiveCache: 1, // use minimum cache (not default)
	})
	if err != nil {
		panic(fmt.Errorf("failed to create carmen state; %s", err))
	}
	// Set cache size to lowest value possible
	carmenStateDb := carmen.CreateCustomStateDBUsing(carmenState, 1024)
	tmpStateDB := evmstore.CreateCarmenStateDb(carmenStateDb)
	return &GenesisBuilder{
		tmpStateDB:    tmpStateDB,
		carmenDir:     carmenDir,
		carmenStateDb: carmenStateDb,
		totalSupply:   new(big.Int),
	}
}

type dummyHeaderReturner struct {
	blocks []ibr.LlrIdxFullBlockRecord
}

func (d dummyHeaderReturner) GetHeader(_ common.Hash, position uint64) *evmcore.EvmHeader {
	if position < uint64(len(d.blocks)) {
		return &evmcore.EvmHeader{
			BaseFee: d.blocks[position].BaseFee,
		}
	}
	return &evmcore.EvmHeader{
		BaseFee: big.NewInt(0),
	}
}

// FinalizeBlockZero finalizes the genesis block 0 by computing the state root hash of
// the initial block and filling in other block header information. This function must
// be called before ExecuteGenesisTxs.
func (b *GenesisBuilder) FinalizeBlockZero(
	rules opera.Rules,
	genesisTime inter.Timestamp,
) (
	blockHash common.Hash,
	stateRoot common.Hash,
	err error,
) {
	if len(b.blocks) > 0 {
		return common.Hash{}, common.Hash{}, errors.New("block zero already finalized")
	}

	// construct state root of initial state
	b.tmpStateDB.EndBlock(0)
	genesisStateRoot := b.tmpStateDB.GetStateHash()

	// construct the block record for the genesis block
	blockBuilder := inter.NewBlockBuilder().
		WithEpoch(1).
		WithNumber(0).
		WithParentHash(common.Hash{}).
		WithStateRoot(genesisStateRoot).
		WithTime(genesisTime).
		WithDuration(0).
		WithGasLimit(rules.Blocks.MaxBlockGas).
		WithGasUsed(0).
		WithBaseFee(gasprice.GetInitialBaseFee(rules.Economy)).
		WithPrevRandao(common.Hash{31: 1})

	llrBlock := ibr.FullBlockRecordFor(blockBuilder.Build(), nil, nil)
	b.blocks = append(b.blocks, ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: *llrBlock,
		Idx:                0,
	})

	return common.Hash(b.blocks[0].BlockHash), genesisStateRoot, nil
}

func (b *GenesisBuilder) ExecuteGenesisTxs(blockProc BlockProc, genesisTxs types.Transactions) error {
	if len(b.blocks) == 0 {
		return errors.New("no block zero - run FinalizeBlockZero first")
	}

	bs, es := b.currentEpoch.BlockState.Copy(), b.currentEpoch.EpochState.Copy()
	es.Rules.Economy.MinGasPrice = big.NewInt(0) // < needed since genesis transactions have gas price 0

	blockCtx := iblockproc.BlockCtx{
		Idx:    bs.LastBlock.Idx + 1,
		Time:   bs.LastBlock.Time + 1,
		Leader: consensus.EventHash{},
	}

	sealer := blockProc.SealerModule.Start(blockCtx, bs, es)
	txListener := blockProc.TxListenerModule.Start(blockCtx, bs, es, b.tmpStateDB)
	evmProcessor := blockProc.EVMModule.Start(blockCtx, b.tmpStateDB, dummyHeaderReturner{b.blocks}, func(l *types.Log) {
		txListener.OnNewLog(l)
	}, es.Rules, es.Rules.EvmChainConfig([]opera.UpgradeHeight{
		{
			Upgrades: es.Rules.Upgrades,
			Height:   0,
		},
	}), common.Hash{0x01}) // non-zero PrevRandao necessary to enable Cancun

	// Execute genesis transactions
	evmProcessor.Execute(genesisTxs)
	bs = txListener.Finalize()

	// Execute pre-internal transactions
	preInternalTxs := blockProc.PreTxTransactor.PopInternalTxs(blockCtx, bs, es, true, b.tmpStateDB)
	evmProcessor.Execute(preInternalTxs)
	bs = txListener.Finalize()

	// Seal epoch
	sealer.Update(bs, es)
	bs, es = sealer.SealEpoch()
	txListener.Update(bs, es)

	// Execute post-internal transactions
	internalTxs := blockProc.PostTxTransactor.PopInternalTxs(blockCtx, bs, es, true, b.tmpStateDB)
	evmProcessor.Execute(internalTxs)

	evmBlock, skippedTxs, receipts := evmProcessor.Finalize()
	for i, r := range receipts {
		if r.Status == 0 {
			return fmt.Errorf("genesis transaction %d of %d reverted", i, len(receipts))
		}
	}
	if len(skippedTxs) != 0 {
		return fmt.Errorf("genesis transaction is skipped (%v)", skippedTxs)
	}
	bs = txListener.Finalize()
	bs.FinalizedStateRoot = consensus.Hash(evmBlock.Root)

	bs.LastBlock = blockCtx

	receiptsStorage := make([]*types.ReceiptForStorage, len(receipts))
	for i, r := range receipts {
		receiptsStorage[i] = (*types.ReceiptForStorage)(r)
	}

	// construct the block record for the genesis block
	blockBuilder := inter.NewBlockBuilder().
		WithEpoch(1).
		WithNumber(uint64(blockCtx.Idx)).
		WithParentHash(common.Hash(b.blocks[len(b.blocks)-1].BlockHash)).
		WithStateRoot(common.Hash(bs.FinalizedStateRoot)).
		WithTime(evmBlock.Time).
		WithDuration(1).
		WithGasLimit(evmBlock.GasLimit).
		WithGasUsed(evmBlock.GasUsed).
		WithBaseFee(evmBlock.BaseFee).
		WithPrevRandao(evmBlock.PrevRandao)

	for txIndex, transaction := range evmBlock.Transactions {
		if !bytes.Equal(transaction.Hash().Bytes(), receipts[txIndex].TxHash.Bytes()) {
			return fmt.Errorf("genesis transaction hash %d of %d does not match with the receipt",
				txIndex, len(receipts))
		}
		blockBuilder.AddTransaction(transaction, receipts[txIndex])
	}

	llrBlock := ibr.FullBlockRecordFor(blockBuilder.Build(), evmBlock.Transactions, receiptsStorage)
	b.blocks = append(b.blocks, ibr.LlrIdxFullBlockRecord{
		LlrFullBlockRecord: *llrBlock,
		Idx:                blockCtx.Idx,
	})

	// add epochs
	b.epochs = append(b.epochs, b.currentEpoch) // safe epoch 1
	b.currentEpoch = ier.LlrIdxFullEpochRecord{ // create epoch 2
		LlrFullEpochRecord: ier.LlrFullEpochRecord{
			BlockState: bs,
			EpochState: es,
		},
		Idx: es.Epoch,
	}
	b.epochs = append(b.epochs, b.currentEpoch)
	return nil
}

type memFile struct {
	*bytes.Buffer
}

func (f *memFile) Close() error {
	*f = memFile{}
	return nil
}

func (b *GenesisBuilder) Build(head genesis.Header) *genesisstore.Store {
	err := b.carmenStateDb.Close()
	if err != nil {
		panic(fmt.Errorf("failed to close genesis carmen state; %s", err))
	}
	return genesisstore.NewStore(func(name string) (io.Reader, error) {
		buf := &memFile{bytes.NewBuffer(nil)}
		if name == genesisstore.BlocksSection(0) {
			for i := len(b.blocks) - 1; i >= 0; i-- {
				_ = rlp.Encode(buf, b.blocks[i])
			}
			return buf, nil
		}
		if name == genesisstore.EpochsSection(0) {
			for i := len(b.epochs) - 1; i >= 0; i-- {
				_ = rlp.Encode(buf, b.epochs[i])
			}
			return buf, nil
		}
		if name == genesisstore.FwsLiveSection(0) {
			err := mptIo.Export(context.Background(), mptIo.NewLog(), filepath.Join(b.carmenDir, "live"), buf)
			if err != nil {
				return nil, err
			}
		}
		if name == genesisstore.FwsArchiveSection(0) {
			err := mptIo.ExportArchive(context.Background(), mptIo.NewLog(), filepath.Join(b.carmenDir, "archive"), buf)
			if err != nil {
				return nil, err
			}
		}
		if buf.Len() == 0 {
			return nil, errors.New("not found")
		}
		return buf, nil
	}, head, func() error {
		err := os.RemoveAll(b.carmenDir)
		*b = GenesisBuilder{}
		return err
	})
}
