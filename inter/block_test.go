package inter

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestBlock_Hash_CachesResult(t *testing.T) {
	require := require.New(t)
	block := &Block{}
	require.Nil(block.hash.Load())
	hash := block.Hash()
	require.NotNil(block.hash.Load())
	require.Equal(hash, *block.hash.Load())
}

func TestBlock_Hash_UpdateOfCachedHashIsRaceFree(t *testing.T) {
	require := require.New(t)

	block := &Block{}
	require.Nil(block.hash.Load())

	// Run multiple goroutines requesting the hash concurrently. If the Hash()
	// function contains a race condition, the test will fail when being run
	// with the --race flag.
	const N = 2
	got := make([]common.Hash, N)
	var wg sync.WaitGroup
	wg.Add(N)
	for i := range N {
		go func() {
			defer wg.Done()
			got[i] = block.Hash()
		}()
	}
	wg.Wait()

	for i := range got {
		require.Equal(got[i], block.Hash())
	}
}

func TestBlock_EncodeRLP_SerializationIsCompatibleToBlockData(t *testing.T) {
	require := require.New(t)

	original := &Block{
		blockData: blockData{
			Number:     1,
			ParentHash: common.Hash{2},
			StateRoot:  common.Hash{3},
		},
	}

	blockData, err := rlp.EncodeToBytes(original)
	require.NoError(err)

	contentData, err := rlp.EncodeToBytes(original.blockData)
	require.NoError(err)

	require.Equal(blockData, contentData)
}

func TestBlock_DecodeRLP_CanRestoreBlockData(t *testing.T) {
	require := require.New(t)

	original := &Block{
		blockData: blockData{
			Number:     1,
			ParentHash: common.Hash{2},
			StateRoot:  common.Hash{3},
		},
	}
	data, err := rlp.EncodeToBytes(original)
	require.NoError(err)

	restored := &Block{}
	require.NoError(rlp.DecodeBytes(data, restored))

	require.Equal(original.Number, restored.Number)
	require.Equal(original.ParentHash, restored.ParentHash)
	require.Equal(original.StateRoot, restored.StateRoot)
}

func TestBlockBuilder_Build_BlockContainsBlockData(t *testing.T) {
	require := require.New(t)

	block := NewBlockBuilder().
		WithNumber(1).
		WithParentHash(common.Hash{2}).
		WithStateRoot(common.Hash{3}).
		Build()

	require.Equal(uint64(1), block.Number)
	require.Equal(common.Hash{2}, block.ParentHash)
	require.Equal(common.Hash{3}, block.StateRoot)
}
