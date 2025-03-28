package gossip

import (
	"math"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/inter"
)

func (s *Store) GetGenesisID() *consensus.Hash {
	if v := s.cache.Genesis.Load(); v != nil {
		val := v.(consensus.Hash)
		return &val
	}
	valBytes, err := s.table.Genesis.Get([]byte("g"))
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if len(valBytes) == 0 {
		return nil
	}
	val := consensus.BytesToHash(valBytes)
	s.cache.Genesis.Store(val)
	return &val
}

func (s *Store) fakeGenesisHash() consensus.EventHash {
	fakeGenesisHash := consensus.EventHash(*s.GetGenesisID())
	for i := range fakeGenesisHash[:8] {
		fakeGenesisHash[i] = 0
	}
	return fakeGenesisHash
}

func (s *Store) SetGenesisID(val consensus.Hash) {
	err := s.table.Genesis.Put([]byte("g"), val.Bytes())
	if err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
	s.cache.Genesis.Store(val)
}

// SetBlock stores chain block.
func (s *Store) SetBlock(n consensus.BlockID, b *inter.Block) {
	s.rlp.Set(s.table.Blocks, n.Bytes(), b)

	// Add to LRU cache.
	s.cache.Blocks.Add(n, b, uint(b.EstimateSize()))
}

// GetBlock returns stored block.
func (s *Store) GetBlock(n consensus.BlockID) *inter.Block {
	// Get block from LRU cache first.
	if c, ok := s.cache.Blocks.Get(n); ok {
		return c.(*inter.Block)
	}

	block, _ := s.rlp.Get(s.table.Blocks, n.Bytes(), &inter.Block{}).(*inter.Block)

	// Add to LRU cache.
	if block != nil {
		s.cache.Blocks.Add(n, block, uint(block.EstimateSize()))
	}

	return block
}

func (s *Store) HasBlock(n consensus.BlockID) bool {
	has, _ := s.table.Blocks.Has(n.Bytes())
	return has
}

func (s *Store) ForEachBlock(fn func(index consensus.BlockID, block *inter.Block)) {
	it := s.table.Blocks.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		var block inter.Block
		err := rlp.DecodeBytes(it.Value(), &block)
		if err != nil {
			s.Log.Crit("Failed to decode block", "err", err)
		}
		fn(consensus.BytesToBlock(it.Key()), &block)
	}
}

// SetBlockIndex stores chain block index.
func (s *Store) SetBlockIndex(id common.Hash, n consensus.BlockID) {
	if err := s.table.BlockHashes.Put(id.Bytes(), n.Bytes()); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}

	s.cache.BlockHashes.Add(id, n, nominalSize)
}

// GetBlockIndex returns stored block index.
func (s *Store) GetBlockIndex(id consensus.EventHash) *consensus.BlockID {
	nVal, ok := s.cache.BlockHashes.Get(id)
	if ok {
		n, ok := nVal.(consensus.BlockID)
		if ok {
			return &n
		}
	}

	buf, err := s.table.BlockHashes.Get(id.Bytes())
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if buf == nil {
		if id == s.fakeGenesisHash() {
			zero := consensus.BlockID(0)
			return &zero
		}
		return nil
	}
	n := consensus.BytesToBlock(buf)

	s.cache.BlockHashes.Add(id, n, nominalSize)

	return &n
}

// SetGenesisBlockIndex stores genesis block index.
func (s *Store) SetGenesisBlockIndex(n consensus.BlockID) {
	if err := s.table.Genesis.Put([]byte("i"), n.Bytes()); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// GetGenesisBlockIndex returns stored genesis block index.
func (s *Store) GetGenesisBlockIndex() *consensus.BlockID {
	buf, err := s.table.Genesis.Get([]byte("i"))
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if buf == nil {
		return nil
	}
	n := consensus.BytesToBlock(buf)

	return &n
}

func (s *Store) GetGenesisTime() inter.Timestamp {
	n := s.GetGenesisBlockIndex()
	if n == nil {
		return 0
	}
	block := s.GetBlock(*n)
	if block == nil {
		return 0
	}
	return block.Time
}

func (s *Store) SetEpochBlock(b consensus.BlockID, e consensus.Epoch) {
	err := s.table.EpochBlocks.Put((math.MaxUint64 - b).Bytes(), e.Bytes())
	if err != nil {
		s.Log.Crit("Failed to set key-value", "err", err)
	}
}

func (s *Store) FindBlockEpoch(b consensus.BlockID) consensus.Epoch {
	if c, ok := s.cache.Blocks.Get(b); ok {
		return c.(*inter.Block).Epoch
	}

	it := s.table.EpochBlocks.NewIterator(nil, (math.MaxUint64 - b).Bytes())
	defer it.Release()
	if !it.Next() {
		return 0
	}
	return consensus.BytesToEpoch(it.Value())
}

func (s *Store) GetBlockTxs(n consensus.BlockID, block *inter.Block) types.Transactions {
	if cached := s.evm.GetCachedEvmBlock(n); cached != nil {
		return cached.Transactions
	}

	transactions := make(types.Transactions, 0, len(block.TransactionHashes))
	for _, txHash := range block.TransactionHashes {
		tx := s.evm.GetTx(txHash)
		if tx == nil {
			log.Crit("Referenced transaction not found", "tx", txHash.String())
			continue
		}
		transactions = append(transactions, tx)
	}

	return transactions
}
