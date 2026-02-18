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

package topicsdb

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/0xsoniclabs/sonic/utils/leap"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=leap_join_test.go -destination=leap_join_test_mock.go -package=topicsdb

func TestFindInBlocks_FindsLogsUsingPattern(t *testing.T) {
	require := require.New(t)

	// Index some logs
	index := NewWithLeapJoin(memorydb.New())

	// Define query parameters.
	from := idx.Block(10)
	to := idx.Block(20)
	pattern := [][]common.Hash{
		{{12: 1}, {12: 2}}, // topic 0: two options for the address
		{{3}},              // topic 1: one option
	}

	// Add some matching log records.
	matching1 := &types.Log{
		BlockNumber: 10,
		Address:     common.Address{1},
		Topics:      []common.Hash{{3}},
		TxHash:      common.Hash{0, 1}, // < needed to be indexed
		Data:        []byte{1, 2, 3},
	}
	matching2 := &types.Log{
		BlockNumber: 15,
		Address:     common.Address{2},
		Topics:      []common.Hash{{3}},
		TxHash:      common.Hash{0, 2},
		Data:        []byte{4, 5},
	}
	matching3 := &types.Log{
		BlockNumber: 12,
		Address:     common.Address{2},
		Topics:      []common.Hash{{3}, {4}}, // more topics are accepted
		TxHash:      common.Hash{0, 3},
		Data:        []byte{6},
	}
	require.NoError(index.Push(matching1, matching2, matching3))

	// also add some non-matching log messages
	require.NoError(index.Push(
		&types.Log{
			BlockNumber: 8, // too early
			Address:     common.Address{1},
			Topics:      []common.Hash{{3}},
			TxHash:      common.Hash{1, 1},
		},
		&types.Log{
			BlockNumber: 21, // too late
			Address:     common.Address{1},
			Topics:      []common.Hash{{3}},
			TxHash:      common.Hash{1, 2},
		},
		&types.Log{
			BlockNumber: 12,
			Address:     common.Address{3}, // non-matching address
			Topics:      []common.Hash{{3}},
			TxHash:      common.Hash{1, 3},
		},
		&types.Log{
			BlockNumber: 12,
			Address:     common.Address{2},
			Topics:      []common.Hash{{4}}, // non-matching topic
			TxHash:      common.Hash{1, 4},
		},
		&types.Log{
			BlockNumber: 12,
			Address:     common.Address{2},
			Topics:      []common.Hash{}, // too few topics
			TxHash:      common.Hash{1, 5},
		},
	))

	logs, err := index.FindInBlocks(
		t.Context(), from, to, pattern,
	)
	require.NoError(err)
	require.Len(logs, 3)

	require.Equal(
		[]*types.Log{matching1, matching3, matching2}, // in block order
		logs,
	)
}

func TestFindInBlocksUsingLeapJoin_ReleasesAllIterators(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	index := NewMock_table(ctrl)

	// A pattern causing 3 iterators to be created.
	pattern := [][]common.Hash{
		{{1}, {2}, {3}},
	}

	iter1 := NewMock_iterator(ctrl)
	iter2 := NewMock_iterator(ctrl)
	iter3 := NewMock_iterator(ctrl)

	index.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter1)
	index.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter2)
	index.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter3)

	// All iterators are empty, and need to be released.
	iter1.EXPECT().Next().Return(false)
	iter1.EXPECT().Error().Return(nil)
	iter1.EXPECT().Release()

	iter2.EXPECT().Next().Return(false)
	iter2.EXPECT().Error().Return(nil)
	iter2.EXPECT().Release()

	iter3.EXPECT().Next().Return(false)
	iter3.EXPECT().Error().Return(nil)
	iter3.EXPECT().Release()

	res, err := findInBlocksUsingLeapJoin(
		t.Context(), 0, 10, pattern, index, nil,
	)

	require.NoError(err)
	require.Nil(res)
}

func TestFindInBlocksUsingLeapJoin_ReturnsEmptyIfBlockRangeIsEmpty(t *testing.T) {
	tests := map[string]struct {
		from, to idx.Block
		skipped  bool
	}{
		"from less than to":        {from: 5, to: 10, skipped: false},
		"from equals to":           {from: 10, to: 10, skipped: false},
		"from one greater than to": {from: 11, to: 10, skipped: true},
		"from greater than to":     {from: 15, to: 10, skipped: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// We use a cancelled context to catch those filter calls that pass
			// the range test before any filter processing is happening. This
			// way, we do not need to set up the indices for the join.
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			logs, err := findInBlocksUsingLeapJoin(
				ctx, tc.from, tc.to, nil, nil, nil,
			)
			require.Empty(t, logs)

			if tc.skipped {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ctx.Err())
			}
		})
	}
}

func TestFindInBLocksUsingLeapJoin_FailsIfNoPatternsAreProvided(t *testing.T) {
	logs, err := findInBlocksUsingLeapJoin(
		context.Background(), 0, 10, nil, nil, nil,
	)
	require.Empty(t, logs)
	require.ErrorContains(t, err, "empty topics")
}

func TestFindInBlocksUsingLeapJoin_CanBeCancelledViaContext(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	index := NewMock_table(ctrl)
	iter := NewMock_iterator(ctrl)
	logs := NewMock_reader(ctrl)

	// A simple query with a single topic, iterating over the result of a single
	// topic iterator.
	index.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter)

	// The iterator produces arbitrary many results.
	iter.EXPECT().Next().Return(true).AnyTimes()
	iter.EXPECT().Key().Return(make([]byte, topicKeySize)).AnyTimes()
	iter.EXPECT().Value().Return(make([]byte, 1)).AnyTimes()
	iter.EXPECT().Error().Return(nil).AnyTimes()
	iter.EXPECT().Release()

	// Those keys are resolved to logs.
	dummySerializedLog := make([]byte, 200)
	ctx, cancel := context.WithCancel(t.Context())
	gomock.InOrder(
		logs.EXPECT().Get(gomock.Any()).Return(dummySerializedLog, nil),
		logs.EXPECT().Get(gomock.Any()).DoAndReturn(func([]byte) ([]byte, error) {
			// cancel the context while the second log is resolved, to test
			// cancellation during the join processing, not just before.
			cancel()
			return dummySerializedLog, nil
		}),
	)

	res, err := findInBlocksUsingLeapJoin(
		ctx, 0, 10, [][]common.Hash{nil, {{1}}}, index, logs,
	)

	require.Nil(res)
	require.ErrorIs(err, context.Canceled)
}

func TestFindInBlocksUsingLeapJoin_FailingLogFetchStopsJoin(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	index := NewMock_table(ctrl)
	iter := NewMock_iterator(ctrl)
	logs := NewMock_reader(ctrl)

	// A simple query with a single topic, iterating over the result of a single
	// topic iterator.
	index.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter)

	// The iterator produces arbitrary many results.
	iter.EXPECT().Next().Return(true).AnyTimes()
	iter.EXPECT().Key().Return(make([]byte, topicKeySize)).AnyTimes()
	iter.EXPECT().Value().Return(make([]byte, 1)).AnyTimes()
	iter.EXPECT().Error().Return(nil).AnyTimes()
	iter.EXPECT().Release()

	// Those keys are resolved to logs.
	dummySerializedLog := make([]byte, 200)
	issue := errors.New("injected log fetch error")
	gomock.InOrder(
		logs.EXPECT().Get(gomock.Any()).Return(dummySerializedLog, nil),
		logs.EXPECT().Get(gomock.Any()).DoAndReturn(func([]byte) ([]byte, error) {
			// return an error while fetching the second log, to test that
			// errors during the join processing stop the join.
			return nil, issue
		}),
	)

	res, err := findInBlocksUsingLeapJoin(
		t.Context(), 0, 10, [][]common.Hash{nil, {{1}}}, index, logs,
	)

	require.Nil(res)
	require.ErrorIs(err, issue)
}

func TestFindInBlocksUsingLeapJoin_ErrorsDuringIndexIterationsAreReported(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	index := NewMock_table(ctrl)
	iter := NewMock_iterator(ctrl)
	logs := NewMock_reader(ctrl)

	// A simple query with a single topic, iterating over the result of a single
	// topic iterator.
	index.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(iter)

	// The iterator fails with an error.
	issue := errors.New("injected index access error")
	iter.EXPECT().Next().Return(false).AnyTimes()
	iter.EXPECT().Error().Return(issue).AnyTimes()
	iter.EXPECT().Release()

	res, err := findInBlocksUsingLeapJoin(
		t.Context(), 0, 10, [][]common.Hash{nil, {{1}}}, index, logs,
	)

	require.Nil(res)
	require.ErrorIs(err, issue)
}

func TestNewTopicIterator_ReturnsNilIteratorForEmptyTopics(t *testing.T) {
	it, raw := newTopicIterator(nil, nil, 0, 0, 0)
	require.Nil(t, it)
	require.Nil(t, raw)
}

func TestNewTopicIterator_CreatesOneIteratorPerTopicOption(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)

	topics := []common.Hash{{1}, {2}, {3}}
	position := uint8(5)
	from := idx.Block(10)
	to := idx.Block(20)

	// Expect an iterator to be created for each topic option.
	it, raw := newTopicIterator(table, topics, int(position), from, to)
	require.Len(raw, len(topics))

	// The actual DB iterators are lazy-initialized on the first Next call.
	for _, topic := range topics {
		prefix := append(topic[:], byte(position))
		start := binary.BigEndian.AppendUint64(nil, uint64(from))

		iter := NewMock_iterator(ctrl)
		table.EXPECT().NewIterator(prefix, start).Return(iter)

		// We simulate all-empty iterators.
		gomock.InOrder(
			iter.EXPECT().Next().Return(false),
			iter.EXPECT().Error().Return(nil),
			iter.EXPECT().Release(),
		)
	}

	require.False(it.Next())
}

func TestNewTopicIterator_SkipsUnionWrapperIfOnlyOneTopicOption(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)

	topics := []common.Hash{{1}} // only one topic option
	position := uint8(5)
	from := idx.Block(10)
	to := idx.Block(20)

	it, raw := newTopicIterator(table, topics, int(position), from, to)
	require.Len(raw, 1)
	require.Equal(it, raw[0])
}

// --- IndexIterator tests ---

var _ leap.Iterator[logrec] = (*indexIterator)(nil)

func TestIndexIterator_Default_IsEmpty(t *testing.T) {
	it := &indexIterator{}
	// is empty
	require.False(t, it.Next())
	// remains empty
	require.False(t, it.Next())
	// current does not panic
	require.Equal(t, logrec{}, it.Current())
}

func TestIndexIterator_Default_CanBeReleased(t *testing.T) {
	it := &indexIterator{}
	it.Release()
}

func TestIndexIterator_newIndexIterator_DoesNotCreateUnderlyingIterator(t *testing.T) {
	iter := newIndexIterator(nil, common.Hash{}, 0, 0, 0)
	require.NotNil(t, iter)
	require.Nil(t, iter.iter)
}

func TestIndexIterator_Next_CreatesUnderlyingIteratorLazily(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)
	iter := NewMock_iterator(ctrl)

	topic := common.Hash{1, 2, 3}
	pos := 5
	from := idx.Block(10)
	to := idx.Block(20)

	it := newIndexIterator(table, topic, pos, from, to)
	require.NotNil(it)
	require.Nil(it.iter)

	// Expect the iterator to be created with the correct prefix and start key
	// when calling Next for the first time.
	prefix := append(topic[:], byte(pos))
	start := binary.BigEndian.AppendUint64(nil, uint64(from))
	table.EXPECT().NewIterator(prefix, start).Return(iter)

	id := NewID(uint64(from)+1, common.Hash{}, 0)
	key := topicKey(topic, uint8(pos), id)
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(key),
		iter.EXPECT().Value().Return([]byte{5}),
		iter.EXPECT().Error().Return(nil),
	)

	require.True(it.Next())
	require.Equal(iter, it.iter)
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not create a new iterator.
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(key),
		iter.EXPECT().Value().Return([]byte{5}),
		iter.EXPECT().Error().Return(nil),
	)
	require.True(it.Next())
}

func TestIndexIterator_Next_AbortsIfIteratorCreationFails(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)

	topic := common.Hash{1, 2, 3}
	pos := 5
	from := idx.Block(10)
	to := idx.Block(20)

	it := newIndexIterator(table, topic, pos, from, to)
	require.NotNil(it)
	require.Nil(it.iter)

	// Let the iterator creation fail by returning nil.
	table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(nil)

	require.False(it.Next())
	require.ErrorContains(it.Error(), "failed to create iterator")
	require.Nil(it.iter)
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not create a new iterator.
	require.False(it.Next())
}

func TestIndexIterator_Next_AbortsIfIteratorHasError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	topic := common.Hash{1, 2, 3}
	from := idx.Block(10)
	pos := 5
	id := NewID(uint64(from)+1, common.Hash{}, 0)
	key := topicKey(topic, uint8(pos), id)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	issue := errors.New("injected issue")
	iter.EXPECT().Next().Return(true)
	iter.EXPECT().Key().Return(key)
	iter.EXPECT().Value().Return([]byte{5})
	iter.EXPECT().Error().Return(issue)

	require.False(it.Next())
	require.ErrorIs(it.Error(), issue)
	require.Equal(iter, it.iter)
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not produce new elements.
	require.False(it.Next())
	require.ErrorIs(it.Error(), issue)
}

func TestIndexIterator_Next_SkipsKeysWithInvalidLength(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	// Simulate an iterator that returns a key with invalid length.
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return([]byte{1, 2, 3}), // too short => skipped
		// The iterator is exhausted after skipping the invalid key.
		iter.EXPECT().Next().Return(false),
		iter.EXPECT().Error().Return(nil),
		iter.EXPECT().Release(),
	)

	require.False(it.Next())
}

func TestIndexIterator_Next_AbortsWithValueLengthMismatch(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	// Simulate an iterator that returns a key with invalid length.
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(make([]byte, topicKeySize)),
		iter.EXPECT().Value().Return(make([]byte, 2)), // invalid length => error
	)

	require.False(it.Next())
	require.ErrorContains(it.Error(), "invalid value length")
}

func TestIndexIterator_Next_IteratorDoesNotRestartAfterExhaustion(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
	}

	gomock.InOrder(
		// Simulate iterator exhaustion.
		iter.EXPECT().Next().Return(false),
		// The iterator is also released since there are no more elements.
		iter.EXPECT().Error(),
		iter.EXPECT().Release(),
	)

	require.False(it.Next())
	require.True(ctrl.Satisfied())

	// Subsequent calls to Next should not create a new iterator.
	require.False(it.Next())
}

func TestIndexIterator_Next_AbortsIfBlockRangeIsExceeded(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	to := idx.Block(20)
	it := &indexIterator{
		table: NewMock_table(ctrl),
		iter:  iter,
		to:    to,
	}

	// Simulate a key with a block number greater than 'to'.
	id := NewID(uint64(to)+1, common.Hash{}, 0)
	key := topicKey(common.Hash{}, 0, id)
	value := []byte{5}
	gomock.InOrder(
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Key().Return(key),
		iter.EXPECT().Value().Return(value),
		iter.EXPECT().Error(),
		// The iterator is also released since there are no more elements.
		iter.EXPECT().Error(),
		iter.EXPECT().Release(),
	)

	require.False(it.Next())
}

func TestIndexIterator_Current_DefaultReturnsZeroValue(t *testing.T) {
	it := &indexIterator{}
	require.Zero(t, it.Current())
}

func TestIndexIterator_Current_ReturnsCachedLogRecord(t *testing.T) {
	require := require.New(t)

	block := uint64(10)
	txHash := common.Hash{1, 2, 3}
	logIndex := uint(5)
	id := NewID(block, txHash, logIndex)

	topicCount := uint8(3)
	logRec := *newLogrec(id, topicCount)

	it := &indexIterator{
		current: logRec,
	}

	require.Equal(logRec, it.Current())
}

func TestIndexIterator_Seek_DefaultOrExhaustedIteratorReturnsFalse(t *testing.T) {
	it := &indexIterator{}
	require.False(t, it.Seek(logrec{}))
}

func TestIndexIterator_Seek_FailsIfThereIsAnExistingIssue(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	issue := errors.New("injected issue")
	it := &indexIterator{
		table: NewMock_table(ctrl),
		err:   issue,
	}

	require.False(it.Seek(logrec{}))
}

func TestIndexIterator_Seek_ReleasesOldIteratorAndCreatesNewOne(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)
	oldIter := NewMock_iterator(ctrl)
	newIter := NewMock_iterator(ctrl)

	prefix := []byte{1, 2, 3}

	it := &indexIterator{
		table:  table,
		iter:   oldIter,
		prefix: prefix,
		from:   idx.Block(0),
		to:     idx.Block(100),
	}

	block := uint64(10)
	id := NewID(block, common.Hash{4, 5, 6}, 7)
	target := *newLogrec(id, 12)

	// Expect the old iterator to be released and a new iterator to be created
	// with the correct prefix and start key.
	gomock.InOrder(
		oldIter.EXPECT().Error().Return(nil),
		oldIter.EXPECT().Release(),
		table.EXPECT().NewIterator(prefix, target.ID.Bytes()).Return(newIter),
		newIter.EXPECT().Next().Return(true),
		newIter.EXPECT().Key().Return(topicKey(common.Hash{}, 0, id)),
		newIter.EXPECT().Value().Return([]byte{12}),
		newIter.EXPECT().Error().Return(nil),
	)

	require.True(it.Seek(target))
	require.Equal(newIter, it.iter)
}

func TestIndexIterator_Seek_FailsIfNewIteratorCreationFails(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	table := NewMock_table(ctrl)
	oldIter := NewMock_iterator(ctrl)

	it := &indexIterator{
		table: table,
		iter:  oldIter,
	}

	block := uint64(10)
	id := NewID(block, common.Hash{4, 5, 6}, 7)
	target := *newLogrec(id, 12)

	gomock.InOrder(
		oldIter.EXPECT().Error().Return(nil),
		oldIter.EXPECT().Release(),
		table.EXPECT().NewIterator(gomock.Any(), gomock.Any()).Return(nil),
	)

	require.False(it.Seek(target))
	require.Nil(it.iter)
	require.ErrorContains(it.Error(), "failed to create iterator")
}

func TestIndexIterator_Error_DefaultHasNoError(t *testing.T) {
	it := &indexIterator{}
	require.NoError(t, it.Error())
}

func TestIndexIterator_Error_ReturnsCollectedError(t *testing.T) {
	require := require.New(t)
	issue := errors.New("injected issue")
	it := &indexIterator{
		err: issue,
	}
	require.ErrorIs(it.Error(), issue)
}

func TestIndexIterator_Release_DefaultDoesNotPanic(t *testing.T) {
	it := &indexIterator{}
	it.Release()
}

func TestIndexIterator_Release_ReleasesNestedIterator(t *testing.T) {
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	gomock.InOrder(
		iter.EXPECT().Error().Return(nil),
		iter.EXPECT().Release(),
	)

	it := &indexIterator{
		iter: iter,
	}

	it.Release()
}

func TestIndexIterator_Release_PreservesIteratorError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	iter := NewMock_iterator(ctrl)

	issue := errors.New("injected issue")
	iter.EXPECT().Error().Return(issue).MinTimes(1)
	iter.EXPECT().Release()

	it := &indexIterator{
		iter: iter,
	}

	it.Release()
	require.ErrorIs(it.Error(), issue)
}

// used for mock generation

type _table interface {
	kvdb.Iteratee
}

var _ _table = (*Mock_table)(nil) // to avoid an unused warning for the _table interface

type _iterator interface {
	kvdb.Iterator
}

var _ _iterator = (*Mock_iterator)(nil) // to avoid an unused warning for the _iterator interface

type _reader interface {
	kvdb.Reader
}

var _ _reader = (*Mock_reader)(nil) // to avoid an unused warning for the _reader interface
