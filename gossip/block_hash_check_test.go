package gossip

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/utils/errlock"
)

func makeTestBlock(number uint64, epoch idx.Epoch) *inter.Block {
	return inter.NewBlockBuilder().
		WithNumber(number).
		WithParentHash(common.Hash{byte(number)}).
		WithStateRoot(common.Hash{byte(number + 1)}).
		WithGasLimit(10_000_000).
		WithBaseFee(big.NewInt(1000)).
		WithEpoch(epoch).
		Build()
}

func makeTestEvent(creator idx.ValidatorID, start idx.Block, hashes []hash.Hash) *inter.EventPayload {
	e := &inter.MutableEventPayload{}
	e.SetCreator(creator)
	e.SetBlockHashes(inter.BlockHashes{
		Start:  start,
		Epoch:  1,
		Hashes: hashes,
	})
	return e.Build()
}

func makeValidators(stakes map[idx.ValidatorID]pos.Weight) *pos.Validators {
	builder := pos.NewBuilder()
	for id, weight := range stakes {
		builder.Set(id, weight)
	}
	return builder.Build()
}

func TestBlockHashChecker_NilErrorLock(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	checker := newBlockHashChecker(store, nil)
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
	})
	checker.reset(1, validators)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	// Event with wrong hash should not panic when errorLock is nil
	event := makeTestEvent(1, 1, []hash.Hash{{0xFF}})
	checker.check(event)
}

func TestBlockHashChecker_NoBlockHashes(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
	})
	checker.reset(1, validators)

	// Event with no block hashes (Start == 0) should be a no-op
	e := &inter.MutableEventPayload{}
	e.SetCreator(1)
	checker.check(e.Build())

	require.NoError(t, errorLock.Check())
}

func TestBlockHashChecker_MatchingHashes(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
	})
	checker.reset(1, validators)

	// Event with matching hash should not trigger
	event := makeTestEvent(1, 1, []hash.Hash{hash.Hash(block.Hash())})
	checker.check(event)

	require.NoError(t, errorLock.Check())
	require.Empty(t, checker.disagreements)
}

func TestBlockHashChecker_BlockNotInStore(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
	})
	checker.reset(1, validators)

	// Block 5 is not in the store — should be skipped
	event := makeTestEvent(1, 5, []hash.Hash{{0xFF}})
	checker.check(event)

	require.NoError(t, errorLock.Check())
	require.Empty(t, checker.disagreements)
}

func TestBlockHashChecker_DisagreementBelowThreshold(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	// Validator 1 has weight 100 out of 300 total (33% < 67%)
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
		2: 100,
		3: 100,
	})
	checker.reset(1, validators)

	event := makeTestEvent(1, 1, []hash.Hash{{0xFF}})
	checker.check(event)

	require.NoError(t, errorLock.Check())
	require.Len(t, checker.disagreements[1], 1)
}

func TestBlockHashChecker_DisagreementExceedsThreshold(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	// Total weight = 300, threshold = 200
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
		2: 100,
		3: 100,
	})
	checker.reset(1, validators)

	wrongHash := hash.Hash{0xFF}

	// First disagreement: validator 1 (100/300 = 33%)
	checker.check(makeTestEvent(1, 1, []hash.Hash{wrongHash}))
	require.NoError(t, errorLock.Check())

	// Second disagreement: validator 2 (200/300 = 67%)
	checker.check(makeTestEvent(2, 1, []hash.Hash{wrongHash}))
	require.NoError(t, errorLock.Check())

	// Third disagreement: validator 3 (300/300 = 100% > 67%)
	require.Panics(t, func() {
		checker.check(makeTestEvent(3, 1, []hash.Hash{wrongHash}))
	})
}

func TestBlockHashChecker_ThresholdExactlyTwoThirds(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	// Total weight = 3, threshold = (3*2)/3 = 2
	// Disagreement of exactly 2 is NOT > 2, so should not trigger
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 1,
		2: 1,
		3: 1,
	})
	checker.reset(1, validators)

	wrongHash := hash.Hash{0xFF}

	checker.check(makeTestEvent(1, 1, []hash.Hash{wrongHash}))
	checker.check(makeTestEvent(2, 1, []hash.Hash{wrongHash}))

	// 2/3 is exactly the threshold — should NOT panic (need strictly greater)
	require.NoError(t, errorLock.Check())

	// Third validator tips it over: 3 > 2
	require.Panics(t, func() {
		checker.check(makeTestEvent(3, 1, []hash.Hash{wrongHash}))
	})
}

func TestBlockHashChecker_SameValidatorCountedOnce(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	// Validator 1 has weight 100 out of 300 (33%)
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
		2: 100,
		3: 100,
	})
	checker.reset(1, validators)

	wrongHash := hash.Hash{0xFF}

	// Same validator sends multiple events — should still only count once
	checker.check(makeTestEvent(1, 1, []hash.Hash{wrongHash}))
	checker.check(makeTestEvent(1, 1, []hash.Hash{wrongHash}))
	checker.check(makeTestEvent(1, 1, []hash.Hash{wrongHash}))

	require.NoError(t, errorLock.Check())
	require.Len(t, checker.disagreements[1], 1)
}

func TestBlockHashChecker_MultipleBlocks(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block1 := makeTestBlock(1, 1)
	block2 := makeTestBlock(2, 1)
	store.SetBlock(1, block1)
	store.SetBlock(2, block2)

	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
		2: 100,
		3: 100,
	})
	checker.reset(1, validators)

	// Disagree on block 1 but agree on block 2
	event := makeTestEvent(1, 1, []hash.Hash{
		{0xFF},                   // wrong hash for block 1
		hash.Hash(block2.Hash()), // correct hash for block 2
	})
	checker.check(event)

	require.NoError(t, errorLock.Check())
	require.Len(t, checker.disagreements[1], 1) // block 1 has disagreement
	require.Empty(t, checker.disagreements[2])  // block 2 has no disagreement
}

func TestBlockHashChecker_EpochReset(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
		2: 100,
		3: 100,
	})
	checker.reset(1, validators)

	wrongHash := hash.Hash{0xFF}

	// Accumulate disagreements in epoch 1
	checker.check(makeTestEvent(1, 1, []hash.Hash{wrongHash}))
	checker.check(makeTestEvent(2, 1, []hash.Hash{wrongHash}))
	require.Len(t, checker.disagreements[1], 2)

	// Reset for new epoch — disagreements should be cleared
	checker.reset(2, validators)
	require.Empty(t, checker.disagreements)

	// After reset, old disagreements don't carry over
	require.NoError(t, errorLock.Check())
}

func TestBlockHashChecker_NilValidators(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)
	// Don't call reset — validators is nil

	block := makeTestBlock(1, 1)
	store.SetBlock(1, block)

	// Should not panic with nil validators
	event := makeTestEvent(1, 1, []hash.Hash{{0xFF}})
	checker.check(event)

	require.NoError(t, errorLock.Check())
}

func TestBlockHashChecker_PartialBlockRange(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	// Only block 2 exists in store, block 3 doesn't
	block2 := makeTestBlock(2, 1)
	store.SetBlock(2, block2)

	// Total weight = 3 — single validator with weight 3 exceeds threshold
	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 3,
	})
	checker.reset(1, validators)

	// Event covers blocks 2 and 3; block 3 is missing so only block 2 is checked
	event := makeTestEvent(1, 2, []hash.Hash{
		{0xFF}, // wrong hash for block 2
		{0xEE}, // block 3 doesn't exist, should be skipped
	})

	require.Panics(t, func() {
		checker.check(event)
	})
}

func TestBlockHashChecker_IndependentBlockTracking(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	errorLock := errlock.New(t.TempDir())
	checker := newBlockHashChecker(store, errorLock)

	block1 := makeTestBlock(1, 1)
	block2 := makeTestBlock(2, 1)
	store.SetBlock(1, block1)
	store.SetBlock(2, block2)

	validators := makeValidators(map[idx.ValidatorID]pos.Weight{
		1: 100,
		2: 100,
		3: 100,
	})
	checker.reset(1, validators)

	// Validator 1 disagrees on block 1
	checker.check(makeTestEvent(1, 1, []hash.Hash{{0xFF}}))
	// Validator 2 disagrees on block 2
	checker.check(makeTestEvent(2, 2, []hash.Hash{{0xEE}}))

	// Neither block has reached 2/3 threshold
	require.NoError(t, errorLock.Check())
	require.Len(t, checker.disagreements[1], 1)
	require.Len(t, checker.disagreements[2], 1)
}
