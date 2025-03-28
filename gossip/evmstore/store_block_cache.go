package evmstore

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/evmcore"
)

func (s *Store) GetCachedEvmBlock(n consensus.BlockID) *evmcore.EvmBlock {
	c, ok := s.cache.EvmBlocks.Get(n)
	if !ok {
		return nil
	}

	return c.(*evmcore.EvmBlock)
}

func (s *Store) SetCachedEvmBlock(n consensus.BlockID, b *evmcore.EvmBlock) {
	var empty = common.Hash{}
	if b.EvmHeader.TxHash == empty {
		panic("You have to cache only completed blocks (with txs)")
	}
	s.cache.EvmBlocks.Add(n, b, uint(b.EstimateSize()))
}
